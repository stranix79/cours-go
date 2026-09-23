# Labo 08 : Robuste

**Chapitre** : [8. Erreurs pour de vrai : wrapping, Is, As, panic](../../cours/08-erreurs-avancees.md)

## Objectif
Écrire un lecteur de configuration dont chaque erreur dit exactement ce qui s'est passé et où, testable par `errors.Is` et `errors.As`, et une fonction `Securise` qui transforme un panic en erreur. À la fin, tu sais construire une chaîne d'erreurs lisible (`charger config: lire /tmp/x/app.conf: fichier introuvable`) et l'interroger sans lire son texte.

## Consignes
1. Lis `config.go` et `main.go`. Deux fichiers d'exemple sont fournis : `exemple.conf` (valide) et `invalide.conf` (port non numérique). `go test .` est rouge, `go run . exemple.conf` affiche un `TODO`.
2. `ErreurValidation.Unwrap()` (TODO 1) : renvoie `ErrInvalide`. C'est ce qui rend `errors.Is(err, ErrInvalide)` vrai pour toute erreur de validation, sans que l'appelant connaisse le type.
3. `lire(chemin)` (TODO 2) : `os.ReadFile`. Si `errors.Is(err, fs.ErrNotExist)`, remplace par ta sentinelle `ErrIntrouvable` (l'appelant n'a pas à connaître `io/fs`) ; sinon emballe l'erreur du système telle quelle. Dans les deux cas le message commence par `lire CHEMIN: `.
4. `parser(data)` (TODO 3) : `bufio.NewScanner(bytes.NewReader(data))`, `strings.Cut(ligne, "=")`. Le numéro de ligne dans le message est vérifié (`TestChargerConfigParserInvalide` attend `ligne 2`).
5. `valider(valeurs)` (TODO 4) : les six cas et leurs textes exacts sont dans le commentaire du squelette et dans `TestChargerConfigValidation`. Chaque faute est un `&ErreurValidation{...}` emballé dans `fmt.Errorf("valider: %w", ...)`. Note : on ne remonte pas l'erreur de `strconv.Atoi`, elle parle de syntaxe Go, pas de configuration.
6. `ChargerConfig(chemin)` (TODO 5) : trois appels, trois `if err != nil { return Config{}, fmt.Errorf("charger config: %w", err) }`. Lance `go test -run TestChargerConfig .` et lis les messages : c'est à la chaîne exacte que les tests comparent.
7. `Securise(f)` (TODO 6) : retour nommé, `defer func() { if r := recover(); r != nil { ... } }()`. Si `r` est une `error` (assertion `r.(error)`), emballe avec `%w` ; sinon `%v`. Quatre sous-tests dans `TestSecurise`.
8. `go test .` vert, `gofmt -l .` vide, `go vet ./...` muet. Puis `go run . invalide.conf` et `go run . absent.conf` : lis les messages de haut en bas.

## Comment lancer
```bash
go test .                              # tes tests (rouge au départ)
go test -v -run TestChargerConfig .    # la chaîne d'erreurs, avec le détail
go test ./solution/                    # la solution (vert)
go run . exemple.conf                  # ton programme
go run ./solution exemple.conf         # la solution
go run ./solution invalide.conf        # une erreur de validation
go run ./solution absent.conf          # un fichier absent
go vet ./... && gofmt -l .
```

## Sortie attendue
```
$ go run ./solution exemple.conf
hote=db-01.prod port=5432 env=prod
$ go run ./solution invalide.conf
erreur : charger config: valider: champ port : pas un entier : pgsql
exit status 1
$ go run ./solution absent.conf
erreur : charger config: lire absent.conf: fichier introuvable
exit status 1
$ go run ./solution
erreur : usage : robuste FICHIER.conf
exit status 2
```

## Pour aller plus loin
- Dans `run`, si `errors.Is(err, ErrIntrouvable)`, écris un `exemple.conf` par défaut au lieu d'échouer. C'est exactement ce que font `git config` ou `gh` la première fois.
- Remplace le `%w` de `ChargerConfig` par `%v` et relance `go test .`. Combien de tests cassent, et pourquoi `errors.Is` ne voit plus rien ?
- Ajoute une validation qui accumule toutes les fautes avec `errors.Join` au lieu de s'arrêter à la première. Qu'est-ce que ça change pour `errors.As` quand deux champs sont faux ?
