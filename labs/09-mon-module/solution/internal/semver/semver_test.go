// Labo 09 : les tests du paquet semver. Table-driven partout, un Example
// par fonction publique (vérifié par go test, affiché par go doc), et deux
// benchmarks. Identique dans internal/semver et solution/internal/semver.
package semver

import (
	"errors"
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	cas := []struct {
		nom    string
		texte  string
		veut   Version
		erreur bool
	}{
		{"simple", "1.2.3", Version{1, 2, 3}, false},
		{"préfixe v", "v1.2.3", Version{1, 2, 3}, false},
		{"espaces et retour", " v1.2.3\n", Version{1, 2, 3}, false},
		{"zéros", "0.0.0", Version{0, 0, 0}, false},
		{"grands nombres", "10.200.3000", Version{10, 200, 3000}, false},
		{"deux parties", "1.2", Version{}, true},
		{"quatre parties", "1.2.3.4", Version{}, true},
		{"lettres", "a.b.c", Version{}, true},
		{"négatif", "1.-2.3", Version{}, true},
		{"vide", "", Version{}, true},
		{"partie vide", "1..3", Version{}, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got, err := Parse(c.texte)
			if c.erreur {
				if !errors.Is(err, ErrInvalide) {
					t.Fatalf("Parse(%q) : erreur %v, attendu ErrInvalide", c.texte, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) : erreur inattendue : %v", c.texte, err)
			}
			if got != c.veut {
				t.Errorf("Parse(%q) = %v, attendu %v", c.texte, got, c.veut)
			}
		})
	}
}

func TestString(t *testing.T) {
	if got := (Version{1, 2, 3}).String(); got != "1.2.3" {
		t.Errorf("String() = %q, attendu \"1.2.3\"", got)
	}
	// fmt utilise String() : c'est ce que voit l'utilisateur de la CLI.
	if got := fmt.Sprint(Version{10, 0, 1}); got != "10.0.1" {
		t.Errorf("Sprint = %q, attendu \"10.0.1\"", got)
	}
}

func TestCompare(t *testing.T) {
	cas := []struct {
		a, b string
		veut int
	}{
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.2.3", "1.2.3", 0},
		{"1.2.0", "1.10.0", -1}, // numérique, pas alphabétique
		{"1.2.3", "1.2.10", -1},
		{"0.9.9", "1.0.0", -1},
		{"1.10.0", "1.9.9", 1},
	}
	for _, c := range cas {
		t.Run(c.a+" vs "+c.b, func(t *testing.T) {
			a, b := doitParser(t, c.a), doitParser(t, c.b)
			if got := Compare(a, b); got != c.veut {
				t.Errorf("Compare(%v, %v) = %d, attendu %d", a, b, got, c.veut)
			}
		})
	}
}

// doitParser est un helper : il parse ou arrête le test. Grâce à t.Helper(),
// un échec est rapporté à la ligne du test appelant.
func doitParser(t *testing.T, s string) Version {
	t.Helper()
	v, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q) : %v", s, err)
	}
	return v
}

func TestBump(t *testing.T) {
	cas := []struct {
		partie string
		veut   string
	}{
		{"major", "2.0.0"},
		{"minor", "1.3.0"},
		{"patch", "1.2.4"},
	}
	for _, c := range cas {
		t.Run(c.partie, func(t *testing.T) {
			v := Version{1, 2, 3}
			got, err := Bump(v, c.partie)
			if err != nil {
				t.Fatalf("Bump : %v", err)
			}
			if got.String() != c.veut {
				t.Errorf("Bump(1.2.3, %q) = %v, attendu %s", c.partie, got, c.veut)
			}
			// Version est une valeur : l'original ne bouge pas.
			if v != (Version{1, 2, 3}) {
				t.Errorf("Bump a modifié l'original : %v", v)
			}
		})
	}
	if _, err := Bump(Version{1, 2, 3}, "mega"); err == nil {
		t.Error("Bump avec une partie inconnue doit échouer")
	}
}

func TestCompatible(t *testing.T) {
	cas := []struct {
		version, contrainte string
		veut                bool
	}{
		{"1.2.3", "^1.2.0", true},
		{"1.9.0", "^1.2.0", true},
		{"2.0.0", "^1.2.0", false}, // changement de majeur
		{"1.1.9", "^1.2.0", false}, // trop vieux
		{"1.2.9", "~1.2.0", true},
		{"1.3.0", "~1.2.0", false}, // ~ n'accepte que les correctifs
		{"1.2.3", "1.2.3", true},
		{"1.2.4", "1.2.3", false},
		{"1.5.0", ">=1.2.0", true},
		{"1.5.0", ">1.5.0", false},
		{"1.5.0", "<2.0.0", true},
		{"2.0.0", "<=2.0.0", true},
		{"1.5.0", ">=1.2.0,<2.0.0", true},
		{"2.1.0", ">=1.2.0,<2.0.0", false},
		{"1.5.0", " >=1.2.0 , <2.0.0 ", true}, // espaces tolérés
	}
	for _, c := range cas {
		t.Run(c.version+" "+c.contrainte, func(t *testing.T) {
			got, err := Compatible(doitParser(t, c.version), c.contrainte)
			if err != nil {
				t.Fatalf("Compatible : erreur inattendue : %v", err)
			}
			if got != c.veut {
				t.Errorf("Compatible(%s, %q) = %v, attendu %v", c.version, c.contrainte, got, c.veut)
			}
		})
	}
}

func TestCompatibleContrainteInvalide(t *testing.T) {
	for _, contrainte := range []string{"", "^abc", ">=1.2", "1.2.3,"} {
		_, err := Compatible(Version{1, 0, 0}, contrainte)
		if !errors.Is(err, ErrInvalide) {
			t.Errorf("Compatible(1.0.0, %q) : erreur %v, attendu ErrInvalide", contrainte, err)
		}
	}
}

// Les Example sont exécutés par go test (la sortie est comparée au commentaire
// Output) et affichés par go doc sous la fonction correspondante.

func ExampleParse() {
	v, err := Parse("v1.2.3")
	fmt.Println(v, err)
	// Output: 1.2.3 <nil>
}

func ExampleBump() {
	v, _ := Bump(Version{1, 2, 3}, "minor")
	fmt.Println(v)
	// Output: 1.3.0
}

func ExampleCompatible() {
	v, _ := Parse("1.5.0")
	ok, _ := Compatible(v, "^1.2.0")
	fmt.Println(ok)
	ok, _ = Compatible(v, "~1.2.0")
	fmt.Println(ok)
	// Output:
	// true
	// false
}

func BenchmarkParse(b *testing.B) {
	for b.Loop() {
		Parse("v10.200.3000")
	}
}

func BenchmarkCompatible(b *testing.B) {
	v := Version{1, 5, 0}
	for b.Loop() {
		Compatible(v, ">=1.2.0,<2.0.0")
	}
}
