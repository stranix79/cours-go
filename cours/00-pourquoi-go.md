# 0. Pourquoi Go, et quand l'utiliser (ou pas)

*Go de zéro à la prod : chapitre 0 sur 16.* [Sommaire](../README.md) · [1. Installer, outiller, lancer](01-installer-et-outiller.md) →

En 2007, trois ingénieurs de Google (Rob Pike, Ken Thompson, celui d'Unix et du C, et Robert Griesemer) attendent que leur programme C++ compile. Ça prend quarante-cinq minutes. Ils ont des serveurs à des milliers d'exemplaires, des programmes qui doivent gérer des dizaines de milliers de connexions en même temps, et des équipes de centaines de développeurs qui doivent pouvoir lire le code des autres. Le C++ compile lentement et laisse chacun inventer son dialecte. Python est agréable mais lent et mono-cœur. Java est lourd et cache la machine. Ils dessinent un langage pendant que la compilation tourne.

Le résultat, c'est Go : un langage **compilé**, avec un **ramasse-miettes**, une syntaxe qui tient sur une page, et la **concurrence** intégrée au langage plutôt que dans une bibliothèque. Douze ans plus tard, Docker, Kubernetes, Prometheus, Terraform, etcd, Consul, Vault, Caddy, Traefik, CockroachDB, Grafana Loki, Hugo, Gitea, Tailscale sont écrits en Go. Si tu fais de l'infra, tu utilises du Go tous les jours sans l'avoir choisi. Ce chapitre explique pourquoi ces projets l'ont choisi, et comment tu décides, toi, pour ton prochain outil.

### 0.1 Ce que Go fait différemment

Cinq choix de conception expliquent presque tout le reste du cours. Ne les apprends pas par cœur, on y reviendra chacun dans son chapitre, mais garde-les en tête : ce sont les réponses aux « mais pourquoi ? » que tu vas te poser.

**1. Compilé en un binaire unique, statique.** `go build` produit un exécutable qui contient tout : ton code, la bibliothèque standard, l'exécutif (le ramasse-miettes, l'ordonnanceur). Pas d'interpréteur à installer, pas de `venv`, pas de bibliothèque partagée à trouver au démarrage. Tu copies un fichier sur le serveur, tu le lances. Et la compilation croisée est intégrée : depuis ton Mac, `GOOS=linux GOARCH=arm64 go build` produit le binaire pour un Raspberry Pi.

**Venant du C :** c'est le confort que tu avais en C avec la liaison statique, sans le Makefile de quarante lignes, sans `autoconf`, et sans te demander quelle version de la glibc tourne sur la cible.

**Venant de Python :** c'est ce qui manque le plus à Python en production. Un script Python, c'est le script plus l'interpréteur plus les paquets plus leur version. Un programme Go, c'est un fichier.

**2. Typé statiquement, mais le compilateur devine.** Chaque variable a un type connu à la compilation, comme en C. Mais tu écris `port := 5432` et le compilateur comprend `int` tout seul. Le programme ne plante pas à l'exécution parce qu'une chaîne s'est glissée là où on attendait un nombre : il ne compile pas. Le compilateur est ton premier test.

**3. Un ramasse-miettes.** Tu ne fais ni `malloc` ni `free`. Tu crées des valeurs, le langage récupère la mémoire quand plus personne ne s'en sert. Le prix : un peu de mémoire et de temps processeur en plus qu'en C, et des pauses de l'ordre de la microseconde, pas de la seconde comme les vieux Java. Pour un outil d'infra, ce prix est presque toujours le bon.

**4. La concurrence dans le langage.** Le mot-clé `go` devant un appel de fonction lance cette fonction en parallèle du reste, dans une *goroutine* : un fil d'exécution qui coûte deux kilo-octets de mémoire, pas deux méga-octets comme un thread système. Tu peux en lancer cent mille. Les goroutines se parlent par des *channels*, des tuyaux typés. Un serveur qui gère dix mille connexions, c'est dix mille goroutines et un code qui se lit comme du code séquentiel. C'est LA raison pour laquelle Docker et Kubernetes sont en Go.

**5. Un langage volontairement petit.** Vingt-cinq mots-clés (C en a trente-deux, Python trente-cinq, C++ plus de quatre-vingt-dix). Pas de classes, pas d'héritage, pas d'exceptions, pas de surcharge d'opérateurs, pas de ternaire, une seule boucle. `gofmt` formate tout le monde pareil, donc il n'y a pas de débat sur les accolades. Résultat : tu peux ouvrir le code source de Kubernetes et le lire. Ce n'est pas vrai de tous les langages.

### 0.2 Le tableau qui compte

Voici comment Go se place par rapport à ce que tu connais. Les ordres de grandeur sont ceux d'un serveur ou d'un outil en ligne de commande typique ; ils varient, mais pas d'un facteur dix.

| | C | Python | Go |
|---|---|---|---|
| Vitesse d'exécution | 1× (référence) | 20 à 100× plus lent | 1,2 à 2× plus lent que C |
| Mémoire | tu gères tout | grosse, gérée | modérée, gérée (GC) |
| Temps de compilation | secondes à minutes | aucun | secondes, même sur un gros projet |
| Déploiement | binaire + libs partagées | interpréteur + venv + paquets | un seul binaire statique |
| Concurrence | threads POSIX, à la main | asyncio (un cœur) ou multiprocessing | goroutines + channels, tous les cœurs |
| Sécurité mémoire | non (débordements, use-after-free) | oui | oui (pas d'arithmétique de pointeur, bornes vérifiées) |
| Gestion des erreurs | codes de retour | exceptions | valeurs de retour `error`, explicites |
| Temps pour être productif | long | court | court : une semaine pour être à l'aise |
| Bibliothèque standard | libc, minimale | énorme | énorme et de qualité (HTTP, TLS, JSON, crypto, tests) |
| Écosystème typique | systèmes, embarqué | scripts, data, IA, web | infra, réseau, CLI, services cloud |

Deux lignes méritent une explication.

**Vitesse.** Go est compilé en code machine, comme C. Il est un peu plus lent parce que le ramasse-miettes travaille en arrière-plan et que le langage vérifie les bornes des tableaux. Pour un outil qui lit des logs, parle à une base de données ou sert des requêtes HTTP, cette différence ne se voit pas : le temps passe dans le réseau et le disque. Par rapport à Python, en revanche, c'est un autre monde : un parseur de logs qui prend dix minutes en Python prend dix secondes en Go, et utilise tous les cœurs sans effort.

**Concurrence.** En Python, `asyncio` te donne de la concurrence sur un seul cœur (le GIL, vu au chapitre 10 du cours Python). Pour utiliser huit cœurs, il faut huit processus. En Go, huit cœurs, c'est le comportement par défaut : l'exécutif répartit tes goroutines sur tous les processeurs de la machine. Tu écris `go traiter(fichier)` dans une boucle, et la machine est pleine.

### 0.3 Quand utiliser Go

Voici les cas où Go est le bon choix, presque sans discussion. Tu reconnaîtras ton quotidien.

**Les outils en ligne de commande que tu distribues.** Un binaire, zéro dépendance, compilé pour Linux, macOS et Windows en une commande. C'est pour ça que `kubectl`, `terraform`, `gh`, `hugo`, `restic`, `rclone` sont en Go. Si tu écris un outil pour ton équipe ou pour ton NAS, et que tu ne veux pas expliquer à chacun comment installer Python 3.12 et un venv, c'est du Go.

**Les services réseau.** Un serveur HTTP, une API, un proxy, un relais WebSocket, un serveur DNS, un agent qui écoute sur un port. La bibliothèque standard contient un serveur HTTP de qualité production (avec TLS, HTTP/2), un client, JSON, et les goroutines font que chaque connexion est traitée en parallèle sans que tu y penses. Caddy, Traefik, le hub de Deckhand : du Go.

**Tout ce qui touche à l'infra et au cloud.** Exporters Prometheus, opérateurs Kubernetes, contrôleurs, agents de supervision, collecteurs de logs, outils de sauvegarde, CLI qui parlent à une API cloud. L'écosystème est là (le client Kubernetes officiel, le SDK AWS, la bibliothèque Prometheus), et un agent en Go consomme quinze méga-octets de RAM là où le même en Python en prend cent.

**Les programmes qui font beaucoup de choses en même temps.** Sonder mille hôtes, télécharger deux cents fichiers, traiter un flux de messages, servir dix mille clients. Si la phrase contient « en parallèle », « en même temps » ou « plusieurs milliers de connexions », c'est du Go.

**Les longs processus qui doivent rester stables.** Un démon qui tourne des mois. Typage statique, pas d'exception surprise à la ligne 4 000 après trois semaines, une mémoire qui ne fuit pas si tu suis les règles, un profileur intégré (`pprof`) pour voir où ça chauffe.

**Le code que d'autres devront lire.** Un langage petit, un formatage unique, pas de magie. Dans deux ans, toi ou un collègue reprendrez ce code sans se demander « mais qu'est-ce que ce décorateur fait, au juste ? ».

### 0.4 Quand ne pas utiliser Go

Un bon outil se reconnaît aussi à ce qu'on sait ne pas lui demander.

**Le script de dix lignes.** Renommer des fichiers, transformer un CSV une fois, appeler une API pour voir ce qu'elle répond. Python ou le shell gagnent : tu ouvres un fichier, tu écris, tu lances. Go te demande un `go.mod`, un `package main`, une fonction `main`, et un `if err != nil` à chaque ligne. Pour un jetable, c'est de la friction inutile. Règle simple : si tu ne vas pas le relancer dans un mois, ce n'est pas du Go.

**La data, la science, l'IA.** NumPy, pandas, PyTorch, scikit-learn, Jupyter : cet écosystème est en Python et il n'est pas près d'en bouger. Go a des bibliothèques, mais elles sont dix ans derrière. Si tu dois manipuler des tableaux de nombres ou entraîner un modèle, c'est Python, sans état d'âme.

**Le bas niveau dur.** Un pilote de périphérique, un noyau, un microcontrôleur avec huit kilo-octets de RAM, un code où chaque cycle et chaque octet comptent, ou qui doit tourner sans ramasse-miettes du tout. C'est C, ou Rust. Go a besoin de son exécutif (quelques méga-octets) et fait des pauses de GC, minuscules mais réelles. (TinyGo existe pour les microcontrôleurs, mais c'est un sous-ensemble.)

**Les interfaces graphiques.** Go n'a rien de sérieux pour faire une application avec des fenêtres. Une app macOS, c'est Swift ; une app web, c'est du JavaScript/TypeScript côté navigateur (et Go peut très bien servir l'API derrière).

**Le code qui a besoin d'une expressivité que Go refuse.** Go n'a pas d'exceptions, pas d'héritage, pas d'ensembles (`set`), pas de compréhensions de liste, des génériques limités, et il t'oblige à écrire des choses que Python fait en une ligne. Si ton problème est essentiellement « transformer des structures de données compliquées », Python sera trois fois plus court et tout aussi rapide (parce que le temps ne passera pas là). Go est verbeux là où Python est dense, c'est un choix : il préfère le code ennuyeux et lisible au code astucieux.

**Quand ton équipe ne le connaît pas et que le projet dure trois semaines.** Le coût d'apprentissage est faible (ce cours), mais pas nul. Pour un projet court dans une équipe Python, reste en Python.

### 0.5 La règle de décision, en une phrase par cas

Quand tu hésites, réponds à ces questions dans l'ordre. La première qui dit oui décide.

1. C'est jetable, ou ça tient en vingt lignes ? **Shell ou Python.**
2. C'est de la data, de la science, de l'IA ? **Python.**
3. Ça tourne sans système d'exploitation, ou chaque octet compte ? **C ou Rust.**
4. Ça a des fenêtres ? **Swift, Kotlin, TypeScript selon la plateforme.**
5. C'est un outil à distribuer, un service réseau, un agent, un truc qui doit tourner longtemps, ou faire plein de choses en parallèle ? **Go.**
6. Aucune des réponses ci-dessus ? Prends le langage que ton équipe lit le mieux.

Et le cas fréquent où les deux marchent : tu as un script Python qui a grandi, qui tourne toutes les nuits, qui prend vingt minutes, et que tu dois maintenant installer sur cinq machines. C'est exactement le moment de le réécrire en Go. Ce cours est fait pour ça.

### 0.6 Ce que Go n'est pas

Trois idées reçues à évacuer avant de commencer.

**« Go, c'est du C moderne. »** Non. Go ressemble au C (accolades, types explicites, pointeurs), mais il a un ramasse-miettes, pas d'arithmétique de pointeurs, des chaînes de caractères dignes de ce nom, des tableaux qui connaissent leur taille, et la concurrence intégrée. Si tu écris du Go comme du C, avec des index partout et des structures allouées à la main, tu écriras du mauvais Go. Le chapitre 4 sur les slices est là pour désapprendre.

**« Go, c'est orienté objet. »** Pas au sens Java. Il y a des types avec des méthodes et des interfaces, mais pas de classes, pas d'héritage, pas de constructeur, pas de `this`. On compose des structures au lieu de les faire hériter. C'est déroutant une semaine, puis on trouve ça plus simple. Chapitres 6 et 7.

**« Go, c'est verbeux à cause des `if err != nil`. »** Oui, il y en a beaucoup. C'est voulu : chaque erreur possible est visible dans le code, à l'endroit où elle peut arriver, et tu décides quoi en faire. Après une semaine, tu ne les vois plus ; après un mois, tu les regrettes dans les autres langages, parce que tu sais que l'exception qui va planter ton script Python est cachée quelque part. Chapitres 5 et 8.

### 0.7 Comment lire ce cours

Le cours suit un fil : **chapitre puis labo**. Lis le chapitre, fais le labo (un squelette à compléter avec des `// TODO`, des tests `go test` qui disent si c'est bon, une solution commentée), passe au suivant.

Les chapitres 1 à 9 posent le langage. Ils sont écrits pour quelqu'un qui vient du C et de Python : chaque notion est comparée à ce que tu connais, et les pièges de traduction sont signalés.

Les chapitres 10 et 11 font de toi quelqu'un qui *comprend* Go : ce qui se passe à la compilation, en mémoire, dans l'ordonnanceur, puis la concurrence pour de vrai, avec ses règles.

Les chapitres 12 à 16 sont ton quotidien : outils en ligne de commande, fichiers et JSON, HTTP client et serveur, bases de données, qualité, build, Docker, métriques, et un projet final qui rassemble tout.

Le code est en Go 1.27. Tout ce qui est dans un bloc `go` a été compilé et exécuté ; la sortie est donnée juste après. Quand une notion a un équivalent en C ou en Python, il est donné dans un encadré **Venant du C** ou **Venant de Python**. Quand Go fait quelque chose qui te paraîtra bizarre, c'est signalé par **Piège**. Il y en a une quinzaine dans tout le cours ; les connaître d'avance fait gagner des jours.

Compte une à deux heures par chapitre. Ne saute pas les labos : lire du Go donne l'impression de comprendre, écrire du Go et se faire disputer par le compilateur fait comprendre.

### À retenir

- Go est compilé, typé statiquement, avec un ramasse-miettes et la concurrence dans le langage : conçu chez Google pour des serveurs et des outils maintenus par des équipes nombreuses.
- Un programme Go, c'est un seul binaire statique, compilé pour n'importe quel OS depuis n'importe quel OS.
- Go est presque aussi rapide que C, vingt à cent fois plus rapide que Python, et utilise tous les cœurs sans effort.
- Utilise Go pour : outils CLI distribués, services réseau, agents et exporters, tout ce qui est parallèle, tout ce qui doit tourner longtemps.
- N'utilise pas Go pour : le script jetable, la data et l'IA, le bas niveau sans OS, les interfaces graphiques.
- Le signal le plus fiable : un script Python qui a grandi, qui tourne tous les jours et qu'il faut installer ailleurs.
- Go est petit et volontairement ennuyeux : c'est ce qui rend le code des autres lisible.

---

[Sommaire](../README.md) · [1. Installer, outiller, lancer](01-installer-et-outiller.md) →
