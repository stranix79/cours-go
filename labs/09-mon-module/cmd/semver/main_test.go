// Labo 09 : test de la ligne de commande via run, sans lancer de processus.
package main

import (
	"bytes"
	"testing"
)

func TestRun(t *testing.T) {
	cas := []struct {
		nom    string
		args   []string
		veut   string
		erreur bool
	}{
		{"compare <", []string{"compare", "1.2.3", "1.10.0"}, "<\n", false},
		{"compare =", []string{"compare", "1.2.3", "v1.2.3"}, "=\n", false},
		{"compare >", []string{"compare", "2.0.0", "1.9.9"}, ">\n", false},
		{"bump minor", []string{"bump", "minor", "1.2.3"}, "1.3.0\n", false},
		{"check oui", []string{"check", "1.5.0", "^1.2.0"}, "oui\n", false},
		{"check non", []string{"check", "2.0.0", "^1.2.0"}, "non\n", false},
		{"pas assez d'arguments", []string{"compare", "1.0.0"}, "", true},
		{"commande inconnue", []string{"diff", "1.0.0", "2.0.0"}, "", true},
		{"version invalide", []string{"compare", "1.0", "2.0.0"}, "", true},
		{"partie inconnue", []string{"bump", "mega", "1.0.0"}, "", true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			var out bytes.Buffer
			err := run(c.args, &out)
			if c.erreur {
				if err == nil {
					t.Fatalf("run(%v) : erreur attendue", c.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("run(%v) : %v", c.args, err)
			}
			if got := out.String(); got != c.veut {
				t.Errorf("run(%v) a écrit %q, attendu %q", c.args, got, c.veut)
			}
		})
	}
}
