// Labo 08, solution : un lecteur de configuration qui explique ses erreurs.
//
// Rôle du fichier : ChargerConfig et ses trois étapes (lire, parser, valider),
// chacune emballant l'erreur de l'étape du dessous. Les sentinelles et le type
// ErreurValidation sont l'API que les appelants inspectent avec errors.Is / As.
// Appelé par main.go ; testé par config_test.go.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

// Les sentinelles : une valeur unique chacune, déclarée une fois. L'appelant
// teste errors.Is(err, ErrIntrouvable) sans se soucier du texte.
var (
	ErrIntrouvable = errors.New("fichier introuvable")
	ErrInvalide    = errors.New("format invalide")
)

// ErreurValidation porte des DONNÉES : quel champ, pourquoi. Une sentinelle ne
// suffirait pas, l'appelant veut afficher le champ fautif.
type ErreurValidation struct {
	Champ  string
	Raison string
}

// Error à receveur pointeur : on renvoie toujours &ErreurValidation{...}, et
// errors.As cherchera un *ErreurValidation.
func (e *ErreurValidation) Error() string {
	return "champ " + e.Champ + " : " + e.Raison
}

// Unwrap fait de chaque ErreurValidation un cas particulier d'ErrInvalide :
// errors.Is(err, ErrInvalide) est vrai pour une erreur de validation, sans
// que l'appelant ait besoin de connaître le type. C'est ainsi que
// *fs.PathError emballe fs.ErrNotExist dans la bibliothèque standard.
func (e *ErreurValidation) Unwrap() error {
	return ErrInvalide
}

// Config est le résultat : trois champs typés, prêts à l'emploi.
type Config struct {
	Hote string
	Port int
	Env  string
}

// ChargerConfig enchaîne les trois étapes. Chaque étape a déjà emballé son
// erreur avec son propre contexte ; ici on ajoute le contexte de plus haut
// niveau, « charger config », et rien d'autre. Le message final se lit :
// charger config: lire /chemin/app.conf: fichier introuvable
func ChargerConfig(chemin string) (Config, error) {
	data, err := lire(chemin)
	if err != nil {
		return Config{}, fmt.Errorf("charger config: %w", err)
	}
	valeurs, err := parser(data)
	if err != nil {
		return Config{}, fmt.Errorf("charger config: %w", err)
	}
	cfg, err := valider(valeurs)
	if err != nil {
		return Config{}, fmt.Errorf("charger config: %w", err)
	}
	return cfg, nil
}

// lire renvoie le contenu du fichier. Le cas « n'existe pas » est traduit en
// notre sentinelle ErrIntrouvable : l'appelant n'a pas à connaître io/fs.
// Les autres erreurs (permission, disque) sont remontées telles quelles,
// emballées avec le chemin.
func lire(chemin string) ([]byte, error) {
	data, err := os.ReadFile(chemin)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("lire %s: %w", chemin, ErrIntrouvable)
	}
	if err != nil {
		return nil, fmt.Errorf("lire %s: %w", chemin, err)
	}
	return data, nil
}

// parser transforme « clé = valeur » ligne par ligne en map. Les lignes vides
// et les commentaires (#) sont ignorés. Une ligne sans « = » ou avec une clé
// vide est un ErrInvalide, emballé avec le numéro de ligne : c'est ce que
// l'utilisateur voudra savoir.
func parser(data []byte) (map[string]string, error) {
	valeurs := make(map[string]string)
	// bytes.NewReader fait d'une []byte un io.Reader ; bufio.Scanner le
	// découpe en lignes. Même code que pour un fichier de 10 Go.
	sc := bufio.NewScanner(bytes.NewReader(data))
	for num := 1; sc.Scan(); num++ {
		ligne := strings.TrimSpace(sc.Text())
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			continue
		}
		cle, val, ok := strings.Cut(ligne, "=")
		cle = strings.TrimSpace(cle)
		if !ok || cle == "" {
			return nil, fmt.Errorf("parser: ligne %d %q: %w", num, ligne, ErrInvalide)
		}
		valeurs[cle] = strings.TrimSpace(val)
	}
	// sc.Err() est nil en fin de flux normale ; une vraie erreur de lecture
	// serait remontée ici. Sur un bytes.Reader elle ne peut pas arriver, mais
	// le code est le même que pour un fichier, donc on la traite.
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("parser: %w", err)
	}
	return valeurs, nil
}

// valider convertit et vérifie chaque champ. Chaque faute est une
// *ErreurValidation (donc aussi un ErrInvalide via Unwrap), emballée avec
// le contexte « valider ».
func valider(valeurs map[string]string) (Config, error) {
	var cfg Config

	cfg.Hote = valeurs["hote"]
	if cfg.Hote == "" {
		return Config{}, fmt.Errorf("valider: %w", &ErreurValidation{Champ: "hote", Raison: "manquant"})
	}

	portTexte, ok := valeurs["port"]
	if !ok {
		return Config{}, fmt.Errorf("valider: %w", &ErreurValidation{Champ: "port", Raison: "manquant"})
	}
	port, err := strconv.Atoi(portTexte)
	if err != nil {
		// On ne remonte pas l'erreur de strconv : elle parle de syntaxe Go
		// (« invalid syntax »), pas de configuration. On la remplace par la
		// nôtre, plus utile, et le message d'origine ne sert à rien ici.
		return Config{}, fmt.Errorf("valider: %w", &ErreurValidation{Champ: "port", Raison: "pas un entier : " + portTexte})
	}
	if port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("valider: %w", &ErreurValidation{Champ: "port", Raison: "hors de 1..65535"})
	}
	cfg.Port = port

	// env est optionnel : « dev » par défaut. La valeur zéro d'une string
	// dans une map absente est "", ce qui rend le défaut naturel.
	cfg.Env = valeurs["env"]
	if cfg.Env == "" {
		cfg.Env = "dev"
	}
	if cfg.Env != "dev" && cfg.Env != "prod" {
		return Config{}, fmt.Errorf("valider: %w", &ErreurValidation{Champ: "env", Raison: "doit être dev ou prod"})
	}
	return cfg, nil
}

// Securise exécute f et transforme un éventuel panic en erreur. C'est le motif
// de la frontière (chapitre 8) : un retour nommé, un defer, recover dedans.
// Si la valeur du panic est déjà une erreur, on l'emballe avec %w pour que
// errors.Is continue de marcher ; sinon on la formate avec %v.
func Securise(f func()) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return // pas de panic : err garde sa valeur (nil)
		}
		if e, ok := r.(error); ok {
			err = fmt.Errorf("panique récupérée: %w", e)
			return
		}
		err = fmt.Errorf("panique récupérée: %v", r)
	}()
	f()
	return nil
}
