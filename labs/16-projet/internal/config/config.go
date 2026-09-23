// Paquet config : lecture et validation du fichier JSON des cibles.
// Appelé par cmd/sondes/main.go au démarrage ; le paquet sonde consomme les Cible.
package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

// ErrInvalide est la sentinelle que toutes les erreurs de validation doivent
// envelopper (fmt.Errorf("%w : ...", ErrInvalide)), pour que l'appelant puisse
// écrire errors.Is(err, config.ErrInvalide).
var ErrInvalide = errors.New("configuration invalide")

// Les deux types de cible connus.
const (
	TypeHTTP = "http"
	TypeTCP  = "tcp"
)

// Duree est un time.Duration qui se lit depuis une chaîne JSON ("5s", "250ms").
// Elle doit implémenter encoding.TextUnmarshaler ET encoding.TextMarshaler.
type Duree time.Duration

// UnmarshalText est appelé par encoding/json quand la valeur JSON est une
// chaîne. Utilise time.ParseDuration ; enveloppe l'erreur avec la chaîne fautive.
func (d *Duree) UnmarshalText(b []byte) error {
	// TODO 1 : parser b avec time.ParseDuration et affecter *d.
	return errors.New("TODO")
}

// MarshalText fait le chemin inverse : Duree(5*time.Second) → "5s".
func (d Duree) MarshalText() ([]byte, error) {
	// TODO 1 : time.Duration(d).String()
	return nil, errors.New("TODO")
}

// Duration rend la valeur utilisable par time.NewTicker, http.Client{Timeout}...
func (d Duree) Duration() time.Duration { return time.Duration(d) }

// Cible est une entrée de la liste "cibles" du JSON.
type Cible struct {
	Nom     string `json:"nom"`
	Type    string `json:"type"`
	URL     string `json:"url,omitempty"`     // si Type == "http"
	Adresse string `json:"adresse,omitempty"` // si Type == "tcp", forme hôte:port
}

// Config est le fichier entier. Les champs absents du JSON gardent la valeur
// de Defaut().
type Config struct {
	Ecoute     string  `json:"ecoute"`     // adresse d'écoute HTTP
	Intervalle Duree   `json:"intervalle"` // délai entre deux tours de sondes
	Timeout    Duree   `json:"timeout"`    // délai maximal d'une sonde
	Workers    int     `json:"workers"`    // sondes en parallèle au maximum
	Cibles     []Cible `json:"cibles"`
}

// Defaut renvoie la configuration de départ.
func Defaut() Config {
	return Config{
		Ecoute:     ":8080",
		Intervalle: Duree(30 * time.Second),
		Timeout:    Duree(5 * time.Second),
		Workers:    4,
	}
}

// Charger ouvre le fichier et délègue à Parser. Déjà écrit : c'est Parser qui
// contient la logique.
func Charger(chemin string) (*Config, error) {
	f, err := os.Open(chemin)
	if err != nil {
		return nil, fmt.Errorf("ouvrir la config : %w", err)
	}
	defer f.Close()
	return Parser(f)
}

// Parser décode le JSON par-dessus Defaut(), refuse les champs inconnus
// (json.Decoder.DisallowUnknownFields), puis appelle Valider.
func Parser(r io.Reader) (*Config, error) {
	// TODO 2 : cfg := Defaut() ; dec := json.NewDecoder(r) ;
	// dec.DisallowUnknownFields() ; dec.Decode(&cfg) ; cfg.Valider() ; &cfg.
	_ = r
	return nil, errors.New("TODO")
}

// Valider vérifie la cohérence globale puis chaque cible. Règles, dans cet
// ordre : Intervalle > 0 ; Timeout > 0 ; Timeout <= Intervalle ; Workers >= 1 ;
// au moins une cible ; chaque cible valide (nom non vide après TrimSpace,
// type http avec une URL http:// ou https:// et un hôte, type tcp avec une
// adresse hôte:port acceptée par net.SplitHostPort) ; pas deux cibles du même
// nom. Chaque erreur enveloppe ErrInvalide.
func (c *Config) Valider() error {
	// TODO 3 : les vérifications ci-dessus. Astuce : une méthode privée
	// (c Cible) valider() error pour une cible, et une map[string]bool pour
	// détecter les doublons.
	return nil
}
