# 6. Structs, méthodes, pointeurs

*Go de zéro à la prod : chapitre 6 sur 16.* ← [5. Fonctions, erreurs et closures](05-fonctions-erreurs.md) · [Sommaire](../README.md) · [7. Interfaces et composition](07-interfaces.md) →

Jusqu'ici tu as manipulé des entiers, des chaînes, des slices et des maps. Un vrai programme manipule des *choses* : un serveur, une sonde, un compte, une connexion. En C tu les décris avec une `struct` et tu écris des fonctions qui prennent un pointeur dessus. En Python tu écris une classe. Go prend la voie du C, la `struct`, et y ajoute juste ce qu'il faut pour que ce soit agréable : des méthodes attachées au type, une notion de constructeur par convention, et la composition à la place de l'héritage.

Ce chapitre est aussi celui des pointeurs. Pas de panique : ce sont les pointeurs du C sans ce qui les rend dangereux. Pas d'arithmétique, pas de `free`, des bornes vérifiées. Ils servent à une seule chose en Go : dire « je veux modifier l'original, pas une copie ». Comprendre ce point règle 80 % des questions que se posent les débutants sur les méthodes.

### 6.1 Déclarer une struct

Une struct est un type composé de champs nommés, chacun avec son type. C'est exactement la `struct` du C, en plus strict et en plus lisible.

```go
package main

import "fmt"

type Serveur struct {
	Nom  string
	IP   string
	Port int
	Prod bool
}

func main() {
	var s1 Serveur
	fmt.Printf("%v\n", s1)
	fmt.Printf("%+v\n", s1)
	s2 := Serveur{Nom: "db-01", IP: "10.0.0.5", Port: 5432}
	fmt.Printf("%+v\n", s2)
	s3 := Serveur{"db-01", "10.0.0.5", 5432, false}
	fmt.Println(s2 == s3)
	s3.Port = 5433
	fmt.Println(s2 == s3, s2.Port, s3.Port)
}
```

```
$ go run .
{  0 false}
{Nom: IP: Port:0 Prod:false}
{Nom:db-01 IP:10.0.0.5 Port:5432 Prod:false}
true
false 5432 5433
```

Ce qu'il faut voir dans cet exemple, ligne par ligne :

- `type Serveur struct { ... }` déclare un nouveau type nommé. Les champs commencent par une majuscule : ils sont exportés (visibles hors du paquet), comme tout identifiant en Go. Un champ `secret string` en minuscule serait privé au paquet.
- `var s1 Serveur` crée une struct à sa **valeur zéro** : chaque champ vaut la valeur zéro de son type (`""`, `0`, `false`, `nil`). Il n'y a pas de mémoire non initialisée en Go. C'est un principe de conception : *la valeur zéro doit être utilisable*. Un `bytes.Buffer` vide, un `sync.Mutex` déverrouillé, une `strings.Builder` vide : tous se déclarent avec `var` et marchent sans initialisation.
- `Serveur{Nom: "db-01", Port: 5432}` est un **littéral nommé** : tu donnes les champs que tu veux, dans l'ordre que tu veux, le reste prend la valeur zéro. C'est la forme à utiliser partout.
- `Serveur{"db-01", "10.0.0.5", 5432, false}` est le littéral positionnel : tous les champs, dans l'ordre de déclaration. Fragile (ajoute un champ, tout casse) : réserve-le aux structs de deux champs évidents comme `Point{1, 2}`.
- `s2 == s3` : deux structs sont comparables avec `==` si tous leurs champs le sont. La comparaison est champ à champ. Une struct qui contient une slice ou une map n'est pas comparable (le compilateur refuse), parce que les slices ne le sont pas.
- `%v` affiche les valeurs, `%+v` ajoute les noms des champs (et `%#v` la syntaxe Go complète, avec le nom du type). Prends l'habitude de `%+v` pour déboguer : c'est le `__repr__` gratuit de Go.

**Venant du C :** même mot, même idée, trois différences. Pas de `typedef struct {...} Serveur;`, le `type` suffit. Les champs sont toujours initialisés. Et `==` marche sur les structs entières, là où le C t'obligeait à un `memcmp` (qui se trompait sur le rembourrage).

**Venant de Python :** une struct est une `@dataclass` sans les méthodes générées : `__init__` est le littéral nommé, `__eq__` est le `==` natif, `__repr__` est `%+v`. Mais les champs sont déclarés une fois pour toutes : impossible d'ajouter `s.couleur = "rouge"` à l'exécution, le compilateur refuse. C'est ce qui rend le code des autres lisible : la liste des champs est dans la déclaration, pas éparpillée dans le code.

### 6.2 Pointeurs : `&`, `*`, `new`, `nil`

Un pointeur est l'adresse d'une valeur en mémoire. `&x` donne l'adresse de `x`, `*p` donne la valeur à l'adresse `p`. Le type d'un pointeur vers un `int` s'écrit `*int`.

```go
package main

import "fmt"

func main() {
	port := 5432
	p := &port
	fmt.Println(port, *p)
	*p = 5433
	fmt.Println(port, *p)
	fmt.Printf("%T\n", p)

	var q *int
	fmt.Println(q == nil)
	q = new(int)
	fmt.Println(*q)
	*q = 22
	fmt.Println(*q)
}
```

```
$ go run .
5432 5432
5433 5433
*int
true
0
22
```

- `p := &port` : `p` est un `*int` qui pointe vers `port`. Modifier `*p` modifie `port`, puisque c'est le même emplacement.
- `var q *int` : un pointeur non initialisé vaut `nil`, le « pointeur vers rien » (le `NULL` du C, le `None` du Python, en un seul mot pour tous les types).
- `new(int)` alloue un `int` à zéro et renvoie son adresse. On l'utilise peu : pour une struct, on écrit `&Serveur{...}`, qui alloue et initialise en une fois.

**Venant du C :** la syntaxe est identique, mais trois choses ont disparu. Pas d'arithmétique (`p++` ne compile pas), donc pas de débordement de tableau par pointeur. Pas de `free` : le ramasse-miettes récupère la mémoire quand plus rien ne pointe dessus. Et prendre l'adresse d'une variable locale et la renvoyer est parfaitement légal : le compilateur voit que la valeur s'échappe et l'alloue sur le tas au lieu de la pile (c'est l'*escape analysis*, chapitre 10). Le bug classique du C, renvoyer un pointeur vers une variable de pile morte, n'existe pas.

**Venant de Python :** Python n'a pas de pointeurs explicites mais tout est référence : passer une liste à une fonction, c'est passer un pointeur. En Go, c'est l'inverse : **tout est copié**, et le pointeur est la façon explicite de dire « pas de copie ». La section suivante est entièrement consacrée à cette différence.

**Piège :** déréférencer un pointeur `nil` fait planter le programme, proprement (message clair, trace de pile), mais planter quand même.

```go
var nilP *Point
fmt.Println(nilP == nil)
fmt.Println(nilP.X)
```

```
true
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x2 addr=0x0 pc=0x100b43660]

goroutine 1 [running]:
main.main()
	/Users/stranix/ex/main.go:18 +0x130
exit status 2
```

C'est l'unique famille de plantage mémoire possible en Go. Quand une fonction renvoie `(*T, error)`, le contrat est : si `err != nil`, ne touche pas au pointeur. Respecte-le, et tu ne verras ce message que dans les cours.

Petit détail de confort : `pp.X` marche sur un pointeur de struct, pas besoin d'écrire `(*pp).X`. Go déréférence tout seul pour l'accès aux champs. Il n'y a pas de `->`.

### 6.3 Tout est passé par valeur

C'est la règle qui explique tout le reste du chapitre : **un argument de fonction est toujours copié**. Un `int`, une `string`, une struct de vingt champs : copiés. Un pointeur aussi est copié, mais la copie d'une adresse pointe vers le même endroit, ce qui donne l'effet « par référence ».

```go
package main

import "fmt"

type Sonde struct {
	Cible  string
	Echecs int
}

func incrementeValeur(s Sonde) {
	s.Echecs++
}

func incrementePointeur(s *Sonde) {
	s.Echecs++
}

func main() {
	s := Sonde{Cible: "10.0.0.5:5432"}
	incrementeValeur(s)
	fmt.Println("après valeur  :", s.Echecs)
	incrementePointeur(&s)
	fmt.Println("après pointeur:", s.Echecs)
}
```

```
$ go run .
après valeur  : 0
après pointeur: 1
```

`incrementeValeur` a travaillé sur sa propre copie de `s`, jetée à la sortie. `incrementePointeur` a reçu l'adresse, et a modifié l'original. Rien de nouveau pour quelqu'un qui vient du C ; c'est le point qui surprend le plus quelqu'un qui vient de Python, où `s.echecs += 1` dans une fonction modifie toujours l'objet appelant.

Et les slices et les maps du chapitre 4 ? Elles sont aussi passées par valeur, mais une slice est un petit en-tête (pointeur vers le tableau, longueur, capacité) : la copie de l'en-tête pointe vers le même tableau. Modifier `s[0]` dans une fonction se voit dehors ; faire `s = append(s, x)` ne se voit pas (l'en-tête copié a changé, pas l'original). Une map est un pointeur déguisé : toujours partagée. Les structs, elles, sont copiées en entier, et c'est là que le pointeur devient nécessaire.

Quand prendre un pointeur, en deux règles :

1. **La fonction doit modifier la valeur** : pointeur, obligatoire.
2. **La struct est grosse** (des dizaines de champs, ou des tableaux dedans) et la fonction est appelée souvent : pointeur, pour éviter la copie. Pour une struct de trois `int`, la copie est plus rapide que l'indirection ; ne prends pas de pointeur par réflexe.

Dans tous les autres cas, passe par valeur : c'est plus simple à lire (l'appelant sait que sa valeur ne bougera pas) et plus sûr en concurrence (chapitre 11 : une copie ne se partage pas).

### 6.4 Méthodes : receveur valeur ou receveur pointeur

Une **méthode** est une fonction attachée à un type. La syntaxe ajoute un paramètre spécial, le **receveur**, entre `func` et le nom :

```go
package main

import "fmt"

type Compteur struct {
	Nom    string
	Valeur int
}

func (c Compteur) Affiche() string {
	return fmt.Sprintf("%s=%d", c.Nom, c.Valeur)
}

func (c Compteur) IncrementeMal() {
	c.Valeur++
}

func (c *Compteur) Incremente() {
	c.Valeur++
}

func main() {
	c := Compteur{Nom: "requetes"}
	c.IncrementeMal()
	fmt.Println(c.Affiche())
	c.Incremente()
	c.Incremente()
	fmt.Println(c.Affiche())
	p := &c
	p.Incremente()
	fmt.Println(p.Affiche())
}
```

```
$ go run .
requetes=0
requetes=2
requetes=3
```

- `func (c Compteur) Affiche()` : receveur **valeur**. La méthode reçoit une copie de `c`. Parfait pour lire.
- `func (c Compteur) IncrementeMal()` : receveur valeur qui essaie de modifier. Elle modifie sa copie, l'original reste à 0. Ce code compile, `go vet` ne dit rien, et c'est **le** bug du débutant Go. Le labo 06 te le fait vivre avec un test qui échoue.
- `func (c *Compteur) Incremente()` : receveur **pointeur**. La méthode reçoit l'adresse, modifie l'original.
- `c.Incremente()` sur une variable `c` (pas un pointeur) marche : Go prend l'adresse tout seul, c'est du sucre pour `(&c).Incremente()`. Et `p.Affiche()` sur un pointeur marche aussi : Go déréférence. Tu n'écris jamais `&` ni `*` pour appeler une méthode.

Le nom du receveur est une ou deux lettres du type (`c` pour `Compteur`, `s` pour `Serveur`), jamais `this` ni `self`. C'est une convention forte : tout le code Go la suit.

**Venant de Python :** `self` est devenu `c`, il est typé, et il est déclaré avant le nom de la méthode plutôt que comme premier paramètre. Mais la différence de fond est ailleurs : en Python, `self` est toujours une référence. En Go, tu choisis à chaque méthode entre une copie et une référence, et le choix est visible dans la signature.

**Venant du C :** `func (c *Compteur) Incremente()` est exactement `void compteur_incremente(Compteur *c)`. Go a juste rangé la fonction avec son type, et te laisse écrire `c.Incremente()`.

La règle pour choisir, et elle tient en trois lignes :

1. La méthode **modifie** le receveur ? Pointeur.
2. La struct est **grosse**, ou contient un `sync.Mutex` (qui ne doit jamais être copié) ? Pointeur.
3. Sinon, valeur. **Et reste cohérent** : si une seule méthode a besoin d'un pointeur, mets toutes les méthodes du type en pointeur. Mélanger les deux est légal mais rend le type pénible à utiliser (section suivante) et à lire.

Dans la pratique, la plupart des types « avec un état » (compte, connexion, cache, serveur) ont toutes leurs méthodes en receveur pointeur, et les petits types « valeur » (point, durée, version, adresse IP) tout en receveur valeur.

**Piège :** on ne peut pas appeler une méthode à receveur pointeur sur une valeur qui n'a pas d'adresse. Le cas concret, c'est l'élément d'une map :

```go
m := map[string]Compteur{"req": {}}
m["req"].Incremente()
```

```
$ go run .
./main.go:11:11: cannot call pointer method Incremente on Compteur
```

Les éléments d'une map ne sont pas adressables (la map peut les déplacer en grandissant). Deux solutions : stocker des pointeurs, `map[string]*Compteur`, ou lire, modifier, réécrire (`c := m["req"]; c.Incremente(); m["req"] = c`). Les éléments d'une slice, eux, sont adressables : `s[0].Incremente()` marche. Le labo 06 utilise une `map[string]*Compte` pour cette raison.

### 6.5 Le « constructeur » `NewX`

Go n'a pas de constructeur. Le littéral `Compteur{Nom: "x"}` suffit tant que la valeur zéro des autres champs est acceptable. Quand elle ne l'est pas (un champ doit être validé, une map doit être créée avec `make`, une ressource doit être ouverte), on écrit une fonction ordinaire nommée `New` suivi du type, qui renvoie un pointeur et une erreur :

```go
package main

import (
	"errors"
	"fmt"
)

type Pool struct {
	nom     string
	max     int
	actives int
}

func NewPool(nom string, max int) (*Pool, error) {
	if max <= 0 {
		return nil, errors.New("max doit être positif")
	}
	return &Pool{nom: nom, max: max}, nil
}

func (p *Pool) Acquerir() bool {
	if p.actives >= p.max {
		return false
	}
	p.actives++
	return true
}

func main() {
	p, err := NewPool("pg", 2)
	if err != nil {
		fmt.Println("erreur :", err)
		return
	}
	fmt.Println(p.Acquerir(), p.Acquerir(), p.Acquerir())
	fmt.Printf("%+v\n", *p)
	_, err = NewPool("vide", 0)
	fmt.Println("erreur :", err)
}
```

```
$ go run .
true true false
{nom:pg max:2 actives:2}
erreur : max doit être positif
```

Trois conventions à retenir, parce que toute la bibliothèque standard les suit (`bytes.NewBuffer`, `strings.NewReader`, `http.NewRequest`, `sql.Open`) :

- Le nom est `NewX` ; s'il n'y a qu'un type dans le paquet, juste `New` (`errors.New`, `list.New`).
- Les champs sont en minuscules, donc privés : personne hors du paquet ne peut fabriquer un `Pool` avec `max: -1`. Le constructeur est le seul chemin, et il valide. C'est l'encapsulation de Go : pas de `private`, juste la casse.
- `&Pool{...}` alloue la struct et renvoie son adresse. Renvoyer un pointeur est naturel quand les méthodes ont un receveur pointeur ; un type « valeur » (une `Version`, une `Duree`) renverra une valeur.

**Venant de Python :** `NewPool` est le `__init__` plus la `@classmethod depuis_ligne` du cours Python, en fonction ordinaire. Sans surcharge, tu en écris autant que tu veux : `NewPool`, `NewPoolDepuisConfig`, `NewPoolDepuisURL`.

### 6.6 Embedding : la composition à la place de l'héritage

Go n'a pas d'héritage. Quand un type doit « contenir » les champs et les méthodes d'un autre, on **embarque** ce type : on le déclare comme champ sans nom.

```go
package main

import "fmt"

type Hote struct {
	Nom string
	IP  string
}

func (h Hote) Adresse() string {
	return h.Nom + " (" + h.IP + ")"
}

type Postgres struct {
	Hote
	Port    int
	Version string
}

type Redis struct {
	Hote
	Port int
}

func main() {
	pg := Postgres{
		Hote:    Hote{Nom: "db-01", IP: "10.0.0.5"},
		Port:    5432,
		Version: "16.4",
	}
	fmt.Println(pg.Nom, pg.IP, pg.Port)
	fmt.Println(pg.Hote.Nom)
	fmt.Println(pg.Adresse())
	fmt.Printf("%+v\n", pg)

	r := Redis{Hote: Hote{Nom: "cache-01", IP: "10.0.0.9"}, Port: 6379}
	fmt.Println(r.Adresse())
}
```

```
$ go run .
db-01 10.0.0.5 5432
db-01
db-01 (10.0.0.5)
{Hote:{Nom:db-01 IP:10.0.0.5} Port:5432 Version:16.4}
cache-01 (10.0.0.9)
```

- `Hote` sans nom de champ dans `Postgres` : c'est l'embedding. Le champ existe quand même, il s'appelle comme le type (`pg.Hote`), et `%+v` le montre bien comme un champ imbriqué.
- **Promotion** : les champs et les méthodes de `Hote` sont accessibles directement sur `pg`. `pg.Nom` est un raccourci pour `pg.Hote.Nom`, `pg.Adresse()` pour `pg.Hote.Adresse()`. Le compilateur réécrit, rien de magique à l'exécution.
- Dans le littéral, le champ embarqué s'initialise par son nom de type : `Hote: Hote{...}`.
- Si `Postgres` déclare sa propre méthode `Adresse()`, elle **masque** celle de `Hote` (on peut toujours appeler `pg.Hote.Adresse()` explicitement). C'est ce qui ressemble le plus à une redéfinition, mais sans polymorphisme : une fonction qui prend un `Hote` ne prend pas un `Postgres`, il faut lui passer `pg.Hote`.

Ce dernier point est la différence avec l'héritage : un `Postgres` **a** un `Hote`, il n'**est** pas un `Hote`. Pour qu'une fonction accepte indifféremment un `Postgres` et un `Redis`, on passera par une interface (chapitre 7), pas par le type embarqué. Go te pousse vers ce que tous les livres de conception conseillent depuis trente ans : composer plutôt qu'hériter. Après une semaine, tu ne regretteras pas `extends`.

**Venant de Python :** c'est `class Postgres(Hote)` pour la promotion des attributs et des méthodes, sans `super()`, sans MRO, et sans `isinstance(pg, Hote)`. L'embedding de plusieurs types (`Hote` et `Metriques` dans la même struct) remplace les mixins, avec une règle simple si deux types embarqués ont un champ du même nom : l'accès direct est ambigu et refusé par le compilateur, il faut qualifier.

On embarque aussi couramment un `sync.Mutex` (chapitre 11) pour que la struct ait des méthodes `Lock` et `Unlock`, ou un `*log.Logger` pour qu'un serveur sache `Printf`. La bibliothèque standard le fait partout ; `bufio.ReadWriter` est un `*Reader` et un `*Writer` embarqués, rien d'autre.

### 6.7 Tags de struct : l'avant-goût JSON

Un champ peut porter une **étiquette** (tag), une chaîne entre accents graves après le type. Le compilateur n'en fait rien, mais les paquets qui lisent la struct par réflexion, `encoding/json` en tête, s'en servent pour savoir comment nommer et traiter chaque champ.

```go
package main

import (
	"encoding/json"
	"fmt"
)

type Cible struct {
	Nom        string   `json:"nom"`
	Adresse    string   `json:"adresse"`
	Port       int      `json:"port"`
	Etiquettes []string `json:"etiquettes,omitempty"`
	secret     string
}

func main() {
	c := Cible{Nom: "db-01", Adresse: "10.0.0.5", Port: 5432, secret: "x"}
	b, err := json.Marshal(c)
	if err != nil {
		fmt.Println("erreur :", err)
		return
	}
	fmt.Println(string(b))

	var d Cible
	err = json.Unmarshal([]byte(`{"nom":"cache-01","adresse":"10.0.0.9","port":6379,"etiquettes":["prod"]}`), &d)
	if err != nil {
		fmt.Println("erreur :", err)
		return
	}
	fmt.Printf("%+v\n", d)
}
```

```
$ go run .
{"nom":"db-01","adresse":"10.0.0.5","port":5432}
{Nom:cache-01 Adresse:10.0.0.9 Port:6379 Etiquettes:[prod] secret:}
```

- `json:"nom"` : dans le JSON, ce champ s'appelle `nom` (sans tag, il s'appellerait `Nom`, avec la majuscule).
- `omitempty` : si le champ est à sa valeur zéro (slice nil, chaîne vide, 0), il n'apparaît pas dans la sortie. C'est pour ça que `etiquettes` manque dans la première ligne.
- `secret` en minuscule n'est pas exporté, donc `encoding/json` ne le voit pas, ni en écriture ni en lecture. C'est une propriété très pratique : un champ privé ne fuit jamais dans une API.
- `json.Unmarshal(données, &d)` prend un **pointeur** : il doit remplir ta struct, donc il lui faut l'adresse. Sans le `&`, ça compile (le paramètre est `any`) mais ne remplit rien. Section 6.3, encore.

Le chapitre 12 revient sur JSON en détail (fichiers, flux, types imbriqués, `time.Time`). Ici, retiens juste que la forme d'une struct Go est aussi la forme d'un document JSON, d'une ligne de base de données, d'un fichier YAML : un tag par format, et le même type sert partout.

### 6.8 Pour le labo

Le [labo 06](../labs/06-compte-bancaire/README.md) te fait écrire un `Compte` bancaire (titulaire, solde en centimes, historique des opérations), son constructeur `NewCompte`, ses méthodes `Deposer`, `Retirer`, `Solde`, `String`, puis une `Banque` qui embarque une map de comptes et fait un `Virement`. Un des tests échoue si tu mets un receveur valeur là où il faut un pointeur : c'est fait exprès, pour que tu voies le bug une fois, dans un test, plutôt qu'en prod.

### À retenir

- Une struct est le `struct` du C avec des champs toujours initialisés, comparable avec `==`, affichable avec `%+v`. Utilise le littéral nommé `T{Champ: valeur}`.
- La valeur zéro d'une struct doit être utilisable : conçois tes types pour que `var x T` marche sans constructeur quand c'est possible.
- Les pointeurs sont ceux du C sans arithmétique ni `free` ; `nil` est le seul cas de plantage, ne déréférence jamais un pointeur renvoyé avec une erreur.
- Tout est passé par valeur. Une struct est copiée en entier ; prends un pointeur pour modifier ou pour éviter de copier une grosse struct.
- Receveur pointeur `func (c *T)` si la méthode modifie ou si la struct est grosse, receveur valeur sinon, et toutes les méthodes d'un type pareil.
- Pas de constructeur : une fonction `NewT(...) (*T, error)` qui valide et renvoie un pointeur, avec des champs privés pour forcer son usage.
- Pas d'héritage : on embarque un type (`struct { Hote; Port int }`), ses champs et méthodes sont promus, mais un `Postgres` n'est pas un `Hote`. Le polymorphisme, c'est le chapitre 7.
- Les tags `` `json:"nom,omitempty"` `` pilotent `encoding/json` ; un champ privé n'est jamais sérialisé ; `Unmarshal` veut un pointeur.

---

← [5. Fonctions, erreurs et closures](05-fonctions-erreurs.md) · [Sommaire](../README.md) · [7. Interfaces et composition](07-interfaces.md) →
