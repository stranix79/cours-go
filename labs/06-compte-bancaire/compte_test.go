// Labo 06 : les tests de compte.go et banque.go.
// Lancé par : go test . (teste TON code, à la racine du labo) ou
// go test ./solution/ (teste la solution). Le fichier est identique aux deux endroits.
package main

import (
	"errors"
	"strings"
	"testing"
)

// TestDeposerModifieLeCompte est LE test du chapitre 6. Si Deposer a un
// receveur valeur (func (c Compte) Deposer), il modifie une copie et le solde
// reste à 0 : ce test échoue. Le compilateur, lui, ne dit rien.
func TestDeposerModifieLeCompte(t *testing.T) {
	c := NewCompte("alice")
	if err := c.Deposer(1000, "test"); err != nil {
		t.Fatalf("Deposer : erreur inattendue : %v", err)
	}
	if got := c.Solde(); got != 1000 {
		t.Errorf("Solde() = %d après un dépôt de 1000, attendu 1000 (receveur valeur ?)", got)
	}
	if got := c.Nombre(); got != 1 {
		t.Errorf("Nombre() = %d après un dépôt, attendu 1 (Journal.Enregistrer à receveur valeur ?)", got)
	}
}

func TestDeposerRetirer(t *testing.T) {
	cas := []struct {
		nom     string
		depots  []int64
		retrait int64
		veut    int64
		veutErr error
	}{
		{"dépôt puis retrait", []int64{10000}, 2500, 7500, nil},
		{"retrait exact", []int64{5000}, 5000, 0, nil},
		{"solde insuffisant", []int64{1000}, 1001, 1000, ErrSoldeInsuffisant},
		{"retrait sur compte vide", nil, 1, 0, ErrSoldeInsuffisant},
		{"retrait négatif", []int64{1000}, -5, 1000, ErrMontantInvalide},
		{"retrait nul", []int64{1000}, 0, 1000, ErrMontantInvalide},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			compte := NewCompte("test")
			for _, d := range c.depots {
				if err := compte.Deposer(d, "dépôt"); err != nil {
					t.Fatalf("Deposer(%d) : %v", d, err)
				}
			}
			err := compte.Retirer(c.retrait, "retrait")
			if !errors.Is(err, c.veutErr) {
				t.Fatalf("Retirer(%d) : erreur %v, attendu %v", c.retrait, err, c.veutErr)
			}
			if got := compte.Solde(); got != c.veut {
				t.Errorf("Solde() = %d, attendu %d", got, c.veut)
			}
		})
	}
}

func TestDeposerMontantInvalide(t *testing.T) {
	c := NewCompte("alice")
	for _, m := range []int64{0, -100} {
		err := c.Deposer(m, "x")
		if !errors.Is(err, ErrMontantInvalide) {
			t.Errorf("Deposer(%d) : erreur %v, attendu ErrMontantInvalide", m, err)
		}
	}
	if c.Solde() != 0 || c.Nombre() != 0 {
		t.Errorf("un dépôt refusé ne doit rien changer : solde %d, opérations %d", c.Solde(), c.Nombre())
	}
}

func TestFormatEuros(t *testing.T) {
	cas := []struct {
		centimes int64
		veut     string
	}{
		{0, "0,00 €"},
		{5, "0,05 €"},
		{150000, "1500,00 €"},
		{2050, "20,50 €"},
	}
	for _, c := range cas {
		if got := FormatEuros(c.centimes); got != c.veut {
			t.Errorf("FormatEuros(%d) = %q, attendu %q", c.centimes, got, c.veut)
		}
	}
}

func TestReleve(t *testing.T) {
	c := NewCompte("alice")
	c.Deposer(150000, "salaire")
	c.Retirer(2050, "courses")
	veut := "Relevé de alice\n" +
		"  dépôt          +1500,00 €  salaire\n" +
		"  retrait        -  20,50 €  courses\n" +
		"Solde : 1479,50 €"
	if got := c.String(); got != veut {
		t.Errorf("String() =\n%s\nattendu\n%s", got, veut)
	}
}

func TestVirement(t *testing.T) {
	b := NewBanque()
	alice, err := b.Ouvrir("alice")
	if err != nil {
		t.Fatalf("Ouvrir(alice) : %v", err)
	}
	bob, err := b.Ouvrir("bob")
	if err != nil {
		t.Fatalf("Ouvrir(bob) : %v", err)
	}
	alice.Deposer(10000, "initial")

	if err := b.Virement("alice", "bob", 4000); err != nil {
		t.Fatalf("Virement : %v", err)
	}
	if alice.Solde() != 6000 || bob.Solde() != 4000 {
		t.Errorf("après virement : alice %d, bob %d ; attendu 6000 et 4000", alice.Solde(), bob.Solde())
	}
	// Le pointeur rendu par Ouvrir et celui de la map sont le même compte.
	bob2, _ := b.Compte("bob")
	if bob2 != bob {
		t.Errorf("Compte(bob) doit rendre le même pointeur que Ouvrir(bob)")
	}
	if !strings.Contains(alice.String(), "virement émis") || !strings.Contains(bob.String(), "virement reçu") {
		t.Errorf("le relevé doit mentionner « virement émis » et « virement reçu » :\n%s\n%s", alice, bob)
	}
}

func TestVirementEchoueSansRienChanger(t *testing.T) {
	b := NewBanque()
	alice, _ := b.Ouvrir("alice")
	bob, _ := b.Ouvrir("bob")
	alice.Deposer(1000, "initial")

	err := b.Virement("alice", "bob", 5000)
	if !errors.Is(err, ErrSoldeInsuffisant) {
		t.Fatalf("Virement trop gros : erreur %v, attendu ErrSoldeInsuffisant", err)
	}
	if alice.Solde() != 1000 || bob.Solde() != 0 {
		t.Errorf("un virement refusé ne doit rien changer : alice %d, bob %d", alice.Solde(), bob.Solde())
	}
	if alice.Nombre() != 1 || bob.Nombre() != 0 {
		t.Errorf("un virement refusé ne doit rien enregistrer : alice %d op, bob %d op", alice.Nombre(), bob.Nombre())
	}

	err = b.Virement("alice", "carol", 100)
	if !errors.Is(err, ErrCompteInconnu) {
		t.Errorf("Virement vers un inconnu : erreur %v, attendu ErrCompteInconnu", err)
	}
	if alice.Solde() != 1000 {
		t.Errorf("un virement vers un inconnu ne doit pas débiter : alice %d", alice.Solde())
	}
}

func TestOuvrirDeuxFois(t *testing.T) {
	b := NewBanque()
	if _, err := b.Ouvrir("alice"); err != nil {
		t.Fatalf("Ouvrir : %v", err)
	}
	_, err := b.Ouvrir("alice")
	if !errors.Is(err, ErrCompteExistant) {
		t.Errorf("Ouvrir deux fois : erreur %v, attendu ErrCompteExistant", err)
	}
}
