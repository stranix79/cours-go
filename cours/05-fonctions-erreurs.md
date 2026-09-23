# 5. Fonctions, erreurs et closures

*Go de zéro à la prod : chapitre 5 sur 16.* ← [4. Slices, maps, range : les collections sous le capot](04-slices-maps.md) · [Sommaire](../README.md) · [6. Structs, méthodes, pointeurs](06-structs-methodes-pointeurs.md) →

Tu sais ce qu'est une fonction. Ce chapitre est là pour ce que les fonctions Go font de différent : elles renvoient plusieurs valeurs, ce qui est la base de toute la gestion d'erreur du langage ; elles sont des valeurs qu'on passe et qu'on renvoie ; et elles capturent leur environnement (closures). Il est aussi là pour ce qu'elles ne font pas : pas de surcharge, pas d'arguments par défaut, pas d'exceptions. La moitié du chapitre parle des erreurs, parce que `if err != nil` est la phrase la plus écrite en Go, et qu'il faut comprendre pourquoi avant de l'accepter.

### 5.1 Déclarer, et renvoyer plusieurs valeurs

```go
func diviser(a, b int) (int, int) {
	return a / b, a % b
}
```

`func`, le nom, les paramètres avec leur type **après** le nom (et un seul type pour `a, b int` quand ils sont identiques), puis le ou les types de retour. Une fonction sans retour n'a rien après les parenthèses (pas de `void`). Une fonction qui renvoie plusieurs valeurs les liste entre parenthèses, et l'appelant les reçoit avec une affectation multiple :

```go
q, r := diviser(17, 5)   // 3 2
q, _ = diviser(9, 3)     // on jette le reste avec le blanc
```

**Il faut récupérer toutes les valeurs, ou aucune.** `v := diviser(9, 3)` ne compile pas (`assignment mismatch: 1 variable but diviser returns 2 values`). `diviser(9, 3)` seul, en jetant tout, compile : c'est ainsi qu'on ignore parfois une erreur, et c'est ce que le linter `errcheck` (chapitre 15) est là pour attraper.

**Venant du C :** pas de prototype, pas de `.h`, l'ordre des déclarations dans le fichier n'a pas d'importance. Les retours multiples remplacent les paramètres de sortie par pointeur (`int f(int a, int *reste)`). Et les arguments sont passés **par valeur**, toujours : une fonction qui reçoit un `int`, une struct ou un tableau en reçoit une copie. Un slice ou une map sont passés par valeur aussi, mais comme ce sont des structures qui pointent vers des données partagées (chapitre 4), les modifications des éléments se voient chez l'appelant.

**Venant de Python :** `return a, b` renvoie un tuple en Python ; en Go, ce sont vraiment deux valeurs, il n'y a pas d'objet intermédiaire, et le nombre de valeurs fait partie de la signature. Les types sont obligatoires et vérifiés, ce n'est pas une annotation.

### 5.2 Retours nommés

Les valeurs de retour peuvent avoir un nom. Elles sont alors déclarées comme des variables locales, initialisées à leur valeur zéro, et un `return` sans argument renvoie leur valeur courante :

```go
func parsePort(s string) (port int, err error) {
	port, err = strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("port %q : %w", s, err)
	}
	if port < 1 || port > 65535 {
		err = fmt.Errorf("port %d hors plage 1-65535", port)
		return
	}
	return
}
```

```
$ go run .
5432 <nil>
0 port "abc" : strconv.Atoi: parsing "abc": invalid syntax
70000 port 70000 hors plage 1-65535
```

Deux usages légitimes : documenter ce que renvoie une fonction quand deux retours ont le même type (`(min, max int)` se lit mieux que `(int, int)`), et permettre à un `defer` de modifier la valeur renvoyée (voir 5.7). En dehors de ça, le `return` nu rend le code plus difficile à suivre dans une fonction longue ; préfère `return port, nil` explicite.

### 5.3 Fonctions variadiques

Un dernier paramètre de la forme `nom ...T` accepte zéro ou plusieurs arguments, reçus sous forme de slice `[]T` :

```go
func minMax(premier int, autres ...int) (min, max int) {
	min, max = premier, premier
	for _, v := range autres {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return
}

minMax(3, 9, 1, 7)      // 1 9
minMax(42)              // 42 42 : autres est un slice vide
ports := []int{80, 443, 22}
minMax(8080, ports...)  // 22 8080 : les trois points étalent le slice
```

Le paramètre obligatoire `premier` garantit qu'on ne peut pas appeler `minMax()` sans rien. `fmt.Println(a ...any)` est variadique : c'est pour ça qu'elle accepte n'importe quel nombre d'arguments. Les trois points à l'appel (`ports...`) sont le `*args` de Python côté appel.

### 5.4 Une fonction est une valeur

Une fonction a un type, `func(string) string` par exemple, et se range dans une variable, une map, un paramètre :

```go
func ping(h string) string    { return "ping " + h }
func restart(h string) string { return "systemctl restart sur " + h }

func appliquer(hotes []string, f func(string) string) []string {
	res := make([]string, 0, len(hotes))
	for _, h := range hotes {
		res = append(res, f(h))
	}
	return res
}

actions := map[string]func(string) string{"ping": ping, "restart": restart}
fmt.Println(actions["restart"]("web01"))
fmt.Println(appliquer([]string{"web01", "db01"}, strings.ToUpper))
fmt.Println(appliquer([]string{"web01"}, func(h string) string {
	return h + ".stranix.net"
}))
```

```
$ go run .
systemctl restart sur web01
[WEB01 DB01]
[web01.stranix.net]
```

La map `actions` est une table de dispatch, comme un tableau de pointeurs de fonctions en C. `strings.ToUpper` est passée telle quelle, sans parenthèses. Et la dernière ligne montre une **fonction anonyme** (le `lambda` de Python, mais avec un corps complet, pas une seule expression) déclarée à l'endroit où on en a besoin.

**Venant du C :** c'est le pointeur de fonction, avec un type lisible (`func(string) string` plutôt que `char *(*f)(char *)`), et sans `void *user_data` à côté, parce que la fonction anonyme peut capturer ce dont elle a besoin. C'est la section suivante.

### 5.5 Closures

Une fonction anonyme qui utilise une variable de la fonction qui l'entoure **garde cette variable vivante**, même après que la fonction extérieure a rendu la main. C'est une *closure* (fermeture) : la fonction plus l'environnement où elle est née.

```go
func nouveauCompteur() func() int {
	n := 0
	return func() int {
		n++
		return n
	}
}

c := nouveauCompteur()
fmt.Println(c(), c(), c())   // 1 2 3
d := nouveauCompteur()
fmt.Println(d())             // 1 : son propre n
```

Chaque appel de `nouveauCompteur` crée un nouveau `n`, et la fonction renvoyée est la seule à y avoir accès. C'est de l'état privé sans struct : un compteur, un cache, un générateur d'identifiants. Le compilateur s'occupe de déplacer `n` sur le tas pour qu'il survive (chapitre 10 explique l'*escape analysis*).

**Piège (historique) :** la variable de boucle. Jusqu'à Go 1.21, `for i := 0; i < 3; i++` créait **une seule** variable `i` pour toute la boucle, et trois closures qui la capturaient voyaient toutes sa valeur finale :

```go
var fonctions []func()
for i := 0; i < 3; i++ {
	fonctions = append(fonctions, func() { fmt.Print(i, " ") })
}
for _, f := range fonctions {
	f()
}
```

```
$ go run .        # go.mod avec go 1.21 ou moins
3 3 3 
$ go run .        # go.mod avec go 1.22 ou plus
0 1 2 
```

Depuis Go 1.22, chaque tour de boucle a sa propre variable, et le résultat est celui qu'on attend. Le comportement dépend de la ligne `go` dans `go.mod`, pas du compilateur installé, ce qui est la raison pour laquelle les labos de ce cours déclarent `go 1.25`. Tu croiseras dans du vieux code l'idiome de contournement `i := i` en début de corps de boucle : il est devenu inutile.

### 5.6 Les erreurs : une valeur de retour, pas une exception

Voici le cœur du chapitre. Go n'a pas d'exceptions. Une fonction qui peut échouer **renvoie une erreur en dernière valeur**, de type `error`, et l'appelant la teste. Par convention, le résultat est le premier retour, l'erreur le dernier, et `err == nil` veut dire « tout va bien » :

```go
n, err := strconv.Atoi(s)
if err != nil {
	return err        // ou : traiter, journaliser, réessayer
}
// ici, n est valide
```

`error` est un type de la bibliothèque standard (une interface, chapitre 7 ; pour l'instant, retiens qu'une valeur `error` a une méthode `Error() string` qui donne son message). Sa valeur zéro est `nil`. On en crée avec deux fonctions :

```go
var ErrVide = errors.New("liste vide")        // une erreur fixe, déclarée au niveau du paquet

func premier(hotes []string) (string, error) {
	if len(hotes) == 0 {
		return "", ErrVide
	}
	return hotes[0], nil
}

func lireConfig(chemin string) (map[string]string, error) {
	contenu, err := os.ReadFile(chemin)
	if err != nil {
		return nil, fmt.Errorf("lecture config : %w", err)   // on enveloppe avec du contexte
	}
	// ...
}
```

```
$ go run .
erreur : liste vide
c'est bien ErrVide
lecture config : open /etc/inexistant.conf: no such file or directory
```

`errors.New` fabrique une erreur avec un message. `fmt.Errorf` fabrique une erreur avec un message formaté, et le verbe `%w` (*wrap*) garde l'erreur d'origine à l'intérieur, ce qui permet à `errors.Is(err, ErrVide)` de la retrouver à travers les couches. Le chapitre 8 détaille `Is`, `As` et les types d'erreur ; ici, l'habitude à prendre est : **chaque niveau ajoute son contexte avec `%w` et remonte**. Le message final se lit comme une pile, de haut en bas : `lecture config : open /etc/x: no such file or directory`.

Sur le style. Oui, tu vas écrire `if err != nil { return err }` cent fois. Ce n'est pas un défaut de conception que les concepteurs auraient oublié de corriger, c'est un choix, avec trois conséquences que tu apprécieras après quelques semaines :

- **chaque point d'échec est visible dans le code**, à l'endroit où il peut arriver. Pas de `try` en haut d'une fonction de 200 lignes qui attrape on ne sait quoi venu d'on ne sait où ;
- **le flot de contrôle est celui qu'on lit** : pas de saut invisible vers un `except` trois fonctions plus haut ;
- **ignorer une erreur est un acte explicite** (`_ = f()` ou `res, _ := f()`), qu'un relecteur ou un linter voit.

Les règles de style : traiter l'erreur *ou* la remonter, jamais les deux (pas de `log` puis `return err`, sinon elle est journalisée à chaque niveau). Messages en minuscules, sans ponctuation finale, parce qu'ils seront enveloppés dans d'autres messages. Retour anticipé : le `if err != nil` renvoie, et le chemin normal continue sans indentation supplémentaire. Le code se lit comme une liste d'étapes, chacune suivie de sa sortie de secours.

**Venant de Python :** c'est le contraire d'*EAFP* (« demande pardon plutôt que permission »). Une erreur Go est un résultat, pas un événement. Il n'y a pas de `finally`, c'est `defer`. Et il n'y a pas d'erreur non attrapée qui remonte jusqu'en haut : si tu ne testes pas `err`, elle est perdue, silencieusement. D'où le linter.

**Venant du C :** c'est le code de retour `-1` plus `errno`, mais typé, transportant un message et un contexte, et impossible à confondre avec un résultat valide puisque c'est une valeur séparée.

### 5.7 `panic` et `recover`, en avant-goût

Il existe un second mécanisme, pour les situations irrécupérables : `panic`. Un index hors bornes, une écriture dans une map `nil`, une division entière par zéro déclenchent un `panic` ; on peut aussi l'appeler soi-même. Il déroule la pile en exécutant les `defer`, et si rien ne l'arrête, le programme s'arrête avec la trace. Un `defer` peut l'attraper avec `recover()` :

```go
func protege() (resultat string) {
	defer func() {
		if r := recover(); r != nil {
			resultat = fmt.Sprint("récupéré : ", r)
		}
	}()
	var m map[string]int
	m["x"] = 1                // panic
	return "jamais atteint"
}
```

```
$ go run .
récupéré : assignment to entry in nil map
```

Le `defer` modifie le retour nommé `resultat`, ce qui est l'usage principal des retours nommés. Retiens la règle d'emploi : **`panic` pour les bugs, `error` pour tout ce qui peut arriver en production** (fichier absent, réseau coupé, saisie invalide). On n'utilise pas `panic` comme un `raise`, et on ne met pas `recover` partout ; le chapitre 8 précise les rares cas où il a sa place (un serveur qui ne doit pas tomber à cause d'une requête).

### 5.8 Pas de surcharge, pas d'arguments par défaut

Deux fonctions ne peuvent pas porter le même nom avec des paramètres différents (`aire redeclared in this block`). Et un paramètre n'a pas de valeur par défaut : tous les arguments sont obligatoires, dans l'ordre, sans nom à l'appel. Go préfère la lisibilité de l'appel à la souplesse de la signature. Trois façons de vivre avec :

- **deux noms** : `Connecter(hote)` et `ConnecterAvecPort(hote, port)`. Simple, fréquent dans la bibliothèque standard (`strings.Replace` / `strings.ReplaceAll`) ;
- **une struct d'options** : la valeur zéro de chaque champ vaut « par défaut », et l'appelant ne renseigne que ce qu'il change :

```go
type Options struct {
	Port    int
	Timeout int
	Verbose bool
}

func connecter(hote string, opts Options) string {
	if opts.Port == 0 {
		opts.Port = 22
	}
	if opts.Timeout == 0 {
		opts.Timeout = 5
	}
	return fmt.Sprintf("%s:%d timeout=%ds verbose=%t", hote, opts.Port, opts.Timeout, opts.Verbose)
}

connecter("web01", Options{})                        // web01:22 timeout=5s verbose=false
connecter("web01", Options{Port: 2222, Verbose: true}) // web01:2222 timeout=5s verbose=true
```

- **les options fonctionnelles** : `NewServeur(adresse, AvecTimeout(10), AvecTLS(cert))`, chaque option étant une fonction qui modifie la configuration. C'est le pattern des grosses bibliothèques (gRPC, zap) ; tu le liras avant de l'écrire, et le chapitre 7 en montre un.

Les structs sont le sujet du chapitre 6 ; celle-ci se lit sans explication, c'est le `struct` du C.

### 5.9 Récursivité

Elle marche comme partout, et les piles des goroutines grandissent à la demande (elles commencent à quelques kilo-octets et peuvent atteindre un giga-octet), donc pas de limite à 1000 niveaux comme en Python :

```go
func factorielle(n int) int {
	if n <= 1 {
		return 1
	}
	return n * factorielle(n-1)
}

factorielle(20)   // 2432902008176640000, le plus grand qui tienne dans un int64
```

Pas d'optimisation des appels terminaux ; pour un parcours d'arborescence de fichiers ou de JSON imbriqué, la récursion est naturelle, pour un million d'itérations, c'est une boucle.

### 5.10 Pour le labo

Le [labo 05](../labs/05-boite-a-outils/README.md) est une boîte à outils : `Slugify`, `ParseKV`, une somme variadique, un compteur en closure, et `Retry`, qui reçoit une fonction et la relance jusqu'à ce qu'elle réussisse. Les tests vérifient chaque fonction, y compris `Retry` avec une fonction qui échoue deux fois puis réussit.

### À retenir

- `func nom(params) (retours)`, types après les noms, plusieurs retours natifs ; il faut tous les récupérer (ou tous les jeter).
- Retours nommés : pour documenter, et pour qu'un `defer` puisse modifier le résultat. Sinon, `return` explicite.
- `nom ...T` reçoit un slice ; `slice...` à l'appel l'étale.
- Une fonction est une valeur : `func(string) string` se range dans une map, se passe en paramètre ; fonction anonyme déclarée sur place.
- Une closure capture les variables de son environnement et les garde vivantes ; depuis Go 1.22, chaque tour de boucle a sa propre variable.
- Une fonction qui peut échouer renvoie `(T, error)` ; `err == nil` = succès ; `errors.New` et `fmt.Errorf("contexte : %w", err)` ; on traite ou on remonte, jamais les deux.
- `panic` pour les bugs, `error` pour la production ; `recover` dans un `defer`, rarement.
- Pas de surcharge ni d'arguments par défaut : deux noms, ou une struct d'options dont la valeur zéro est le défaut.

---

← [4. Slices, maps, range : les collections sous le capot](04-slices-maps.md) · [Sommaire](../README.md) · [6. Structs, méthodes, pointeurs](06-structs-methodes-pointeurs.md) →
