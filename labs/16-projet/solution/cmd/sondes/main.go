// Labo 16, solution : sondes, le service de supervision.
// Lancé par : go run ./solution/cmd/sondes -config sondes.json   (depuis labs/16-projet)
//
// Quoi : le point d'entrée. main lit les options et construit le logger ; run
// assemble les paquets (config → sonde → api), lance la boucle de sondes et le
// serveur HTTP, et orchestre l'arrêt propre. Toute la logique est ailleurs :
// ce fichier ne fait que brancher.
package main

import (
	"context"   // le contexte racine, annulé par le signal
	"errors"    // errors.Is pour reconnaître http.ErrServerClosed
	"flag"      // options de la ligne de commande
	"fmt"       // messages sur stderr
	"log/slog"  // journal JSON
	"net/http"  // http.Server
	"os"        // os.Exit, os.Getenv, os.Stdout
	"os/signal" // signal.NotifyContext
	"sync"      // WaitGroup pour attendre la fin de la boucle
	"syscall"   // syscall.SIGTERM (Docker, systemd)
	"time"      // délais

	"cours-go/labs/16-projet/solution/internal/api"
	"cours-go/labs/16-projet/solution/internal/config"
	"cours-go/labs/16-projet/solution/internal/sonde"
)

// version est remplacée à la compilation par -ldflags "-X main.version=1.2.3"
// (voir le Makefile). Sans ça, "dev".
var version = "dev"

func main() {
	// Le chemin de la config : l'option -config, sinon la variable SONDES_CONFIG,
	// sinon sondes.json. Le flag l'emporte sur l'environnement : c'est ce que
	// fait tout outil de la maison (Prometheus, Docker...).
	defautConfig := os.Getenv("SONDES_CONFIG")
	if defautConfig == "" {
		defautConfig = "sondes.json"
	}
	chemin := flag.String("config", defautConfig, "fichier JSON des cibles")
	ecoute := flag.String("ecoute", "", "adresse d'écoute HTTP (remplace celle de la config)")
	flag.Parse()

	// Un logger JSON sur stdout : une ligne par événement, que Docker, systemd
	// ou Loki collectent. slog.SetDefault le rend disponible partout.
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	cfg, err := config.Charger(*chemin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sondes :", err)
		os.Exit(2) // 2 : erreur d'usage, la convention des outils Unix
	}
	if *ecoute != "" {
		cfg.Ecoute = *ecoute
	}

	// Le contexte racine est annulé au premier SIGINT (Ctrl-C) ou SIGTERM
	// (docker stop, systemctl stop). stop() restaure le comportement par défaut
	// des signaux : un second Ctrl-C tue le programme si l'arrêt propre traîne.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg, log); err != nil {
		log.Error("arrêt sur erreur", "erreur", err)
		os.Exit(1)
	}
}

// run est séparé de main pour être testable et pour que les defer s'exécutent
// (os.Exit court-circuite les defer). Il rend la main quand ctx est annulé et
// que tout est arrêté, ou sur une erreur fatale du serveur HTTP.
func run(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	etat := sonde.NouvelEtat()
	moteur := &sonde.Moteur{
		Sondeur:    sonde.NouveauSondeur(cfg.Timeout.Duration()),
		Cibles:     cfg.Cibles,
		Intervalle: cfg.Intervalle.Duration(),
		Workers:    cfg.Workers,
		Etat:       etat,
		Log:        log,
	}

	// La boucle de sondes dans sa goroutine ; le WaitGroup permet d'attendre
	// qu'elle ait vraiment fini avant de rendre la main.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		moteur.Boucle(ctx)
	}()

	srv := &http.Server{
		Addr:              cfg.Ecoute,
		Handler:           api.NouveauMux(etat, version, log),
		ReadHeaderTimeout: 5 * time.Second, // contre les clients qui ouvrent et se taisent (slowloris)
	}
	// ListenAndServe bloque : on le lance dans une goroutine et on récupère
	// son erreur par un channel bufferisé (la goroutine ne doit pas rester
	// bloquée sur l'envoi si personne ne lit).
	erreurs := make(chan error, 1)
	go func() {
		log.Info("démarrage", "version", version, "ecoute", cfg.Ecoute,
			"cibles", len(cfg.Cibles), "intervalle", cfg.Intervalle.Duration().String(), "workers", cfg.Workers)
		erreurs <- srv.ListenAndServe()
	}()

	// On attend la première des deux choses : le signal, ou une erreur du
	// serveur (port déjà pris, typiquement).
	select {
	case err := <-erreurs:
		return fmt.Errorf("serveur HTTP : %w", err)
	case <-ctx.Done():
	}

	// Arrêt propre : Shutdown ferme l'écoute, laisse finir les requêtes en
	// cours (au plus 5 s), puis rend la main. Un contexte NEUF : celui de la
	// fonction est déjà annulé, Shutdown n'attendrait rien.
	log.Info("arrêt demandé")
	ctxArret, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := srv.Shutdown(ctxArret)
	wg.Wait() // la boucle a vu ctx.Done() et a fini son tour en cours
	// ListenAndServe renvoie ErrServerClosed après Shutdown : ce n'est pas une
	// erreur, c'est la preuve que l'arrêt a bien eu lieu.
	if errServe := <-erreurs; !errors.Is(errServe, http.ErrServerClosed) {
		return errServe
	}
	log.Info("arrêté", "verifications", etat.Total())
	return err
}
