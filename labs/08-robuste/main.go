// Labo 08 : le programme qui charge une configuration.
// Lancé par : go run . exemple.conf   (ou go run ./solution exemple.conf)
//
// main est minuscule : il appelle run et transforme son erreur en code de
// sortie. Rien à modifier ici.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		if errors.Is(err, errUsage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

var errUsage = errors.New("usage : robuste FICHIER.conf")

// run contient la logique : testable, puisqu'elle prend ses arguments et un
// io.Writer au lieu de lire os.Args et d'écrire sur os.Stdout.
func run(args []string, w io.Writer) error {
	if len(args) != 1 {
		return errUsage
	}
	cfg, err := ChargerConfig(args[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "hote=%s port=%d env=%s\n", cfg.Hote, cfg.Port, cfg.Env)
	return nil
}
