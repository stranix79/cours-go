// Labo 12 : logstat, la ligne de commande.
// Lancé par : go run . <count|top|json> [-v] [-n 5] [fichier]
//
// run() fait tout le travail à partir de ses arguments et de ses flux ; main()
// ne fait que lui passer les vrais (os.Args, os.Stdin...) et sortir avec son
// code. Les tests appellent run() avec des tampons : pas de processus.
package main

import (
	"fmt"
	"io"
	"os"
)

const usage = `usage : logstat <count|top|json> [-v] [-n N] [fichier]
  count   nombre de lignes par niveau
  top     les N messages les plus fréquents (-n, défaut 5)
  json    total, comptes par niveau et top, en JSON indenté
Sans fichier, lit l'entrée standard. -v affiche le détail sur stderr.
`

// run exécute logstat et rend le code de sortie : 0 succès, 1 erreur
// d'exécution (fichier illisible, ligne malformée), 2 erreur d'usage.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	// TODO 7 : pas d'argument → usage sur stderr, 2. Sous-commande inconnue
	// → message + usage sur stderr, 2. Sinon un flag.NewFlagSet(cmd,
	// flag.ContinueOnError) avec SetOutput(stderr), les options -v (bool) et
	// -n (int, 5) ; Parse(args[1:]) en erreur → 2.
	//
	// TODO 8 : un slog.Logger sur stderr (TextHandler), niveau Debug si -v,
	// Info sinon. Source = os.Open(fs.Arg(0)) s'il y a un argument (erreur →
	// message sur stderr, 1 ; defer Close), stdin sinon. Un logger.Debug pour
	// dire d'où on lit.
	//
	// TODO 9 : Lire(source) ; erreur → "logstat : <err>" sur stderr, 1.
	// Puis selon cmd : count → FormatCompte(Compter(entrees)) sur stdout ;
	// top → une ligne "%5d  %s\n" par Frequence de Top(entrees, n) ;
	// json → EncoderJSON(stdout, Stats{...}). Rends 0.
	fmt.Fprint(stderr, usage)
	return 2
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
