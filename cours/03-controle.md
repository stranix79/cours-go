# 3. Contrôle : if, for, switch, defer

*Go de zéro à la prod : chapitre 3 sur 16.* ← [2. Variables, types, constantes, chaînes](02-variables-types-chaines.md) · [Sommaire](../README.md) · [4. Slices, maps, range : les collections sous le capot](04-slices-maps.md) →

Le contrôle de flux en Go tient en quatre mots-clés : `if`, `for`, `switch`, `defer`. Pas de `while`, pas de `do`, pas de ternaire, un `goto` que personne n'utilise. Si tu viens du C, tu vas reconnaître presque tout ; ce qui change, ce sont trois détails qui rendent le code plus court (l'instruction d'initialisation du `if`, le `switch` sans `break`, le `for` qui fait tout) et un mot-clé qui n'existe ni en C ni en Python : `defer`, qui exécute quelque chose à la sortie de la fonction. C'est lui qui remplace le `finally`, le `goto cleanup` et le `with` de Python.

### 3.1 `if`, et l'instruction d'initialisation

Pas de parenthèses autour de la condition, accolades obligatoires même pour une seule ligne, et la condition doit être un `bool` : `if n` avec un entier ne compile pas, c'est `if n != 0`.

```go
charge := 0.85
if charge > 0.9 {
	fmt.Println("critique")
} else if charge > 0.7 {
	fmt.Println("attention")
} else {
	fmt.Println("ok")
}
```

Le `else` doit être sur la même ligne que l'accolade fermante du `if`. Le mettre à la ligne suivante donne `syntax error: unexpected keyword else, expected }`, à cause des points-virgules automatiques du chapitre 1. Il n'y a qu'un style, c'est celui-là.

La nouveauté : un `if` peut commencer par une **instruction d'initialisation**, séparée de la condition par un point-virgule. La variable déclarée n'existe que dans le `if` et ses `else` :

```go
if port, err := strconv.Atoi("5432"); err != nil {
	fmt.Println("port invalide :", err)
} else if port < 1024 {
	fmt.Println("port privilégié", port)
} else {
	fmt.Println("port applicatif", port)
}
```

```
$ go run .
port applicatif 5432
```

C'est l'idiome Go par excellence, tu le verras des centaines de fois sous la forme `if err := faire(); err != nil { return err }`. Il fait deux choses : il garde la variable au plus près de son usage, et il l'empêche de fuir dans le reste de la fonction. Essaie d'utiliser `port` après le bloc : `undefined: port`.

**Venant du C :** même logique, sans parenthèses, avec accolades obligatoires, et avec ce préambule qui évite de déclarer `int port; char *err;` trois lignes plus haut. La condition est un vrai booléen : pas de `if (ptr)`, pas de `if (n)`.

**Venant de Python :** pas de `elif`, c'est `else if`. Pas de valeurs « fausses » implicites : une chaîne vide, un zéro, un slice vide ne sont pas des conditions. `if len(hotes) == 0`, pas `if not hotes`.

### 3.2 `for` : la seule boucle

Go n'a qu'un mot-clé de boucle, et il prend trois formes selon ce que tu écris entre `for` et l'accolade.

**La forme C**, avec initialisation, condition et post-instruction, sans parenthèses :

```go
for i := 0; i < 3; i++ {
	fmt.Print(i, " ")
}
```

**La forme `while`**, avec une condition seule :

```go
essais := 0
for essais < 3 {
	essais++
}
```

**La boucle infinie**, sans rien, dont on sort par `break` ou `return` :

```go
n := 0
for {
	n++
	if n == 5 {
		break
	}
}
```

Et depuis Go 1.22, `for i := range 3` compte de 0 à 2, ce qui remplace la forme C dans le cas le plus fréquent :

```go
for i := range 3 {
	fmt.Print(i, " ")
}
for _, h := range []string{"web01", "db01"} {
	fmt.Print(h, " ")
}
```

```
$ go run .
0 1 2 
essais : 3
n : 5
0 1 2 
web01 db01 
```

`range` parcourt un entier, un slice, une map, une chaîne, un channel, et donne à chaque tour un index (ou une clé) et une valeur. C'est la boucle que tu écriras le plus, et le chapitre 4 la détaille pour chaque type. Le `_` (l'identifiant blanc) jette l'index quand on ne s'en sert pas, sinon le compilateur se plaint qu'il est inutilisé.

**Venant du C :** `i++` est une instruction, pas une expression : `x := i++` ne compile pas, et `++i` n'existe pas. Tu ne peux pas non plus mettre deux variables dans l'initialisation avec une virgule à la C ; c'est `for i, j := 0, 10; i < j; i, j = i+1, j-1`. Pas de `do ... while` ; quand il te manque, c'est `for { ...; if !cond { break } }`.

**Venant de Python :** `for i := range 3` est ton `for i in range(3)`, `for _, v := range liste` est ton `for v in liste`, et `for i, v := range liste` est ton `enumerate`. La forme `while` de Python est le `for cond {}`.

### 3.3 `switch` : sans `break`, avec ou sans expression

Le `switch` de Go ressemble à celui du C, avec deux différences qui corrigent ses deux défauts historiques. **Un `case` ne tombe pas dans le suivant** : pas besoin de `break`, chaque `case` est un bloc indépendant. Et **un `case` peut lister plusieurs valeurs**, ou être n'importe quelle expression, pas seulement une constante entière.

```go
switch code {
case 200, 201, 204:
	fmt.Println(code, "succès")
case 301, 302:
	fmt.Println(code, "redirection")
case 404:
	fmt.Println(code, "introuvable")
case 500, 502, 503:
	fmt.Println(code, "erreur serveur")
default:
	fmt.Println(code, "inconnu")
}
```

```
$ go run .
200 succès
301 redirection
404 introuvable
503 erreur serveur
999 inconnu
```

Le `switch` peut aussi n'avoir **aucune expression** : chaque `case` est alors une condition booléenne, et le premier vrai gagne. C'est une chaîne de `if / else if` plus lisible :

```go
switch {
case charge > 0.9:
	fmt.Println("critique")
case charge > 0.7:
	fmt.Println("attention")
default:
	fmt.Println("ok")
}
```

Comme `if`, il accepte une instruction d'initialisation : `switch os := runtime.GOOS; os { case "darwin": ... }`. Il marche sur des chaînes, ce que le C ne sait pas faire.

Si tu veux vraiment le comportement du C où un `case` continue dans le suivant, le mot-clé `fallthrough` existe. Il doit être la dernière instruction du `case`, et il saute dans le `case` suivant *sans* tester sa condition :

```go
niveau := 2
switch niveau {
case 3:
	fmt.Println("debug")
	fallthrough
case 2:
	fmt.Println("info")
	fallthrough
case 1:
	fmt.Println("erreur")
}
```

```
$ go run .
info
erreur
```

Tu l'écriras rarement. Quand un `fallthrough` semble nécessaire, c'est souvent qu'une liste de valeurs dans le `case` (`case 2, 3:`) ferait mieux.

**Venant du C :** enlève tous tes `break`, ils sont implicites. Le `switch` accepte des chaînes et des conditions. Un `break` dans un `case` sort du `switch` (pas de la boucle qui l'entoure : voir 3.4 pour ça).

**Venant de Python :** le `match` de Python 3.10 sait déstructurer des tuples et des classes ; le `switch` de Go ne fait que comparer des valeurs ou évaluer des conditions. Le `switch` sans expression est ce qui remplace tes longues chaînes `if / elif`.

### 3.4 `break`, `continue` et les étiquettes

`break` sort de la boucle (ou du `switch`) la plus proche, `continue` passe au tour suivant. Comme en C. Le cas qui coince en C, c'est sortir de deux boucles imbriquées d'un coup : on y met un drapeau, ou un `goto`. Go a les **étiquettes** : un nom suivi de `:` devant la boucle, et `break nom` ou `continue nom` la vise directement.

```go
hotes := []string{"web01", "web02", "db01"}
ports := []int{22, 80, 443}
cible := "db01"
boucle:
	for _, h := range hotes {
		for _, p := range ports {
			if h == cible && p == 80 {
				fmt.Println("trouvé", h, p)
				break boucle
			}
			fmt.Println("test", h, p)
		}
	}
```

```
$ go run .
test web01 22
test web01 80
test web01 443
test web02 22
test web02 80
test web02 443
test db01 22
trouvé db01 80
```

Sans l'étiquette, `break` ne sortirait que de la boucle sur les ports, et la boucle sur les hôtes continuerait. Même chose avec `continue boucle` pour passer directement à l'hôte suivant. C'est utile aussi pour un `break` dans un `switch` à l'intérieur d'un `for` : sans étiquette, il sort du `switch` et la boucle continue, ce qui surprend tout le monde une fois.

**Piège :** le `break` dans un `switch` dans un `for` ne sort pas du `for`. Quand tu veux quitter la boucle depuis un `case`, mets une étiquette sur le `for`.

### 3.5 `defer` : à la sortie de la fonction

`defer` devant un appel de fonction reporte cet appel **à la fin de la fonction courante**, quelle que soit la façon dont elle se termine : `return` normal, `return` anticipé au milieu d'un `if err != nil`, ou même un `panic`. C'est le mécanisme qui garantit qu'un fichier est fermé, un verrou relâché, une connexion rendue.

```go
func traiter(nom string) {
	fmt.Println("ouverture", nom)
	defer fmt.Println("fermeture", nom)
	fmt.Println("traitement", nom)
}
```

```
$ go run .
ouverture app.log
traitement app.log
fermeture app.log
```

L'idiome, que tu écriras à chaque ouverture de ressource, met le `defer` juste après la vérification d'erreur, de sorte qu'on ne puisse pas oublier de fermer, même si dix `return` suivent :

```go
f, err := os.Open(chemin)
if err != nil {
	return err
}
defer f.Close()
// ... tout le reste de la fonction peut sortir n'importe où, f sera fermé
```

Deux règles à connaître, parce qu'elles surprennent.

**Les `defer` s'exécutent dans l'ordre inverse de leur déclaration** (dernier entré, premier sorti, comme une pile). C'est logique : ce qui a été ouvert en dernier doit être fermé en premier.

```go
for i := 1; i <= 3; i++ {
	defer fmt.Println("defer", i)
}
fmt.Println("fin de ordre")
```

```
fin de ordre
defer 3
defer 2
defer 1
```

**Les arguments sont évalués au moment du `defer`, pas au moment de l'exécution.** La fonction est reportée, ses arguments non :

```go
x := 1
defer fmt.Println("x au moment du defer :", x)
x = 2
defer func() {
	fmt.Println("x dans la closure :", x)
}()
x = 3
```

```
x dans la closure : 3
x au moment du defer : 1
```

Le premier `defer` a figé `x` à 1 en le passant comme argument. Le second est une fonction anonyme (une *closure*, chapitre 5) qui lit `x` au moment où elle s'exécute, donc 3. Les deux comportements sont utiles ; il faut juste savoir lequel on écrit. Et la closure s'exécute avant, parce qu'elle a été déclarée après.

**Venant du C :** `defer` est le `goto cleanup` en propre, et le mécanisme derrière `__attribute__((cleanup))` de GCC. Chaque `return` n'a plus besoin de refaire les `fclose` et les `free` ; il y a un seul endroit, à côté de l'ouverture.

**Venant de Python :** c'est le `finally` et le `with` réunis, sans indentation supplémentaire. `with open(f) as fh:` devient `fh, err := os.Open(f); ...; defer fh.Close()`.

**Piège :** `defer` est attaché à la **fonction**, pas au bloc. Un `defer f.Close()` dans une boucle qui ouvre mille fichiers gardera mille fichiers ouverts jusqu'à la fin de la fonction. Dans ce cas, sors le corps de la boucle dans une fonction à part, pour que chaque tour ferme le sien. Le chapitre 8 reviendra sur `defer` avec `recover`, pour attraper un `panic`.

### 3.6 Ce que Go n'a pas, et par quoi le remplacer

**Pas de `while`.** C'est `for cond { }`. Un mot-clé de moins, aucune perte.

**Pas de ternaire.** `x = cond ? a : b` n'existe pas, et il n'y a pas d'expression `if`. On écrit :

```go
etat := "ok"
if charge > 0.9 {
	etat = "critique"
}
```

C'est une ligne de plus, c'est voulu : les concepteurs ont jugé que les ternaires imbriqués nuisent plus que le ternaire simple n'aide. Tu vas râler une semaine, puis oublier.

**Pas de `do ... while`**, pas de `for ... else` à la Python, pas de compréhensions.

**`goto` existe**, avec des étiquettes, et le compilateur interdit de sauter par-dessus une déclaration de variable ou dans un bloc. Tu le croiseras dans du code généré ou dans quelques boucles très optimisées de la bibliothèque standard. Toi, tu ne l'écris pas : `break étiquette`, `continue étiquette` et `return` couvrent tous les cas légitimes.

**Pas d'exceptions.** Le `try / except` de Python n'a pas d'équivalent ; les erreurs sont des valeurs de retour, et c'est le sujet du chapitre 5. Le `panic` existe pour les situations irrécupérables (index hors bornes, déréférencement de `nil`), et `defer` plus `recover` permettent de l'attraper, mais ce n'est pas un mécanisme de contrôle de flux normal.

### 3.7 Pour le labo

Le [labo 03](../labs/03-devine-nombre/README.md) est un jeu de devinette : une boucle de lecture au clavier, un `switch` pour comparer, un `defer` qui annonce le nombre d'essais à la fin, et un générateur de nombres aléatoires avec une graine fixe pour que les tests puissent rejouer la même partie.

### À retenir

- `if cond {}` sans parenthèses, accolades obligatoires, `else` sur la ligne du `}`, condition strictement booléenne.
- `if x, err := f(); err != nil {}` : l'instruction d'initialisation limite la variable au `if`. C'est l'idiome n°1 de Go.
- Une seule boucle, `for`, en trois formes : C, `while`, infinie ; plus `for i := range n` et `for i, v := range collection`.
- `switch` sans `break` (pas de chute), plusieurs valeurs par `case`, ou sans expression pour remplacer les `if / else if` ; `fallthrough` pour la chute explicite, rare.
- `break étiquette` et `continue étiquette` pour viser une boucle extérieure ; un `break` dans un `switch` ne sort pas du `for`.
- `defer` reporte un appel à la fin de la fonction : ordre inverse (pile), arguments évalués tout de suite. `defer f.Close()` juste après le `if err != nil`.
- Pas de `while`, pas de ternaire, pas d'exceptions, `goto` à ne pas utiliser.

---

← [2. Variables, types, constantes, chaînes](02-variables-types-chaines.md) · [Sommaire](../README.md) · [4. Slices, maps, range : les collections sous le capot](04-slices-maps.md) →
