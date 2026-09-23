# Labo 01 : Bonjour, machine

**Chapitre** : [1. Installer, outiller, lancer](../../cours/01-installer-et-outiller.md)

## Objectif
Écrire, lancer, compiler et compiler en croisé ton premier programme Go : un `hello` qui dit sur quelle machine et quel système il tourne. À la fin tu sais livrer un binaire Go pour n'importe quel serveur.

## Consignes
1. Lis `main.go`. Il compile et se lance tel quel (`go run .`), mais il n'affiche que la première ligne. Le module est déjà créé (`go.mod`) ; regarde son contenu.
2. Complète les `// TODO` : affiche le nom de la machine (`os.Hostname()` renvoie deux valeurs, un nom et une erreur ; pour ce labo, ignore l'erreur avec `_`), le système et l'architecture (`runtime.GOOS`, `runtime.GOARCH`), la version de Go (`runtime.Version()`) et le nombre de cœurs (`runtime.NumCPU()`). Il faudra ajouter `"os"` et `"runtime"` dans le bloc `import`.
3. Lance `go run .` puis `go vet ./...` : aucun message, c'est bon signe. Lance `gofmt -l .` : s'il affiche `main.go`, ton fichier n'est pas formaté, corrige avec `gofmt -w main.go` et regarde ce qui a changé.
4. Compile : `go build -o hello .` puis `./hello`. Regarde la taille du binaire avec `ls -la hello` et sa nature avec `file hello`.
5. Compile pour un serveur Linux et pour Windows : `GOOS=linux GOARCH=amd64 go build -o hello-linux .` et `GOOS=windows GOARCH=amd64 go build -o hello.exe .`. Vérifie avec `file` : le premier est un ELF x86-64 lié statiquement, le second un PE32+. Si tu as un serveur Linux sous la main, copie `hello-linux` dessus et lance-le : il n'a besoin de rien.
6. Bonus : `go build -ldflags="-s -w" -o hello-petit .` retire la table des symboles et les infos de débogage. Compare les tailles. C'est ce qu'on fait pour les binaires qu'on distribue.

## Comment lancer
```bash
go run .                 # ton programme
go run ./solution        # la version corrigée, commentée
go vet ./... && gofmt -l .
go build -o hello . && ./hello
GOOS=linux GOARCH=amd64 go build -o hello-linux . && file hello-linux
```

## Sortie attendue
Les valeurs dépendent de ta machine, la forme non :
```
$ go run .
Bonjour depuis Go.
Machine   : mbp-stranix.local
Système   : darwin/arm64
Go        : go1.27.1
Cœurs     : 8
```

## Pour aller plus loin
- `go tool dist list` affiche toutes les paires système/architecture supportées. Compte-les. Cherche `linux/riscv64`.
- `go build -x .` montre chaque commande que `go build` exécute. Tu y verras le compilateur (`compile`) et l'éditeur de liens (`link`), les mêmes rôles qu'avec `cc` et `ld`.
- Remplace `fmt.Println` par `fmt.Printf("Système   : %s/%s\n", runtime.GOOS, runtime.GOARCH)` et lance `go vet` après avoir volontairement retiré un argument : `vet` attrape l'erreur avant l'exécution. C'est le genre de filet que Go tend partout.
