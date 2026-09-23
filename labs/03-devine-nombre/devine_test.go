package main

import "testing"

func TestNouveauSecret(t *testing.T) {
	t.Run("même graine, même nombre", func(t *testing.T) {
		a := NouveauSecret(42, 100)
		b := NouveauSecret(42, 100)
		if a != b {
			t.Errorf("NouveauSecret(42) a donné %d puis %d", a, b)
		}
	})
	t.Run("toujours entre 1 et max", func(t *testing.T) {
		for graine := int64(0); graine < 200; graine++ {
			n := NouveauSecret(graine, 10)
			if n < 1 || n > 10 {
				t.Fatalf("NouveauSecret(%d, 10) = %d, hors de [1, 10]", graine, n)
			}
		}
	})
	t.Run("des graines différentes donnent des nombres différents", func(t *testing.T) {
		vus := map[int]bool{}
		for graine := int64(0); graine < 50; graine++ {
			vus[NouveauSecret(graine, 100)] = true
		}
		if len(vus) < 10 {
			t.Errorf("50 graines n'ont donné que %d valeurs distinctes", len(vus))
		}
	})
}

func TestEvaluer(t *testing.T) {
	cas := []struct {
		nom         string
		secret      int
		proposition int
		attendu     string
	}{
		{"trop petit", 50, 10, "plus grand"},
		{"trop grand", 50, 90, "plus petit"},
		{"juste en dessous", 50, 49, "plus grand"},
		{"juste au-dessus", 50, 51, "plus petit"},
		{"trouvé", 50, 50, "gagné"},
		{"borne basse", 1, 1, "gagné"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if obtenu := Evaluer(c.secret, c.proposition); obtenu != c.attendu {
				t.Errorf("Evaluer(%d, %d) = %q, attendu %q", c.secret, c.proposition, obtenu, c.attendu)
			}
		})
	}
}

func TestJouer(t *testing.T) {
	cas := []struct {
		nom     string
		secret  int
		entrees []int
		attendu int
	}{
		{"du premier coup", 42, []int{42}, 1},
		{"dichotomie", 42, []int{50, 25, 37, 43, 40, 42}, 6},
		{"les essais après la victoire ne comptent pas", 42, []int{42, 1, 2, 3}, 1},
		{"jamais trouvé", 42, []int{1, 2, 3}, 0},
		{"aucune proposition", 42, nil, 0},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if obtenu := Jouer(c.secret, c.entrees); obtenu != c.attendu {
				t.Errorf("Jouer(%d, %v) = %d, attendu %d", c.secret, c.entrees, obtenu, c.attendu)
			}
		})
	}
}
