# cours-go : Go de zéro à la prod, pour qui a déjà codé

```
$ whoami
stranix · le renard qui aime savoir ce qui se passe sous le capot

$ cat cours-go/README.md
```

Tu as déjà codé. Du C, du Python, un peu de Java. Tu fais tourner des serveurs, des bases de données, des conteneurs, du monitoring. Et tu as remarqué que les outils que tu utilises tous les jours, Docker, Kubernetes, Prometheus, Terraform, Grafana Loki, Caddy, sont tous écrits dans le même langage. Ce n'est pas un hasard.

Ce dépôt est un cours **qui part de zéro en Go** mais pas de zéro en programmation. Chaque notion est comparée à ce que tu connais (C, Python), et chaque chapitre répond d'abord à la question que tu te poses vraiment : *pourquoi c'est comme ça, et dans quel cas je m'en sers ?* Le chapitre 0 est entièrement consacré à « quand utiliser Go, et quand ne pas l'utiliser ».

Tout a été écrit et exécuté avec **Go 1.27** sur un Mac Apple Silicon, mais rien n'est spécifique à macOS : Linux et Windows font aussi bien, et les binaires que tu produiras tourneront sur les trois.

## Comment ça marche

Dix-sept chapitres, seize labos. Le labo donne un squelette à compléter avec des `// TODO`, une solution commentée ligne par ligne dans `solution/`, et, à partir du labo 02, des tests `go test` qui te disent si c'est bon. Compte une à deux heures par chapitre.

| Chapitre | Labo |
|---|---|
| [0. Pourquoi Go, et quand l'utiliser (ou pas)](cours/00-pourquoi-go.md) | aucun |
| [1. Installer, outiller, lancer](cours/01-installer-et-outiller.md) | [01-hello](labs/01-hello/README.md) |
| [2. Variables, types, constantes, chaînes](cours/02-variables-types-chaines.md) | [02-convertisseur](labs/02-convertisseur/README.md) |
| [3. Contrôle : if, for, switch, defer](cours/03-controle.md) | [03-devine-nombre](labs/03-devine-nombre/README.md) |
| [4. Slices, maps, range : les collections sous le capot](cours/04-slices-maps.md) | [04-inventaire](labs/04-inventaire/README.md) |
| [5. Fonctions, erreurs et closures](cours/05-fonctions-erreurs.md) | [05-boite-a-outils](labs/05-boite-a-outils/README.md) |
| [6. Structs, méthodes, pointeurs](cours/06-structs-methodes-pointeurs.md) | [06-compte-bancaire](labs/06-compte-bancaire/README.md) |
| [7. Interfaces et composition](cours/07-interfaces.md) | [07-formes-et-io](labs/07-formes-et-io/README.md) |
| [8. Erreurs pour de vrai : wrapping, Is, As, panic](cours/08-erreurs-avancees.md) | [08-robuste](labs/08-robuste/README.md) |
| [9. Paquets, modules, tests](cours/09-paquets-modules-tests.md) | [09-mon-module](labs/09-mon-module/README.md) |
| [10. Sous le capot : compilation, mémoire, GC, goroutines](cours/10-sous-le-capot.md) | [10-sous-le-capot](labs/10-sous-le-capot/README.md) |
| [11. Concurrence : goroutines, channels, select, context](cours/11-concurrence.md) | [11-pinger-parallele](labs/11-pinger-parallele/README.md) |
| [12. Fichiers, JSON, ligne de commande DevOps](cours/12-fichiers-json-cli.md) | [12-cli-devops](labs/12-cli-devops/README.md) |
| [13. HTTP : client, serveur, API JSON](cours/13-http.md) | [13-api-todo](labs/13-api-todo/README.md) |
| [14. Bases de données : database/sql, SQLite, PostgreSQL](cours/14-bases-de-donnees.md) | [14-sqlite-postgres](labs/14-sqlite-postgres/README.md) |
| [15. Qualité, build, release, Docker, observabilité](cours/15-qualite-build-release.md) | [15-release](labs/15-release/README.md) |
| [16. Projet final et pour aller plus loin](cours/16-projet-final-et-suite.md) | [16-projet](labs/16-projet/README.md) |

Antisèches : [tools/go-antiseche.md](tools/go-antiseche.md) (la syntaxe sur deux pages) et [tools/c-python-vs-go.md](tools/c-python-vs-go.md) (la table de correspondance C / Python / Go).

## Démarrer en deux minutes

```bash
git clone https://github.com/stranix79/cours-go.git
cd cours-go
go version                      # 1.25 minimum, 1.27 recommandé
cd labs/01-hello
go run .                        # ton premier programme
go run ./solution               # la version corrigée, commentée
```

À partir du labo 02 : `go test .` dans le dossier du labo te dit si ta version passe, et `go test ./solution/` lance les mêmes tests sur la solution. Chaque labo est un module Go indépendant : pas de `venv`, pas de `pip`, le compilateur suffit.

## Ce que tu sauras faire à la fin

- dire, en une phrase, si un problème donné mérite du Go, du Python, du C ou du shell ;
- écrire du Go idiomatique, pas du C ni du Python traduit mot à mot ;
- structurer un projet en paquets et modules, avec des tests, un linter et un `Makefile` ;
- gérer les erreurs comme Go le veut, sans exceptions, sans se noyer dans les `if err != nil` ;
- lancer des milliers de goroutines et les faire communiquer proprement, avec annulation et délais ;
- écrire des outils en ligne de commande pour ton quotidien DevOps : fichiers, JSON, processus ;
- écrire un client et un serveur HTTP, une API JSON, un exporter Prometheus ;
- parler à SQLite et à PostgreSQL ;
- produire un binaire statique pour Linux, macOS et Windows, et l'emballer dans une image Docker de 10 Mo ;
- comprendre ce qui se passe sous le capot : compilation, pile, tas, ramasse-miettes, ordonnanceur.

## Pourquoi en français, pourquoi comme ça

Parce que les cours Go en français s'arrêtent à `fmt.Println`, et que la documentation officielle, excellente, suppose que tu sais déjà pourquoi tu es là. Ici, on commence par le pourquoi, on compare à ce que tu connais, et on va jusqu'au bout : un service qui tourne en production, dans un conteneur, avec ses métriques.

Licence MIT. Le cours est vivant : une erreur, une question, une idée de labo, ouvre une issue. On en parle aussi sur [stranix.net](https://stranix.net).

```
$ ls /ventures
asm-m1/    cours-python/    cours-go/    deckhand/
```
