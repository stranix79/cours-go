# Labo 03 : Devine le nombre

**Chapitre** : [3. Contrôle : if, for, switch, defer](../../cours/03-controle.md)

## Objectif
Un jeu de devinette au clavier : le programme tire un nombre entre 1 et 100, tu proposes, il répond « plus grand », « plus petit » ou « gagné », et à la fin il annonce le nombre d'essais. Tu vas écrire un `switch` sans expression, une boucle de lecture avec `bufio.Scanner`, un `defer` avec une closure, et surtout rendre l'aléatoire testable avec une graine.

## Consignes
1. Lis `main.go`. Le `main` lit déjà une graine optionnelle en argument (`go run . 42`) ; les trois fonctions et la boucle sont à écrire. Lance `go test .` : tout est rouge.
2. `NouveauSecret(graine int64, max int) int` renvoie un entier entre 1 et `max` inclus, tiré avec un générateur créé par `rand.New(rand.NewSource(graine))` (paquet `math/rand`). Deux appels avec la même graine doivent donner le même nombre. Avec la graine 42, le secret est 6.
3. `Evaluer(secret, proposition int) string` renvoie `"plus grand"` si le secret est plus grand que la proposition, `"plus petit"` s'il est plus petit, `"gagné"` sinon. Écris-la avec un `switch` sans expression (chapitre 3.3).
4. `Jouer(secret int, entrees []int) (essais int)` rejoue une partie sur une liste de propositions et renvoie le nombre d'essais jusqu'à la victoire. Si la liste est épuisée sans trouver, renvoie 0. Une proposition qui suit la victoire ne compte pas : `Jouer(42, []int{42, 1, 2})` vaut 1.
5. Dans `main`, ajoute le `defer` qui affiche `Essais : N` à la sortie, puis la boucle : `scanner := bufio.NewScanner(os.Stdin)`, `for scanner.Scan()`, `strconv.Atoi(scanner.Text())`. Une ligne qui n'est pas un nombre affiche `Ce n'est pas un nombre. ` et ne compte pas. Après « gagné », `return` : le `defer` fait le reste. Pense à ajouter `"bufio"` aux imports.
6. Le `defer` doit lire la valeur finale de `essais`. Teste d'abord `defer fmt.Println("Essais :", essais)` et regarde ce qu'il affiche : c'est le piège du chapitre 3.5. Corrige avec une fonction anonyme.
7. Lance `go test .` (vert), puis joue : `go run .` sans argument tire un nombre différent à chaque fois ; `go run . 42` donne toujours 6, ce qui permet de comparer avec la sortie attendue.

## Comment lancer
```bash
go test .                                  # tes fonctions
go test ./solution/                        # la solution
go run .                                   # partie aléatoire, Ctrl-D pour abandonner
go run . 42                                # partie reproductible (secret : 6)
printf '50\n25\n12\nabc\n6\n' | go run . 42   # une partie rejouée depuis un tube
go vet ./... && gofmt -l .
```

## Sortie attendue
```
$ printf '50\n25\n12\nabc\n6\n' | go run . 42
Je pense à un nombre entre 1 et 100 (graine 42).
Proposition ? plus petit
Proposition ? plus petit
Proposition ? plus petit
Proposition ? Ce n'est pas un nombre. Proposition ? gagné
Essais : 4
```
Les réponses se collent aux invites parce que l'entrée vient d'un tube : au clavier, chaque proposition est sur sa ligne.

## Pour aller plus loin
- Remplace la boucle de `main` par un appel à `Jouer` : il faudra d'abord lire toutes les propositions dans un slice. Qu'est-ce qu'on perd ? (Indice : les invites.) C'est la tension classique entre code testable et code interactif ; le chapitre 7 la résout avec les interfaces `io.Reader` / `io.Writer`.
- `math/rand/v2` (Go 1.22) est le successeur du paquet : `rand.New(rand.NewPCG(graine, 0))`. Lis `go doc math/rand/v2` et adapte `NouveauSecret`. Le secret pour la graine 42 change-t-il ?
- Ajoute une étiquette sur la boucle et remplace le `return` par un `break` étiqueté depuis un `switch` sur le verdict. Vérifie que le `defer` s'exécute toujours.
