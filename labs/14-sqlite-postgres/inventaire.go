// Labo 14 : le dépôt Inventaire, à compléter.
//
// Ce fichier définit la donnée (Hote), le contrat (l'interface Store) et une
// implémentation database/sql, SQLStore, qui doit marcher telle quelle sur
// SQLite (modernc.org/sqlite) et sur PostgreSQL (pgx en mode database/sql).
// Ouvrir est fournie ; les six méthodes de SQLStore sont à écrire.
// Appelé par : main.go (les commandes) et inventaire_test.go.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	// Les pilotes s'enregistrent dans database/sql par leur init() : on les
	// importe pour l'effet de bord, d'où le `_`.
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// Hote est une ligne de la table hotes (voir migrations/0001_hotes.sql).
type Hote struct {
	Nom  string `json:"nom"`
	IP   string `json:"ip"`
	Role string `json:"role"`
	// nil = jamais vu (NULL en base). Toujours en UTC, à la microseconde.
	DernierVu *time.Time `json:"dernier_vu,omitempty"`
}

// Les erreurs sentinelles : l'appelant les teste avec errors.Is.
var (
	ErrExiste      = errors.New("hôte déjà présent")
	ErrIntrouvable = errors.New("hôte introuvable")
)

// Store est le contrat du dépôt.
type Store interface {
	// Ajouter insère un hôte ; ErrExiste (enveloppée) si le nom est déjà pris.
	Ajouter(ctx context.Context, h Hote) error
	// Trouver renvoie l'hôte de ce nom ; ErrIntrouvable sinon.
	Trouver(ctx context.Context, nom string) (Hote, error)
	// Lister renvoie les hôtes d'un rôle, triés par nom ; role vide = tous.
	Lister(ctx context.Context, role string) ([]Hote, error)
	// MarquerVu enregistre la date de dernier contact ; ErrIntrouvable sinon.
	MarquerVu(ctx context.Context, nom string, quand time.Time) error
	// Supprimer retire un hôte ; ErrIntrouvable sinon.
	Supprimer(ctx context.Context, nom string) error
	// Importer insère tous les hôtes dans UNE transaction : tout ou rien.
	// Renvoie le nombre insérés ; ErrExiste si l'un existe déjà, et dans ce
	// cas aucun n'est inséré.
	Importer(ctx context.Context, hotes []Hote) (int, error)
	// Close rend le pool de connexions.
	Close() error
}

// SQLStore est l'implémentation database/sql de Store.
type SQLStore struct {
	db       *sql.DB // le POOL de connexions, pas une connexion
	dialecte string  // "sqlite" ou "pgx"
}

// Vérification à la compilation que *SQLStore implémente Store.
var _ Store = (*SQLStore)(nil)

// Ouvrir devine le pilote depuis le DSN, ouvre le pool, vérifie la connexion
// et applique les migrations. Fournie : lis-la, elle contient deux réglages
// qui comptent (le pool à 1 pour SQLite, le Ping).
func Ouvrir(ctx context.Context, dsn string) (*SQLStore, error) {
	pilote := "sqlite"
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		pilote = "pgx"
	}

	// sql.Open ne se connecte pas : il prépare le pool.
	db, err := sql.Open(pilote, dsn)
	if err != nil {
		return nil, fmt.Errorf("ouvrir (%s) : %w", pilote, err)
	}

	if pilote == "sqlite" {
		// Une base ":memory:" est propre à chaque connexion, et SQLite n'a
		// qu'un écrivain à la fois : une seule connexion dans le pool.
		db.SetMaxOpenConns(1)
	} else {
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(30 * time.Minute)
	}

	// Ping force une vraie connexion : les erreurs de DSN sortent ici.
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("joindre la base (%s) : %w", pilote, err)
	}

	if err := migrer(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrer : %w", err)
	}

	return &SQLStore{db: db, dialecte: pilote}, nil
}

// Dialecte renvoie "sqlite" ou "pgx".
func (s *SQLStore) Dialecte() string { return s.dialecte }

// Close ferme le pool.
func (s *SQLStore) Close() error { return s.db.Close() }

// normaliser ramène une date en UTC à la microseconde : ce que PostgreSQL
// sait stocker. Applique-la à tout ce que tu écris ET à tout ce que tu lis.
func normaliser(t time.Time) time.Time {
	return t.UTC().Truncate(time.Microsecond)
}

// Écris chaque requête UNE fois, avec les paramètres $1, $2... : c'est la
// syntaxe de PostgreSQL, et le pilote SQLite l'accepte aussi. Jamais de
// fmt.Sprintf pour mettre une valeur dans du SQL.
const (
	sqlInsert = `INSERT INTO hotes (nom, ip, role, dernier_vu) VALUES ($1, $2, $3, $4)
	             ON CONFLICT (nom) DO NOTHING`
	// TODO : sqlSelect, sqlLister, sqlVu, sqlDelete (voir les consignes).
)

// Ajouter insère un hôte.
// TODO 1 : ExecContext avec sqlInsert (un *time.Time nil part comme NULL),
// puis res.RowsAffected() : 0 ligne = le nom existait déjà, renvoie
// fmt.Errorf("ajouter %s : %w", h.Nom, ErrExiste). Normalise DernierVu avant.
func (s *SQLStore) Ajouter(ctx context.Context, h Hote) error {
	return errors.New("TODO Ajouter")
}

// Trouver lit un hôte par son nom.
// TODO 2 : QueryRowContext + Scan dans les champs, avec un sql.NullTime pour
// dernier_vu (Valid=false quand NULL). errors.Is(err, sql.ErrNoRows) devient
// ErrIntrouvable.
func (s *SQLStore) Trouver(ctx context.Context, nom string) (Hote, error) {
	return Hote{}, errors.New("TODO Trouver")
}

// Lister renvoie les hôtes d'un rôle (tous si role est vide), triés par nom.
// TODO 3 : QueryContext, defer rows.Close(), boucle rows.Next() + Scan,
// et rows.Err() après la boucle. Le filtre tient dans le SQL :
// WHERE $1 = <chaîne vide> OR role = $1 (regarde sqlLister dans la solution si tu bloques).
func (s *SQLStore) Lister(ctx context.Context, role string) ([]Hote, error) {
	return nil, errors.New("TODO Lister")
}

// MarquerVu met à jour dernier_vu.
// TODO 4 : UPDATE ; RowsAffected() == 0 → ErrIntrouvable.
func (s *SQLStore) MarquerVu(ctx context.Context, nom string, quand time.Time) error {
	return errors.New("TODO MarquerVu")
}

// Supprimer retire un hôte.
// TODO 5 : DELETE ; RowsAffected() == 0 → ErrIntrouvable.
func (s *SQLStore) Supprimer(ctx context.Context, nom string) error {
	return errors.New("TODO Supprimer")
}

// Importer insère une liste d'hôtes dans une seule transaction.
// TODO 6 : BeginTx, defer tx.Rollback(), tx.PrepareContext(sqlInsert),
// une boucle d'ExecContext (0 ligne affectée → ErrExiste et return : le
// Rollback annule tout), puis tx.Commit(). Piège : n'utilise que tx (pas
// s.db) tant que la transaction est ouverte, SQLite n'a qu'une connexion.
func (s *SQLStore) Importer(ctx context.Context, hotes []Hote) (int, error) {
	return 0, errors.New("TODO Importer")
}
