// Labo 12, solution : logstat, la logique. Analyse de lignes de log,
// comptage par niveau, messages les plus fréquents, sortie JSON.
// Appelé par cli.go ; testé par logstat_test.go sans fichier ni processus.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// Entree est une ligne de log comprise : "2026-09-23T10:00:00Z LEVEL message".
type Entree struct {
	Horodatage time.Time
	Niveau     string
	Message    string
}

// Frequence est un message et son nombre d'occurrences. Les tags json
// donnent les noms de clés en minuscules dans la sortie de `logstat json`.
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

// OrdreNiveaux fixe l'ordre d'affichage des niveaux connus.
var OrdreNiveaux = []string{"DEBUG", "INFO", "WARN", "ERROR"}

// ordreNiveau rend la position d'un niveau, ou len(OrdreNiveaux) s'il est
// inconnu : les niveaux inconnus passent donc après les connus.
func ordreNiveau(niveau string) int {
	for i, n := range OrdreNiveaux {
		if n == niveau {
			return i
		}
	}
	return len(OrdreNiveaux)
}

// ParseLigne découpe "horodatage NIVEAU message". SplitN avec 3 limite le
// découpage à trois morceaux : le troisième garde tous les espaces du message.
func ParseLigne(ligne string) (Entree, error) {
	champs := strings.SplitN(ligne, " ", 3)
	if len(champs) < 3 {
		return Entree{}, errors.New("format attendu : horodatage NIVEAU message")
	}
	// time.RFC3339 est la constante de format "2006-01-02T15:04:05Z07:00" :
	// Go décrit les formats de date avec une date de référence, pas des %Y.
	horodatage, err := time.Parse(time.RFC3339, champs[0])
	if err != nil {
		return Entree{}, fmt.Errorf("horodatage : %w", err)
	}
	return Entree{Horodatage: horodatage, Niveau: champs[1], Message: champs[2]}, nil
}

// Lire analyse un flux ligne par ligne. Un io.Reader plutôt qu'un chemin :
// la même fonction sert pour un fichier, stdin, ou une string dans les tests.
func Lire(r io.Reader) ([]Entree, error) {
	var entrees []Entree
	sc := bufio.NewScanner(r)
	numero := 0
	for sc.Scan() {
		numero++
		ligne := sc.Text()
		if strings.TrimSpace(ligne) == "" {
			continue // ligne vide : ignorée, mais comptée dans la numérotation
		}
		e, err := ParseLigne(ligne)
		if err != nil {
			// %w enveloppe l'erreur d'origine : errors.Is / errors.As marchent encore.
			return nil, fmt.Errorf("ligne %d : %w", numero, err)
		}
		entrees = append(entrees, e)
	}
	if err := sc.Err(); err != nil { // Scan() rend false à la fin ET en cas d'erreur
		return nil, err
	}
	return entrees, nil
}

// Compter rend le nombre d'entrées par niveau.
func Compter(entrees []Entree) map[string]int {
	comptes := make(map[string]int)
	for _, e := range entrees {
		comptes[e.Niveau]++ // une clé absente vaut 0 : pas de test d'existence
	}
	return comptes
}

// Top rend les n messages les plus fréquents, égalités tranchées par
// l'ordre alphabétique pour que la sortie soit reproductible.
func Top(entrees []Entree, n int) []Frequence {
	comptes := make(map[string]int)
	for _, e := range entrees {
		comptes[e.Message]++
	}
	// On verse la map dans une slice : une map n'a pas d'ordre, une slice se trie.
	freqs := make([]Frequence, 0, len(comptes))
	for message, nombre := range comptes {
		freqs = append(freqs, Frequence{Message: message, Nombre: nombre})
	}
	sort.Slice(freqs, func(i, j int) bool {
		if freqs[i].Nombre != freqs[j].Nombre {
			return freqs[i].Nombre > freqs[j].Nombre // le plus fréquent d'abord
		}
		return freqs[i].Message < freqs[j].Message // puis alphabétique
	})
	if n < 0 {
		n = 0
	}
	if n > len(freqs) {
		n = len(freqs)
	}
	return freqs[:n]
}

// FormatCompte : niveaux connus dans l'ordre de OrdreNiveaux, puis les
// autres par ordre alphabétique, une ligne par niveau.
func FormatCompte(parNiveau map[string]int) string {
	niveaux := make([]string, 0, len(parNiveau))
	for n := range parNiveau {
		niveaux = append(niveaux, n)
	}
	sort.Slice(niveaux, func(i, j int) bool {
		oi, oj := ordreNiveau(niveaux[i]), ordreNiveau(niveaux[j])
		if oi != oj {
			return oi < oj
		}
		return niveaux[i] < niveaux[j] // deux niveaux inconnus : alphabétique
	})
	var b strings.Builder
	for _, n := range niveaux {
		fmt.Fprintf(&b, "%-5s %d\n", n, parNiveau[n])
	}
	return b.String()
}

// EncoderJSON écrit s indenté. NewEncoder écrit directement dans le Writer
// (stdout, un fichier, un tampon de test) et ajoute le retour à la ligne
// final ; les clés d'une map sont triées, la sortie est donc stable.
func EncoderJSON(w io.Writer, s Stats) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}
