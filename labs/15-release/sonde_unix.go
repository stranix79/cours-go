//go:build unix

// Labo 15 : la vraie sonde, compilée seulement sur Linux, macOS et les BSD
// (la ligne //go:build unix ci-dessus est une contrainte de build).
// Utilisé par : main.go.
package main

import (
	"errors"
)

// sondeSysteme interroge le noyau avec l'appel système statfs(2).
type sondeSysteme struct{}

// TODO 7 : déclare `var st syscall.Statfs_t`, appelle syscall.Statfs(chemin, &st),
// enveloppe l'erreur avec %w (fmt.Errorf), puis convertis les blocs en octets :
// Total = uint64(st.Bsize) * st.Blocks, Libre = uint64(st.Bsize) * st.Bavail.
// Les conversions en uint64 sont obligatoires : Bsize n'a pas le même type
// sur macOS (uint32) et sur Linux (int64). Il faudra importer "fmt" et "syscall".
func (sondeSysteme) Statfs(chemin string) (Espace, error) {
	return Espace{}, errors.New("TODO")
}
