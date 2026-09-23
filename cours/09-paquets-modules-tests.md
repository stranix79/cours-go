# 9. Paquets, modules, tests

*Go de zéro à la prod : chapitre 9 sur 16.* ← [8. Erreurs pour de vrai : wrapping, Is, As, panic](08-erreurs-avancees.md) · [Sommaire](../README.md) · [10. Sous le capot : compilation, mémoire, GC, goroutines](10-sous-le-capot.md) →

Jusqu'ici, chaque exemple tenait dans un `main.go`. Un vrai outil a un point d'entrée, deux ou trois paquets internes, des tests, et un `go.mod` qui fixe ses dépendances. Ce chapitre est celui où tu apprends à **organiser** : où mettre les fichiers, ce qu'un autre paquet peut voir, comment les modules et les versions marchent, et surtout comment écrire des tests, parce qu'en Go les tests sont dans la boîte, sans rien installer, et que tout le monde les écrit de la même façon.

Fil rouge : un petit module `portscan`, avec un exécutable dans `cmd/portscan` et un paquet `internal/ports` qui transforme `"22,80,8000-8010"` en liste de ports. Tout ce qui suit a été lancé dessus.

### 9.1 Un paquet par dossier

Un **paquet** est un dossier. Tous les fichiers `.go` d'un dossier déclarent le même `package nom` et forment une seule unité de compilation : ils voient les fonctions, types et variables les uns des autres sans import. Le nom du paquet est en général celui du dossier, court, en minuscules, sans underscore (`ports`, `config`, `http`, pas `portParser` ni `port_utils`).

Voici l'arborescence de `portscan` :

```
portscan/
├── go.mod
├── cmd/
│   └── portscan/
│       └── main.go          package main : lit les arguments, appelle ports.Parse, affiche
└── internal/
    ├── ports/
    │   ├── ports.go         package ports : Parse et ses helpers
    │   └── ports_test.go    les tests de ports
    └── config/
        ├── config.go
        └── config_test.go
```

Trois conventions que tu retrouveras dans presque tous les projets Go, et qu'il vaut mieux suivre :

- **`cmd/nom/main.go`** : un dossier par exécutable. `go build ./cmd/portscan` produit le binaire `portscan`. Un module peut avoir plusieurs commandes (`cmd/serveur`, `cmd/migrate`). Le `main` y est minuscule : il lit les arguments et appelle un paquet.
- **`internal/`** : tout paquet placé sous un dossier `internal` ne peut être importé que par le code du même module. Le compilateur l'impose :

```
$ go build .
package exemple.com/autre
	main.go:6:2: use of internal package github.com/stranix79/portscan/internal/ports not allowed
```

C'est le vrai `private` de Go, à l'échelle du module. Mets sous `internal/` tout ce qui n'est pas une API que tu promets de maintenir, c'est-à-dire, au début, tout.

- **`pkg/`** : tu le verras dans Kubernetes et des projets de 2015. C'est le dossier des paquets publics réutilisables. Il est inutile : un paquet à la racine ou dans un sous-dossier ordinaire est public par défaut. N'en crée pas.

Pour un petit outil, l'arbre minimal est `main.go` à la racine plus un ou deux paquets dans `internal/`. Pour une bibliothèque, le paquet principal à la racine (`github.com/x/semver` importe comme `semver`) et les détails dans `internal/`. Pas de `src/`, pas de `lib/`, pas de dossier par « couche » : Go range par **sujet**, pas par rôle.

**Venant de Python :** un paquet Go est un dossier comme un paquet Python, mais sans `__init__.py`, et sans hiérarchie d'import implicite : `internal/ports` n'a rien à voir avec `internal/config`, chacun s'importe par son chemin complet. Et il n'y a pas de « module = fichier » : le découpage en fichiers dans un paquet est libre, purement pour la lecture.

**Venant du C :** un paquet est l'unité qui remplace le couple `.h`/`.c`. Ce qui est exporté est le `.h`, le reste est le `.c`, et le compilateur gère la dépendance sans que tu écrives un en-tête.

### 9.2 Exporté, non exporté

La règle du chapitre 1, sans exception : **majuscule = exporté, minuscule = privé au paquet**. Ça vaut pour les fonctions, les types, les champs de struct, les méthodes, les variables et constantes de paquet. Dans `ports.go` :

```go
// Package ports analyse des listes de ports au format "22,80,8000-8010".
package ports

// Max est le plus grand numéro de port TCP.
const Max = 65535

// Parse transforme "22,80,8000-8002" en [22 80 8000 8001 8002].
// Les espaces autour des éléments sont ignorés. Un port hors de 1..Max,
// une plage inversée ou un élément vide renvoient une erreur.
func Parse(spec string) ([]int, error) { ... }

func bornes(elem string) (int, int, error) { ... }

func unPort(s string) (int, error) { ... }
```

`Parse` et `Max` sont l'API du paquet, `bornes` et `unPort` sont des détails. Depuis `cmd/portscan/main.go`, on écrit `ports.Parse(...)` et `ports.Max` ; `ports.bornes` ne compile pas. Une bonne API de paquet exporte peu : deux ou trois fonctions, un ou deux types. Si tu hésites, laisse en minuscule ; passer en majuscule plus tard ne casse personne, l'inverse si.

Les commentaires au-dessus de chaque symbole exporté ne sont pas décoratifs : ce sont les **commentaires de doc**, lus par `go doc` et pkg.go.dev. La convention est stricte : le commentaire commence par le nom du symbole (`// Parse transforme...`), fait une phrase complète, et celui du paquet commence par `// Package ports ...`. `golangci-lint` te le rappelle si tu l'oublies.

```
$ go doc ./internal/ports
package ports // import "github.com/stranix79/portscan/internal/ports"

Package ports analyse des listes de ports au format "22,80,8000-8010".

const Max = 65535
func Parse(spec string) ([]int, error)
$ go doc ./internal/ports Parse
package ports // import "github.com/stranix79/portscan/internal/ports"

func Parse(spec string) ([]int, error)
    Parse transforme "22,80,8000-8002" en [22 80 8000 8001 8002]. Les espaces
    autour des éléments sont ignorés. Un port hors de 1..Max, une plage inversée
    ou un élément vide renvoient une erreur.
```

C'est ta doc, générée depuis ton code, dans ton terminal, sans Sphinx ni Doxygen. Elle sera fausse le jour où tu changeras `Parse` sans toucher au commentaire ; c'est le seul risque, et la relecture de code le couvre.

### 9.3 `init()`, et pourquoi l'éviter

Un paquet peut déclarer une fonction `init()` (sans argument, sans retour, et il peut y en avoir plusieurs). Elle est appelée automatiquement quand le paquet est chargé, après l'initialisation de ses variables de paquet et avant `main`.

```go
var seuil = calculerSeuil()

func calculerSeuil() int {
	fmt.Println("1. initialisation de la variable seuil")
	return 80
}

func init() {
	fmt.Println("2. init() du paquet main, seuil =", seuil)
}

func main() {
	fmt.Println("3. main()")
}
```

```
$ go run .
1. initialisation de la variable seuil
2. init() du paquet main, seuil = 80
3. main()
```

L'ordre : les variables de paquet dans l'ordre de leurs dépendances, puis les `init()`, paquet par paquet en suivant les imports, puis `main`. Tu verras `init()` dans deux cas légitimes : enregistrer un pilote (`database/sql` fonctionne ainsi, `import _ "github.com/lib/pq"` déclenche l'`init` du pilote, chapitre 14), et des tables calculées une fois.

**Piège :** pour tout le reste, `init()` est un piège. Il s'exécute sans être appelé, donc on ne le voit pas dans le flux du programme ; il ne peut pas renvoyer d'erreur (il `panic` ou `log.Fatal`, section 8.7) ; il rend le paquet impossible à tester avec une autre configuration ; et il s'exécute dès qu'un test importe le paquet, même pour tester autre chose. Si tu as besoin d'initialiser quelque chose, écris une fonction `New...` ou `Ouvrir...` que `main` appelle explicitement. Une variable de paquet initialisée par une expression simple (`var ErrX = errors.New(...)`, une map de constantes) est très bien ; une variable de paquet qui ouvre un fichier ou lit l'environnement ne l'est pas.

### 9.4 Modules, `go.mod`, `go.sum`, versions

Un **module** est l'unité de versionnement et de distribution : un arbre de paquets avec un `go.mod` à sa racine. Tu l'as créé au chapitre 1 avec `go mod init`. Les trois lignes du fichier :

```
module github.com/stranix79/portscan

go 1.25
```

- `module chemin` : le préfixe d'import de tous les paquets du module. `internal/ports` s'importe donc `github.com/stranix79/portscan/internal/ports`. Si le module est publié, c'est aussi l'URL où `go get` ira le chercher ; pour un projet privé, n'importe quel chemin marche (`cours-go/labs/09-mon-module` dans ce cours).
- `go 1.25` : la version minimale du langage que le module exige. Elle change ce que le compilateur accepte (la sémantique de `for` par exemple a changé en 1.22). Mets la version que tu utilises vraiment.

Ajoute une dépendance et regarde ce qui se passe :

```
$ go get github.com/google/go-cmp@latest
go: added github.com/google/go-cmp v0.7.0
$ cat go.mod
module github.com/stranix79/portscan

go 1.25

require github.com/google/go-cmp v0.7.0 // indirect
$ cat go.sum
github.com/google/go-cmp v0.7.0 h1:wk8382ETsv4JYUZwIsn6YpYiWiBsYLSJiTsyBybVuN8=
github.com/google/go-cmp v0.7.0/go.mod h1:pXiqmnSA92OHEEa9HXL2W4E7lf9JzCmGVUdgjX3N/iU=
```

- `require module version` : la dépendance et sa version exacte. `// indirect` parce qu'aucun fichier ne l'importe encore ; dès qu'un `import` l'utilisera, `go mod tidy` retirera le commentaire.
- `go.sum` : l'empreinte cryptographique de chaque version téléchargée. Au prochain `go build`, sur ta machine ou en CI, Go vérifie que ce qu'il télécharge correspond. Si quelqu'un remplace le contenu de `v0.7.0` sur GitHub, la compilation échoue. **Commite les deux fichiers.** Le `go.sum` n'est pas un fichier de verrou à ignorer, c'est ta chaîne de confiance.

Les versions suivent le **versionnement sémantique** : `vMAJEUR.MINEUR.CORRECTIF`. Correctif = bug corrigé, mineur = fonctionnalité ajoutée sans rien casser, majeur = API incompatible. Go va plus loin que les autres écosystèmes : à partir de `v2`, le majeur fait partie du chemin d'import (`github.com/x/lib/v2`), donc deux majeurs incompatibles peuvent coexister dans un même programme. Et l'algorithme de choix est le *minimal version selection* : Go prend la **plus petite** version qui satisfait tous les `require`, pas la plus récente. Un build est reproductible sans fichier de verrou parce que `go.mod` est déjà exact.

Deux commandes à connaître :

- `go mod tidy` : ajoute ce qui est importé mais absent, retire ce qui est présent mais plus importé, met `go.sum` à jour. Après l'avoir lancé ici, la ligne `require` a disparu (rien ne l'importe). Lance-le avant chaque commit.
- `go get module@version` pour changer de version (`@latest`, `@v1.4.0`, `@master`) ; `go list -m -u all` pour voir ce qui a une mise à jour.

Et la directive `replace`, pour travailler sur une dépendance en local :

```
replace github.com/stranix79/portscan => ../portscan
```

C'est ce que l'exemple d'`internal/` utilisait pour importer `portscan` depuis un module voisin sans le publier. Utile pour tester une correction dans une bibliothèque avant de la pousser ; à retirer avant de commiter, ou à réserver au développement.

**Venant de Python :** `go.mod` est `pyproject.toml` plus le fichier de verrou, `go.sum` la partie hash, et il n'y a pas de `venv` parce que le cache `~/go/pkg/mod` est partagé et versionné. Deux projets qui veulent deux versions d'une bibliothèque ne se gênent pas. Et il n'y a pas d'index central comme PyPI : un module se télécharge depuis son dépôt git (via un proxy avec cache, `proxy.golang.org`, et une base d'empreintes publique, `sum.golang.org`, qui vérifie que tout le monde reçoit la même chose).

### 9.5 `go test` : fichiers `_test.go` et `testing.T`

Un test Go est une fonction `TestXxx(t *testing.T)` dans un fichier `xxx_test.go`, dans le même dossier que le code. `go test` compile le paquet avec ses fichiers de test, lance chaque `TestXxx`, et affiche `ok` ou `FAIL`. Pas de framework à installer, pas de découverte à configurer : le nom du fichier et le nom de la fonction suffisent.

Le test de `config.Lire` (le paquet lit un fichier `clé=valeur`) montre les quatre méthodes de `t` que tu utiliseras :

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func ecrireFichier(t *testing.T, contenu string) string {
	t.Helper()
	chemin := filepath.Join(t.TempDir(), "app.conf")
	if err := os.WriteFile(chemin, []byte(contenu), 0o644); err != nil {
		t.Fatalf("écrire le fichier de test : %v", err)
	}
	return chemin
}

func TestLire(t *testing.T) {
	chemin := ecrireFichier(t, "host = db-01\n# commentaire\n\nport=5432\n")
	got, err := Lire(chemin)
	if err != nil {
		t.Fatalf("Lire : %v", err)
	}
	if got["host"] != "db-01" || got["port"] != "5432" {
		t.Errorf("Lire = %v, attendu host=db-01 port=5432", got)
	}
	if len(got) != 2 {
		t.Errorf("Lire : %d clés, attendu 2", len(got))
	}
}

func TestLireIntrouvable(t *testing.T) {
	_, err := Lire("/nulle/part/app.conf")
	if err == nil {
		t.Fatal("Lire : erreur attendue sur un fichier absent")
	}
}
```

```
$ go test -v ./internal/config
=== RUN   TestLire
--- PASS: TestLire (0.00s)
=== RUN   TestLireIntrouvable
--- PASS: TestLireIntrouvable (0.00s)
PASS
ok  	github.com/stranix79/portscan/internal/config	0.414s
```

- `t.Errorf` signale un échec **et continue** : le test rapportera toutes les assertions fausses. `t.Fatalf` signale et **arrête** le test : à utiliser quand la suite n'a pas de sens (l'erreur d'ouverture, le pointeur nil). Pas d'`assert` : on écrit un `if` et on formate le message soi-même, avec « obtenu, attendu ». C'est plus long qu'un `assert a == b`, mais le message dit exactement ce qu'on veut lire.
- `t.TempDir()` crée un dossier temporaire vide, unique, supprimé à la fin du test. C'est le `tmp_path` de pytest. Jamais de fichier de test à commiter ni à nettoyer.
- `t.Helper()` dans une fonction utilitaire : quand `ecrireFichier` échoue, la ligne rapportée est celle du **test** qui l'a appelée, pas celle de l'utilitaire. Sans `Helper`, on passerait son temps à chercher qui a appelé.
- Le fichier de test est dans le paquet `config` lui-même : il a accès aux fonctions privées. (Il existe aussi la forme `package config_test`, qui teste le paquet de l'extérieur, comme un utilisateur ; on la réserve aux exemples de la section 9.7.)

Voici ce que ça donne quand un test échoue. J'ai ajouté un test volontairement faux :

```
$ go test ./internal/config
--- FAIL: TestVolontairementFaux (0.00s)
    echec_test.go:9: port = "5432", attendu "5433"
FAIL
FAIL	github.com/stranix79/portscan/internal/config	0.375s
FAIL
```

Fichier, ligne, message : le mien. `go test` ne dit rien d'autre, et n'a rien besoin de deviner.

**Venant de Python :** c'est pytest sans la magie. Pas de fixture par injection, pas d'introspection de l'`assert`, pas de paramétrage par décorateur : des fonctions, des `if`, des messages. Ce qui manque en confort est gagné en lisibilité : un test Go se lit sans connaître l'outil. Et `go test` est plus rapide que pytest ne le sera jamais, parce que le paquet est compilé une fois et mis en cache : relance sans rien changer, et tu verras `(cached)`.

Les options utiles : `-v` pour voir chaque test, `-run 'Nom'` pour n'en lancer que certains (une expression régulière sur le nom, `-run 'TestParse/plage'` pour un sous-test), `-count=1` pour ignorer le cache, `./...` pour tout le module, `-failfast` pour s'arrêter au premier échec.

### 9.6 Tests table-driven avec `t.Run`

Le motif Go pour « un test, dix cas » : une slice de structs anonymes, une boucle, et `t.Run` pour donner un nom à chaque cas. C'est le `parametrize` de pytest, écrit à la main.

```go
func TestParse(t *testing.T) {
	cas := []struct {
		nom    string
		spec   string
		veut   []int
		erreur bool
	}{
		{"un port", "22", []int{22}, false},
		{"liste", "22, 80,443", []int{22, 80, 443}, false},
		{"plage", "8000-8002", []int{8000, 8001, 8002}, false},
		{"mélange", "22,8000-8001", []int{22, 8000, 8001}, false},
		{"vide", "", nil, true},
		{"pas un entier", "ssh", nil, true},
		{"hors bornes", "70000", nil, true},
		{"plage inversée", "90-80", nil, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got, err := Parse(c.spec)
			if c.erreur {
				if err == nil {
					t.Fatalf("Parse(%q) : erreur attendue, obtenu %v", c.spec, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) : erreur inattendue : %v", c.spec, err)
			}
			if !slices.Equal(got, c.veut) {
				t.Errorf("Parse(%q) = %v, attendu %v", c.spec, got, c.veut)
			}
		})
	}
}
```

```
$ go test -v -run TestParse ./internal/ports
=== RUN   TestParse
=== RUN   TestParse/un_port
=== RUN   TestParse/liste
=== RUN   TestParse/plage
=== RUN   TestParse/mélange
=== RUN   TestParse/vide
=== RUN   TestParse/pas_un_entier
=== RUN   TestParse/hors_bornes
=== RUN   TestParse/plage_inversée
--- PASS: TestParse (0.00s)
    --- PASS: TestParse/un_port (0.00s)
    --- PASS: TestParse/liste (0.00s)
    ...
PASS
ok  	github.com/stranix79/portscan/internal/ports	0.203s
```

Chaque cas a un nom (les espaces deviennent des `_`), échoue indépendamment, et se relance seul avec `-run 'TestParse/plage'`. Ajouter un cas coûte une ligne dans la table. Un cas d'erreur se décrit par un booléen `erreur` (ou par une sentinelle attendue, `veutErr error`, testée avec `errors.Is`, comme au labo 08). `slices.Equal` compare deux slices élément par élément ; pour des structs imbriquées, `reflect.DeepEqual` marche, et la bibliothèque `go-cmp` de Google donne un diff lisible, c'est la seule dépendance de test que beaucoup de projets s'autorisent.

**Piège :** avant Go 1.22, la variable `c` de la boucle était partagée par toutes les itérations, et un `t.Run` avec `t.Parallel()` voyait toujours le dernier cas. Le `for` a été corrigé en 1.22 (chaque itération a sa variable), et `go 1.25` dans le `go.mod` t'y donne droit. Si tu lis un vieux `c := c` en tête de boucle dans du code, c'est la parade de l'époque, inutile aujourd'hui.

### 9.7 Exemples, benchmarks, couverture, `-race`

**Un exemple est un test qui sert de doc.** Une fonction `ExampleXxx` dans un fichier de test, avec un commentaire `// Output:` en dernière ligne : `go test` l'exécute et compare la sortie standard au commentaire. Et `go doc` et pkg.go.dev l'affichent sous la fonction `Xxx`. Une doc qui ne peut pas être fausse.

```go
func ExampleParse() {
	p, _ := Parse("22,8000-8002")
	fmt.Println(p)
	// Output: [22 8000 8001 8002]
}
```

```
$ go test -v -run Example ./internal/ports
=== RUN   ExampleParse
--- PASS: ExampleParse (0.00s)
PASS
```

**Un benchmark** est une fonction `BenchmarkXxx(b *testing.B)` avec une boucle `for b.Loop()` (Go 1.24 ; avant, `for i := 0; i < b.N; i++`). L'outil ajuste le nombre d'itérations pour mesurer pendant environ une seconde :

```go
func BenchmarkParse(b *testing.B) {
	for b.Loop() {
		Parse("22,80,443,8000-8100")
	}
}
```

```
$ go test -bench . -benchmem -run '^$' ./internal/ports
goos: darwin
goarch: arm64
pkg: github.com/stranix79/portscan/internal/ports
cpu: Apple M1 Pro
BenchmarkParse-10    	 2878003	       407.9 ns/op	    2048 B/op	       6 allocs/op
PASS
```

408 nanosecondes et 6 allocations par appel. Le `-run '^$'` empêche de relancer les tests en même temps, `-benchmem` ajoute les colonnes mémoire, et `-10` est le nombre de cœurs utilisés. Deux chiffres à comparer avant et après une optimisation, pas des chiffres absolus.

**La couverture** dit quelles lignes les tests ont exécutées :

```
$ go test -cover ./...
	github.com/stranix79/portscan/cmd/portscan		coverage: 0.0% of statements
ok  	github.com/stranix79/portscan/internal/ports	0.320s	coverage: 96.6% of statements
```

`go test -coverprofile=c.out ./... && go tool cover -html=c.out` ouvre un rapport où les lignes jamais exécutées sont en rouge. Le pourcentage ne prouve rien (`cmd/` à 0 % est normal, c'est un `main`) ; les lignes rouges disent où regarder.

**`-race`** compile le paquet avec le détecteur de concurrence et signale tout accès concurrent non protégé à une variable. C'est un avant-goût du chapitre 11, mais il faut savoir qu'il existe dès maintenant : lance `go test -race ./...` en CI, toujours. Sur une fonction qui incrémente un compteur depuis mille goroutines sans verrou :

```
$ go test -race .
==================
WARNING: DATA RACE
Read at 0x00c00011c178 by goroutine 9:
  racedemo.Compter.func1()
      /Users/stranix/racedemo/compteur.go:12 +0x68

Previous write at 0x00c00011c178 by goroutine 10:
  racedemo.Compter.func1()
      /Users/stranix/racedemo/compteur.go:12 +0x78
...
```

Fichier, ligne, et les deux goroutines en conflit. Sans `-race`, ce test aurait pu passer neuf fois sur dix.

### 9.8 Et testify ?

Tu croiseras `github.com/stretchr/testify` dans beaucoup de projets : `assert.Equal(t, attendu, obtenu)`, `require.NoError(t, err)`, des mocks. C'est bien fait, et si l'équipe l'utilise, utilise-le. Mais ce cours reste sur la bibliothèque standard, pour deux raisons : un `if got != want { t.Errorf(...) }` se lit sans doc, et une dépendance de moins est une dépendance de moins. Le jour où tu veux des diffs lisibles sur des structs, ajoute `go-cmp` ; le jour où tu veux des mocks, demande-toi d'abord si une petite interface (chapitre 7) et un faux écrit à la main en dix lignes ne feraient pas mieux. En Go, la réponse est presque toujours oui.

### 9.9 Pour le labo

Le [labo 09](../labs/09-mon-module/README.md) te fait construire ton premier vrai module : `cmd/semver/main.go` et `internal/semver/` avec `Parse`, `Compare`, `Bump` et `Compatible` (contraintes `^` et `~`, comme npm et cargo), des tests table-driven, un `Example` qui sert de doc, un benchmark, et un README qui montre `go test ./... -cover` et `go doc`. C'est la version Go du labo 12 du cours Python, et tu verras que le squelette du test est le même.

### À retenir

- Un paquet = un dossier, un nom court en minuscules ; `cmd/nom/` pour chaque exécutable, `internal/` pour tout ce qui n'est pas une API promise, pas de `pkg/`.
- Majuscule = exporté ; commente chaque symbole exporté en commençant par son nom, `go doc ./paquet` te le rend.
- `init()` s'exécute sans être appelé et ne peut pas échouer proprement : réserve-le aux pilotes, écris des `New...` pour le reste.
- `go.mod` fixe le chemin du module, la version de Go et les dépendances exactes ; `go.sum` les empreintes ; commite les deux ; `go mod tidy` avant chaque commit ; `replace` pour une dépendance locale.
- Un test = `func TestXxx(t *testing.T)` dans `xxx_test.go` ; `t.Errorf` continue, `t.Fatalf` arrête ; `t.TempDir()` pour les fichiers ; `t.Helper()` dans les utilitaires.
- Table-driven : une slice de cas, `t.Run(nom, ...)`, `-run 'Test/cas'` pour en relancer un.
- `ExampleXxx` avec `// Output:` est une doc vérifiée ; `BenchmarkXxx` avec `b.Loop()` mesure ; `-cover` montre les lignes mortes ; `-race` attrape les accès concurrents.
- La bibliothèque standard suffit pour tester ; testify existe, `go-cmp` est la seule dépendance de test vraiment utile.

---

← [8. Erreurs pour de vrai : wrapping, Is, As, panic](08-erreurs-avancees.md) · [Sommaire](../README.md) · [10. Sous le capot : compilation, mémoire, GC, goroutines](10-sous-le-capot.md) →
