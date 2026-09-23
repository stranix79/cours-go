// Labo 08 : le test de run, la fonction derrière main. On lui passe une
// slice d'arguments et un bytes.Buffer : pas de processus à lancer.
package main

import (
	"bytes"
	"errors"
	"testing"
)

func TestRun(t *testing.T) {
	chemin := ecrire(t, "hote = db-01\nport = 5432\nenv = prod\n")
	var out bytes.Buffer
	if err := run([]string{chemin}, &out); err != nil {
		t.Fatalf("run : %v", err)
	}
	if got := out.String(); got != "hote=db-01 port=5432 env=prod\n" {
		t.Errorf("run a écrit %q", got)
	}
	if err := run(nil, &out); !errors.Is(err, errUsage) {
		t.Errorf("run sans argument : erreur %v, attendu errUsage", err)
	}
	if err := run([]string{"/nulle/part.conf"}, &out); !errors.Is(err, ErrIntrouvable) {
		t.Errorf("run sur un fichier absent : erreur %v, attendu ErrIntrouvable", err)
	}
}
