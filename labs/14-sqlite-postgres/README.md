# Labo 14 : Inventaire, sur SQLite et PostgreSQL

**Chapitre** : [14. Bases de données : database/sql, SQLite, PostgreSQL](../../cours/14-bases-de-donnees.md)

## Objectif
Écrire un dépôt (*repository*) d'hôtes derrière une interface `Store`, avec une seule implémentation `database/sql` qui tourne sans modification sur SQLite (pilote pur Go `modernc.org/sqlite`, pas de CGO) et sur PostgreSQL (pilote `pgx` en mode `database/sql`). Les migrations SQL sont embarquées dans le binaire et appliquées au démarrage, un import en masse se fait dans une transaction, et les tests tournent sur SQLite en mémoire en moins d'une seconde.

Le programme est un outil en ligne de commande, `inventaire`, dont la base est choisie par la variable d'environnement `INVENTAIRE_DSN` : un chemin de fichier SQLite (défaut `inventaire.db`) ou une URL `postgres://...`.

## Consignes
1. Lis les fichiers fournis : `main.go` (complet : les commandes `ajouter`, `lister`, `vu`, `supprimer`, `importer`, `version`, toute la logique dans `executer`), `migrations/0001_hotes.sql` et `0002_index_role.sql` (le schéma, en SQL commun aux deux moteurs : `TEXT`, `TIMESTAMP`, pas de `SERIAL` ni d'`AUTOINCREMENT`, le nom d'hôte est la clé primaire), et dans `inventaire.go` le type `Hote`, l'interface `Store`, les sentinelles `ErrExiste` et `ErrIntrouvable`, et la fonction `Ouvrir` (fournie). Note ses deux réglages : `SetMaxOpenConns(1)` pour SQLite (une base `:memory:` est propre à chaque connexion du pool) et `PingContext` (parce que `sql.Open` ne se connecte pas).
2. Lance `go test .` : tout échoue sur `migrer : TODO migrer`. C'est ton point de départ.
3. `migrations.go`, TODO 7 et 8 : `migrer` crée la table `schema_version` (`sqlSchemaVersion`), lit la version courante avec `Version`, parcourt `fs.ReadDir(migrationsFS, "migrations")` (déjà trié par nom), saute les fichiers dont `numeroDe` est inférieur ou égal à la version courante, et appelle `appliquer` pour les autres. `appliquer` exécute le contenu du fichier puis `INSERT INTO schema_version (version, nom) VALUES ($1, $2)`, le tout dans une transaction (`BeginTx`, `defer tx.Rollback()`, `Commit`). Quand `TestMigrationsAppliquees` et `TestMigrationsIdempotentes` passent (version 2 après une, puis deux ouvertures du même fichier), continue.
4. `inventaire.go`, TODO 1 à 5, les cinq méthodes simples. Écris chaque requête une fois, en constante, avec des paramètres `$1`, `$2`... (PostgreSQL les impose, le pilote SQLite les accepte). Cas limites imposés par les tests :
   - `Ajouter` sur un nom existant renvoie une erreur qui enveloppe `ErrExiste` et ne modifie pas la ligne existante. La requête `sqlInsert` fournie fait `ON CONFLICT (nom) DO NOTHING` : regarde `RowsAffected()`, 0 ligne veut dire « existait déjà ». Cette détection est portable, le code d'erreur du moteur ne l'est pas.
   - `Trouver` transforme `sql.ErrNoRows` en `ErrIntrouvable` (avec `errors.Is`). La colonne `dernier_vu` peut être NULL : scanne-la dans un `sql.NullTime` et ne remplis `DernierVu` que si `Valid` est vrai. Passe la date relue par `normaliser`.
   - `Lister("")` renvoie tout, `Lister("web")` filtre, toujours trié par nom, en une seule requête avec un seul paramètre (`WHERE $1 = '' OR role = $1`). N'oublie ni `defer rows.Close()` ni `rows.Err()` après la boucle. Un rôle inconnu renvoie une liste vide, pas une erreur.
   - `MarquerVu` et `Supprimer` : `UPDATE` / `DELETE`, puis `RowsAffected() == 0` devient `ErrIntrouvable`. `MarquerVu` reçoit une heure quelconque (fuseau local, nanosecondes) et doit la stocker en UTC à la microseconde (`normaliser`), pour qu'elle se relise identique sur les deux moteurs.
5. TODO 6, `Importer` : une transaction, une requête préparée (`tx.PrepareContext(ctx, sqlInsert)`), une boucle d'`ExecContext`. Au premier doublon, renvoie `0` et une erreur qui enveloppe `ErrExiste` : le `defer tx.Rollback()` annule ce qui précède, la table doit être exactement comme avant (`TestImporter/tout_ou_rien`). `Importer(nil)` renvoie `0, nil`. **Piège** : avec SQLite, le pool n'a qu'une connexion et la transaction la tient ; un appel à `s.db.QueryRow` pendant la transaction attend une connexion qui ne viendra jamais. Tout passe par `tx`.
6. `gofmt -l .` vide, `go vet ./...` muet, `go test .` vert. Puis essaie l'outil sur un fichier (bloc suivant) et, si tu as un PostgreSQL sous la main, sur PostgreSQL : même binaire, même SQL, seul le DSN change.

## Comment lancer
```bash
go test .                                # tes tests, sur SQLite en mémoire
go test ./solution/                      # les mêmes sur la solution
go test -race ./solution/                # avec le détecteur de course

# L'outil sur un fichier SQLite (créé au premier lancement, dans le .gitignore)
export INVENTAIRE_DSN=inventaire.db
go run . version
go run . importer exemples/hotes.json
go run . lister
go run . lister web
go run . vu db-01
sqlite3 inventaire.db 'SELECT * FROM schema_version'   # ce que le programme a écrit

# Le même outil sur PostgreSQL (aucune vérification du cours n'en dépend)
export INVENTAIRE_DSN='postgres://postgres:secret@localhost:5432/inventaire?sslmode=disable'
go run . version                         # crée les tables au premier lancement
go run . importer exemples/hotes.json
psql "$INVENTAIRE_DSN" -c '\d hotes'

# Les tests sur PostgreSQL : la table hotes est vidée avant chaque test
INVENTAIRE_TEST_DSN="$INVENTAIRE_DSN" go test -v ./solution/
```

Pour un PostgreSQL jetable : `docker run --rm -e POSTGRES_PASSWORD=secret -e POSTGRES_DB=inventaire -p 5432:5432 postgres:16`, ou avec Homebrew `initdb -D /tmp/pg -U postgres && pg_ctl -D /tmp/pg -o "-p 5432" start` puis `createdb -U postgres inventaire`.

## Sortie attendue
```
$ export INVENTAIRE_DSN=inventaire.db
$ go run . version
moteur sqlite, schéma version 2
$ go run . importer exemples/hotes.json
4 hôte(s) importé(s)
$ go run . ajouter web-01 10.9.9.9 web
erreur : ajouter web-01 : hôte déjà présent
exit status 1
$ go run . lister
NOM        IP              ROLE     DERNIER VU
bastion    10.0.0.2        admin    jamais
db-01      10.0.0.21       db       jamais
web-01     10.0.0.11       web      jamais
web-02     10.0.0.12       web      jamais
$ go run . lister web
NOM        IP              ROLE     DERNIER VU
web-01     10.0.0.11       web      jamais
web-02     10.0.0.12       web      jamais
$ go run . supprimer bastion
supprimé bastion
$ go run . supprimer fantome
erreur : supprimer fantome : hôte introuvable
exit status 1
```
`go run . vu db-01` affiche l'heure courante (en UTC), puis `lister` la montre dans la colonne DERNIER VU. Avec `INVENTAIRE_DSN=postgres://...`, `version` répond `moteur pgx, schéma version 2` et tout le reste est identique.

## Pour aller plus loin
- Ajoute une migration `0003_hotes_os.sql` (`ALTER TABLE hotes ADD COLUMN os TEXT NOT NULL DEFAULT ''`), relance : `version` passe à 3 sans toucher aux données. Puis ajoute le champ dans `Hote`, les requêtes et l'affichage. C'est le cycle de vie normal d'un schéma.
- Remplace le `sql.NullTime` de `Trouver` par un scan direct dans `&h.DernierVu` (un `**time.Time`) : `database/sql` met `nil` pour NULL et alloue sinon. Moins de lignes, mais le NULL devient invisible ; décide ce que tu préfères.
- Écris une seconde implémentation `MemStore` (une `map[string]Hote` derrière un `sync.Mutex`) qui satisfait `Store`, et fais tourner les mêmes tests dessus avec une table `[]struct{nom string; ouvrir func(t *testing.T) Store}`. Si ça passe, ton interface est bien dessinée.
