# Labo 15 : diskexporter, un exporter Prometheus prêt à livrer

**Chapitre** : [15. Qualité, build, release, Docker, observabilité](../../cours/15-qualite-build-release.md)

## Objectif
Écrire un petit exporter Prometheus qui expose l'espace disque des points de montage sur `/metrics`, au format texte écrit à la main, avec un `/healthz`, une version injectée à la compilation, un arrêt propre sur SIGTERM, et tout ce qu'il faut autour pour le livrer : `Makefile`, `Dockerfile` multi-étapes vers `scratch`, `.goreleaser.yaml`, `.golangci.yml`, et un workflow GitHub Actions d'exemple. C'est le squelette de tous tes futurs outils d'infra en Go.

Le code est découpé en trois fichiers : `sonde.go` (l'interface `Sonde` et sa fausse implémentation, déjà écrites), `sonde_unix.go` (la vraie sonde, `syscall.Statfs`, à compléter), `metrics.go` (collecte et format d'exposition, à compléter), `main.go` (options, routeur, `run`, à compléter). Le fichier `sonde_autres.go` fournit une sonde vide pour Windows, pour que `GOOS=windows go build` passe : ne le touche pas.

## Consignes
1. Lis `sonde.go` et `metrics_test.go`. Le test `TestFormaterMetriques` contient la page `/metrics` complète attendue : c'est ta spécification. Lance `go test .` : tout échoue, c'est normal.
2. **TODO 1**, `decouperPoints(liste string) []string` dans `main.go` : coupe sur la virgule, enlève les espaces autour de chaque morceau, ignore les morceaux vides. `"/, /var ,/data"` donne `["/", "/var", "/data"]` ; `""` donne `nil`.
3. **TODO 2**, `collecter(s Sonde, points []string) ([]Mesure, error)` dans `metrics.go` : appelle `s.Statfs` pour chaque point. Un point en erreur n'arrête pas la boucle : on accumule les erreurs et on renvoie `errors.Join(erreurs...)` avec les mesures qui ont réussi. Si tout réussit, l'erreur est `nil`.
4. **TODO 3**, `echapperLabel` et `ratioUtilise` : la barre oblique inverse, le guillemet et le saut de ligne s'échappent (`strings.NewReplacer`) ; le ratio vaut `(Total - Libre) / Total` en `float64`, et `0` si `Total` vaut `0`.
5. **TODO 4**, `formaterMetriques(w io.Writer, version string, mesures []Mesure, nbErreurs int)` : écris la page dans l'ordre du test : `diskexporter_build_info`, `diskexporter_scrape_errors`, puis les familles `disk_total_bytes`, `disk_free_bytes`, `disk_used_ratio`, chacune avec `# HELP`, `# TYPE ... gauge`, et une ligne par mesure avec le label `mountpoint`. Le ratio est formaté avec `strconv.FormatFloat(x, 'f', 4, 64)`. Écris une closure `famille(nom, aide, valeur)` pour ne pas répéter la boucle trois fois.
6. **TODO 5**, `handlerMetrics`, `handlerHealthz` et `nouveauRouteur` : le handler collecte, formate dans un `bytes.Buffer`, pose `Content-Type: text/plain; version=0.0.4; charset=utf-8`, puis écrit. En cas d'erreur de collecte, journalise avec `slog.Warn` et réponds quand même (`nbErreurs = len(points) - len(mesures)`). `/healthz` répond `ok\n`. Le routeur utilise les motifs `"GET /metrics"` et `"GET /healthz"` : un `POST` doit recevoir 405, une route inconnue 404.
7. **TODO 6**, `run(ctx, addr, s, points) error` : un `http.Server` avec `ReadHeaderTimeout`, `ListenAndServe` dans une goroutine qui envoie son erreur dans un channel bufferisé, un `select` entre ce channel et `ctx.Done()`, puis `Shutdown` avec un contexte neuf de 5 secondes. `run` renvoie `nil` après un arrêt demandé, et l'erreur tout de suite si le port est déjà pris.
8. **TODO 7**, `sondeSysteme.Statfs` dans `sonde_unix.go` : `syscall.Statfs(chemin, &st)`, puis `Total = uint64(st.Bsize) * st.Blocks` et `Libre = uint64(st.Bsize) * st.Bavail`. Compare avec `df -k /` : les chiffres doivent correspondre à 1 % près.
9. `go test -race .` vert, `gofmt -l .` vide, `go vet ./...` muet, `golangci-lint run` à zéro. Puis `make build VERSION=1.0.0 && bin/diskexporter -version` : la version doit être `1.0.0`, injectée par `-ldflags`. Lance `bin/diskexporter -points /,/tmp`, interroge `curl localhost:9101/metrics` dans un autre terminal, puis `Ctrl-C` : le journal JSON doit montrer « arrêt demandé » et le processus doit rendre la main.
10. `make cross` produit quatre binaires dans `bin/`. Si tu as Docker : `make docker` puis `docker run --rm -p 9101:9101 diskexporter:dev` ; l'image fait une dizaine de Mo et tourne en utilisateur 65534. Si tu as goreleaser : `make release` fabrique les archives dans `dist/` sans rien publier.

## Comment lancer
```bash
go test .                          # tes TODO (échoue tant qu'ils ne sont pas faits)
go test -race ./solution/          # la solution
go vet ./... && gofmt -l . && golangci-lint run ./...
make build VERSION=1.0.0 && bin/diskexporter -version
bin/diskexporter -points /,/tmp &  # puis dans un autre terminal :
curl -s localhost:9101/healthz
curl -s localhost:9101/metrics | grep -v '^#'
kill %1                            # SIGTERM : arrêt propre
make cross && ls bin/              # quatre binaires
make docker                        # si Docker est installé
make clean
```

## Sortie attendue
```
$ go test -race ./solution/
ok  	cours-go/labs/15-release/solution	1.4s

$ make build VERSION=1.0.0 && bin/diskexporter -version
go build -trimpath -ldflags "-s -w -X main.version=1.0.0" -o bin/diskexporter .
diskexporter 1.0.0

$ bin/diskexporter -points / &
{"time":"...","level":"INFO","msg":"diskexporter démarre","addr":":9101","version":"1.0.0","points":["/"]}
$ curl -s localhost:9101/healthz
ok
$ curl -s localhost:9101/metrics | grep -v '^#'
diskexporter_build_info{version="1.0.0"} 1
diskexporter_scrape_errors 0
disk_total_bytes{mountpoint="/"} 494384795648
disk_free_bytes{mountpoint="/"} 200489050112
disk_used_ratio{mountpoint="/"} 0.5945
$ kill %1
{"time":"...","level":"INFO","msg":"arrêt demandé"}
```
Les chiffres de ton disque diffèrent, le reste non. La page complète sur des chiffres fixes est dans `TestFormaterMetriques`.

## Pour aller plus loin
- Remplace `formaterMetriques` par `github.com/prometheus/client_golang` : un `prometheus.NewGaugeVec` par métrique, `promhttp.Handler()` sur `/metrics`. Compare la page : tu gagnes les métriques du runtime Go (`go_goroutines`, `go_memstats_*`) et `process_*` sans rien écrire.
- Ajoute un `/readyz` qui répond 503 tant que la première collecte n'a pas réussi, et branche-le dans le `Dockerfile` avec `HEALTHCHECK`. Réfléchis à pourquoi `/healthz` et `/readyz` ne doivent pas dire la même chose.
- Ouvre `pprof` sur un second port réservé (`127.0.0.1:6060`, `import _ "net/http/pprof"`) et regarde `go tool pprof http://127.0.0.1:6060/debug/pprof/heap` pendant que `curl` boucle sur `/metrics`. Un exporter qui fuit se voit là, pas dans les logs.
