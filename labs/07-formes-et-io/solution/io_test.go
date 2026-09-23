// Labo 07 : les tests de io.go. Aucun fichier n'est touché : strings.NewReader
// joue le fichier d'entrée, bytes.Buffer joue le fichier de sortie.
// Lancé par : go test . (ton code) ou go test ./solution/ (la solution).
package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestCompter(t *testing.T) {
	cas := []struct {
		nom    string
		texte  string
		lignes int
		mots   int
		octets int
	}{
		{"vide", "", 0, 0, 0},
		{"une ligne", "a b\n", 1, 2, 4},
		{"sans retour final", "sans retour final", 0, 3, 17},
		{"lignes vides", "\n\n\n", 3, 0, 3},
		{"tabulations et espaces multiples", "db-01\t10.0.0.5   5432\n  \n", 2, 3, 25},
		{"utf-8", "é\n", 1, 1, 3},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			lignes, mots, octets, err := Compter(strings.NewReader(c.texte))
			if err != nil {
				t.Fatalf("Compter : erreur inattendue : %v", err)
			}
			if lignes != c.lignes || mots != c.mots || octets != c.octets {
				t.Errorf("Compter(%q) = %d, %d, %d ; attendu %d, %d, %d",
					c.texte, lignes, mots, octets, c.lignes, c.mots, c.octets)
			}
		})
	}
}

// lecteurCasse est un io.Reader qui échoue : trois lignes suffisent pour
// simuler un disque qui tombe. C'est ça, la puissance des petites interfaces.
type lecteurCasse struct{}

var errDisque = errors.New("erreur d'entrée/sortie simulée")

func (lecteurCasse) Read(p []byte) (int, error) { return 0, errDisque }

func TestCompterErreurDeLecture(t *testing.T) {
	_, _, _, err := Compter(lecteurCasse{})
	if !errors.Is(err, errDisque) {
		t.Fatalf("Compter sur un Reader cassé : erreur %v, attendu errDisque emballée", err)
	}
}

func TestEcrireRapport(t *testing.T) {
	formes := []Forme{Rectangle{L: 3, H: 4}, Cercle{R: 1}, Triangle{A: 3, B: 4, C: 5}}
	var buf bytes.Buffer
	if err := EcrireRapport(&buf, formes); err != nil {
		t.Fatalf("EcrireRapport : %v", err)
	}
	veut := "rectangle 3x4    aire=   12.00  périmètre=   14.00\n" +
		"cercle r=1       aire=    3.14  périmètre=    6.28\n" +
		"triangle 3-4-5   aire=    6.00  périmètre=   12.00\n" +
		"total            aire=   21.14\n"
	if got := buf.String(); got != veut {
		t.Errorf("EcrireRapport a écrit :\n%s\nattendu :\n%s", got, veut)
	}
}

func TestEcrireRapportVide(t *testing.T) {
	var buf bytes.Buffer
	if err := EcrireRapport(&buf, nil); err != nil {
		t.Fatalf("EcrireRapport(nil) : %v", err)
	}
	if got := buf.String(); got != "total            aire=    0.00\n" {
		t.Errorf("EcrireRapport(nil) a écrit %q", got)
	}
}

// ecrivainPlein est un io.Writer qui refuse tout : disque plein.
type ecrivainPlein struct{}

var errPlein = errors.New("disque plein simulé")

func (ecrivainPlein) Write(p []byte) (int, error) { return 0, errPlein }

func TestEcrireRapportErreurEcriture(t *testing.T) {
	err := EcrireRapport(ecrivainPlein{}, []Forme{Cercle{R: 1}})
	if !errors.Is(err, errPlein) {
		t.Fatalf("EcrireRapport sur un Writer cassé : erreur %v, attendu errPlein emballée", err)
	}
}
