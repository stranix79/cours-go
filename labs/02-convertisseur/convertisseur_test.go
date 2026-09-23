package main

import (
	"testing"
	"time"
)

// Un test table-driven : une liste de cas (entrée, sortie attendue), et une
// boucle qui lance un sous-test par cas avec t.Run. Le nom du sous-test
// apparaît dans la sortie de `go test -v`.

func TestFahrenheit(t *testing.T) {
	cas := []struct {
		nom     string
		celsius float64
		attendu float64
	}{
		{"ébullition", 100, 212},
		{"corps humain", 37, 98.6},
		{"point commun", -40, -40},
		{"zéro", 0, 32},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			obtenu := Fahrenheit(c.celsius)
			if diff := obtenu - c.attendu; diff > 0.01 || diff < -0.01 {
				t.Errorf("Fahrenheit(%v) = %v, attendu %v", c.celsius, obtenu, c.attendu)
			}
		})
	}
}

func TestCelsius(t *testing.T) {
	cas := []struct {
		nom        string
		fahrenheit float64
		attendu    float64
	}{
		{"ébullition", 212, 100},
		{"corps humain", 98.6, 37},
		{"gel", 32, 0},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			obtenu := Celsius(c.fahrenheit)
			if diff := obtenu - c.attendu; diff > 0.01 || diff < -0.01 {
				t.Errorf("Celsius(%v) = %v, attendu %v", c.fahrenheit, obtenu, c.attendu)
			}
		})
	}
}

func TestTailleLisible(t *testing.T) {
	cas := []struct {
		nom     string
		octets  int64
		attendu string
	}{
		{"zéro", 0, "0 o"},
		{"moins de 1 Ko", 512, "512 o"},
		{"limite basse", 1023, "1023 o"},
		{"1 Ko pile", 1024, "1.0 Ko"},
		{"1.5 Ko", 1536, "1.5 Ko"},
		{"10 Mo", 10 * 1024 * 1024, "10.0 Mo"},
		{"2.5 Go", 2560 * 1024 * 1024, "2.5 Go"},
		{"3 To", 3 * 1024 * 1024 * 1024 * 1024, "3.0 To"},
		{"au-delà du To on reste en To", 2048 * 1024 * 1024 * 1024 * 1024, "2048.0 To"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if obtenu := TailleLisible(c.octets); obtenu != c.attendu {
				t.Errorf("TailleLisible(%d) = %q, attendu %q", c.octets, obtenu, c.attendu)
			}
		})
	}
}

func TestParseDuree(t *testing.T) {
	cas := []struct {
		nom     string
		texte   string
		attendu time.Duration
		erreur  bool
	}{
		{"heures et minutes", "1h30m", 90 * time.Minute, false},
		{"secondes", "45s", 45 * time.Second, false},
		{"millisecondes", "250ms", 250 * time.Millisecond, false},
		{"texte invalide", "abc", 0, true},
		{"vide", "", 0, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			obtenu, err := ParseDuree(c.texte)
			if c.erreur {
				if err == nil {
					t.Fatalf("ParseDuree(%q) : erreur attendue, obtenu nil", c.texte)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDuree(%q) : erreur inattendue : %v", c.texte, err)
			}
			if obtenu != c.attendu {
				t.Errorf("ParseDuree(%q) = %v, attendu %v", c.texte, obtenu, c.attendu)
			}
		})
	}
}

func TestSecondes(t *testing.T) {
	cas := []struct {
		nom     string
		duree   time.Duration
		attendu int64
	}{
		{"1h30m", 90 * time.Minute, 5400},
		{"45s", 45 * time.Second, 45},
		{"tronque les millisecondes", 1500 * time.Millisecond, 1},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if obtenu := Secondes(c.duree); obtenu != c.attendu {
				t.Errorf("Secondes(%v) = %d, attendu %d", c.duree, obtenu, c.attendu)
			}
		})
	}
}
