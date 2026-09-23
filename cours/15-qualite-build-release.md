# 15. Qualité, build, release, Docker, observabilité

*Go de zéro à la prod : chapitre 15 sur 16.* ← [14. Bases de données : database/sql, SQLite, PostgreSQL](14-bases-de-donnees.md) · [Sommaire](../README.md) · [16. Projet final et pour aller plus loin](16-projet-final-et-suite.md) →

Ton programme marche sur ta machine, lancé avec `go run`. Ce chapitre est celui qui le fait sortir de là : vérifié par des outils, versionné, compilé pour cinq systèmes, emballé dans une image de dix méga-octets, configuré par l'environnement, et capable de dire comment il va (logs, métriques, santé) une fois qu'il tourne chez quelqu'un d'autre. C'est le chapitre le plus DevOps du cours, et tout ce qu'il contient tient sur des outils que tu connais déjà d'un autre côté : `make`, Docker, Prometheus, GitHub Actions. Ce qui change, c'est à quel point Go rend chaque étape courte.

### 15.1 Trois outils, zéro débat : gofmt, go vet, golangci-lint

Tu connais `gofmt` depuis le chapitre 1 : il n'a pas d'options, tout le monde formate pareil. `go vet` est livré avec lui : une analyse statique gratuite qui attrape ce que le compilateur laisse passer. Et `golangci-lint` enchaîne une centaine d'analyseurs derrière un seul fichier de configuration. Un programme volontairement sale, pour voir ce que chacun attrape :

```go
func main() {
	var port int = 5432
	port = 5433 // la première valeur n'a jamais servi
	fmt.Printf("port %s\n", port) // %s avec un int
	os.Setenv("PGPORT", "5432")   // erreur ignorée
	fmt.Println("port", port)
}
```

```
$ gofmt -l .
$ go vet ./...
main.go:11:19: fmt.Printf format %s has arg port of wrong type int
$ golangci-lint run ./...
main.go:12:2: G104: Errors unhandled (gosec)
main.go:11:19: printf: fmt.Printf format %s has arg port of wrong type int (govet)
main.go:9:6: ineffectual assignment to port (ineffassign)
3 issues:
* gosec: 1
* govet: 1
* ineffassign: 1
```

`gofmt` ne dit rien : le fichier est formaté, ce n'est pas son sujet. `go vet` attrape le `%s` sur un entier, qui aurait affiché `port %!s(int=5433)` en production. `golangci-lint` reprend `vet` et ajoute l'affectation qui n'a jamais servi et l'erreur de `Setenv` ignorée. Aucun des trois ne dit que `var port int = 5432` s'écrit `port := 5432` : ça, c'est une revue de code.

La configuration tient en quinze lignes, dans `.golangci.yml` à la racine du module (format v2, celui de golangci-lint 2.x) :

```yaml
version: "2"
linters:
  default: standard        # errcheck, govet, ineffassign, staticcheck, unused
  enable:
    - gosec                # fichiers, exec, TLS, permissions
  exclusions:
    presets:
      - std-error-handling # ne signale plus fmt.Fprint*, defer x.Close()...
formatters:
  enable:
    - gofmt
```

Le preset `standard` est le bon point de départ : cinq analyseurs qui ne se trompent presque jamais. Le preset d'exclusion `std-error-handling` évite qu'`errcheck` te reproche chaque `defer f.Close()` et chaque `fmt.Fprintf(w, ...)`, où ignorer l'erreur est l'usage. Ajoute des analyseurs un par un quand tu en ressens le besoin, pas cinquante d'un coup.

**Venant de Python :** `gofmt` est `ruff format` sans aucune option, `go vet` est ce que `mypy` fait pour toi en plus du compilateur, et `golangci-lint` est `ruff check` avec ses règles. La différence : le compilateur Go a déjà refusé les imports inutilisés, les variables non lues et les types incompatibles avant que ces outils ne tournent. Il reste moins à attraper.

### 15.2 Le Makefile et la version dans le binaire

Un projet Go n'a pas besoin de `make` : `go build` suffit. Mais un `Makefile` de trente lignes fixe les options une fois pour toutes, et surtout **injecte la version** dans le binaire. Le mécanisme : une variable `string` du paquet `main`, écrasée par l'éditeur de liens avec `-ldflags "-X main.version=1.4.0"`.

```go
// Remplie à l'édition de liens : go build -ldflags "-X main.version=1.4.0"
var version = "dev"

func main() {
	fmt.Println("pingeur", version)
}
```

```
$ go run .
pingeur dev
$ CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=1.4.0" -o pingeur .
$ ./pingeur
pingeur 1.4.0
$ go version -m pingeur
pingeur: go1.27.1
	path	exemple.com/pingeur
	build	-trimpath=true
	build	CGO_ENABLED=0
	build	GOARCH=arm64
	build	GOOS=darwin
```

Quatre options à connaître par cœur : `-X paquet.variable=valeur` écrit dans une variable (elle doit être une `var string` initialisée par un littéral, pas une `const`) ; `-s -w` retire la table des symboles et les infos de débogage (30 % de taille en moins, les traces de panique restent lisibles) ; `-trimpath` enlève les chemins de ta machine du binaire ; `CGO_ENABLED=0` force un binaire vraiment statique, sans libc. Et `go version -m` relit tout ça dans n'importe quel binaire Go, même celui d'un autre : c'est ainsi qu'on sait avec quoi a été compilé le `prometheus` ou le `caddy` d'un serveur (`debug.ReadBuildInfo()` donne la même chose depuis le programme lui-même).

**Venant du C :** `-ldflags` est ce que tu passais à `ld` via `LDFLAGS`, et `-s -w` est `strip`. Mais il n'y a ni `-l`, ni `-I`, ni `configure`, ni `.o` intermédiaires à lister : le `Makefile` ne décrit plus la compilation, seulement les commandes qu'on veut retenir.

Le `Makefile` du labo, en résumé (la version vient du dernier tag git, `dev` s'il n'y en a pas) :

```makefile
BIN     := diskexporter
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
export CGO_ENABLED := 0

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BIN) .
test:
	go test -race -count=1 ./...
lint:
	@test -z "$$(gofmt -l .)" || (gofmt -l .; exit 1)
	go vet ./...
	golangci-lint run ./...
```

Deckhand a exactement ce `Makefile`, avec `-X github.com/stranix79/deckhand/internal/version.Version=$(VERSION)` : quand la variable n'est pas dans `main`, on donne son chemin d'import complet.

### 15.3 Compilation croisée en boucle, et embed

Le chapitre 1 a montré `GOOS=linux GOARCH=amd64 go build`. En production, on veut toutes les cibles d'un coup :

```
$ for os in linux darwin windows; do for arch in amd64 arm64; do
    ext=""; [ $os = windows ] && ext=.exe
    GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o dist/pingeur-$os-$arch$ext .
  done; done
$ ls -la dist
 1621298 pingeur-darwin-arm64
 1556640 pingeur-linux-amd64
 1573024 pingeur-linux-arm64
 1703424 pingeur-windows-amd64.exe
 ...
$ file dist/pingeur-linux-arm64
dist/pingeur-linux-arm64: ELF 64-bit LSB executable, ARM aarch64, statically linked, stripped
```

Six binaires en quelques secondes, depuis un Mac, sans rien installer. Le Raspberry Pi et le serveur x86 sont servis par la même commande. C'est la cible `cross` du `Makefile` du labo.

Et quand le programme a besoin de fichiers (une page HTML, un schéma SQL, un modèle de configuration), ils vont **dans** le binaire avec `embed` (chapitre 12) :

```go
//go:embed static
var assets embed.FS

sub, _ := fs.Sub(assets, "static")
mux.Handle("GET /", http.FileServerFS(sub))
```

Le dossier `static/` est lu à la compilation ; le binaire reste seul à livrer. Deckhand embarque ainsi son interface web et ses migrations SQL.

### 15.4 Reproductible : trimpath, go.sum, mod=readonly

Un build reproductible, c'est le même binaire octet pour octet quand on recompile la même source. Go le fait presque gratuitement :

```
$ CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=1.4.0" -o p1 .
$ CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=1.4.0" -o p2 .
$ shasum -a 256 p1 p2
98a7815aff7ee8fb2fd69c53579972578983bc4575c7ae058ac454cdcff1c1ff  p1
98a7815aff7ee8fb2fd69c53579972578983bc4575c7ae058ac454cdcff1c1ff  p2
```

Trois conditions. `-trimpath`, sinon `/Users/stranix/...` est dans le binaire et change d'une machine à l'autre. `go.sum` commité : il contient l'empreinte de chaque dépendance, et `go build` refuse un module dont l'empreinte ne correspond pas ; `go mod verify` revérifie tout le cache. Et en CI, `GOFLAGS=-mod=readonly` : `go build` échoue si `go.mod` aurait dû changer, au lieu de le modifier en silence.

**Piège :** la version de Go compte aussi. Deux versions du compilateur donnent deux binaires différents. Le `go.mod` fixe la version minimale ; `actions/setup-go` avec `go-version-file: go.mod` en CI, et une image `golang:1.27` précise dans le `Dockerfile`, pas `golang:latest`.

### 15.5 goreleaser : de `git tag` à la page Releases

goreleaser lit un `.goreleaser.yaml`, compile toutes les cibles, fabrique les archives, calcule les sommes de contrôle, écrit le changelog depuis les commits et crée la GitHub Release. La configuration minimale :

```yaml
version: 2
project_name: diskexporter
builds:
  - main: .
    binary: diskexporter
    env: [CGO_ENABLED=0]
    flags: [-trimpath]
    ldflags: [-s -w -X main.version={{.Version}}]
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
archives:
  - formats: [tar.gz]
checksum:
  name_template: checksums.txt
release:
  github: {owner: stranix79, name: cours-go}
```

`{{.Version}}` est le tag git sans son `v`. Une répétition locale, sans tag ni jeton, pour voir ce que ça produit :

```
$ goreleaser release --snapshot --clean
    • building    target=linux_arm64_v8.0
    • building    target=darwin_arm64_v8.0
  • archiving     name=dist/diskexporter_0.0.0-SNAPSHOT-none_linux_amd64.tar.gz
$ ls dist
checksums.txt  diskexporter_0.0.0-SNAPSHOT-none_darwin_arm64.tar.gz  diskexporter_0.0.0-SNAPSHOT-none_linux_amd64.tar.gz ...
$ dist/diskexporter_darwin_arm64_v8.0/diskexporter -version
diskexporter 0.0.0-SNAPSHOT-none
```

Quatre secondes pour cinq cibles. La vraie release, c'est `git tag v1.0.0 && git push --tags`, et le workflow CI (section 15.11) lance `goreleaser release` avec le jeton que GitHub Actions lui fournit. Le `.goreleaser.yaml` de Deckhand est celui-là, à trois lignes près (`-X .../internal/version.Version={{.Version}}`, le changelog depuis GitHub). Pour la formule Homebrew, deux écoles : une section `brews` dans goreleaser, qui pousse la formule dans ton dépôt `homebrew-tap` à chaque release ; ou un script à toi, comme `scripts/update-tap.sh` de Deckhand, qui écrit une formule compilant depuis le tarball du tag. Dans les deux cas, `brew install stranix79/tap/deckhand` marche, et personne ne télécharge un `.tar.gz` à la main deux fois.

### 15.6 Docker : dix méga-octets, sans shell, sans root

Un service Go se livre en image, et l'image ne contient que le binaire. Le `Dockerfile` multi-étapes du labo :

```dockerfile
FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download                  # couche en cache tant que go.mod ne change pas
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/diskexporter .

FROM scratch                         # rien : ni shell, ni libc, ni /etc/passwd
COPY --from=build /out/diskexporter /diskexporter
USER 65534:65534                     # nobody, en numérique puisqu'il n'y a pas de /etc/passwd
EXPOSE 9101
ENTRYPOINT ["/diskexporter"]
```

La première étape pèse 800 Mo et disparaît. L'image finale pèse la taille du binaire, six à dix méga-octets. Pas de shell : `docker exec -it ... sh` ne marche pas, et c'est une qualité, un attaquant qui entre n'a rien à exécuter. Deckhand préfère `gcr.io/distroless/static-debian12:nonroot` à `scratch` : même principe, mais avec `/etc/passwd`, les certificats racines et les fuseaux horaires déjà en place, et un utilisateur `nonroot` nommé.

**Piège :** `scratch` n'a pas de certificats. Un client HTTPS y échoue avec `x509: certificate signed by unknown authority`. Deux remèdes : `COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/` dans le `Dockerfile`, ou `import _ "golang.org/x/crypto/x509roots/fallback"` dans le code, qui embarque les racines Mozilla. Même chose pour les fuseaux : `import _ "time/tzdata"` embarque la base des zones, sinon `time.LoadLocation("Europe/Brussels")` échoue.

**Venant de Python :** l'image Python la plus fine, `python:3.13-slim`, fait 150 Mo avant ton code, et il faut choisir entre `slim` et `alpine` selon les wheels. Ici la question ne se pose pas : le binaire est statique, `scratch` suffit, et la taille est divisée par vingt.

Avec une base de données à côté, `docker compose` orchestre les deux, en attendant que PostgreSQL soit vraiment prêt :

```yaml
services:
  db:
    image: postgres:17
    environment: {POSTGRES_USER: sondes, POSTGRES_PASSWORD: sondes, POSTGRES_DB: sondes}
    healthcheck: {test: ["CMD-SHELL", "pg_isready -U sondes"], interval: 5s, retries: 10}
  app:
    build: .
    environment: {SONDES_DSN: "postgres://sondes:sondes@db:5432/sondes?sslmode=disable"}
    ports: ["8080:8080"]
    depends_on:
      db: {condition: service_healthy}
```

`depends_on` seul attend que le conteneur démarre, pas que PostgreSQL accepte des connexions ; `condition: service_healthy` avec `pg_isready` attend le vrai moment. Ton service, lui, doit quand même réessayer `Ping` quelques secondes au démarrage (chapitre 14) : en production, la base redémarre aussi.

### 15.7 La configuration vient de l'environnement

Les [douze facteurs](https://12factor.net/fr/) le disent depuis 2011 : ce qui change entre ta machine et la production (adresse d'écoute, DSN, niveau de log) vient de variables d'environnement, jamais d'un fichier commité. En Go, c'est `os.Getenv`, ou `os.LookupEnv` quand « absente » et « vide » ne veulent pas dire la même chose :

```go
type Config struct {
	Addr       string
	DSN        string
	Intervalle time.Duration
}

func envOr(cle, defaut string) string {
	if v, ok := os.LookupEnv(cle); ok && v != "" {
		return v
	}
	return defaut
}

func charger() (Config, error) {
	cfg := Config{Addr: envOr("SONDES_ADDR", ":8080"), DSN: os.Getenv("SONDES_DSN")}
	if cfg.DSN == "" {
		return cfg, fmt.Errorf("SONDES_DSN est obligatoire")
	}
	var err error
	if cfg.Intervalle, err = time.ParseDuration(envOr("SONDES_INTERVALLE", "30s")); err != nil {
		return cfg, fmt.Errorf("SONDES_INTERVALLE : %w", err)
	}
	return cfg, nil
}
```

```
$ go run .
config : SONDES_DSN est obligatoire
exit status 2
$ SONDES_DSN=postgres://sondes@db:5432/sondes SONDES_INTERVALLE=1m go run .
{Addr::8080 DSN:postgres://sondes@db:5432/sondes Intervalle:1m0s}
```

Tout est lu et validé une fois, au démarrage, et le programme refuse de partir avec une configuration fausse : mieux qu'un plantage deux heures plus tard sur la première requête. Un préfixe par programme (`SONDES_`, `DECKHAND_`) évite les collisions. Les secrets (mots de passe, jetons) suivent le même chemin et ne s'affichent jamais, ni dans les logs ni dans `%+v` d'une struct : si tu journalises la config, masque le DSN d'abord.

### 15.8 Journaliser en JSON avec slog

`log/slog` (Go 1.21) est le journal structuré de la bibliothèque standard : un message, puis des paires clé-valeur. Le même appel produit du texte pour ton terminal (`slog.NewTextHandler`, le défaut) ou du JSON pour Loki (`slog.NewJSONHandler`) :

```go
slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))
slog.Info("sonde terminée", "cible", "db-01:5432", "duree", 12*time.Millisecond, "ok", true)

log := slog.With("composant", "worker", "id", 3) // attributs répétés sur chaque ligne
log.Warn("cible injoignable", "cible", "web-02:443", "err", errors.New("connection refused"))

slog.Info("requête", slog.Group("http", "methode", "GET", "chemin", "/metrics", "statut", 200))
```

```
{"time":"2026-09-23T12:30:43.185088+02:00","level":"INFO","msg":"sonde terminée","cible":"db-01:5432","duree":12000000,"ok":true}
{"time":"2026-09-23T12:30:43.185105+02:00","level":"WARN","msg":"cible injoignable","composant":"worker","id":3,"cible":"web-02:443","err":"connection refused"}
{"time":"2026-09-23T12:30:43.185113+02:00","level":"INFO","msg":"requête","http":{"methode":"GET","chemin":"/metrics","statut":200}}
```

Sur la sortie standard, toujours : Docker, systemd et Kubernetes la collectent, ton programme n'ouvre pas de fichier de log. Deux détails à voir dans la sortie : le `time.Duration` sort en nanosecondes en JSON (`12000000`, là où le handler texte écrirait `12ms`), donc filtre sur `duree > 1e9` dans Loki ou convertis en millisecondes toi-même ; et `slog.With` crée un logger enfant qu'on passe à un composant, si bien que chaque ligne du worker porte `composant=worker` sans le répéter à la main. Une fois qu'un champ est une clé JSON, `{composant="worker"} | json | duree > 1000000` dans Grafana remplace un `grep` fragile.

### 15.9 Métriques Prometheus : à la main, puis avec client_golang

Le format d'exposition de Prometheus est du texte, et il est assez simple pour l'écrire soi-même. Une famille de métriques, c'est une ligne `# HELP`, une ligne `# TYPE`, puis une ligne par jeu de labels :

```go
var requetes atomic.Int64 // compteur sûr entre goroutines, sans mutex

func metrics(w http.ResponseWriter, r *http.Request) {
	requetes.Add(1)
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintln(w, "# HELP sondes_requetes_total Requêtes reçues depuis le démarrage.")
	fmt.Fprintln(w, "# TYPE sondes_requetes_total counter")
	fmt.Fprintf(w, "sondes_requetes_total{chemin=\"/metrics\"} %d\n", requetes.Load())
	fmt.Fprintln(w, "# HELP go_goroutines Goroutines vivantes.")
	fmt.Fprintln(w, "# TYPE go_goroutines gauge")
	fmt.Fprintf(w, "go_goroutines %d\n", runtime.NumGoroutine())
}
// dans main : http.HandleFunc("GET /metrics", metrics)
```

```
$ curl -s -i 127.0.0.1:9102/metrics
HTTP/1.1 200 OK
Content-Type: text/plain; version=0.0.4; charset=utf-8
Content-Length: 234

# HELP sondes_requetes_total Requêtes reçues depuis le démarrage.
# TYPE sondes_requetes_total counter
sondes_requetes_total{chemin="/metrics"} 2
# HELP go_goroutines Goroutines vivantes.
# TYPE go_goroutines gauge
go_goroutines 3
```

Vingt lignes, et Prometheus peut déjà le *scraper*. Les règles à respecter : un `counter` ne fait que monter (`_total` en suffixe), une `gauge` monte et descend, les unités sont dans le nom (`_bytes`, `_seconds`), les valeurs de labels sont entre guillemets doubles et échappées (`\"`, `\\`, `\n`). C'est ce que fait le labo, avec une interface devant `syscall.Statfs` pour que les tests aient des chiffres fixes.

Quand il faut des histogrammes, des labels dynamiques, ou simplement ne pas réinventer l'échappement, la bibliothèque officielle fait tout ça en cinq lignes :

```go
var sondesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "sondes_total", Help: "Sondes exécutées, par résultat.",
}, []string{"resultat"})

sondesTotal.WithLabelValues("ok").Add(41)
sondesTotal.WithLabelValues("echec").Inc()
http.Handle("GET /metrics", promhttp.Handler())
```

```
$ curl -s 127.0.0.1:9103/metrics | grep -E '^(sondes_total|go_goroutines|process_resident)'
go_goroutines 6
process_resident_memory_bytes 1.4860288e+07
sondes_total{resultat="echec"} 1
sondes_total{resultat="ok"} 41
```

Après `go get github.com/prometheus/client_golang@latest`, la page fait cent vingt-six lignes pour deux valeurs à toi : `promhttp` ajoute les métriques du runtime (`go_goroutines`, `go_memstats_*`, `go_gc_*`) et du processus (`process_cpu_seconds_total`, `process_resident_memory_bytes`). C'est ce que Deckhand expose, et c'est ce que tu veux dès que le service compte. Le format à la main reste utile pour un exporter minuscule sans dépendance, ou pour comprendre ce que tu lis dans un `curl`.

### 15.10 Santé, arrêt propre, pprof

**`/healthz` et `/readyz`** ne disent pas la même chose. La première (*liveness*) répond « le processus est vivant » : si elle échoue, l'orchestrateur redémarre le conteneur. La seconde (*readiness*) répond « je peux servir » : cache chargé, base jointe ; si elle échoue, l'orchestrateur cesse d'envoyer du trafic mais ne redémarre rien. Confondre les deux, c'est faire redémarrer un service en boucle parce que sa base de données est en maintenance.

```go
var pret atomic.Bool

mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok\n")) })
mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
	if !pret.Load() {
		http.Error(w, "pas prêt : chargement initial", http.StatusServiceUnavailable)
		return
	}
	w.Write([]byte("ready\n"))
})
```

```
$ curl -s -i 127.0.0.1:8080/healthz | head -1
HTTP/1.1 200 OK
$ curl -s -i 127.0.0.1:8080/readyz | sed -n '1p;$p'
HTTP/1.1 503 Service Unavailable
pas prêt : chargement initial
$ sleep 2; curl -s -i 127.0.0.1:8080/readyz | sed -n '1p;$p'
HTTP/1.1 200 OK
ready
```

**L'arrêt propre** est celui du chapitre 13, et il est dans `run` du labo : `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)` annule le contexte quand Docker ou systemd envoient SIGTERM, `srv.Shutdown(ctxAvecDelai)` ferme le port, laisse finir les requêtes en cours, et rend la main. Sans ça, `docker stop` attend dix secondes puis tue le processus au milieu d'une écriture. Le test `TestRunArretPropre` du labo vérifie ce comportement sans signal, juste en annulant le contexte : c'est la raison d'écrire `run(ctx, addr) error` plutôt que de tout mettre dans `main`.

**pprof** est le profileur intégré. Un import vide et un serveur sur un port interne, et tu peux regarder la mémoire et les goroutines d'un service en production, sans le redémarrer :

```go
import _ "net/http/pprof" // enregistre /debug/pprof/* sur http.DefaultServeMux

go func() { log.Println(http.ListenAndServe("127.0.0.1:6060", nil)) }()
```

```
$ curl -s '127.0.0.1:6060/debug/pprof/goroutine?debug=1' | head -1
goroutine profile: total 4
```

`go tool pprof -top http://127.0.0.1:6060/debug/pprof/heap` télécharge le profil mémoire et affiche les fonctions qui allouent le plus ; `-http=:8081` ouvre le graphe dans le navigateur. Un exporter qui fuit se voit là, pas dans les logs.

**Piège :** `net/http/pprof` s'enregistre sur `http.DefaultServeMux`. Si ton service utilise `http.ListenAndServe(addr, nil)`, le profileur est exposé sur le port public, avec la liste de tes goroutines et un moyen de saturer ton processeur. Toujours un `ServeMux` à toi pour le trafic, et `DefaultServeMux` sur `127.0.0.1:6060` seulement, joignable par `ssh -L` ou depuis le pod.

### 15.11 CI : GitHub Actions en vingt lignes

```yaml
name: CI
on:
  push: {branches: [main], tags: ["v*"]}
  pull_request:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: {go-version-file: go.mod}
      - run: go vet ./...
      - run: go test -race -count=1 ./...
      - uses: golangci/golangci-lint-action@v8
        with: {version: latest}
```

Pas de matrice de versions comme en Python : la promesse de compatibilité de Go fait qu'on teste avec la version du `go.mod` et c'est tout. `-race` en CI, toujours : c'est là qu'on a le temps. Le fichier complet du labo (`.github/workflows/ci.yml`) ajoute un job `release` qui ne tourne que sur un tag `v*` et lance `goreleaser/goreleaser-action` avec le `GITHUB_TOKEN` fourni par Actions, rien à configurer. C'est le modèle à copier à la racine de ton propre dépôt.

### 15.12 Pour le labo

Le [labo 15](../labs/15-release/README.md) te fait écrire `diskexporter`, un exporter Prometheus de l'espace disque : `/metrics` au format texte écrit à la main, `/healthz`, version injectée, arrêt propre testé sans signal, et tout l'emballage : `Makefile`, `Dockerfile` vers `scratch`, `.goreleaser.yaml`, `.golangci.yml`, workflow CI. La vérification n'exige que `go test` ; Docker et goreleaser sont là si tu les as.

### À retenir

- `gofmt`, `go vet`, `golangci-lint` avec le preset `standard` : trois commandes dans la cible `lint` du `Makefile`, lancées avant chaque commit et en CI.
- La version entre dans le binaire par `-ldflags "-X main.version=..."` ; `go version -m` relit le build de n'importe quel binaire Go.
- `CGO_ENABLED=0 go build -trimpath -ldflags "-s -w"` donne un binaire statique, petit et reproductible ; une boucle sur `GOOS`/`GOARCH` couvre toutes tes machines.
- goreleaser transforme `git tag` en GitHub Release avec archives, sommes de contrôle et formule Homebrew ; `--snapshot` pour répéter en local.
- Une image Docker Go, c'est `golang:1.27` pour compiler puis `scratch` ou `distroless` pour tourner : dix méga-octets, pas de shell, pas de root ; pense aux certificats et aux fuseaux.
- La configuration vient de l'environnement, validée au démarrage ; les logs vont sur stdout en JSON avec `slog` ; les métriques sur `/metrics` (à la main ou `promhttp`).
- `/healthz` dit « vivant », `/readyz` dit « prêt » ; `Shutdown` sur SIGTERM ; pprof sur un port interne seulement.
- La CI tient en vingt lignes : `setup-go` avec la version du `go.mod`, `go test -race`, `golangci-lint`, goreleaser sur les tags.

---

← [14. Bases de données : database/sql, SQLite, PostgreSQL](14-bases-de-donnees.md) · [Sommaire](../README.md) · [16. Projet final et pour aller plus loin](16-projet-final-et-suite.md) →
