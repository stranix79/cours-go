// Labo 13, solution : une API REST de tâches en bibliothèque standard pure.
// Lancé par : go run ./solution [-addr :8080]   puis curl http://localhost:8080/taches
//
// main() ne fait que lire le drapeau -addr et installer l'arrêt sur Ctrl-C ;
// toute la logique est dans run(), testable sans signal ni processus.
package main

import (
	"context"   // le context qui porte l'ordre d'arrêt
	"errors"    // errors.Is pour reconnaître http.ErrServerClosed
	"flag"      // le drapeau -addr
	"log"       // journal sur stderr
	"net/http"  // http.Server
	"os"        // os.Interrupt, os.Exit
	"os/signal" // signal.NotifyContext
	"syscall"   // syscall.SIGTERM (ce que docker stop et systemd envoient)
	"time"      // les délais
)

// run démarre le serveur HTTP sur addr et le sert jusqu'à ce que ctx soit
// annulé ; il rend alors la main proprement et renvoie nil. Toute autre
// erreur (port occupé...) est renvoyée. Prendre un context et une adresse en
// paramètres, c'est ce qui permet au test de lancer run sur un port libre et
// de l'arrêter sans signal.
func run(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: nouveauRouteur(NouveauMagasin()),
		// Sans ReadHeaderTimeout, un client qui ouvre une connexion et
		// n'envoie jamais ses en-têtes occupe une goroutine pour toujours
		// (attaque Slowloris). golangci-lint le signale (gosec G112).
		ReadHeaderTimeout: 5 * time.Second,
	}

	// ListenAndServe bloque : on le lance dans une goroutine et on récupère
	// son erreur par un channel bufferisé (taille 1 : la goroutine peut
	// écrire et finir même si personne ne lit).
	erreurs := make(chan error, 1)
	go func() {
		erreurs <- srv.ListenAndServe()
	}()

	// On attend le premier des deux événements : l'ordre d'arrêt, ou une
	// erreur du serveur (typiquement le port déjà pris, immédiatement).
	select {
	case err := <-erreurs:
		// ListenAndServe ne renvoie jamais nil : soit une vraie erreur, soit
		// http.ErrServerClosed après Shutdown (impossible ici, on n'a pas
		// encore appelé Shutdown), donc c'est une vraie erreur.
		return err
	case <-ctx.Done():
	}

	// Shutdown ferme l'écoute, laisse finir les requêtes en cours, puis
	// rend la main. Le context borne cette attente à 5 s : au-delà, les
	// connexions restantes sont coupées et Shutdown renvoie l'erreur du
	// context. On part d'un context neuf : ctx est déjà annulé.
	arret, annule := context.WithTimeout(context.Background(), 5*time.Second)
	defer annule()
	if err := srv.Shutdown(arret); err != nil {
		return err
	}
	// Après Shutdown, ListenAndServe a renvoyé http.ErrServerClosed : on le
	// lit pour ne pas laisser la goroutine bloquée (elle ne l'est pas grâce
	// au buffer, mais on veut aussi ignorer proprement cette valeur).
	if err := <-erreurs; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func main() {
	addr := flag.String("addr", ":8080", "adresse d'écoute")
	flag.Parse()

	// signal.NotifyContext renvoie un context annulé au premier SIGINT
	// (Ctrl-C) ou SIGTERM (docker stop, systemctl stop). stop() rend le
	// comportement par défaut des signaux : un second Ctrl-C tue le
	// programme si l'arrêt propre traîne.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("écoute sur %s", *addr)
	if err := run(ctx, *addr); err != nil {
		log.Println("erreur :", err)
		os.Exit(1)
	}
	log.Println("arrêt propre")
}
