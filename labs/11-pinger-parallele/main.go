// Labo 11 : pinger, sonde des ports TCP en parallèle.
// Lancé par : go run . [-timeout 2s] [-p 8] hote:port [hote:port...]
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
	timeout := flag.Duration("timeout", TimeoutDefaut, "délai de connexion par cible")
	parallelisme := flag.Int("p", 8, "nombre de sondes simultanées")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage : pinger [-timeout 2s] [-p 8] hote:port [hote:port...]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

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
	// suivantes échouent avec "context canceled", et on affiche quand même.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	debut := time.Now()
	res := SonderTous(ctx, cibles, *parallelisme)
	fmt.Print(FormatTableau(res))
	fmt.Fprintf(os.Stderr, "%d cibles en %s\n", len(res), time.Since(debut).Round(time.Millisecond))

	for _, r := range res {
		if !r.OK {
			os.Exit(1) // au moins une cible injoignable : code 1 pour le shell
		}
	}
}
