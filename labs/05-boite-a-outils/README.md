# Labo 05 : Boîte à outils DevOps

**Chapitre** : [5. Fonctions, erreurs et closures](../../cours/05-fonctions-erreurs.md)

## Objectif
Écrire cinq fonctions réutilisables : un `Slugify`, un parseur `clé=valeur`, une somme variadique, un compteur en closure, et un `Retry` qui reçoit une fonction, la relance tant qu'elle échoue, et renvoie une erreur enveloppée. C'est le chapitre 5 au complet : retours multiples, variadique, fonction comme valeur, closure, convention `(T, error)`, `fmt.Errorf` avec `%w`.

## Consignes
1. Lis `main.go` et lance `go test .` : cinq tests rouges. Le `main` fait une démo de chaque fonction avec des valeurs fixes ; `go run .` te montre l'état d'avancement.
2. `Slugify(s string) string` : `"Serveur Web #1 (prod)"` donne `"serveur-web-1-prod"`. Minuscules (`strings.ToLower`), tout ce qui n'est pas lettre ou chiffre (`unicode.IsLetter`, `unicode.IsDigit`, sur des `rune` : parcours avec `range`) devient un tiret, jamais deux tirets de suite, jamais aux extrémités. `"Été 2026"` donne `"été-2026"` (les accents sont des lettres). Une chaîne vide ou sans lettre donne `""`. Construis le résultat avec un `strings.Builder`. Ajoute `"unicode"` aux imports.
3. `ParseKV(texte string) map[string]string` : `"host=web01 port=22"` donne `map[host:web01 port:22]`. Coupe sur les blancs avec `strings.Fields`, puis chaque morceau au premier `=` avec `strings.Cut` (qui renvoie trois valeurs, comme `partition` en Python). Un morceau sans `=` ou avec une clé vide est ignoré ; `"url=a=b"` donne `url` vers `"a=b"` ; en cas de clé répétée, la dernière gagne. Crée la map avec `make` : une map `nil` plante à l'écriture.
4. `Somme(nombres ...int) int` additionne ses arguments ; `Somme()` vaut 0. Le test l'appelle avec `Somme(c.nombres...)` : les trois points étalent un slice.
5. `NouveauCompteur() func() int` renvoie une fonction qui renvoie 1, puis 2, puis 3. Deux compteurs créés séparément ont chacun leur état : c'est la closure du chapitre 5.5.
6. `Retry(essais int, attente time.Duration, f func() error) error` appelle `f()` jusqu'à ce qu'elle renvoie `nil`, au plus `essais` fois, avec `time.Sleep(attente)` entre deux tentatives (pas après la dernière). Elle renvoie `nil` dès le premier succès. Si tout échoue, elle renvoie `fmt.Errorf("échec après %d essais : %w", essais, derniere)` : le `%w` est vérifié par le test avec `errors.Is`. Si `essais < 1`, renvoie `fmt.Errorf("retry : %w", ErrEssais)`. Le test « échoue deux fois puis réussit » compte les appels avec une closure, exactement comme la fonction `instable` de `main`.
7. `go test .` vert, `go run .` identique à la sortie attendue, `go vet ./...`, `gofmt -l .`.

## Comment lancer
```bash
go test .                        # tes fonctions
go test -v -run TestRetry .      # un test en détail
go test ./solution/              # la solution
go run .
go run ./solution
go vet ./... && gofmt -l .
```

## Sortie attendue
```
$ go run .
serveur-web-1-prod
map[host:web01 port:22 user:root]
6 0
1 2 3
instable : échec 1
instable : échec 2
instable : ok
retry : <nil>
retry : échec après 3 essais : service injoignable
```

## Pour aller plus loin
- Rends `Retry` progressive : l'attente double à chaque échec (*exponential backoff*), avec un plafond. C'est ce que font tous les clients HTTP sérieux. Ajoute un cas de test avec `time.Since`.
- `ParseKV` renvoie des chaînes. Écris `ParseKVInt(texte string) (map[string]int, error)` qui convertit chaque valeur avec `strconv.Atoi` et renvoie la première erreur rencontrée, enveloppée avec la clé fautive.
- `Slugify` garde les accents. Pour les retirer (`"été"` vers `"ete"`), cherche `golang.org/x/text/unicode/norm` et la décomposition NFD : c'est une dépendance hors bibliothèque standard, à ajouter avec `go get`, ce que le chapitre 9 détaille.
