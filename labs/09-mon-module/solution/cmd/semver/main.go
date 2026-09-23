// semver est la ligne de commande du labo 09 :
//
//	semver compare 1.2.3 1.10.0        affiche <, = ou >
//	semver bump minor 1.2.3            affiche 1.3.0
//	semver check 1.5.0 "^1.2.0"        affiche oui ou non
//
// Lancé par : go run ./solution/cmd/semver compare 1.2.3 1.10.0
// (depuis labs/09-mon-module). Le paquet internal/semver fait tout le travail ;
// ce fichier ne fait que lire les arguments et afficher.
package main

import (
	"fmt"
	"io"
	"os"

	"cours-go/labs/09-mon-module/solution/internal/semver"
)

func main() {
	// Motif run() error : main ne fait que traduire l'erreur en code de sortie.
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

// run est testable : arguments en slice, sortie dans un io.Writer.
func run(args []string, w io.Writer) error {
	if len(args) != 3 {
		return fmt.Errorf("usage : semver compare A B | bump PARTIE V | check V CONTRAINTE")
	}
	switch args[0] {
	case "compare":
		a, err := semver.Parse(args[1])
		if err != nil {
			return err
		}
		b, err := semver.Parse(args[2])
		if err != nil {
			return err
		}
		// Un tableau indexé par Compare+1 : -1 -> "<", 0 -> "=", 1 -> ">".
		symboles := [...]string{"<", "=", ">"}
		fmt.Fprintln(w, symboles[semver.Compare(a, b)+1])
	case "bump":
		v, err := semver.Parse(args[2])
		if err != nil {
			return err
		}
		suivante, err := semver.Bump(v, args[1])
		if err != nil {
			return err
		}
		fmt.Fprintln(w, suivante)
	case "check":
		v, err := semver.Parse(args[1])
		if err != nil {
			return err
		}
		ok, err := semver.Compatible(v, args[2])
		if err != nil {
			return err
		}
		if ok {
			fmt.Fprintln(w, "oui")
		} else {
			fmt.Fprintln(w, "non")
		}
	default:
		return fmt.Errorf("commande inconnue %q (compare, bump, check)", args[0])
	}
	return nil
}
