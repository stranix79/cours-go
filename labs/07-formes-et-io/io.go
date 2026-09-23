// Labo 07 : la partie io.Reader / io.Writer.
// Rôle du fichier : Compter, qui lit depuis n'importe quel io.Reader, et
// EcrireRapport, qui écrit dans n'importe quel io.Writer.
package main

import (
	"errors"
	"io"
)

// Compter fait ce que fait wc : lignes (nombre de '\n'), mots (séquences
// séparées par des blancs, voir strings.Fields) et octets (len de chaque
// morceau lu). Une entrée vide donne 0, 0, 0 et pas d'erreur. Un texte sans
// '\n' final compte 0 ligne (comme wc). io.EOF n'est PAS une erreur : c'est
// la fin normale. Toute autre erreur du Reader est emballée avec %w.
// TODO 5 : bufio.NewReader(r) puis une boucle sur br.ReadString('\n').
func Compter(r io.Reader) (lignes, mots, octets int, err error) {
	return 0, 0, 0, errors.New("TODO")
}

// EcrireRapport écrit une ligne par forme puis une ligne de total, dans w.
// Format exact (voir le test) :
//
//	rectangle 3x4    aire=   12.00  périmètre=   14.00
//	total            aire=   21.14
//
// c'est-à-dire fmt.Fprintf(w, "%-16v aire=%8.2f  périmètre=%8.2f\n", f, f.Aire(), f.Perimetre())
// puis "%-16s aire=%8.2f\n" avec "total" et Total(formes).
// Chaque Fprintf renvoie (n int, err error) : teste err et emballe-la.
// TODO 6.
func EcrireRapport(w io.Writer, formes []Forme) error {
	return errors.New("TODO")
}
