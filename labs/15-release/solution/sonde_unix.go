//go:build unix

// Labo 15, solution : la vraie sonde, pour Linux, macOS et les BSD.
// Utilisé par : main.go (sondeSysteme{} passée à run).
//
// La ligne //go:build unix en tête est une CONTRAINTE DE BUILD : ce fichier
// n'est compilé que si GOOS est un Unix (linux, darwin, freebsd...). Sur
// Windows, c'est sonde_autres.go qui prend le relais. Le nom du fichier
// (_unix) n'a pas d'effet ; seule la ligne //go:build compte. (Les suffixes
// _linux.go ou _windows.go, eux, sont reconnus par le nom.)
package main

import (
	"fmt"
	"syscall" // l'appel système statfs(2), exposé par la bibliothèque standard
)

// sondeSysteme interroge le noyau. Struct vide : elle n'a pas d'état, mais
// elle a une méthode, ce qui suffit pour satisfaire l'interface Sonde.
type sondeSysteme struct{}

// Statfs appelle statfs(2) et convertit les blocs en octets. Les champs
// Bsize, Blocks et Bavail existent sur tous les Unix, avec des types qui
// varient (Bsize est uint32 sur macOS, int64 sur Linux) : d'où les
// conversions explicites en uint64 avant la multiplication.
func (sondeSysteme) Statfs(chemin string) (Espace, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(chemin, &st); err != nil {
		// %w garde l'erreur système (ENOENT, EACCES...) accessible avec errors.Is.
		return Espace{}, fmt.Errorf("statfs %s : %w", chemin, err)
	}
	taille := uint64(st.Bsize)
	return Espace{
		Total: taille * st.Blocks,
		Libre: taille * st.Bavail,
	}, nil
}
