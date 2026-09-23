# Labo 06 : Compte bancaire

**Chapitre** : [6. Structs, méthodes, pointeurs](../../cours/06-structs-methodes-pointeurs.md)

## Objectif
Écrire un type `Compte` avec ses méthodes, un `Journal` embarqué, un constructeur, et une `Banque` qui fait des virements entre comptes rangés dans une map. Et surtout : voir de tes yeux, dans un test qui échoue, ce qui se passe quand une méthode qui modifie a un receveur valeur.

## Consignes
1. Lis `compte.go`, `banque.go` et `main.go`. Tout compile (`go vet ./...` ne dit rien) mais les méthodes sont vides. Lance `go test .` : les tests échouent avec `TODO`. Lance `go run .` : le scénario s'arrête au premier TODO.
2. `Journal.Enregistrer(op)` et `Journal.Nombre()` (TODO 1 et 2) : `append` sur la slice `Operations`, et sa longueur. Choisis le receveur en te demandant si la méthode modifie le Journal.
3. `Compte.Deposer(centimes, libelle) error` (TODO 3) : refuse `centimes <= 0` avec `fmt.Errorf("déposer %d: %w", centimes, ErrMontantInvalide)`, sinon ajoute au solde et enregistre une `Operation{Type: "dépôt", ...}`. Puis `Solde()` (TODO 5). Lance `go test -run TestDeposer .` : si `TestDeposerModifieLeCompte` échoue alors que ta logique est juste, regarde le receveur.
4. `Compte.Retirer` (TODO 4) : même chose, plus `ErrSoldeInsuffisant` si le solde ne suffit pas. Un retrait refusé ne change rien et n'enregistre rien. Les tests vérifient les deux sentinelles avec `errors.Is`, donc emballe avec `%w`.
5. `FormatEuros(centimes) string` (TODO 7) : `150000` devient `1500,00 €`, `5` devient `0,05 €`. Puis `Compte.String()` (TODO 6) au format exact donné dans le commentaire du squelette (`%-14s` pour le type, signe, `%9s` pour le montant). `TestReleve` compare caractère par caractère.
6. `Banque.Ouvrir`, `Banque.Compte` (TODO 8 et 9) : la map contient des `*Compte`. Le pointeur que tu renvoies dans `Ouvrir` doit être celui que tu ranges dans la map (`TestVirement` le vérifie avec `!=` sur les pointeurs).
7. `Banque.Virement(de, vers, centimes)` (TODO 10) : retrouve les deux comptes, `Retirer` sur la source, `Deposer` sur la cible, chaque erreur emballée dans `fmt.Errorf("virement: %w", err)`. Si le retrait échoue, le dépôt ne doit pas avoir lieu. Après succès, change le `Type` de la dernière opération de chaque compte en `"virement émis"` / `"virement reçu"`.
8. `go test .` tout vert, `gofmt -l .` vide, `go vet ./...` muet. Compare ta version avec `solution/`, en particulier les receveurs et la façon dont chaque erreur est emballée.

## Comment lancer
```bash
go test .                    # tes tests contre ton code (rouge au départ)
go test -v -run TestDeposer . # un seul test, avec le détail
go test ./solution/          # les mêmes tests contre la solution (vert)
go run .                     # ton scénario
go run ./solution            # le scénario corrigé
go vet ./... && gofmt -l .
```

## Sortie attendue
```
$ go run ./solution
erreur : virement: retirer 1000,00 € (solde 500,00 €): solde insuffisant
erreur : virement: compte "carol": compte inconnu
Relevé de alice
  dépôt          +1500,00 €  salaire
  retrait        -  20,50 €  courses
  virement émis  - 500,00 €  vers bob
Solde : 979,50 €
Relevé de bob
  virement reçu  + 500,00 €  de alice
Solde : 500,00 €
opérations : 3 pour alice, 1 pour bob
```

## Pour aller plus loin
- Change le receveur de `Deposer` en valeur (`func (c Compte) Deposer`), relance `go test .` et lis le message de `TestDeposerModifieLeCompte`. Remets le pointeur. Tu ne feras plus jamais l'erreur.
- Remplace `map[string]*Compte` par `map[string]Compte` dans `Banque` et regarde ce que dit le compilateur sur `Virement`. Pourquoi les éléments d'une map ne sont-ils pas adressables ?
- Ajoute un champ `Date time.Time` à `Operation`, rempli par `time.Now()` dans `Enregistrer`. Comment garder `TestReleve` reproductible ? (Indice : une fonction `horloge func() time.Time` dans le `Journal`, remplacée dans le test.)
