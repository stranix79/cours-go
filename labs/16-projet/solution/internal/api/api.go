// Paquet api, solution : les handlers HTTP du service.
//
// Quoi : NouveauMux assemble les trois routes (/healthz, /api/status,
// /metrics) et le middleware de journalisation. Le format Prometheus est dans
// metrics.go. Ce paquet LIT l'état, il n'écrit jamais dedans.
// Qui l'appelle : cmd/sondes/main.go construit le mux et le donne à
// http.Server ; les tests l'appellent via httptest.
package api

import (
	"encoding/json" // réponses JSON
	"log/slog"      // journal structuré du middleware
	"net/http"      // ServeMux, Handler, ResponseWriter
	"time"          // horodatages et durée des requêtes

	"cours-go/labs/16-projet/solution/internal/sonde"
)

// Serveur porte ce dont les handlers ont besoin. Les handlers sont des
// méthodes : elles accèdent à l'état sans variable globale.
type Serveur struct {
	etat    *sonde.Etat
	version string
	demarre time.Time
}

// Statut est la réponse de /api/status. C'est un type distinct de
// sonde.Resultat : l'API choisit ses noms de champs JSON et ses unités
// (latence en millisecondes flottantes) sans lier le format public à la
// struct interne. Changer l'un ne casse pas l'autre.
type Statut struct {
	Version       string        `json:"version"`
	Demarre       time.Time     `json:"demarre"`
	Verifications uint64        `json:"verifications"`
	Cibles        []StatutCible `json:"cibles"`
}

// StatutCible est une cible dans la réponse de /api/status.
type StatutCible struct {
	Nom       string    `json:"nom"`
	Type      string    `json:"type"`
	Up        bool      `json:"up"`
	LatenceMs float64   `json:"latence_ms"`
	Erreur    string    `json:"erreur,omitempty"`
	Quand     time.Time `json:"quand"`
}

// NouveauMux construit le routeur. La syntaxe "GET /chemin" est celle du
// ServeMux de Go 1.22 (chapitre 13) : une autre méthode sur le même chemin
// reçoit 405, un chemin inconnu 404, sans rien écrire.
func NouveauMux(etat *sonde.Etat, version string, log *slog.Logger) http.Handler {
	s := &Serveur{etat: etat, version: version, demarre: time.Now()}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /api/status", s.status)
	mux.HandleFunc("GET /metrics", s.metrics)
	return Journaliser(log, mux) // le middleware enveloppe tout le mux
}

// repondreJSON encode v avec le bon en-tête et le statut demandé. Le
// Content-Type doit être posé AVANT WriteHeader : après, les en-têtes sont
// déjà partis sur le réseau.
func repondreJSON(w http.ResponseWriter, statut int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statut)
	_ = json.NewEncoder(w).Encode(v) // l'erreur ne peut venir que d'un client parti
}

// healthz répond « je suis vivant » : le processus tourne et sert HTTP. C'est
// ce que Docker ou Kubernetes interrogent. On y met la version pour savoir ce
// qui tourne d'un simple curl.
func (s *Serveur) healthz(w http.ResponseWriter, r *http.Request) {
	repondreJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"version": s.version,
	})
}

// status transforme les Resultat internes en StatutCible et renvoie le tout.
func (s *Serveur) status(w http.ResponseWriter, r *http.Request) {
	resultats := s.etat.Resultats()
	reponse := Statut{
		Version:       s.version,
		Demarre:       s.demarre,
		Verifications: s.etat.Total(),
		Cibles:        make([]StatutCible, 0, len(resultats)), // [] et non null en JSON si vide
	}
	for _, r := range resultats {
		reponse.Cibles = append(reponse.Cibles, StatutCible{
			Nom:       r.Cible,
			Type:      r.Type,
			Up:        r.Up,
			LatenceMs: float64(r.Latence) / float64(time.Millisecond),
			Erreur:    r.Erreur,
			Quand:     r.Quand,
		})
	}
	repondreJSON(w, http.StatusOK, reponse)
}

// metrics écrit le format d'exposition texte de Prometheus (metrics.go). Le
// Content-Type avec version=0.0.4 est celui que Prometheus attend.
func (s *Serveur) metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	FormatMetriques(w, s.etat.Resultats(), s.etat.Total())
}

// enregistreur enveloppe le ResponseWriter pour capturer le code de statut :
// http.ResponseWriter ne permet pas de le relire une fois écrit. L'embarquement
// de l'interface (chapitre 7) délègue toutes les autres méthodes.
type enregistreur struct {
	http.ResponseWriter
	statut int
}

// WriteHeader mémorise le code puis le transmet.
func (e *enregistreur) WriteHeader(code int) {
	e.statut = code
	e.ResponseWriter.WriteHeader(code)
}

// Journaliser est un middleware : il prend un Handler et en rend un autre qui
// journalise chaque requête (méthode, chemin, statut, durée, client) après
// l'avoir servie. Un log nil désactive le middleware proprement.
func Journaliser(log *slog.Logger, suivant http.Handler) http.Handler {
	if log == nil {
		return suivant
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		debut := time.Now()
		e := &enregistreur{ResponseWriter: w, statut: http.StatusOK} // 200 si le handler n'appelle pas WriteHeader
		suivant.ServeHTTP(e, r)
		log.Info("requête",
			"methode", r.Method,
			"chemin", r.URL.Path,
			"statut", e.statut,
			"duree_ms", float64(time.Since(debut))/float64(time.Millisecond),
			"client", r.RemoteAddr,
		)
	})
}
