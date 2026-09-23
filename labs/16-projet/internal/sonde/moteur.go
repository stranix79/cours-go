// Paquet sonde : moteur.go, le worker pool (Tour) et la boucle périodique
// (Boucle). Lancé par cmd/sondes/main.go dans une goroutine.
package sonde

import (
	"context"
	"log/slog"
	"time"

	"cours-go/labs/16-projet/internal/config"
)

// Moteur regroupe ce qu'il faut pour sonder.
type Moteur struct {
	Sondeur    Sondeur
	Cibles     []config.Cible
	Intervalle time.Duration
	Workers    int
	Etat       *Etat
	Log        *slog.Logger // nil accepté : utiliser slog.Default()
}

// journal renvoie le logger à utiliser, jamais nil.
func (m *Moteur) journal() *slog.Logger {
	if m.Log == nil {
		return slog.Default()
	}
	return m.Log
}

// Tour sonde chaque cible une fois, avec au plus m.Workers sondes en
// parallèle (1 si Workers < 1), et enregistre chaque Resultat dans m.Etat.
// Quand Enregistrer renvoie vrai, journalise "changement d'état" avec la
// cible, up, la latence et l'erreur.
//
// Patron attendu (chapitre 11) : un channel non bufferisé de config.Cible,
// Workers goroutines qui le consomment avec range, un sync.WaitGroup ; la
// distribution des cibles dans un select avec ctx.Done() pour pouvoir
// abandonner ; close(travail) puis wg.Wait().
func (m *Moteur) Tour(ctx context.Context) {
	// TODO 12
	_ = ctx
	_ = m.journal()
}

// Boucle fait un Tour immédiatement, puis un Tour à chaque tick d'un
// time.Ticker(m.Intervalle), jusqu'à ce que ctx soit annulé. Arrête le
// Ticker en sortant (defer ticker.Stop()).
func (m *Moteur) Boucle(ctx context.Context) {
	// TODO 13
	_ = ctx
}
