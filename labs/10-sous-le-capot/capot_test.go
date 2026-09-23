package capot

import (
	"os"
	"strings"
	"testing"
)

func TestNouveauParValeur(t *testing.T) {
	s := NouveauParValeur("db1")
	if s.Nom != "db1" || s.Port != 5432 {
		t.Errorf("attendu {db1 5432}, obtenu %+v", s)
	}
}

func TestNouveauParPointeur(t *testing.T) {
	p := NouveauParPointeur("db2")
	if p == nil {
		t.Fatal("attendu un pointeur non nil")
	}
	if p.Nom != "db2" || p.Port != 5432 {
		t.Errorf("attendu {db2 5432}, obtenu %+v", *p)
	}
}

// TestCompteurCasse ne s'exécute que si RACE=1 est posé : sans le détecteur
// il échoue « par hasard » (un nombre différent à chaque fois), avec
// go test -race il échoue à coup sûr en signalant la course. Il est là pour
// que tu VOIES le problème, pas pour être vert.
func TestCompteurCasse(t *testing.T) {
	if os.Getenv("RACE") != "1" {
		t.Skip("témoin de course : lance RACE=1 go test -race -run Casse .")
	}
	c := &CompteurCasse{}
	Marteler(c, 100, 1000)
	if c.Valeur() != 100_000 {
		t.Errorf("compteur cassé : attendu 100000, obtenu %d", c.Valeur())
	}
}

func TestCompteurSur(t *testing.T) {
	cas := []struct {
		nom        string
		goroutines int
		fois       int
	}{
		{"une goroutine", 1, 1000},
		{"cent goroutines", 100, 1000},
		{"mille goroutines courtes", 1000, 10},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			cpt := &CompteurSur{}
			Marteler(cpt, c.goroutines, c.fois)
			if got := cpt.Valeur(); got != c.goroutines*c.fois {
				t.Errorf("attendu %d, obtenu %d", c.goroutines*c.fois, got)
			}
		})
	}
}

func TestConcat(t *testing.T) {
	cas := []struct {
		nom  string
		mots []string
		veut string
	}{
		{"vide", nil, ""},
		{"un mot", []string{"GET"}, "GET"},
		{"trois mots", []string{"GET", "/metrics", "200"}, "GET /metrics 200"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := ConcatPlus(c.mots); got != c.veut {
				t.Errorf("ConcatPlus : attendu %q, obtenu %q", c.veut, got)
			}
			if got := ConcatBuilder(c.mots); got != c.veut {
				t.Errorf("ConcatBuilder : attendu %q, obtenu %q", c.veut, got)
			}
		})
	}
}

func TestEstPremier(t *testing.T) {
	cas := []struct {
		n    int
		veut bool
	}{
		{0, false}, {1, false}, {2, true}, {3, true}, {4, false},
		{17, true}, {25, false}, {97, true}, {1_000_003, true}, {1_000_001, false},
	}
	for _, c := range cas {
		if got := EstPremier(c.n); got != c.veut {
			t.Errorf("EstPremier(%d) : attendu %v, obtenu %v", c.n, c.veut, got)
		}
	}
}

func TestComptePremiers(t *testing.T) {
	cas := []struct {
		nom  string
		max  int
		veut int
	}{
		{"moins de 2", 2, 0},
		{"moins de 10", 10, 4},
		{"moins de 100", 100, 25},
		{"moins de 10000", 10_000, 1229},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := ComptePremiers(c.max); got != c.veut {
				t.Errorf("attendu %d, obtenu %d", c.veut, got)
			}
		})
	}
}

// mots1000 est le jeu de données des benchmarks : mille mots de log.
var mots1000 = strings.Fields(strings.Repeat("GET /metrics 200 OK 12ms ", 200))

func BenchmarkConcatPlus(b *testing.B) {
	for b.Loop() {
		ConcatPlus(mots1000)
	}
}

func BenchmarkConcatBuilder(b *testing.B) {
	for b.Loop() {
		ConcatBuilder(mots1000)
	}
}

func BenchmarkComptePremiers(b *testing.B) {
	for b.Loop() {
		ComptePremiers(200_000)
	}
}
