package main

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

// serveurLocal ouvre un vrai serveur TCP sur un port libre de 127.0.0.1,
// accepte et referme chaque connexion, et rend le port. Fermé à la fin du test.
func serveurLocal(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen : %v", err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return // listener fermé : fin du test
			}
			conn.Close()
		}
	}()
	return l.Addr().(*net.TCPAddr).Port
}

// portFerme rend un port sur lequel personne n'écoute : on en réserve un,
// puis on le libère aussitôt.
func portFerme(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen : %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func TestParseCible(t *testing.T) {
	cas := []struct {
		nom    string
		entree string
		veut   Cible
		erreur bool
	}{
		{"hôte et port", "db1:5432", Cible{Hote: "db1", Port: 5432}, false},
		{"adresse IPv4", "127.0.0.1:22", Cible{Hote: "127.0.0.1", Port: 22}, false},
		{"adresse IPv6", "[::1]:443", Cible{Hote: "::1", Port: 443}, false},
		{"sans port", "db1", Cible{}, true},
		{"port non numérique", "db1:pg", Cible{}, true},
		{"port zéro", "db1:0", Cible{}, true},
		{"port trop grand", "db1:70000", Cible{}, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got, err := ParseCible(c.entree)
			if (err != nil) != c.erreur {
				t.Fatalf("erreur attendue : %v, obtenue : %v", c.erreur, err)
			}
			if !c.erreur && got != c.veut {
				t.Errorf("attendu %+v, obtenu %+v", c.veut, got)
			}
		})
	}
}

func TestSonderLocalRepond(t *testing.T) {
	port := serveurLocal(t)
	r := Sonder(context.Background(), "127.0.0.1", port, time.Second)
	if !r.OK || r.Err != nil {
		t.Fatalf("attendu OK, obtenu OK=%v err=%v", r.OK, r.Err)
	}
	if r.Cible.Port != port || r.Cible.Hote != "127.0.0.1" {
		t.Errorf("la cible doit être recopiée dans le résultat, obtenu %+v", r.Cible)
	}
	if r.Duree <= 0 || r.Duree > time.Second {
		t.Errorf("durée incohérente : %v", r.Duree)
	}
}

func TestSonderPortFerme(t *testing.T) {
	port := portFerme(t)
	r := Sonder(context.Background(), "127.0.0.1", port, time.Second)
	if r.OK || r.Err == nil {
		t.Fatalf("attendu KO avec une erreur, obtenu OK=%v err=%v", r.OK, r.Err)
	}
}

func TestSonderContexteAnnule(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // annulé AVANT la sonde
	debut := time.Now()
	r := Sonder(ctx, "192.0.2.1", 80, 5*time.Second) // adresse de documentation : ne répond jamais
	if r.OK || !errors.Is(r.Err, context.Canceled) {
		t.Fatalf("attendu context.Canceled, obtenu OK=%v err=%v", r.OK, r.Err)
	}
	if time.Since(debut) > 500*time.Millisecond {
		t.Errorf("une sonde annulée doit rendre la main tout de suite, a pris %v", time.Since(debut))
	}
}

func TestSonderTousOrdreEtParallelisme(t *testing.T) {
	ouvert := serveurLocal(t)
	ferme := portFerme(t)
	cibles := []Cible{
		{Hote: "127.0.0.1", Port: ouvert},
		{Hote: "127.0.0.1", Port: ferme},
		{Hote: "127.0.0.1", Port: ouvert},
		{Hote: "127.0.0.1", Port: ferme},
		{Hote: "127.0.0.1", Port: ouvert},
	}
	for _, p := range []int{1, 2, 10, 0} {
		res := SonderTous(context.Background(), cibles, p)
		if len(res) != len(cibles) {
			t.Fatalf("parallélisme %d : attendu %d résultats, obtenu %d", p, len(cibles), len(res))
		}
		for i, r := range res {
			if r.Cible != cibles[i] {
				t.Errorf("parallélisme %d : résultat %d pour %v, attendu %v (ordre non respecté)", p, i, r.Cible, cibles[i])
			}
			if veutOK := cibles[i].Port == ouvert; r.OK != veutOK {
				t.Errorf("parallélisme %d : cible %v : OK=%v, attendu %v (err=%v)", p, r.Cible, r.OK, veutOK, r.Err)
			}
		}
	}
}

func TestSonderTousVide(t *testing.T) {
	res := SonderTous(context.Background(), nil, 4)
	if len(res) != 0 {
		t.Errorf("attendu aucun résultat, obtenu %d", len(res))
	}
}

func TestSonderTousAnnulation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var cibles []Cible
	for i := 0; i < 20; i++ {
		cibles = append(cibles, Cible{Hote: "192.0.2.1", Port: 80, Timeout: 5 * time.Second})
	}
	debut := time.Now()
	res := SonderTous(ctx, cibles, 2)
	if time.Since(debut) > time.Second {
		t.Fatalf("avec un contexte annulé, SonderTous doit rendre la main vite, a pris %v", time.Since(debut))
	}
	if len(res) != len(cibles) {
		t.Fatalf("attendu %d résultats même annulés, obtenu %d", len(cibles), len(res))
	}
	for i, r := range res {
		if r.OK || !errors.Is(r.Err, context.Canceled) {
			t.Errorf("résultat %d : attendu context.Canceled, obtenu OK=%v err=%v", i, r.OK, r.Err)
		}
	}
}

func TestFormatTableau(t *testing.T) {
	res := []Resultat{
		{Cible: Cible{Hote: "db1", Port: 5432}, OK: true, Duree: 1500 * time.Microsecond},
		{Cible: Cible{Hote: "web1", Port: 443}, OK: false, Duree: 2 * time.Second, Err: errors.New("i/o timeout")},
	}
	veut := "CIBLE                  ETAT    DUREE  DETAIL\n" +
		"db1:5432               OK        2ms  \n" +
		"web1:443               KO         2s  i/o timeout\n"
	if got := FormatTableau(res); got != veut {
		t.Errorf("tableau :\n%s\nattendu :\n%s", got, veut)
	}
	if !strings.HasPrefix(FormatTableau(nil), "CIBLE") {
		t.Error("même sans résultat, l'en-tête doit être présent")
	}
}
