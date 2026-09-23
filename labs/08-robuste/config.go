// Labo 08 : un lecteur de configuration qui explique ses erreurs.
// Rôle du fichier : ChargerConfig et ses trois étapes (lire, parser, valider),
// chacune emballant l'erreur de l'étape du dessous. Appelé par main.go ;
// testé par config_test.go.
package main

import (
	"errors"
	"fmt"
)

// Les sentinelles : à emballer avec %w, à tester avec errors.Is.
var (
	ErrIntrouvable = errors.New("fichier introuvable")
	ErrInvalide    = errors.New("format invalide")
)

// ErreurValidation porte des DONNÉES : quel champ, pourquoi.
type ErreurValidation struct {
	Champ  string
	Raison string
}

// Error à receveur pointeur : on renvoie toujours &ErreurValidation{...}.
func (e *ErreurValidation) Error() string {
	return "champ " + e.Champ + " : " + e.Raison
}

// Unwrap fait de chaque ErreurValidation un cas particulier d'ErrInvalide :
// errors.Is(err, ErrInvalide) devient vrai pour une erreur de validation.
// TODO 1 : renvoie la sentinelle.
func (e *ErreurValidation) Unwrap() error {
	return nil
}

// Config est le résultat.
type Config struct {
	Hote string
	Port int
	Env  string
}

// ChargerConfig enchaîne lire, parser, valider. Chaque erreur est emballée
// avec fmt.Errorf("charger config: %w", err), rien de plus : les étapes ont
// déjà mis leur propre contexte. Message final attendu par les tests pour un
// fichier absent : « charger config: lire CHEMIN: fichier introuvable ».
// TODO 5 : à écrire en dernier, quand les trois étapes marchent.
func ChargerConfig(chemin string) (Config, error) {
	return Config{}, fmt.Errorf("charger config: %w", errors.New("TODO"))
}

// lire renvoie le contenu du fichier (os.ReadFile). Si le fichier n'existe pas
// (errors.Is(err, fs.ErrNotExist)), renvoie fmt.Errorf("lire %s: %w", chemin,
// ErrIntrouvable). Toute autre erreur est emballée de la même façon, telle quelle.
// TODO 2.
func lire(chemin string) ([]byte, error) {
	return nil, errors.New("TODO")
}

// parser transforme « clé = valeur » ligne par ligne en map (espaces autour
// de la clé et de la valeur retirés). Lignes vides et commentaires (#)
// ignorés. Une ligne sans « = » ou avec une clé vide renvoie
// fmt.Errorf("parser: ligne %d %q: %w", numéro, ligne, ErrInvalide).
// TODO 3 : bufio.NewScanner(bytes.NewReader(data)), strings.Cut(ligne, "=").
func parser(data []byte) (map[string]string, error) {
	return nil, errors.New("TODO")
}

// valider convertit et vérifie chaque champ. Chaque faute est renvoyée comme
// fmt.Errorf("valider: %w", &ErreurValidation{Champ: ..., Raison: ...}) avec,
// exactement (les tests comparent) :
//   - hote absent ou vide        -> {"hote", "manquant"}
//   - port absent                -> {"port", "manquant"}
//   - port pas un entier (Atoi)  -> {"port", "pas un entier : " + texte}
//   - port hors de 1..65535      -> {"port", "hors de 1..65535"}
//   - env absent                 -> "dev" par défaut, pas d'erreur
//   - env ni dev ni prod         -> {"env", "doit être dev ou prod"}
//
// TODO 4.
func valider(valeurs map[string]string) (Config, error) {
	return Config{}, errors.New("TODO")
}

// Securise exécute f et transforme un éventuel panic en erreur : retour nommé,
// defer, recover. Si la valeur du panic est une error, emballe-la avec %w
// (errors.Is doit marcher) ; sinon formate-la avec %v. Sans panic : nil.
// TODO 6.
func Securise(f func()) (err error) {
	f()
	return errors.New("TODO")
}
