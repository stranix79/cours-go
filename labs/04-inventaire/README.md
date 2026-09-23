# Labo 04 : Inventaire de serveurs

**Chapitre** : [4. Slices, maps, range : les collections sous le capot](../../cours/04-slices-maps.md)

## Objectif
Gérer un inventaire de serveurs, `map[string][]string` (rôle vers liste d'hôtes) : ajouter, retirer, lister trié, compter, détecter les hôtes présents dans plusieurs rôles. Et provoquer, puis corriger, le partage mémoire des sous-slices, le piège n°1 du chapitre.

## Consignes
1. Lis `main.go`. Le type `Inventaire` est un nom pour `map[string][]string`. Toutes les fonctions reçoivent la map en paramètre ; comme une map se comporte comme une référence, `Ajouter` et `Retirer` la modifient chez l'appelant sans rien renvoyer d'autre qu'un booléen. Lance `go test .`.
2. `Ajouter(inv, role, hote) bool` ajoute l'hôte à la fin de la liste du rôle (l'ordre d'ajout compte pour `Lister`) et renvoie `true`. Si l'hôte y est déjà, ne l'ajoute pas et renvoie `false`. Un rôle absent est créé au passage : `inv[role]` sur une clé absente donne un slice `nil`, sur lequel `slices.Contains` et `append` marchent. Il te faudra `import "slices"`.
3. `Retirer(inv, role, hote) bool` enlève l'hôte (`slices.Index` puis `slices.Delete(s, i, i+1)`, à réaffecter comme `append`) et renvoie `false` s'il n'était pas là. Un rôle qui se retrouve vide est supprimé de la map avec `delete`.
4. `Roles(inv) []string` renvoie les rôles triés : `slices.Sorted(maps.Keys(inv))` avec `import "maps"`. `Hotes(inv) []string` renvoie tous les hôtes, triés, sans doublon (`slices.Sort` puis `slices.Compact`, dans cet ordre : `Compact` ne retire que les doublons consécutifs). Les deux renvoient une liste vide pour un inventaire vide.
5. `CompterParRole(inv) map[string]int` et `Doublons(inv) []string` (les hôtes présents dans plus d'un rôle, triés ; `nil` s'il n'y en a pas). Pour les doublons, compte les apparitions dans une `map[string]int` : `vus[h]++` marche sans initialisation.
6. `Lister(inv) string` produit une ligne par rôle, rôles triés, hôtes dans l'ordre d'ajout, avec un `strings.Builder` : `"postgres: db01, db02\nweb: web01, web02\n"`.
7. `Premiers(hotes, n)` est fournie dans sa version naïve, `return hotes[:n]`. Lance `go test -v -run TestPremiers .` : deux sous-tests passent, deux échouent, et leurs messages expliquent pourquoi. Lance aussi `go run .` et regarde la ligne `Intacts` : l'`append` sur le résultat a écrasé un hôte de la liste d'origine. Corrige avec `make` + `copy` (ou `slices.Clone`), et relance les deux.
8. `go test .` vert, `go vet ./...`, `gofmt -l .`.

## Comment lancer
```bash
go test .                              # tes fonctions
go test -v -run TestPremiers .         # le piège des sous-slices, en détail
go test ./solution/                    # la solution
go run .                               # la démo
go run ./solution
go vet ./... && gofmt -l .
```

## Sortie attendue
```
$ go run .
monitoring: web01
postgres: db01, db02
web: web01, web02
Hôtes    : [db01 db02 web01 web02]
Doublons : [web01]
Comptes  : map[monitoring:1 postgres:2 web:2]
Rôles    : [postgres web]
Premiers : [db01 db02 intrus]
Intacts  : [db01 db02 web01 web02]
```
Avec la version naïve de `Premiers`, la dernière ligne affiche `[db01 db02 intrus web02]` : `web01` a été écrasé par l'`append`, parce que la sous-slice avait encore de la capacité dans le tableau de `tous`. Note que `Comptes` s'affiche trié par clé : `fmt` trie les maps à l'affichage, mais `range` ne le fait pas.

## Pour aller plus loin
- Remplace `make` + `copy` par `hotes[:n:n]` (la forme à trois index, chapitre 4.4). Les tests passent-ils ? Lequel échoue, et pourquoi ? (Indice : borner la capacité empêche l'`append` d'écraser, mais pas la modification en place.)
- Écris `RolesDe(inv, hote) []string` : les rôles d'un hôte, triés. C'est l'index inverse ; si tu l'appelles souvent, tu voudras une seconde map `map[string][]string` hôte vers rôles, maintenue par `Ajouter` et `Retirer`.
- `go test -bench . -benchmem` après avoir écrit un `BenchmarkHotes` (chapitre 9) : compare `make([]string, 0, n)` et `var tous []string` sur un inventaire de 10 000 hôtes. Le nombre d'allocations par appel est dans la colonne `allocs/op`.
