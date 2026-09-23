//go:build unix

// Labo 15 : test de la vraie sonde, seulement sur Unix (même contrainte de
// build que sonde_unix.go). Les valeurs dépendent de la machine : on vérifie
// des invariants, pas des chiffres.
package main

import "testing"

func TestSondeSysteme(t *testing.T) {
	e, err := sondeSysteme{}.Statfs("/")
	if err != nil {
		t.Fatalf("Statfs(/) : %v", err)
	}
	if e.Total == 0 {
		t.Error("Total doit être strictement positif pour /")
	}
	if e.Libre > e.Total {
		t.Errorf("Libre (%d) ne peut pas dépasser Total (%d)", e.Libre, e.Total)
	}
	if _, err := (sondeSysteme{}).Statfs("/ce/chemin/n/existe/pas"); err == nil {
		t.Error("un chemin inexistant doit renvoyer une erreur")
	}
}
