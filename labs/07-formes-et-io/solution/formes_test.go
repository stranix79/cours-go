// Labo 07 : les tests de formes.go.
// Lancé par : go test . (ton code) ou go test ./solution/ (la solution).
package main

import (
	"math"
	"testing"
)

// proche compare deux flottants à 1e-9 près : on ne compare JAMAIS des
// float64 avec ==, le calcul de pi*r*r n'est pas exact.
func proche(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestFormes(t *testing.T) {
	cas := []struct {
		nom       string
		forme     Forme
		aire      float64
		perimetre float64
		texte     string
	}{
		{"rectangle 3x4", Rectangle{L: 3, H: 4}, 12, 14, "rectangle 3x4"},
		{"rectangle carré", Rectangle{L: 2.5, H: 2.5}, 6.25, 10, "rectangle 2.5x2.5"},
		{"cercle unité", Cercle{R: 1}, math.Pi, 2 * math.Pi, "cercle r=1"},
		{"cercle r=2", Cercle{R: 2}, 4 * math.Pi, 4 * math.Pi, "cercle r=2"},
		{"triangle 3-4-5", Triangle{A: 3, B: 4, C: 5}, 6, 12, "triangle 3-4-5"},
		{"triangle équilatéral", Triangle{A: 2, B: 2, C: 2}, math.Sqrt(3), 6, "triangle 2-2-2"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := c.forme.Aire(); !proche(got, c.aire) {
				t.Errorf("Aire() = %g, attendu %g", got, c.aire)
			}
			if got := c.forme.Perimetre(); !proche(got, c.perimetre) {
				t.Errorf("Perimetre() = %g, attendu %g", got, c.perimetre)
			}
			// Chaque forme doit aussi être un fmt.Stringer : on le vérifie par
			// une assertion de type, sans planter si elle échoue.
			s, ok := c.forme.(interface{ String() string })
			if !ok {
				t.Fatalf("%T n'a pas de méthode String()", c.forme)
			}
			if got := s.String(); got != c.texte {
				t.Errorf("String() = %q, attendu %q", got, c.texte)
			}
		})
	}
}

func TestTotal(t *testing.T) {
	cas := []struct {
		nom    string
		formes []Forme
		veut   float64
	}{
		{"vide", nil, 0},
		{"un rectangle", []Forme{Rectangle{L: 2, H: 3}}, 6},
		{"mélange", []Forme{Rectangle{L: 3, H: 4}, Cercle{R: 1}, Triangle{A: 3, B: 4, C: 5}}, 18 + math.Pi},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := Total(c.formes); !proche(got, c.veut) {
				t.Errorf("Total() = %g, attendu %g", got, c.veut)
			}
		})
	}
}
