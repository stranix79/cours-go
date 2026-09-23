// Labo 08 : les tests de config.go. Chaque test écrit son fichier dans
// t.TempDir() : rien à commiter, rien à nettoyer, jamais de collision.
// Lancé par : go test . (ton code) ou go test ./solution/ (la solution).
package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ecrire crée un fichier de test et rend son chemin. t.Helper() fait que
// l'échec éventuel est attribué à la ligne du test appelant, pas à celle-ci.
func ecrire(t *testing.T, contenu string) string {
	t.Helper()
	chemin := filepath.Join(t.TempDir(), "app.conf")
	if err := os.WriteFile(chemin, []byte(contenu), 0o644); err != nil {
		t.Fatalf("écrire le fichier de test : %v", err)
	}
	return chemin
}

func TestChargerConfigOK(t *testing.T) {
	cas := []struct {
		nom     string
		contenu string
		veut    Config
	}{
		{"complet", "hote = db-01\nport = 5432\nenv = prod\n", Config{"db-01", 5432, "prod"}},
		{"env par défaut", "hote=db-01\nport=5432\n", Config{"db-01", 5432, "dev"}},
		{"commentaires et lignes vides", "# sonde\n\nhote = db-01\n\n  port = 22  \n", Config{"db-01", 22, "dev"}},
		{"valeur avec =", "hote = db-01\nport = 5432\nenv=prod\n", Config{"db-01", 5432, "prod"}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got, err := ChargerConfig(ecrire(t, c.contenu))
			if err != nil {
				t.Fatalf("ChargerConfig : erreur inattendue : %v", err)
			}
			if got != c.veut {
				t.Errorf("ChargerConfig = %+v, attendu %+v", got, c.veut)
			}
		})
	}
}

func TestChargerConfigIntrouvable(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "absent.conf")
	_, err := ChargerConfig(chemin)
	if !errors.Is(err, ErrIntrouvable) {
		t.Fatalf("fichier absent : erreur %v, attendu ErrIntrouvable dans la chaîne", err)
	}
	// La chaîne doit se lire de haut en bas : charger config: lire CHEMIN: fichier introuvable
	veut := "charger config: lire " + chemin + ": fichier introuvable"
	if got := err.Error(); got != veut {
		t.Errorf("message = %q\nattendu   %q", got, veut)
	}
	// Une erreur de fichier absent n'est PAS une erreur de format.
	if errors.Is(err, ErrInvalide) {
		t.Errorf("un fichier absent ne doit pas être ErrInvalide")
	}
}

func TestChargerConfigParserInvalide(t *testing.T) {
	_, err := ChargerConfig(ecrire(t, "hote = db-01\nport 5432\n"))
	if !errors.Is(err, ErrInvalide) {
		t.Fatalf("ligne sans = : erreur %v, attendu ErrInvalide", err)
	}
	if !strings.Contains(err.Error(), "ligne 2") {
		t.Errorf("le message doit donner le numéro de ligne : %q", err)
	}
	var ev *ErreurValidation
	if errors.As(err, &ev) {
		t.Errorf("une erreur de syntaxe n'est pas une ErreurValidation : %v", err)
	}
}

func TestChargerConfigValidation(t *testing.T) {
	cas := []struct {
		nom     string
		contenu string
		champ   string
		raison  string
	}{
		{"hote manquant", "port = 5432\n", "hote", "manquant"},
		{"port manquant", "hote = db-01\n", "port", "manquant"},
		{"port pas un entier", "hote = db-01\nport = pgsql\n", "port", "pas un entier : pgsql"},
		{"port trop grand", "hote = db-01\nport = 70000\n", "port", "hors de 1..65535"},
		{"port zéro", "hote = db-01\nport = 0\n", "port", "hors de 1..65535"},
		{"env inconnu", "hote = db-01\nport = 5432\nenv = staging\n", "env", "doit être dev ou prod"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			_, err := ChargerConfig(ecrire(t, c.contenu))
			if err == nil {
				t.Fatal("erreur attendue, obtenu nil")
			}
			var ev *ErreurValidation
			if !errors.As(err, &ev) {
				t.Fatalf("erreur %v : pas d'ErreurValidation dans la chaîne", err)
			}
			if ev.Champ != c.champ || ev.Raison != c.raison {
				t.Errorf("ErreurValidation{%q, %q}, attendu {%q, %q}", ev.Champ, ev.Raison, c.champ, c.raison)
			}
			// Grâce à Unwrap, une erreur de validation EST un ErrInvalide.
			if !errors.Is(err, ErrInvalide) {
				t.Errorf("errors.Is(err, ErrInvalide) doit être vrai : ErreurValidation.Unwrap manque ?")
			}
			if !strings.HasPrefix(err.Error(), "charger config: valider: champ "+c.champ) {
				t.Errorf("message = %q, attendu le préfixe « charger config: valider: champ %s »", err, c.champ)
			}
		})
	}
}

func TestSecurise(t *testing.T) {
	t.Run("sans panic", func(t *testing.T) {
		appele := false
		if err := Securise(func() { appele = true }); err != nil {
			t.Errorf("Securise : erreur inattendue : %v", err)
		}
		if !appele {
			t.Error("Securise n'a pas appelé f")
		}
	})
	t.Run("panic de l'exécutif", func(t *testing.T) {
		err := Securise(func() {
			var m map[string]int
			m["x"] = 1
		})
		if err == nil {
			t.Fatal("Securise doit rendre une erreur après un panic")
		}
		if !strings.Contains(err.Error(), "nil map") {
			t.Errorf("le message doit contenir la cause du panic : %q", err)
		}
	})
	t.Run("panic avec une erreur", func(t *testing.T) {
		cause := errors.New("état impossible")
		err := Securise(func() { panic(cause) })
		if !errors.Is(err, cause) {
			t.Errorf("Securise doit emballer l'erreur du panic avec %%w : %v", err)
		}
	})
	t.Run("panic avec une chaîne", func(t *testing.T) {
		err := Securise(func() { panic("boum") })
		if err == nil || !strings.Contains(err.Error(), "boum") {
			t.Errorf("Securise : %v", err)
		}
	})
}
