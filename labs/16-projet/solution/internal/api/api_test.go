package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cours-go/labs/16-projet/solution/internal/sonde"
)

// etatDeTest construit un état avec deux cibles, une up et une down.
func etatDeTest() *sonde.Etat {
	e := sonde.NouvelEtat()
	quand := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	e.Enregistrer(sonde.Resultat{Cible: "web", Type: "http", Up: true, Latence: 12 * time.Millisecond, Quand: quand})
	e.Enregistrer(sonde.Resultat{Cible: "pg", Type: "tcp", Up: false, Latence: 2 * time.Second, Erreur: "dial tcp: connection refused", Quand: quand})
	return e
}

func appeler(t *testing.T, h http.Handler, methode, chemin string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(methode, chemin, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealthz(t *testing.T) {
	mux := NouveauMux(etatDeTest(), "1.2.3", nil)
	rec := appeler(t, mux, http.MethodGet, "/healthz")
	if rec.Code != http.StatusOK {
		t.Fatalf("statut = %d, attendu 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, attendu application/json", ct)
	}
	var corps struct {
		OK      bool   `json:"ok"`
		Version string `json:"version"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&corps); err != nil {
		t.Fatalf("JSON illisible : %v (%q)", err, rec.Body.String())
	}
	if !corps.OK || corps.Version != "1.2.3" {
		t.Errorf("corps = %+v", corps)
	}
}

func TestStatus(t *testing.T) {
	mux := NouveauMux(etatDeTest(), "test", nil)
	rec := appeler(t, mux, http.MethodGet, "/api/status")
	if rec.Code != http.StatusOK {
		t.Fatalf("statut = %d, attendu 200 (%s)", rec.Code, rec.Body.String())
	}
	var s Statut
	if err := json.NewDecoder(rec.Body).Decode(&s); err != nil {
		t.Fatalf("JSON illisible : %v", err)
	}
	if s.Verifications != 2 {
		t.Errorf("verifications = %d, attendu 2", s.Verifications)
	}
	if len(s.Cibles) != 2 {
		t.Fatalf("attendu 2 cibles, obtenu %d", len(s.Cibles))
	}
	t.Run("triées par nom", func(t *testing.T) {
		if s.Cibles[0].Nom != "pg" || s.Cibles[1].Nom != "web" {
			t.Errorf("ordre : %s, %s", s.Cibles[0].Nom, s.Cibles[1].Nom)
		}
	})
	t.Run("latence en millisecondes", func(t *testing.T) {
		if s.Cibles[1].LatenceMs != 12 {
			t.Errorf("latence_ms = %v, attendu 12", s.Cibles[1].LatenceMs)
		}
	})
	t.Run("erreur présente seulement si down", func(t *testing.T) {
		if s.Cibles[0].Up || s.Cibles[0].Erreur == "" {
			t.Errorf("pg doit être down avec une erreur : %+v", s.Cibles[0])
		}
		if !s.Cibles[1].Up || s.Cibles[1].Erreur != "" {
			t.Errorf("web doit être up sans erreur : %+v", s.Cibles[1])
		}
	})
}

func TestStatusVide(t *testing.T) {
	mux := NouveauMux(sonde.NouvelEtat(), "test", nil)
	rec := appeler(t, mux, http.MethodGet, "/api/status")
	if !strings.Contains(rec.Body.String(), `"cibles":[]`) {
		t.Errorf("un état vide doit donner une liste vide, pas null : %s", rec.Body.String())
	}
}

func TestMetrics(t *testing.T) {
	mux := NouveauMux(etatDeTest(), "test", nil)
	rec := appeler(t, mux, http.MethodGet, "/metrics")
	if rec.Code != http.StatusOK {
		t.Fatalf("statut = %d, attendu 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, attendu text/plain", ct)
	}
	corps := rec.Body.String()
	for _, ligne := range []string{
		`sonde_up{cible="pg",type="tcp"} 0`,
		`sonde_up{cible="web",type="http"} 1`,
		`sonde_latence_secondes{cible="web",type="http"} 0.012`,
		`sonde_verifications_total 2`,
		`# TYPE sonde_verifications_total counter`,
	} {
		if !strings.Contains(corps, ligne+"\n") {
			t.Errorf("ligne absente : %q\n--- corps ---\n%s", ligne, corps)
		}
	}
}

func TestFormatMetriques(t *testing.T) {
	var b bytes.Buffer
	FormatMetriques(&b, []sonde.Resultat{
		{Cible: `dis"que\1`, Type: "tcp", Up: true, Latence: 1500 * time.Millisecond},
	}, 7)
	attendu := "# HELP sonde_up 1 si la cible a répondu à la dernière vérification, 0 sinon.\n" +
		"# TYPE sonde_up gauge\n" +
		"sonde_up{cible=\"dis\\\"que\\\\1\",type=\"tcp\"} 1\n" +
		"# HELP sonde_latence_secondes Durée de la dernière vérification, en secondes.\n" +
		"# TYPE sonde_latence_secondes gauge\n" +
		"sonde_latence_secondes{cible=\"dis\\\"que\\\\1\",type=\"tcp\"} 1.5\n" +
		"# HELP sonde_verifications_total Nombre de vérifications effectuées depuis le démarrage.\n" +
		"# TYPE sonde_verifications_total counter\n" +
		"sonde_verifications_total 7\n"
	if b.String() != attendu {
		t.Errorf("sortie inattendue :\n--- obtenu ---\n%s--- attendu ---\n%s", b.String(), attendu)
	}
}

func TestRoutes(t *testing.T) {
	mux := NouveauMux(etatDeTest(), "test", nil)
	cas := []struct {
		nom     string
		methode string
		chemin  string
		statut  int
	}{
		{"chemin inconnu", http.MethodGet, "/inconnu", http.StatusNotFound},
		{"POST refusé sur status", http.MethodPost, "/api/status", http.StatusMethodNotAllowed},
		{"DELETE refusé sur metrics", http.MethodDelete, "/metrics", http.StatusMethodNotAllowed},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			rec := appeler(t, mux, c.methode, c.chemin)
			if rec.Code != c.statut {
				t.Errorf("statut = %d, attendu %d", rec.Code, c.statut)
			}
		})
	}
}

func TestJournaliser(t *testing.T) {
	var sortie bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&sortie, nil))
	mux := NouveauMux(etatDeTest(), "test", log)
	appeler(t, mux, http.MethodGet, "/healthz")
	appeler(t, mux, http.MethodGet, "/inconnu")
	lignes := strings.Split(strings.TrimSpace(sortie.String()), "\n")
	if len(lignes) != 2 {
		t.Fatalf("attendu 2 lignes de log, obtenu %d :\n%s", len(lignes), sortie.String())
	}
	if !strings.Contains(lignes[0], `"chemin":"/healthz"`) || !strings.Contains(lignes[0], `"statut":200`) {
		t.Errorf("ligne 1 inattendue : %s", lignes[0])
	}
	if !strings.Contains(lignes[1], `"chemin":"/inconnu"`) || !strings.Contains(lignes[1], `"statut":404`) {
		t.Errorf("ligne 2 inattendue : %s", lignes[1])
	}
	if !strings.Contains(lignes[0], `"methode":"GET"`) || !strings.Contains(lignes[0], `"duree_ms"`) {
		t.Errorf("méthode et durée attendues : %s", lignes[0])
	}
}
