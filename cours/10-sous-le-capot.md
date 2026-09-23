# 10. Sous le capot : compilation, mémoire, GC, goroutines

*Go de zéro à la prod : chapitre 10 sur 16.* ← [9. Paquets, modules, tests](09-paquets-modules-tests.md) · [Sommaire](../README.md) · [11. Concurrence : goroutines, channels, select, context](11-concurrence.md) →

Tu sais écrire du Go. Ce chapitre explique ce qui se passe quand il compile et quand il tourne, parce que c'est là que se prennent les décisions qui comptent en production : pourquoi ce binaire fait 2 Mo, pourquoi cette fonction alloue et pas celle-là, pourquoi le processus consomme 200 Mo alors que les données en font 50, pourquoi cent mille goroutines tiennent là où cent mille threads planteraient. Rien de tout ça n'est indispensable pour écrire un premier outil, mais tout est indispensable pour comprendre un outil qui se comporte bizarrement à 3 h du matin.

Le fil du chapitre : la chaîne de compilation, puis la mémoire (pile, tas, ramasse-miettes), puis l'exécution (goroutines et ordonnanceur), puis la représentation des types que tu manipules depuis le chapitre 4, et enfin les deux outils qui te diront la vérité : `pprof` et le détecteur de courses.

### 10.1 La chaîne de compilation

`go build` cache une chaîne classique : un compilateur, un éditeur de liens. L'option `-x` l'affiche. Sur le programme `bonjour` du chapitre 1 (sortie abrégée, les chemins raccourcis) :

```
$ go build -x -o hello .
WORK=/var/folders/.../T/go-build3126535936
mkdir -p $WORK/b001/
.../pkg/tool/darwin_arm64/compile -o $WORK/b001/_pkg_.a -trimpath "$WORK/b001=>" -p main -lang=go1.25 -complete ...
.../pkg/tool/darwin_arm64/link -o $WORK/b001/exe/a.out -importcfg $WORK/b001/importcfg.link ...
```

Deux outils, `compile` et `link`, écrits en Go, livrés avec Go. `compile` produit un fichier objet par paquet (`_pkg_.a`) ; `link` assemble ton paquet, la bibliothèque standard et l'exécutif (le ramasse-miettes, l'ordonnanceur, tout ce que ce chapitre décrit) en un exécutable. Tu ne vois pas `fmt` être compilé : il est déjà dans le cache de compilation (`go env GOCACHE`), partagé entre tous tes projets, et invalidé par empreinte de contenu. C'est pour ça qu'une compilation Go prend une seconde et pas une minute : seul ce qui a changé est recompilé, et « changé » est décidé par hachage, pas par date.

**Venant du C :** `compile` joue le rôle de `cc -c`, `link` celui de `ld`. Mais il n'y a ni préprocesseur ni fichiers d'en-tête : un paquet exporte ses symboles dans son fichier objet, et l'import d'un paquet lit cet objet. Pas de `#include` recompilé mille fois, pas de `-I`, pas de `-l`. Et `cgo`, le pont vers le C, est **désactivé quand on compile en croisé** et inutile tant qu'on n'importe pas `"C"` : la bibliothèque standard n'en a pas besoin, même pour le réseau et le DNS.

Le résultat est un binaire autonome. Sur Linux, il est réellement statique ; sur macOS, l'exécutif passe par `libSystem` (le noyau macOS l'impose), mais il n'y a aucune autre dépendance :

```
$ GOOS=linux GOARCH=amd64 go build -o hello-linux . && file hello-linux
hello-linux: ELF 64-bit LSB executable, x86-64, statically linked, Go BuildID=..., with debug_info, not stripped
$ ls -la hello hello-petit
2429746 hello
1587714 hello-petit        # go build -ldflags="-s -w"
```

Deux mégaoctets pour cinq lignes : l'exécutif, `fmt` et ce que `fmt` tire (`reflect`, `unicode`, `os`). `-ldflags="-s -w"` retire la table des symboles et les informations DWARF de débogage : un tiers de moins, et c'est ce qu'on livre. Le binaire conserve les noms de fonctions nécessaires aux traces de panique : `-s -w` ne rend pas les crashs illisibles, il empêche seulement `dlv` (le débogueur) de poser des points d'arrêt par ligne.

**Venant de Python :** il n'y a pas d'étape « compilation » visible en Python, mais il y en a une (le bytecode, chapitre 10 du cours Python). En Go, la compilation produit du code machine natif pour l'architecture cible ; il n'y a pas de machine virtuelle, pas de JIT, pas d'interpréteur au démarrage. Un binaire Go démarre en quelques millisecondes.

### 10.2 Pile ou tas : l'analyse d'échappement

En C, tu décides : une variable locale vit sur la pile, `malloc` vit sur le tas, et `return &locale` est un bug. En Go, tu n'as pas `malloc`, tu écris `&s` ou `make` ou `new` comme tu veux, et **le compilateur décide** où la valeur vit. La règle qu'il applique s'appelle l'analyse d'échappement : si une valeur peut être encore référencée après la fin de la fonction qui l'a créée (elle « s'échappe »), elle va sur le tas. Sinon, sur la pile, gratuite, libérée au `return`.

Pourquoi ça compte : la pile ne coûte rien (un décalage de pointeur), le tas coûte une allocation *et* du travail au ramasse-miettes plus tard. Un programme Go lent est très souvent un programme qui alloue trop. Et le compilateur te dit ce qu'il a décidé avec `-gcflags=-m` :

```go
type Serveur struct {
	Nom  string
	Port int
}

// parValeur renvoie une copie : le Serveur peut vivre sur la pile de l'appelant.
func parValeur(nom string) Serveur {
	s := Serveur{Nom: nom, Port: 5432}
	return s
}

// parPointeur renvoie l'adresse d'une variable locale : elle doit survivre
// à la fonction, donc elle s'échappe vers le tas.
func parPointeur(nom string) *Serveur {
	s := Serveur{Nom: nom, Port: 5432}
	return &s
}

// somme ne garde rien : le tableau reste sur la pile.
func somme(n int) int {
	tab := make([]int, 8)
	for i := range tab {
		tab[i] = i * n
	}
	total := 0
	for _, v := range tab {
		total += v
	}
	return total
}

func main() {
	a := parValeur("db1")
	b := parPointeur("db2")
	fmt.Println(a.Nom, b.Nom, somme(3))
}
```

```
$ go build -gcflags='-m -l' .
./main.go:12:16: leaking param: nom to result ~r0 level=0
./main.go:19:18: leaking param: nom
./main.go:20:2: moved to heap: s
./main.go:26:13: make([]int, 8) does not escape
./main.go:40:13: ... argument does not escape
./main.go:40:15: a.Nom escapes to heap
./main.go:40:22: b.Nom escapes to heap
./main.go:40:33: somme(3) escapes to heap
```

Ligne par ligne (les numéros sont ceux du fichier complet, avec `package` et `import` ; le `-l` désactive l'inlining pour que la lecture soit plus simple ; sans lui tu verrais aussi des lignes `can inline parValeur`, c'est le compilateur qui recopie les petites fonctions à l'endroit de l'appel) :

- `leaking param: nom to result` : dans `parValeur`, le paramètre `nom` ressort dans la valeur de retour. Pas d'allocation, juste une information pour l'appelant.
- `moved to heap: s` : c'est la ligne qui compte. Dans `parPointeur`, `s` est déplacée sur le tas parce qu'on renvoie son adresse. Ce qui serait un bug en C est légal en Go, et coûte une allocation.
- `make([]int, 8) does not escape` : la slice de `somme` reste sur la pile. Le compilateur y arrive parce que la taille est constante et petite ; `make([]int, n)` avec `n` variable irait sur le tas.
- `a.Nom escapes to heap`, `somme(3) escapes to heap` : les arguments passés à `fmt.Println` s'échappent, parce que `Println` prend des `...any` (une interface, section 10.5) et que le compilateur ne peut pas suivre ce qu'elle en fait. C'est le prix de `fmt` ; dans une boucle chaude on l'évite.

**Piège :** ne pas conclure « les pointeurs sont lents, je renvoie tout par valeur ». Une struct de 200 octets copiée à chaque appel coûte aussi. La règle pratique : renvoie par valeur les petites structs (quelques champs), par pointeur les grosses ou celles qui ont un état à partager (un mutex, une connexion). Et mesure avec `-gcflags=-m` quand ça compte.

### 10.3 Le ramasse-miettes

Ce qui vit sur le tas doit être libéré. Go le fait avec un ramasse-miettes (*garbage collector*, GC) **concurrent** : il tourne dans ses propres goroutines pendant que ton programme continue, sans l'arrêter, sauf deux très courtes pauses par cycle pour poser et retirer ses repères. L'algorithme, en deux phrases : il marque tout ce qui est joignable depuis les racines (piles, globales) en peignant les objets en trois couleurs (blanc = pas encore vu, gris = vu mais ses pointeurs pas encore suivis, noir = fini), puis balaie tout ce qui est resté blanc. Un mécanisme de barrière d'écriture fait que ton programme peut modifier des pointeurs pendant le marquage sans que le GC rate un objet.

Ce que tu contrôles tient en deux variables d'environnement :

- `GOGC` (100 par défaut) : le GC démarre un cycle quand le tas a grossi de ce pourcentage depuis la fin du cycle précédent. Avec 100, un programme dont les données vivantes font 50 Mo déclenche un cycle à 100 Mo. `GOGC=400` : moins de cycles, plus de mémoire. `GOGC=off` : jamais.
- `GOMEMLIMIT` (Go 1.19) : un plafond de mémoire ; le GC devient plus agressif à l'approche. C'est *la* variable pour un conteneur avec une limite : `GOMEMLIMIT=900MiB` dans un conteneur limité à 1 Gio, et l'OOM-killer ne passe plus.

Un programme qui alloue deux millions de petites structures en ne gardant que les mille dernières, et qui lit `runtime.MemStats` à la fin :

```go
var avant runtime.MemStats
runtime.ReadMemStats(&avant)
var fenetre []*Ligne
for i := 0; i < 2_000_000; i++ {
	l := &Ligne{Horodatage: time.Now(), Message: fmt.Sprintf("requête %d", i)}
	fenetre = append(fenetre, l)
	if len(fenetre) > 1000 {
		fenetre = fenetre[1:] // on oublie la plus vieille
	}
}
var apres runtime.MemStats
runtime.ReadMemStats(&apres)
fmt.Printf("Alloué au total  : %d Mo\n", apres.TotalAlloc/1024/1024)
fmt.Printf("Vivant maintenant: %d Ko\n", apres.HeapAlloc/1024)
fmt.Printf("Réservé à l'OS   : %d Mo\n", apres.Sys/1024/1024)
fmt.Printf("Cycles de GC     : %d\n", apres.NumGC-avant.NumGC)
fmt.Printf("Pause totale GC  : %s\n", time.Duration(apres.PauseTotalNs-avant.PauseTotalNs))
```

```
$ go run .
Alloué au total  : 181 Mo
Vivant maintenant: 1985 Ko
Réservé à l'OS   : 13 Mo
Cycles de GC     : 52
Pause totale GC  : 1.993968ms
Pause max        : 78.125µs
$ GOGC=400 go run .
Réservé à l'OS   : 29 Mo
Cycles de GC     : 12
Pause totale GC  : 510.63µs
$ GOGC=off go run .
Réservé à l'OS   : 196 Mo
Cycles de GC     : 0
$ GOGC=off GOMEMLIMIT=64MiB go run .
Réservé à l'OS   : 63 Mo
Cycles de GC     : 3
```

Lis les nombres : 181 Mo alloués, 2 Mo vivants, 13 Mo demandés à l'OS, 52 cycles, moins de 2 ms de pause au total et 78 µs au pire. C'est ce que « pauses sub-milliseconde » veut dire : un exporter Prometheus qui répond en 5 ms ne verra jamais son GC. Avec `GOGC=off`, tout reste en mémoire ; avec `GOMEMLIMIT`, le GC ne tourne que lorsqu'il le faut pour rester sous le plafond. Pour voir chaque cycle en direct : `GODEBUG=gctrace=1 ./programme`.

Les champs de `runtime.MemStats` qui servent en prod : `HeapAlloc` (vivant), `Sys` (réservé), `NumGC`, `PauseTotalNs`. Ce sont exactement les métriques `go_memstats_*` qu'expose le client Prometheus Go (chapitre 15), et c'est là que tu regardes en premier quand un service grossit.

**Venant du C :** pas de `free`, donc pas de double free, pas de use-after-free, pas de fuite par oubli. Il reste une façon de fuir : garder une référence dans une map ou une slice globale qu'on ne vide jamais. Le GC ne peut pas deviner que tu ne t'en serviras plus.

**Venant de Python :** CPython libère par comptage de références, immédiatement, et un GC ne passe que pour les cycles. Go n'a pas de compteur (donc pas de coût à chaque affectation, et les cycles ne posent aucun problème), mais la libération est différée jusqu'au prochain cycle. Conséquence : ne compte jamais sur « la mémoire redescend tout de suite », et ferme tes fichiers avec `defer`, pas en espérant un destructeur.

### 10.4 Goroutines et threads : l'ordonnanceur

Une goroutine est un fil d'exécution géré par l'exécutif Go, pas par le noyau. Trois différences avec un thread POSIX :

1. **Sa pile démarre à 2 Ko** et grandit à la demande (l'exécutif la recopie dans un bloc deux fois plus grand quand elle déborde). Un thread a une pile fixe de 8 Mo réservée d'avance.
2. **Elle est ordonnancée en espace utilisateur**, sans appel système : un changement de goroutine coûte quelques centaines de nanosecondes, un changement de thread quelques microsecondes.
3. **Elles sont multiplexées sur peu de threads** : c'est un ordonnanceur M:N. `GOMAXPROCS` threads exécutent du code Go en même temps (par défaut le nombre de cœurs, ou la limite CPU du conteneur depuis Go 1.25), et l'exécutif fait tourner les goroutines dessus. Quand une goroutine bloque sur le réseau, elle est mise de côté par le *netpoller* (`epoll` sur Linux, `kqueue` sur macOS) et le thread prend la suivante.

La preuve par cent mille :

```go
const n = 100_000
var wg sync.WaitGroup
for i := 0; i < n; i++ {
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Second) // simule une attente réseau
	}()
}
```

```
$ go run .
cœurs : 10  GOMAXPROCS : 10
goroutines vivantes : 100001
mémoire de piles    : 195 Mo (soit 2051 octets par goroutine)
durée totale : 2.1s
$ ps -M <pid>       # pendant l'exécution
threads OS : 13
```

Cent mille goroutines, treize threads système, 2 Ko chacune, deux secondes en tout. Le même programme avec cent mille `pthread_create` demanderait 800 Go de pile virtuelle et le noyau refuserait bien avant.

Ce que fait `go f()` sous le capot : l'exécutif alloue une structure `g` (l'état de la goroutine : pile, position, statut), la met dans la file d'exécution du processeur logique courant (`P`), et continue. Un thread (`M`) libre prendra la goroutine, l'exécutera jusqu'à ce qu'elle bloque, cède ou dépasse 10 ms (depuis Go 1.14, l'ordonnanceur est préemptif : une boucle infinie sans appel de fonction ne monopolise plus un cœur). Le mot-clé lui-même ne bloque jamais et ne rend rien : c'est le chapitre 11 qui explique comment récupérer un résultat.

**Venant de Python :** pas de GIL. Dix goroutines de calcul pur sur dix cœurs vont dix fois plus vite, sans `multiprocessing`, sans copier les données. Et les goroutines sont préemptives : pas d'`await` à placer, une fonction ordinaire devient concurrente avec un mot-clé devant. Le prix : une donnée partagée doit être protégée (section 10.7), là où l'asyncio garantissait qu'une coroutine n'est interrompue qu'à un `await`.

### 10.5 Ce qu'il y a dans une string, une slice, une map, une interface

Depuis le chapitre 4 tu passes des slices et des maps à des fonctions « par valeur » et tu vois quand même les modifications. Voici pourquoi, en octets. `unsafe.Sizeof` donne la taille de l'en-tête, pas des données :

```
string    : 16 octets
slice     : 24 octets
map       : 8 octets
interface : 16 octets
pointeur  : 8 octets
```

**Une string**, c'est un pointeur vers des octets immuables et une longueur. Copier une string, c'est copier 16 octets ; `s[:4]` produit un nouvel en-tête qui pointe dans les mêmes octets, sans copie :

```
s := "postgres"          t := s[:4]

s ─► [ ptr │ len=8 ]      t ─► [ ptr │ len=4 ]
        │                         │
        ▼                         ▼
      [p o s t g r e s]  ◄────────┘  (les mêmes octets)
```

**Une slice**, c'est un pointeur vers un tableau, une longueur et une capacité. Passer une slice à une fonction copie ces 24 octets ; la fonction écrit dans le même tableau. Mais si elle fait `append` au-delà de la capacité, un nouveau tableau est alloué et l'appelant ne le voit pas : c'est le piège du chapitre 4, et il s'explique ici.

```
sl := []int{1, 2, 3}      sl2 := sl[1:]

sl  ─► [ ptr │ len=3 │ cap=3 ]
          │
          ▼
        [ 1 │ 2 │ 3 ]
              ▲
          ┌───┘
sl2 ─► [ ptr │ len=2 │ cap=2 ]      sl2[0] = 99  modifie sl[1]
```

**Une map**, c'est un seul pointeur (8 octets) vers une table de hachage sur le tas. La copier, c'est copier le pointeur : une fonction qui reçoit une map modifie la map de l'appelant, toujours. Et une map `nil` (déclarée, jamais `make`) se lit mais ne s'écrit pas : `panic: assignment to entry in nil map`.

**Une interface**, c'est deux mots : un pointeur vers la description du type dynamique (et sa table de méthodes) et un pointeur vers la valeur.

```
var s Sondeur = &Postgres{...}

s ─► [ type=*Postgres │ data ] ─► Postgres{...}
           │
           └─► table : Sonder() → (*Postgres).Sonder
```

Deux conséquences que tu as déjà rencontrées : mettre une valeur dans une interface alloue souvent (la valeur est copiée sur le tas, c'est le `escapes to heap` de `fmt.Println` en 10.2), et une interface qui contient un pointeur `nil` typé n'est **pas** `nil`, parce que son mot « type » est rempli (le piège du chapitre 7 : `var p *Postgres; var s Sondeur = p; s != nil`).

**Venant du C :** une string Go est `struct { char *ptr; size_t len; }` sans `\0` final, une slice est `struct { T *ptr; size_t len, cap; }`. Tout est passé par valeur, comme en C, et ces valeurs contiennent des pointeurs, comme en C. Il n'y a pas de magie « par référence » : il y a des structs de deux ou trois mots.

### 10.6 Mesurer : pprof

Quand un programme est lent, on ne devine pas, on profile. Go embarque le profileur : `runtime/pprof` enregistre, `go tool pprof` lit. Un calcul volontairement naïf, compte de nombres premiers par divisions successives et comptage de mots :

```go
func main() {
	f, err := os.Create("cpu.prof")
	if err != nil {
		panic(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	texte := strings.Repeat("GET /metrics 200 OK ", 200_000)
	fmt.Println("mots distincts :", len(compteMots(texte)))
	fmt.Println("premiers < 3 000 000 :", comptePremiers(3_000_000))
}
```

```
$ go build -o prof . && ./prof
mots distincts : 4
premiers < 3 000 000 : 216816
$ go tool pprof -top -nodecount=8 prof cpu.prof
Duration: 406.33ms, Total samples = 260ms (63.99%)
      flat  flat%   sum%        cum   cum%
     200ms 76.92% 76.92%      210ms 80.77%  main.estPremier (inline)
      30ms 11.54% 88.46%       30ms 11.54%  runtime.madvise
      20ms  7.69% 96.15%       20ms  7.69%  strings.Fields
      10ms  3.85%   100%       10ms  3.85%  runtime.asyncPreempt
         0     0%   100%       20ms  7.69%  main.compteMots
         0     0%   100%      210ms 80.77%  main.comptePremiers (inline)
```

`flat` est le temps passé dans la fonction elle-même, `cum` le temps incluant ce qu'elle appelle. 77 % du temps dans `estPremier` : c'est là qu'il faut travailler, et nulle part ailleurs. `-list=estPremier` descend à la ligne :

```
$ go tool pprof -list=estPremier prof cpu.prof
      90ms       90ms     15:	for d := 2; d*d <= n; d++ {
     110ms      120ms     16:		if n%d == 0 {
```

Et `go tool pprof -http=:8080 prof cpu.prof` ouvre un graphe d'appels et un *flame graph* dans le navigateur. Le même outil lit les profils mémoire (`pprof.WriteHeapProfile`), et, dans un serveur HTTP, `import _ "net/http/pprof"` expose tout ça sur `/debug/pprof/` : tu profiles un service en production, en direct, sans le redémarrer (chapitre 13). Pour un bout de code isolé, un benchmark `go test -bench=. -cpuprofile=cpu.prof` produit le même fichier : c'est ce que fait le labo.

**Piège :** un profil trop court ne montre que l'exécutif (`pthread_cond_wait`, `kevent`, `madvise`). Le profileur échantillonne 100 fois par seconde ; en dessous d'une seconde de calcul, les chiffres sont du bruit. Fais tourner le calcul plus longtemps, ou profile un benchmark avec `-benchtime=5s`.

### 10.7 Le détecteur de courses

Deux goroutines qui lisent et écrivent la même variable sans synchronisation, c'est une *course* (*data race*), et en Go comme en C le résultat est indéfini : la valeur peut être fausse, la map peut se corrompre, le programme peut planter une fois par mois. Le compilateur ne le voit pas. Mais `-race` instrumente chaque accès mémoire et le signale à l'exécution :

```go
type Compteur struct{ n int }

func (c *Compteur) Inc() { c.n++ }

func Incrementer(c *Compteur) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				c.Inc()
			}
		}()
	}
	wg.Wait()
}
```

```
$ go test .
--- FAIL: TestCompteur (0.00s)
    compteur_test.go:9: attendu 100000, obtenu 62022
$ go test -race .
==================
WARNING: DATA RACE
Read at 0x00c0000122a8 by goroutine 101:
  exp/race.(*Compteur).Inc()
      compteur.go:8 +0x90
Previous write at 0x00c0000122a8 by goroutine 9:
  exp/race.(*Compteur).Inc()
      compteur.go:8 +0xa0
Goroutine 101 (running) created at:
  exp/race.Incrementer()
      compteur.go:15 +0x54
```

Sans `-race`, le test échoue avec un nombre différent à chaque fois (62022 ici : `c.n++` est trois instructions, lire, ajouter, écrire, et deux goroutines les entrelacent). Avec `-race`, le rapport nomme l'adresse, les deux accès, la ligne, et la goroutine qui a créé chacun. Le programme tourne cinq à dix fois plus lentement et consomme plus de mémoire : on ne livre pas un binaire `-race`, mais on lance **toujours** `go test -race ./...` en intégration continue. Le chapitre 11 donne les remèdes (mutex, channels, atomiques).

**Venant du C :** c'est ThreadSanitizer, le même moteur (`-fsanitize=thread`), intégré à l'outil standard. Sauf qu'en Go tu n'as rien à installer et que tout le monde l'utilise.

### 10.8 Pour le labo

Le [labo 10](../labs/10-sous-le-capot/README.md) est une série d'expériences guidées : lire la sortie de `-gcflags=-m` sur deux fonctions, faire échouer un compteur avec `go test -race` puis le réparer, mesurer avec un benchmark ce que coûte la concaténation de strings par rapport à `strings.Builder`, et profiler un calcul avec `go tool pprof`. Ce sont les quatre gestes que tu referas sur tes propres programmes.

### À retenir

- `go build` enchaîne `compile` (un objet par paquet, mis en cache par empreinte) et `link` (un binaire autonome). `-ldflags="-s -w"` retire un tiers de la taille ; `GOOS`/`GOARCH` compilent en croisé sans cgo.
- Le compilateur décide pile ou tas par analyse d'échappement ; `go build -gcflags=-m` le montre. `moved to heap` = une allocation. Renvoyer un pointeur vers une locale est légal, et coûte.
- Le GC est concurrent, à pauses de l'ordre de la centaine de microsecondes. `GOGC` règle la fréquence, `GOMEMLIMIT` plafonne : mets-le dans tout conteneur limité en mémoire.
- `runtime.MemStats` (`HeapAlloc`, `Sys`, `NumGC`, `PauseTotalNs`) est ce que tu regardes quand un service grossit ; ce sont les `go_memstats_*` de Prometheus.
- Une goroutine : 2 Ko de pile qui grandit, ordonnancée en espace utilisateur, multiplexée sur `GOMAXPROCS` threads. Cent mille goroutines = treize threads. Pas de GIL.
- Une string = pointeur + longueur ; une slice = pointeur + longueur + capacité ; une map = un pointeur ; une interface = type + valeur. Tout est passé par valeur, et ces valeurs contiennent des pointeurs.
- `pprof` répond à « où passe le temps » ; ne profile pas moins d'une seconde de calcul.
- `go test -race ./...` trouve les courses que le compilateur ne voit pas. En CI, toujours.

---

← [9. Paquets, modules, tests](09-paquets-modules-tests.md) · [Sommaire](../README.md) · [11. Concurrence : goroutines, channels, select, context](11-concurrence.md) →
