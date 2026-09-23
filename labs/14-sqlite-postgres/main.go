// Labo 14 : l'outil en ligne de commande `inventaire` (fourni complet, rien à faire ici).
//
// Le DSN vient de la variable d'environnement INVENTAIRE_DSN (un chemin de
// fichier SQLite par défaut, ou une URL postgres://...). La commande vient de
// la ligne de commande. main() ne fait que lire l'environnement et les
// arguments ; toute la logique est dans executer, qui est testée.
// Lancé par : INVENTAIRE_DSN=inventaire.db go run . lister
package main

import (
	"context"
	"encoding/json" // pour la commande importer
	"errors"
	"fmt"
	"io" // executer écrit dans un io.Writer, pas sur os.Stdout en dur
	"os"
	"time"
)

// dsnParDefaut : un fichier SQLite dans le dossier courant. Il est dans le
// .gitignore du labo.
const dsnParDefaut = "inventaire.db"

const usage = `usage : inventaire <commande> [arguments]
  ajouter NOM IP ROLE      insère un hôte
  lister [ROLE]            liste les hôtes (d'un rôle), triés par nom
  vu NOM                   note que l'hôte vient de répondre
  supprimer NOM            retire un hôte
  importer FICHIER.json    insère tous les hôtes du fichier (tout ou rien)
  version                  affiche le moteur et la version du schéma
La base est choisie par INVENTAIRE_DSN : un chemin de fichier SQLite
(défaut : inventaire.db) ou une URL postgres://utilisateur:motdepasse@hote/base`

func main() {
	dsn := os.Getenv("INVENTAIRE_DSN")
	if dsn == "" {
		dsn = dsnParDefaut
	}
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	// Un délai global : si la base ne répond pas en 10 s, on abandonne.
	ctx, annuler := context.WithTimeout(context.Background(), 10*time.Second)
	defer annuler()

	store, err := Ouvrir(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := executer(ctx, store, os.Args[1:], os.Stdout, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

// executer interprète une commande. `maintenant` est passé en paramètre pour
// que les tests fixent l'heure ; `out` pour qu'ils capturent la sortie.
func executer(ctx context.Context, store *SQLStore, args []string, out io.Writer, maintenant time.Time) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	commande, reste := args[0], args[1:]

	switch commande {
	case "ajouter":
		if len(reste) != 3 {
			return errors.New("ajouter NOM IP ROLE")
		}
		if err := store.Ajouter(ctx, Hote{Nom: reste[0], IP: reste[1], Role: reste[2]}); err != nil {
			return err
		}
		fmt.Fprintf(out, "ajouté %s\n", reste[0])

	case "lister":
		role := ""
		if len(reste) == 1 {
			role = reste[0]
		}
		hotes, err := store.Lister(ctx, role)
		if err != nil {
			return err
		}
		afficher(out, hotes)

	case "vu":
		if len(reste) != 1 {
			return errors.New("vu NOM")
		}
		if err := store.MarquerVu(ctx, reste[0], maintenant); err != nil {
			return err
		}
		fmt.Fprintf(out, "%s vu le %s\n", reste[0], normaliser(maintenant).Format(time.DateTime))

	case "supprimer":
		if len(reste) != 1 {
			return errors.New("supprimer NOM")
		}
		if err := store.Supprimer(ctx, reste[0]); err != nil {
			return err
		}
		fmt.Fprintf(out, "supprimé %s\n", reste[0])

	case "importer":
		if len(reste) != 1 {
			return errors.New("importer FICHIER.json")
		}
		hotes, err := chargerJSON(reste[0])
		if err != nil {
			return err
		}
		n, err := store.Importer(ctx, hotes)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%d hôte(s) importé(s)\n", n)

	case "version":
		v, err := Version(ctx, store.db)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "moteur %s, schéma version %d\n", store.Dialecte(), v)

	default:
		return fmt.Errorf("commande inconnue %q\n%s", commande, usage)
	}
	return nil
}

// afficher écrit un tableau à colonnes fixes ; "jamais" pour un NULL.
func afficher(out io.Writer, hotes []Hote) {
	fmt.Fprintf(out, "%-10s %-15s %-8s %s\n", "NOM", "IP", "ROLE", "DERNIER VU")
	for _, h := range hotes {
		vu := "jamais"
		if h.DernierVu != nil {
			vu = h.DernierVu.Format(time.DateTime) // "2006-01-02 15:04:05"
		}
		fmt.Fprintf(out, "%-10s %-15s %-8s %s\n", h.Nom, h.IP, h.Role, vu)
	}
}

// chargerJSON lit un tableau JSON d'hôtes (voir exemples/hotes.json).
func chargerJSON(chemin string) ([]Hote, error) {
	f, err := os.Open(chemin)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var hotes []Hote
	if err := json.NewDecoder(f).Decode(&hotes); err != nil {
		return nil, fmt.Errorf("lire %s : %w", chemin, err)
	}
	return hotes, nil
}
