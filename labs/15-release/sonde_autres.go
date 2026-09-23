//go:build !unix

// Labo 15 : la sonde de remplacement pour les systèmes non Unix (Windows...).
// Utilisé par : main.go, seulement quand GOOS n'est pas un Unix. Sans ce
// fichier, GOOS=windows go build échouerait : sondeSysteme n'existerait pas.
// (Déjà écrit, rien à faire ici.)
package main

import "errors"

type sondeSysteme struct{}

func (sondeSysteme) Statfs(chemin string) (Espace, error) {
	return Espace{}, errors.New("statfs non disponible sur ce système : " + chemin)
}
