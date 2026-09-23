# 7. Interfaces et composition

*Go de zéro à la prod : chapitre 7 sur 16.* ← [6. Structs, méthodes, pointeurs](06-structs-methodes-pointeurs.md) · [Sommaire](../README.md) · [8. Erreurs pour de vrai : wrapping, Is, As, panic](08-erreurs-avancees.md) →

Le chapitre 6 s'est arrêté sur une frustration : un `Postgres` qui embarque un `Hote` n'est pas un `Hote`, et une fonction qui veut « n'importe quoi qui a une adresse » ne sait pas comment le dire. La réponse de Go tient en un mot, **interface**, et en une règle qui n'existe ni en C, ni en Java, ni tout à fait en Python : un type satisfait une interface **sans le déclarer**. Il suffit qu'il ait les bonnes méthodes.

C'est le mécanisme le plus important du langage. Toute la bibliothèque standard est bâtie dessus : `io.Reader`, `io.Writer`, `error`, `fmt.Stringer`, `http.Handler`. Quand tu comprends pourquoi une fonction qui écrit dans un `io.Writer` marche aussi bien avec le terminal, un fichier, un tampon mémoire, une connexion réseau ou un flux compressé, tu as compris Go.

### 7.1 Une interface est un ensemble de méthodes

Une interface déclare des méthodes, sans les implémenter. Tout type qui possède ces méthodes, avec exactement ces signatures, **est** de ce type interface. Pas de `implements`, pas d'héritage, pas d'enregistrement : le compilateur vérifie tout seul.

```go
package main

import (
	"fmt"
	"math"
)

type Forme interface {
	Aire() float64
}

type Rectangle struct{ L, H float64 }
type Cercle struct{ R float64 }

func (r Rectangle) Aire() float64 { return r.L * r.H }
func (c Cercle) Aire() float64    { return math.Pi * c.R * c.R }

func Total(formes []Forme) float64 {
	total := 0.0
	for _, f := range formes {
		total += f.Aire()
	}
	return total
}

func main() {
	formes := []Forme{Rectangle{3, 4}, Cercle{1}}
	for _, f := range formes {
		fmt.Printf("%T %.2f\n", f, f.Aire())
	}
	fmt.Printf("total : %.2f\n", Total(formes))
}
```

```
$ go run .
main.Rectangle 12.00
main.Cercle 3.14
total : 15.14
```

- `type Forme interface { Aire() float64 }` : « une Forme, c'est ce qui sait calculer son aire ». Rien de plus.
- `Rectangle` et `Cercle` ne mentionnent jamais `Forme`. Ils ont une méthode `Aire() float64`, donc ils sont des `Forme`. C'est la **satisfaction implicite**.
- `[]Forme{Rectangle{3, 4}, Cercle{1}}` : une slice de l'interface, qui contient des valeurs de types concrets différents. `%T` montre le type concret rangé dedans.
- `Total` ne sait rien des rectangles ni des cercles. Ajoute demain un `Triangle` avec une méthode `Aire`, `Total` marchera sans être modifié ni recompilé.

Une valeur d'interface contient deux choses : le **type concret** et la **valeur** (ou un pointeur vers elle). C'est ce couple que `%T` et `%v` affichent, et c'est ce qui permet le piège de la section 7.5.

**Venant de Python :** c'est le duck typing du chapitre 9 du cours Python, rendu officiel et vérifié à la compilation. `Protocol` de `typing` est exactement une interface Go : un ensemble de méthodes, satisfaction structurelle. La différence : en Python, tu découvres à l'exécution qu'il manque une méthode ; en Go, ça ne compile pas.

**Venant du C :** l'équivalent le plus proche est une struct de pointeurs de fonctions (`struct ops { double (*aire)(void *); }`) que chaque « type » remplit. Go génère cette table pour toi, la remplit automatiquement, et vérifie les signatures. C'est ce qu'il y a derrière : une valeur d'interface, c'est un pointeur vers une table de méthodes plus un pointeur vers les données.

Ce que le compilateur dit quand il manque une méthode, ou qu'elle a la mauvaise signature, est très lisible. Ici, `Aire` a un receveur pointeur et on passe une valeur :

```go
type Triangle struct{ B, H float64 }

func (t *Triangle) Aire() float64 { return t.B * t.H / 2 }

var f Forme = Triangle{3, 4}
```

```
$ go run .
./main.go:12:16: cannot use Triangle{…} (value of struct type Triangle) as Forme value in variable declaration: Triangle does not implement Forme (method Aire has pointer receiver)
```

**Piège :** l'ensemble des méthodes d'un type `T` ne contient que celles à receveur valeur ; celui de `*T` contient les deux. Donc si une méthode de l'interface a un receveur pointeur, seul `*Triangle` satisfait `Forme`, et il faut écrire `&Triangle{3, 4}`. C'est une raison de plus pour la règle « toutes les méthodes d'un type ont le même genre de receveur » du chapitre 6. Et pour vérifier à la compilation qu'un type satisfait bien une interface sans attendre d'en avoir besoin, la bibliothèque standard utilise une ligne idiomatique, à mettre juste sous le type :

```go
var _ Forme = (*Triangle)(nil)
```

Elle déclare une variable jetable de type `Forme` initialisée avec un pointeur nul de `*Triangle`. Si `*Triangle` cesse de satisfaire `Forme`, la compilation échoue à cette ligne, avec le message ci-dessus.

### 7.2 Les petites interfaces de la bibliothèque standard

La règle de conception de Go : **plus une interface est petite, plus elle est utile**. Une méthode, c'est l'idéal ; deux ou trois, c'est bien ; dix, c'est un signe que quelque chose ne va pas. Les quatre interfaces que tu vas croiser partout ont une seule méthode chacune.

**`error`** : tu l'utilises depuis le chapitre 5 sans savoir que c'est une interface. La voici en entier :

```go
type error interface {
	Error() string
}
```

N'importe quel type avec une méthode `Error() string` est une erreur. `errors.New` renvoie un pointeur vers une petite struct privée qui a cette méthode. Le chapitre 8 en tire tout un système.

**`fmt.Stringer`** : `String() string`. Si ton type l'a, `fmt.Println`, `%v` et `%s` l'appellent au lieu d'afficher les champs bruts. C'est le `__str__` de Python.

```go
package main

import (
	"fmt"
	"time"
)

type Sonde struct {
	Cible   string
	Latence time.Duration
	OK      bool
}

func (s Sonde) String() string {
	etat := "KO"
	if s.OK {
		etat = "OK"
	}
	return fmt.Sprintf("%-16s %s en %v", s.Cible, etat, s.Latence)
}

func main() {
	s := Sonde{Cible: "10.0.0.5:5432", Latence: 3 * time.Millisecond, OK: true}
	fmt.Println(s)
	fmt.Printf("%v | %s\n", s, s)
	fmt.Printf("%+v\n", s)
}
```

```
$ go run .
10.0.0.5:5432    OK en 3ms
10.0.0.5:5432    OK en 3ms | 10.0.0.5:5432    OK en 3ms
10.0.0.5:5432    OK en 3ms
```

Remarque que `time.Duration` s'affiche `3ms` : c'est aussi un `Stringer`. Que `%+v` appelle `String()` lui aussi (pour voir les champs bruts d'un type qui a un `String`, utilise `%#v`). Et qu'une slice `[]Sonde` s'affiche en appelant `String()` sur chaque élément. **Piège :** dans `String()`, n'écris jamais `fmt.Sprintf("%v", s)` avec `s` lui-même, c'est une récursion infinie (le formateur rappelle `String`). Formate les champs un par un.

**`io.Reader`** et **`io.Writer`** : les deux interfaces les plus importantes de Go.

```go
type Reader interface{ Read(p []byte) (n int, err error) }
type Writer interface{ Write(p []byte) (n int, err error) }
```

Un `Reader` remplit le tampon qu'on lui donne et dit combien d'octets il a mis ; un `Writer` prend un tampon et dit combien il en a écrit. Fichiers, connexions réseau, `os.Stdin`, `os.Stdout`, tampons mémoire, flux compressés, corps de requête HTTP, tous sont l'un ou l'autre, ou les deux. Les deux sections suivantes montrent ce que ça permet.

### 7.3 `io.Writer` : une fonction, cinq destinations

Écris une fonction qui prend un `io.Writer`, et elle marche avec tout ce qui sait écrire.

```go
package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

func ecrireRapport(w io.Writer, hotes []string) error {
	for i, h := range hotes {
		if _, err := fmt.Fprintf(w, "%d. %s\n", i+1, h); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	hotes := []string{"db-01", "cache-01", "web-01"}

	ecrireRapport(os.Stdout, hotes)

	var buf bytes.Buffer
	ecrireRapport(&buf, hotes)
	fmt.Printf("buffer : %d octets, première ligne %q\n", buf.Len(), buf.String()[:8])

	f, err := os.Create("rapport.txt")
	if err != nil {
		fmt.Println("erreur :", err)
		return
	}
	defer f.Close()
	ecrireRapport(f, hotes)

	gz, err := os.Create("rapport.txt.gz")
	if err != nil {
		fmt.Println("erreur :", err)
		return
	}
	defer gz.Close()
	zw := gzip.NewWriter(gz)
	ecrireRapport(zw, hotes)
	zw.Close()
}
```

```
$ go run .
1. db-01
2. cache-01
3. web-01
buffer : 31 octets, première ligne "1. db-01"
$ ls -la rapport.txt*
-rw-r--r--  1 stranix  staff  31 rapport.txt
-rw-r--r--  1 stranix  staff  56 rapport.txt.gz
$ gunzip -c rapport.txt.gz
1. db-01
2. cache-01
3. web-01
```

La même fonction `ecrireRapport`, sans une ligne de changement, a écrit sur le terminal, dans un tampon mémoire, dans un fichier, et dans un fichier compressé. Quelques détails :

- `fmt.Fprintf(w, ...)` est `Printf` avec une destination : le `F` veut dire « dans ce Writer ». `Printf` n'est que `Fprintf(os.Stdout, ...)`.
- `bytes.Buffer` a un receveur pointeur pour `Write`, donc c'est `&buf` qui est un `io.Writer`, pas `buf`. La valeur zéro d'un `Buffer` est utilisable, d'où le simple `var buf bytes.Buffer`.
- `gzip.NewWriter(gz)` prend un `io.Writer` et en renvoie un autre : c'est un **décorateur**. Le gzip compresse ce qu'on lui écrit et l'envoie au fichier. Tu peux empiler : un `bufio.Writer` sur un `gzip.Writer` sur un `tls.Conn`. Le `zw.Close()` explicite est obligatoire pour vider le dernier bloc compressé.
- Dans la même famille : `os.Stderr`, `io.MultiWriter(a, b)` qui duplique vers plusieurs Writers (le `tee` du shell), `io.Discard` qui avale tout (`/dev/null`), et toute connexion réseau.

C'est exactement pour ça que tester du code Go est facile : en prod la fonction reçoit un fichier ou une connexion, dans le test elle reçoit un `bytes.Buffer` dont on lit le contenu. Le labo 07 le fait.

### 7.4 `io.Reader` : lire de n'importe où

Symétriquement, une fonction qui prend un `io.Reader` lit depuis un fichier, un flux réseau, une chaîne, ou le corps d'une requête HTTP.

```go
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func compterLignes(r io.Reader) (int, error) {
	n := 0
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		n++
	}
	return n, sc.Err()
}

func main() {
	n, err := compterLignes(strings.NewReader("a\nb\nc\n"))
	fmt.Println(n, err)

	nb, err := io.Copy(os.Stdout, io.LimitReader(strings.NewReader("ligne copiée telle quelle\n"), 12))
	fmt.Println()
	fmt.Println(nb, err)
}
```

```
$ go run .
3 <nil>
ligne copié
12 <nil>
```

- `strings.NewReader("...")` transforme une chaîne en `io.Reader` : l'outil numéro un pour tester une fonction qui lit, sans fichier. Un `*os.File` ouvert avec `os.Open` est aussi un Reader : la même fonction compte les lignes de `/var/log/system.log` sans changer.
- `bufio.NewScanner(r)` découpe n'importe quel Reader en lignes (ou en mots avec `sc.Split(bufio.ScanWords)`). C'est ce qu'on utilise pour lire un fichier de log ligne à ligne sans le charger en mémoire.
- `io.Copy(dst, src)` branche un Reader sur un Writer et laisse couler. `io.LimitReader` coupe après N octets. Note que 12 octets de la chaîne s'arrêtent après le `é`, qui en prend deux : les Readers voient des octets, pas des caractères (chapitre 2).

Le paquet `io` compose ces deux interfaces en d'autres, plus larges. Voici la définition exacte de `io.ReadWriter` :

```go
type ReadWriter interface {
	Reader
	Writer
}
```

Embarquer une interface dans une autre en fait l'union des méthodes : un `ReadWriter` est tout ce qui a `Read` **et** `Write`. Un `*os.File` en est un, une `net.Conn` aussi. Il existe `io.ReadCloser`, `io.WriteCloser`, `io.ReadWriteCloser`, tous bâtis pareil. Et tu composes les tiennes : `type Stockage interface { Lecteur; Ecrivain }`.

### 7.5 `any`, assertion de type, type switch

L'interface vide, `interface{}`, n'exige aucune méthode : tout type la satisfait. Depuis Go 1.18, elle a un alias, `any`. Tu l'as croisée dans `fmt.Println(a ...any)`. C'est la porte de sortie du typage statique, à utiliser avec parcimonie : une fois une valeur dans un `any`, il faut la ressortir pour en faire quelque chose.

```go
package main

import "fmt"

func decrire(v any) string {
	switch x := v.(type) {
	case nil:
		return "rien"
	case int:
		return fmt.Sprintf("entier %d", x)
	case string:
		return fmt.Sprintf("chaîne de %d octets", len(x))
	case error:
		return "erreur : " + x.Error()
	default:
		return fmt.Sprintf("type inconnu %T", x)
	}
}

func main() {
	var v any = 5432
	n, ok := v.(int)
	fmt.Println(n, ok)
	s, ok := v.(string)
	fmt.Printf("%q %v\n", s, ok)

	for _, x := range []any{nil, 22, "db-01", 3.14, fmt.Errorf("boum")} {
		fmt.Println(decrire(x))
	}
	fmt.Println(v.(string))
}
```

```
$ go run .
5432 true
"" false
rien
entier 22
chaîne de 5 octets
type inconnu float64
erreur : boum
panic: interface conversion: interface {} is int, not string

goroutine 1 [running]:
main.main()
	/Users/stranix/ex/main.go:32 +0x26c
exit status 2
```

- `n, ok := v.(int)` est une **assertion de type** : « je crois que `v` contient un `int` ». Avec deux valeurs de retour, `ok` dit si c'est vrai, et `n` vaut la valeur zéro sinon. C'est la forme sûre, à utiliser toujours.
- `v.(string)` sans le `ok` est la forme qui **panique** si le type est faux (dernière ligne). Ne l'utilise que quand le type est garanti par construction.
- `switch x := v.(type)` est le **type switch** : il teste les types les uns après les autres, et dans chaque branche `x` a le type de la branche (un `int` dans `case int`, une `error` dans `case error`). Un `case` peut aussi tester une interface : `case error` attrape tout ce qui a `Error()`. `case nil` attrape l'interface vide.

**Venant de Python :** `v.(int)` est `isinstance(v, int)` plus la conversion, et le type switch est une chaîne de `isinstance`. Mais en Python tout objet est déjà « any » ; en Go, tu n'y as recours que quand tu as volontairement effacé le type. Si tu écris `any` dans une signature de fonction, demande-toi si une interface à une méthode ne dirait pas mieux ce que tu attends.

### 7.6 Le piège du nil typé

C'est le piège le plus célèbre de Go, et il vient directement du fait qu'une interface contient un type et une valeur. Une interface est `nil` seulement si **les deux** sont nil. Un pointeur nil rangé dans une interface donne une interface non nil, avec un type et une valeur nulle.

```go
package main

import "fmt"

type ErreurConfig struct{ Champ string }

func (e *ErreurConfig) Error() string { return "champ invalide : " + e.Champ }

func valider(port int) *ErreurConfig {
	if port <= 0 {
		return &ErreurConfig{"port"}
	}
	return nil
}

func validerBien(port int) error {
	if port <= 0 {
		return &ErreurConfig{"port"}
	}
	return nil
}

func main() {
	var err error = valider(5432)
	fmt.Println(err == nil)
	fmt.Printf("%T %v\n", err, err)

	err = validerBien(5432)
	fmt.Println(err == nil)
}
```

```
$ go run .
false
*main.ErreurConfig <nil>
true
```

`valider` renvoie un `*ErreurConfig` nil. Rangé dans une variable `error`, il devient une interface dont le type est `*main.ErreurConfig` et la valeur nil : `err == nil` est **faux**, et le code appelant croit qu'il y a une erreur. `validerBien` renvoie le type `error` directement ; son `return nil` est un vrai nil d'interface.

**Piège :** la règle qui en découle : **une fonction qui renvoie une erreur a toujours le type de retour `error`, jamais un type concret** (`*ErreurConfig`, `*MonErreur`). Le chapitre 8 montre comment récupérer le type concret proprement, avec `errors.As`. Le même piège existe avec n'importe quelle interface, mais c'est avec `error` qu'on se fait avoir.

### 7.7 Accepter des interfaces, renvoyer des structs

C'est la règle de conception qui découle de tout le chapitre. Une fonction **prend en paramètre** l'interface la plus petite qui lui suffit (`io.Reader` plutôt que `*os.File`, `Forme` plutôt que `Rectangle`) : elle accepte ainsi le plus de types possibles, et elle est testable avec un faux. Une fonction **renvoie** un type concret (`*Pool`, `*bytes.Buffer`) : l'appelant a accès à toutes ses méthodes, et pourra lui-même le ranger dans l'interface de son choix.

Corollaire : **définis les interfaces du côté de celui qui les utilise**, pas du côté de celui qui les implémente. En Java, la bibliothèque définit `Stockage` et tes classes l'implémentent. En Go, c'est ta fonction `Sauvegarder(s Stockage)` qui déclare la petite interface dont elle a besoin, et n'importe quel type venu d'ailleurs, même d'une bibliothèque qui n'a jamais entendu parler de toi, la satisfait s'il a la méthode. C'est ce qui rend possible d'écrire un `io.Writer` en trois lignes pour brancher une bibliothèque tierce sur ton système de logs.

### 7.8 `sort.Interface` contre `slices.SortFunc`

Un dernier exemple qui montre à la fois une interface à trois méthodes et pourquoi on ne l'utilise plus. `sort.Sort` (2012) demande un type qui satisfait `sort.Interface` : `Len() int`, `Less(i, j int) bool`, `Swap(i, j int)`. On déclare un type nommé sur la slice, on lui attache les trois méthodes, on convertit et on trie :

```go
type ParCharge []Hote

func (p ParCharge) Len() int           { return len(p) }
func (p ParCharge) Less(i, j int) bool { return p[i].Charge < p[j].Charge }
func (p ParCharge) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

sort.Sort(ParCharge(hotes))
```

Ça marche, tu le verras dans du code ancien, et c'est un bon exemple d'interface : `sort` ne sait rien de `Hote`, il ne connaît que trois opérations. Mais depuis Go 1.21 et les génériques, `slices.SortFunc` fait la même chose avec une closure de comparaison qui renvoie négatif, zéro ou positif, et `cmp.Compare` l'écrit pour toi :

```go
hotes := []Hote{{"web-01", 0.7}, {"db-01", 0.2}, {"cache-01", 0.5}}
slices.SortFunc(hotes, func(a, b Hote) int {
	return cmp.Compare(a.Charge, b.Charge)
})
fmt.Println(hotes)
```

```
$ go run .
[{db-01 0.2} {cache-01 0.5} {web-01 0.7}]
```

Trois lignes au lieu de six, pas de type intermédiaire, et le compilateur spécialise le code pour `Hote` (donc c'est plus rapide). Pour du code neuf, `slices.SortFunc`, et `slices.Sort` tout court pour les types ordonnés (`[]int`, `[]string`). L'interface reste le bon outil quand le comportement est vraiment ouvert (des formes, des stockages, des destinations d'écriture) ; quand il s'agit juste de passer une fonction, passe une fonction.

### 7.9 Pour le labo

Le [labo 07](../labs/07-formes-et-io/README.md) a deux parties. La première : une interface `Forme` avec `Aire` et `Perimetre`, trois types qui la satisfont, un `Total`, et `String()` pour chacun. La seconde, celle qui compte : `Compter(r io.Reader)` qui compte lignes, mots et octets comme `wc`, testée avec `strings.NewReader`, et `EcrireRapport(w io.Writer, ...)` testée avec un `bytes.Buffer`. Tu écriras du code qui marche sur un fichier de 10 Go et se teste avec une chaîne de vingt caractères.

### À retenir

- Une interface est un ensemble de méthodes ; tout type qui les a la satisfait, sans le déclarer. Le compilateur vérifie, et `var _ I = (*T)(nil)` le force.
- Petites interfaces : `error`, `fmt.Stringer`, `io.Reader`, `io.Writer` ont une seule méthode, et c'est ce qui les rend universelles.
- Une fonction qui prend un `io.Writer` écrit sur le terminal, un fichier, un tampon, un gzip, un réseau, sans changer ; c'est aussi ce qui la rend testable avec `bytes.Buffer`.
- Composer des interfaces = embarquer (`io.ReadWriter`), comme pour les structs.
- `any` accepte tout ; on en ressort avec `v.(T)` (avec `ok`) ou un `switch v.(type)`. Si tu écris `any` dans une signature, demande-toi si une interface à une méthode ne serait pas plus juste.
- Une interface est `nil` seulement si son type et sa valeur sont nil : un pointeur nil dedans n'est pas nil. Renvoie toujours `error`, jamais `*MonErreur`.
- Accepte des interfaces, renvoie des structs, et définis l'interface du côté de celui qui l'utilise.
- `slices.SortFunc` avec `cmp.Compare` pour trier ; `sort.Interface` existe encore, tu le liras, tu ne l'écriras plus.

---

← [6. Structs, méthodes, pointeurs](06-structs-methodes-pointeurs.md) · [Sommaire](../README.md) · [8. Erreurs pour de vrai : wrapping, Is, As, panic](08-erreurs-avancees.md) →
