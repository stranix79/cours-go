# Labo 16 : sondes, le projet final

**Chapitre** : [16. Projet final et pour aller plus loin](../../cours/16-projet-final-et-suite.md)

## Objectif
Écrire un service de supervision complet, `sondes` : il lit une liste de cibles (HTTP et TCP) dans un fichier JSON, les vérifie en parallèle à intervalle régulier, garde le dernier état de chacune en mémoire, et expose le tout en JSON pour les humains et au format Prometheus pour Grafana. Il journalise en JSON, s'arrête proprement sur Ctrl-C ou `docker stop`, se compile en un binaire statique et s'emballe dans une image Docker de quelques méga-octets. C'est un Uptime Kuma de poche, et c'est tout le cours dans un seul programme : config et JSON (ch. 12), interfaces (ch. 7), erreurs enveloppées (ch. 8), goroutines, channels, context et mutex (ch. 11), HTTP (ch. 13), build et observabilité (ch. 15).

Ce README est le cahier des charges. Les tests fournis sont la spécification exécutable : quand ils passent tous, c'est fini.

## Consignes

### L'architecture (fournie)

```
labs/16-projet/
├── cmd/sondes/main.go          point d'entrée : options, logger, signal, run()
├── internal/config/config.go   lecture + validation du JSON       (TODO 1 à 3)
├── internal/sonde/sonde.go     Resultat, Sondeur, SonderHTTP/TCP  (TODO 4 à 7)
├── internal/sonde/etat.go      Etat en mémoire, mutex             (TODO 8 à 11)
├── internal/sonde/moteur.go    Tour (worker pool), Boucle (ticker) (TODO 12, 13)
├── internal/api/api.go         routes, handlers, middleware       (TODO 14 à 18)
├── internal/api/metrics.go     format Prometheus                  (TODO 19)
├── sondes.json                 config d'exemple
├── Makefile, Dockerfile, .golangci.yml
└── solution/                   la même arborescence, complète et commentée
```

Le squelette compile tel quel (`go vet ./...` passe) et chaque paquet a ses tests. Les dépendances vont dans un seul sens : `api` et `sonde` connaissent `config`, `api` connaît `sonde`, `main` connaît les trois, et rien ne connaît `main`.

### Le fichier de configuration

```json
{
  "ecoute": ":8080",
  "intervalle": "5s",
  "timeout": "2s",
  "workers": 4,
  "cibles": [
    {"nom": "web-local", "type": "http", "url": "http://127.0.0.1:8000/"},
    {"nom": "ssh-local", "type": "tcp", "adresse": "127.0.0.1:22"}
  ]
}
```

| Champ | Type | Défaut | Règle |
|---|---|---|---|
| `ecoute` | chaîne | `":8080"` | adresse d'écoute de l'API |
| `intervalle` | durée Go (`"30s"`, `"1m"`) | `30s` | > 0 |
| `timeout` | durée Go | `5s` | > 0 et ≤ `intervalle` |
| `workers` | entier | `4` | ≥ 1 : nombre maximal de sondes en parallèle |
| `cibles[].nom` | chaîne | | obligatoire, non vide, unique |
| `cibles[].type` | `"http"` ou `"tcp"` | | autre valeur refusée |
| `cibles[].url` | chaîne | | si `http` : `http://` ou `https://` avec un hôte |
| `cibles[].adresse` | chaîne | | si `tcp` : `hôte:port` |

Un champ inconnu (`"intervale"`) est une erreur, pas un silence. Aucune cible est une erreur. Toutes les erreurs de validation enveloppent `config.ErrInvalide`.

### Les endpoints

| Route | Réponse |
|---|---|
| `GET /healthz` | `200`, `{"ok":true,"version":"..."}` |
| `GET /api/status` | `200`, un `api.Statut` : `version`, `demarre`, `verifications`, `cibles[]` (`nom`, `type`, `up`, `latence_ms`, `erreur` si down, `quand`), cibles triées par nom, `[]` et pas `null` si vide |
| `GET /metrics` | `200`, `text/plain; version=0.0.4`, les trois métriques ci-dessous |
| autre chemin | `404` ; autre méthode sur une route connue : `405` (c'est le ServeMux de Go 1.22 qui le fait) |

Les métriques, dans cet ordre, avec `# HELP` et `# TYPE` :

```
sonde_up{cible="NOM",type="TYPE"} 0|1                    gauge, une ligne par cible
sonde_latence_secondes{cible="NOM",type="TYPE"} 0.0018   gauge, une ligne par cible
sonde_verifications_total N                              counter
```

Chaque requête est journalisée par le middleware : `methode`, `chemin`, `statut`, `duree_ms`, `client`.

### Les étapes, de bas en haut

1. **`internal/config`** (TODO 1 à 3). `Duree.UnmarshalText` et `MarshalText` avec `time.ParseDuration` ; `Parser` qui décode par-dessus `Defaut()` avec `DisallowUnknownFields` puis appelle `Valider` ; `Valider` qui applique le tableau ci-dessus dans l'ordre et s'arrête à la première erreur. `go test ./internal/config` doit passer avant de continuer.
2. **`internal/sonde/sonde.go`** (TODO 4 à 7). `NouveauSondeur` donne un `http.Client{Timeout: timeout}` ; `SonderHTTP` fait un GET avec `http.NewRequestWithContext`, ferme et vide le corps, renvoie une erreur `statut HTTP NNN` à partir de 400 ; `SonderTCP` fait `net.Dialer{Timeout}.DialContext` puis `Close` ; `Sonder` aiguille, chronomètre, et remplit le `Resultat` (une cible injoignable est un résultat down, jamais une erreur du programme).
3. **`internal/sonde/etat.go`** (TODO 8 à 11). La map créée dans `NouvelEtat` ; `Enregistrer` sous `Lock`, qui renvoie vrai à la première vue et à chaque bascule up/down ; `Resultats` sous `RLock`, qui renvoie une copie triée par nom ; `Total`.
4. **`internal/sonde/moteur.go`** (TODO 12, 13). `Tour` : le worker pool du chapitre 11, `Workers` goroutines sur un channel non bufferisé, `WaitGroup`, distribution dans un `select` avec `ctx.Done()`, journalisation des changements d'état seulement. `Boucle` : un tour tout de suite, puis un par tick, `defer ticker.Stop()`, retour quand `ctx` est annulé. Les tests vérifient que 6 sondes de 20 ms sur 2 workers prennent environ 60 ms (ni 20, ni 120), et que `Boucle` s'arrête quand on le lui demande.
5. **`internal/api`** (TODO 14 à 19). Les routes en syntaxe `"GET /chemin"`, les trois handlers, `FormatMetriques` à l'octet près (le test compare la sortie complète, avec un nom de cible qui contient un guillemet et une barre oblique inverse), et `Journaliser` avec un `ResponseWriter` enveloppé pour capturer le statut.
6. **`cmd/sondes/main.go`** (TODO 20). `run` assemble tout, lance `Boucle` dans une goroutine et `ListenAndServe` dans une autre, attend le premier des deux événements (erreur du serveur ou annulation du contexte), puis `Shutdown` avec un contexte neuf de 5 s, `wg.Wait()`, et journalise `arrêté`. Le processus doit rendre la main en moins d'une seconde après Ctrl-C.
7. **Livraison.** `make build` (version injectée par `-ldflags -X main.version`), `make cross` (quatre binaires), `make lint` (les avertissements `unused` du squelette disparaissent quand tout est écrit), `make docker` si tu as Docker (image `scratch`, utilisateur non root, une dizaine de Mo).

### Cas limites à respecter (la plupart sont testés)

- Un `http.Client` sans `Timeout` n'est pas accepté : une sonde vers un serveur qui ne répond jamais doit revenir en `timeout`, pas jamais.
- `Tour` avec un contexte déjà annulé rend la main tout de suite, même si les sondes durent une seconde.
- `Etat.Resultats` renvoie une copie : modifier la slice renvoyée ne change pas l'état.
- Le log de changement d'état ne contient `erreur` que si la cible est down.
- `/api/status` sur un état vide donne `"cibles":[]`.
- `Journaliser(nil, h)` renvoie `h` tel quel.

## Comment lancer
```bash
go vet ./... && gofmt -l .              # le squelette compile, rien à formater
go test ./internal/...                  # TES paquets : rouge tant que les TODO ne sont pas faits
go test ./solution/...                  # la solution : vert
go test -race ./...                     # tout, avec le détecteur de course
python3 -m http.server 8000 --bind 127.0.0.1 &   # une cible HTTP locale pour sondes.json
make run                                # ou : go run ./cmd/sondes -config sondes.json
make run-solution                       # la solution, mêmes options
curl -s localhost:8080/healthz
curl -s localhost:8080/api/status | python3 -m json.tool
curl -s localhost:8080/metrics
make cross && ls bin/                   # linux/amd64, linux/arm64, darwin/arm64, windows/amd64
```

Sans Docker sur ta machine, `make docker` n'est pas requis : lis le `Dockerfile`, il est commenté.

## Sortie attendue
Avec `python3 -m http.server 8000` qui tourne, ni SSH ni PostgreSQL en écoute sur la machine, et le service lancé avec `-ecoute 127.0.0.1:8090` (le port 8080 était pris chez moi ; l'option l'emporte sur la config). Les horodatages, les latences et le port client varient ; la forme, non. La ligne `Date:` de curl est omise.

```
$ curl -s -i localhost:8090/healthz
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Content-Length: 28

{"ok":true,"version":"dev"}

$ curl -s localhost:8090/api/status | python3 -m json.tool
{
    "version": "dev",
    "demarre": "2026-09-23T12:36:41.707072+02:00",
    "verifications": 3,
    "cibles": [
        {
            "nom": "postgres-local",
            "type": "tcp",
            "up": false,
            "latence_ms": 1.029625,
            "erreur": "dial tcp 127.0.0.1:5432: connect: connection refused",
            "quand": "2026-09-23T12:36:41.707128+02:00"
        },
        {
            "nom": "ssh-local",
            "type": "tcp",
            "up": false,
            "latence_ms": 1.004209,
            "erreur": "dial tcp 127.0.0.1:22: connect: connection refused",
            "quand": "2026-09-23T12:36:41.707137+02:00"
        },
        {
            "nom": "web-local",
            "type": "http",
            "up": true,
            "latence_ms": 2.573958,
            "quand": "2026-09-23T12:36:41.707074+02:00"
        }
    ]
}

$ curl -s localhost:8090/metrics
# HELP sonde_up 1 si la cible a répondu à la dernière vérification, 0 sinon.
# TYPE sonde_up gauge
sonde_up{cible="postgres-local",type="tcp"} 0
sonde_up{cible="ssh-local",type="tcp"} 0
sonde_up{cible="web-local",type="http"} 1
# HELP sonde_latence_secondes Durée de la dernière vérification, en secondes.
# TYPE sonde_latence_secondes gauge
sonde_latence_secondes{cible="postgres-local",type="tcp"} 0.001029625
sonde_latence_secondes{cible="ssh-local",type="tcp"} 0.001004209
sonde_latence_secondes{cible="web-local",type="http"} 0.002573958
# HELP sonde_verifications_total Nombre de vérifications effectuées depuis le démarrage.
# TYPE sonde_verifications_total counter
sonde_verifications_total 3

$ curl -s -i -X POST localhost:8090/api/status | head -1
HTTP/1.1 405 Method Not Allowed
```

Le journal du service pendant ce temps, puis quand on arrête `http.server` (au tour suivant, `web-local` passe down) et qu'on fait Ctrl-C :

```
{"time":"2026-09-23T12:36:41.707954+02:00","level":"INFO","msg":"démarrage","version":"dev","ecoute":"127.0.0.1:8090","cibles":3,"intervalle":"5s","workers":4}
{"time":"2026-09-23T12:36:41.708148+02:00","level":"INFO","msg":"changement d'état","cible":"ssh-local","up":false,"latence_ms":1.004209,"erreur":"dial tcp 127.0.0.1:22: connect: connection refused"}
{"time":"2026-09-23T12:36:41.708159+02:00","level":"INFO","msg":"changement d'état","cible":"postgres-local","up":false,"latence_ms":1.029625,"erreur":"dial tcp 127.0.0.1:5432: connect: connection refused"}
{"time":"2026-09-23T12:36:41.709649+02:00","level":"INFO","msg":"changement d'état","cible":"web-local","up":true,"latence_ms":2.573958}
{"time":"2026-09-23T12:36:42.931442+02:00","level":"INFO","msg":"requête","methode":"GET","chemin":"/healthz","statut":200,"duree_ms":0.069209,"client":"127.0.0.1:50132"}
{"time":"2026-09-23T12:36:42.94292+02:00","level":"INFO","msg":"requête","methode":"GET","chemin":"/api/status","statut":200,"duree_ms":0.079167,"client":"127.0.0.1:50133"}
{"time":"2026-09-23T12:36:42.953525+02:00","level":"INFO","msg":"requête","methode":"GET","chemin":"/metrics","statut":200,"duree_ms":0.081791,"client":"127.0.0.1:50134"}
{"time":"2026-09-23T12:36:42.964271+02:00","level":"INFO","msg":"requête","methode":"POST","chemin":"/api/status","statut":405,"duree_ms":0.014625,"client":"127.0.0.1:50135"}
{"time":"2026-09-23T12:36:51.731233+02:00","level":"INFO","msg":"changement d'état","cible":"web-local","up":false,"latence_ms":0.885208,"erreur":"Get \"http://127.0.0.1:8000/\": dial tcp 127.0.0.1:8000: connect: connection refused"}
^C
{"time":"2026-09-23T12:36:53.557858+02:00","level":"INFO","msg":"arrêt demandé"}
{"time":"2026-09-23T12:36:53.558058+02:00","level":"INFO","msg":"arrêté","verifications":9}
```

Trois choses à vérifier dans ce journal : les trois premières sondes ont des horodatages à quelques microsecondes d'écart (elles ont tourné en parallèle) ; `web-local` n'apparaît qu'aux changements d'état, pas à chaque tour ; entre `arrêt demandé` et `arrêté`, il s'écoule moins d'une milliseconde, et le processus a rendu la main.

## Pour aller plus loin
- Ajoute un type de cible `dns` (résoudre un nom avec `net.DefaultResolver.LookupHost` et le contexte) : une constante dans `config`, un cas dans `valider`, un cas dans `Sonder`, et tout le reste marche sans changement. C'est ce que l'interface `Sondeur` et le `switch` sur le type t'achètent.
- Remplace `FormatMetriques` par `github.com/prometheus/client_golang` : un `GaugeVec` pour `sonde_up`, un `CounterVec` pour le total, `promhttp.Handler()` sur `/metrics`. Compare le nombre de lignes et la taille du binaire.
- Rends l'état persistant : une table `resultats` dans SQLite avec le paquet `database/sql` du chapitre 14, écrite par `Enregistrer`, et un endpoint `GET /api/historique/{cible}` qui renvoie les cent derniers résultats. L'interface `Sondeur` ne change pas, le `Moteur` non plus.
