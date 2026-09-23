// Labo 07 : le scénario de démonstration.
// Lancé par : go run .   (ou go run ./solution pour la version corrigée)
//
// main ne fait qu'assembler : une liste de formes, un rapport sur os.Stdout,
// puis Compter sur un texte fixe. Rien à modifier ici.
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	formes := []Forme{
		Rectangle{L: 3, H: 4},
		Cercle{R: 1},
		Triangle{A: 3, B: 4, C: 5},
	}
	if err := EcrireRapport(os.Stdout, formes); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}

	texte := "db-01 10.0.0.5 5432\ncache-01 10.0.0.9 6379\n\nweb-01\t10.0.0.12 443\n"
	lignes, mots, octets, err := Compter(strings.NewReader(texte))
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
	fmt.Printf("%d lignes, %d mots, %d octets\n", lignes, mots, octets)
}
