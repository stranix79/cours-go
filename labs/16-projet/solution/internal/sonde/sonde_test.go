package sonde

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"cours-go/labs/16-projet/solution/internal/config"
)

func TestSonderHTTP(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	erreur := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boum", http.StatusInternalServerError)
	}))
	defer erreur.Close()
	lent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer lent.Close()
	ferme := httptest.NewServer(http.NotFoundHandler())
	adresseFermee := ferme.URL
	ferme.Close()

	client := &http.Client{Timeout: 50 * time.Millisecond}
	cas := []struct {
		nom     string
		url     string
		veutErr string // sous-chaîne attendue dans l'erreur, vide si succès
	}{
		{"serveur qui répond 200", ok.URL, ""},
		{"serveur qui répond 500", erreur.URL, "500"},
		{"serveur fermé", adresseFermee, "connect"},
		{"serveur trop lent", lent.URL, "Timeout"},
		{"url invalide", "://pas-une-url", "missing protocol"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			debut := time.Now()
			err := SonderHTTP(context.Background(), client, c.url)
			if c.veutErr == "" {
				if err != nil {
					t.Fatalf("erreur inattendue : %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("attendu une erreur, obtenu nil")
			}
			if !strings.Contains(err.Error(), c.veutErr) {
				t.Errorf("erreur %q ne contient pas %q", err, c.veutErr)
			}
			if d := time.Since(debut); d > 250*time.Millisecond {
				t.Errorf("la sonde a pris %s, le timeout n'est pas respecté", d)
			}
		})
	}
}

func TestSonderTCP(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	adresse := l.Addr().String()
	t.Run("port ouvert", func(t *testing.T) {
		if err := SonderTCP(context.Background(), adresse, time.Second); err != nil {
			t.Fatalf("erreur inattendue : %v", err)
		}
	})
	l.Close()
	t.Run("port fermé", func(t *testing.T) {
		if err := SonderTCP(context.Background(), adresse, time.Second); err == nil {
			t.Fatal("attendu une erreur sur un port fermé")
		}
	})
	t.Run("contexte déjà annulé", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := SonderTCP(ctx, "127.0.0.1:1", time.Second); err == nil {
			t.Fatal("attendu une erreur avec un contexte annulé")
		}
	})
}

func TestSondeurReseau(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	s := NouveauSondeur(time.Second)
	if s == nil || s.Client == nil {
		t.Fatal("NouveauSondeur doit renvoyer un sondeur avec un client")
	}
	t.Run("cible http up", func(t *testing.T) {
		r := s.Sonder(context.Background(), config.Cible{Nom: "web", Type: config.TypeHTTP, URL: srv.URL})
		if !r.Up || r.Cible != "web" || r.Type != "http" || r.Erreur != "" {
			t.Errorf("résultat inattendu : %+v", r)
		}
		if r.Latence <= 0 || r.Quand.IsZero() {
			t.Errorf("latence et horodatage doivent être renseignés : %+v", r)
		}
	})
	t.Run("cible tcp down", func(t *testing.T) {
		r := s.Sonder(context.Background(), config.Cible{Nom: "pg", Type: config.TypeTCP, Adresse: "127.0.0.1:1"})
		if r.Up || r.Erreur == "" {
			t.Errorf("attendu down avec une erreur : %+v", r)
		}
	})
	t.Run("type inconnu", func(t *testing.T) {
		r := s.Sonder(context.Background(), config.Cible{Nom: "x", Type: "icmp"})
		if r.Up || r.Erreur == "" {
			t.Errorf("attendu down avec une erreur : %+v", r)
		}
	})
}

func TestEtat(t *testing.T) {
	e := NouvelEtat()
	if e.Total() != 0 || len(e.Resultats()) != 0 {
		t.Fatal("un état neuf doit être vide")
	}
	t.Run("première vue est un changement", func(t *testing.T) {
		if !e.Enregistrer(Resultat{Cible: "web", Up: true}) {
			t.Error("la première fois doit compter comme un changement")
		}
	})
	t.Run("même état n'est pas un changement", func(t *testing.T) {
		if e.Enregistrer(Resultat{Cible: "web", Up: true}) {
			t.Error("up → up ne doit pas être un changement")
		}
	})
	t.Run("passage down est un changement", func(t *testing.T) {
		if !e.Enregistrer(Resultat{Cible: "web", Up: false, Erreur: "refus"}) {
			t.Error("up → down doit être un changement")
		}
	})
	e.Enregistrer(Resultat{Cible: "pg", Up: true})
	e.Enregistrer(Resultat{Cible: "api", Up: true})
	t.Run("résultats triés par nom", func(t *testing.T) {
		liste := e.Resultats()
		if len(liste) != 3 {
			t.Fatalf("attendu 3 résultats, obtenu %d", len(liste))
		}
		if liste[0].Cible != "api" || liste[1].Cible != "pg" || liste[2].Cible != "web" {
			t.Errorf("ordre inattendu : %v %v %v", liste[0].Cible, liste[1].Cible, liste[2].Cible)
		}
		if liste[2].Up || liste[2].Erreur != "refus" {
			t.Errorf("le dernier résultat de web doit être conservé : %+v", liste[2])
		}
	})
	t.Run("total compte chaque vérification", func(t *testing.T) {
		if e.Total() != 5 {
			t.Errorf("Total = %d, attendu 5", e.Total())
		}
	})
}

// sondeurFictif répond sans réseau, compte les appels et mesure le nombre
// maximal de sondes en cours en même temps (pour vérifier le worker pool).
type sondeurFictif struct {
	delai   time.Duration
	up      bool
	appels  atomic.Int64
	enCours atomic.Int64
	maximum atomic.Int64
}

func (f *sondeurFictif) Sonder(ctx context.Context, c config.Cible) Resultat {
	n := f.enCours.Add(1)
	defer f.enCours.Add(-1)
	for {
		m := f.maximum.Load()
		if n <= m || f.maximum.CompareAndSwap(m, n) {
			break
		}
	}
	f.appels.Add(1)
	select {
	case <-time.After(f.delai):
	case <-ctx.Done():
	}
	return Resultat{Cible: c.Nom, Type: c.Type, Up: f.up, Quand: time.Now()}
}

func cibles(n int) []config.Cible {
	liste := make([]config.Cible, n)
	for i := range liste {
		liste[i] = config.Cible{Nom: "cible-" + string(rune('a'+i)), Type: config.TypeTCP, Adresse: "127.0.0.1:1"}
	}
	return liste
}

func TestMoteurTour(t *testing.T) {
	f := &sondeurFictif{delai: 20 * time.Millisecond, up: true}
	m := &Moteur{Sondeur: f, Cibles: cibles(6), Workers: 2, Etat: NouvelEtat()}
	debut := time.Now()
	m.Tour(context.Background())
	duree := time.Since(debut)

	if got := f.appels.Load(); got != 6 {
		t.Errorf("attendu 6 sondes, obtenu %d", got)
	}
	if got := len(m.Etat.Resultats()); got != 6 {
		t.Errorf("attendu 6 résultats dans l'état, obtenu %d", got)
	}
	if got := f.maximum.Load(); got > 2 {
		t.Errorf("au plus 2 sondes en parallèle attendues, observé %d", got)
	}
	if got := f.maximum.Load(); got < 2 {
		t.Errorf("les sondes doivent être parallèles, observé %d en même temps", got)
	}
	// 6 sondes de 20 ms sur 2 workers : 3 vagues, soit ~60 ms, jamais 120.
	if duree > 110*time.Millisecond {
		t.Errorf("le tour a pris %s, les workers ne travaillent pas en parallèle", duree)
	}
}

func TestMoteurTourAnnule(t *testing.T) {
	f := &sondeurFictif{delai: time.Second, up: true}
	m := &Moteur{Sondeur: f, Cibles: cibles(4), Workers: 1, Etat: NouvelEtat()}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	debut := time.Now()
	m.Tour(ctx)
	if d := time.Since(debut); d > 200*time.Millisecond {
		t.Errorf("Tour avec un contexte annulé a pris %s, il doit rendre la main tout de suite", d)
	}
}

func TestMoteurBoucle(t *testing.T) {
	f := &sondeurFictif{up: true}
	m := &Moteur{Sondeur: f, Cibles: cibles(2), Intervalle: 20 * time.Millisecond, Workers: 2, Etat: NouvelEtat()}
	ctx, cancel := context.WithTimeout(context.Background(), 130*time.Millisecond)
	defer cancel()
	fini := make(chan struct{})
	go func() {
		m.Boucle(ctx)
		close(fini)
	}()
	select {
	case <-fini:
	case <-time.After(2 * time.Second):
		t.Fatal("Boucle ne s'est pas arrêtée après l'annulation du contexte")
	}
	// Un tour immédiat puis un tour toutes les 20 ms pendant 130 ms : au moins
	// 4 tours de 2 cibles, même sur une machine chargée.
	if got := f.appels.Load(); got < 8 {
		t.Errorf("attendu au moins 8 sondes (tour immédiat + ticks), obtenu %d", got)
	}
	if m.Etat.Total() != uint64(f.appels.Load()) {
		t.Errorf("Total = %d, appels = %d : chaque sonde doit être enregistrée", m.Etat.Total(), f.appels.Load())
	}
}
