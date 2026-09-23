// Labo 14, solution : les migrations de schéma.
//
// Les fichiers migrations/NNNN_nom.sql sont embarqués dans le binaire avec
// //go:embed, puis appliqués dans l'ordre au démarrage, chacun dans sa propre
// transaction. Une table schema_version note ce qui a déjà été appliqué :
// relancer le programme n'applique que les nouveautés.
// Appelé par : Ouvrir (inventaire.go) ; Version est utilisée par main.go
// (commande `version`) et par les tests.
package main

import (
	"context"
	"database/sql"
	"embed"   // le type embed.FS
	"fmt"     // fmt.Errorf
	"io/fs"   // fs.ReadDir sur l'embed.FS
	"strconv" // Atoi pour le numéro de version
	"strings" // Cut pour séparer "0001" de "_hotes.sql"
)

// La directive go:embed, sur la ligne juste au-dessus de la variable, demande
// au compilateur de copier les fichiers qui correspondent au motif dans le
// binaire. Le dossier migrations/ n'a plus besoin d'exister sur le serveur.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// sqlSchemaVersion crée la table de suivi si elle manque. Ce SQL est le seul
// qui n'est PAS dans un fichier : il faut bien une première fois.
const sqlSchemaVersion = `CREATE TABLE IF NOT EXISTS schema_version (
    version     INTEGER PRIMARY KEY,
    nom         TEXT NOT NULL,
    applique_le TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)`

// numeroDe extrait le numéro de version d'un nom de fichier :
// "0002_index_role.sql" donne 2. Le préfixe numérique fixe l'ordre, le reste
// du nom est pour les humains.
func numeroDe(nom string) (int, error) {
	prefixe, _, ok := strings.Cut(nom, "_")
	if !ok {
		return 0, fmt.Errorf("migration %q : nom attendu NNNN_description.sql", nom)
	}
	n, err := strconv.Atoi(prefixe)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("migration %q : numéro invalide", nom)
	}
	return n, nil
}

// Version lit la version courante du schéma : le plus grand numéro appliqué,
// 0 si la table est vide. COALESCE transforme le NULL de MAX() sur une table
// vide en 0, ce qui évite un sql.NullInt64.
func Version(ctx context.Context, db *sql.DB) (int, error) {
	var v int
	err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&v)
	if err != nil {
		return 0, fmt.Errorf("version du schéma : %w", err)
	}
	return v, nil
}

// migrer applique les migrations dont le numéro est supérieur à la version
// courante, dans l'ordre des noms de fichiers.
func migrer(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, sqlSchemaVersion); err != nil {
		return fmt.Errorf("créer schema_version : %w", err)
	}
	courante, err := Version(ctx, db)
	if err != nil {
		return err
	}

	// fs.ReadDir renvoie les entrées triées par nom : avec des numéros sur
	// quatre chiffres, l'ordre alphabétique est l'ordre numérique.
	entrees, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("lire les migrations embarquées : %w", err)
	}

	for _, e := range entrees {
		version, err := numeroDe(e.Name())
		if err != nil {
			return err
		}
		if version <= courante {
			continue // déjà appliquée lors d'un démarrage précédent
		}
		contenu, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return fmt.Errorf("lire %s : %w", e.Name(), err)
		}
		if err := appliquer(ctx, db, version, e.Name(), string(contenu)); err != nil {
			return err
		}
	}
	return nil
}

// appliquer exécute une migration et enregistre son numéro, dans une seule
// transaction : si le SQL échoue à mi-chemin, la table schema_version ne la
// croit pas appliquée, et la relance du programme réessaie. SQLite et
// PostgreSQL savent annuler un CREATE TABLE (DDL transactionnel) ; MySQL non.
func appliquer(ctx context.Context, db *sql.DB, version int, nom, contenu string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migration %s : %w", nom, err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Un fichier peut contenir plusieurs ordres séparés par des ";" : sans
	// argument, les deux pilotes les exécutent en séquence.
	if _, err := tx.ExecContext(ctx, contenu); err != nil {
		return fmt.Errorf("migration %s : %w", nom, err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO schema_version (version, nom) VALUES ($1, $2)`, version, nom)
	if err != nil {
		return fmt.Errorf("migration %s : noter la version : %w", nom, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migration %s : %w", nom, err)
	}
	return nil
}
