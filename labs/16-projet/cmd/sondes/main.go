// Labo 16 : sondes, le service de supervision.
// Lancé par : go run ./cmd/sondes -config sondes.json   (depuis labs/16-projet)
//
// main lit les options et construit le logger ; run assemble les paquets,
// lance la boucle de sondes et le serveur HTTP, et orchestre l'arrêt propre.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"cours-go/labs/16-projet/internal/config"
)

// version est remplacée à la compilation par -ldflags "-X main.version=..."
// (voir le Makefile).
var version = "dev"

func main() {
	// Le chemin de la config : l'option -config, sinon SONDES_CONFIG, sinon
	// sondes.json.
	defautConfig := os.Getenv("SONDES_CONFIG")
	if defautConfig == "" {
		defautConfig = "sondes.json"
	}
	chemin := flag.String("config", defautConfig, "fichier JSON des cibles")
	ecoute := flag.String("ecoute", "", "adresse d'écoute HTTP (remplace celle de la config)")
	flag.Parse()

	// Journal JSON sur stdout, une ligne par événement.
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	cfg, err := config.Charger(*chemin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sondes :", err)
		os.Exit(2)
	}
	if *ecoute != "" {
		cfg.Ecoute = *ecoute
	}

	// Contexte annulé au premier SIGINT ou SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg, log); err != nil {
		log.Error("arrêt sur erreur", "erreur", err)
		os.Exit(1)
	}
}

// run rend la main quand ctx est annulé et que tout est arrêté, ou sur une
// erreur fatale du serveur HTTP. À faire, dans l'ordre :
//  1. etat := sonde.NouvelEtat() ; un sonde.Moteur avec sonde.NouveauSondeur,
//     les cibles, l'intervalle, les workers, l'état et le log ;
//  2. lancer moteur.Boucle(ctx) dans une goroutine, suivie par un WaitGroup ;
//  3. un http.Server{Addr: cfg.Ecoute, Handler: api.NouveauMux(etat, version,
//     log), ReadHeaderTimeout: 5 * time.Second} ; ListenAndServe dans une
//     goroutine qui envoie son erreur dans un channel bufferisé (taille 1) ;
//  4. select sur ce channel (erreur fatale : port pris) et sur ctx.Done() ;
//  5. à l'annulation : srv.Shutdown avec un contexte NEUF (context.Background
//     + WithTimeout 5 s), wg.Wait(), lire l'erreur de ListenAndServe et
//     ignorer http.ErrServerClosed (errors.Is), journaliser "arrêté".
func run(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	// TODO 20
	_, _, _ = ctx, cfg, log
	return errors.New("TODO")
}
