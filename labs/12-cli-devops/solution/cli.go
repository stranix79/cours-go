// Labo 12, solution : logstat, la ligne de commande.
// Lancé par : go run ./solution <count|top|json> [-v] [-n 5] [fichier]
//
// run() reçoit ses arguments et ses trois flux en paramètres : c'est ce qui
// permet aux tests de l'appeler avec des tampons, sans processus ni fichier.
// main() lui passe les vrais et sort avec le code rendu.
package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
)

const usage = `usage : logstat <count|top|json> [-v] [-n N] [fichier]
  count   nombre de lignes par niveau
  top     les N messages les plus fréquents (-n, défaut 5)
  json    total, comptes par niveau et top, en JSON indenté
Sans fichier, lit l'entrée standard. -v affiche le détail sur stderr.
`

// run exécute logstat et rend le code de sortie : 0 succès, 1 erreur
// d'exécution, 2 erreur d'usage. Il n'appelle jamais os.Exit ni os.Stdout :
// tout passe par ses paramètres.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	// 1. La sous-commande, avant les options : logstat top -n 3 fichier.
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	cmd := args[0]
	if cmd != "count" && cmd != "top" && cmd != "json" {
		fmt.Fprintf(stderr, "logstat : sous-commande inconnue %q\n%s", cmd, usage)
		return 2
	}

	// 2. Les options de la sous-commande. ContinueOnError rend l'erreur au
	// lieu de quitter le processus (ExitOnError tuerait aussi les tests) ;
	// SetOutput envoie l'aide et les erreurs de flag sur NOTRE stderr.
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	verbeux := fs.Bool("v", false, "détail sur stderr")
	n := fs.Int("n", 5, "nombre de messages pour top et json")
	if err := fs.Parse(args[1:]); err != nil {
		return 2 // flag a déjà écrit le message et l'usage sur stderr
	}

	// 3. Les logs : texte sur stderr, niveau Debug seulement avec -v. On ne
	// touche pas au logger global (slog.SetDefault) pour rester testable.
	niveau := slog.LevelInfo
	if *verbeux {
		niveau = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: niveau}))

	// 4. La source : le fichier donné, sinon stdin. `source` est un io.Reader
	// dans les deux cas, Lire ne fait pas la différence.
	var source io.Reader = stdin
	nom := "stdin"
	if fs.NArg() > 0 {
		nom = fs.Arg(0)
		f, err := os.Open(nom)
		if err != nil {
			fmt.Fprintln(stderr, "logstat :", err) // ex : open x.log: no such file or directory
			return 1
		}
		defer f.Close()
		source = f
	}
	logger.Debug("lecture", "source", nom, "commande", cmd)

	// 5. L'analyse : une ligne malformée est une erreur d'exécution (1), pas
	// d'usage (2) : l'utilisateur a bien appelé l'outil, ce sont les données
	// qui sont mauvaises.
	entrees, err := Lire(source)
	if err != nil {
		fmt.Fprintln(stderr, "logstat :", err)
		return 1
	}
	logger.Debug("analyse terminée", "entrees", len(entrees))

	// 6. Le résultat, sur stdout uniquement.
	switch cmd {
	case "count":
		fmt.Fprint(stdout, FormatCompte(Compter(entrees)))
	case "top":
		for _, f := range Top(entrees, *n) {
			fmt.Fprintf(stdout, "%5d  %s\n", f.Nombre, f.Message)
		}
	case "json":
		stats := Stats{Total: len(entrees), ParNiveau: Compter(entrees), Top: Top(entrees, *n)}
		if err := EncoderJSON(stdout, stats); err != nil {
			fmt.Fprintln(stderr, "logstat :", err)
			return 1
		}
	}
	return 0
}

func main() {
	// os.Exit n'exécute pas les defer : c'est pour ça que run() fait tout,
	// y compris fermer le fichier, et que main ne fait que sortir.
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
