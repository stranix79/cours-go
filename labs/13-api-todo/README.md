# Labo 13 : API REST de tâches en bibliothèque standard

**Chapitre** : [13. HTTP : client, serveur, API JSON](../../cours/13-http.md)

## Objectif
Écrire une API REST complète (lister, créer, lire, remplacer, supprimer des tâches) avec `net/http` seulement : le `ServeMux` de Go 1.22 pour le routage par méthode et paramètre de chemin, un stockage en mémoire protégé par un mutex, un middleware de journalisation, des erreurs en JSON, un arrêt propre sur Ctrl-C, et des tests `httptest` qui couvrent tout ça sans ouvrir un port. À la fin tu sais faire ce que FastAPI faisait pour toi au chapitre 17 du cours Python, et tu sais pourquoi la stdlib suffit souvent.

## Consignes
Trois fichiers à compléter, dans cet ordre : `taches.go` (le stockage), `api.go` (les handlers), `main.go` (le serveur). Les tests de `api_test.go` te guident : lance `go test .` après chaque TODO et regarde ce qui passe au vert.

1. **`taches.go`, le magasin.** `NouveauMagasin()` initialise la map et met `suivant` à 1. `Liste()` renvoie les tâches triées par `ID` croissant, dans une slice **vide et non nil** quand il n'y en a pas (le JSON doit être `[]`, pas `null`). `Ajoute(titre)` crée une tâche non faite avec le prochain ID. `Lit(id)` renvoie `(Tache, bool)`. `Remplace(id, t)` met à jour titre et état, **garde l'ID de l'URL** (celui reçu dans `t` est ignoré), renvoie `false` si `id` n'existe pas. `Supprime(id)` renvoie `false` si la tâche n'existait pas. Chaque méthode verrouille `mu` : le test `TestMagasinConcurrent` fait 50 ajouts en parallèle, et `go test -race .` doit être vert.
2. **`api.go`, les réponses.** `repondJSON(w, statut, v)` pose `Content-Type: application/json; charset=utf-8`, écrit le statut avec `WriteHeader`, puis encode `v` avec `json.NewEncoder(w).Encode`. `repondErreur(w, statut, message)` répond `{"erreur": "message"}` via `repondJSON`.
3. **`api.go`, les routes.** Dans `nouveauRouteur`, enregistre `GET /taches` (200, la liste), `POST /taches` (201 et la tâche créée ; 400 si le JSON est invalide ou si le titre est vide après `strings.TrimSpace`), `GET /taches/{id}` (200 ou 404), `PUT /taches/{id}` (200 et la tâche mise à jour, 400 si corps invalide, 404 si inconnue), `DELETE /taches/{id}` (204 sans corps, ou 404). L'`{id}` se lit avec `r.PathValue("id")` et se convertit avec `strconv.Atoi` ; s'il n'est pas numérique, réponds 400 `{"erreur": "id invalide : abc"}`. Toutes les erreurs sont en JSON. Le `ServeMux` répond de lui-même 405 quand la méthode ne correspond pas et 404 quand le chemin est inconnu, tu n'as rien à faire pour ça.
4. **`api.go`, le middleware.** `journalise(suivant)` mesure la durée, enveloppe `w` dans un `enregistreur` (statut initial 200), appelle `suivant.ServeHTTP`, puis écrit `METHODE /chemin -> statut (durée)` avec `log.Printf`. `enregistreur.WriteHeader` retient le statut avant de le transmettre.
5. **`main.go`, le serveur.** `run(ctx, addr)` construit un `http.Server` avec `Addr`, `Handler: nouveauRouteur(NouveauMagasin())` et `ReadHeaderTimeout: 5 * time.Second`, lance `ListenAndServe` dans une goroutine qui envoie son erreur dans un channel bufferisé, attend le premier de `ctx.Done()` ou de cette erreur, puis appelle `srv.Shutdown` avec un context à délai de 5 s. Après un arrêt demandé par `ctx`, `run` renvoie `nil` (l'erreur `http.ErrServerClosed` renvoyée par `ListenAndServe` n'en est pas une : reconnais-la avec `errors.Is`). Un port déjà occupé doit faire renvoyer l'erreur immédiatement. Le test `TestRunArretPropre` lance `run` sur un port libre, vérifie qu'il répond, annule le context et exige que `run` rende la main en moins d'une seconde.
6. Lance `gofmt -l .`, `go vet ./...`, `go test .` et `go test -race .`. Puis lance le serveur et joue avec `curl` (bloc ci-dessous), et arrête-le avec Ctrl-C : la dernière ligne du journal doit être `arrêt propre`.

## Comment lancer
```bash
go test .                        # tes tests (échouent tant que les TODO restent)
go test -race .                  # le détecteur de data races, sur ton magasin
go test ./solution/              # les mêmes tests sur la solution
go run . -addr 127.0.0.1:8080    # ou go run ./solution, puis dans un autre terminal :
curl -s localhost:8080/taches
curl -s -X POST localhost:8080/taches -d '{"titre":"sauvegarder pg-01"}'
curl -s -X POST localhost:8080/taches -d '{"titre":"renouveler le certificat"}'
curl -s -i -X POST localhost:8080/taches -d '{"titre":""}'
curl -s -X PUT localhost:8080/taches/1 -d '{"titre":"sauvegarder pg-01","faite":true}'
curl -s localhost:8080/taches
curl -s -i -X DELETE localhost:8080/taches/2
curl -s localhost:8080/taches/2
curl -s localhost:8080/taches/abc
curl -s -i -X PATCH localhost:8080/taches | head -2
```

## Sortie attendue
Côté client (les en-têtes `Date` varient) :
```
$ curl -s localhost:8080/taches
[]
$ curl -s -X POST localhost:8080/taches -d '{"titre":"sauvegarder pg-01"}'
{"id":1,"titre":"sauvegarder pg-01","faite":false}
$ curl -s -X POST localhost:8080/taches -d '{"titre":"renouveler le certificat"}'
{"id":2,"titre":"renouveler le certificat","faite":false}
$ curl -s -i -X POST localhost:8080/taches -d '{"titre":""}'
HTTP/1.1 400 Bad Request
Content-Type: application/json; charset=utf-8
Content-Length: 38

{"erreur":"le titre est obligatoire"}
$ curl -s -X PUT localhost:8080/taches/1 -d '{"titre":"sauvegarder pg-01","faite":true}'
{"id":1,"titre":"sauvegarder pg-01","faite":true}
$ curl -s localhost:8080/taches
[{"id":1,"titre":"sauvegarder pg-01","faite":true},{"id":2,"titre":"renouveler le certificat","faite":false}]
$ curl -s -i -X DELETE localhost:8080/taches/2
HTTP/1.1 204 No Content

$ curl -s localhost:8080/taches/2
{"erreur":"tâche introuvable"}
$ curl -s localhost:8080/taches/abc
{"erreur":"id invalide : abc"}
$ curl -s -i -X PATCH localhost:8080/taches | head -2
HTTP/1.1 405 Method Not Allowed
Allow: GET, HEAD, POST
```
Côté serveur, puis Ctrl-C (les heures et durées varient) :
```
2026/09/23 12:27:20 écoute sur 127.0.0.1:8080
2026/09/23 12:27:20 GET /taches -> 200 (248µs)
2026/09/23 12:27:20 POST /taches -> 201 (290µs)
2026/09/23 12:27:20 POST /taches -> 201 (31µs)
2026/09/23 12:27:20 POST /taches -> 400 (50µs)
2026/09/23 12:27:21 PUT /taches/1 -> 200 (52µs)
2026/09/23 12:27:21 GET /taches -> 200 (89µs)
2026/09/23 12:27:21 DELETE /taches/2 -> 204 (4µs)
2026/09/23 12:27:21 GET /taches/2 -> 404 (24µs)
2026/09/23 12:27:21 GET /taches/abc -> 400 (21µs)
2026/09/23 12:27:21 PATCH /taches -> 405 (11µs)
^C2026/09/23 12:27:21 arrêt propre
```

## Pour aller plus loin
- Le 405 du `ServeMux` est en texte brut (`Content-Type: text/plain`). Enveloppe le mux dans un handler qui intercepte les 404 et 405 pour les renvoyer en JSON, comme les autres erreurs : il te faut un `enregistreur` qui retient aussi le corps.
- Ajoute `?faite=true` sur `GET /taches` (`r.URL.Query().Get("faite")`) pour filtrer, et un test par valeur (`true`, `false`, absent, invalide).
- Remplace `log.Printf` par `slog` avec un `JSONHandler` (chapitre 15), et ajoute l'adresse du client (`r.RemoteAddr`, ou `X-Forwarded-For` derrière nginx) dans la ligne de journal.
