# 16. Projet final et pour aller plus loin

*Go de zéro à la prod : chapitre 16 sur 16.* ← [15. Qualité, build, release, Docker, observabilité](15-qualite-build-release.md) · [Sommaire](../README.md)

Quinze chapitres, quinze labos, chacun sur un sujet. Il manque une chose : un programme où tout se tient ensemble, où le type choisi dans `config` a des conséquences dans `api`, où les goroutines de sondes écrivent dans une structure que les handlers HTTP lisent au même moment, où l'arrêt propre doit attendre tout le monde. C'est ce chapitre : d'abord le projet, son architecture et ses pièges, puis ce qu'il te reste à découvrir en Go une fois le cours fini.

### 16.1 Le projet : sondes

Un service de supervision, comme un Uptime Kuma de poche. Tu lui donnes des cibles (un site en HTTP, un port TCP), il les vérifie en parallèle à intervalle régulier, garde le dernier état de chacune, l'expose en JSON pour les humains et au format Prometheus pour Grafana, journalise en JSON, s'arrête proprement et tient dans une image Docker de quelques méga-octets. C'est l'outil que tu aurais écrit en Python au chapitre 21 du cours précédent, et c'est exactement le genre de programme pour lequel Go a été conçu.

Le cahier des charges complet est dans le [README du labo](../labs/16-projet/README.md). En résumé :

| Brique | Ce qu'elle fait | Chapitres mobilisés |
|---|---|---|
| `internal/config` | lit le JSON, applique les défauts, valide, enveloppe `ErrInvalide` | 8, 12 |
| `internal/sonde` | `SonderHTTP`, `SonderTCP` avec `context`, l'interface `Sondeur` | 7, 11, 13 |
| `internal/sonde.Etat` | le dernier résultat par cible, derrière un `RWMutex` | 6, 11 |
| `internal/sonde.Moteur` | worker pool borné, `Ticker`, arrêt sur `ctx.Done()` | 11 |
| `internal/api` | `ServeMux` 1.22, `/healthz`, `/api/status`, `/metrics`, middleware `slog` | 13, 15 |
| `cmd/sondes` | flags, `signal.NotifyContext`, `http.Server.Shutdown` | 12, 13 |
| Livraison | `Makefile`, `-ldflags -X`, compilation croisée, `Dockerfile` vers `scratch` | 15 |
| Tests | `httptest` pour l'API, `net.Listen("tcp", "127.0.0.1:0")` pour les sondes, un `Sondeur` fictif pour le moteur | 9, 13 |

**Venant de Python :** le statusboard du cours Python faisait la même chose avec asyncio, une `Queue` par abonné et SQLModel sur SQLite. Ici, une goroutine par sonde remplace `asyncio.gather`, un channel remplace la `Queue`, un `sync.WaitGroup` remplace `TaskGroup`, et la base disparaît : l'état tient dans une map derrière un mutex. Le programme Python faisait près de 700 lignes et dépendait de FastAPI, SQLModel, httpx et uvicorn ; celui-ci fait 500 lignes de code, tests exclus, et ne dépend de rien. Ce n'est pas que Go est meilleur : c'est que ce problème-là est le sien.

### 16.2 Architecture et trois décisions

```
 sondes.json ──▶ config.Charger ──▶ *config.Config
                                          │
                       ┌──────────────────┴──────────────────┐
                       ▼                                     ▼
              ┌─────────────────┐                  ┌──────────────────┐
              │  sonde.Moteur   │                  │  http.Server     │
              │  Boucle(ctx)    │                  │  api.NouveauMux  │
              │   │ Ticker      │                  │   Journaliser    │
              │   ▼             │    Enregistrer   │   GET /healthz   │
              │  Tour(ctx) ─────┼──┐               │   GET /api/status│◀── curl, Grafana
              │   ├ worker 1 ───┼─▶│  sonde.Etat   │   GET /metrics   │◀── Prometheus
              │   ├ worker 2 ───┼─▶│  RWMutex      │◀──Resultats()────┤
              │   └ worker n ───┼─▶│  map[nom]     │                  │
              └────────┬────────┘  └───────────────┘                  │
                       │ SonderHTTP / SonderTCP                       │
                       ▼                                              │
                 cibles réseau                                        │
                                                                      │
 SIGINT/SIGTERM ──▶ signal.NotifyContext ──▶ ctx annulé ──▶ Shutdown ─┘
                                                     └──▶ Boucle sort, wg.Wait()
```

Trois décisions à comprendre avant de coder, parce qu'elles reviennent dans tout service de ce genre.

**Un worker pool borné plutôt qu'une goroutine par cible.** Lancer `go sonder(c)` pour chaque cible est tentant et marche jusqu'à mille cibles. Au-delà, mille connexions s'ouvrent au même instant, la cible en face voit une rafale, et ton propre système manque de descripteurs de fichiers. Le pool du chapitre 11 (un channel, N workers, un `WaitGroup`) borne le parallélisme à `workers` et lisse la charge. Les goroutines sont gratuites, les sockets ne le sont pas.

**L'état en mémoire derrière un mutex plutôt qu'une base.** Le service ne garde que le dernier résultat par cible : une map de cinquante entrées. Une base serait une dépendance, un fichier, un schéma, pour rien. Le `RWMutex` laisse les handlers HTTP lire en parallèle et fait attendre les écritures des workers. Et `Resultats()` renvoie une copie triée : le handler travaille sans tenir le verrou, et la map n'est jamais exposée. Si un jour il faut l'historique, c'est le chapitre 14 et une table `resultats` ; l'interface ne change pas.

**Le `ServeMux` de la bibliothèque standard plutôt que chi.** Trois routes, une méthode chacune, pas de groupe, pas de paramètre de chemin : le `ServeMux` de Go 1.22 suffit et gère les 405 tout seul. Deckhand utilise chi parce qu'il a une quarantaine de routes, des groupes par préfixe qui portent chacun leurs middlewares (recover, compression, métriques, authentification) ; à trois routes, chi serait une dépendance pour rien. La règle du chapitre 13 : commence par la stdlib, ajoute chi le jour où tu écris `r.PathValue` pour la dixième fois.

### 16.3 Les étapes, de bas en haut

Ne commence pas par `main`. Commence par ce qui se teste sans rien lancer, et remonte.

1. **`config`.** `Duree` qui implémente `encoding.TextUnmarshaler`, `Parser` qui décode par-dessus les défauts, `Valider` qui refuse tout ce qui est louche. `go test ./internal/config` vert avant de continuer : tout le reste suppose une config saine.
2. **`sonde.go`.** Les deux fonctions de sonde, chacune testée contre un serveur local : `httptest.NewServer` pour HTTP, `net.Listen("tcp", "127.0.0.1:0")` pour TCP (le port 0 demande au système un port libre). Le test du timeout est le plus instructif : un handler qui dort 300 ms, un client à 50 ms, et la sonde doit revenir en moins de 250 ms.
3. **`etat.go`.** Vingt lignes, mais c'est là que `go test -race` va te surveiller. Chaque méthode prend le verrou, `Resultats` copie.
4. **`moteur.go`.** Le worker pool. Le test mesure : six sondes de 20 ms sur deux workers doivent prendre environ 60 ms. Si tu obtiens 20 ms, tes workers ne sont pas bornés ; 120 ms, ils ne sont pas parallèles.
5. **`api`.** Les handlers et `FormatMetriques`, testés avec `httptest.NewRecorder`. Le test du format Prometheus compare la sortie à l'octet près, avec un nom de cible qui contient un guillemet.
6. **`main`.** Le branchement et l'arrêt propre. Lance-le, `curl`, Ctrl-C, et regarde le journal : `arrêt demandé` puis `arrêté` en moins d'une milliseconde.
7. **`Makefile` et `Dockerfile`.** `make build`, `make cross`, `make lint`. Le `Dockerfile` est fourni et commenté ; si tu as Docker, `make docker` puis `docker run -p 8080:8080 sondes:dev`.

### 16.4 Ce qu'on évalue

- `go test ./...` vert, **et** `go test -race ./...` vert. Le second est celui qui compte : un test qui passe sans `-race` et échoue avec est une course de données, et elle finira en production.
- `gofmt -l .` vide, `go vet ./...` muet, `golangci-lint run` sans avertissement.
- L'arrêt propre en moins d'une seconde : Ctrl-C, le journal dit `arrêté`, le processus rend la main. Aucun `os.Exit` au milieu de `run`.
- Aucune goroutine qui fuit : après `Boucle` et `Shutdown`, il ne reste que `main`. `runtime.NumGoroutine()` à la fin de `run` doit être petit et stable.
- `/metrics` scrappable par un vrai Prometheus : `scrape_configs: [{job_name: sondes, static_configs: [{targets: ["localhost:8080"]}]}]`, et `sonde_up` apparaît dans l'explorateur.
- Une cible injoignable est un résultat down, jamais une erreur du programme. Le service tourne quand tout est en panne autour de lui : c'est précisément à ce moment-là qu'on a besoin de lui.

Pour voir ce que `-race` attrape, voici l'`Etat` qu'on écrit en premier, sans mutex, et un test qui l'appelle depuis quatre goroutines :

```go
type Etat struct {
	derniers map[string]bool
	total    int
}

func (e *Etat) Enregistrer(cible string, up bool) {
	e.derniers[cible] = up
	e.total++
}
```

```
$ go test .
ok  	exemple.com/course	0.324s

$ go test -race .
==================
WARNING: DATA RACE
Write at 0x00c00008c870 by goroutine 11:
  runtime.mapaccess2_faststr()
  exemple.com/course.(*Etat).Enregistrer()
      etat.go:12 +0x9c
  exemple.com/course.TestEtatConcurrent.func1()
      etat_test.go:15 +0x78
Previous write at 0x00c00008c870 by goroutine 8:
  runtime.mapaccess2_faststr()
  exemple.com/course.(*Etat).Enregistrer()
      etat.go:12 +0x9c
--- FAIL: TestEtatConcurrent (0.00s)
FAIL
```

Sans `-race`, le test passe : quatre écritures qui se marchent dessus donnent le bon total neuf fois sur dix. Avec, le détecteur montre les deux goroutines, la ligne exacte (`etat.go:12`, l'écriture dans la map), et le test échoue. Les chemins sont abrégés ici ; en vrai, ils sont complets. Sur une map, la course finit d'ailleurs par un `fatal error: concurrent map writes` en production, sans `-race` pour t'avertir avant.

### 16.5 Les pièges que tu vas rencontrer

Chacun coûte une heure si on ne sait pas qu'il existe.

**Piège : le contexte qui s'arrête en chemin.** `Boucle(ctx)` reçoit le contexte, le passe à `Tour(ctx)`, qui le passe à `Sonder(ctx, c)`, qui le passe à `http.NewRequestWithContext(ctx, ...)` et à `DialContext(ctx, ...)`. Si un seul maillon utilise `context.Background()` à la place, un Ctrl-C pendant une sonde vers un hôte muet attend le timeout complet. Le contexte se passe de main en main jusqu'à la socket.

**Piège : `http.Client` sans `Timeout`.** `http.DefaultClient` n'a pas de délai. Une cible qui accepte la connexion et ne répond jamais bloque le worker pour toujours, puis le suivant, puis tous. `&http.Client{Timeout: cfg.Timeout}`, toujours, et un test qui le prouve.

**Piège : le mutex copié.** `func (e Etat) Enregistrer(...)` avec un récepteur par valeur copie la struct, donc le mutex : chaque appel verrouille une copie et personne n'est protégé. `go vet` le signale (`passes lock by value`). Récepteur pointeur, et `Etat` ne circule que comme `*Etat`.

**Piège : la goroutine qui fuit sur un channel.** `erreurs <- srv.ListenAndServe()` dans une goroutine : si le channel n'est pas bufferisé et que personne ne lit (parce qu'on est sorti par `ctx.Done()`), la goroutine reste bloquée à jamais. `make(chan error, 1)` : l'envoi passe même sans lecteur, la goroutine finit.

**Piège : le `Ticker` jamais arrêté.** `time.NewTicker` sans `defer ticker.Stop()` continue de tourner après le retour de `Boucle`. Ici ce n'est qu'un ticker, mais dans un handler appelé mille fois par minute, c'est mille tickers.

**Piège : `resp.Body` jamais fermé.** Chaque réponse HTTP non fermée est une connexion qui ne revient pas dans le pool du client. Après une centaine de sondes, le client ouvre une nouvelle connexion à chaque fois, puis le système refuse. `defer resp.Body.Close()` juste après la vérification de l'erreur, et vider le corps avec `io.Copy(io.Discard, ...)` pour que la connexion soit réutilisable.

### 16.6 Pour aller plus loin : les génériques

Le cours n'a pas utilisé les génériques (Go 1.18, 2022), parce qu'un service comme `sondes` n'en a pas besoin. Mais tu les croiseras dans les bibliothèques, et il faut savoir les lire. Un *paramètre de type* est un type inconnu au moment où on écrit la fonction, fixé au moment où on l'appelle : `func Map[T, R any](xs []T, f func(T) R) []R`. Entre crochets, les paramètres de type et leur *contrainte* : `any` accepte tout, `cmp.Ordered` accepte ce qui supporte `<` (entiers, flottants, chaînes), et une interface fait office de contrainte sur mesure.

```go
package main

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Map applique f à chaque élément et renvoie la slice des résultats.
func Map[T, R any](xs []T, f func(T) R) []R {
	out := make([]R, 0, len(xs))
	for _, x := range xs {
		out = append(out, f(x))
	}
	return out
}

// Filter garde les éléments pour lesquels garder renvoie vrai.
func Filter[T any](xs []T, garder func(T) bool) []T {
	var out []T
	for _, x := range xs {
		if garder(x) {
			out = append(out, x)
		}
	}
	return out
}

// Max : la contrainte cmp.Ordered autorise x > m. Avec any, ça ne compile pas.
func Max[T cmp.Ordered](xs []T) T {
	var m T
	for i, x := range xs {
		if i == 0 || x > m {
			m = x
		}
	}
	return m
}

type Serveur struct {
	Nom string
	CPU int
}

func main() {
	serveurs := []Serveur{{"web-01", 4}, {"db-01", 16}, {"web-02", 4}, {"cache-01", 8}}

	noms := Map(serveurs, func(s Serveur) string { return s.Nom })
	fmt.Println("noms      :", noms)
	gros := Filter(serveurs, func(s Serveur) bool { return s.CPU >= 8 })
	fmt.Println("gros      :", gros)
	fmt.Println("max cpu   :", Max(Map(serveurs, func(s Serveur) int { return s.CPU })))
	fmt.Println("max nom   :", Max(noms))

	// La stdlib a déjà l'essentiel : slices, maps, cmp (Go 1.21).
	slices.SortFunc(serveurs, func(a, b Serveur) int { return cmp.Compare(b.CPU, a.CPU) })
	fmt.Println("par cpu   :", serveurs)
	fmt.Println("contient  :", slices.Contains(noms, "db-01"))
	parRole := map[string]int{"web": 2, "db": 1, "cache": 1}
	fmt.Println("rôles     :", slices.Sorted(maps.Keys(parRole)))
	fmt.Println("majuscule :", strings.Join(Map(noms, strings.ToUpper), ", "))
}
```

```
$ go run .
noms      : [web-01 db-01 web-02 cache-01]
gros      : [{db-01 16} {cache-01 8}]
max cpu   : 16
max nom   : web-02
par cpu   : [{db-01 16} {cache-01 8} {web-01 4} {web-02 4}]
contient  : true
rôles     : [cache db web]
majuscule : WEB-01, DB-01, WEB-02, CACHE-01
```

Le compilateur déduit `T` et `R` à l'appel : `Map(serveurs, ...)` sait que `T` est `Serveur` et `R` est `string` sans qu'on l'écrive. Et `Max(noms)` compile parce que `string` est dans `cmp.Ordered` ; `Max(serveurs)` ne compile pas, et c'est le but.

**Venant de C :** c'est ce que tu faisais avec des macros ou du `void *` et un `sizeof`, mais vérifié par le compilateur, et sans `qsort` qui prend un pointeur de fonction non typé.

**Venant de Python :** `def premier[T](xs: list[T]) -> T` de Python 3.12 est la même idée, sauf qu'ici c'est vérifié à la compilation et pas seulement par mypy.

Quand **ne pas** en écrire : quand une seule concrétisation existe (une fonction générique appelée avec un seul type est une fonction normale avec des crochets en plus) ; quand une interface suffit (un `Sondeur` est une interface, pas un `Moteur[S Sondeur]`) ; quand `slices`, `maps` et `cmp` ont déjà la fonction (`slices.Contains`, `slices.Index`, `slices.SortFunc`, `slices.Max`, `maps.Keys`, `min` et `max` intégrés). La règle des auteurs de Go : écris le code trois fois avec des types concrets, et généralise seulement si les trois versions sont identiques au type près. Les génériques se découvrent, ils ne se conçoivent pas.

**Piège :** une méthode ne peut pas avoir ses propres paramètres de type. `func (e *Etat) Filtrer[T any](...)` ne compile pas ; il faut une fonction `Filtrer[T any](e *Etat, ...)` ou rendre le type lui-même générique.

### 16.7 Lire la bibliothèque standard

Le code de la stdlib est dans `$(go env GOROOT)/src`, il est propre, commenté, et c'est du Go que tu sais lire maintenant. Deux fichiers pour commencer, avec la ligne où entrer.

**`net/http/server.go`** (4 292 lignes en Go 1.27). Ligne 89, l'interface la plus importante du paquet, deux lignes :

```go
type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}
```

Puis cherche `func (srv *Server) Serve(l net.Listener) error` : la boucle d'acceptation. Dépouillée de sa gestion d'erreurs, elle tient en quatre lignes qui expliquent pourquoi un serveur Go encaisse dix mille connexions :

```go
	for {
		rw, err := l.Accept()
		...
		c := s.newConn(rw)
		c.setState(c.rwc, StateNew, runHooks) // before Serve can return
		go c.serve(connCtx)
	}
```

Une goroutine par connexion, pas par requête ni par thread. `func (c *conn) serve(ctx context.Context)` (ligne 1929) lit les requêtes de cette connexion l'une après l'autre, et pour chacune appelle `serverHandler{c.server}.ServeHTTP(w, w.req)` (ligne 2137) : c'est là que ton `mux` est appelé, et c'est le `defer func() { if err := recover() ...` au début de `serve` qui fait qu'un panic dans ton handler tue la connexion et pas le serveur. Tout ce que le chapitre 13 affirmait est dans ces cent lignes.

**`sync/waitgroup.go`** (260 lignes). Un compteur atomique 64 bits qui range le nombre de tâches dans les 32 bits hauts et le nombre de `Wait` en attente dans les 32 bits bas. `Done` est littéralement `wg.Add(-1)`. Le commentaire au-dessus de `Add` dit en toutes lettres la règle que tu as apprise au chapitre 11 :

```
// Note that calls with a positive delta that occur when the counter is zero
// must happen before a Wait. [...]
// Typically this means the calls to Add should execute before the statement
// creating the goroutine or other event to be waited for.
```

Et depuis Go 1.25, `wg.Go(func() { ... })` fait `Add(1)`, `go`, et `defer Done()` pour toi : lis-le, il fait douze lignes, et tu comprendras pourquoi il existe (un panic dans `f` ne doit pas débloquer `Wait` avant de tuer le programme). `go doc sync.WaitGroup.Go` te donne la signature ; le fichier te donne la raison.

Le réflexe : quand une fonction de la stdlib te surprend, `go doc` d'abord, puis le fichier. `grep -n "func (srv \*Server) Shutdown" $(go env GOROOT)/src/net/http/server.go` et tu sais exactement ce que fait l'arrêt propre de ton service.

```
$ go doc net/http.Server.Shutdown
func (s *Server) Shutdown(ctx context.Context) error
    Shutdown gracefully shuts down the server without interrupting any active
    connections. Shutdown works by first closing all open listeners, then
    closing all idle connections, and then waiting indefinitely for connections
    to return to idle and then shut down. If the provided context expires before
    the shutdown is complete, Shutdown returns the context's error, otherwise it
    returns any error returned from closing the Server's underlying Listener(s).

    When Shutdown is called, Serve, ServeTLS, ListenAndServe, and
    ListenAndServeTLS immediately return ErrServerClosed. Make sure the program
    doesn't exit and waits instead for Shutdown to return.
    [...]
```

Deux phrases de cette doc expliquent deux lignes de ton `run` : « `ListenAndServe` immediately return `ErrServerClosed` », d'où le `errors.Is(err, http.ErrServerClosed)` qui ne traite pas cette erreur comme une panne ; et « if the provided context expires », d'où le contexte neuf de cinq secondes, parce que celui de `run` est déjà annulé quand on arrive là.

### 16.8 Livres, documents, proverbes, communauté

**Trois livres.** *The Go Programming Language* (Donovan et Kernighan, 2015) : le K&R de Go, écrit par le même Kernighan ; antérieur aux modules et aux génériques, mais tout le reste y est, et la partie sur les interfaces et la concurrence n'a pas été dépassée. *100 Go Mistakes and How to Avoid Them* (Teiva Harsanyi, 2022) : cent erreurs classées par thème, chacune avec le code faux, le code juste et le pourquoi ; c'est le livre à lire après ce cours, une erreur par jour. *Learning Go* (Jon Bodner, 2e édition, 2024) : à jour, génériques et modules compris, le plus proche de ce cours dans l'esprit.

**Deux documents officiels.** [Effective Go](https://go.dev/doc/effective_go), quarante pages, à lire une fois maintenant et une fois dans six mois. Et le [Go Memory Model](https://go.dev/ref/mem), dix pages ardues, qui définit exactement ce que « happens before » veut dire entre goroutines : à lire quand `-race` t'a montré quelque chose que tu ne comprends pas.

**Cinq proverbes**, parmi la vingtaine de Rob Pike ([go-proverbs.github.io](https://go-proverbs.github.io/)) :

- *Don't communicate by sharing memory, share memory by communicating.* Un channel qui passe une valeur plutôt qu'un mutex qui protège une variable, quand les deux sont possibles.
- *Concurrency is not parallelism.* Le worker pool est concurrent (plusieurs choses en cours) ; qu'il soit parallèle (plusieurs cœurs) est le problème de l'ordonnanceur.
- *The bigger the interface, the weaker the abstraction.* `Sondeur` a une méthode. `io.Reader` a une méthode. C'est ce qui permet de les remplacer dans les tests.
- *Make the zero value useful.* Un `sync.Mutex` vide est un mutex prêt ; un `bytes.Buffer` vide est un buffer prêt. Quand tu conçois un type, vise ça.
- *Clear is better than clever.* Le `for` avec un `break` étiqueté dans `Tour` n'est pas élégant. Il est clair.

**La communauté.** Le [Gophers Slack](https://invite.slack.golangbridge.org/) (canal `#newbies`, personne ne se moque), [r/golang](https://www.reddit.com/r/golang/), le podcast [Go Time](https://changelog.com/gotime), et [GopherCon](https://www.gophercon.com/) dont toutes les conférences sont en ligne : commence par *Rethinking Classical Concurrency Patterns* (Bryan C. Mills, GopherCon 2018), puis *Simplicity is Complicated* (Rob Pike, dotGo 2015), qui explique en vingt minutes pourquoi le langage est si petit. Le [blog officiel](https://go.dev/blog) publie les notes de chaque version : vingt minutes tous les six mois.

### 16.9 Des projets à lire

Lire du code écrit par d'autres est la deuxième moitié de l'apprentissage. Quatre projets, du plus petit au plus gros, avec par où entrer.

**Deckhand** ([github.com/stranix79/deckhand](https://github.com/stranix79/deckhand)), un outil de présentation avec un hub multi-utilisateurs : c'est le cours en vrai. `internal/hub/hub.go` monte un routeur chi avec ses groupes et ses middlewares ; `internal/session/ws.go` tient les WebSocket avec `coder/websocket` ; `internal/hub/db.go` ouvre un `pgxpool` ; `migrations/embed.go` embarque les `.sql` avec `//go:embed` pour golang-migrate ; `.goreleaser.yaml` produit cinq binaires et une formule Homebrew ; le `Dockerfile` finit sur distroless. Compare avec ce que tu viens d'écrire : les mêmes briques, un cran plus loin.

**Caddy** ([github.com/caddyserver/caddy](https://github.com/caddyserver/caddy)), le serveur web avec HTTPS automatique. Lis `modules/caddyhttp/server.go` pour voir un `http.Server` de production, et `caddyconfig/` pour un système de configuration par modules enregistrés à l'init. C'est ce que devient `net/http` quand on lui demande tout.

**Prometheus** ([github.com/prometheus/prometheus](https://github.com/prometheus/prometheus)). Pas le serveur entier : le paquet `scrape/` (comment il interroge ton `/metrics`, avec quels timeouts, et ce qu'il fait d'une cible qui ne répond pas) et la bibliothèque `client_golang` que tu remplaceras à `FormatMetriques` le jour où tu voudras des histogrammes.

**Hugo** ([github.com/gohugoio/hugo](https://github.com/gohugoio/hugo)), le générateur de sites. Un exemple de programme en ligne de commande énorme (cobra, des centaines de flags) qui reste lisible, et une utilisation intensive de `text/template`, le paquet que ce cours a laissé de côté.

### 16.10 Tes prochains projets

Cinq idées, dans l'ordre de difficulté, pour que Go devienne ton outil et pas un souvenir de cours. Chacune tient dans un week-end de départ et peut grandir pendant un an.

1. **Un exporter maison.** Ce que tu surveilles aujourd'hui avec un script cron et un `grep` : l'état d'un onduleur, les certificats qui expirent, le lag d'une réplication PostgreSQL, la température d'un NAS. Une struct, un `/metrics`, un `Dockerfile` : c'est le labo 15 avec ta métrique à toi. Un dashboard Grafana par-dessus, et c'est en production.
2. **Une CLI pour ton NAS ou ton hyperviseur.** Une API REST en face (Synology, Proxmox, TrueNAS, Home Assistant), un client `net/http` du chapitre 13, cobra ou le paquet `flag`, et des commandes que tu tapes tous les jours. Compilé pour ton Mac et pour le Raspberry Pi d'un seul `make cross`.
3. **Un agent de supervision.** `sondes` qui grandit : historique en SQLite (chapitre 14), notifications (webhook, mail, Matrix), une page HTML embarquée avec `embed`, et un fichier de config rechargé sur SIGHUP. Tu as déjà 80 % du code.
4. **Un bot Matrix ou Slack.** Une boucle qui lit des événements sur un WebSocket ou du long-polling, un `select` qui multiplexe les commandes et l'arrêt, et des commandes qui interrogent tes outils. Le chapitre 11 en vrai, avec des humains à l'autre bout.
5. **Un plugin `kubectl`.** Un binaire nommé `kubectl-quelquechose` dans le `PATH` devient `kubectl quelquechose`. Avec `client-go` (lourd, mais c'est le SDK officiel), tu peux écrire en cent lignes l'outil que tu refaisais en `kubectl get ... -o json | jq` chaque semaine.

Le fil qui relie les cinq : ce que tu fais en shell avec des `curl | jq` et des variables non quotées, ou en Python avec un `venv` à installer sur chaque machine, écris-le en Go dès que ça doit tourner ailleurs que sur ton poste. Un binaire, un `scp`, c'est lancé.

### 16.11 Pour le labo

Le [Labo 16](../labs/16-projet/README.md) est le cahier des charges détaillé de `sondes` : le schéma JSON, les endpoints, les vingt `TODO` numérotés dans l'ordre des étapes de 16.3, les tests de chaque paquet, et une solution commentée ligne par ligne à ne lire qu'après. Quand `go test -race ./...` est vert, que `make lint` se tait et que Ctrl-C affiche `arrêté`, tu as écrit un service Go de production. Compare-le avec le `hello` du labo 01.

### À retenir

- Un service se construit de bas en haut : config, sondes, état, moteur, API, `main`, livraison. Chaque couche est testée avant la suivante, et `main` ne fait que brancher.
- Trois patrons qui tiennent jusqu'à des milliers de cibles : un worker pool borné, un état en mémoire derrière un `RWMutex` qui rend des copies, le `ServeMux` de la stdlib.
- Le contexte se passe de main en main jusqu'à la socket ; `http.Client` a toujours un `Timeout` ; `resp.Body` se ferme ; le `Ticker` s'arrête ; le channel d'erreur a un buffer de 1.
- `go test -race` est le juge. Une course détectée en test est une panne évitée en production.
- Les génériques se lisent partout et s'écrivent rarement : `slices`, `maps` et `cmp` couvrent l'essentiel, et une interface vaut souvent mieux qu'un paramètre de type.
- Le code de la stdlib est dans `$(go env GOROOT)/src` : `net/http/server.go` et `sync/waitgroup.go` expliquent mieux que n'importe quel article ce que tu utilises.
- Après ce cours : *100 Go Mistakes*, Effective Go, les proverbes, et surtout un projet à toi qui tourne quelque part.

---

← [15. Qualité, build, release, Docker, observabilité](15-qualite-build-release.md) · [Sommaire](../README.md)
