# Labo 07 : Formes et entrées-sorties

**Chapitre** : [7. Interfaces et composition](../../cours/07-interfaces.md)

## Objectif
Deux parties. D'abord une interface `Forme` et trois types qui la satisfont sans le dire, pour sentir la satisfaction implicite. Ensuite, la partie qui compte vraiment : une fonction qui lit depuis un `io.Reader` et une qui écrit dans un `io.Writer`, testées sans toucher un seul fichier, avec `strings.NewReader` et `bytes.Buffer`. À la fin tu sais écrire du code qui marche sur un fichier de 10 Go et se teste avec une chaîne de vingt caractères.

## Consignes
1. Lis `formes.go`, `io.go`, `main.go`. Tout compile, les corps sont vides. `go test .` est rouge.
2. `Rectangle`, `Cercle`, `Triangle` (TODO 1 à 3) : `Aire()`, `Perimetre()` et `String()`. Les textes attendus sont `rectangle 3x4`, `cercle r=1`, `triangle 3-4-5` (verbe `%g` : il affiche `3`, pas `3.000000`, et `2.5` tel quel). Pour le triangle, formule de Héron. Retire les deux lignes `var _ = ...` en bas de `formes.go` quand tu utilises `math` et `fmt` pour de vrai.
3. `Total(formes []Forme)` (TODO 4). Lance `go test -run 'TestFormes|TestTotal' .` : vert.
4. `Compter(r io.Reader) (lignes, mots, octets int, err error)` (TODO 5). Enveloppe `r` dans un `bufio.NewReader`, boucle sur `ReadString('\n')`. Cas limites dans les tests : entrée vide, pas de `\n` final (0 ligne, comme `wc`), lignes vides, tabulations, un `é` qui fait 2 octets. `io.EOF` termine la boucle sans erreur ; toute autre erreur est emballée avec `%w` (`TestCompterErreurDeLecture` passe un Reader qui échoue, et vérifie avec `errors.Is`).
5. `EcrireRapport(w io.Writer, formes []Forme) error` (TODO 6). Le format exact est dans le commentaire du squelette et dans `TestEcrireRapport`. Teste l'erreur de chaque `Fprintf` : `TestEcrireRapportErreurEcriture` passe un Writer qui refuse tout.
6. Regarde les types `lecteurCasse` et `ecrivainPlein` dans `io_test.go` : trois lignes chacun. C'est ce que permet une interface à une méthode.
7. `go test .` vert, `gofmt -l .` vide, `go vet ./...` muet. Puis compare avec `solution/`.

## Comment lancer
```bash
go test .                           # tes tests (rouge au départ)
go test -v -run TestCompter .       # la partie io seule, avec le détail
go test ./solution/                 # la solution (vert)
go run .                            # ton scénario
go run ./solution                   # le scénario corrigé
go vet ./... && gofmt -l .
```

## Sortie attendue
```
$ go run ./solution
rectangle 3x4    aire=   12.00  périmètre=   14.00
cercle r=1       aire=    3.14  périmètre=    6.28
triangle 3-4-5   aire=    6.00  périmètre=   12.00
total            aire=   21.14
4 lignes, 9 mots, 65 octets
```

## Pour aller plus loin
- Dans `main.go`, remplace `strings.NewReader(texte)` par `os.Stdin` et lance `go run ./solution < /etc/hosts`, puis compare avec `wc /etc/hosts`. Tu viens d'écrire `wc` sans changer `Compter`.
- Ajoute un type `Hexagone{Cote float64}` avec ses trois méthodes, sans toucher à `Total` ni à `EcrireRapport`. Combien de lignes as-tu dû modifier ailleurs ? Zéro, c'est le but.
- Branche `EcrireRapport` sur un `gzip.NewWriter(fichier)` comme dans le chapitre, et vérifie avec `gunzip -c`. N'oublie pas le `Close()` du gzip.
