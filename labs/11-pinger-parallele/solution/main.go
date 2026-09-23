// Labo 11, solution : pinger, sonde des ports TCP en parallèle.
// Lancé par : go run ./solution [-timeout 2s] [-p 8] hote:port [hote:port...]
//
// main ne contient aucune logique : il lit les options, construit les cibles,
// branche le Ctrl-C sur un contexte, appelle SonderTous et affiche.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"
)

func main() {
	// Les options : un délai par cible et le nombre de sondes simultanées.
	timeout := flag.Duration("timeout", TimeoutDefaut, "délai de connexion par cible")
	parallelisme := flag.Int("p", 8, "nombre de sondes simultanées")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage : pinger [-timeout 2s] [-p 8] hote:port [hote:port...]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2) // erreur d'usage : code 2, comme flag lui-même
	}

	// Chaque argument restant est une cible "hote:port".
	var cibles []Cible
	for _, arg := range flag.Args() {
		c, err := ParseCible(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pinger : cible %q : %v\n", arg, err)
			os.Exit(2)
		}
		c.Timeout = *timeout
		cibles = append(cibles, c)
	}

	// Un Ctrl-C annule le contexte : les sondes en cours s'arrêtent, les
	// suivantes échouent avec "context canceled", et on affiche quand même
	// ce qu'on a. stop() libère le gestionnaire de signal à la sortie.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	debut := time.Now()
	res := SonderTous(ctx, cibles, *parallelisme)
	fmt.Print(FormatTableau(res))                         // le résultat : stdout
	fmt.Fprintf(os.Stderr, "%d cibles en %s\n", len(res), // le commentaire : stderr
		time.Since(debut).Round(time.Millisecond))

	// Code de sortie : 1 dès qu'une cible est injoignable, pour que
	// `pinger ... && deploy` fasse ce qu'on attend.
	for _, r := range res {
		if !r.OK {
			os.Exit(1)
		}
	}
}
