// Labo 15, solution : diskexporter, un mini exporter Prometheus pour l'espace disque.
// Lancé par : go run ./solution -points /,/tmp   (depuis labs/15-release)
// Ou bien  : make build SRC=./solution && bin/diskexporter -addr :9101
//
// Ce fichier contient le point d'entrée (main), la lecture des options, le
// routeur HTTP et la fonction run qui fait tourner le serveur jusqu'à
// l'annulation du contexte. Tout ce qui est testable est hors de main() :
// decouperPoints, nouveauRouteur et run sont appelées par les tests.
package main

import (
	"context"   // annulation et délais : le contexte qui arrête le serveur
	"errors"    // errors.Is pour reconnaître http.ErrServerClosed
	"flag"      // options de la ligne de commande (-addr, -points, -version)
	"fmt"       // formatage des messages et des erreurs
	"log/slog"  // journal structuré : une ligne JSON par événement
	"net/http"  // le serveur HTTP de la bibliothèque standard
	"os"        // os.Stdout, os.Exit, os.Interrupt
	"os/signal" // signal.NotifyContext : SIGINT/SIGTERM annulent le contexte
	"strings"   // découpage de la liste des points de montage
	"syscall"   // syscall.SIGTERM, le signal que Docker et systemd envoient
	"time"      // délais du serveur et de l'arrêt
)

// version est remplacée à l'édition de liens par :
//
//	go build -ldflags "-X main.version=1.2.3"
//
// Elle DOIT être une variable de type string, initialisée par une constante
// littérale (pas une constante, pas le résultat d'une fonction) : c'est la
// seule forme que -X sait écraser. Sans -ldflags, elle vaut "dev".
var version = "dev"

func main() {
	// flag.String renvoie un POINTEUR vers la valeur ; on le déréférence
	// après flag.Parse() avec *addr. Le troisième argument est l'aide
	// affichée par -h.
	addr := flag.String("addr", ":9101", "adresse d'écoute (hôte:port)")
	points := flag.String("points", "/", "points de montage à mesurer, séparés par des virgules")
	afficherVersion := flag.Bool("version", false, "affiche la version et quitte")
	flag.Parse()

	// -version : on affiche et on sort. C'est ce que le Makefile et le
	// README utilisent pour vérifier que -ldflags a bien fait son travail.
	if *afficherVersion {
		fmt.Println("diskexporter", version)
		return
	}

	// Journal en JSON sur la sortie standard : Docker, systemd, Loki le
	// récupèrent tel quel. slog.SetDefault fait que slog.Info(...) partout
	// dans le programme passe par ce handler.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// signal.NotifyContext renvoie un contexte annulé au premier SIGINT
	// (Ctrl-C) ou SIGTERM (docker stop, systemctl stop). stop() libère le
	// handler de signal : un second Ctrl-C tue alors le processus tout de
	// suite, ce qui est le comportement qu'on veut si l'arrêt propre coince.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// sondeSysteme{} est la vraie sonde (syscall.Statfs) ; les tests passent
	// une fausseSonde. C'est l'interface Sonde qui rend ce choix possible.
	if err := run(ctx, *addr, sondeSysteme{}, decouperPoints(*points)); err != nil {
		slog.Error("arrêt sur erreur", "err", err)
		os.Exit(1)
	}
}

// decouperPoints transforme "/, /tmp,,/var" en ["/", "/tmp", "/var"] :
// on coupe sur la virgule, on enlève les espaces, on ignore les vides.
// Appelée par main ; testée directement.
func decouperPoints(liste string) []string {
	var points []string
	for _, p := range strings.Split(liste, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			points = append(points, p)
		}
	}
	return points
}

// nouveauRouteur assemble les routes de l'exporter. Renvoyer un http.Handler
// (et non un *http.ServeMux) permet aux tests de l'appeler avec httptest
// sans ouvrir de port, et à run de le brancher sur un http.Server.
// Les motifs "GET /metrics" (Go 1.22+) répondent 405 aux autres méthodes.
func nouveauRouteur(s Sonde, points []string) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", handlerMetrics(s, points, version))
	mux.Handle("GET /healthz", handlerHealthz())
	return mux
}

// run fait tourner le serveur HTTP sur addr jusqu'à ce que ctx soit annulé,
// puis l'arrête proprement (les requêtes en cours finissent, 5 s maximum).
// Elle renvoie nil après un arrêt demandé, ou l'erreur si le serveur n'a
// pas pu démarrer (port déjà pris, par exemple).
func run(ctx context.Context, addr string, s Sonde, points []string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: nouveauRouteur(s, points),
		// Sans ce délai, un client qui ouvre une connexion et n'envoie jamais
		// ses en-têtes occupe une goroutine pour toujours (attaque slowloris).
		ReadHeaderTimeout: 5 * time.Second,
	}

	// ListenAndServe bloque tant que le serveur tourne : on le lance dans une
	// goroutine et on récupère son résultat par un channel bufferisé (taille 1,
	// pour que la goroutine puisse écrire même si plus personne ne lit).
	erreurs := make(chan error, 1)
	go func() {
		slog.Info("diskexporter démarre", "addr", addr, "version", version, "points", points)
		erreurs <- srv.ListenAndServe()
	}()

	// Deux façons de sortir : le serveur meurt tout seul (erreur de bind),
	// ou le contexte est annulé (signal reçu, ou test qui appelle cancel).
	select {
	case err := <-erreurs:
		// ListenAndServe ne renvoie jamais nil : soit une vraie erreur, soit
		// http.ErrServerClosed après un Shutdown, qui n'arrive pas ici.
		return fmt.Errorf("serveur : %w", err)
	case <-ctx.Done():
	}

	// Arrêt propre : Shutdown ferme le port d'écoute, attend la fin des
	// requêtes en cours, et rend la main. On lui donne un contexte NEUF avec
	// son propre délai : ctx est déjà annulé, on ne peut pas s'en servir.
	slog.Info("arrêt demandé")
	ctxArret, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxArret); err != nil {
		return fmt.Errorf("arrêt : %w", err)
	}

	// La goroutine de ListenAndServe a maintenant écrit ErrServerClosed dans
	// le channel : on le vide pour la forme, et on vérifie que c'est bien lui.
	if err := <-erreurs; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serveur : %w", err)
	}
	return nil
}
