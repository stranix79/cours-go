# 2. Variables, types, constantes, chaînes

*Go de zéro à la prod : chapitre 2 sur 16.* ← [1. Installer, outiller, lancer](01-installer-et-outiller.md) · [Sommaire](../README.md) · [3. Contrôle : if, for, switch, defer](03-controle.md) →

Go est typé statiquement, comme le C : chaque variable a un type fixé à la compilation, et il ne changera pas. Mais contrairement au C, tu n'as presque jamais besoin de l'écrire, parce que le compilateur le déduit de la valeur. Et contrairement à Python, une chaîne ne se mélange jamais avec un entier, un `int` ne se mélange jamais avec un `float64`, et le programme qui essaie ne compile pas. Ce chapitre pose ces règles, puis s'attarde sur les chaînes de caractères, parce qu'elles sont l'endroit où Go diffère le plus de ce que tu connais : une chaîne Go est une suite d'octets UTF-8, pas de caractères.

### 2.1 Déclarer : `var`, `:=` et la valeur zéro

Deux façons de créer une variable. La forme longue, avec le mot-clé `var`, le nom, puis le type (**dans cet ordre**, l'inverse du C) :

```go
var port int = 5432
var hote string = "db01"
var actif bool
var charge float64
var nom string
```

Les trois dernières n'ont pas de valeur initiale. En C, elles contiendraient n'importe quoi. En Go, **toute variable déclarée sans valeur reçoit la valeur zéro de son type** : `0` pour les nombres, `false` pour les booléens, `""` (la chaîne vide) pour les chaînes, `nil` pour les pointeurs, slices, maps et fonctions. Il n'y a pas de variable non initialisée en Go. C'est une des raisons pour lesquelles un programme Go ne fait pas n'importe quoi au démarrage.

La forme courte, `:=`, déclare et initialise en déduisant le type de la valeur. C'est celle que tu écriras dans 95 % des cas :

```go
timeout := 30                    // int
chemin := "/var/log/postgresql"  // string
ratio := 0.75                    // float64
```

```go
package main

import "fmt"

func main() {
	var port int = 5432
	var hote string = "db01"
	var actif bool
	var charge float64
	var nom string
	fmt.Println(port, hote, actif, charge, nom)
	fmt.Printf("%q\n", nom)

	timeout := 30
	chemin := "/var/log/postgresql"
	ratio := 0.75
	fmt.Printf("%T %T %T\n", timeout, chemin, ratio)
}
```

```
$ go run .
5432 db01 false 0 
""
int string float64
```

Le `%q` montre la chaîne vide entre guillemets, sinon on ne la verrait pas ; `%T` affiche le type. Les deux sont détaillés en 2.7.

Quand utiliser l'une ou l'autre ? `:=` dès que tu as une valeur. `var` quand tu veux la valeur zéro (`var compteur int`, `var sb strings.Builder`), quand tu déclares hors d'une fonction (`:=` n'y est pas permis), ou quand tu veux un type différent de celui que la déduction choisirait (`var octets int64 = 0`, parce que `0` seul donnerait un `int`).

**Venant de Python :** `:=` ressemble à `=`, mais ce n'est pas une étiquette qu'on recolle. Une fois que `port := 5432` a été écrit, `port` est un `int` pour toujours : `port = "cinq mille"` ne compile pas. Et `:=` **déclare** : l'écrire deux fois pour la même variable dans le même bloc est une erreur (`no new variables on left side of :=`). Pour réaffecter, c'est `=`.

**Piège :** une variable déclarée et jamais utilisée est une erreur de compilation, pas un avertissement :

```
$ go run .
./main.go:6:2: declared and not used: port
```

Idem pour un import inutilisé (`"os" imported and not used`). Tu vas te faire attraper dix fois le premier jour, en particulier quand tu commentes une ligne pour tester. Le réflexe : `_ = port` pour faire taire le compilateur le temps de bricoler, et retirer la ligne avant de commiter. La règle vaut pour les variables locales seulement, pas pour les variables de paquet ni pour les paramètres de fonction.

### 2.2 Les types de base

Go a peu de types, et ils ont tous une taille connue. Ceux que tu utiliseras :

| Type | Taille | Usage |
|---|---|---|
| `int`, `uint` | 64 bits sur toute machine 64 bits | l'entier par défaut, les index, les compteurs |
| `int8`, `int16`, `int32`, `int64` | ce que le nom dit | quand la taille compte : fichiers, protocoles, bases de données |
| `uint8`, `uint16`, `uint32`, `uint64` | idem, non signés | octets, masques, identifiants |
| `float32`, `float64` | IEEE 754 | `float64` par défaut, c'est le `double` du C |
| `bool` | 1 octet | `true`, `false` |
| `string` | 16 octets (pointeur + longueur) | texte, en UTF-8 |
| `byte` | alias de `uint8` | un octet |
| `rune` | alias de `int32` | un point de code Unicode |

**Venant du C :** `int` est 64 bits sur une machine 64 bits, il n'y a pas de `long`, `short`, `unsigned` en préfixe, ni de `char`. Le `char` est remplacé par deux types selon ce que tu veux dire : `byte` pour un octet, `rune` pour un caractère. Il n'y a pas de promotion implicite : un `int32` plus un `int64`, ça ne compile pas.

Les entiers débordent silencieusement, comme en C (pas de vérification, pas d'exception, pas d'entier illimité comme en Python) :

```go
var petit uint8 = 255
petit++
fmt.Println(petit)          // 0
var grand int32 = 70000
fmt.Println(int16(grand))   // 4464 : 70000 modulo 65536
```

La division entière tronque vers zéro et le reste garde le signe du dividende, comme en C, pas comme en Python : `-7/2` donne `-3` et `-7%2` donne `-1`. Le `float64` a les arrondis d'IEEE 754 que tu connais : `0.1+0.2 != 0.3`. Pour de l'argent, on compte en centimes dans un `int64`.

### 2.3 Conversions : toujours explicites

Il n'y a pas de cast implicite en Go. Aucun. Un `int` ne devient pas un `float64` tout seul, un `int32` ne devient pas un `int64`, un `int` ne devient pas une chaîne. Tu écris la conversion sous la forme `Type(valeur)` :

```go
package main

import "fmt"

func main() {
	var octets int64 = 1536
	ko := float64(octets) / 1024
	fmt.Println(ko)

	n := 7
	d := 2.0
	fmt.Println(float64(n) / d)
	fmt.Println(n / 2)
	x := 3.99
	fmt.Println(int(x))
}
```

```
$ go run .
1.5
3.5
3
3
```

`int(x)` tronque vers zéro, comme `(int)x` en C. Et si tu oublies la conversion :

```
$ go run .
./main.go:8:14: invalid operation: n / d (mismatched types int and float64)
```

C'est la ligne d'erreur que tu verras le plus souvent les premiers jours. Elle est volontaire : dans les langages à promotion implicite, le bug où un `int` devient un `float` sans qu'on le voie est classique, et Go préfère t'obliger à l'écrire.

**Piège :** `string(65)` ne donne pas `"65"` mais `"A"` (le caractère de code 65). `go vet` te le signale, et la vraie conversion nombre vers texte est `strconv.Itoa(65)`, vue en 2.6. Dans l'autre sens, `int("42")` n'existe pas du tout : c'est `strconv.Atoi`.

### 2.4 Constantes et `iota`

`const` déclare une valeur fixée à la compilation. Contrairement à Python, elle est vraiment constante, et contrairement au `#define` du C, elle est typée et connue du compilateur :

```go
const PortPostgres = 5432
const Version = "1.4.2"
const Pi = 3.14159265358979
```

Ces trois constantes sont *non typées* : elles n'ont pas de type fixé tant qu'on ne les utilise pas, ce qui permet `var f float64 = Pi * 2` et `var i int = PortPostgres + 1` sans conversion. Les constantes sont aussi calculées avec une précision arbitraire : `const grand = 1 << 100` est légal, alors qu'aucune variable ne pourrait le contenir.

Pour les énumérations, Go n'a pas de mot-clé `enum`. Il a `iota`, un compteur qui vaut 0 sur la première ligne d'un bloc `const` et s'incrémente à chaque ligne :

```go
type Niveau int

const (
	Debug Niveau = iota // 0
	Info                // 1
	Warn                // 2
	Erreur              // 3
)

const (
	_  = iota             // 0, jeté
	Ko = 1 << (10 * iota) // 1 << 10
	Mo                    // 1 << 20
	Go                    // 1 << 30
	To                    // 1 << 40
)
```

Une ligne sans expression répète l'expression de la ligne précédente avec le nouvel `iota`. `type Niveau int` déclare un nouveau type dont la représentation est un `int` : c'est ainsi qu'on donne un nom à une famille de constantes (chapitre 6 pour les types nommés).

```
$ go run .
0 1 2 3
1024 1048576 1073741824 1099511627776
6.28318530717958 5433
main.Niveau int
```

**Venant du C :** `iota` remplace l'`enum` et les `#define` en cascade. Comme en C, rien n'empêche de mettre `Niveau(42)` dans une variable de type `Niveau` ; ce n'est pas un type fermé.

### 2.5 Chaînes : des octets UTF-8, immuables

Voilà le point important du chapitre. Une `string` en Go est une **suite d'octets immuable**, et par convention ces octets sont de l'UTF-8. Le langage ne t'empêche pas d'y mettre n'importe quels octets, mais tout l'écosystème suppose de l'UTF-8, et les littéraux dans ton code source en sont.

Conséquence : `len(s)` compte des octets, pas des caractères, et `s[i]` renvoie un octet (`byte`), pas un caractère.

```go
package main

import (
	"fmt"
	"unicode/utf8"
)

func main() {
	s := "été"
	fmt.Println(len(s))                    // octets
	fmt.Println(utf8.RuneCountInString(s)) // caractères
	fmt.Println(s[0], s[1])                // deux octets du premier é
	fmt.Printf("% x\n", s)                 // tous les octets en hexa
	for i, r := range s {                  // range décode l'UTF-8
		fmt.Printf("%d:%c ", i, r)
	}
	fmt.Println()
	fmt.Println(s[2:]) // sous-chaîne à partir de l'octet 2
}
```

```
$ go run .
5
3
195 169
c3 a9 74 c3 a9
0:é 2:t 3:é 
té
```

`"été"` fait 5 octets pour 3 caractères, parce que `é` s'écrit `c3 a9` en UTF-8. `range` sur une chaîne fait le travail de décodage et donne, à chaque tour, l'index de l'octet où commence le caractère et le caractère lui-même sous forme de `rune`. C'est la seule boucle qui sache faire ça ; `for i := 0; i < len(s); i++` te donnerait des octets.

**`byte` et `rune`.** Un `byte` est un `uint8` : un octet. Une `rune` est un `int32` : un point de code Unicode, c'est-à-dire un caractère. Un littéral entre apostrophes est une `rune` : `'A'` vaut 65, `'é'` vaut 233. Une chaîne se convertit dans les deux sens :

```go
b := []byte(s) // copie des octets
r := []rune(s) // décodage en caractères
```

Modifier `r[0]` puis `string(r)` donne `"Eté"`. Modifier `b[0]` sur `"été"` casse le premier `é` (il ne reste que son second octet, ce qui donne `E�té`). Règle : pour manipuler des caractères, passe par `[]rune` ; pour manipuler des octets (protocoles, fichiers binaires), `[]byte`.

**Immuable.** On ne modifie pas une chaîne : `s[0] = 'x'` ne compile pas (`cannot assign to s[0]`). On en construit une autre avec `+`, ou avec `strings.Builder` quand il y a beaucoup de morceaux (chapitre 4). Sous le capot, une `string` est un pointeur vers des octets plus une longueur, seize octets en tout ; la passer à une fonction ne copie pas le texte, et `len` est immédiat.

**Venant du C :** pas de `\0` final, pas de `strlen` qui parcourt, pas de tampon à dimensionner, pas de dépassement. Mais l'idée que le texte est fait d'octets est la même, et `s[i]` est bien un `char` au sens C. Le `char` pour « un caractère » est devenu `rune`.

**Venant de Python :** une `str` Python est une suite de caractères, et `bytes` est à part. En Go, la `string` joue les deux rôles : c'est des octets, qu'on interprète comme de l'UTF-8 quand on a besoin de caractères. `len("été")` vaut 3 en Python et 5 en Go. Pour le compte de caractères, `utf8.RuneCountInString`.

Deux formes de littéraux : entre guillemets doubles, avec les échappements habituels (`"\t"`, `"\n"`, `"\""`), et entre accents graves, où rien n'est interprété et où les retours à la ligne sont conservés :

```go
multi := `chemin: C:\logs\app.log
deux lignes`
```

C'est la chaîne brute (`r"..."` en Python), pratique pour les expressions régulières, les chemins Windows et les blocs de JSON dans les tests. Il n'y a pas de guillemets simples pour les chaînes : `'a'` est une `rune`, `"a"` une `string`.

### 2.6 `strings` et `strconv`

Il n'y a pas de méthodes sur les chaînes au sens Python (`s.upper()`). Tout est dans le paquet `strings`, sous forme de fonctions qui prennent la chaîne en premier argument et en renvoient une nouvelle. Les indispensables :

```go
ligne := "  web01,db02, cache03 ,web01  "
strings.TrimSpace(ligne)                  // "web01,db02, cache03 ,web01"
strings.Split("a,b,c", ",")               // []string{"a", "b", "c"}
strings.Fields("  ERROR   disk   full  ") // coupe sur les blancs : [ERROR disk full]
strings.Join([]string{"a", "b"}, " | ")   // "a | b"
strings.Contains(ligne, "db")             // true
strings.HasPrefix("web01", "web")         // true
strings.HasSuffix("app.log", ".log")      // true
strings.Index("app.log", ".")             // 3 ; -1 si absent
strings.ToUpper("erreur")                 // "ERREUR"
strings.ReplaceAll("a-b-c", "-", "_")     // "a_b_c"
strings.Repeat("-", 20)                   // une ligne de tirets
```

La conversion entre texte et nombres est dans `strconv` (*string conversion*). Elle renvoie toujours deux valeurs, le résultat et une erreur, parce qu'un texte peut ne pas être un nombre :

```go
n, err := strconv.Atoi("5432")      // 5432 <nil>
n, err = strconv.Atoi("54a32")      // 0 strconv.Atoi: parsing "54a32": invalid syntax
f, err := strconv.ParseFloat("0.75", 64)
b, err := strconv.ParseBool("true")
strconv.Itoa(8080) + "/tcp"         // "8080/tcp"
strconv.FormatInt(255, 16)          // "ff"
```

`Atoi` et `Itoa` (*ASCII to integer* et l'inverse) sont les deux à retenir. Ce `n, err :=` est ta première rencontre avec la convention d'erreur de Go, qui a droit à tout le chapitre 5 : la fonction renvoie un résultat et une erreur, et `err` vaut `nil` quand tout va bien.

### 2.7 `fmt.Printf` et ses verbes

`fmt.Println` affiche ses arguments séparés par des espaces, avec un retour à la ligne. `fmt.Printf` prend un format, comme le `printf` du C, avec des verbes qui commencent par `%`. Et `fmt.Sprintf` fait pareil mais renvoie la chaîne au lieu de l'afficher. Ceux que tu utiliseras :

| Verbe | Effet |
|---|---|
| `%v` | la valeur, dans un format par défaut raisonnable ; marche pour tout |
| `%+v` | idem, avec le nom des champs pour une struct |
| `%T` | le type de la valeur |
| `%d` | entier décimal ; `%5d` sur 5 colonnes, `%-5d` aligné à gauche, `%05d` avec des zéros |
| `%f` | flottant ; `%.2f` deux décimales, `%8.1f` sur 8 colonnes |
| `%s` | chaîne ; `%10s`, `%-10s` pour aligner |
| `%q` | chaîne entre guillemets, échappée : pour voir les espaces et les vides |
| `%t` | booléen |
| `%c` | le caractère d'une rune ; `%U` son code Unicode |
| `%x` `%X` `%o` `%b` | hexa, hexa majuscule, octal, binaire |
| `%p` | un pointeur |
| `%%` | le signe `%` |

```go
type Sonde struct {
	Nom  string
	Port int
	OK   bool
}
s := Sonde{"pg-primaire", 5432, true}
charge := 0.7345
fmt.Printf("%v\n", s)
fmt.Printf("%+v\n", s)
fmt.Printf("%T\n", s)
fmt.Printf("%d %5d|%-5d|%05d\n", 42, 42, 42, 42)
fmt.Printf("%f %.2f %8.1f|%e\n", charge, charge, charge, charge)
fmt.Printf("%s|%10s|%-10s|%q\n", "web01", "web01", "web01", "web01")
fmt.Printf("%t %c %U %x %X %o %b\n", true, 'é', 'é', 255, 255, 8, 5)
fmt.Printf("%v %v %v\n", []int{1, 2}, map[string]int{"a": 1}, nil)
```

```
{pg-primaire 5432 true}
{Nom:pg-primaire Port:5432 OK:true}
main.Sonde
42    42|42   |00042
0.734500 0.73      0.7|7.345000e-01
web01|     web01|web01     |"web01"
true é U+00E9 ff FF 10 101
[1 2] map[a:1] <nil>
```

`%v` est le verbe paresseux et il est très bien : il affiche correctement un entier, une chaîne, un slice, une map, une struct. `%+v` est celui du débogage. Les largeurs (`%-12s %6d`) font des tableaux alignés dans le terminal, ce dont tu auras besoin dès le labo de ce chapitre.

**Venant du C :** c'est le même `printf`, à trois différences près. Il n'y a pas de `%lu`, `%lld`, `%zu` : `%d` marche pour tous les entiers, le compilateur connaît la taille. `%v` n'existe pas en C et remplace la moitié des verbes. Et surtout, une erreur de verbe ne plante pas : `fmt.Printf("%d", "oups")` affiche `%!d(string=oups)`, et `go vet` la signale avant même de lancer (`fmt.Printf format %d has arg "oups" of wrong type string`). Lance `go vet` à chaque fois.

**Venant de Python :** pas de f-string. `fmt.Sprintf("%s:%d", hote, port)` est l'équivalent de `f"{hote}:{port}"`. C'est un peu plus long, et on s'y fait.

### 2.8 Pour le labo

Le [labo 02](../labs/02-convertisseur/README.md) est un convertisseur : températures, octets en taille lisible (`1536` donne `1.5 Ko`), durées façon `1h30m`. Il te fait manipuler `int64`, `float64`, les conversions explicites, `strconv`, et `Printf` avec des colonnes alignées. Et c'est le premier labo avec des tests : `go test .` te dira si ta version est juste.

### À retenir

- `var nom type` ou `nom := valeur` ; le type est fixé pour toujours, et toute variable a une valeur zéro (`0`, `false`, `""`, `nil`), jamais du n'importe quoi.
- Variable ou import inutilisé = erreur de compilation. `_ = x` pour bricoler, jamais pour commiter.
- Types de base : `int` (64 bits), `int64`, `uint8`, `float64`, `bool`, `string`, `byte` (= `uint8`), `rune` (= `int32`). Les entiers débordent en silence.
- Aucune conversion implicite : `float64(n)`, `int(x)`, `strconv.Atoi("42")`, `strconv.Itoa(42)`. Jamais `string(42)`.
- `const` est vraiment constant et non typé ; `iota` fabrique les énumérations.
- Une `string` est une suite d'octets UTF-8 immuable : `len` compte les octets, `s[i]` est un `byte`, `range` décode en `rune`.
- Pas de méthodes sur les chaînes : `strings.Split`, `strings.TrimSpace`, `strings.Join`, `strings.Contains`… et `strconv` pour les nombres.
- `fmt.Printf` avec `%v %+v %T %d %s %q %.2f`, `%-10s` pour aligner ; `go vet` vérifie les verbes.

---

← [1. Installer, outiller, lancer](01-installer-et-outiller.md) · [Sommaire](../README.md) · [3. Contrôle : if, for, switch, defer](03-controle.md) →
