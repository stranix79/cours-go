# Labo 11 : Pinger parallèle

**Chapitre** : [11. Concurrence : goroutines, channels, select, context](../../cours/11-concurrence.md)

## Objectif
Écrire `pinger`, un outil qui sonde des ports TCP en parallèle avec un nombre de sondes simultanées réglable, un délai par cible, une annulation propre par Ctrl-C, et des résultats affichés dans l'ordre des cibles quel que soit l'ordre d'arrivée. C'est le worker pool du chapitre, avec un contexte, sur du vrai réseau. Les tests montent un vrai serveur TCP local sur un port libre : ils sont déterministes, sans réseau extérieur, et tournent en moins d'une seconde.

## Consignes
1. Lis `pinger.go` (la logique, avec les `// TODO`) et `main.go` (fourni, il ne fait que lire les options et afficher). Lance `go test .` : tout échoue, mais ça compile. `go vet ./...` et `gofmt -l .` ne disent rien.
2. `ParseCible(s string) (Cible, error)` : `"hote:port"` ou `"[ipv6]:port"` via `net.SplitHostPort`, port converti avec `strconv.Atoi` et validé entre 1 et 65535. Toute autre forme est une erreur (le message importe peu, mais garde celui de `SplitHostPort` quand c'est lui qui échoue : il est clair). `go test -run ParseCible .`.
3. `Sonder(ctx, hote, port, timeout) Resultat` : note l'heure, construis `net.Dialer{Timeout: timeout}` et appelle `DialContext(ctx, "tcp", adresse)`. Le résultat contient la cible, `OK` (pas d'erreur), `Duree` et `Err`. Ferme la connexion si elle a réussi. Cas limites testés : un port fermé (`connection refused`, immédiat), et un contexte déjà annulé, qui doit rendre `context.Canceled` **sans attendre** le délai. `go test -run Sonder .` (ça lance aussi les tests de `SonderTous`, qui échouent encore : normal).
4. `SonderTous(ctx, cibles, parallelisme) []Resultat` : le worker pool. Un channel de tâches qui transporte les **index** des cibles, `parallelisme` workers (`wg.Go`) qui lisent ces index, sondent (avec `c.Timeout`, ou `TimeoutDefaut` si zéro) et envoient `(index, Resultat)` sur un channel de résultats ; une goroutine qui alimente les tâches puis **ferme** le channel ; une goroutine qui fait `wg.Wait()` puis ferme les résultats ; et la boucle finale qui range chaque résultat à `res[index]`. Cas limites : `parallelisme <= 0` vaut 1 ; aucune cible rend une slice vide ; un contexte annulé rend tous les résultats en erreur, vite. Le test vérifie l'ordre avec un parallélisme de 1, 2, 10 et 0.
5. `FormatTableau(res) string` : un en-tête puis une ligne par résultat, format `"%-22s %-4s %8s  %s\n"` (cible, `OK`/`KO`, durée arrondie à la milliseconde, texte de l'erreur ou vide). Le test compare la chaîne exacte : respecte le format à l'espace près.
6. `go test .` puis `go test -race .` : tout vert. Essaie l'outil sur de vraies cibles (adapte au réseau où tu es) et regarde le code de sortie : 1 dès qu'une cible est KO, 2 sur une erreur d'usage. Lance-le avec `-p 1` puis `-p 8` sur plusieurs cibles qui ne répondent pas et compare le temps total affiché sur stderr : c'est la différence entre séquentiel et parallèle. Pendant une sonde longue, tape Ctrl-C : le tableau s'affiche quand même, avec `context canceled` sur les cibles restantes.

## Comment lancer
```bash
go test .                       # tes tests (< 1 s)
go test -race .                 # avec le détecteur de courses
go run . -timeout 500ms 127.0.0.1:22 127.0.0.1:1 192.0.2.1:80 ; echo "code $?"
go run . -p 1 192.0.2.1:80 192.0.2.1:81 192.0.2.1:82     # séquentiel
go run . -p 3 192.0.2.1:80 192.0.2.1:81 192.0.2.1:82     # parallèle
go test ./solution/ && go run ./solution 127.0.0.1:22
```

## Sortie attendue
Avec un service local sur le port 9999 (par exemple `python3 -m http.server 9999` dans un autre terminal), un port fermé et une adresse qui ne répond jamais (192.0.2.1 est réservée à la documentation) :
```
$ go run . -timeout 500ms 127.0.0.1:9999 127.0.0.1:1 192.0.2.1:80
CIBLE                  ETAT    DUREE  DETAIL
127.0.0.1:9999         OK        1ms
127.0.0.1:1            KO        1ms  dial tcp 127.0.0.1:1: connect: connection refused
192.0.2.1:80           KO      504ms  dial tcp 192.0.2.1:80: i/o timeout
3 cibles en 505ms
$ echo $?
1
$ go run . -p 1 -timeout 300ms 192.0.2.1:80 192.0.2.1:81 192.0.2.1:82 2>&1 | tail -1
3 cibles en 911ms
$ go run . -p 3 -timeout 300ms 192.0.2.1:80 192.0.2.1:81 192.0.2.1:82 2>&1 | tail -1
3 cibles en 302ms
$ go test .
ok  	cours-go/labs/11-pinger-parallele	0.3s
```

## Pour aller plus loin
- Remplace la plomberie de `SonderTous` par `errgroup` (`go get golang.org/x/sync/errgroup`) : `g, ctx := errgroup.WithContext(ctx)`, `g.SetLimit(parallelisme)`, un `g.Go` par cible qui écrit dans `res[i]`, puis `g.Wait()`. Trente lignes deviennent dix ; compare les deux versions et décide laquelle tu préfères lire dans six mois.
- Ajoute une option `-json` qui sort les résultats avec `encoding/json` (chapitre 12) : l'erreur, de type `error`, ne se sérialise pas toute seule, il faut un champ `Erreur string` rempli avec `err.Error()`.
- Fais tourner l'outil en boucle toutes les 10 secondes (`time.Ticker`) et arrête-le proprement au Ctrl-C : c'est le squelette d'un exporter Prometheus, que le chapitre 13 complète avec un serveur HTTP.
