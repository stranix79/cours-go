// Paquet config, solution : lecture et validation du fichier JSON des cibles.
//
// Quoi : le type Config avec ses valeurs par défaut, le type Duree qui accepte
// "5s" ou "250ms" dans le JSON, Charger et Parser pour lire, Valider pour
// refuser une configuration bancale AVANT de démarrer quoi que ce soit.
// Qui l'appelle : cmd/sondes/main.go au démarrage ; le paquet sonde consomme
// les Cible ; le paquet api ne le connaît pas.
package config

import (
	"encoding/json" // décodage du fichier JSON
	"errors"        // errors.New pour la sentinelle ErrInvalide
	"fmt"           // fmt.Errorf avec %w pour envelopper les erreurs
	"io"            // io.Reader : Parser ne sait pas d'où vient le JSON
	"net"           // net.SplitHostPort pour vérifier "hôte:port"
	"net/url"       // url.Parse pour vérifier une URL http(s)
	"os"            // os.Open pour lire le fichier
	"strings"       // strings.TrimSpace pour refuser un nom fait d'espaces
	"time"          // time.Duration et time.ParseDuration
)

// ErrInvalide est la sentinelle enveloppée par toutes les erreurs de
// validation. L'appelant écrit errors.Is(err, config.ErrInvalide) et ne dépend
// pas du texte du message (chapitre 8).
var ErrInvalide = errors.New("configuration invalide")

// Les deux types de cible connus. Des constantes exportées plutôt que des
// chaînes en dur répétées dans trois paquets.
const (
	TypeHTTP = "http"
	TypeTCP  = "tcp"
)

// Duree est un time.Duration qui sait se lire depuis une chaîne JSON.
// Sans ce type, encoding/json lirait time.Duration comme un entier en
// nanosecondes : 5000000000 pour cinq secondes, illisible dans un fichier.
type Duree time.Duration

// UnmarshalText est la méthode de l'interface encoding.TextUnmarshaler.
// encoding/json l'appelle automatiquement quand la valeur JSON est une chaîne
// et que le type de destination l'implémente. Le récepteur est un pointeur :
// on modifie la valeur en place.
func (d *Duree) UnmarshalText(b []byte) error {
	v, err := time.ParseDuration(string(b)) // "5s", "250ms", "1m30s"
	if err != nil {
		return fmt.Errorf("durée %q : %w", string(b), err)
	}
	*d = Duree(v)
	return nil
}

// MarshalText fait le chemin inverse (encoding.TextMarshaler) : une Duree
// écrite en JSON redevient "5s", pas 5000000000. Utile pour afficher la config.
func (d Duree) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

// Duration rend la valeur utilisable par time.NewTicker, http.Client{Timeout}...
// Une simple conversion, mais nommée : cfg.Intervalle.Duration() se lit mieux
// que time.Duration(cfg.Intervalle) partout dans le code.
func (d Duree) Duration() time.Duration { return time.Duration(d) }

// Cible est une entrée de la liste "cibles" du JSON. Selon Type, c'est URL
// (http) ou Adresse (tcp) qui est renseigné ; omitempty évite d'écrire le
// champ vide si on réencode la config.
type Cible struct {
	Nom     string `json:"nom"`
	Type    string `json:"type"`
	URL     string `json:"url,omitempty"`
	Adresse string `json:"adresse,omitempty"`
}

// Config est le fichier entier. Les champs absents du JSON gardent la valeur
// posée par Defaut() : c'est pour ça que Parser part de Defaut() et non d'une
// Config vide.
type Config struct {
	Ecoute     string  `json:"ecoute"`     // adresse d'écoute HTTP, ":8080"
	Intervalle Duree   `json:"intervalle"` // délai entre deux tours de sondes
	Timeout    Duree   `json:"timeout"`    // délai maximal d'une sonde
	Workers    int     `json:"workers"`    // sondes en parallèle au maximum
	Cibles     []Cible `json:"cibles"`
}

// Defaut renvoie la configuration de départ : ce qu'on obtient avec un JSON
// qui ne contient que "cibles".
func Defaut() Config {
	return Config{
		Ecoute:     ":8080",
		Intervalle: Duree(30 * time.Second),
		Timeout:    Duree(5 * time.Second),
		Workers:    4,
	}
}

// Charger ouvre le fichier et délègue à Parser. Séparer les deux permet de
// tester Parser avec une chaîne, sans fichier.
func Charger(chemin string) (*Config, error) {
	f, err := os.Open(chemin)
	if err != nil {
		return nil, fmt.Errorf("ouvrir la config : %w", err)
	}
	defer f.Close() // fermé quoi qu'il arrive, même si Parser échoue
	return Parser(f)
}

// Parser décode le JSON par-dessus les défauts, puis valide. Un champ inconnu
// dans le fichier est une erreur : une faute de frappe ("intervale") serait
// sinon silencieusement ignorée et le défaut appliqué.
func Parser(r io.Reader) (*Config, error) {
	cfg := Defaut()
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("lire la config : %w", err)
	}
	if err := cfg.Valider(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Valider vérifie la cohérence globale puis chaque cible. On s'arrête à la
// première erreur : l'utilisateur corrige, relance, et voit la suivante. Toutes
// les erreurs enveloppent ErrInvalide.
func (c *Config) Valider() error {
	if c.Intervalle <= 0 {
		return fmt.Errorf("%w : intervalle doit être > 0", ErrInvalide)
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("%w : timeout doit être > 0", ErrInvalide)
	}
	if c.Timeout > c.Intervalle {
		// Sinon un tour peut durer plus longtemps que l'intervalle et les tours
		// s'empilent.
		return fmt.Errorf("%w : timeout (%s) ne peut pas dépasser l'intervalle (%s)",
			ErrInvalide, c.Timeout.Duration(), c.Intervalle.Duration())
	}
	if c.Workers < 1 {
		return fmt.Errorf("%w : workers doit être >= 1", ErrInvalide)
	}
	if len(c.Cibles) == 0 {
		return fmt.Errorf("%w : aucune cible", ErrInvalide)
	}
	// Un ensemble de noms déjà vus : la map sert de set (chapitre 4).
	vus := make(map[string]bool, len(c.Cibles))
	for i, cible := range c.Cibles {
		if err := cible.valider(); err != nil {
			// Deux %w dans un même Errorf sont permis depuis Go 1.20 : l'erreur
			// résultante « est » à la fois ErrInvalide et l'erreur de la cible.
			return fmt.Errorf("%w : cible %d : %w", ErrInvalide, i, err)
		}
		if vus[cible.Nom] {
			return fmt.Errorf("%w : cible %q en double", ErrInvalide, cible.Nom)
		}
		vus[cible.Nom] = true
	}
	return nil
}

// valider (minuscule : privé au paquet) vérifie une cible seule. Le nom est le
// label Prometheus et la clé de l'état : il ne peut pas être vide.
func (c Cible) valider() error {
	if strings.TrimSpace(c.Nom) == "" {
		return errors.New("nom vide")
	}
	switch c.Type {
	case TypeHTTP:
		u, err := url.Parse(c.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("url %q : attendu http://hôte ou https://hôte", c.URL)
		}
	case TypeTCP:
		if _, _, err := net.SplitHostPort(c.Adresse); err != nil {
			return fmt.Errorf("adresse %q : attendu hôte:port", c.Adresse)
		}
	default:
		return fmt.Errorf("type %q inconnu (http ou tcp)", c.Type)
	}
	return nil
}
