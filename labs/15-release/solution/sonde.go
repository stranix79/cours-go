// Labo 15, solution : l'interface Sonde et sa fausse implémentation.
// Utilisé par : metrics.go (collecter), main.go (sondeSysteme), les tests (fausseSonde).
//
// Pourquoi une interface ? Parce que l'appel système Statfs dépend de la
// machine : impossible d'écrire un test dont la sortie attendue est fixe.
// Avec l'interface, le code qui formate les métriques ne sait pas d'où
// viennent les chiffres, et les tests lui donnent des chiffres connus.
package main

import "fmt"

// Espace est ce qu'une sonde renvoie pour un point de montage : la taille
// totale et l'espace libre, en octets. Libre est l'espace disponible pour
// un utilisateur non root (Bavail, pas Bfree), ce que df affiche aussi.
type Espace struct {
	Total uint64
	Libre uint64
}

// Sonde sait mesurer un point de montage. Une seule méthode : c'est la taille
// idéale d'une interface Go (comme io.Reader ou fmt.Stringer).
type Sonde interface {
	Statfs(chemin string) (Espace, error)
}

// fausseSonde est une Sonde en mémoire : une table chemin -> Espace. Elle
// sert aux tests et à la sortie attendue du README, qui doit être la même
// sur toutes les machines.
type fausseSonde map[string]Espace

// Statfs rend l'entrée de la table, ou une erreur si le chemin est inconnu,
// comme le ferait le vrai appel système sur un chemin qui n'existe pas.
func (f fausseSonde) Statfs(chemin string) (Espace, error) {
	e, ok := f[chemin]
	if !ok {
		return Espace{}, fmt.Errorf("point de montage inconnu : %s", chemin)
	}
	return e, nil
}
