// Labo 15 : l'interface Sonde et sa fausse implémentation pour les tests.
// Utilisé par : metrics.go (collecter), main.go (sondeSysteme), les tests (fausseSonde).
package main

import "fmt"

// Espace est ce qu'une sonde renvoie pour un point de montage, en octets.
// Libre est l'espace disponible pour un utilisateur non root (comme df).
type Espace struct {
	Total uint64
	Libre uint64
}

// Sonde sait mesurer un point de montage. Une seule méthode, exprès.
type Sonde interface {
	Statfs(chemin string) (Espace, error)
}

// fausseSonde est une Sonde en mémoire : une table chemin -> Espace.
type fausseSonde map[string]Espace

// Statfs rend l'entrée de la table, ou une erreur si le chemin est inconnu.
// (Déjà écrite : les tests s'en servent dès le premier TODO.)
func (f fausseSonde) Statfs(chemin string) (Espace, error) {
	e, ok := f[chemin]
	if !ok {
		return Espace{}, fmt.Errorf("point de montage inconnu : %s", chemin)
	}
	return e, nil
}
