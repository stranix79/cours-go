// Paquet api : les handlers HTTP du service (/healthz, /api/status, /metrics)
// et le middleware de journalisation. Ce paquet LIT l'état, il n'écrit jamais
// dedans. Appelé par cmd/sondes/main.go et par les tests (httptest).
package api

import (
	"log/slog"
	"net/http"
	"time"

	"cours-go/labs/16-projet/internal/sonde"
)

// Serveur porte ce dont les handlers ont besoin ; les handlers sont ses
// méthodes.
type Serveur struct {
	etat    *sonde.Etat
	version string
	demarre time.Time
}

// Statut est la réponse JSON de /api/status. Un type distinct de
// sonde.Resultat : le format public ne dépend pas de la struct interne.
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

// NouveauMux construit le routeur avec la syntaxe "GET /chemin" du ServeMux
// de Go 1.22, et l'enveloppe dans Journaliser(log, mux).
func NouveauMux(etat *sonde.Etat, version string, log *slog.Logger) http.Handler {
	s := &Serveur{etat: etat, version: version, demarre: time.Now()}
	mux := http.NewServeMux()
	// TODO 14 : mux.HandleFunc("GET /healthz", s.healthz), puis /api/status et
	// /metrics ; renvoyer Journaliser(log, mux).
	_ = s
	return mux
}

// healthz répond 200 avec {"ok":true,"version":"..."} en JSON
// (Content-Type application/json).
func (s *Serveur) healthz(w http.ResponseWriter, r *http.Request) {
	// TODO 15
}

// status répond 200 avec un Statut : version, démarrage, total des
// vérifications, et une StatutCible par résultat de s.etat.Resultats()
// (latence en millisecondes flottantes). Une liste vide doit donner [] en
// JSON, pas null : créer la slice avec make, même de longueur 0.
func (s *Serveur) status(w http.ResponseWriter, r *http.Request) {
	// TODO 16
}

// metrics pose Content-Type "text/plain; version=0.0.4; charset=utf-8" et
// appelle FormatMetriques(w, s.etat.Resultats(), s.etat.Total()).
func (s *Serveur) metrics(w http.ResponseWriter, r *http.Request) {
	// TODO 17
}

// Journaliser est un middleware : il rend un Handler qui sert la requête puis
// journalise avec log.Info("requête", ...) les attributs methode, chemin,
// statut, duree_ms, client. Si log est nil, renvoie suivant tel quel.
//
// Pour connaître le statut, enveloppe le ResponseWriter dans une struct qui
// embarque http.ResponseWriter et redéfinit WriteHeader pour le mémoriser
// (200 par défaut si le handler ne l'appelle pas).
func Journaliser(log *slog.Logger, suivant http.Handler) http.Handler {
	// TODO 18
	_ = log
	return suivant
}
