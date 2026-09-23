# 1. Installer, outiller, lancer

*Go de zéro à la prod : chapitre 1 sur 16.* ← [0. Pourquoi Go, et quand l'utiliser (ou pas)](00-pourquoi-go.md) · [Sommaire](../README.md) · [2. Variables, types, constantes, chaînes](02-variables-types-chaines.md) →

Bonne nouvelle : l'outillage Go tient dans une seule commande, `go`, livrée avec le compilateur. Pas de gestionnaire de paquets à part, pas d'environnement virtuel, pas de système de build à choisir. `go build`, `go run`, `go test`, `go fmt`, `go vet`, `go mod` : tout est là, et tout le monde utilise les mêmes. Ce chapitre installe Go, écrit un premier programme, et explique ce que chaque commande fait vraiment, parce que tu vas les taper cent fois par jour.

### 1.1 Installer

**macOS** : `brew install go`. **Linux** : télécharge l'archive sur [go.dev/dl](https://go.dev/dl/), décompresse-la dans `/usr/local`, ajoute `/usr/local/go/bin` au `PATH` (les paquets des distributions sont souvent en retard de plusieurs versions ; évite-les). **Windows** : l'installeur `.msi` sur la même page.

```
$ go version
go version go1.27.1 darwin/arm64
```

Une version par an et demi, rétro-compatible : un programme écrit pour Go 1.0 en 2012 compile encore avec Go 1.27. C'est une promesse officielle, et elle est tenue. Tu ne vivras pas la migration Python 2 vers 3.

Deux répertoires à connaître, que `go env` t'affiche :

- `GOPATH` (`~/go` par défaut) : là où `go install` dépose les binaires des outils (`~/go/bin`, à ajouter au `PATH`) et où le cache des modules téléchargés vit (`~/go/pkg/mod`).
- `GOROOT` : l'installation de Go elle-même, avec le code source de la bibliothèque standard. Tu iras le lire : c'est du Go propre et commenté.

**Venant de Python :** il n'y a pas de `venv` parce qu'il n'y a pas de paquets installés « dans le système ». Chaque projet déclare ses dépendances dans son `go.mod` (section 1.5), avec leurs versions exactes, et le cache est partagé. Deux projets qui veulent deux versions d'une même bibliothèque cohabitent sans rien faire.

### 1.2 Le premier programme

Crée un dossier, et dedans un fichier `main.go` :

```go
package main

import "fmt"

func main() {
	fmt.Println("Bonjour depuis Go")
}
```

Cinq lignes, chacune obligatoire. Prends-les une par une, parce que tu vas les retrouver dans chaque programme.

- `package main` : chaque fichier Go appartient à un *paquet* (l'unité de code, comme un module Python ou une bibliothèque C). Le paquet `main` est spécial : c'est celui qui produit un exécutable. Un paquet qui s'appelle autrement produit une bibliothèque.
- `import "fmt"` : on déclare ce qu'on utilise. `fmt` (prononcé « fumt ») est le paquet de formatage de la bibliothèque standard, l'équivalent de `printf` et compagnie. **Piège :** importer un paquet qu'on n'utilise pas est une *erreur de compilation*, pas un avertissement. Pareil pour une variable déclarée et jamais lue. Go refuse le code mort. Ça agace le premier jour, puis on comprend que le code qu'on lit ne contient jamais de faux indices.
- `func main()` : le point d'entrée, comme `main` en C ou le bloc `if __name__ == "__main__"` en Python. Pas d'arguments, pas de valeur de retour (le code de sortie se fixe avec `os.Exit`).
- `fmt.Println(...)` : la fonction `Println` du paquet `fmt`. La majuscule n'est pas décorative : en Go, **ce qui commence par une majuscule est public (exporté), ce qui commence par une minuscule est privé au paquet**. Pas de `public`, pas de `_` de convention : la casse décide. C'est le mécanisme de visibilité de tout le langage.
- Les accolades sont obligatoires, et l'accolade ouvrante est **sur la même ligne** que `func`. Mettre `{` à la ligne suivante est une erreur de compilation (à cause de l'insertion automatique de points-virgules, expliquée en 1.4). Il n'y a donc qu'un seul style d'accolades en Go, et c'est celui-là.

Lance-le :

```
$ go run .
Bonjour depuis Go
```

### 1.3 `go run`, `go build`, `go install` : trois façons de lancer

`go run .` compile le paquet du dossier courant dans un fichier temporaire, le lance, et le jette. C'est le mode « je bricole ». Le point signifie « le paquet ici » ; tu peux aussi écrire `go run main.go`, mais prends l'habitude du point : dès que ton programme aura deux fichiers, seul le point marchera.

`go build .` compile et dépose l'exécutable dans le dossier courant, nommé comme le dossier (ou `-o monnom` pour choisir). C'est le mode « je livre » :

```
$ go build -o bonjour .
$ ls -la bonjour
-rwxr-xr-x  1 stranix  staff  2318754 bonjour
$ ./bonjour
Bonjour depuis Go
$ file bonjour
bonjour: Mach-O 64-bit executable arm64
```

Deux méga-octets pour cinq lignes ? Oui : le binaire contient l'exécutif Go (ramasse-miettes, ordonnanceur) et tout `fmt`. Il n'a besoin de rien d'autre pour tourner : copie-le sur un autre Mac Apple Silicon, il marche. Et pour un autre système, il suffit de deux variables d'environnement :

```
$ GOOS=linux GOARCH=amd64 go build -o bonjour-linux .
$ file bonjour-linux
bonjour-linux: ELF 64-bit LSB executable, x86-64, statically linked
```

Tu viens de compiler pour un serveur Linux x86 depuis un Mac ARM, sans rien installer. `GOOS` peut valoir `linux`, `darwin`, `windows`, `freebsd`… et `GOARCH` `amd64`, `arm64`, `arm`, `riscv64`… `go tool dist list` donne les combinaisons. C'est ce qui rend Go si agréable pour les outils d'infra : un `Makefile` de cinq lignes produit les binaires pour toutes les machines que tu gères.

**Venant du C :** pas de `-l`, pas de `-I`, pas de `LDFLAGS`, pas de liaison dynamique par défaut. Le compilateur trouve les paquets par leur nom d'import, et lie statiquement. (Il existe un pont vers le C, `cgo`, qui réintroduit tout ça ; on l'évite quand on peut, et ce cours n'y touche pas.)

`go install .` compile et dépose l'exécutable dans `~/go/bin`. C'est ainsi qu'on installe un outil écrit en Go depuis son code source, le sien ou celui d'un autre : `go install github.com/junegunn/fzf@latest` télécharge, compile et installe `fzf`. Pas de `pip install`, pas de paquet système : le code source suffit.

### 1.4 `gofmt` et les points-virgules invisibles

Go a une règle que les autres langages n'ont pas : **le formatage n'est pas un choix**. L'outil `gofmt` (ou `go fmt ./...`) réécrit ton fichier dans l'unique style officiel : tabulations, position des accolades, alignement des commentaires, ordre des imports. Tout le code Go du monde est formaté pareil. Configure ton éditeur pour le lancer à la sauvegarde (VS Code le fait avec l'extension Go ; Vim, Zed, GoLand aussi) et n'y pense plus jamais.

C'est aussi ce qui explique la règle des accolades. La grammaire de Go a des points-virgules, comme le C, mais le *lexer* les insère tout seul à la fin de chaque ligne qui se termine par un identifiant, une valeur, `)`, `]` ou `}`. Donc :

```go
func main()
{
```

est lu comme `func main();` puis `{`, ce qui n'a pas de sens. D'où l'accolade sur la même ligne, et d'où l'absence de point-virgule dans le code que tu écris. Même mécanisme pour les listes multi-lignes : la dernière virgule est obligatoire, parce que sans elle le lexer insère un point-virgule avant l'accolade fermante.

```go
ports := []int{
	22,
	80,
	443, // cette virgule est obligatoire
}
```

**Venant de Python :** Go a des accolades mais tu n'as pas à en discuter, et l'indentation n'a pas de sens pour le compilateur, seulement pour `gofmt` qui la remet. Le meilleur des deux mondes.

### 1.5 Modules et `go.mod`

Un *module* est un ensemble de paquets versionnés ensemble : ton projet. Il est défini par un fichier `go.mod` à sa racine. On le crée une fois :

```
$ go mod init exemple.com/bonjour
go: creating new go.mod: module exemple.com/bonjour
$ cat go.mod
module exemple.com/bonjour

go 1.27
```

Le nom du module est un chemin, par convention celui où le code sera publié (`github.com/stranix79/deckhand`). Pour un projet local, n'importe quel chemin marche (`cours-go/labs/01-hello` dans ce cours). Ce chemin est le préfixe de tous les imports internes au module : un paquet dans le sous-dossier `internal/config` s'importe avec `import "exemple.com/bonjour/internal/config"`. Le chapitre 9 y revient.

Les dépendances s'ajoutent avec `go get` et s'inscrivent dans `go.mod` avec leur version exacte, et dans `go.sum` avec leur empreinte cryptographique :

```
$ go get github.com/spf13/cobra@latest
$ cat go.mod
module exemple.com/bonjour

go 1.27

require github.com/spf13/cobra v1.10.1
```

`go mod tidy` ajoute ce qui manque et retire ce qui n'est plus importé. Lance-le avant chaque commit. Et `go build` télécharge tout seul ce dont il a besoin : sur un clone frais, il n'y a rien à installer.

**Piège :** `go run` et `go build` refusent de travailler hors d'un module (message `go: go.mod file not found`). Un dossier avec un `main.go` et pas de `go.mod`, ça ne se lance pas. Réflexe : `go mod init nom` avant tout.

### 1.6 `go vet`, `go test`, et les autres

Les commandes que tu utiliseras tous les jours, à connaître dès maintenant même si les chapitres suivants les détaillent :

| Commande | Ce qu'elle fait |
|---|---|
| `go run .` | compile et lance, sans laisser de fichier |
| `go build ./...` | compile tous les paquets du module (`./...` = « ici et tous les sous-dossiers ») |
| `go test ./...` | lance tous les tests (`*_test.go`), chapitre 9 |
| `go vet ./...` | analyse statique : `Printf` avec mauvais arguments, verrous copiés, etc. Gratuit, toujours le lancer |
| `go fmt ./...` | formate |
| `go mod tidy` | met `go.mod` et `go.sum` d'équerre |
| `go doc fmt.Println` | la doc d'un symbole, dans le terminal |
| `go env` | les variables d'environnement de l'outil |
| `go clean -cache` | vide le cache de compilation (rarement utile) |

Pour les outils tiers, deux à installer tout de suite :

```
$ brew install golangci-lint      # ou : go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
$ go install golang.org/x/tools/gopls@latest   # le serveur de langage, pour l'éditeur
```

`golangci-lint run ./...` enchaîne une trentaine d'analyseurs (variables inutilisées, erreurs ignorées, code inaccessible, etc.). `gopls` est ce que VS Code, Vim, Zed utilisent pour la complétion, le renommage et les erreurs en direct.

### 1.7 Lire la documentation

Go est un langage où l'on lit la doc dans le terminal et le code source de la bibliothèque standard, parce que les deux sont bons.

```
$ go doc fmt.Println
package fmt // import "fmt"

func Println(a ...any) (n int, err error)
    Println formats using the default formats for its operands and writes to
    standard output. Spaces are always added between operands and a newline
    is appended. It returns the number of bytes written and any write error
    encountered.
```

Cette signature t'apprend déjà deux choses sur Go : `...any` est un nombre variable d'arguments de n'importe quel type (chapitre 5), et la fonction renvoie **deux** valeurs, le nombre d'octets écrits et une erreur. On ignore les deux dans un `Println`, mais elles sont là. En ligne, [pkg.go.dev](https://pkg.go.dev/fmt) montre la même doc pour tous les paquets, standard et tiers, générée depuis les commentaires du code. Et le [Tour of Go](https://go.dev/tour/) et [Effective Go](https://go.dev/doc/effective_go) sont les deux documents officiels à lire en complément de ce cours, dans cet ordre.

### 1.8 Pour le labo

Le [labo 01](../labs/01-hello/README.md) te fait créer un module, écrire un programme qui affiche la machine et le système sur lesquels il tourne, le compiler pour trois systèmes, et vérifier avec `file` que le résultat est bien un binaire statique. Dix minutes, mais après ça tu sais livrer un programme Go, ce qui prend une matinée dans la plupart des autres langages.

### À retenir

- Une seule commande, `go`, fait tout : `run`, `build`, `test`, `vet`, `fmt`, `mod`, `doc`, `install`.
- Un programme = `package main` + `func main()`, dans un module défini par `go.mod` (`go mod init` d'abord).
- Ce qui commence par une majuscule est exporté (public), le reste est privé au paquet.
- Un import ou une variable inutilisés font échouer la compilation : Go refuse le code mort.
- `gofmt` impose le style, l'accolade est sur la ligne du `func`, la virgule finale est obligatoire dans les listes multi-lignes.
- `go build` produit un binaire statique ; `GOOS=linux GOARCH=amd64 go build` compile pour une autre machine sans rien installer.
- Les dépendances vivent dans `go.mod` (versions) et `go.sum` (empreintes) ; `go mod tidy` avant chaque commit.
- La doc est dans le terminal (`go doc`) et sur pkg.go.dev ; le code de la bibliothèque standard se lit.

---

← [0. Pourquoi Go, et quand l'utiliser (ou pas)](00-pourquoi-go.md) · [Sommaire](../README.md) · [2. Variables, types, constantes, chaînes](02-variables-types-chaines.md) →
