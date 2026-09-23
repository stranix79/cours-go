// Labo 15 : diskexporter, un mini exporter Prometheus pour l'espace disque.
// Lancé par : go run . -points /,/tmp   puis   curl localhost:9101/metrics
// Ou bien  : make build && bin/diskexporter -version
//
// Ce fichier : le point d'entrée, les options, le routeur et run().
// Les tests (main_test.go) appellent decouperPoints, nouveauRouteur et run.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// version est écrasée à l'édition de liens : go build -ldflags "-X main.version=1.2.3".
// Ne la transforme pas en constante, -X ne saurait plus la remplacer.
var version = "dev"

func main() {
	addr := flag.String("addr", ":9101", "adresse d'écoute (hôte:port)")
	points := flag.String("points", "/", "points de montage à mesurer, séparés par des virgules")
	afficherVersion := flag.Bool("version", false, "affiche la version et quitte")
	flag.Parse()

	if *afficherVersion {
		fmt.Println("diskexporter", version)
		return
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, *addr, sondeSysteme{}, decouperPoints(*points)); err != nil {
		slog.Error("arrêt sur erreur", "err", err)
		os.Exit(1)
	}
}

// decouperPoints transforme "/, /tmp,,/var" en ["/", "/tmp", "/var"].
// TODO 1 : strings.Split sur la virgule, strings.TrimSpace sur chaque morceau,
// ignorer les morceaux vides. Une chaîne vide donne un slice nil.
func decouperPoints(liste string) []string {
	return nil
}

// nouveauRouteur assemble les routes : "GET /metrics" et "GET /healthz".
// TODO 5 : http.NewServeMux(), mux.Handle avec les motifs à méthode (Go 1.22+),
// handlerMetrics(s, points, version) et handlerHealthz(). Renvoie le mux.
func nouveauRouteur(s Sonde, points []string) http.Handler {
	return http.NewServeMux()
}

// run fait tourner le serveur sur addr jusqu'à l'annulation de ctx, puis
// l'arrête proprement. Renvoie nil après un arrêt demandé, l'erreur sinon.
// TODO 6 : un http.Server avec Addr, Handler et ReadHeaderTimeout ; ListenAndServe
// dans une goroutine qui envoie son erreur dans un channel bufferisé ; un select
// entre ce channel (erreur de démarrage : la renvoyer) et ctx.Done() ; puis
// srv.Shutdown avec un contexte neuf de 5 s (context.WithTimeout sur
// context.Background(), pas sur ctx qui est déjà annulé).
func run(ctx context.Context, addr string, s Sonde, points []string) error {
	return errors.New("TODO")
}
