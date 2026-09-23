# Labo 12 : CLI DevOps, `logstat`

**Chapitre** : [12. Fichiers, JSON, ligne de commande DevOps](../../cours/12-fichiers-json-cli.md)

## Objectif
Écrire `logstat`, un outil en ligne de commande qui analyse un fichier de log au format `2026-09-23T10:00:00Z NIVEAU message` (ou l'entrée standard) et répond à trois questions : combien de lignes par niveau (`count`), quels sont les messages les plus fréquents (`top`), et le tout en JSON pour `jq` (`json`). Avec le contrat Unix complet : sous-commandes et options via `flag.NewFlagSet`, résultat sur stdout, erreurs et logs `slog` sur stderr, codes de sortie 0, 1 et 2. Toute la logique est dans des fonctions testées sans fichier, et la ligne de commande elle-même est testée en appelant `run(args, stdin, stdout, stderr)` avec des tampons.

## Consignes
1. Lis `logstat.go` (la logique, six `// TODO`), `cli.go` (la ligne de commande, `run` et `main`) et `exemple.log` (30 lignes, 4 niveaux). Lance `go test .` : tout échoue, mais ça compile. Dans `run`, le squelette affiche l'usage et rend 2 : c'est déjà le bon comportement sans argument.
2. `ParseLigne(ligne string) (Entree, error)` : `strings.SplitN(ligne, " ", 3)`, moins de trois champs est une erreur, l'horodatage est lu avec `time.Parse(time.RFC3339, ...)`, le message garde ses espaces. `go test -run ParseLigne .`
3. `Lire(r io.Reader) ([]Entree, error)` : un `bufio.Scanner`, les lignes vides sont ignorées (mais comptent dans la numérotation), une ligne malformée arrête tout avec `fmt.Errorf("ligne %d : %w", numero, err)`, et n'oublie pas `sc.Err()` après la boucle. Un flux vide rend zéro entrée sans erreur.
4. `Compter(entrees) map[string]int` et `Top(entrees, n) []Frequence` : `Top` trie par nombre décroissant puis message croissant (une map n'a pas d'ordre : verse-la dans une slice et `sort.Slice`), coupe à `n`, rend une slice vide si `n <= 0`, tout si `n` dépasse le nombre de messages distincts.
5. `FormatCompte(parNiveau) string` : une ligne `"%-5s %d\n"` par niveau, dans l'ordre de `OrdreNiveaux` (`DEBUG`, `INFO`, `WARN`, `ERROR`) puis les niveaux inconnus par ordre alphabétique (`ordreNiveau` est fourni). `EncoderJSON(w, s) error` : `json.NewEncoder(w)`, `SetIndent("", "  ")`, `Encode`. Le test compare la chaîne exacte.
6. `run(args, stdin, stdout, stderr) int` dans `cli.go`, dans cet ordre : pas d'argument ou sous-commande inconnue, usage sur stderr et 2 ; `flag.NewFlagSet(cmd, flag.ContinueOnError)` avec `SetOutput(stderr)`, `-v` et `-n` (défaut 5), erreur de `Parse` et 2 ; un `slog.Logger` `TextHandler` sur stderr, niveau `Debug` si `-v` ; la source est `os.Open(fs.Arg(0))` s'il y a un argument (erreur : message sur stderr et 1, sinon `defer f.Close()`), `stdin` sinon ; `Lire`, erreur et 1 ; puis `count`, `top` (`"%5d  %s\n"` par ligne) ou `json` (`Stats{Total, ParNiveau, Top}`) sur stdout, et 0. Jamais `os.Exit` ni `os.Stdout` dans `run` : les tests te le feraient payer.
7. `go test .` puis `go vet ./...` et `gofmt -l .` : tout vert, rien à signaler. Essaie l'outil à la main sur `exemple.log`, en pipe (`grep ERROR exemple.log | go run . top`), sur un fichier absent, avec `-v`, et vérifie chaque code de sortie avec `echo $?`.

## Comment lancer
```bash
go test .                                   # tes tests, sans processus ni réseau
go run . count exemple.log
go run . top -n 3 exemple.log
go run . json -n 2 exemple.log | jq .par_niveau.ERROR
grep ERROR exemple.log | go run . top -n 2  # stdin
go run . count -v exemple.log 2>/dev/null   # les logs partent sur stderr, pas le résultat
go run . count nexiste.log ; echo "code $?"
go test ./solution/ && go run ./solution count exemple.log
```

## Sortie attendue
```
$ go run . count exemple.log
DEBUG 2
INFO  15
WARN  7
ERROR 6
$ go run . top -n 3 exemple.log
   11  requête servie
    5  connexion refusée db1:5432
    5  réponse lente web1
$ go run . json -n 1 exemple.log
{
  "total": 30,
  "par_niveau": {
    "DEBUG": 2,
    "ERROR": 6,
    "INFO": 15,
    "WARN": 7
  },
  "top": [
    {
      "message": "requête servie",
      "nombre": 11
    }
  ]
}
$ grep ERROR exemple.log | go run . top -n 2
    5  connexion refusée db1:5432
    1  disque plein /var/lib/postgresql
$ go run . count -v exemple.log 2>&1 >/dev/null
time=2026-09-23T12:30:00.000+02:00 level=DEBUG msg=lecture source=exemple.log commande=count
time=2026-09-23T12:30:00.000+02:00 level=DEBUG msg="analyse terminée" entrees=30
$ echo "cassée" | go run . count ; echo "code $?"
logstat : ligne 1 : format attendu : horodatage NIVEAU message
code 1
$ go run . grep ; echo "code $?"
logstat : sous-commande inconnue "grep"
usage : logstat <count|top|json> [-v] [-n N] [fichier]
...
code 2
```
(`go run` rend 1 quand le programme sort avec un code non nul et affiche `exit status N` ; pour voir le vrai code, `go build -o logstat . && ./logstat grep ; echo $?`.)

## Pour aller plus loin
- Ajoute une option `-since 2026-09-23T10:00:20Z` (un `flag.String` converti avec `time.Parse`) qui ignore les entrées antérieures : c'est l'occasion d'utiliser `Horodatage`, que l'outil lit mais n'exploite pas encore.
- Fais lire à `Lire` un fichier compressé quand le nom finit par `.gz` : `compress/gzip` rend un `io.Reader` à partir d'un autre, et rien ne change dans `ParseLigne`. C'est la force de l'interface `io.Reader`.
- Réécris `cli.go` avec cobra (`go get github.com/spf13/cobra`) : trois commandes, l'aide générée, la complétion shell. Compare le nombre de lignes et le nombre de dépendances dans `go.sum`, et décide à partir de quelle taille d'outil ça vaut le coup.
