# Labo 10 : Sous le capot

**Chapitre** : [10. Sous le capot : compilation, mémoire, GC, goroutines](../../cours/10-sous-le-capot.md)

## Objectif
Faire toi-même les quatre gestes du chapitre sur un petit paquet `capot` : lire l'analyse d'échappement, voir une course de données puis la corriger avec un mutex, mesurer avec un benchmark ce que coûte la concaténation naïve de strings, et trouver avec `pprof` où passe le temps d'un calcul. Le paquet est une bibliothèque (pas de `main`) : tout se lance par `go build`, `go test` et `go tool pprof`.

## Consignes
1. Lance `go test .` : tout échoue sauf le témoin `TestCompteurCasse`, qui est **sauté** (`SKIP`) tant que la variable d'environnement `RACE` ne vaut pas `1`. Lance `go vet ./...` et `gofmt -l .` : rien à signaler, le squelette compile.
2. **Échappement** (`echappe.go`). Complète `NouveauParValeur(nom string) Serveur` (une variable locale `s` avec `Nom: nom, Port: 5432`, renvoyée par valeur) et `NouveauParPointeur(nom string) *Serveur` (la même, renvoyée par `&s`). Puis :
   ```bash
   go build -gcflags='-m -l' .
   ```
   Tu dois lire `moved to heap: s` sur la ligne du `s :=` de `NouveauParPointeur`, et rien de tel pour `NouveauParValeur`. Explique-toi la différence avant de lire la solution : qui a besoin de `s` après le `return` ?
3. **Course de données** (`compteur.go`). `CompteurCasse` est fourni et volontairement faux ; `Marteler` (le banc d'essai) aussi. D'abord regarde la casse :
   ```bash
   RACE=1 go test -run Casse .            # échoue avec un nombre différent à chaque fois
   RACE=1 go test -race -run Casse .      # échoue à coup sûr, avec le rapport WARNING: DATA RACE
   ```
   Lis le rapport : il donne les deux accès (`Read at` / `Previous write at`), la ligne, et la goroutine qui a créé chacun. Puis complète `CompteurSur` : `Inc()` et `Valeur()` prennent `c.mu.Lock()` avec `defer c.mu.Unlock()`. `Valeur()` aussi : une lecture pendant une écriture est une course. Vérifie avec `go test -race -run Sur .`.
4. **Concaténation** (`concat.go`). `ConcatPlus(mots []string) string` assemble les mots séparés par un espace avec `+=` ; `ConcatBuilder` fait pareil avec `strings.Builder` (`WriteString`, puis `String()`). Aucun espace final ; `nil` donne `""`. Quand `go test -run Concat .` passe :
   ```bash
   go test -run XXX -bench=Concat -benchmem .
   ```
   Compare les colonnes `ns/op`, `B/op` et `allocs/op`. Le rapport entre les deux `allocs/op` est la vraie leçon : mille mots, deux mille allocations d'un côté, une douzaine de l'autre.
5. **Profil CPU** (`premiers.go`). Complète `EstPremier(n int) bool` (divisions successives tant que `d*d <= n` ; 0 et 1 ne sont pas premiers) et `ComptePremiers(max int) int` (les premiers strictement inférieurs à `max`). Quand `go test -run Premiers .` passe :
   ```bash
   go test -run XXX -bench=Premiers -cpuprofile=cpu.prof .
   go tool pprof -top -nodecount=6 10-sous-le-capot.test cpu.prof
   go tool pprof -list=EstPremier 10-sous-le-capot.test cpu.prof
   ```
   La première commande produit `cpu.prof` et le binaire de test `10-sous-le-capot.test` (le nom du dossier). `-top` doit montrer `EstPremier` en tête avec plus de 90 % du temps ; `-list` descend à la ligne : c'est le `n%d` qui coûte. `go tool pprof -http=:8080 10-sous-le-capot.test cpu.prof` ouvre la même chose en graphe dans le navigateur.
6. Termine par `go test -race .` : tout vert, le témoin sauté. Supprime `cpu.prof` et `*.test` avant de commiter (ou ajoute-les à `.gitignore`).

## Comment lancer
```bash
go test .                                          # tes tests
go test -race .                                    # avec le détecteur de courses
RACE=1 go test -race -run Casse .                  # le témoin : DOIT échouer
go build -gcflags='-m -l' .                        # l'analyse d'échappement
go test -run XXX -bench=. -benchmem .              # les trois benchmarks
go test ./solution/ && go test -race ./solution/   # la solution, commentée
```

## Sortie attendue
Les nanosecondes dépendent de ta machine ; les ordres de grandeur et le nombre d'allocations non :
```
$ go build -gcflags='-m -l' .
./echappe.go:20:23: leaking param: nom to result ~r0 level=0
./echappe.go:29:25: leaking param: nom
./echappe.go:30:2: moved to heap: s

$ RACE=1 go test -race -run Casse .
==================
WARNING: DATA RACE
Read at 0x00c000012345 by goroutine 8:
  cours-go/labs/10-sous-le-capot.(*CompteurCasse).Inc()
...
--- FAIL: TestCompteurCasse (0.01s)

$ go test -run XXX -bench=. -benchmem .
BenchmarkConcatPlus-10        	    1946	    588514 ns/op	 5362864 B/op	    1998 allocs/op
BenchmarkConcatBuilder-10     	  149739	      7696 ns/op	   17912 B/op	      13 allocs/op
BenchmarkComptePremiers-10    	     218	   5485428 ns/op	       0 B/op	       0 allocs/op

$ go tool pprof -top -nodecount=3 10-sous-le-capot.test cpu.prof
      flat  flat%   sum%        cum   cum%
    1060ms 96.36% 96.36%     1060ms 96.36%  cours-go/labs/10-sous-le-capot.EstPremier (inline)
      40ms  3.64%   100%     1100ms   100%  cours-go/labs/10-sous-le-capot.ComptePremiers (inline)
         0     0%   100%     1100ms   100%  cours-go/labs/10-sous-le-capot.BenchmarkComptePremiers

$ go test -race .
ok  	cours-go/labs/10-sous-le-capot	1.4s
```

## Pour aller plus loin
- Remplace le mutex de `CompteurSur` par un `atomic.Int64` (`sync/atomic`) : `Inc` devient `c.n.Add(1)`, `Valeur` devient `c.n.Load()`. Relance `go test -race .` puis compare avec un benchmark : sur un seul mot machine, l'atomique est plusieurs fois plus rapide que le verrou.
- Ajoute un profil mémoire au benchmark de concaténation : `-memprofile=mem.prof`, puis `go tool pprof -sample_index=alloc_space -top 10-sous-le-capot.test mem.prof`. Tu verras `ConcatPlus` allouer des mégaoctets là où `ConcatBuilder` en alloue des kilooctets.
- Lance le benchmark des premiers avec `GOGC=off` puis `GOMAXPROCS=1` devant la commande. Rien ne change : ce calcul n'alloue pas et ne parallélise pas. C'est le genre de vérification qui évite de « tuner » le GC d'un programme qui n'en a pas besoin.
