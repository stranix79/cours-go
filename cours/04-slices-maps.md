# 4. Slices, maps, range : les collections sous le capot

*Go de zéro à la prod : chapitre 4 sur 16.* ← [3. Contrôle : if, for, switch, defer](03-controle.md) · [Sommaire](../README.md) · [5. Fonctions, erreurs et closures](05-fonctions-erreurs.md) →

Go a trois collections dans le langage : le **tableau** (taille fixe, presque jamais utilisé directement), le **slice** (la « liste » de Go, qui est en réalité une fenêtre sur un tableau) et la **map** (le dictionnaire). Pas de set, pas de tuple, pas de liste chaînée dans le langage. Ce chapitre explique comment un slice est fait en mémoire, parce que c'est la seule façon de comprendre le piège n°1 de Go, celui que tout le monde rencontre la première semaine : deux slices qui partagent le même tableau, et une modification de l'un qui apparaît dans l'autre.

### 4.1 Tableaux : une valeur, taille comprise

Un tableau Go a une taille fixe, et cette taille fait **partie du type** : `[3]int` et `[4]int` sont deux types différents, incompatibles. Un tableau est une **valeur** : l'affecter ou le passer à une fonction le copie en entier.

```go
var ports [3]int         // [0 0 0], valeur zéro de chaque case
ports[0] = 22
dns := [2]string{"1.1.1.1", "8.8.8.8"}
copie := dns             // copie complète
copie[0] = "9.9.9.9"
fmt.Println(dns, copie)  // [1.1.1.1 8.8.8.8] [9.9.9.9 8.8.8.8]
auto := [...]int{80, 443, 8080} // le compilateur compte : [3]int
```

**Venant du C :** c'est le point de rupture avec le C, où un tableau est un pointeur déguisé qui oublie sa taille dès qu'on le passe à une fonction. En Go, le tableau connaît sa taille, la vérifie à chaque accès (index hors bornes = `panic`, pas de débordement silencieux), et se copie comme un `int`. Tu n'écriras presque jamais de tableau : dans le code Go de tous les jours, ce sont des slices. Les tableaux servent de stockage derrière les slices, et pour quelques cas où la taille fixe est le sens même de la chose (une adresse IPv4 : `[4]byte`, un hash SHA-256 : `[32]byte`).

### 4.2 Slices : un pointeur, une longueur, une capacité

Un slice est une **vue** sur un tableau. Concrètement, c'est une petite structure de trois mots :

```
slice "hotes"                    tableau sous-jacent (en mémoire)
+-----------+                    +-------+-------+-------+-------+
| pointeur  | -----------------> | web01 | web02 | db01  | db02  |
| len   = 4 |                    +-------+-------+-------+-------+
| cap   = 4 |                      [0]     [1]     [2]     [3]
+-----------+
```

- le **pointeur** vers le premier élément visible ;
- `len`, la **longueur** : le nombre d'éléments qu'on voit, et qu'on a le droit d'indexer ;
- `cap`, la **capacité** : le nombre d'éléments disponibles dans le tableau à partir du pointeur, donc la place avant qu'il faille réallouer.

Cette structure fait 24 octets, et c'est elle qui est copiée quand on passe un slice à une fonction, pas les données. Deux slices peuvent pointer sur le même tableau. C'est toute l'histoire de ce chapitre.

```go
hotes := []string{"web01", "web02"}  // littéral : pas de taille entre les crochets
fmt.Println(hotes, len(hotes), cap(hotes))
var vide []string                    // valeur zéro : nil, len 0
fmt.Println(vide == nil, len(vide))
hotes = append(hotes, "db01")
fmt.Println(hotes, len(hotes), cap(hotes))
fmt.Println(hotes[1:3], hotes[:1], hotes[1:])
```

```
$ go run .
[web01 web02] 2 2
true 0
[web01 web02 db01] 3 4
[web02 db01] [web01] [web02 db01]
```

`[]string` sans taille est un slice ; `[2]string` avec taille est un tableau. Le slice `nil` (valeur zéro) est utilisable tel quel : `len` vaut 0, `range` ne fait rien, `append` marche. On ne teste presque jamais `== nil`, on teste `len(s) == 0`.

Les sous-slices `s[debut:fin]` fonctionnent comme en Python : début inclus, fin exclue, chaque borne optionnelle. Mais contrairement à Python, **elles ne copient rien** : elles fabriquent une nouvelle structure (pointeur, len, cap) qui regarde le même tableau. Voir 4.4.

### 4.3 `append` et la réallocation

`append(s, valeurs...)` ajoute à la fin et **renvoie le slice résultant**, qu'il faut réaffecter : `s = append(s, x)`. Pourquoi renvoyer ? Parce que si le tableau est plein (`len == cap`), `append` en alloue un plus grand, y copie tout, et le nouveau slice pointe ailleurs. Regarde l'adresse du premier élément et la capacité :

```go
var s []int
for i := 0; i < 10; i++ {
	s = append(s, i)
	fmt.Printf("len=%d cap=%d %p\n", len(s), cap(s), &s[0])
}
```

```
$ go run .
len=1 cap=1 0x5080bc90e020
len=2 cap=2 0x5080bc90e040
len=3 cap=4 0x5080bc938020
len=4 cap=4 0x5080bc938020
len=5 cap=8 0x5080bc916040
len=6 cap=8 0x5080bc916040
len=7 cap=8 0x5080bc916040
len=8 cap=8 0x5080bc916040
len=9 cap=16 0x5080bc93a000
len=10 cap=16 0x5080bc93a000
```

La capacité double à chaque fois qu'elle est dépassée (1, 2, 4, 8, 16), et l'adresse change à ce moment-là : le tableau a été réalloué. Entre deux réallocations, `append` écrit simplement dans la place libre. C'est amorti en O(1) par ajout, comme `list.append` en Python et `std::vector` en C++.

**Piège :** oublier de réaffecter. `append(s, x)` seul compile (le résultat est jeté) et `s` n'a pas changé. `go vet` le signale, mais prends le réflexe : `s = append(s, x)`, toujours.

**Venant du C :** c'est le `realloc` que tu écrivais à la main, avec la stratégie de doublement, sans le `if (!p) goto fail`. Et quand tu connais la taille finale, `make` (4.5) évite toutes les réallocations.

### 4.4 Le piège n°1 : les sous-slices partagent la mémoire

Voici le programme que tout débutant Go écrit un jour, et qui le surprend :

```go
hotes := []string{"web01", "web02", "db01", "db02"}
bases := hotes[2:]       // une vue sur les deux derniers
bases[0] = "pg01"        // on modifie la vue...
fmt.Println(hotes)       // ...et l'original change
fmt.Println(bases, len(bases), cap(bases))
```

```
$ go run .
[web01 web02 pg01 db02]
[pg01 db02] 2 2
```

Rien n'a été copié : `bases` est une structure de 24 octets dont le pointeur vise la case 2 du tableau de `hotes`.

```
hotes                     tableau sous-jacent
+-----------+             +-------+-------+-------+-------+
| ptr       | ----------> | web01 | web02 | pg01  | db02  |
| len = 4   |             +-------+-------+-------+-------+
| cap = 4   |                               ^
+-----------+                               |
bases                                       |
+-----------+                               |
| ptr       | ------------------------------+
| len = 2   |
| cap = 2   |
+-----------+
```

Ça se complique avec `append`. Un `append` sur une sous-slice qui a encore de la capacité **écrit dans le tableau partagé**, et écrase ce que l'original avait là :

```go
web := hotes[:2]                 // [web01 web02], len 2, cap 4 : il reste 2 cases
web = append(web, "cache01")     // écrit dans hotes[2] !
fmt.Println(hotes)               // [web01 web02 cache01 db02]
```

`web` voyait deux éléments mais avait une capacité de quatre (le tableau continue derrière). `append` a utilisé la case libre... qui était `hotes[2]`. À l'inverse, `bases` avait `cap == len`, donc `append(bases, "pg03")` a réalloué : `bases` est parti sur un tableau neuf et `hotes` n'a plus rien vu.

La règle : **une sous-slice est un alias tant qu'aucune réallocation n'a eu lieu**. Si tu veux une copie indépendante, fais-la explicitement :

```go
copie := make([]string, len(hotes))
n := copy(copie, hotes)          // copie min(len(dst), len(src)) éléments, renvoie n
copie[0] = "modifié"
fmt.Println(n, hotes[0], copie[0]) // 4 web01 modifié
```

Ou, depuis Go 1.21, `copie := slices.Clone(hotes)`. Il existe aussi une forme à trois index, `s[debut:fin:capmax]`, qui borne la capacité pour forcer la réallocation au prochain `append` : `sur := hotes[2:3:3]` a `len 1, cap 1`, et `append(sur, x)` ne touchera jamais `hotes`. C'est l'outil de qui rend une sous-slice à un appelant sans lui donner accès au reste du tableau.

**Venant de Python :** `liste[2:]` copie en Python. En Go, jamais. Une fonction qui reçoit un slice et le modifie modifie le tableau de l'appelant, comme une liste passée à une fonction Python, et une fonction qui fait `append` peut ou non affecter l'appelant selon la capacité. C'est LE point à garder en tête ; le labo de ce chapitre te le fait toucher du doigt.

**Piège :** trier ou modifier une sous-slice « pour voir » modifie l'original. Garder une petite sous-slice d'un énorme slice (les dix premiers octets d'un fichier de 2 Go lu en mémoire) garde tout le tableau vivant pour le ramasse-miettes. Dans les deux cas, `slices.Clone`.

### 4.5 `make` et le paquet `slices`

`make` crée un slice avec une longueur et, optionnellement, une capacité :

```go
zeros := make([]int, 3)          // [0 0 0], len 3, cap 3
tampon := make([]byte, 0, 4096)  // vide, mais 4096 places réservées
```

La seconde forme est celle qu'on utilise quand on sait combien on va `append` : aucune réallocation pendant la boucle. Attention à la première : `make([]int, 3)` puis `append` ajoute *après* trois zéros, ce qui est rarement ce qu'on veut. Pour un slice qu'on remplit avec `append`, c'est `make([]T, 0, n)`.

Depuis Go 1.21, le paquet `slices` de la bibliothèque standard fournit ce qu'on réécrivait à la main :

```go
ports := []int{443, 22, 8080, 80}
slices.Sort(ports)                     // tri en place : [22 80 443 8080]
slices.Contains(ports, 22)             // true
slices.Index(ports, 443)               // 2, ou -1
slices.Max(ports)                      // 8080
slices.Equal(a, b)                     // même longueur, mêmes éléments
c := slices.Clone(a)                   // copie indépendante
slices.Compact([]int{1, 1, 2, 2, 3})   // [1 2 3] : supprime les doublons CONSÉCUTIFS
slices.Reverse(ports)
```

`slices.Sort` marche pour tout type ordonnable (nombres, chaînes) grâce aux génériques, sans rien écrire. Pour trier sur un critère (par nom, par taille), `slices.SortFunc(s, func(a, b T) int { ... })`, vu au chapitre 6 avec les structs. Avant Go 1.21, c'était `sort.Ints`, `sort.Strings` ; tu les croiseras dans du vieux code.

### 4.6 Maps : le dictionnaire

Une `map[K]V` associe des clés de type `K` à des valeurs de type `V`. C'est une table de hachage, comme le `dict` Python. La clé peut être n'importe quel type comparable avec `==` : chaîne, entier, booléen, struct sans slice dedans. Pas un slice, pas une map.

```go
roles := map[string]string{
	"web01": "web",
	"db01":  "postgres",
}
roles["cache01"] = "redis"          // ajout ou remplacement
fmt.Println(roles, len(roles))
fmt.Println(roles["db01"])          // postgres
fmt.Println(roles["inconnu"] == "") // true : la valeur zéro, PAS une erreur

r, ok := roles["inconnu"]           // la forme à deux valeurs
fmt.Printf("%q %v\n", r, ok)        // "" false
if r, ok := roles["web01"]; ok {    // l'idiome
	fmt.Println("web01 est", r)
}

delete(roles, "cache01")            // sans effet si absent
```

```
$ go run .
map[cache01:redis db01:postgres web01:web] 3
postgres
true
"" false
web01 est web
```

**Lire une clé absente ne plante pas** et renvoie la valeur zéro du type de la valeur. C'est parfois ce qu'on veut (un compteur : `compteur[mot]++` marche sans initialisation), et parfois un bug silencieux (une chaîne vide qui passe pour une valeur). D'où la forme à deux résultats, `v, ok := m[k]`, qui dit si la clé existait. Utilise-la chaque fois que « absent » et « présent avec la valeur zéro » sont deux cas différents.

```go
compteur := make(map[string]int)
for _, h := range []string{"web", "db", "web", "web"} {
	compteur[h]++       // absent = 0, donc 0+1
}
fmt.Println(compteur)  // map[db:1 web:3]
```

**L'ordre de parcours est aléatoire**, et volontairement : l'exécutif change le point de départ à chaque `range` pour que personne ne dépende de l'ordre.

```go
ports := map[string]int{"ssh": 22, "http": 80, "https": 443, "pg": 5432, "redis": 6379, "dns": 53}
for i := 0; i < 3; i++ {
	for k := range ports {
		fmt.Print(k, " ")
	}
	fmt.Println()
}
```

```
$ go run .
dns ssh http https pg redis 
dns ssh http https pg redis 
ssh http https pg redis dns 
```

Pour un affichage stable, on trie les clés. Depuis Go 1.23, `maps.Keys` renvoie un itérateur et `slices.Sorted` le consomme :

```go
cles := slices.Sorted(maps.Keys(compteur)) // [db web]
for _, k := range cles {
	fmt.Println(k, compteur[k])
}
```

**Piège :** la valeur zéro d'une map est `nil`. On peut la lire (tout est absent) mais **écrire dedans plante** : `panic: assignment to entry in nil map`. `var m map[string]int` suivi de `m["x"] = 1` est un crash classique. Crée toujours avec `make(map[string]int)` ou un littéral `map[string]int{}`.

**Venant de Python :** `d[k]` lève `KeyError` en Python ; en Go, c'est la valeur zéro. Le `d.get(k, defaut)` est donc le comportement par défaut, et `if k in d` devient `if _, ok := m[k]; ok`. Les dicts Python gardent l'ordre d'insertion depuis 3.7 ; les maps Go, jamais.

### 4.7 `range` sur tout

`range` est la même boucle pour toutes les collections, avec une ou deux variables selon le type :

| Sur | Première variable | Seconde variable |
|---|---|---|
| slice, tableau | index | copie de l'élément |
| map | clé | valeur |
| string | index de l'octet | la `rune` décodée |
| entier `n` (Go 1.22) | 0 à n-1 | aucune |
| channel (chapitre 11) | l'élément reçu | aucune |

```go
for i, v := range []int{10, 20} { ... }   // 0 10, 1 20
for i := range []int{10, 20} { ... }      // 0 1 : index seul
for _, v := range []int{10, 20} { ... }   // 10 20 : valeur seule
for k, v := range m { ... }
for i, r := range "ok é" { ... }          // 0 'o', 1 'k', 2 ' ', 3 'é'
for i := range 3 { ... }                  // 0 1 2
```

**Piège :** la seconde variable est une **copie**. `for _, v := range nums { v *= 2 }` ne modifie rien dans `nums`. Pour modifier en place, passe par l'index : `for i := range nums { nums[i] *= 2 }`. Ce sera encore plus important au chapitre 6 avec les structs.

### 4.8 Pas de set, et `strings.Builder`

Go n'a pas de type ensemble. L'idiome est une map dont la valeur ne sert à rien, `map[string]struct{}`, parce que `struct{}` (la structure vide) occupe zéro octet :

```go
vus := map[string]struct{}{}
for _, h := range []string{"web01", "db01", "web01"} {
	if _, deja := vus[h]; deja {
		fmt.Println("doublon :", h)
	}
	vus[h] = struct{}{}
}
fmt.Println(len(vus)) // 2
```

Certains écrivent `map[string]bool` avec `if vus[h]`, plus court à lire et à peine plus gros ; les deux se voient dans du vrai code. Pour l'union et l'intersection, tu les écris à la main en trois lignes, ou tu prends un paquet tiers si tu en as vraiment beaucoup.

Dernier outil du chapitre : construire une chaîne par morceaux. Comme les chaînes sont immuables, `s += morceau` dans une boucle réalloue et recopie à chaque tour. `strings.Builder` accumule dans un tampon et ne fabrique la chaîne qu'à la fin :

```go
var sb strings.Builder            // la valeur zéro est prête à l'emploi
for i := 1; i <= 3; i++ {
	fmt.Fprintf(&sb, "ligne %d\n", i)
}
sb.WriteString("fin")
fmt.Println(sb.String())
fmt.Println(sb.Len())             // 27 (des octets)
```

C'est le `"".join(liste)` de Python et le tampon `char *` à `realloc` du C, sans rien gérer. Dès que tu construis une chaîne dans une boucle, c'est ça.

### 4.9 Pour le labo

Le [labo 04](../labs/04-inventaire/README.md) est un inventaire de serveurs : une map de rôle vers liste d'hôtes, à laquelle on ajoute, dont on retire, qu'on liste trié, qu'on compte, dont on détecte les doublons. Et un exercice qui te fait provoquer le partage mémoire des sous-slices, puis le corriger avec `copy`.

### À retenir

- Un tableau `[N]T` est une valeur de taille fixe, copiée à l'affectation ; on ne l'utilise presque jamais directement.
- Un slice `[]T` est (pointeur, len, cap) sur un tableau : 24 octets, passé par copie de cette structure, données partagées.
- `s = append(s, x)`, toujours réaffecté : quand `len == cap`, `append` réalloue en doublant et le slice change d'adresse.
- Une sous-slice `s[a:b]` ne copie rien : modifier l'une modifie l'autre, et `append` peut écraser l'original s'il reste de la capacité. `copy` ou `slices.Clone` pour une copie.
- `make([]T, 0, n)` pour réserver, `make([]T, n)` pour n zéros ; paquet `slices` : `Sort`, `Contains`, `Index`, `Clone`, `Compact`.
- `map[K]V` : clé absente = valeur zéro, `v, ok := m[k]` pour savoir, `delete`, ordre de parcours aléatoire, `slices.Sorted(maps.Keys(m))` pour trier. Une map `nil` plante à l'écriture : `make` d'abord.
- `range` donne (index, copie) sur un slice, (clé, valeur) sur une map, (index d'octet, rune) sur une chaîne, 0..n-1 sur un entier. Modifier la copie ne modifie rien.
- Pas de set : `map[string]struct{}`. Pas de `+=` de chaînes en boucle : `strings.Builder`.

---

← [3. Contrôle : if, for, switch, defer](03-controle.md) · [Sommaire](../README.md) · [5. Fonctions, erreurs et closures](05-fonctions-erreurs.md) →
