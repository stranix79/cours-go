# 13. HTTP : client, serveur, API JSON

*Go de zéro à la prod : chapitre 13 sur 16.* ← [12. Fichiers, JSON, ligne de commande DevOps](12-fichiers-json-cli.md) · [Sommaire](../README.md) · [14. Bases de données : database/sql, SQLite, PostgreSQL](14-bases-de-donnees.md) →

Si Go est devenu le langage de l'infra, c'est en grande partie à cause d'un seul paquet : `net/http`. Il contient un client complet (HTTP/1.1 et HTTP/2, TLS, proxies, cookies) et un serveur de production (celui qui fait tourner Kubernetes, Prometheus, Caddy, Gitea) sans rien installer. Là où Python te fait choisir entre `requests` et `httpx`, puis entre Flask, Django et FastAPI, puis un serveur ASGI, Go te donne une seule réponse, dans la bibliothèque standard, maintenue par l'équipe du langage.

Ce chapitre fait les deux côtés du câble : le client (appeler une API, décoder du JSON, délais, erreurs), le serveur (router, répondre en JSON, journaliser, s'arrêter proprement), puis les tests sans ouvrir de port. Les exemples clients parlent à l'API du labo 13, lancée en arrière-plan sur `127.0.0.1:8080`.

### 13.1 Le client en cinq lignes, et le piège qui va avec

```go
client := &http.Client{Timeout: 5 * time.Second}
resp, err := client.Get("http://127.0.0.1:8080/taches")
if err != nil {
	log.Fatal(err)
}
defer resp.Body.Close()

corps, err := io.ReadAll(resp.Body)
if err != nil {
	log.Fatal(err)
}
fmt.Println("statut :", resp.StatusCode, resp.Status)
fmt.Println("type   :", resp.Header.Get("Content-Type"))
fmt.Println("corps  :", string(corps))
```

```
statut : 200 200 OK
type   : application/json; charset=utf-8
corps  : []
```

Trois choses à voir, dans l'ordre où elles te mordront.

**`resp.Body` est un flux, pas une chaîne.** Seuls les en-têtes sont lus quand `Get` revient. `io.ReadAll` lit le corps entier (bien pour du JSON) ; pour un gros fichier, `io.Copy(fichier, resp.Body)` sans jamais tout charger. C'est le même `io.Reader` qu'au chapitre 7.

**`defer resp.Body.Close()` est obligatoire, juste après le test d'erreur.** Sans lui, la connexion TCP n'est pas rendue au pool du client et, après quelques dizaines d'appels, plus de descripteurs. `go vet` ne le voit pas (golangci-lint avec `bodyclose`, oui). Quand `err != nil`, `resp` est nil : d'où l'ordre test, puis `defer`.

**`err` ne dit rien du statut HTTP.** Un 404 ou un 500 est une réponse valide : `err` est nil, et `resp.StatusCode` porte l'information. Les constantes `http.StatusOK`, `http.StatusNotFound` évitent les nombres magiques ; `resp.Status` est le texte (`"200 OK"`).

**Piège :** `http.Get(url)` (la fonction du paquet, sans client) utilise `http.DefaultClient`, qui n'a **aucun délai**. Un serveur qui accepte la connexion et ne répond jamais bloque ta goroutine pour toujours. Réflexe : ne jamais utiliser `http.Get`, `http.Post` ni `http.DefaultClient` dans du code qui tourne longtemps ; construire un `&http.Client{Timeout: ...}` une fois (il est sûr en concurrence et gère un pool de connexions) et le réutiliser partout.

**Venant de Python :** `requests.get(url, timeout=5)` fait tout ça en une ligne, avec `.json()` et `.raise_for_status()`. En Go, chaque étape est visible : délai, lecture, fermeture, statut. Plus long à écrire, impossible à faire à moitié.

### 13.2 Requêtes avec contexte, JSON, en-têtes, retry

Pour tout ce qui n'est pas un `GET` trivial, on construit la requête à la main avec `http.NewRequestWithContext`, puis on l'envoie avec `client.Do`. Le contexte (chapitre 11) porte le délai et l'annulation ; un contexte annulé interrompt la requête en cours, même au milieu de la lecture du corps.

```go
ctx, annule := context.WithTimeout(context.Background(), 2*time.Second)
defer annule()

req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://127.0.0.1:8080/taches",
	strings.NewReader(`{"titre":"vider /var/log sur web-02"}`))
if err != nil {
	log.Fatal(err)
}
req.Header.Set("Content-Type", "application/json")

resp, err := http.DefaultClient.Do(req)   // ici le délai vient du contexte
if err != nil {
	log.Fatal(err)
}
defer resp.Body.Close()
if resp.StatusCode != http.StatusCreated {
	log.Fatalf("statut inattendu : %s", resp.Status)
}
var creee Tache
if err := json.NewDecoder(resp.Body).Decode(&creee); err != nil {
	log.Fatal(err)
}
fmt.Printf("créée : %+v\n", creee)
```

```
créée : {ID:1 Titre:vider /var/log sur web-02 Faite:false}
  #1 vider /var/log sur web-02      faite=false
```

Le corps de la requête est un `io.Reader` : `strings.NewReader` pour une chaîne, `bytes.NewReader` pour le résultat d'un `json.Marshal`, un `*os.File` pour envoyer un fichier. Côté réponse, `json.NewDecoder(resp.Body).Decode(&v)` lit le flux directement, et marche aussi bien pour une struct que pour une slice (la seconde ligne de la sortie vient d'un `GET /taches` décodé dans un `[]Tache`).

Un serveur qui redémarre renvoie des 503 pendant quelques secondes. Un client d'infra réessaie, avec une attente qui double, et jamais sur un 4xx (c'est notre requête qui est fausse, pas le serveur qui est malade) :

```go
func getAvecRetry(ctx context.Context, client *http.Client, url string, essais int) (*http.Response, error) {
	attente := 100 * time.Millisecond
	var derniere error
	for i := 1; i <= essais; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}
		if err == nil {
			resp.Body.Close()   // on ne garde pas cette réponse : on la ferme
			derniere = fmt.Errorf("statut %s", resp.Status)
		} else {
			derniere = err
		}
		log.Printf("essai %d/%d : %v, nouvel essai dans %s", i, essais, derniere, attente)
		select {
		case <-time.After(attente):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		attente *= 2
	}
	return nil, fmt.Errorf("abandon après %d essais : %w", essais, derniere)
}
```

```
2026/09/23 12:29:31 essai 1/5 : statut 503 Service Unavailable, nouvel essai dans 100ms
2026/09/23 12:29:31 essai 2/5 : statut 503 Service Unavailable, nouvel essai dans 200ms
réponse : 200 OK
```

Le `select` entre `time.After` et `ctx.Done()` est le patron du chapitre 11 : on attend, mais on reste annulable. En vrai, ajoute un peu d'aléa à l'attente (*jitter*) pour que cent clients qui ont vu le même 503 ne reviennent pas à la même milliseconde.

### 13.3 Le serveur : un handler, c'est une fonction

Côté serveur, tout repose sur une interface d'une méthode, `http.Handler` : `ServeHTTP(ResponseWriter, *Request)`. Un handler reçoit la requête (`*http.Request` : méthode, URL, en-têtes, corps, contexte) et un `http.ResponseWriter` où il écrit statut, en-têtes et corps. Le serveur lance **une goroutine par connexion** et y appelle ton handler : c'est ce qui permet de tenir dix mille connexions sans rien configurer, et c'est pourquoi tout ce que tes handlers partagent (une map, un compteur) doit être sous mutex (chapitre 11, labo 13).

Le plus petit serveur :
```go
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Bonjour %s, tu as demandé %s\n", r.RemoteAddr, r.URL.Path)
})
log.Fatal(http.ListenAndServe("127.0.0.1:8090", nil))
```

```
$ curl -s -i localhost:8090/hotes/web-01
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8

Bonjour 127.0.0.1:49230, tu as demandé /hotes/web-01
```

`http.HandleFunc` enregistre une fonction sur le routeur par défaut ; `http.ListenAndServe(addr, nil)` sert avec ce routeur. `w` est un `io.Writer` : tout ce qui écrit dans un writer (`fmt.Fprintf`, `io.Copy`, `json.NewEncoder`) écrit dans la réponse. Le `Content-Type` est deviné sur les premiers octets, le statut 200 posé au premier `Write`.

**Venant du C :** c'est la boucle `accept` et le `fork` ou le thread par client que tu aurais écrits, avec le parseur HTTP en prime. Ici l'`accept` est dans `ListenAndServe`, le « thread » est une goroutine de quelques kilo-octets, et le parseur a vingt ans de correctifs de sécurité.

`http.HandleFunc(motif, f)` convertit `f` en `http.Handler` grâce à un type de la stdlib qui mérite d'être lu : `type HandlerFunc func(ResponseWriter, *Request)`, avec une méthode `ServeHTTP` qui appelle la fonction. Une fonction peut avoir des méthodes ; `HandlerFunc(f)` est donc une fonction *qui est* un `Handler`. Tu t'en serviras dans chaque middleware (13.6).

### 13.4 Le routeur de Go 1.22 : méthodes et paramètres de chemin

Jusqu'à Go 1.21, le `ServeMux` de la stdlib ne savait router que par préfixe de chemin : pas de méthode, pas de `{id}`, et c'est pour ça que tout le monde installait gorilla/mux, chi ou gin. Depuis Go 1.22, le motif peut contenir la méthode et des segments nommés :

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /hotes/{nom}", func(w http.ResponseWriter, r *http.Request) {
	nom := r.PathValue("nom")
	ip, ok := hotes[nom]
	if !ok {
		http.Error(w, "hôte inconnu : "+nom, http.StatusNotFound)
		return
	}
	fmt.Fprintln(w, ip)
})
mux.HandleFunc("/{$}", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "racine, et seulement la racine") })
log.Fatal(http.ListenAndServe("127.0.0.1:8090", mux))
```

```
$ curl -s localhost:8090/hotes/pg-01
10.0.0.21
$ curl -s -i localhost:8090/hotes/inconnu
HTTP/1.1 404 Not Found
Content-Type: text/plain; charset=utf-8

hôte inconnu : inconnu
$ curl -s -i -X DELETE localhost:8090/hotes/pg-01 | head -3
HTTP/1.1 405 Method Not Allowed
Allow: GET, HEAD
Content-Type: text/plain; charset=utf-8
$ curl -s localhost:8090/
racine, et seulement la racine
$ curl -s -i localhost:8090/autre | head -1
HTTP/1.1 404 Not Found
```

Les règles du motif :

- `"GET /hotes"` : cette méthode, ce chemin exact. `GET` couvre aussi `HEAD`. Sans méthode, toutes sont acceptées.
- `"{nom}"` capture un segment, lu avec `r.PathValue("nom")` ; `"{reste...}"` en fin de motif capture tout ce qui suit, `/` compris.
- `"/"` seul est un préfixe qui attrape tout ce qui n'a pas de motif plus précis ; `"/{$}"` veut dire « exactement la racine ».
- Le plus précis gagne (`/hotes/{nom}` bat `/`). Deux motifs de même précision qui se recouvrent font paniquer `HandleFunc` au démarrage : mieux qu'un routage silencieusement faux.
- Le mux répond 405 (avec `Allow`) quand le chemin existe mais pas la méthode, 404 quand rien ne correspond, en texte brut : une API tout JSON doit les intercepter (piste dans le labo).

`http.Error(w, message, statut)` est le raccourci pour une erreur en texte : `Content-Type`, statut, message. Pour une API JSON, on écrit son propre `repondErreur` (13.7).
### 13.5 `http.Server`, délais, arrêt propre

`http.ListenAndServe` est un raccourci pour la démo. En production on construit un `http.Server`, pour les délais et l'arrêt propre.

```go
srv := &http.Server{
	Addr:              "127.0.0.1:8090",
	Handler:           mux,
	ReadHeaderTimeout: 5 * time.Second,   // le temps d'envoyer ses en-têtes
	ReadTimeout:       10 * time.Second,  // en-têtes + corps
	WriteTimeout:      30 * time.Second,  // la réponse entière ; IdleTimeout pour le keep-alive
}

ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

go func() {
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}()
<-ctx.Done()   // bloque jusqu'à Ctrl-C ou docker stop
log.Println("signal reçu, arrêt en cours (les requêtes en cours finissent)")

arret, annule := context.WithTimeout(context.Background(), 5*time.Second)
defer annule()
if err := srv.Shutdown(arret); err != nil {
	log.Println("arrêt forcé :", err)
}
log.Println("arrêt propre")
```

Test : un `curl /lent` (le handler dort deux secondes) lancé juste avant le Ctrl-C.
```
2026/09/23 12:29:49 écoute sur 127.0.0.1:8090
2026/09/23 12:29:50 signal reçu, arrêt en cours (les requêtes en cours finissent)
2026/09/23 12:29:52 arrêt propre
```

Et `curl` a bien reçu `fini` : le serveur a cessé d'accepter à 12:29:50, a attendu la requête en cours, a rendu la main à 12:29:52. C'est ce que Kubernetes attend d'un pod entre `SIGTERM` et le `SIGKILL` trente secondes plus tard, et `docker stop` pendant dix secondes. Trois mécanismes se combinent :

- `signal.NotifyContext` transforme un signal en annulation de contexte. Le même contexte se passe à tout ce qui doit s'arrêter (boucle de sondes, pool de workers) : un signal, tout s'arrête.
- `ListenAndServe` bloque, donc il part dans une goroutine. Après `Shutdown`, il renvoie `http.ErrServerClosed`, qui n'est pas une erreur : on la filtre avec `errors.Is`.
- `Shutdown(ctx)` ferme l'écoute, attend la fin des requêtes en cours, abandonne si le contexte expire. On lui donne un contexte **neuf** avec délai, pas celui du signal, déjà annulé.

**Piège :** `ReadHeaderTimeout` à zéro veut dire « infini ». Un client qui envoie ses en-têtes un octet par minute (attaque Slowloris) occupe une goroutine pour toujours ; mille comme lui et ton serveur ne répond plus. golangci-lint le signale (`gosec` G112). Cinq secondes est raisonnable ; `ReadTimeout` et `WriteTimeout` dépendent de l'usage (un upload de 1 Go n'a pas les délais d'une API JSON) ; pour du long polling ou des WebSockets, pas de `WriteTimeout`, le délai se gère par requête avec le contexte.

### 13.6 Middleware : envelopper un handler

Un *middleware* est une fonction qui prend un `http.Handler` et en renvoie un autre qui fait quelque chose avant ou après. Pas de framework, pas de décorateur : une fonction et l'interface.

```go
func journalise(suivant http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		debut := time.Now()
		suivant.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(debut).Round(time.Microsecond))
	})
}

func recupere(suivant http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				log.Printf("panic dans %s : %v", r.URL.Path, p)
				http.Error(w, "erreur interne", http.StatusInternalServerError)
			}
		}()
		suivant.ServeHTTP(w, r)
	})
}

h := journalise(recupere(entetes(mux)))   // entetes pose Server et X-Content-Type-Options
log.Fatal(http.ListenAndServe("127.0.0.1:8090", h))
```

Avec une route `/boum` dont le handler écrit dans une map nil :
```
$ curl -s -i localhost:8090/boum
HTTP/1.1 500 Internal Server Error
Content-Type: text/plain; charset=utf-8
Server: sondes/1.2.0
X-Content-Type-Options: nosniff

erreur interne
```

Et dans le journal : `panic dans /boum : assignment to entry in nil map`, puis `GET /boum (87µs)`.

L'emboîtement se lit de l'extérieur vers l'intérieur : `journalise` voit tout, `recupere` protège tout ce qui est en dessous, `entetes` pose ses en-têtes juste avant le handler. `net/http` récupère déjà les panics (il journalise et ferme la connexion) mais sans répondre au client : `recupere` en fait un 500 propre, et c'est là qu'on branche l'alerte.

Ce `journalise` ne connaît pas le statut de la réponse, parce que `ResponseWriter` ne l'expose pas. La solution, à écrire au labo : envelopper `w` dans une struct qui embarque `http.ResponseWriter` et redéfinit `WriteHeader` pour retenir le code. La composition du chapitre 7, appliquée à une interface de la stdlib.

### 13.7 Une API JSON : encoder, décoder, répondre

Une API JSON, c'est deux fonctions et une discipline. Les fonctions (plus `repondErreur(w, statut, message)`, qui appelle `repondJSON` avec `map[string]string{"erreur": message}`) :

```go
func repondJSON(w http.ResponseWriter, statut int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statut)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Println("encodage JSON :", err)   // trop tard pour changer le statut
	}
}

func decodeTache(r *http.Request) (Tache, error) {
	var t Tache
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)   // 1 Mo maximum
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()                        // {"title": ...} devient une erreur
	if err := dec.Decode(&t); err != nil {
		return Tache{}, errors.New("JSON invalide : " + err.Error())
	}
	if strings.TrimSpace(t.Titre) == "" {
		return Tache{}, errors.New("le titre est obligatoire")
	}
	return t, nil
}
```

La discipline :

- **En-têtes avant `WriteHeader`, `WriteHeader` avant le corps.** Dès le premier octet écrit, le statut est parti (200 par défaut) et `w.Header().Set` n'a plus d'effet. Le cas classique : `http.Error` après avoir déjà écrit la moitié d'une réponse ; `net/http` journalise `superfluous response.WriteHeader call`.
- **Un seul format d'erreur**, `{"erreur": "..."}`, avec le bon statut : 400 requête mal formée, 404 absent, 409 conflit. 201 pour une création (la ressource dans le corps), 204 pour une suppression (sans corps ni `Content-Type`).
- **Limiter le corps** avec `http.MaxBytesReader` : un client ne doit pas te faire allouer un gigaoctet. `DisallowUnknownFields` transforme une faute de frappe côté client en 400 explicite au lieu d'un champ ignoré en silence.
- **Valider après avoir décodé.** Le décodeur vérifie la forme (un booléen là où on attend un booléen), pas le sens (un titre vide).

**Venant de Python :** FastAPI et pydantic font tout ça pour toi, doc OpenAPI comprise. En Go, tu écris ces vingt lignes une fois par projet et tu sais exactement ce qui se passe entre l'octet reçu et la struct. Pour OpenAPI, des outils existent (swag, oapi-codegen), mais ce n'est pas gratuit : c'est le prix de la stdlib.

### 13.8 Tester sans ouvrir de port : `httptest`

Le paquet `net/http/httptest` a deux outils, un par côté du câble.

`httptest.NewRecorder` est un faux `ResponseWriter` qui retient tout : tu appelles ton handler (ou ton routeur complet) directement, sans réseau, et tu inspectes statut, en-têtes, corps.

```go
func TestSanteRecorder(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	Sante(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatalf("statut %d, corps %s", rec.Code, rec.Body.String())
	}
}
```

`httptest.NewServer` lance un vrai serveur sur un port libre de `127.0.0.1`, pour tester un **client** (le tien, ou une bibliothèque) contre un handler que tu contrôles :

```go
func TestSanteServeur(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(Sante))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/healthz")   // srv.URL vaut http://127.0.0.1:<port libre>
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	corps, _ := io.ReadAll(resp.Body)
	t.Logf("serveur de test sur %s, réponse : %s", srv.URL, corps)
}
```

```
$ go test -v ./testex
=== RUN   TestSanteRecorder
--- PASS: TestSanteRecorder (0.00s)
=== RUN   TestSanteServeur
    sante_test.go:34: serveur de test sur http://127.0.0.1:49283, réponse : {"statut":"ok"}
--- PASS: TestSanteServeur (0.00s)
PASS
ok  	ch13/testex	0.346s
```

La règle : `NewRecorder` pour tes handlers (des centaines de tests par seconde, aucun port), `NewServer` pour ton code client ou ce qui exige une vraie connexion. Pour l'arrêt propre, on lance `run(ctx, addr)` dans une goroutine sur un port libre, on annule `ctx`, on vérifie que `run` rend la main : c'est le dernier test du labo. Ce découpage (`main` lit les drapeaux, `run` fait tout et prend un contexte) est le patron de tous les services Go bien écrits.

### 13.9 Fichiers statiques, `embed`, TLS

Un service a souvent une page ou deux à servir (un tableau de bord, une doc). Avec `embed` (chapitre 12) les fichiers sont dans le binaire : rien à copier à côté, rien à monter dans le conteneur.
```go
//go:embed static
var static embed.FS

racine, err := fs.Sub(static, "static")   // static/index.html devient /index.html
mux.Handle("/", http.FileServerFS(racine))
mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{"version":"1.2.0"}` + "\n"))
})
```

```
$ curl -s -i localhost:8090/ | head -4
HTTP/1.1 200 OK
Accept-Ranges: bytes
Content-Type: text/html; charset=utf-8
$ curl -s localhost:8090/style.css
body { font-family: monospace; background: #111; color: #eee; }
$ curl -s localhost:8090/api/version
{"version":"1.2.0"}
```

`http.FileServerFS` sert n'importe quel `fs.FS` : un `embed.FS`, ou `os.DirFS("/var/www")`. Il gère `index.html`, les types MIME, le cache et les requêtes partielles (`Range`). Le binaire avec ses deux fichiers fait 8,4 Mo. C'est ainsi que Deckhand embarque son interface web entière (`web/`) et Prometheus la sienne : un seul fichier à déployer.

TLS tient en une ligne de plus, si tu as un certificat et sa clé :

```go
log.Fatal(http.ListenAndServeTLS("127.0.0.1:8443", "cert.pem", "key.pem", nil))
```

```
$ go run $(go env GOROOT)/src/crypto/tls/generate_cert.go --host 127.0.0.1,localhost --ecdsa-curve P256
2026/09/23 12:29:56 wrote cert.pem
2026/09/23 12:29:56 wrote key.pem
$ curl -s --cacert cert.pem https://localhost:8443/
TLS 304, chiffrement 1303
```

`304` est TLS 1.3, `1303` la suite `TLS_CHACHA20_POLY1305_SHA256` : les défauts de `crypto/tls` sont bons, on n'y touche pas. `generate_cert.go`, livré avec Go, fabrique un certificat auto-signé pour le développement. Pour un vrai domaine : `golang.org/x/crypto/acme/autocert` obtient et renouvelle les certificats Let's Encrypt depuis ton programme (dix lignes, c'est ce que fait Caddy), ou, plus souvent en infra, un reverse proxy devant (13.10) qui termine TLS et parle HTTP en clair au service sur le réseau interne.

### 13.10 Ce qu'il y a autour : chi, WebSocket, nginx

**Les routeurs tiers.** Avant Go 1.22, on installait [chi](https://github.com/go-chi/chi), [gin](https://github.com/gin-gonic/gin) ou [echo](https://github.com/labstack/echo) pour avoir méthodes et paramètres de chemin. Aujourd'hui la stdlib suffit pour une API de taille moyenne, et c'est ce que ce cours utilise. chi garde deux avantages : les **groupes de routes** (`r.Route("/api", func(r chi.Router) { r.Use(authentification); r.Get("/decks", ...) })`, un middleware appliqué à tout un sous-arbre) et des middlewares prêts (`RequestID`, `RealIP`, `Logger`, `Recoverer`, `Timeout`). Il est fait de handlers `net/http` standard : tout ce chapitre s'y branche tel quel. C'est pour ces groupes que Deckhand l'a choisi (`internal/hub/hub.go`, `internal/ui/ui.go`) : routes publiques, authentifiées, d'API, chacune avec sa pile de middlewares. gin et echo ont leur propre type de contexte, incompatible avec `http.Handler` : un choix qu'on ne fait plus pour un nouveau projet.

**WebSocket.** `net/http` ne l'implémente pas. Deux bibliothèques : `gorilla/websocket` (l'historique) et [coder/websocket](https://github.com/coder/websocket) (contexte partout, sans dépendance, choisi par Deckhand pour synchroniser les présentations en direct : `internal/session/ws.go`). Un handler HTTP normal appelle `websocket.Accept(w, r, nil)`, puis lit et écrit des messages en boucle, avec `ctx` pour s'arrêter. Une goroutine par client, comme pour tout le reste.

**Le reverse proxy devant.** En production, ton service écoute sur `127.0.0.1:8080` ou sur le réseau du conteneur, et nginx, Caddy ou Traefik est devant : TLS, compression, limitation de débit, plusieurs services derrière un seul port 443. Conséquence : `r.RemoteAddr` est l'adresse du proxy. L'adresse réelle est dans `X-Forwarded-For`, **que le proxy pose et que tu ne dois lire que si tu es sûr qu'il y a un proxy** : un client qui parle directement au service y met ce qu'il veut. Côté nginx :

```nginx
location / {
    proxy_pass         http://127.0.0.1:8080;
    proxy_set_header   Host              $host;
    proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header   X-Forwarded-Proto $scheme;
}
```

Et si le proxy, c'est toi (un service qui en expose un autre), `httputil.NewSingleHostReverseProxy` le fait en trois lignes ; c'est le cœur de Traefik.

### 13.11 Pour le labo

Le [labo 13](../labs/13-api-todo/README.md) assemble tout ce chapitre : une API REST de tâches (`GET/POST /taches`, `GET/PUT/DELETE /taches/{id}`) en stdlib pure, un magasin en mémoire sous mutex, un middleware qui journalise le statut et la durée, des erreurs JSON, un `run(ctx, addr)` avec arrêt propre, et une suite de tests `httptest` qui couvre chaque code de statut. Ce sera le squelette du serveur d'API du projet final.

### À retenir

- Client : un `&http.Client{Timeout: ...}` réutilisé, `NewRequestWithContext` pour le délai et l'annulation, `defer resp.Body.Close()` juste après le test d'erreur, `resp.StatusCode` pour le statut (l'erreur ne le couvre pas), `json.NewDecoder(resp.Body).Decode(&v)` pour lire.
- `http.DefaultClient` et `http.Get` n'ont pas de délai : jamais dans un programme qui tourne longtemps.
- Serveur : un handler est une fonction `(w http.ResponseWriter, r *http.Request)`, `http.Handler` une interface d'une méthode, `HandlerFunc` le pont entre les deux ; une goroutine par connexion, donc mutex sur tout ce qui est partagé. Le `ServeMux` de Go 1.22 route par méthode et paramètre (`"GET /taches/{id}"`, `r.PathValue("id")`), répond 405 et 404 tout seul, `"/{$}"` pour la racine exacte.
- En production : un `http.Server` avec `ReadHeaderTimeout`, `signal.NotifyContext` pour le signal, `Shutdown` avec un contexte neuf à délai, `http.ErrServerClosed` n'est pas une erreur.
- Middleware = `func(http.Handler) http.Handler` ; journalisation, recover, en-têtes ; l'ordre se lit de l'extérieur vers l'intérieur.
- API JSON : en-têtes, puis `WriteHeader`, puis le corps ; un seul format d'erreur ; `MaxBytesReader` et `DisallowUnknownFields` à l'entrée ; 201 pour créer, 204 pour supprimer.
- `httptest.NewRecorder` pour tester les handlers sans port, `httptest.NewServer` pour tester un client ; `embed` + `http.FileServerFS` pour les fichiers statiques ; chi si tu veux des groupes de routes, nginx devant pour TLS et `X-Forwarded-For`.

---

← [12. Fichiers, JSON, ligne de commande DevOps](12-fichiers-json-cli.md) · [Sommaire](../README.md) · [14. Bases de données : database/sql, SQLite, PostgreSQL](14-bases-de-donnees.md) →
