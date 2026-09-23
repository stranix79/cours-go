// Paquet sonde, solution : moteur.go, le worker pool et la boucle périodique.
//
// Quoi : Moteur.Tour sonde toutes les cibles une fois avec au plus Workers
// sondes en parallèle ; Moteur.Boucle enchaîne les tours à chaque tick jusqu'à
// l'annulation du contexte.
// Qui l'appelle : cmd/sondes/main.go lance Boucle dans une goroutine ; les
// tests appellent Tour et Boucle avec un Sondeur fictif.
package sonde

import (
	"context"  // l'annulation vient de main (signal) ou des tests
	"log/slog" // journal structuré
	"sync"     // sync.WaitGroup pour attendre les workers
	"time"     // time.Ticker

	"cours-go/labs/16-projet/solution/internal/config"
)

// Moteur regroupe ce qu'il faut pour sonder : qui sonde, quoi, à quel rythme,
// avec combien de workers, où stocker, où journaliser. Une struct plutôt que
// six paramètres de fonction.
type Moteur struct {
	Sondeur    Sondeur
	Cibles     []config.Cible
	Intervalle time.Duration
	Workers    int
	Etat       *Etat
	Log        *slog.Logger // nil accepté : slog.Default() est utilisé
}

// journal renvoie le logger à utiliser, jamais nil.
func (m *Moteur) journal() *slog.Logger {
	if m.Log == nil {
		return slog.Default()
	}
	return m.Log
}

// Tour sonde chaque cible une fois. Le patron est le worker pool du
// chapitre 11 : un channel de travail, Workers goroutines qui le consomment,
// un WaitGroup pour attendre la fin. Le channel n'est pas bufferisé : l'envoi
// bloque tant qu'aucun worker n'est libre, ce qui borne le parallélisme.
func (m *Moteur) Tour(ctx context.Context) {
	workers := m.Workers
	if workers < 1 {
		workers = 1
	}
	travail := make(chan config.Cible)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1) // AVANT le go, sinon Wait peut passer avant que la goroutine démarre
		go func() {
			defer wg.Done()
			for c := range travail { // se termine quand le channel est fermé et vide
				r := m.Sondeur.Sonder(ctx, c)
				if m.Etat.Enregistrer(r) {
					// On ne journalise que les changements : un tour toutes les
					// 30 s sur 50 cibles ferait sinon 144 000 lignes par jour.
					attrs := []any{"cible", r.Cible, "up", r.Up,
						"latence_ms", float64(r.Latence) / float64(time.Millisecond)}
					if !r.Up {
						attrs = append(attrs, "erreur", r.Erreur)
					}
					m.journal().Info("changement d'état", attrs...)
				}
			}
		}()
	}
	// Distribution du travail. Le select permet d'abandonner la distribution
	// si le contexte est annulé pendant qu'on attend un worker libre ; le
	// break étiqueté sort du for, un break nu ne sortirait que du select.
distribution:
	for _, c := range m.Cibles {
		select {
		case travail <- c:
		case <-ctx.Done():
			break distribution
		}
	}
	close(travail) // les workers finissent leur cible en cours puis sortent du range
	wg.Wait()      // et on attend qu'ils soient tous sortis
}

// Boucle fait un tour tout de suite (pour que /api/status ait quelque chose
// à dire dès le démarrage), puis un tour à chaque tick, jusqu'à l'annulation.
// Le Ticker est arrêté par defer : un Ticker oublié tourne pour toujours.
func (m *Moteur) Boucle(ctx context.Context) {
	m.Tour(ctx)
	ticker := time.NewTicker(m.Intervalle)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.Tour(ctx)
		}
	}
}
