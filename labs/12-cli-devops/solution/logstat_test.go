package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// journal est un petit log de test : 5 lignes, 3 niveaux, des doublons.
const journal = `2026-09-23T10:00:00Z INFO démarrage
2026-09-23T10:00:01Z ERROR connexion refusée db1:5432

2026-09-23T10:00:02Z INFO requête servie
2026-09-23T10:00:03Z WARN réponse lente web1
2026-09-23T10:00:04Z ERROR connexion refusée db1:5432
`

func entreesTest(t *testing.T) []Entree {
	t.Helper()
	entrees, err := Lire(strings.NewReader(journal))
	if err != nil {
		t.Fatalf("Lire : %v", err)
	}
	return entrees
}

func TestParseLigne(t *testing.T) {
	cas := []struct {
		nom    string
		ligne  string
		veut   Entree
		erreur bool
	}{
		{
			"ligne complète",
			"2026-09-23T10:00:06Z ERROR connexion refusée db1:5432",
			Entree{time.Date(2026, 9, 23, 10, 0, 6, 0, time.UTC), "ERROR", "connexion refusée db1:5432"},
			false,
		},
		{
			"message d'un mot",
			"2026-09-23T10:00:00Z INFO démarrage",
			Entree{time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC), "INFO", "démarrage"},
			false,
		},
		{"sans message", "2026-09-23T10:00:00Z INFO", Entree{}, true},
		{"ligne vide", "", Entree{}, true},
		{"horodatage invalide", "hier INFO démarrage", Entree{}, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got, err := ParseLigne(c.ligne)
			if (err != nil) != c.erreur {
				t.Fatalf("erreur attendue : %v, obtenue : %v", c.erreur, err)
			}
			if c.erreur {
				return
			}
			if !got.Horodatage.Equal(c.veut.Horodatage) || got.Niveau != c.veut.Niveau || got.Message != c.veut.Message {
				t.Errorf("attendu %+v, obtenu %+v", c.veut, got)
			}
		})
	}
}

func TestLire(t *testing.T) {
	t.Run("lignes vides ignorées", func(t *testing.T) {
		entrees := entreesTest(t)
		if len(entrees) != 5 {
			t.Fatalf("attendu 5 entrées, obtenu %d", len(entrees))
		}
		if entrees[4].Message != "connexion refusée db1:5432" {
			t.Errorf("dernière entrée inattendue : %+v", entrees[4])
		}
	})
	t.Run("ligne malformée numérotée", func(t *testing.T) {
		_, err := Lire(strings.NewReader("2026-09-23T10:00:00Z INFO ok\n\ncassée\n"))
		if err == nil || !strings.Contains(err.Error(), "ligne 3") {
			t.Fatalf("attendu une erreur mentionnant « ligne 3 », obtenu %v", err)
		}
	})
	t.Run("flux vide", func(t *testing.T) {
		entrees, err := Lire(strings.NewReader(""))
		if err != nil || len(entrees) != 0 {
			t.Errorf("attendu 0 entrée sans erreur, obtenu %d, %v", len(entrees), err)
		}
	})
}

func TestCompter(t *testing.T) {
	got := Compter(entreesTest(t))
	veut := map[string]int{"INFO": 2, "ERROR": 2, "WARN": 1}
	if len(got) != len(veut) {
		t.Fatalf("attendu %v, obtenu %v", veut, got)
	}
	for k, v := range veut {
		if got[k] != v {
			t.Errorf("%s : attendu %d, obtenu %d", k, v, got[k])
		}
	}
}

func TestTop(t *testing.T) {
	entrees := entreesTest(t)
	cas := []struct {
		nom  string
		n    int
		veut []Frequence
	}{
		{"le premier", 1, []Frequence{{"connexion refusée db1:5432", 2}}},
		{"égalité tranchée par l'alphabet", 3, []Frequence{
			{"connexion refusée db1:5432", 2}, {"démarrage", 1}, {"requête servie", 1},
		}},
		{"n plus grand que le nombre de messages", 10, []Frequence{
			{"connexion refusée db1:5432", 2}, {"démarrage", 1}, {"requête servie", 1}, {"réponse lente web1", 1},
		}},
		{"n nul", 0, []Frequence{}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got := Top(entrees, c.n)
			if len(got) != len(c.veut) {
				t.Fatalf("attendu %v, obtenu %v", c.veut, got)
			}
			for i := range c.veut {
				if got[i] != c.veut[i] {
					t.Errorf("position %d : attendu %v, obtenu %v", i, c.veut[i], got[i])
				}
			}
		})
	}
}

func TestFormatCompte(t *testing.T) {
	got := FormatCompte(map[string]int{"ERROR": 2, "INFO": 12, "TRACE": 1, "WARN": 5, "AUDIT": 3})
	veut := "INFO  12\nWARN  5\nERROR 2\nAUDIT 3\nTRACE 1\n"
	if got != veut {
		t.Errorf("attendu :\n%s\nobtenu :\n%s", veut, got)
	}
}

func TestEncoderJSON(t *testing.T) {
	var buf bytes.Buffer
	s := Stats{Total: 3, ParNiveau: map[string]int{"INFO": 2, "ERROR": 1}, Top: []Frequence{{"ok", 2}}}
	if err := EncoderJSON(&buf, s); err != nil {
		t.Fatalf("EncoderJSON : %v", err)
	}
	veut := `{
  "total": 3,
  "par_niveau": {
    "ERROR": 1,
    "INFO": 2
  },
  "top": [
    {
      "message": "ok",
      "nombre": 2
    }
  ]
}
`
	if buf.String() != veut {
		t.Errorf("attendu :\n%s\nobtenu :\n%s", veut, buf.String())
	}
}

// lancer appelle run avec des tampons et rend code, stdout, stderr.
func lancer(args []string, stdin string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

func TestRun(t *testing.T) {
	fichier := filepath.Join(t.TempDir(), "app.log")
	if err := os.WriteFile(fichier, []byte(journal), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("sans argument : usage et code 2", func(t *testing.T) {
		code, out, errs := lancer(nil, "")
		if code != 2 || out != "" || !strings.Contains(errs, "usage") {
			t.Errorf("code=%d stdout=%q stderr=%q", code, out, errs)
		}
	})
	t.Run("sous-commande inconnue : code 2", func(t *testing.T) {
		code, _, errs := lancer([]string{"grep"}, "")
		if code != 2 || !strings.Contains(errs, "grep") {
			t.Errorf("code=%d stderr=%q", code, errs)
		}
	})
	t.Run("option inconnue : code 2", func(t *testing.T) {
		code, _, _ := lancer([]string{"count", "-x"}, "")
		if code != 2 {
			t.Errorf("code=%d", code)
		}
	})
	t.Run("count sur un fichier", func(t *testing.T) {
		code, out, errs := lancer([]string{"count", fichier}, "")
		if code != 0 || errs != "" {
			t.Fatalf("code=%d stderr=%q", code, errs)
		}
		if veut := "INFO  2\nWARN  1\nERROR 2\n"; out != veut {
			t.Errorf("attendu %q, obtenu %q", veut, out)
		}
	})
	t.Run("top -n 2 sur stdin", func(t *testing.T) {
		code, out, _ := lancer([]string{"top", "-n", "2"}, journal)
		if code != 0 {
			t.Fatalf("code=%d", code)
		}
		if veut := "    2  connexion refusée db1:5432\n    1  démarrage\n"; out != veut {
			t.Errorf("attendu %q, obtenu %q", veut, out)
		}
	})
	t.Run("json", func(t *testing.T) {
		code, out, _ := lancer([]string{"json", "-n", "1", fichier}, "")
		if code != 0 {
			t.Fatalf("code=%d", code)
		}
		for _, attendu := range []string{`"total": 5`, `"ERROR": 2`, `"message": "connexion refusée db1:5432"`} {
			if !strings.Contains(out, attendu) {
				t.Errorf("sortie JSON sans %s :\n%s", attendu, out)
			}
		}
		if strings.Count(out, `"message"`) != 1 {
			t.Errorf("-n 1 doit limiter le top à un message :\n%s", out)
		}
	})
	t.Run("fichier absent : code 1", func(t *testing.T) {
		code, out, errs := lancer([]string{"count", filepath.Join(t.TempDir(), "nexiste.log")}, "")
		if code != 1 || out != "" || errs == "" {
			t.Errorf("code=%d stdout=%q stderr=%q", code, out, errs)
		}
	})
	t.Run("ligne malformée : code 1 et numéro de ligne", func(t *testing.T) {
		code, _, errs := lancer([]string{"count"}, "2026-09-23T10:00:00Z INFO ok\npas une ligne de log\n")
		if code != 1 || !strings.Contains(errs, "ligne 2") {
			t.Errorf("code=%d stderr=%q", code, errs)
		}
	})
	t.Run("-v journalise sur stderr", func(t *testing.T) {
		code, out, errs := lancer([]string{"count", "-v", fichier}, "")
		if code != 0 || !strings.Contains(errs, "DEBUG") || strings.Contains(out, "DEBUG") {
			t.Errorf("code=%d stdout=%q stderr=%q", code, out, errs)
		}
	})
}
