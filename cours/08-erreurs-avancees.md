# 8. Erreurs pour de vrai : wrapping, Is, As, panic

*Go de zéro à la prod : chapitre 8 sur 16.* ← [7. Interfaces et composition](07-interfaces.md) · [Sommaire](../README.md) · [9. Paquets, modules, tests](09-paquets-modules-tests.md) →

Le chapitre 5 t'a appris le réflexe : une fonction qui peut échouer renvoie une `error` en dernier, et l'appelant fait `if err != nil { return err }`. Ça suffit pour un script. Ça ne suffit pas pour un programme qui tourne des mois : quand le message « permission denied » arrive dans les logs à trois heures du matin, tu veux savoir **quel** fichier, ouvert par **quelle** étape, pour **quelle** opération. Et le code appelant veut pouvoir distinguer « le fichier n'existe pas » (on crée une config par défaut) de « permission refusée » (on s'arrête).

Go n'a pas d'exceptions, mais il a, depuis Go 1.13, un petit système cohérent : on **emballe** les erreurs en remontant, on les **interroge** avec `errors.Is` et `errors.As`, et le message final se lit comme un chemin. Ce chapitre te donne ce système en entier, puis traite les deux cas où Go abandonne le flux normal : `panic` et `os.Exit`.

### 8.1 `error` est une interface, `errors.New` en fabrique une

Tu l'as vu au chapitre 7 : `error` est l'interface à une méthode `Error() string`. Toute valeur qui a cette méthode est une erreur, et `errors.New("texte")` en fabrique une avec un message fixe.

Quand une erreur représente une **condition connue** que l'appelant voudra tester, on la déclare une fois, au niveau du paquet, dans une variable exportée qui commence par `Err`. C'est une **erreur sentinelle**.

```go
package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

var ErrPortInvalide = errors.New("port invalide")

func validerPort(p int) error {
	if p < 1 || p > 65535 {
		return ErrPortInvalide
	}
	return nil
}

func main() {
	err := validerPort(70000)
	fmt.Println(err)
	fmt.Println(err == ErrPortInvalide)
	fmt.Println(errors.Is(err, ErrPortInvalide))
	fmt.Println(validerPort(22) == nil)

	r := strings.NewReader("ab")
	buf := make([]byte, 4)
	for {
		n, err := r.Read(buf)
		fmt.Printf("n=%d err=%v\n", n, err)
		if err == io.EOF {
			break
		}
	}
}
```

```
$ go run .
port invalide
true
true
true
n=2 err=<nil>
n=0 err=EOF
```

- `var ErrPortInvalide = errors.New(...)` : une seule valeur, créée au démarrage. `validerPort` la renvoie telle quelle, et l'appelant compare avec `==` : c'est la même adresse. Deux `errors.New("port invalide")` séparés seraient deux erreurs différentes, même message ; c'est pour ça qu'on la déclare une fois.
- `io.EOF` est la sentinelle la plus célèbre de la bibliothèque standard : chaque `Read` la renvoie quand il n'y a plus rien à lire. Tu la croiseras dans chaque boucle de lecture. `os.ErrNotExist`, `sql.ErrNoRows`, `context.Canceled` sont de la même famille.
- `errors.Is(err, ErrPortInvalide)` donne ici la même chose que `==`. La différence apparaît à la section suivante, quand l'erreur a été emballée. Prends l'habitude d'`errors.Is` partout ; `==` ne marche que sur une erreur nue.

**Venant du C :** une sentinelle est un `errno` avec un nom et un message, mais c'est une valeur qu'on renvoie, pas une variable globale qu'on va lire après coup. Impossible de l'oublier ou de la faire écraser par un autre appel.

**Venant de Python :** une sentinelle joue le rôle d'une classe d'exception qu'on attrape avec `except FileNotFoundError`. Sauf qu'on ne « lève » rien : on renvoie, et l'appelant décide. Rien ne remonte tout seul la pile ; si tu ne traites pas une erreur, elle disparaît silencieusement. C'est le prix du système, et `golangci-lint` te signale chaque erreur ignorée.

### 8.2 Emballer avec `%w`, remonter une chaîne lisible

Le principe : chaque fonction qui reçoit une erreur d'une fonction plus basse y ajoute **son contexte** avant de la renvoyer. Pas en remplaçant l'erreur (on perdrait la cause), pas en concaténant des chaînes (on perdrait la valeur), mais en l'**emballant** avec `fmt.Errorf` et le verbe `%w`.

```go
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

func lireFichier(chemin string) ([]byte, error) {
	b, err := os.ReadFile(chemin)
	if err != nil {
		return nil, fmt.Errorf("lire %s: %w", chemin, err)
	}
	return b, nil
}

func chargerConfig(chemin string) error {
	_, err := lireFichier(chemin)
	if err != nil {
		return fmt.Errorf("charger config: %w", err)
	}
	return nil
}

func main() {
	err := chargerConfig("/etc/pgbouncer/pgbouncer.ini")
	fmt.Println(err)
	fmt.Println(errors.Is(err, os.ErrNotExist))
	fmt.Println(errors.Is(err, fs.ErrNotExist))
	fmt.Println(errors.Unwrap(err))

	err = chargerConfig("/etc/master.passwd")
	fmt.Println(err)
	fmt.Println(errors.Is(err, fs.ErrPermission))

	var pe *fs.PathError
	if errors.As(err, &pe) {
		fmt.Printf("op=%s path=%s err=%v\n", pe.Op, pe.Path, pe.Err)
	}
}
```

```
$ go run .
charger config: lire /etc/pgbouncer/pgbouncer.ini: open /etc/pgbouncer/pgbouncer.ini: no such file or directory
true
true
lire /etc/pgbouncer/pgbouncer.ini: open /etc/pgbouncer/pgbouncer.ini: no such file or directory
charger config: lire /etc/master.passwd: open /etc/master.passwd: permission denied
true
op=open path=/etc/master.passwd err=permission denied
```

Regarde le premier message. Il se lit de gauche à droite comme une pile d'appels, du plus haut niveau (« charger config ») au plus bas (« no such file or directory »), et chaque étage a ajouté ce qu'il savait. C'est **la** convention Go pour les messages d'erreur : `contexte: cause`, avec deux points et une espace, pas de majuscule, pas de point final, pas de « erreur : » (l'appelant l'ajoutera s'il l'affiche). Chaque étage ne dit que ce que l'étage du dessous ne sait pas : `lireFichier` ajoute le chemin, `chargerConfig` ajoute l'opération, `os.ReadFile` avait déjà mis `open` et le chemin. Évite de répéter ce que la cause dit déjà.

- `%w` (*wrap*) dans `fmt.Errorf` produit une erreur qui **contient** l'erreur d'origine. `%v` produirait juste un texte, et `errors.Is` ne verrait plus rien derrière. Un seul `%w` par `Errorf` en règle générale (plusieurs sont permis depuis Go 1.20, c'est rare).
- `errors.Unwrap(err)` rend l'erreur emballée, un étage en dessous. On ne l'appelle presque jamais soi-même : `Is` et `As` parcourent la chaîne pour nous.
- `errors.Is(err, cible)` descend la chaîne et dit si **un des étages** est `cible`. Ici, tout en bas, `os.ReadFile` a renvoyé une erreur qui emballe `fs.ErrNotExist` (`os.ErrNotExist` est le même objet, un alias). Deux niveaux d'emballage plus haut, `Is` la trouve toujours.
- `errors.As(err, &pe)` descend la chaîne et cherche un étage **du type** `*fs.PathError` ; s'il le trouve, il le copie dans `pe`. On récupère ainsi les champs de l'erreur concrète (`Op`, `Path`) sans casser l'emballage. Note le `&pe` : `As` a besoin de l'adresse d'une variable du type cherché pour la remplir, comme `json.Unmarshal`.

**Venant de Python :** `%w` est `raise NouvelleErreur(...) from err`, et la chaîne `contexte: cause` est ce que Python affiche en « The above exception was the direct cause of the following exception ». `errors.Is` remplace `except Type`, `errors.As` remplace `except Type as e` pour lire ses attributs. Ce que Go n'a pas, c'est la trace de pile automatique : la chaîne de contextes la remplace, et elle est plus lisible dans un log, parce que c'est toi qui as choisi les mots.

**Piège :** `errors.Is(err, ErrX)` avec `err` construit par `fmt.Errorf("...: %v", ErrX)` renvoie `false`. Le `%v` a transformé l'erreur en texte. Si tu ne vois pas pourquoi un `Is` échoue, cherche un `%v` ou un `err.Error()` dans la chaîne.

### 8.3 Un type d'erreur avec des champs

Quand l'appelant a besoin de **données** sur l'erreur (quel champ, quelle ligne, quel code HTTP), une sentinelle ne suffit plus. On déclare un type struct avec les champs voulus et une méthode `Error()`.

```go
package main

import (
	"errors"
	"fmt"
	"strconv"
)

type ErreurValidation struct {
	Champ  string
	Raison string
}

func (e *ErreurValidation) Error() string {
	return "champ " + e.Champ + " : " + e.Raison
}

func parserPort(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, &ErreurValidation{Champ: "port", Raison: "pas un entier"}
	}
	if n < 1 || n > 65535 {
		return 0, &ErreurValidation{Champ: "port", Raison: "hors de 1..65535"}
	}
	return n, nil
}

func charger(s string) error {
	if _, err := parserPort(s); err != nil {
		return fmt.Errorf("charger section [db]: %w", err)
	}
	return nil
}

func main() {
	for _, s := range []string{"5432", "abc", "99999"} {
		err := charger(s)
		if err == nil {
			fmt.Println(s, ": ok")
			continue
		}
		fmt.Println(err)
		var ev *ErreurValidation
		if errors.As(err, &ev) {
			fmt.Printf("   -> champ=%q raison=%q\n", ev.Champ, ev.Raison)
		}
	}
}
```

```
$ go run .
5432 : ok
charger section [db]: champ port : pas un entier
   -> champ="port" raison="pas un entier"
charger section [db]: champ port : hors de 1..65535
   -> champ="port" raison="hors de 1..65535"
```

Trois règles pour ce genre de type :

1. `Error()` a un receveur pointeur, et on renvoie `&ErreurValidation{...}`. Une erreur est une valeur d'identité (deux validations distinctes ne sont pas « égales »), le pointeur exprime ça, et c'est ce que `errors.As` attend avec `var ev *ErreurValidation`.
2. La fonction renvoie **`error`**, jamais `*ErreurValidation` : c'est le piège du nil typé du chapitre 7. `parserPort` renvoie `(int, error)`, et le `return 0, &ErreurValidation{...}` se convertit en `error` sans rien dire.
3. Le type est exporté si un autre paquet doit l'inspecter avec `As`. Sinon, garde-le privé : c'est un détail.

Sentinelle ou type ? **Sentinelle** quand l'appelant veut juste savoir *que* c'est arrivé (`ErrIntrouvable`, `ErrDejaExistant`, `io.EOF`). **Type** quand il veut savoir *quoi* (quel champ, quel statut, quelle ligne). Et un type peut aussi emballer : ajoute un champ `Err error` et une méthode `Unwrap() error { return e.Err }`, et `errors.Is` continue de descendre à travers lui. C'est ainsi que `*fs.PathError` emballe `fs.ErrNotExist`.

### 8.4 `errors.Join` : plusieurs erreurs en une

Une validation qui s'arrête à la première faute oblige l'utilisateur à relancer dix fois. Depuis Go 1.20, `errors.Join` regroupe plusieurs erreurs en une seule, que `Is` et `As` parcourent toutes.

```go
package main

import (
	"errors"
	"fmt"
)

var ErrDNS = errors.New("résolution DNS")
var ErrTLS = errors.New("certificat expiré")

func verifierHote(h string) error {
	var errs []error
	if h == "" {
		errs = append(errs, ErrDNS)
	}
	if len(h) < 6 {
		errs = append(errs, ErrTLS)
	}
	return errors.Join(errs...)
}

func main() {
	err := verifierHote("")
	fmt.Println(err)
	fmt.Println(errors.Is(err, ErrDNS), errors.Is(err, ErrTLS))
	fmt.Println(verifierHote("db-01.prod") == nil)

	var errs []error
	for _, h := range []string{"", "web", "db-01.prod"} {
		if e := verifierHote(h); e != nil {
			errs = append(errs, fmt.Errorf("hôte %q: %w", h, e))
		}
	}
	fmt.Println(errors.Join(errs...))
}
```

```
$ go run .
résolution DNS
certificat expiré
true true
true
hôte "": résolution DNS
certificat expiré
hôte "web": certificat expiré
```

`errors.Join` d'une slice vide (ou de que des `nil`) renvoie `nil` : la fonction peut accumuler puis renvoyer `Join(errs...)` sans tester. Le message met chaque erreur sur sa ligne. C'est l'outil pour « vérifie tous les hôtes et dis-moi tout ce qui ne va pas », ou pour un `Close()` qui doit fermer trois ressources et rapporter chaque échec.

### 8.5 Quand emballer, quand ne pas emballer

Emballer partout mécaniquement donne des messages qui répètent trois fois le même chemin. Ne rien emballer donne un `permission denied` nu, inutilisable. La règle en trois points :

- **Emballe quand tu ajoutes une information** que la couche du dessous n'avait pas : le nom de la ressource, l'opération en cours, l'identifiant de la requête. `fmt.Errorf("ouvrir config %s: %w", chemin, err)`.
- **Renvoie tel quel** (`return err`) quand tu n'as rien à ajouter, typiquement dans une petite fonction intermédiaire qui ne fait que passer l'appel. Pas de `fmt.Errorf("erreur: %w", err)`, ça n'apporte rien.
- **N'emballe pas avec `%w` quand tu ne veux pas exposer** l'erreur interne comme partie de ton API. Si ta bibliothèque emballe `sql.ErrNoRows` avec `%w`, tes utilisateurs vont écrire `errors.Is(err, sql.ErrNoRows)`, et le jour où tu changes de base de données, tu les casses. Dans ce cas, renvoie ta propre sentinelle (`ErrIntrouvable`) et mets la cause avec `%v` dans le message, pour les logs.

Et au sommet, dans `main` ou dans le gestionnaire HTTP, **on traite** : on affiche, on logue, on renvoie un code de sortie ou un statut HTTP. C'est le seul endroit où l'erreur cesse de remonter. Une erreur traitée à mi-chemin et aussi renvoyée est loguée deux fois : logue **ou** renvoie, jamais les deux.

### 8.6 `panic` et `recover`

Un `panic` arrête la fonction en cours, exécute ses `defer`, puis remonte à l'appelant, exécute ses `defer`, et ainsi de suite jusqu'à `main` : le programme s'arrête avec un message et la trace de pile. C'est ce que tu as vu au chapitre 6 avec le pointeur nil, et au chapitre 7 avec l'assertion de type ratée. L'exécutif panique pour les erreurs de programmation : index hors bornes, écriture dans une map nil, division entière par zéro, déréférencement de nil.

Toi, tu écris `panic(...)` dans un seul cas : **un état impossible**, qui révèle un bug, et dont il n'y a rien à faire à part corriger le code. Jamais pour un fichier absent, une entrée utilisateur invalide, un réseau qui tombe : ça, c'est une `error`.

`recover()` est le pendant : appelé dans une fonction `defer`, il arrête la remontée du panic et renvoie sa valeur. Hors d'un `defer`, il renvoie `nil` et ne fait rien.

```go
package main

import "fmt"

func Securise(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panique récupérée : %v", r)
		}
	}()
	f()
	return nil
}

func main() {
	err := Securise(func() {
		var m map[string]int
		m["x"] = 1
	})
	fmt.Println(err)

	err = Securise(func() {
		s := []int{1, 2, 3}
		i := 5
		fmt.Println(s[i])
	})
	fmt.Println(err)

	err = Securise(func() { fmt.Println("tout va bien") })
	fmt.Println(err)

	panic("état impossible : pool sans connexion")
}
```

```
$ go run .
panique récupérée : assignment to entry in nil map
panique récupérée : runtime error: index out of range [5] with length 3
tout va bien
<nil>
panic: état impossible : pool sans connexion

goroutine 1 [running]:
main.main()
	/Users/stranix/ex/main.go:34 +0xe0
exit status 2
```

Le motif à mémoriser est `Securise` : un **retour nommé** `(err error)` (chapitre 5), un `defer` avec une closure qui appelle `recover()`, et qui affecte `err` si un panic a eu lieu. C'est le seul cas où un retour nommé est vraiment nécessaire : le `defer` s'exécute après le `return`, et c'est la variable de retour qu'il modifie.

**Venant de Python :** `panic`/`recover` ressemble à `raise`/`except`, et c'est bien pour ça qu'il faut résister : **ce n'est pas un mécanisme de contrôle de flux**. Du code Go qui utilise `panic` pour signaler « fichier introuvable » et `recover` trois niveaux plus haut pour le rattraper est du Python écrit en Go, et personne ne pourra le lire. Le `panic` est réservé aux bugs, `recover` aux frontières.

Quelles frontières ? Deux, principalement. Un **serveur** : une requête qui panique ne doit pas tuer les mille autres en cours. Et une **bibliothèque** qui appelle du code fourni par l'utilisateur (un plugin, une closure) et qui ne veut pas planter le programme hôte. Voici la première, sous forme de *middleware* HTTP (le chapitre 13 explique `http.Handler` ; ici, regarde juste le `defer`) :

```go
func recupere(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				log.Printf("panique sur %s : %v", r.URL.Path, p)
				http.Error(w, "erreur interne", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
```

```
$ go run .
/sante 200 OK
2026/09/23 12:25:36 panique sur /boum : assignment to entry in nil map
/boum 500 Internal Server Error
/sante 200 OK
```

Le gestionnaire de `/boum` écrit dans une map nil, le middleware attrape, logue, répond 500, et la requête suivante passe. (Le serveur `net/http` de la bibliothèque standard fait déjà ça tout seul pour ne pas mourir ; le middleware sert à loguer proprement et à répondre quelque chose au client.)

### 8.7 `os.Exit`, `log.Fatal`, et les `defer` qui ne tournent pas

Deux façons de terminer un programme avec un code de sortie, et un piège commun aux deux.

```go
func main() {
	defer fmt.Println("ce defer ne s'affichera jamais")
	if len(os.Args) > 1 && os.Args[1] == "exit" {
		fmt.Fprintln(os.Stderr, "erreur : config introuvable")
		os.Exit(2)
	}
	log.Fatalf("config introuvable : %s", "/etc/app.ini")
}
```

```
$ go run .
2026/09/23 12:25:36 config introuvable : /etc/app.ini
exit status 1
$ go build -o f . && ./f exit ; echo "rc=$?"
erreur : config introuvable
rc=2
```

- `os.Exit(code)` termine **immédiatement**. Les `defer` en attente ne s'exécutent pas, les fichiers ne sont pas vidés, rien. C'est le `exit()` du C. Code 0 pour succès, 1 pour une erreur générale, 2 pour une erreur d'usage (arguments invalides), par convention Unix.
- `log.Fatal`/`log.Fatalf` = écrire le message sur stderr avec l'horodatage, puis `os.Exit(1)`. Même conséquence pour les `defer`.

**Piège :** ni l'un ni l'autre ne doivent apparaître ailleurs que dans `main` (ou dans un `init`, section 9.3). Une bibliothèque qui fait `log.Fatal` tue le programme de son utilisateur sans lui laisser fermer sa base de données. Et même dans `main`, le motif propre est de garder `main` minuscule : une fonction `run() error` qui contient toute la logique et dont les `defer` s'exécutent normalement, et un `main` qui fait `if err := run(); err != nil { fmt.Fprintln(os.Stderr, "erreur :", err); os.Exit(1) }`. Tous les labos à partir du 09 sont écrits ainsi.

### 8.8 Avant-goût : `log/slog`

Le paquet `log` que tu viens de voir est celui de 2012 : une ligne de texte avec une date. Depuis Go 1.21, `log/slog` est le logueur **structuré** de la bibliothèque standard : chaque message a un niveau et des paires clé-valeur, et la sortie peut être du texte ou du JSON, ce que Loki, Elasticsearch ou `jq` savent lire.

```go
package main

import (
	"errors"
	"log/slog"
	"os"
)

func main() {
	slog.Info("démarrage", "port", 9119, "env", "prod")
	slog.Warn("latence élevée", "cible", "db-01", "ms", 250)
	err := errors.New("connexion refusée")
	slog.Error("sonde échouée", "cible", "cache-01", "err", err)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("démarrage", "port", 9119, "env", "prod")
	logger.Error("sonde échouée", "cible", "cache-01", "err", err)
}
```

```
$ go run .
2026/09/23 12:25:35 INFO démarrage port=9119 env=prod
2026/09/23 12:25:35 WARN latence élevée cible=db-01 ms=250
2026/09/23 12:25:35 ERROR sonde échouée cible=cache-01 err="connexion refusée"
{"time":"2026-09-23T12:25:35.169452+02:00","level":"INFO","msg":"démarrage","port":9119,"env":"prod"}
{"time":"2026-09-23T12:25:35.169461+02:00","level":"ERROR","msg":"sonde échouée","cible":"cache-01","err":"connexion refusée"}
```

Les arguments vont par deux, clé puis valeur ; une erreur passée en valeur est affichée par son message. Le chapitre 15 y revient (niveaux, handler par défaut, contexte). Retiens seulement que pour un service, c'est `slog` dès la première ligne, et jamais `fmt.Println` pour loguer.

### 8.9 Pour le labo

Le [labo 08](../labs/08-robuste/README.md) te fait écrire un lecteur de configuration robuste : deux sentinelles `ErrIntrouvable` et `ErrInvalide`, un type `ErreurValidation{Champ, Raison}`, une fonction `ChargerConfig(chemin)` dont chaque étape emballe la précédente pour produire un message du genre `charger config: lire /tmp/x/app.conf: open ...: no such file or directory`, des tests qui vérifient la chaîne avec `errors.Is` et `errors.As` sur des fichiers dans `t.TempDir()`, et la fonction `Securise` de la section 8.6.

### À retenir

- `error` est une interface ; `errors.New` fabrique une erreur à message fixe ; une sentinelle est une `var ErrX = errors.New(...)` exportée, testée avec `errors.Is`.
- Emballe avec `fmt.Errorf("contexte: %w", err)` en remontant, en ajoutant à chaque étage seulement ce que l'étage du dessous ne savait pas. Le message final se lit comme un chemin : `charger config: lire x: open x: permission denied`.
- `errors.Is(err, ErrX)` cherche une sentinelle dans la chaîne ; `errors.As(err, &cible)` cherche un type et en récupère les champs. `%v` casse la chaîne, `%w` la préserve.
- Un type d'erreur est une struct avec `Error()` à receveur pointeur, renvoyée comme `error` (jamais comme `*MonErreur`).
- `errors.Join` regroupe plusieurs erreurs ; `nil` s'il n'y en a pas.
- `panic` seulement pour un état impossible ; `recover` seulement dans un `defer`, aux frontières (serveur, plugin), avec le motif du retour nommé.
- `os.Exit` et `log.Fatal` n'exécutent pas les `defer` ; réserve-les à `main`, et écris un `run() error`.
- `log/slog` pour loguer : structuré, niveaux, JSON en une ligne de configuration.

---

← [7. Interfaces et composition](07-interfaces.md) · [Sommaire](../README.md) · [9. Paquets, modules, tests](09-paquets-modules-tests.md) →
