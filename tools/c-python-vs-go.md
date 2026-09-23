# C / Python / Go : la table de correspondance

*Pour qui vient du C et de Python et cherche « comment on dit en Go ». La colonne **Piège** signale ce qui se comporte autrement qu'attendu quand on traduit mot à mot. Renvois vers le [cours](../README.md) et l'[antisèche](go-antiseche.md).*

## Déclarer, typer, nommer

| Quoi | C | Python | Go | Piège |
|---|---|---|---|---|
| Déclarer une variable | `int n = 5;` | `n = 5` | `n := 5` ou `var n int = 5` | `:=` déclare, `=` affecte. `:=` dans un bloc interne crée une **nouvelle** variable qui masque l'externe (`err` redéclaré dans un `if`). Une variable déclarée et jamais lue ne compile pas (chap. 2). |
| Type | fixé à la déclaration | porté par la valeur | fixé, mais déduit par `:=` | pas de conversion implicite du tout : `int + int64` ne compile pas, `float64(n)` obligatoire. |
| Constante | `#define N 5`, `const int` | `N = 5` (convention) | `const N = 5` ; `iota` pour énumérer | une `const` non typée prend le type de l'usage ; pas de `const` pour une slice, une map ou une struct (utiliser `var`). |
| Entier | 32/64 bits, débordement silencieux | illimité | `int` (64 bits), `int8`…`uint64`, débordement silencieux | pas d'entier illimité : `math/big` pour ça. `uint` piège classique : avec `var u uint`, `u - 1` fait 2^64-1 sans erreur (et la constante `uint(0) - 1` ne compile même pas). |
| Flottant | `float`, `double` | `float` | `float32`, `float64` | `0.1 + 0.2 != 0.3` partout. Une constante `3` va dans un `float64` sans conversion, une variable `int` non. |
| Booléen | `0` / non-0 | `True` / `False` ; `0`, `""`, `[]` sont faux | `true` / `false`, et rien d'autre | `if n {` avec `n int` ne compile pas ; écrire `if n != 0 {`. Pas de « truthy ». |
| Null | `NULL` | `None` | `nil` (pointeur, slice, map, chan, func, interface) | une **slice nil** se lit et s'`append` ; une **map nil** se lit mais l'écriture panique. Une interface qui contient un pointeur nil n'est pas nil (chap. 7). |
| Chaîne | `char*` terminé par `\0` | `str` Unicode immuable | `string` immuable, octets UTF-8 | `len("é") == 2` (octets). `s[0]` est un `byte`, pas un caractère. `for _, r := range s` itère sur les runes. Pas de `s[i] = 'x'`. |
| Caractère | `'a'` (un `int`) | `"a"` (str de longueur 1) | `'a'` (un `rune` = `int32`) | `string(65)` donne `"A"`, `string(n)` pour convertir un nombre est faux : `strconv.Itoa`. |
| Nommage | `snake_case` | `snake_case`, `PascalCase` | `camelCase` / `PascalCase` ; **la majuscule initiale exporte** | pas de `public`/`private`, pas de `_` : `nom` est privé au paquet, `Nom` est public. Les acronymes restent en capitales (`ServeHTTP`, `hostID`). |
| Commentaire | `/* */`, `//` | `#`, docstring | `//`, `/* */` ; le commentaire juste au-dessus d'un symbole exporté est sa doc | la doc commence par le nom du symbole : `// Sonder vérifie…`. `go doc` et pkg.go.dev la lisent. |

## Opérateurs et contrôle

| Quoi | C | Python | Go | Piège |
|---|---|---|---|---|
| Division | `7 / 2 == 3` | `7 / 2 == 3.5`, `7 // 2 == 3` | `7 / 2 == 3` (entiers), `7.0 / 2 == 3.5` | comme en C : `float64(a) / float64(b)` pour un vrai quotient. |
| Modulo | signe du dividende | signe du diviseur | signe du dividende (comme C) : `-7 % 2 == -1` | pas comme Python. |
| Incrément | `i++`, `++i`, `n = i++` | `i += 1` | `i++` **instruction seulement** | `n = i++` et `++i` ne compilent pas. |
| Ternaire | `c ? a : b` | `a if c else b` | n'existe pas | un `if` de quatre lignes, ou une petite fonction. |
| Logique | `&&`, `\|\|`, `!` | `and`, `or`, `not` | `&&`, `\|\|`, `!` | évaluation paresseuse comme C, mais renvoie un `bool`, jamais l'opérande (pas de `x or défaut`). |
| Bits | `& \| ^ ~ << >>` | idem | `& \| ^ << >>`, `^x` (not), `&^` (and not) | `~` n'existe pas : `^x`. |
| Blocs | `{ }` | indentation | `{ }` obligatoires, `{` sur la même ligne | `if (c)\n{` ne compile pas (point-virgule inséré). Pas de parenthèses autour de la condition (chap. 1). |
| `if` | `if (c) {} else if {} else {}` | `if:` / `elif:` / `else:` | `if c {} else if c2 {} else {}` ; `if x := f(); x > 0 {` | la variable de l'instruction courte vit dans le `if`/`else` seulement. |
| `switch` | `switch/case/break` | `match` | `switch x { case 1, 2: … default: }` | **pas de fallthrough** par défaut (`fallthrough` explicite). `switch {` sans expression remplace une chaîne de `else if`. `break` dans un `switch` sort du `switch`, pas de la boucle. |
| `for` compteur | `for (i = 0; i < n; i++)` | `for i in range(n):` | `for i := 0; i < n; i++ {` ou `for i := range n {` (1.22) | c'est la **seule** boucle : `for cond {` est le `while`, `for {` l'infini. |
| `for` sur collection | index à la main | `for x in xs:` | `for i, x := range xs {` | `x` est une **copie** de l'élément : modifier `x` ne modifie pas la slice ; écrire `xs[i].Champ = …`. Depuis 1.22 chaque itération a sa propre variable (fini le piège des closures). |
| `while` | `while (c)` | `while c:` | `for c {` | pas de `do … while` : `for { …; if !c { break } }`. |
| `goto` | oui | non | oui, et `break`/`continue` avec étiquette | `break Externe` pour sortir d'une boucle imbriquée, plus propre qu'un drapeau. |

## Fonctions

| Quoi | C | Python | Go | Piège |
|---|---|---|---|---|
| Définir | `int f(int a, int b) { }` | `def f(a, b):` | `func f(a, b int) int { }` | le type **après** le nom, et `a, b int` groupe deux paramètres du même type. |
| Retour multiple | struct ou pointeur de sortie | tuple | `func f() (int, error)` ; `n, err := f()` | pas un tuple : chaque valeur doit être reçue ou ignorée avec `_`. Convention : l'erreur en dernier (chap. 5). |
| Arguments par défaut / nommés | non | `def f(a, b=2)`, `f(b=3)` | **non** | une struct d'options `Config{Port: 80}` (les champs absents valent zéro), ou des fonctions d'options. |
| Nombre variable | `va_list` | `*args` | `func f(xs ...int)` ; `f(s...)` | une seule variadique, en dernier ; `s...` déballe une slice. |
| Passage | par valeur, pointeur pour modifier | par référence d'objet | **par valeur, toujours** ; `*T` pour modifier | une struct est copiée à l'appel ; une slice ou une map est un petit en-tête copié qui pointe vers les mêmes données : modifier les éléments se voit, `append` ou réaffecter non (chap. 4, 6). |
| Fonction anonyme | pointeur de fonction | `lambda` | `func(x int) int { return x * 2 }` | c'est une closure complète, multi-instructions, qui capture les variables par référence. |
| Fonctions comme valeurs | pointeur | naturel | naturel : `var f func(int) int = carre` | un type fonction s'écrit `func(int) error`, on peut lui donner des méthodes (`http.HandlerFunc`). |
| Portée | bloc | fonction | **bloc** (comme C) | une variable déclarée dans un `if` ou un `for` n'existe pas après. |
| `main` | `int main(int argc, char **argv)` | `if __name__ == "__main__":` | `func main()` dans `package main`, sans arguments ni retour | `os.Args` pour les arguments, `os.Exit(1)` pour le code de retour ; `defer` ne s'exécute pas après `os.Exit`. |
| Generic | macros, `void*` | duck typing | `func Max[T cmp.Ordered](a, b T) T` (1.18+) | rarement nécessaires : interfaces et `slices`/`maps` couvrent 90 % des cas (chap. 16). |

## Erreurs

| Quoi | C | Python | Go | Piège |
|---|---|---|---|---|
| Signaler | code de retour + `errno` | `raise` | `return valeur, err` ; `errors.New`, `fmt.Errorf` | pas d'exceptions : chaque appel qui peut échouer renvoie `error`, testée **tout de suite** avec `if err != nil` (chap. 5). |
| Attraper | `if (r < 0)` | `try / except` | `if err != nil { return fmt.Errorf("contexte : %w", err) }` | oublier de tester une erreur compile (golangci-lint `errcheck` le voit). Ne pas journaliser **et** renvoyer : l'un ou l'autre. |
| Type d'erreur | `errno` entier | classes d'exceptions | `errors.Is(err, ErrX)` (valeur), `errors.As(err, &cible)` (type) | `err == ErrX` rate les erreurs enveloppées : toujours `errors.Is`. `%w` dans `fmt.Errorf`, pas `%v`, sinon la chaîne est coupée (chap. 8). |
| Chaîner | non | `raise … from e` | `fmt.Errorf("lire %s : %w", p, err)` | contexte devant, en minuscules, sans point final, sans « erreur : » (le message final se lit de gauche à droite : `main : lire config : open x : no such file`). |
| Plantage | segfault | traceback | `panic` + trace de toutes les goroutines | `panic` est pour les bugs (index hors bornes, nil déréférencé), pas pour « fichier absent ». `recover` dans un `defer`, seulement à la frontière (middleware HTTP). |
| Nettoyage | à la main | `finally`, `with` | `defer f.Close()` juste après l'ouverture | `defer` dans une boucle s'accumule jusqu'à la fin de la **fonction**, pas de l'itération. |
| Assertion | `assert()` | `assert` | n'existe pas | on écrit un `if` et un `panic`, ou un test. |

## Données composites

| Quoi | C | Python | Go | Piège |
|---|---|---|---|---|
| Tableau fixe | `int t[10]` | (liste) | `[10]int` | copié par **valeur** à l'affectation et à l'appel. Rare : on utilise la slice. |
| Tableau dynamique | `malloc` + `realloc` | `list` | `[]int` (slice) : `make([]int, 0, n)`, `append`, `s[a:b]` | une tranche **partage** la mémoire : modifier `s[1:3][0]` modifie `s[1]`. `append` peut réallouer ou non : toujours `s = append(s, x)`. `slices.Clone` pour copier (chap. 4). |
| Dictionnaire | à écrire | `dict` | `map[string]int` ; `v, ok := m[k]` ; `delete(m, k)` | clé absente = valeur zéro sans erreur (d'où le `, ok`). Ordre d'itération **aléatoire** par conception. Pas sûre en écriture concurrente : mutex. |
| Ensemble | à écrire | `set` | `map[string]struct{}` ou `map[string]bool` | pas de type set ; `struct{}` occupe zéro octet. |
| Structure | `struct` | `@dataclass`, classe | `type Hote struct { Nom string; Port int }` | valeur zéro utilisable sans constructeur ; copiée à l'affectation ; comparable avec `==` si tous ses champs le sont. |
| Énumération | `enum` | `Enum` | `const ( A = iota; B; C )` avec un type nommé | ce sont des entiers : rien n'empêche `Etat(42)`. `String()` à écrire (ou `go generate` + `stringer`). |
| Tuple | non | `(1, "a")` | n'existe pas | retour multiple pour les fonctions, struct anonyme `struct{ a int; b string }` sinon. |
| Copie | `memcpy`, `=` sur struct | `copy.deepcopy` | `=` copie une struct ou un tableau ; `slices.Clone`, `maps.Clone` pour une collection | copier une struct qui contient une slice copie l'en-tête, pas les éléments : copie superficielle. |
| Chaîne de format | `printf("%d %s")` | `f"{n} {s}"` | `fmt.Printf("%d %s", n, s)`, `fmt.Sprintf`, `%v` pour tout | `go vet` vérifie les verbes contre les arguments. `Println` ajoute des espaces, `Print` non. |
| JSON | à la main | `json.loads` | `json.Unmarshal(data, &v)` sur une struct avec tags | seuls les champs **exportés** (majuscule) sont encodés ; `omitempty` pour omettre les zéros ; un `int` JSON devient `float64` dans une `map[string]any` (chap. 12). |

## Mémoire, pointeurs, objets

| Quoi | C | Python | Go | Piège |
|---|---|---|---|---|
| Allocation | `malloc` / `free` | automatique | automatique (GC) : `&Hote{}` ou `new(Hote)` | pas de `free`. Le compilateur décide pile ou tas (analyse d'échappement, chap. 10) ; renvoyer `&local` est **valide**. |
| Pointeur | `int *p = &n; *p = 3;` | non | `p := &n; *p = 3` | pas d'arithmétique (`p++` interdit), pas de cast libre. `p.Champ` sans `->`. `nil` déréférencé = panic, pas segfault. |
| Struct vs classe | `struct` | `class` | `struct` + méthodes `func (h *Hote) M()` | pas de classe, pas d'héritage, pas de constructeur : une fonction `NouvelHote()` par convention (chap. 6). |
| `this` | pointeur explicite | `self` | le **receveur** nommé librement : `func (h *Hote)` | receveur valeur = copie (ne modifie pas), receveur pointeur = modifie. Mélanger les deux sur un type est une source de bugs. |
| Héritage | non | `class B(A)` | **composition** : `type Serveur struct { Hote }` promeut champs et méthodes | ce n'est pas du sous-typage : un `*Serveur` n'est pas un `*Hote`, et une méthode promue ne peut pas être « surchargée » par polymorphisme (chap. 7). |
| Interface | non | duck typing, `Protocol` | `type Sondeur interface { Sonder() error }`, satisfaite implicitement | comme un `Protocol` vérifié à la compilation. Définir l'interface côté **consommateur**, petite (chap. 7). |
| Mutabilité | tout mutable | dépend du type | tout mutable sauf `string` et les constantes | les chaînes se reconstruisent (`strings.Builder` en boucle). |
| Destructeur | `free` | `__del__`, `with` | `defer x.Close()` | pas de finaliseur fiable ; libérer explicitement avec `defer`. |
| Taille en mémoire | `sizeof` | `sys.getsizeof` | `unsafe.Sizeof` | rarement utile ; l'ordre des champs joue sur l'alignement comme en C. |
| Concurrence | `pthread`, mutex à la main | `threading` (GIL), `asyncio`, `multiprocessing` | `go f()`, channels, `select`, `sync.Mutex`, `context` | tous les cœurs sans effort, mais une map ou un compteur partagés sans mutex sont une **data race** : `go test -race` (chap. 11). `main` qui se termine tue toutes les goroutines. |

## Compilation, exécution, organisation

| Quoi | C | Python | Go | Piège |
|---|---|---|---|---|
| Chaîne d'outils | préprocesseur, `cc`, `ld`, `make` | interpréteur | `go build` : tout en un, lié statiquement | pas de `Makefile` nécessaire (on en écrit un de dix lignes pour les raccourcis, chap. 15). |
| En-tête / import | `#include "x.h"` | `import x` | `import "chemin/module/paquet"` | un import inutilisé ne compile pas. Pas d'import cyclique. Pas de `from x import *`. |
| Unité de code | fichier `.c` | module `.py` | **dossier** = paquet ; plusieurs fichiers partagent le même espace | tous les fichiers d'un dossier ont le même `package`. `internal/` n'est importable que depuis le module parent (chap. 9). |
| Dépendances | système, `-lfoo` | `pip`, venv | `go.mod` + `go.sum`, cache partagé `~/go/pkg/mod` | pas de venv : les versions sont dans `go.mod`. `go mod tidy` avant de committer. |
| Préprocesseur | `#ifdef` | non | `//go:build linux` en tête de fichier (build tags), `runtime.GOOS` | un fichier `x_linux.go` n'est compilé que sur Linux : `go vet` avec `GOOS=windows` pour vérifier l'autre branche. |
| Exécutable | `./a.out` + libs | `python3 x.py` + interpréteur | un binaire statique, `GOOS=linux GOARCH=arm64 go build` | `CGO_ENABLED=0` pour être vraiment statique (obligatoire dans `scratch`, chap. 15). |
| Vitesse | référence | 20 à 100× plus lent | 1,2 à 2× plus lent que C | le GC coûte un peu ; le vrai gain est la concurrence facile (chap. 0). |
| Formatage | style au choix | PEP 8, `ruff format` | `gofmt`, un seul style, tabulations | ne se discute pas. `gofmt -l .` en CI. |
| Lint | `-Wall`, clang-tidy | `ruff`, `mypy` | `go vet`, `golangci-lint`, `staticcheck` | `go vet` toujours ; `golangci-lint` avec `.golangci.yml` (chap. 15). |
| Tests | à la main, cmocka | `pytest` | `go test`, fichiers `_test.go`, `func TestX(t *testing.T)` | pas d'`assert` : `if got != want { t.Errorf(...) }`. Table-driven + `t.Run`. `-race`, `-cover`, `-bench` intégrés (chap. 9). |
| Débogueur | `gdb` | `pdb` | `dlv` (Delve) | `fmt.Printf("%+v")` et `go test -v` couvrent l'essentiel ; `pprof` pour le profilage (chap. 15). |
| Journal | `fprintf(stderr)` | `logging` | `log` (simple), `log/slog` (structuré, JSON) | `log.Fatal` appelle `os.Exit(1)` : les `defer` ne s'exécutent pas. |

## Le réflexe à prendre

En C, tu penses en octets et en adresses ; en Python, en objets et en étiquettes. En Go, pense en **valeurs qui se copient et en petits en-têtes qui partagent** : une struct est copiée, une slice ou une map est un en-tête de trois mots qui pointe vers des données partagées, un pointeur est explicite et sans arithmétique, une interface est une paire (type, valeur). Le reste (receveur pointeur ou valeur, `append` qui réalloue, la map nil, l'interface qui contient un nil) en découle. C'est le chapitre 4 pour les collections, le 6 pour les structs, le 10 pour ce qui se passe en mémoire.
