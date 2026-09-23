// Labo 08, solution : le programme qui charge une configuration.
// Lancé par : go run ./solution exemple.conf   (depuis labs/08-robuste)
//
// main est minuscule : il appelle run et transforme son erreur en code de
// sortie. Les defer de run s'exécutent normalement ; os.Exit n'est appelé
// qu'après, dans main. C'est le motif « run() error » du chapitre 8.
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
		// Code 2 pour une erreur d'usage, 1 pour le reste : convention Unix.
		if errors.Is(err, errUsage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

var errUsage = errors.New("usage : robuste FICHIER.conf")

// run contient toute la logique : testable, puisqu'elle prend ses arguments
// et un io.Writer au lieu de lire os.Args et d'écrire sur os.Stdout.
func run(args []string, w io.Writer) error {
	if len(args) != 1 {
		return errUsage
	}
	cfg, err := ChargerConfig(args[0])
	if err != nil {
		// Ici on pourrait distinguer les cas : créer une config par défaut si
		// errors.Is(err, ErrIntrouvable), afficher le champ fautif si
		// errors.As(err, &ev)... Pour ce labo, on remonte telle quelle.
		return err
	}
	fmt.Fprintf(w, "hote=%s port=%d env=%s\n", cfg.Hote, cfg.Port, cfg.Env)
	return nil
}
