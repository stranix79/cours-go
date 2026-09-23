# Labo 02 : Convertisseur d'unités

**Chapitre** : [2. Variables, types, constantes, chaînes](../../cours/02-variables-types-chaines.md)

## Objectif
Écrire cinq petites fonctions de conversion (températures, octets en taille lisible, durées) et les afficher dans un tableau aligné. Tu vas manipuler `float64`, `int64`, `time.Duration`, les conversions explicites, `fmt.Sprintf` et ses verbes, et pour la première fois des tests : `go test .` te dit ce qui reste à faire.

## Consignes
1. Lis `main.go` : il compile et se lance tel quel, mais toutes les fonctions renvoient une valeur zéro. Lance `go test .` : les cinq tests échouent. Lance `go test -v .` pour voir les sous-tests un par un, avec leur nom.
2. `Fahrenheit(celsius float64) float64` renvoie `celsius × 9/5 + 32`, et `Celsius(fahrenheit float64) float64` fait l'inverse. Vérifie que `-40` donne `-40` dans les deux sens.
3. `TailleLisible(octets int64) string` renvoie `"512 o"` en dessous de 1024 (entier, sans décimale), puis `"1.0 Ko"`, `"1.5 Ko"`, `"10.0 Mo"`, `"2.5 Go"`, `"3.0 To"` (base 1024, une décimale, unités `o`, `Ko`, `Mo`, `Go`, `To`). Au-delà du To, on reste en To : `"2048.0 To"`. Attention aux conversions : `octets` est un `int64`, la division doit se faire en `float64`, et Go ne convertit rien tout seul.
4. `ParseDuree(texte string) (time.Duration, error)` utilise `time.ParseDuration` (qui comprend `"1h30m"`, `"45s"`, `"250ms"`) et renvoie `(durée, nil)` en cas de succès, ou `(0, erreur)` sinon. Enveloppe l'erreur avec `fmt.Errorf("durée %q invalide : %w", texte, err)` : le message final doit contenir le texte fautif. Une chaîne vide est invalide.
5. `Secondes(d time.Duration) int64` renvoie le nombre entier de secondes : `1h30m` donne `5400`, et `1500ms` donne `1` (tronqué). `time.Duration` est un `int64` de nanosecondes ; `d / time.Second` est encore un `Duration`, à convertir.
6. Quand `go test .` est vert, lance `go run .` et compare avec la sortie attendue. Puis `go vet ./...` et `gofmt -l .` (rien à afficher).

## Comment lancer
```bash
go test .                # tes fonctions (rouge tant que les TODO restent)
go test -v -run TestTailleLisible .   # un seul test, en détail
go test ./solution/      # la solution (vert)
go run .                 # le tableau de démo
go run ./solution
go vet ./... && gofmt -l .
```

## Sortie attendue
```
$ go run .
Températures
   100.0 °C =  212.0 °F
    37.0 °C =   98.6 °F
   -40.0 °C =  -40.0 °F
    98.6 °F =   37.0 °C
Tailles
               512 o = 512 o
              1536 o = 1.5 Ko
          10485760 o = 10.0 Mo
     3298534883328 o = 3.0 To
Durées
  1h30m  =   5400 s
  45s    =     45 s
  2h     =   7200 s
  abc    : durée "abc" invalide : time: invalid duration "abc"
```

## Pour aller plus loin
- `go doc time.Duration` : lis les méthodes `Hours`, `Minutes`, `Seconds` (qui renvoient des `float64`) et `String`. Que donne `fmt.Println(90 * time.Minute)` ? Pourquoi ?
- Remplace `%.1f` par `%.2f` dans `TailleLisible` et regarde combien de tests cassent. C'est le rôle d'un test : figer un comportement.
- Ajoute `Po` (péta-octet) au tableau `unites`. Un tableau `[...]string` recalcule sa taille tout seul, et la boucle utilise `len(unites)` : rien d'autre à changer. Ajoute un cas de test.
