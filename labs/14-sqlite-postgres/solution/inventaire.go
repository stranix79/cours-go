// Labo 14, solution : le dépôt Inventaire.
//
// Ce fichier définit la donnée (Hote), le contrat (l'interface Store) et une
// implémentation unique, SQLStore, écrite avec database/sql. Le même code
// parle à SQLite (pilote modernc.org/sqlite, pur Go) et à PostgreSQL (pilote
// pgx en mode database/sql) : seul le DSN change.
// Appelé par : main.go (les commandes) et inventaire_test.go (sur SQLite en
// mémoire, ou sur PostgreSQL si INVENTAIRE_TEST_DSN est définie).
package main

import (
	"context"      // annulation et délais, passés à chaque requête
	"database/sql" // l'API générique : DB, Tx, Rows, NullTime...
	"errors"       // errors.New et errors.Is
	"fmt"          // fmt.Errorf avec %w pour envelopper
	"strings"      // HasPrefix pour deviner le pilote depuis le DSN
	"time"         // le champ dernier_vu

	// Les deux pilotes ne sont importés que pour leur effet de bord : leur
	// init() appelle sql.Register("sqlite", ...) et sql.Register("pgx", ...).
	// L'identifiant blanc `_` dit au compilateur qu'on n'utilise aucun symbole.
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// Hote est une ligne de la table hotes. Les tags json servent à la commande
// `importer` (fichier JSON) et à rien d'autre : database/sql ne lit pas les
// tags, c'est Scan qui fait la correspondance colonne par colonne.
type Hote struct {
	Nom  string `json:"nom"`
	IP   string `json:"ip"`
	Role string `json:"role"`
	// Un pointeur pour représenter le NULL : nil = jamais vu.
	// Toujours stocké en UTC, à la microseconde (la précision de PostgreSQL).
	DernierVu *time.Time `json:"dernier_vu,omitempty"`
}

// Les erreurs sentinelles du dépôt : l'appelant les teste avec errors.Is,
// sans avoir à connaître les codes d'erreur de SQLite ou de PostgreSQL.
var (
	ErrExiste      = errors.New("hôte déjà présent")
	ErrIntrouvable = errors.New("hôte introuvable")
)

// Store est le contrat du dépôt. main.go et les tests ne connaissent que
// cette interface : on pourrait la réimplémenter en mémoire ou avec pgx natif
// sans toucher au reste du programme.
type Store interface {
	// Ajouter insère un hôte ; ErrExiste si le nom est déjà pris.
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
	// Renvoie le nombre insérés ; ErrExiste (enveloppée) si l'un existe déjà,
	// et dans ce cas aucun n'est inséré.
	Importer(ctx context.Context, hotes []Hote) (int, error)
	// Close rend le pool de connexions.
	Close() error
}

// SQLStore est l'implémentation database/sql de Store.
type SQLStore struct {
	db       *sql.DB // le POOL de connexions, pas une connexion
	dialecte string  // "sqlite" ou "pgx", pour l'affichage et les réglages
}

// Cette ligne ne fait rien à l'exécution : elle demande au compilateur de
// vérifier que *SQLStore implémente bien Store. Si une méthode manque ou
// change de signature, l'erreur apparaît ici, pas chez l'appelant.
var _ Store = (*SQLStore)(nil)

// Ouvrir devine le pilote depuis le DSN, ouvre le pool, vérifie la connexion
// et applique les migrations manquantes. C'est la seule fonction qui connaît
// les deux moteurs.
func Ouvrir(ctx context.Context, dsn string) (*SQLStore, error) {
	pilote := "sqlite"
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		pilote = "pgx"
	}

	// sql.Open ne se connecte PAS : il valide le nom du pilote et prépare le
	// pool. La première connexion réelle a lieu au premier Ping ou à la
	// première requête.
	db, err := sql.Open(pilote, dsn)
	if err != nil {
		return nil, fmt.Errorf("ouvrir (%s) : %w", pilote, err)
	}

	if pilote == "sqlite" {
		// SQLite n'accepte qu'un écrivain à la fois, et une base ":memory:"
		// est PROPRE À CHAQUE CONNEXION : avec un pool de deux connexions,
		// la seconde verrait une base vide. Une seule connexion règle les
		// deux problèmes.
		db.SetMaxOpenConns(1)
	} else {
		// PostgreSQL : un pool raisonnable. Chaque connexion coûte un
		// processus côté serveur ; 10 suffit largement à ce programme.
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(30 * time.Minute)
	}

	// Ping force une vraie connexion : c'est ici qu'un mot de passe faux ou un
	// serveur éteint se manifestent, au démarrage et pas sur la première
	// requête d'un utilisateur.
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("joindre la base (%s) : %w", pilote, err)
	}

	// Les migrations (migrations.go) amènent le schéma à la dernière version.
	if err := migrer(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrer : %w", err)
	}

	return &SQLStore{db: db, dialecte: pilote}, nil
}

// Dialecte renvoie "sqlite" ou "pgx" ; main.go l'affiche dans `version`.
func (s *SQLStore) Dialecte() string { return s.dialecte }

// Close ferme le pool. Toujours appelée en defer par l'appelant de Ouvrir.
func (s *SQLStore) Close() error { return s.db.Close() }

// normaliser ramène une date en UTC à la microseconde, pour que ce qu'on
// relit soit égal (time.Equal) à ce qu'on a écrit, sur les deux moteurs.
func normaliser(t time.Time) time.Time {
	return t.UTC().Truncate(time.Microsecond)
}

// Les requêtes sont écrites une seule fois, avec les paramètres $1, $2...
// C'est la syntaxe de PostgreSQL, et SQLite l'accepte aussi (il y voit un
// paramètre nommé "1", que le pilote modernc relie au premier argument).
// Le SQL est identique sur les deux moteurs : c'est ce qui rend ce dépôt
// portable sans une ligne de code par dialecte.
const (
	sqlInsert = `INSERT INTO hotes (nom, ip, role, dernier_vu) VALUES ($1, $2, $3, $4)
	             ON CONFLICT (nom) DO NOTHING`
	sqlSelect = `SELECT nom, ip, role, dernier_vu FROM hotes WHERE nom = $1`
	sqlLister = `SELECT nom, ip, role, dernier_vu FROM hotes
	             WHERE $1 = '' OR role = $1 ORDER BY nom`
	sqlVu     = `UPDATE hotes SET dernier_vu = $2 WHERE nom = $1`
	sqlDelete = `DELETE FROM hotes WHERE nom = $1`
)

// Ajouter insère un hôte. Le doublon est détecté avec ON CONFLICT DO NOTHING
// plus le nombre de lignes affectées : zéro ligne = le nom existait. C'est
// portable, alors que lire le code d'erreur (23505 chez PostgreSQL, "UNIQUE
// constraint failed" chez SQLite) ne l'est pas.
func (s *SQLStore) Ajouter(ctx context.Context, h Hote) error {
	if h.DernierVu != nil {
		t := normaliser(*h.DernierVu)
		h.DernierVu = &t
	}
	// Exec pour tout ce qui ne renvoie pas de lignes. Un *time.Time nil est
	// envoyé comme NULL, un non-nil comme sa valeur : database/sql déréférence.
	res, err := s.db.ExecContext(ctx, sqlInsert, h.Nom, h.IP, h.Role, h.DernierVu)
	if err != nil {
		return fmt.Errorf("ajouter %s : %w", h.Nom, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ajouter %s : %w", h.Nom, err)
	}
	if n == 0 {
		return fmt.Errorf("ajouter %s : %w", h.Nom, ErrExiste)
	}
	return nil
}

// scanner lit une ligne (Row ou Rows : les deux ont Scan) dans un Hote.
// L'interface locale évite de dupliquer la fonction pour QueryRow et Query.
type scanneur interface {
	Scan(dest ...any) error
}

func scanner(ligne scanneur) (Hote, error) {
	var h Hote
	// sql.NullTime : Valid=false quand la colonne est NULL. On aurait pu
	// scanner directement dans &h.DernierVu (un **time.Time) : database/sql
	// sait mettre nil pour NULL. NullTime rend le NULL visible dans le code.
	var vu sql.NullTime
	if err := ligne.Scan(&h.Nom, &h.IP, &h.Role, &vu); err != nil {
		return Hote{}, err
	}
	if vu.Valid {
		t := normaliser(vu.Time)
		h.DernierVu = &t
	}
	return h, nil
}

// Trouver lit un hôte par son nom. QueryRow ne renvoie jamais nil : l'erreur
// (dont sql.ErrNoRows) sort au moment du Scan.
func (s *SQLStore) Trouver(ctx context.Context, nom string) (Hote, error) {
	h, err := scanner(s.db.QueryRowContext(ctx, sqlSelect, nom))
	if errors.Is(err, sql.ErrNoRows) {
		return Hote{}, fmt.Errorf("trouver %s : %w", nom, ErrIntrouvable)
	}
	if err != nil {
		return Hote{}, fmt.Errorf("trouver %s : %w", nom, err)
	}
	return h, nil
}

// Lister renvoie les hôtes d'un rôle (tous si role est vide), triés par nom.
// Le filtre est fait en SQL (`$1 = chaîne vide OR role = $1`), un seul paramètre.
func (s *SQLStore) Lister(ctx context.Context, role string) ([]Hote, error) {
	rows, err := s.db.QueryContext(ctx, sqlLister, role)
	if err != nil {
		return nil, fmt.Errorf("lister : %w", err)
	}
	// Rows tient une connexion du pool tant qu'il n'est pas fermé ou lu
	// jusqu'au bout. Le defer garantit la fermeture même en cas d'erreur.
	defer rows.Close()

	var hotes []Hote
	for rows.Next() {
		h, err := scanner(rows)
		if err != nil {
			return nil, fmt.Errorf("lister : %w", err)
		}
		hotes = append(hotes, h)
	}
	// rows.Next renvoie false à la fin ET en cas d'erreur réseau : rows.Err
	// fait la différence. L'oublier, c'est prendre une liste tronquée pour
	// une liste complète.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lister : %w", err)
	}
	return hotes, nil
}

// MarquerVu met à jour dernier_vu. RowsAffected == 0 veut dire que le nom
// n'existe pas (UPDATE sans ligne n'est pas une erreur SQL).
func (s *SQLStore) MarquerVu(ctx context.Context, nom string, quand time.Time) error {
	res, err := s.db.ExecContext(ctx, sqlVu, nom, normaliser(quand))
	if err != nil {
		return fmt.Errorf("marquer vu %s : %w", nom, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("marquer vu %s : %w", nom, ErrIntrouvable)
	}
	return nil
}

// Supprimer retire un hôte, même logique que MarquerVu.
func (s *SQLStore) Supprimer(ctx context.Context, nom string) error {
	res, err := s.db.ExecContext(ctx, sqlDelete, nom)
	if err != nil {
		return fmt.Errorf("supprimer %s : %w", nom, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("supprimer %s : %w", nom, ErrIntrouvable)
	}
	return nil
}

// Importer insère une liste d'hôtes dans une seule transaction.
// Tout ou rien : au premier doublon, on renvoie l'erreur et le Rollback du
// defer annule ce qui a été inséré avant.
func (s *SQLStore) Importer(ctx context.Context, hotes []Hote) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil) // nil = options par défaut (isolation, lecture-écriture)
	if err != nil {
		return 0, fmt.Errorf("importer : %w", err)
	}
	// Le motif canonique : Rollback en defer. Après un Commit réussi, il
	// renvoie sql.ErrTxDone et ne fait rien ; sur toute sortie prématurée
	// (return d'erreur, panic), il annule. On ignore sa valeur de retour.
	defer tx.Rollback() //nolint:errcheck

	// Une requête préparée : envoyée une fois au serveur, exécutée N fois avec
	// des arguments différents. Sur un import de 10 000 lignes, ça compte.
	// Piège : avec SQLite limité à une connexion, tout doit passer par tx
	// (pas par s.db) tant que la transaction est ouverte, sinon l'appel
	// attend une connexion que la transaction ne rendra jamais.
	stmt, err := tx.PrepareContext(ctx, sqlInsert)
	if err != nil {
		return 0, fmt.Errorf("importer : %w", err)
	}
	defer stmt.Close()

	inseres := 0
	for _, h := range hotes {
		if h.DernierVu != nil {
			t := normaliser(*h.DernierVu)
			h.DernierVu = &t
		}
		res, err := stmt.ExecContext(ctx, h.Nom, h.IP, h.Role, h.DernierVu)
		if err != nil {
			return 0, fmt.Errorf("importer %s : %w", h.Nom, err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return 0, fmt.Errorf("importer %s : %w", h.Nom, ErrExiste)
		}
		inseres++
	}

	// Commit rend tout visible d'un coup. S'il échoue (réseau coupé, conflit
	// de sérialisation), rien n'est écrit.
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("importer : %w", err)
	}
	return inseres, nil
}
