// Labo 07, solution : la partie io.Reader / io.Writer.
//
// Rôle du fichier : Compter, qui lit depuis n'importe quel io.Reader, et
// EcrireRapport, qui écrit dans n'importe quel io.Writer. En prod : un fichier
// et le terminal. Dans les tests : strings.NewReader et bytes.Buffer.
package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Compter fait ce que fait wc : lignes (nombre de '\n'), mots (séquences
// séparées par des blancs) et octets. Elle lit en flux, ligne par ligne, donc
// un fichier de 10 Go passe sans le charger en mémoire. Les quatre retours
// sont nommés : la signature se lit toute seule, et un simple « return »
// suffit à la fin.
func Compter(r io.Reader) (lignes, mots, octets int, err error) {
	// bufio.Reader ajoute un tampon devant n'importe quel Reader : on lui
	// demande des lignes, il lit le Reader par blocs de 4 Ko derrière.
	br := bufio.NewReader(r)
	for {
		// ReadString rend tout jusqu'au '\n' INCLUS. À la fin du flux, il rend
		// ce qui reste (peut-être sans '\n') avec err == io.EOF.
		ligne, errLecture := br.ReadString('\n')
		octets += len(ligne)
		// strings.Fields découpe sur les blancs (espaces, tabulations, '\n')
		// et ignore les blancs multiples : exactement la définition d'un mot.
		mots += len(strings.Fields(ligne))
		if strings.HasSuffix(ligne, "\n") {
			lignes++
		}
		if errLecture == io.EOF {
			// Fin normale : on ne la remonte pas, c'est un succès.
			return lignes, mots, octets, nil
		}
		if errLecture != nil {
			// Vraie erreur (disque, réseau, Reader cassé) : on l'emballe avec
			// le contexte et on rend les compteurs partiels.
			return lignes, mots, octets, fmt.Errorf("compter: lecture: %w", errLecture)
		}
	}
}

// EcrireRapport écrit une ligne par forme puis le total, dans w. Elle ne sait
// pas si w est le terminal, un fichier, un tampon de test ou une connexion
// réseau, et c'est le but. Chaque Fprintf peut échouer (disque plein,
// connexion coupée) : on teste et on remonte, avec le contexte.
func EcrireRapport(w io.Writer, formes []Forme) error {
	for _, f := range formes {
		// %-16v : la forme via son String(), cadrée à gauche sur 16 colonnes.
		// %8.2f : le nombre sur 8 colonnes avec deux décimales.
		_, err := fmt.Fprintf(w, "%-16v aire=%8.2f  périmètre=%8.2f\n", f, f.Aire(), f.Perimetre())
		if err != nil {
			return fmt.Errorf("écrire rapport: %w", err)
		}
	}
	if _, err := fmt.Fprintf(w, "%-16s aire=%8.2f\n", "total", Total(formes)); err != nil {
		return fmt.Errorf("écrire rapport: %w", err)
	}
	return nil
}
