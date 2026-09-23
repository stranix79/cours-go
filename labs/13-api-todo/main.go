// Labo 13 : une API REST de tâches en bibliothèque standard pure.
// Lancé par : go run . [-addr :8080]   puis curl http://localhost:8080/taches
//
// main() ne fait que lire le drapeau -addr et installer l'arrêt sur Ctrl-C ;
// toute la logique est dans run(), testable sans signal ni processus.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// run démarre le serveur HTTP sur addr et le sert jusqu'à ce que ctx soit
// annulé ; il rend alors la main proprement (les requêtes en cours finissent,
// délai maximal 5 s) et renvoie nil. Toute autre erreur (port occupé...) est
// renvoyée.
func run(ctx context.Context, addr string) error {
	// TODO 12 : construis un http.Server{Addr: addr, Handler:
	// nouveauRouteur(NouveauMagasin()), ReadHeaderTimeout: 5 * time.Second},
	// lance ListenAndServe dans une goroutine, attends ctx.Done() ou une
	// erreur, puis Shutdown avec un context à délai.
	return errors.New("TODO")
}

func main() {
	addr := flag.String("addr", ":8080", "adresse d'écoute")
	flag.Parse()

	// signal.NotifyContext renvoie un context annulé au premier SIGINT ou
	// SIGTERM : c'est ce qui déclenche l'arrêt propre dans run.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("écoute sur %s", *addr)
	if err := run(ctx, *addr); err != nil {
		log.Println("erreur :", err)
		os.Exit(1)
	}
	log.Println("arrêt propre")
}
