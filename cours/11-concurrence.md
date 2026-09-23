# 11. Concurrence : goroutines, channels, select, context

*Go de zéro à la prod : chapitre 11 sur 16.* ← [10. Sous le capot : compilation, mémoire, GC, goroutines](10-sous-le-capot.md) · [Sommaire](../README.md) · [12. Fichiers, JSON, ligne de commande DevOps](12-fichiers-json-cli.md) →

Ton script de supervision sonde cinquante serveurs à la suite, deux secondes de timeout chacun : quand trois sont en panne, le tour prend six secondes de plus, et le tableau de bord attend. En Python tu aurais sorti `ThreadPoolExecutor` ou `asyncio` ; en C, `pthread_create` et un tableau de handles. En Go, c'est le mot-clé `go`, et surtout ce qui va autour : des tuyaux typés pour faire circuler les résultats (les channels), un aiguillage pour attendre plusieurs choses à la fois (`select`), et un objet pour dire « arrête tout » à toute une arborescence de goroutines (`context`). Ce chapitre est le cœur de ce qui fait choisir Go pour l'infra ; prends le temps de faire tourner chaque exemple.

### 11.1 Lancer et attendre : `go` et `sync.WaitGroup`

`go f()` lance `f` dans une nouvelle goroutine et continue tout de suite. Rien n'attend la goroutine, rien ne récupère son résultat, et si `main` se termine, tout le programme s'arrête, goroutines comprises : `go sonde("web1", 10*time.Millisecond)` en première ligne d'un `main` qui se termine aussitôt n'affichera jamais rien. Pour attendre, on compte : `sync.WaitGroup` est un compteur. `Add(1)` avant de lancer, `Done()` quand la goroutine finit, `Wait()` bloque jusqu'à zéro :

```go
func sonde(nom string, d time.Duration) {
	time.Sleep(d)
	fmt.Println(nom, "répond après", d)
}

var wg sync.WaitGroup
debut := time.Now()
for _, s := range []string{"db1", "db2", "cache"} {
	wg.Add(1)
	go func() {
		defer wg.Done()
		sonde(s, 100*time.Millisecond)
	}()
}
wg.Wait()
fmt.Println("trois sondes en", time.Since(debut).Round(10*time.Millisecond))
```

```
db1 répond après 100ms
db2 répond après 100ms
cache répond après 100ms
trois sondes en 100ms
```

Trois attentes de 100 ms en 100 ms, et l'ordre des trois premières lignes n'est pas garanti : les goroutines finissent quand elles finissent. Depuis Go 1.25, `wg.Go(f)` fait le `Add`, lance la goroutine et fait le `Done` pour toi : la boucle devient `wg.Go(func() { sonde(s, 50*time.Millisecond) })`, suivie du même `wg.Wait()`. C'est la forme à utiliser, et celle du reste du chapitre.

**Venant du C :** `go` est `pthread_create` sans le `pthread_t` à conserver, et `WaitGroup` remplace la boucle de `pthread_join`. Il n'y a pas de valeur de retour : une goroutine ne « rend » rien, elle communique (11.2) ou écrit dans une case qui lui est réservée (11.7).

### 11.2 Channels : des tuyaux typés

Un channel est une file d'attente typée et synchronisée, que des goroutines utilisent pour se passer des valeurs. `make(chan T)` en crée un ; `ch <- v` envoie ; `v := <-ch` reçoit ; `close(ch)` annonce qu'il n'y aura plus rien.

**Non bufferisé** (`make(chan T)`), c'est un rendez-vous : l'envoi bloque jusqu'à ce qu'un receveur soit là, et réciproquement. Aucune valeur n'est « posée quelque part ». **Bufferisé** (`make(chan T, n)`), le tuyau a `n` places : l'envoi ne bloque que s'il est plein, la réception que s'il est vide.

```go
// producteur ne peut qu'envoyer : chan<- string.
func producteur(sortie chan<- string) {
	for _, h := range []string{"db1", "db2", "db3"} {
		fmt.Println("envoi de", h)
		sortie <- h
	}
	close(sortie) // plus rien ne viendra : les lecteurs peuvent s'arrêter
}

func main() {
	rdv := make(chan string)
	go producteur(rdv)
	time.Sleep(50 * time.Millisecond) // le producteur est bloqué sur son premier envoi
	for h := range rdv {              // range s'arrête quand le channel est fermé
		fmt.Println("  reçu", h)
	}

	buf := make(chan int, 3)
	buf <- 1
	buf <- 2 // pas de lecteur, et pourtant ça passe : le tampon a 3 places
	close(buf)
	v, ok := <-buf
	fmt.Println(v, ok, "reste :", len(buf))
	<-buf
	v, ok = <-buf // fermé et vide : valeur zéro, ok=false
	fmt.Println(v, ok)
}
```

```
envoi de db1
  reçu db1
envoi de db2
envoi de db3
  reçu db2
  reçu db3
1 true reste : 1
0 false
```

Quatre choses à lire dans cette sortie. Le producteur affiche « envoi de db1 » puis attend 50 ms qu'on le lise : c'est le rendez-vous. `for h := range rdv` lit jusqu'à la fermeture, sans compter. Un channel fermé se lit encore (les valeurs en attente, puis la valeur zéro), et la forme `v, ok := <-ch` dit si `v` est une vraie valeur (`ok == true`) ou le signal de fin. Enfin le type `chan<- string` (envoi seul) et `<-chan string` (réception seule) sont des contrats vérifiés à la compilation : une fonction qui reçoit un `<-chan` ne peut ni envoyer ni fermer. Utilise-les dans toutes les signatures ; c'est gratuit et ça documente qui produit et qui consomme.

Qui ferme ? **Le producteur, jamais le consommateur**, et un seul producteur. S'il y en a plusieurs, c'est un `WaitGroup` qui attend qu'ils aient tous fini, puis une goroutine dédiée qui ferme (l'exemple du worker pool le montre). Et on ne ferme que si quelqu'un a besoin de savoir que c'est fini : un channel que personne ne parcourt avec `range` peut rester ouvert et être ramassé par le GC.

### 11.3 `select` : attendre plusieurs choses

`select` est un `switch` sur des opérations de channel : il bloque jusqu'à ce qu'**une** des branches soit possible, et l'exécute. Si plusieurs sont prêtes, il en tire une au hasard. C'est l'outil des délais et des annulations :

```go
lent := make(chan string)
go func() {
	time.Sleep(2 * time.Second)
	lent <- "réponse de db1"
}()

select {
case r := <-lent:
	fmt.Println(r)
case <-time.After(500 * time.Millisecond):
	fmt.Println("délai dépassé : db1 ne répond pas")
}
```

```
délai dépassé : db1 ne répond pas
```

`time.After(d)` renvoie un channel qui recevra une valeur après `d` : une branche « au bout de 500 ms, abandonne ». Une branche `default` s'exécute si rien n'est prêt : `select` ne bloque plus, c'est un sondage (« y a-t-il quelque chose ? sinon je continue »). Sans `default` et sans aucune branche jamais prête, `select {}` bloque pour toujours (on s'en sert, dans un `main` de serveur qui ne doit jamais sortir).

### 11.4 Le pattern worker pool

Cent mille goroutines sont possibles, mais cent mille connexions simultanées vers le même serveur ne sont pas polies. On borne : `n` goroutines ouvrières (*workers*) lisent des tâches dans un channel et déposent les résultats dans un autre. La borne, c'est la taille du pool ; la file, c'est le channel.

```go
type Tache struct{ ID int; Hote string }
type Resultat struct{ ID int; Sortie string }

func worker(id int, taches <-chan Tache, resultats chan<- Resultat) {
	for t := range taches {
		time.Sleep(100 * time.Millisecond) // le vrai travail : une sonde, une requête...
		resultats <- Resultat{t.ID, fmt.Sprintf("%s ok (worker %d)", t.Hote, id)}
	}
}

func main() {
	hotes := []string{"web1", "web2", "db1", "db2", "cache", "lb", "mq", "dns"}
	taches := make(chan Tache)
	resultats := make(chan Resultat)

	var wg sync.WaitGroup
	for w := 1; w <= 3; w++ {
		wg.Go(func() { worker(w, taches, resultats) })
	}
	go func() { // quand tous les workers ont fini, plus aucun résultat ne viendra
		wg.Wait()
		close(resultats)
	}()

	debut := time.Now()
	go func() {
		for i, h := range hotes {
			taches <- Tache{i, h}
		}
		close(taches) // les workers sortent de leur range
	}()

	sorties := make([]string, len(hotes))
	for r := range resultats {
		sorties[r.ID] = r.Sortie // on remet dans l'ordre grâce à l'ID
	}
	for _, s := range sorties {
		fmt.Println(s)
	}
	fmt.Println("8 tâches, 3 workers :", time.Since(debut).Round(10*time.Millisecond))
}
```

```
web1 ok (worker 1)
web2 ok (worker 2)
db1 ok (worker 3)
db2 ok (worker 2)
...
8 tâches, 3 workers : 310ms
```

Huit tâches de 100 ms, trois workers : trois vagues, 300 ms. Note les trois goroutines de plomberie : celle qui alimente `taches` puis le ferme, celle qui attend les workers puis ferme `resultats`, et `main` qui consomme. Les résultats arrivent dans le désordre et c'est l'ID qui les range. C'est `ThreadPoolExecutor(max_workers=3)` écrit à la main, sans GIL, et c'est exactement la structure du labo.

### 11.5 Le pattern pipeline

Un pipeline enchaîne des étages : chaque fonction reçoit un channel en lecture, rend un channel en lecture, et fait son travail dans une goroutine qu'elle lance elle-même. Un étage a toujours la même forme :

```go
func filtrerErreurs(in <-chan string) <-chan string {
	out := make(chan string)
	go func() {
		defer close(out) // quand l'entrée est fermée, on ferme la sortie
		for l := range in {
			if strings.Contains(l, " ERROR ") {
				out <- l
			}
		}
	}()
	return out
}
```

et on les compose comme des `|` de shell : `for l := range majuscules(filtrerErreurs(lire(journal)))`. Chaque étage tourne dans sa goroutine, le `defer close(out)` propage la fin de gauche à droite, et la mémoire reste bornée : une ligne à la fois dans chaque tuyau. Pour un fichier de 10 Go, c'est la différence entre « ça passe » et « OOM ». La section 11.9 en donne une version complète, exécutée, avec un étage parallèle.

### 11.6 `context.Context` : annuler et borner

Un timeout dans un `select`, c'est bien pour une opération. Mais une requête HTTP qui lance trois requêtes SQL qui lancent chacune une résolution DNS : quand le client raccroche, il faut tout arrêter, à tous les étages. C'est le rôle de `context.Context`, un objet qu'on passe en **premier argument** de toute fonction qui attend, bloque ou appelle le réseau (la convention est absolue dans la bibliothèque standard : `http.NewRequestWithContext`, `db.QueryContext`, `net.Dialer.DialContext`, `exec.CommandContext`).

Un contexte a un channel `Done()` qui se ferme quand il est annulé, et une méthode `Err()` qui dit pourquoi. On en dérive d'autres qui héritent de l'annulation :

```go
// requeteLente simule une requête qui prend d, mais s'arrête si ctx est annulé.
func requeteLente(ctx context.Context, nom string, d time.Duration) error {
	select {
	case <-time.After(d):
		fmt.Println(nom, ": terminé")
		return nil
	case <-ctx.Done():
		fmt.Println(nom, ": abandonné,", ctx.Err())
		return ctx.Err()
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel() // toujours, même si le délai a déjà expiré : libère les ressources

	err := requeteLente(ctx, "lente", 2*time.Second)
	fmt.Println("DeadlineExceeded ?", errors.Is(err, context.DeadlineExceeded))
}
```

```
lente : abandonné, context deadline exceeded
DeadlineExceeded ? true
```

`context.Background()` est la racine, jamais annulée. `WithTimeout` (ou `WithDeadline`) dérive un contexte qui s'annule tout seul ; `WithCancel` en dérive un que tu annules toi-même en appelant `cancel()` (depuis une autre goroutine, un gestionnaire de signal, un bouton) : `ctx.Err()` vaut alors `context.Canceled`. Annuler un parent annule tous ses enfants : c'est la **propagation**. Les deux erreurs possibles sont `context.DeadlineExceeded` et `context.Canceled`, à tester avec `errors.Is`. Et chaque `WithX` rend une fonction `cancel` qu'il faut appeler (`defer cancel()`), sinon `go vet` râle et le contexte reste en mémoire jusqu'à son délai.

### 11.7 Mutex, Once, atomic : protéger un état

Le mot d'ordre de Go est « ne communique pas en partageant la mémoire ; partage la mémoire en communiquant ». Les channels transportent la **propriété** d'une donnée d'une goroutine à une autre. Mais il reste des cas où plusieurs goroutines doivent vraiment lire et écrire le même état : un compteur, un cache, une table de statistiques. Là, un channel serait contorsionné, et c'est `sync.Mutex` :

```go
type Stats struct {
	mu      sync.Mutex
	parCode map[int]int
}

func (s *Stats) Note(code int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.parCode[code]++
}

s := &Stats{parCode: make(map[int]int)}
var wg sync.WaitGroup
for i := 0; i < 1000; i++ {
	wg.Go(func() { s.Note(200 + (i%3)*100) })
}
wg.Wait()
fmt.Println(s.parCode)
```

```
map[200:334 300:333 400:333]
```

Les règles : le mutex vit **à côté** de la donnée qu'il protège, dans la même struct, en champ non exporté ; `Lock` puis `defer Unlock` sur la première ligne de la méthode ; le type est passé par pointeur (`go vet` refuse de copier un mutex). `sync.RWMutex` a en plus `RLock` pour les lecteurs simultanés, utile quand on lit mille fois plus qu'on n'écrit. `sync.Once` garantit qu'une initialisation (ouvrir un pool de connexions, charger une config) ne se fait qu'une fois, même appelée de partout : `once.Do(func() { ... })` trois fois n'exécute la fonction qu'à la première. Et `sync/atomic` (`atomic.Int64`, `atomic.Bool`) fait une opération indivisible sans verrou, pour un compteur ou un drapeau : une phrase suffit, `var total atomic.Int64` puis `total.Add(1)` et `total.Load()` remplacent le mutex quand il n'y a qu'un mot à protéger.

Comment choisir : **un channel quand une donnée passe** d'une goroutine à une autre (tâches, résultats, événements) ; **un mutex quand une donnée reste** et que plusieurs y touchent (cache, compteurs, config rechargée à chaud). Et `errgroup` (`golang.org/x/sync/errgroup`, hors bibliothèque standard mais maintenu par l'équipe Go) est le `WaitGroup` des cas réels : il collecte la première erreur, annule le contexte des autres goroutines, et `SetLimit(n)` en fait un worker pool en trois lignes. Regarde-le quand tu auras écrit deux fois la plomberie de 11.4.

### 11.8 Les six pièges

**Piège 1 : la goroutine qui fuit.** Une goroutine bloquée sur un channel que personne ne lira plus reste en mémoire pour toujours. Le cas typique : un `select` avec timeout qui abandonne, pendant que la goroutine lancée essaie encore d'envoyer sur un channel non bufferisé.

```go
func chercher(hote string) string {
	ch := make(chan string) // non bufferisé : l'envoi attend un lecteur
	go func() {
		time.Sleep(10 * time.Millisecond)
		ch <- hote + " trouvé" // personne ne lira : bloquée pour toujours
	}()
	select {
	case r := <-ch:
		return r
	case <-time.After(time.Millisecond):
		return hote + " : trop lent"
	}
}
```

Après cent appels, `runtime.NumGoroutine()` rend 101 : cent goroutines bloquées sur leur envoi, pour toujours. Deux remèdes : `make(chan string, 1)` (l'envoi ne bloque plus, la goroutine finit, le channel est ramassé), ou passer un contexte à la goroutine pour qu'elle abandonne aussi. Surveille `runtime.NumGoroutine()` dans tes tests et la métrique `go_goroutines` en prod : une courbe qui monte est une fuite.

**Piège 2 : le deadlock.** Toutes les goroutines attendent, aucune ne peut avancer. L'exécutif le détecte et tue le programme :

```go
ch := make(chan int)
ch <- 1 // personne ne lit : bloqué pour toujours
```

```
fatal error: all goroutines are asleep - deadlock!
goroutine 1 [chan send]:
```

Le message dit dans quel état est chaque goroutine (`chan send`, `chan receive`, `semacquire` pour un `Wait` ou un `Lock`). Cause habituelle : un envoi sans receveur, un `wg.Wait()` avec un `Done` oublié, ou un `Lock` pris deux fois.

**Piège 3 : fermer deux fois** et **piège 4 : écrire sur un channel fermé.** Les deux paniquent, immédiatement, sans rattrapage possible :

```
panic: close of closed channel
panic: send on closed channel
```

C'est pour ça que la règle « un seul producteur ferme » existe. Lire un channel fermé, en revanche, est toujours sûr.

**Piège 5 : partager sans protéger.** `c.n++` depuis deux goroutines, une map lue et écrite en même temps (`fatal error: concurrent map writes`, sans même `-race`). Le chapitre 10 t'a montré le détecteur : `go test -race` sur tout ce qui contient le mot `go`.

**Piège 6 : la variable de boucle.** Historique, mais tu le rencontreras dans du code ancien :

```go
for i := 0; i < 3; i++ {
	wg.Go(func() { fmt.Print(i, " ") })
}
```

```
$ go run .          # go.mod avec go 1.21
3 3 3
$ go run .          # go.mod avec go 1.22 ou plus
1 0 2
```

Avant Go 1.22, `i` était une seule variable partagée par toutes les itérations, et les goroutines la lisaient après la fin de la boucle. Depuis Go 1.22 (la version est celle du `go.mod`, pas celle du compilateur), chaque itération a sa propre variable. Le piège n'a pas disparu pour une variable déclarée **hors** de la boucle (`var hote string; for _, hote = range ...`) : là, toutes les goroutines voient la dernière valeur. Déclare dans la boucle, toujours.

### 11.9 Deux exemples complets

**Sonder N hôtes avec timeout.** Une goroutine par cible, chacune écrit dans sa propre case du tableau de résultats (pas de partage : la case `i` n'appartient qu'à la goroutine `i`), et le contexte borne le tout :

```go
type Resultat struct {
	Cible string
	OK    bool
	Duree time.Duration
	Err   error
}

func sonder(ctx context.Context, cible string, timeout time.Duration) Resultat {
	debut := time.Now()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", cible)
	r := Resultat{Cible: cible, Duree: time.Since(debut), Err: err}
	if err == nil {
		r.OK = true
		conn.Close()
	}
	return r
}

func sonderTous(ctx context.Context, cibles []string, timeout time.Duration) []Resultat {
	resultats := make([]Resultat, len(cibles)) // une case par cible : pas de partage
	var wg sync.WaitGroup
	for i, c := range cibles {
		wg.Go(func() { resultats[i] = sonder(ctx, c, timeout) })
	}
	wg.Wait()
	return resultats
}

debut := time.Now()
for _, r := range sonderTous(context.Background(), cibles, 500*time.Millisecond) {
	fmt.Printf("%-18s %-5v %6s  %v\n", r.Cible, r.OK, r.Duree.Round(time.Millisecond), r.Err)
}
fmt.Println("total :", time.Since(debut).Round(10*time.Millisecond))
```

```
127.0.0.1:49863    true     1ms  <nil>
127.0.0.1:1        false    1ms  dial tcp 127.0.0.1:1: connect: connection refused
192.0.2.1:80       false  502ms  dial tcp 192.0.2.1:80: i/o timeout
total : 500ms
```

Trois cibles (la première est un `net.Listen("tcp", "127.0.0.1:0")` ouvert par le programme lui-même sur un port libre, la deuxième un port fermé), dont une qui ne répondra jamais (192.0.2.1 est une adresse réservée à la documentation, aucun paquet ne revient) : 500 ms en tout, le temps du plus lent, pas la somme. `DialContext` respecte le contexte : un `cancel()` venu d'ailleurs interromprait la connexion en cours.

**Un pipeline lire → transformer → écrire**, avec l'étage du milieu parallélisé :

```go
type Entree struct{ Niveau, Message string }

func lire(r io.Reader) <-chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			out <- sc.Text()
		}
	}()
	return out
}

func transformer(in <-chan string, n int) <-chan Entree {
	out := make(chan Entree)
	var wg sync.WaitGroup
	for range n { // n goroutines lisent le même channel d'entrée
		wg.Go(func() {
			for l := range in {
				champs := strings.SplitN(l, " ", 3) // horodatage NIVEAU message
				if len(champs) < 3 {
					continue // ligne malformée : ignorée
				}
				out <- Entree{Niveau: champs[1], Message: champs[2]}
			}
		})
	}
	go func() { wg.Wait(); close(out) }() // fermer quand TOUS les workers ont fini
	return out
}

enc := json.NewEncoder(os.Stdout) // l'étage « écrire » : une ligne JSON par entrée
for e := range transformer(lire(strings.NewReader(journal)), 4) {
	enc.Encode(e)
}
```

```
{"Niveau":"WARN","Message":"web1 lent (1832 ms)"}
{"Niveau":"INFO","Message":"tour terminé"}
{"Niveau":"ERROR","Message":"db1:5432 connexion refusée"}
{"Niveau":"INFO","Message":"démarrage du sondeur"}
```

L'ordre de sortie n'est plus celui d'entrée : quatre goroutines transforment en parallèle, la première qui finit écrit la première. Si l'ordre compte, on numérote les lignes en entrée et on réordonne en sortie, comme dans le worker pool. `lire` lit un `io.Reader`, donc un fichier, un socket ou une chaîne, sans changer une ligne : le chapitre 12 s'en sert sur de vrais fichiers.

### 11.10 Pour le labo

Le [labo 11](../labs/11-pinger-parallele/README.md) fait de l'exemple des sondes un outil complet : une fonction `Sonder` bornée par un contexte, un `SonderTous` en worker pool avec un parallélisme réglable et des résultats dans l'ordre des cibles, l'annulation par contexte, et des tests qui montent un vrai serveur TCP local pour être déterministes.

### À retenir

- `go f()` lance et n'attend rien ; `sync.WaitGroup` (et `wg.Go` depuis 1.25) attend. `main` qui se termine tue tout.
- Un channel non bufferisé est un rendez-vous ; bufferisé, une file bornée. `close` par le producteur, `range` chez le consommateur, `v, ok := <-ch` pour distinguer la fin. `chan<-` et `<-chan` dans les signatures.
- `select` attend la première branche prête ; avec `time.After` c'est un délai, avec `default` un sondage. Worker pool : `n` goroutines qui lisent un channel de tâches ; pipeline : des étages qui rendent chacun un `<-chan`. Les deux ferment proprement de gauche à droite.
- `context.Context` en premier argument de tout ce qui attend ; `WithTimeout`/`WithCancel` dérivent, `defer cancel()` toujours, `ctx.Done()` dans un `select`, `ctx.Err()` pour la raison.
- Channel quand la donnée passe, mutex quand elle reste ; `sync.Once` pour l'initialisation, `atomic` pour un mot, `errgroup` quand il y a des erreurs à collecter.
- Les pièges : goroutine qui fuit (channel bufferisé ou contexte), deadlock (« all goroutines are asleep »), double `close`, envoi sur fermé, partage sans mutex (`-race`), variable déclarée hors de la boucle.

---

← [10. Sous le capot : compilation, mémoire, GC, goroutines](10-sous-le-capot.md) · [Sommaire](../README.md) · [12. Fichiers, JSON, ligne de commande DevOps](12-fichiers-json-cli.md) →
