# 14. Bases de données : database/sql, SQLite, PostgreSQL

*Go de zéro à la prod : chapitre 14 sur 16.* ← [13. HTTP : client, serveur, API JSON](13-http.md) · [Sommaire](../README.md) · [15. Qualité, build, release, Docker, observabilité](15-qualite-build-release.md) →

L'API du chapitre 13 garde ses tâches en mémoire : on redémarre, tout disparaît. Il est temps de parler à une vraie base. Bonne nouvelle pour toi qui administres PostgreSQL : Go ne te cache rien. Pas d'ORM obligatoire, pas de couche magique, tu écris ton SQL, tu passes tes paramètres, tu lis tes lignes. La bibliothèque standard fournit une API générique, `database/sql`, et des *pilotes* tiers s'y branchent : un pour SQLite, un pour PostgreSQL, un pour MySQL. Ton code ne change pas quand tu changes de moteur, sauf le SQL lui-même, et c'est très bien ainsi : tu sais mieux que n'importe quel ORM quel SQL tu veux envoyer.

Ce chapitre suit l'ordre dans lequel tu vas te cogner aux choses : le pool, les requêtes, le NULL, les transactions, le contexte, l'injection, puis les deux moteurs (SQLite pour le développement et les tests, PostgreSQL pour la production), les migrations, et ce qu'un DBA doit savoir sur ce qu'un programme Go fait à son serveur.

### 14.1 `database/sql` : un pool, pas une connexion

`database/sql` définit les types (`DB`, `Tx`, `Rows`, `Row`, `Result`) et laisse aux pilotes le protocole réseau. Un pilote s'enregistre auprès du paquet dans sa fonction `init()` ; on l'importe donc pour son effet de bord, avec l'identifiant blanc :

```go
import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib" // enregistre le pilote "pgx"
	_ "modernc.org/sqlite"             // enregistre le pilote "sqlite"
)
```

Puis `sql.Open(nomDuPilote, dsn)`. Le *DSN* (*data source name*) dit où est la base : un chemin de fichier pour SQLite, une URL `postgres://utilisateur:motdepasse@hote:5432/base?sslmode=require` pour PostgreSQL. Première surprise, et elle coûte une heure à tout le monde :

**Piège :** `sql.Open` ne se connecte pas. Il vérifie que le pilote existe et prépare un *pool* de connexions vide. Un mot de passe faux, un serveur éteint, un port fermé : `Open` réussit quand même. La première vraie connexion a lieu à la première requête, ou au `Ping`, qu'on appelle donc toujours au démarrage pour échouer tôt et avec un message clair.

```go
db, err := sql.Open("pgx", "postgres://postgres@127.0.0.1:5498/inventaire?sslmode=disable")
fmt.Println("Open  :", err)
ctx, annuler := context.WithTimeout(context.Background(), 2*time.Second)
defer annuler()
fmt.Println("Ping  :", db.PingContext(ctx))

db2, _ := sql.Open("sqlite", "/tmp/ch14-demo.db")
fmt.Println("Ping  :", db2.PingContext(ctx))
fmt.Printf("Stats : %d ouverte(s), %d en attente\n", db2.Stats().OpenConnections, db2.Stats().InUse)
```

```
Open  : <nil>
Ping  : failed to connect to `user=postgres database=inventaire`: 127.0.0.1:5498 (127.0.0.1): dial error: dial tcp 127.0.0.1:5498: connect: connection refused
Ping  : <nil>
Stats : 1 ouverte(s), 0 en attente
```

Un `*sql.DB` est donc un **pool** : un objet unique, partagé par toutes les goroutines du programme (il est sûr en concurrence), qui ouvre des connexions à la demande, les réutilise et les ferme quand elles vieillissent. On en crée un par programme, jamais un par requête HTTP, et on le règle une fois :

```go
db.SetMaxOpenConns(10)                  // au plus 10 connexions simultanées (les autres attendent)
db.SetMaxIdleConns(5)                   // en garder 5 ouvertes au repos
db.SetConnMaxLifetime(30 * time.Minute) // recycler une connexion après 30 min
```

**Venant de Python :** `sqlite3.connect()` et `psycopg.connect()` renvoient *une* connexion, et c'est à toi (ou à SQLAlchemy) de faire le pool. Ici, le pool est dans la bibliothèque standard et tu ne vois jamais une connexion individuelle, sauf dans une transaction.

**Venant du C :** avec libpq, tu gères des `PGconn*` un par un et tu écris la file d'attente toi-même. `database/sql` est cette file d'attente, avec le recyclage et la sécurité entre threads en plus.

### 14.2 Exec, QueryRow, Query, Scan

Trois méthodes couvrent tout, et chacune a sa variante `...Context` (section 14.5) que tu utiliseras en vrai :

| Méthode | Pour | Renvoie |
|---|---|---|
| `db.Exec(sql, args...)` | INSERT, UPDATE, DELETE, DDL | un `Result` (`RowsAffected`, `LastInsertId`) |
| `db.QueryRow(sql, args...)` | un SELECT qui renvoie une ligne | un `*Row` : on appelle `.Scan` dessus |
| `db.Query(sql, args...)` | un SELECT qui renvoie N lignes | un `*Rows` à parcourir avec `Next` et `Scan` |

`Scan` copie les colonnes de la ligne courante dans les variables dont on passe l'adresse, dans l'ordre des colonnes du SELECT. Pas de tag, pas de réflexion : la correspondance est positionnelle, et le compilateur ne peut pas la vérifier pour toi. Nomme tes colonnes explicitement dans le SELECT (jamais `SELECT *`) et garde le `Scan` juste en dessous.

```go
res, err := db.Exec(`INSERT INTO sondes (cible, port) VALUES ($1, $2)`, "10.0.0.11", 443)
id, _ := res.LastInsertId()
n, _ := res.RowsAffected()
fmt.Printf("insérée : id=%d, %d ligne\n", id, n)

var s Sonde
err = db.QueryRow(`SELECT id, cible, port, active FROM sondes WHERE port = $1`, 5432).
	Scan(&s.ID, &s.Cible, &s.Port, &s.Active)
fmt.Printf("QueryRow : %+v, err=%v\n", s, err)

err = db.QueryRow(`SELECT id FROM sondes WHERE port = $1`, 80).Scan(&s.ID)
fmt.Println("QueryRow sans résultat :", err, "| ErrNoRows ?", err == sql.ErrNoRows)

rows, err := db.Query(`SELECT id, cible, port FROM sondes WHERE active = $1 ORDER BY port`, true)
if err != nil {
	log.Fatal(err)
}
defer rows.Close()
for rows.Next() {
	var s Sonde
	if err := rows.Scan(&s.ID, &s.Cible, &s.Port); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  %d %s:%d\n", s.ID, s.Cible, s.Port)
}
fmt.Println("rows.Err :", rows.Err())
```

```
insérée : id=1, 1 ligne
QueryRow : {ID:2 Cible:10.0.0.21 Port:5432 Active:false}, err=<nil>
QueryRow sans résultat : sql: no rows in result set | ErrNoRows ? true
  3 10.0.0.2:22
  1 10.0.0.11:443
rows.Err : <nil>
```

Quatre règles à graver :

1. `QueryRow` ne renvoie jamais d'erreur directement : elle sort au `Scan`. Une ligne absente donne `sql.ErrNoRows`, à tester avec `errors.Is` et à transformer en erreur métier (« hôte introuvable ») avant de remonter.
2. `Rows` tient une connexion du pool tant qu'il n'est pas fermé. `defer rows.Close()` juste après le test d'erreur, sinon, à la dixième requête oubliée, le pool est vide et tout le programme attend.
3. `rows.Next()` renvoie `false` à la fin **et** en cas d'erreur en cours de lecture (réseau coupé, serveur redémarré). `rows.Err()` après la boucle fait la différence entre « fini » et « tronqué ». L'oublier, c'est prendre une liste incomplète pour une liste complète, silencieusement.
4. `LastInsertId` marche avec SQLite et MySQL, pas avec PostgreSQL (qui n'a pas cette notion) : là on écrit `INSERT ... RETURNING id` et on lit avec `QueryRow`.

### 14.3 Le NULL

Un `string` Go ne peut pas être NULL, un `int` non plus. Scanner une colonne NULL dans un `string` est une erreur :

```go
var os string
err := db.QueryRow(`SELECT os FROM hotes WHERE nom = 'nas'`).Scan(&os)
fmt.Println("dans un string :", err)

var osN sql.NullString
db.QueryRow(`SELECT os FROM hotes WHERE nom = 'nas'`).Scan(&osN)
fmt.Printf("NullString      : Valid=%v String=%q\n", osN.Valid, osN.String)

var osP *string
db.QueryRow(`SELECT os FROM hotes WHERE nom = 'nas'`).Scan(&osP)
fmt.Println("pointeur (nas)  :", osP)
db.QueryRow(`SELECT os FROM hotes WHERE nom = 'web-01'`).Scan(&osP)
fmt.Println("pointeur (web)  :", *osP)

db.QueryRow(`SELECT COALESCE(os, 'inconnu') FROM hotes WHERE nom = 'nas'`).Scan(&os)
fmt.Println("COALESCE        :", os)
```

```
dans un string : sql: Scan error on column index 0, name "os": converting NULL to string is unsupported
NullString      : Valid=false String=""
pointeur (nas)  : <nil>
pointeur (web)  : debian
COALESCE        : inconnu
```

Trois façons, à choisir selon le sens du NULL : `sql.NullString` (et `NullInt64`, `NullTime`, `NullBool`, ou le générique `sql.Null[T]`) rend l'absence explicite dans le code ; un pointeur (`*string`, `*time.Time`) est plus court et se sérialise naturellement en JSON `null` ; `COALESCE` côté SQL quand une valeur par défaut a un sens métier. Dans l'autre sens, c'est symétrique : passer un pointeur nil ou un `NullString{Valid: false}` en argument envoie un NULL.

### 14.4 Transactions : Begin, Commit, Rollback et le defer

`db.BeginTx(ctx, nil)` prend une connexion du pool, ouvre une transaction et renvoie un `*sql.Tx` qui a les mêmes méthodes `Exec`, `QueryRow`, `Query`, mais sur *cette* connexion. Tout ce qui passe par `tx` est dans la transaction ; tout ce qui passe par `db` pendant ce temps est en dehors, sur une autre connexion. Le motif canonique tient en trois lignes et une habitude :

```go
func virer(ctx context.Context, db *sql.DB, de, vers string, go_ int) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // sans effet après Commit ; annule tout sur return anticipé

	if _, err := tx.ExecContext(ctx, `UPDATE quotas SET go = go - $1 WHERE projet = $2`, go_, de); err != nil {
		return err
	}
	var reste int
	if err := tx.QueryRowContext(ctx, `SELECT go FROM quotas WHERE projet = $1`, de).Scan(&reste); err != nil {
		return err
	}
	if reste < 0 {
		return errors.New("quota insuffisant") // le defer fait le Rollback
	}
	if _, err := tx.ExecContext(ctx, `UPDATE quotas SET go = go + $1 WHERE projet = $2`, go_, vers); err != nil {
		return err
	}
	return tx.Commit()
}
```

```
virer 30 : <nil>
virer 90 : quota insuffisant
  logs     80 Go
  sondes   70 Go
```

Le second virement a bien débité `sondes` de 90 avant de constater que le solde était négatif ; le `return` a déclenché le `defer tx.Rollback()`, et la table est comme avant. Après un `Commit` réussi, `Rollback` renvoie `sql.ErrTxDone` et ne fait rien : on ignore sa valeur de retour, c'est le seul endroit du cours où c'est légitime. Le `nil` passé à `BeginTx` est un `*sql.TxOptions` : `&sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: true}` si tu veux autre chose que le défaut du serveur.

**Venant de Python :** `with con:` fait le commit ou le rollback à la sortie du bloc. Le `defer tx.Rollback()` plus `return tx.Commit()` est l'équivalent, écrit à la main, et il a un avantage : on voit où le commit a lieu.

### 14.5 Le contexte : délais et annulation

Toutes les méthodes existent en version `...Context` : `ExecContext`, `QueryRowContext`, `QueryContext`, `BeginTx`, `PingContext`. Les versions sans contexte sont là pour la compatibilité ; en production, tu utilises toujours celles avec. Le contexte (chapitre 11) porte un délai ou une annulation, et le pilote la transmet au serveur :

```go
ctx, annuler := context.WithTimeout(context.Background(), 500*time.Millisecond)
defer annuler()

debut := time.Now()
var un int
err := db.QueryRowContext(ctx, `SELECT pg_sleep(3), 1`).Scan(nil, &un)
fmt.Printf("après %v : %v\n", time.Since(debut).Round(time.Millisecond), err)
fmt.Println("DeadlineExceeded ?", errors.Is(err, context.DeadlineExceeded))

time.Sleep(100 * time.Millisecond)
var actives int
db.QueryRow(`SELECT count(*) FROM pg_stat_activity
             WHERE query LIKE 'SELECT pg_sleep%' AND state = 'active' AND pid <> pg_backend_pid()`).Scan(&actives)
fmt.Println("requêtes pg_sleep encore actives côté serveur :", actives)
```

```
après 511ms : timeout: context deadline exceeded
DeadlineExceeded ? true
requêtes pg_sleep encore actives côté serveur : 0
```

La requête devait durer 3 s, le client a rendu la main après 500 ms, et pgx a envoyé une demande d'annulation au serveur : aucune requête zombie dans `pg_stat_activity`. C'est ton `statement_timeout`, mais décidé par l'appelant, par requête, et propagé depuis le handler HTTP : si le client HTTP raccroche, le contexte de la requête est annulé, et le SELECT de 3 s qu'elle avait lancé aussi. (Le `Scan(nil, &un)` ignore la première colonne, celle de `pg_sleep`.)

### 14.6 Jamais de Sprintf dans du SQL : les paramètres

```go
saisie := "x' OR '1'='1" // ce que tape un visiteur malveillant

requete := fmt.Sprintf("SELECT login FROM comptes WHERE login = '%s'", saisie)
fmt.Println(requete)
fmt.Println("  lignes :", compter(requete))

fmt.Println("SELECT login FROM comptes WHERE login = $1")
fmt.Println("  lignes :", compter("SELECT login FROM comptes WHERE login = $1", saisie))
```

```
SELECT login FROM comptes WHERE login = 'x' OR '1'='1'
  lignes : 3
SELECT login FROM comptes WHERE login = $1
  lignes : 0
```

Avec la concaténation, la saisie est devenue du SQL et la condition est toujours vraie : trois comptes au lieu de zéro. Avec un paramètre, la valeur est envoyée à part, jamais interprétée : zéro ligne, comme il se doit. Les données passent **toujours** par les paramètres ; la seule chose qu'on ne peut pas paramétrer, c'est un nom de table ou de colonne, et là on valide contre une liste blanche avant de concaténer.

La syntaxe des paramètres dépend du moteur : PostgreSQL veut `$1, $2` (numérotés, réutilisables : `$1` deux fois dans la requête, un seul argument), MySQL et SQLite veulent `?` (positionnels). SQLite accepte aussi `$1`, `:nom`, `@nom` : le labo écrit tout en `$1` et le même SQL tourne sur les deux moteurs. Si tu dois viser MySQL, tu auras une fonction qui réécrit `$n` en `?`, ou deux jeux de requêtes ; `sqlc` (section 14.10) le fait pour toi.

### 14.7 SQLite avec modernc.org/sqlite

SQLite, tu connais : une base complète dans un fichier, sans serveur. Le pilote historique, `mattn/go-sqlite3`, embarque le code C de SQLite et exige CGO, donc un compilateur C sur chaque machine de build et un binaire qui n'est plus tout à fait statique. `modernc.org/sqlite` est une **traduction automatique du C de SQLite en Go** : pur Go, `CGO_ENABLED=0`, compilation croisée en une commande comme tout le reste. Il est un peu plus lent que la version C (de l'ordre de 1,5× sur les écritures), ce qui n'a aucune importance pour un outil d'infra. C'est celui du labo.

DSN : un chemin (`inventaire.db`, créé s'il manque), ou `:memory:`. Les réglages passent dans l'URL : `file:inventaire.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)` pour qu'un écrivain attende 5 s au lieu d'échouer sur `database is locked`, et pour le journal WAL qui laisse les lecteurs lire pendant qu'on écrit.

**Piège :** une base `:memory:` est propre à **chaque connexion**. Or `*sql.DB` est un pool : la deuxième connexion qu'il ouvre voit une base vide, sans tes tables.

```go
db, _ := sql.Open("sqlite", ":memory:")
db.Exec(`CREATE TABLE t (n INTEGER)`)
tx, _ := db.Begin()          // occupe la connexion n°1...
defer tx.Rollback()
var n int
err := db.QueryRow(`SELECT COUNT(*) FROM t`).Scan(&n) // ... prend la n°2 : une autre base
fmt.Println("avec 2 connexions :", err)

db2, _ := sql.Open("sqlite", ":memory:")
db2.SetMaxOpenConns(1)
db2.Exec(`CREATE TABLE t (n INTEGER)`)
err = db2.QueryRow(`SELECT COUNT(*) FROM t`).Scan(&n)
fmt.Println("avec 1 connexion  :", err, n)
```

```
avec 2 connexions : SQL logic error: no such table: t (1)
avec 1 connexion  : <nil> 0
```

`db.SetMaxOpenConns(1)` règle le problème, et il est de toute façon juste pour SQLite, qui n'accepte qu'un écrivain à la fois. Corollaire : avec une seule connexion, une transaction ouverte tient *la* connexion, et un appel à `db.Query` pendant la transaction attend une connexion que la transaction ne rendra jamais. Dans une transaction, tout passe par `tx`.

### 14.8 PostgreSQL avec pgx : mode `database/sql` ou mode natif

[pgx](https://github.com/jackc/pgx) est *le* pilote PostgreSQL en Go (l'ancien, `lib/pq`, est en maintenance). Il a deux visages :

**Le mode `database/sql`** : `import _ "github.com/jackc/pgx/v5/stdlib"` et `sql.Open("pgx", dsn)`. Tout ce qui précède s'applique. C'est le bon choix quand le même code doit tourner sur SQLite et PostgreSQL (le labo), ou quand une bibliothèque attend un `*sql.DB`.

**Le mode natif** : `pgx.Connect` (une connexion) ou `pgxpool.New` (un pool, celui qu'on utilise dans un serveur). L'API ressemble à `database/sql` mais parle PostgreSQL sans traduction : les types natifs (tableaux, `jsonb`, `inet`, intervalles) arrivent directement dans des types Go, le protocole binaire est utilisé, `COPY` est disponible pour les imports massifs, et `pgx.CollectRows` supprime le `Scan` à la main. C'est ce que Deckhand utilise pour son hub (`internal/hub/db.go` : `pgxpool.New(ctx, dsn)`).

```go
pool, err := pgxpool.New(ctx, "postgres://postgres@127.0.0.1:5499/inventaire?sslmode=disable&pool_max_conns=4")
defer pool.Close()

var version string
pool.QueryRow(ctx, `SHOW server_version`).Scan(&version)
fmt.Println("serveur :", version)

rows, err := pool.Query(ctx, `SELECT nom, ip, role, dernier_vu AS derniervu FROM hotes ORDER BY nom`)
hotes, err := pgx.CollectRows(rows, pgx.RowToStructByName[Hote]) // colonnes → champs de même nom
for _, h := range hotes {
	fmt.Printf("  %-8s %-10s %-6s vu=%v\n", h.Nom, h.IP, h.Role, h.DernierVu != nil)
}

var ports []int32
pool.QueryRow(ctx, `SELECT ARRAY[22, 80, 443]`).Scan(&ports) // un tableau SQL dans un slice Go
fmt.Println("ports :", ports)

s := pool.Stat()
fmt.Printf("pool : %d/%d connexions, %d au repos\n", s.TotalConns(), s.MaxConns(), s.IdleConns())
```

```
serveur : 16.15 (Homebrew)
  bastion  10.0.0.2   admin  vu=false
  db-01    10.0.0.21  db     vu=true
  web-01   10.0.0.11  web    vu=false
  web-02   10.0.0.12  web    vu=false
ports : [22 80 443]
pool : 1/4 connexions, 1 au repos
```

Les erreurs du serveur arrivent en `*pgconn.PgError`, avec le code SQLSTATE que tu connais, à extraire avec `errors.As` (chapitre 8) :

```go
_, err := db.Exec(`INSERT INTO hotes (nom, ip) VALUES ('web-01', '10.0.0.11')`)
fmt.Println("err :", err)
var pgErr *pgconn.PgError
if errors.As(err, &pgErr) {
	fmt.Println("code :", pgErr.Code, "| contrainte :", pgErr.ConstraintName)
	fmt.Println("doublon ?", pgErr.Code == "23505")
}
```

```
err : ERREUR: la valeur d'une clé dupliquée rompt la contrainte unique « hotes_pkey » (SQLSTATE 23505)
code : 23505 | contrainte : hotes_pkey
doublon ? true
```

(Le message est en français parce que le serveur de test a été initialisé avec la locale du Mac ; ne fais jamais de `strings.Contains` sur un message d'erreur, teste le code.) Le labo évite même ce code, non portable vers SQLite, avec `INSERT ... ON CONFLICT DO NOTHING` et `RowsAffected`.

Le DSN, lui, ne va jamais dans le code ni dans git : il contient un mot de passe. Il vient d'une variable d'environnement (`os.Getenv("INVENTAIRE_DSN")`, chapitre 15 sur la configuration), et pgx comprend aussi les variables standard de libpq (`PGHOST`, `PGUSER`, `PGPASSWORD`, `PGSSLMODE`) et le fichier `~/.pgpass`, exactement comme `psql`.

### 14.9 Migrations : le schéma versionné avec le code

`CREATE TABLE IF NOT EXISTS` suffit le premier jour. Le deuxième mois, il faut ajouter une colonne à une table qui a des données, et là il faut des **migrations** : des fichiers SQL numérotés, appliqués une fois chacun, dans l'ordre, avec une table qui note où on en est. Le labo écrit ce mécanisme en soixante lignes : les fichiers `migrations/0001_hotes.sql`, `0002_index_role.sql` sont embarqués dans le binaire avec `//go:embed migrations/*.sql` (chapitre 12), et au démarrage le programme lit `MAX(version)` dans `schema_version`, applique les fichiers de numéro supérieur, chacun dans sa transaction avec l'insertion de son numéro. Le binaire porte son schéma : pas de script à copier sur le serveur, pas de « tu as pensé à jouer le SQL ? ».

Quand le projet grossit, [golang-migrate](https://github.com/golang-migrate/migrate) fait la même chose avec en plus les migrations descendantes (`0001_init.down.sql`), une CLI, et le support de vingt moteurs. Deckhand l'utilise, ses migrations sont dans `migrations/` avec un `embed.go` de quatre lignes, appliquées par le hub à son démarrage. Deux règles dans tous les cas : une migration appliquée ne se modifie plus jamais (on en ajoute une nouvelle), et une migration est petite et réversible mentalement, parce que tu la joueras en production un vendredi.

**Piège :** PostgreSQL et SQLite savent annuler un `CREATE TABLE` ou un `ALTER TABLE` dans une transaction (DDL transactionnel). MySQL non : une migration MySQL qui échoue à mi-chemin laisse le schéma à moitié modifié, et il faut réparer à la main.

### 14.10 sqlc, GORM, et l'avis d'un DBA

Écrire `Scan(&h.Nom, &h.IP, &h.Role, &vu)` pour chaque requête est répétitif, et personne ne t'en voudra de chercher mieux. Deux écoles :

**[sqlc](https://sqlc.dev)** : tu écris tes requêtes SQL dans un fichier, annotées d'un nom et d'un type de résultat (`-- name: TrouverHote :one`), et `sqlc generate` produit le code Go typé (`func (q *Queries) TrouverHote(ctx, nom string) (Hote, error)`) avec le `Scan` écrit pour toi, vérifié contre le schéma à la génération. Le SQL reste du SQL, que tu peux coller dans `psql` et expliquer avec `EXPLAIN`. Pour quelqu'un qui administre PostgreSQL, c'est le bon outil : zéro magie, et une colonne renommée casse la génération, pas la production. Il cible pgx nativement.

**[GORM](https://gorm.io)** : l'ORM de Go, à la SQLAlchemy ou Hibernate. Tu déclares des structs, il génère le schéma, les requêtes, les jointures, avec un langage de chaînage (`db.Where("role = ?", "web").Order("nom").Find(&hotes)`). Il fait gagner du temps sur un CRUD simple et en fait perdre dès que tu veux savoir *quel* SQL part vers le serveur, ou pourquoi il y a cent requêtes là où tu en attendais une. Un DBA qui l'utilise finit par lire les logs du serveur pour comprendre son propre programme. Avis personnel, partagé par une bonne partie de la communauté : `database/sql` ou pgx à la main jusqu'à une vingtaine de requêtes, sqlc au-delà, GORM seulement si l'équipe le connaît déjà et que la base n'est pas le cœur du sujet.

### 14.11 Ce qu'un DBA PostgreSQL doit savoir sur un programme Go

Ce que le programme fait à ton serveur, et où le régler :

- **Le pool est côté client, et chaque connexion est un processus côté serveur.** `SetMaxOpenConns(10)` (ou `pool_max_conns=10` dans le DSN pgx) sur vingt instances du service, ce sont deux cents `postgres` potentiels. Compte, et compare avec `max_connections`. Par défaut, `database/sql` n'a **pas** de maximum, et pgxpool en met 4 ou le nombre de cœurs. Fixe-le toujours.
- **Le contexte est ton `statement_timeout`**, mais côté client, par requête, et il annule proprement côté serveur (section 14.5). Garde quand même un `statement_timeout` serveur comme filet, pour les requêtes lancées sans contexte.
- **pgx prépare et met en cache les requêtes** par connexion (mode `QueryExecModeCacheStatement` par défaut) : la première exécution d'un SQL donné fait un `PREPARE`, les suivantes réutilisent le plan. C'est plus rapide, et c'est incompatible avec PgBouncer en mode *transaction*, qui ne garantit pas de retomber sur la même connexion. Dans ce cas : `default_query_exec_mode=simple_protocol` dans le DSN, ou PgBouncer récent avec `max_prepared_statements`.
- **`sslmode`** : `disable` en local, `require` au minimum ailleurs, `verify-full` avec le CA quand tu peux. le défaut de pgx est `prefer` : TLS si le serveur le propose, en clair sinon, et le certificat n'est vérifié qu'à partir de `verify-ca` : exactement comme `psql`.
- **`SetConnMaxLifetime`** évite qu'une connexion survive à un basculement (failover, PgBouncer redémarré, DNS changé) : 30 minutes est raisonnable.
- Une requête `Query` dont on n'a pas lu toutes les lignes ni appelé `Close` garde sa connexion **et** sa transaction implicite ouvertes : c'est le `idle in transaction` que tu vois dans `pg_stat_activity` et qui bloque un `VACUUM`. Le `defer rows.Close()` de la section 14.2 n'est pas décoratif.

### 14.12 Tester : SQLite en mémoire, PostgreSQL en option

Les tests du labo ouvrent `:memory:` : une base neuve par test, créée par les migrations, détruite à la fermeture, en quelques millisecondes et sans rien installer. C'est possible parce que le code parle `database/sql` avec un SQL commun aux deux moteurs. Le même fichier de tests tourne sur PostgreSQL en posant `INVENTAIRE_TEST_DSN=postgres://...` : la table est vidée avant chaque test. En intégration continue (chapitre 15), un conteneur `postgres:16` à côté du job fait exactement ça.

La limite est honnête : SQLite n'est pas PostgreSQL. Il n'a pas `RETURNING` sur toutes les versions, pas de tableaux, pas de `jsonb`, un typage laxiste (une chaîne s'insère dans une colonne `INTEGER` sans erreur), et il ne vérifie pas les clés étrangères sans `PRAGMA foreign_keys = ON`. Dès que ton SQL utilise une spécificité de PostgreSQL, teste sur PostgreSQL, et l'approche moderne est [testcontainers-go](https://golang.testcontainers.org) : le test lance lui-même un conteneur PostgreSQL jetable, et `go test` reste la seule commande à taper.

### 14.13 Pour le labo

Le [labo 14](../labs/14-sqlite-postgres/README.md) te fait écrire le dépôt `Inventaire` : une interface `Store`, une implémentation `database/sql` qui tourne telle quelle sur SQLite et sur PostgreSQL selon `INVENTAIRE_DSN`, des migrations embarquées avec une table `schema_version`, un import en masse dans une transaction qui annule tout au premier doublon, et des tests sur SQLite en mémoire. Une soirée, et tu auras écrit la couche d'accès aux données que tu réutiliseras dans chaque service.

### À retenir

- `database/sql` est l'API générique, un pilote tiers par moteur, importé avec `_` pour son `init()`. `modernc.org/sqlite` (pur Go) pour SQLite, `pgx/v5` pour PostgreSQL.
- `sql.Open` ne se connecte pas : `PingContext` au démarrage. `*sql.DB` est un pool partagé, un par programme, avec `SetMaxOpenConns` toujours fixé.
- `Exec` pour écrire, `QueryRow` + `Scan` pour une ligne (`sql.ErrNoRows` si absente), `Query` + `Next` + `Scan` pour N lignes, avec `defer rows.Close()` et `rows.Err()` après la boucle.
- Le NULL se lit dans un `sql.NullString` ou un pointeur, ou disparaît avec `COALESCE`.
- Transaction : `BeginTx`, `defer tx.Rollback()`, tout par `tx`, `return tx.Commit()`.
- Les versions `...Context` de tout : le délai est décidé par l'appelant et annulé jusque sur le serveur.
- Les valeurs passent par `$1`, `$2` (ou `?`), jamais par `fmt.Sprintf`. Pas d'exception.
- SQLite `:memory:` est propre à chaque connexion : `SetMaxOpenConns(1)`. Migrations embarquées avec `embed` et une table de version ; sqlc plutôt que GORM quand le `Scan` à la main devient lourd.

---

← [13. HTTP : client, serveur, API JSON](13-http.md) · [Sommaire](../README.md) · [15. Qualité, build, release, Docker, observabilité](15-qualite-build-release.md) →
