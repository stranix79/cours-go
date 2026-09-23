//go:build !unix

// Labo 15, solution : la sonde de remplacement pour les systèmes non Unix.
// Utilisé par : main.go, seulement quand GOOS n'est pas un Unix (windows, js...).
//
// Sans ce fichier, `GOOS=windows go build` échouerait : sondeSysteme ne
// serait définie nulle part. On fournit le même type avec la même méthode,
// qui renvoie une erreur honnête. L'exporter compile et démarre partout ;
// il ne mesure que là où il sait le faire.
package main

import "errors"

type sondeSysteme struct{}

func (sondeSysteme) Statfs(chemin string) (Espace, error) {
	return Espace{}, errors.New("statfs non disponible sur ce système : " + chemin)
}
