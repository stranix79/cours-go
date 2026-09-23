// Labo 14 : les migrations de schéma, à compléter.
//
// Les fichiers migrations/NNNN_nom.sql sont embarqués dans le binaire
// (go:embed) et appliqués dans l'ordre au démarrage. La table schema_version
// note ce qui a déjà été appliqué. numeroDe et Version sont fournies ;
// migrer et appliquer sont à écrire.
// Appelé par : Ouvrir (inventaire.go), main.go (commande version), les tests.
package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

// La directive go:embed copie les fichiers qui correspondent au motif dans
// le binaire, dans un système de fichiers en lecture seule.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// sqlSchemaVersion crée la table de suivi si elle manque.
const sqlSchemaVersion = `CREATE TABLE IF NOT EXISTS schema_version (
    version     INTEGER PRIMARY KEY,
    nom         TEXT NOT NULL,
    applique_le TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)`

// numeroDe extrait le numéro d'un nom de fichier : "0002_index_role.sql" → 2.
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

// Version lit la version courante du schéma (0 si rien n'a été appliqué).
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
// TODO 7 :
//  1. ExecContext(sqlSchemaVersion) puis courante := Version(...).
//  2. entrees, _ := fs.ReadDir(migrationsFS, "migrations") : déjà triées par nom.
//  3. Pour chaque entrée : version := numeroDe(nom) ; si version <= courante,
//     continue ; sinon contenu := migrationsFS.ReadFile("migrations/"+nom)
//     et appliquer(ctx, db, version, nom, string(contenu)).
func migrer(ctx context.Context, db *sql.DB) error {
	_ = fs.ReadDir // à retirer quand tu utilises fs.ReadDir pour de vrai
	return errors.New("TODO migrer")
}

// appliquer exécute une migration et note son numéro dans schema_version,
// dans une seule transaction (si le SQL échoue, la version n'est pas notée).
// TODO 8 : BeginTx, defer tx.Rollback(), tx.ExecContext(ctx, contenu),
// tx.ExecContext(INSERT INTO schema_version (version, nom) VALUES ($1, $2)),
// tx.Commit().
func appliquer(ctx context.Context, db *sql.DB, version int, nom, contenu string) error {
	return errors.New("TODO appliquer")
}
