# Labo 09 : Mon module

**Chapitre** : [9. Paquets, modules, tests](../../cours/09-paquets-modules-tests.md)

## Objectif
Construire ton premier vrai module Go, avec l'arborescence standard : `cmd/semver/` pour l'exécutable, `internal/semver/` pour la logique, des tests table-driven, un `Example` qui sert de doc, un benchmark, la couverture. Le sujet est le même que le labo 12 du cours Python (versions sémantiques et contraintes `^` / `~`), pour que tu compares les deux façons de tester.

## Consignes
1. Regarde l'arborescence : `go.mod` à la racine, `cmd/semver/main.go` (complet, ne pas toucher), `internal/semver/semver.go` (à compléter), `internal/semver/semver_test.go` (les tests, fournis). La solution est dans `solution/` avec la même arborescence. `go test ./internal/...` est rouge, `go test ./solution/...` est vert.
2. Lis `semver_test.go` en entier avant d'écrire une ligne. Repère la table de `TestParse`, le helper `doitParser` avec `t.Helper()`, les trois `Example` et les deux `Benchmark`.
3. `Parse(s)` (TODO 1) : `strings.TrimSpace`, `strings.TrimPrefix(texte, "v")`, `strings.Split` sur `.`, `strconv.Atoi` sur chaque partie. Toute faute renvoie une erreur emballant `ErrInvalide`. `go test -run TestParse ./internal/...`.
4. `Compare(a, b)` (TODO 2) : -1, 0 ou 1, champ par champ. `1.2.0 < 1.10.0` : on compare des entiers, pas des chaînes.
5. `Bump(v, partie)` (TODO 3) : un `switch` sur `"major"`, `"minor"`, `"patch"`, remise à zéro des parties de droite. `Version` est une valeur : tu renvoies une nouvelle `Version`, l'original ne change pas (le test le vérifie).
6. `Compatible(v, contrainte)` (TODO 4) : d'abord le cas `,` (toutes les contraintes doivent être vraies, et toutes doivent être valides), puis la détection de l'opérateur (`>=` et `<=` avant `>` et `<`), `Parse` de la borne, `Compare`, et conclusion. Quinze cas dans `TestCompatible`.
7. Quand `go test ./...` est vert : `go test ./... -cover` (vise 100 % sur `internal/semver`), `go test -bench . ./internal/semver`, `go doc ./internal/semver` et `go doc ./internal/semver Compatible`. Ta doc, c'est tes commentaires.
8. Lance ta CLI : `go run ./cmd/semver compare 1.2.3 1.10.0`, `go run ./cmd/semver bump minor 1.2.3`, `go run ./cmd/semver check 1.5.0 '^1.2.0'`. Puis `go build -o semver ./cmd/semver` : tu as un binaire.

## Comment lancer
```bash
go test ./internal/...              # tes tests (rouge au départ)
go test -v -run TestParse ./internal/semver
go test ./solution/...              # la solution (vert)
go test ./... -cover                # tout le module, avec la couverture
go test -bench . -run '^$' ./internal/semver
go doc ./internal/semver            # la doc du paquet, depuis tes commentaires
go doc ./internal/semver Compatible
go run ./cmd/semver compare 1.2.3 1.10.0
go run ./solution/cmd/semver check 1.5.0 '^1.2.0'
go vet ./... && gofmt -l .
```

## Sortie attendue
```
$ go test ./solution/... -cover
ok  	cours-go/labs/09-mon-module/solution/cmd/semver	0.581s	coverage: 78.1% of statements
ok  	cours-go/labs/09-mon-module/solution/internal/semver	0.312s	coverage: 100.0% of statements

$ go test -bench . -run '^$' ./solution/internal/semver
goos: darwin
goarch: arm64
pkg: cours-go/labs/09-mon-module/solution/internal/semver
cpu: Apple M1 Pro
BenchmarkParse-10         	22398768	        54.41 ns/op
BenchmarkCompatible-10    	 6214410	       183.6 ns/op
PASS

$ go doc ./solution/internal/semver
package semver // import "cours-go/labs/09-mon-module/solution/internal/semver"

Package semver analyse, compare et fait évoluer des versions sémantiques de la
forme MAJEUR.MINEUR.CORRECTIF (https://semver.org), et vérifie des contraintes à
la npm / cargo : « ^1.2.0 », « ~1.2.0 », « >=1.2.0,<2.0.0 ».

var ErrInvalide = errors.New("version invalide")
func Compare(a, b Version) int
func Compatible(v Version, contrainte string) (bool, error)
type Version struct{ ... }
    func Bump(v Version, partie string) (Version, error)
    func Parse(s string) (Version, error)

$ go run ./solution/cmd/semver compare 1.2.3 1.10.0
<
$ go run ./solution/cmd/semver bump minor 1.2.3
1.3.0
$ go run ./solution/cmd/semver check 1.5.0 '^1.2.0'
oui
$ go run ./solution/cmd/semver check 2.0.0 '^1.2.0'
non
$ go run ./solution/cmd/semver compare 1.2 1.0.0
erreur : parse "1.2": attendu MAJEUR.MINEUR.CORRECTIF: version invalide
exit status 1
```
Les temps et les nanosecondes dépendent de ta machine ; les pourcentages et le reste non.

## Pour aller plus loin
- `go test -coverprofile=c.out ./internal/... && go tool cover -html=c.out` : les lignes rouges sont celles qu'aucun test n'exécute. Écris le cas qui manque.
- Ajoute les préversions (`1.2.3-beta.1 < 1.2.3`), en commençant par les cas dans la table de `TestCompare`. C'est la partie de semver qui fait pleurer tout le monde ; les tests d'abord.
- Crée un dossier voisin avec son propre `go.mod` et une directive `replace cours-go/labs/09-mon-module => ../09-mon-module`, puis essaie d'importer `internal/semver` depuis là. Lis le message du compilateur : c'est `internal/` qui fait son travail.
