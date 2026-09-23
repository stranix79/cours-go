// Labo 12 : logstat, la logique. Analyse de lignes de log, comptage par
// niveau, messages les plus fréquents, sortie JSON. Tout est testable sans
// fichier ni processus ; cli.go branche ces fonctions sur la ligne de commande.
package main

import (
	"errors"
	"io"
	"time"
)

// Entree est une ligne de log comprise : "2026-09-23T10:00:00Z LEVEL message".
type Entree struct {
	Horodatage time.Time
	Niveau     string
	Message    string
}

// Frequence est un message et son nombre d'occurrences.
type Frequence struct {
	Message string `json:"message"`
	Nombre  int    `json:"nombre"`
}

// Stats est ce que la sous-commande json sérialise.
type Stats struct {
	Total     int            `json:"total"`
	ParNiveau map[string]int `json:"par_niveau"`
	Top       []Frequence    `json:"top"`
}

// OrdreNiveaux fixe l'ordre d'affichage des niveaux connus ; les autres
// viennent après, par ordre alphabétique.
var OrdreNiveaux = []string{"DEBUG", "INFO", "WARN", "ERROR"}

// ordreNiveau rend la position d'un niveau dans OrdreNiveaux, ou len(OrdreNiveaux)
// s'il est inconnu. Fourni : sert à FormatCompte.
func ordreNiveau(niveau string) int {
	for i, n := range OrdreNiveaux {
		if n == niveau {
			return i
		}
	}
	return len(OrdreNiveaux)
}

// ParseLigne découpe "horodatage NIVEAU message" (trois champs, le message
// pouvant contenir des espaces). L'horodatage est au format RFC 3339.
func ParseLigne(ligne string) (Entree, error) {
	// TODO 1 : strings.SplitN(ligne, " ", 3) ; moins de 3 champs → erreur
	// "format attendu : horodatage NIVEAU message" ; time.Parse(time.RFC3339,
	// champs[0]) ; renvoie Entree{horodatage, champs[1], champs[2]}.
	return Entree{}, errors.New("TODO")
}

// Lire analyse toutes les lignes d'un flux. Les lignes vides sont ignorées ;
// une ligne malformée arrête tout avec une erreur "ligne N : ...".
func Lire(r io.Reader) ([]Entree, error) {
	// TODO 2 : bufio.NewScanner(r), un compteur de lignes, ParseLigne sur
	// chaque ligne non vide, fmt.Errorf("ligne %d : %w", n, err) en cas
	// d'échec, et sc.Err() après la boucle.
	return nil, errors.New("TODO")
}

// Compter rend le nombre d'entrées par niveau.
func Compter(entrees []Entree) map[string]int {
	// TODO 3 : une map, une boucle, un ++.
	return nil
}

// Top rend les n messages les plus fréquents, du plus fréquent au moins
// fréquent ; à nombre égal, par ordre alphabétique du message. n <= 0 rend
// une slice vide ; n plus grand que le nombre de messages distincts rend tout.
func Top(entrees []Entree, n int) []Frequence {
	// TODO 4 : compte les messages dans une map, verse-la dans une
	// []Frequence, trie avec sort.Slice (Nombre décroissant puis Message
	// croissant), coupe à n.
	return nil
}

// FormatCompte met en forme les comptes par niveau, dans l'ordre de
// OrdreNiveaux puis alphabétique, au format "%-5s %d\n" (ex : "INFO  12").
func FormatCompte(parNiveau map[string]int) string {
	// TODO 5 : récupère les clés, trie-les avec sort.Slice en comparant
	// ordreNiveau puis le nom, et écris chaque ligne dans un strings.Builder.
	return ""
}

// EncoderJSON écrit s en JSON indenté de deux espaces, suivi d'un retour
// à la ligne.
func EncoderJSON(w io.Writer, s Stats) error {
	// TODO 6 : json.NewEncoder(w), SetIndent("", "  "), Encode(s).
	return errors.New("TODO")
}
