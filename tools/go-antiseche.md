# Antisèche Go 1.27

*La syntaxe sur deux pages. Les renvois « chap. N » pointent vers le [cours](../README.md). Tout ce qui est ici a été compilé avec Go 1.27.*

## Squelette d'un programme (chap. 1)

```go
package main                      // main = exécutable ; autre nom = bibliothèque

import (
	"fmt"                         // un import inutilisé = erreur de compilation
	"os"
)

func main() {                     // pas d'arguments, pas de retour ; os.Exit(1) pour le code
	fmt.Println("Bonjour", os.Args[1:])
}
```

`go mod init nom` avant tout ; `go run .` ; `go build -o bin .` ; `gofmt -w .` ; `{` sur la ligne du `func`, virgule finale obligatoire dans les listes multi-lignes. Majuscule = exporté (public), minuscule = privé au paquet.

## Variables, constantes, types de base (chap. 2)

```go
var port int = 5432               // déclaration complète
var nom string                    // valeur zéro : "" (0, false, nil selon le type)
port := 5432                      // déclaration courte, type déduit ; seulement dans une fonction
a, b := 1, "deux"                 // multiple ; x, y = y, x échange deux variables de même type
const Timeout = 5 * time.Second   // constante typée par l'usage ; iota pour énumérer
const (
	Bas = iota                    // 0
	Moyen                         // 1
	Haut                          // 2
)
```

| Type | Exemples | Notes |
|---|---|---|
| `int`, `int8`…`int64`, `uint`…`uint64`, `uintptr` | `42`, `0xFF`, `1_000_000` | `int` = 64 bits sur toute machine 64 bits ; débordement silencieux comme en C |
| `float32`, `float64` | `3.14`, `1e-9` | `float64` par défaut ; `math.Inf(1)`, `math.NaN()` |
| `bool` | `true`, `false` | pas de conversion implicite : `if n {` refuse un `int` |
| `string` | `"utf-8 ✓"`, `` `brut\n` `` | immuable, octets UTF-8 ; `len` = octets, `for range` = runes |
| `byte` (= `uint8`), `rune` (= `int32`) | `'a'`, `'é'` | un caractère est un `rune`, une chaîne indexée donne un `byte` |
| `[N]T`, `[]T`, `map[K]V`, `*T`, `struct{}`, `func(...)`, `chan T`, `interface{}` (= `any`) | | chap. 4, 6, 7, 11 |

Conversions **toujours explicites** : `float64(n)`, `int(f)` (tronque), `string(rune)`, `[]byte(s)`, `string(b)`. Chaîne vers nombre : `strconv.Atoi("42")`, `strconv.ParseFloat(s, 64)`, `strconv.ParseBool` ; nombre vers chaîne : `strconv.Itoa(42)`, `fmt.Sprintf("%d", n)`. Jamais `string(42)` (ça donne `"*"`, `go vet` le signale).

## Chaînes (chap. 2)

```go
s := "web-01.stranix.net"
len(s); s[0]; s[4:6]; s + ".lan"                  // octets ; b := s[0] est un byte ; tranches par octet
for i, r := range s { _ = r }                     // itère sur les runes (i = index d'octet)
strings.Contains(s, "web"); strings.HasPrefix(s, "web"); strings.Split(s, "."); strings.Join(parts, ".")
strings.TrimSpace(s); strings.ToUpper(s); strings.ReplaceAll(s, "-", "_"); strings.Fields("a  b")
strings.Cut(s, ".")                                // avant, après, trouvé := strings.Cut(...)
fmt.Sprintf("%s:%d %v %+v %T %q %x %5.2f %-10s %08b", h, p, v, v, v, s, n, f, s, n)
var sb strings.Builder; sb.WriteString("a"); sb.String()   // concaténer en boucle
utf8.RuneCountInString(s)                          // nombre de caractères
```

`%v` valeur, `%+v` avec noms de champs, `%#v` syntaxe Go, `%T` type, `%q` entre guillemets, `%w` pour envelopper une erreur (`fmt.Errorf` seulement).

## Contrôle (chap. 3)

```go
if err := f(); err != nil { return err }          // instruction courte puis condition ; pas de parenthèses
if x > 0 && y < 10 || !ok { }                    // && || ! ; pas de ternaire
for i := 0; i < n; i++ { }                        // la seule boucle
for cond { }                                      // while
for { break }                                     // infinie
for i := range 10 { }                             // 0..9 (Go 1.22)
for i, v := range slice { }; for k, v := range m { }; for _, v := range s { }
switch etat {                                     // pas de break nécessaire, pas de fallthrough par défaut
case "up", "ok":  // plusieurs valeurs
case "down":
	fallthrough                                   // explicite, rare
default:
}
switch { case n < 0: ... case n == 0: ... }       // switch sans expression = chaîne de if
defer f.Close()                                   // exécuté à la sortie de la fonction, LIFO ; arguments évalués tout de suite
// goto, break étiquette, continue étiquette existent ; break sort du switch OU de la boucle la plus proche
```

## Slices, maps, tableaux (chap. 4)

```go
var s []int                                       // nil, len 0, utilisable avec append
s = make([]int, 0, 10)                            // longueur 0, capacité 10
s = []string{"a", "b"}; s = append(s, "c", "d"); s = append(s, autre...)
s[1:3]; s[:2]; s[2:]; len(s); cap(s)              // une tranche PARTAGE le tableau sous-jacent
copy(dst, src); s2 := slices.Clone(s)             // copier pour de vrai
slices.Sort(s); slices.SortFunc(s, func(a, b T) int { return cmp.Compare(a.X, b.X) })
slices.Contains(s, x); slices.Index(s, x); slices.Reverse(s); slices.Max(s); slices.Equal(a, b)
s = slices.Delete(s, i, j); s = slices.Insert(s, i, v)
tab := [3]int{1, 2, 3}                            // tableau : taille fixe, copié par valeur ; rare, sauf [16]byte

m := map[string]int{"web-01": 22}                 // map nil : lecture OK, écriture = panic ; make(map[K]V) sinon
m["pg-01"] = 5432; v := m["x"]                    // clé absente : valeur zéro
v, ok := m["x"]                                   // ok = présent ; delete(m, "x") ; len(m)
for k, v := range m { }                           // ordre ALÉATOIRE ; trier : maps.Keys + slices.Sorted
slices.Sorted(maps.Keys(m))                       // []string trié (Go 1.23 itérateurs)
```

**Piège** : `append` peut réallouer ; toujours réaffecter (`s = append(s, x)`). Une slice passée à une fonction : les éléments sont partagés, pas la longueur.

## Fonctions et erreurs (chap. 5, 8)

```go
func lire(chemin string, max int) ([]byte, error) { }          // retours multiples ; error en dernier
func min(xs ...int) int { }                                     // variadique ; min(s...) pour passer une slice
func compte() (n int, err error) { defer func() { n++ }(); return 1, nil }   // retours nommés (rares, pour defer)
carre := func(x int) int { return x * x }                       // fonction anonyme (closure sur les variables voisines)
func generateur() func() int { n := 0; return func() int { n++; return n } }

if err != nil { return fmt.Errorf("lire %s : %w", chemin, err) }   // envelopper avec %w, contexte devant
var ErrIntrouvable = errors.New("introuvable")                      // erreur sentinelle
errors.Is(err, ErrIntrouvable); errors.Is(err, os.ErrNotExist)     // compare à travers les %w
var pe *fs.PathError; if errors.As(err, &pe) { pe.Path }           // extrait un type
type ErreurConfig struct{ Cle string }                             // erreur personnalisée
func (e *ErreurConfig) Error() string { return "clé manquante : " + e.Cle }
errors.Join(err1, err2)                                            // plusieurs erreurs en une
panic("impossible"); defer func() { if r := recover(); r != nil { } }()   // pour les bugs, pas pour les erreurs
```

Pas d'exceptions : une fonction qui peut échouer renvoie `error`, l'appelant teste **tout de suite**. Ignorer une erreur s'écrit `_ = f()` et se voit.

## Structs, méthodes, pointeurs (chap. 6)

```go
type Hote struct {
	Nom  string `json:"nom"`                     // tag : lu par encoding/json
	IP   net.IP
	Tags []string
	mu   sync.Mutex                              // non exporté ; jamais copier une struct qui a un mutex
}
h := Hote{Nom: "web-01"}; p := &Hote{Nom: "pg-01"}; var z Hote      // littéral, pointeur, valeur zéro
h.Nom; p.Nom                                                        // même syntaxe, pas de ->
func (h Hote) Affiche() string { }                                  // receveur valeur : copie, ne modifie pas
func (h *Hote) Renomme(n string) { h.Nom = n }                      // receveur pointeur : modifie ; règle : tout en pointeur si un l'est
func NouvelHote(nom string) *Hote { return &Hote{Nom: nom} }        // « constructeur » = fonction Nouveau...
type Serveur struct { Hote; Port int }                              // embarquer : Serveur.Nom promu, méthodes aussi
type Port int; func (p Port) Ouvert() bool { }                      // méthode sur n'importe quel type nommé
new(T) == &T{}; p == nil; *p                                        // pas d'arithmétique de pointeurs
```

Struct comparable avec `==` si tous ses champs le sont ; utilisable comme clé de map. Copie = affectation (valeur), pas de constructeur ni de destructeur.

## Interfaces (chap. 7)

```go
type Sondeur interface { Sonder(ctx context.Context) error }   // satisfaite IMPLICITEMENT par tout type qui a la méthode
type ReadCloser interface { io.Reader; io.Closer }               // composition d'interfaces
var s Sondeur = &SondeHTTP{}                                     // l'affectation vérifie à la compilation
var _ Sondeur = (*SondeHTTP)(nil)                                // assertion de compilation, dans le fichier du type
if h, ok := s.(*SondeHTTP); ok { }                               // assertion de type ; sans ok = panic si faux
switch v := x.(type) { case int: case string: case error: default: }   // type switch
any                                                              // = interface{} : n'importe quoi ; à éviter sauf JSON/fmt
```

Les interfaces qui comptent : `error` (`Error() string`), `fmt.Stringer` (`String() string`), `io.Reader`, `io.Writer`, `io.Closer`, `sort.Interface`, `http.Handler`. Petites (1 à 3 méthodes), définies **côté consommateur**, nommées en `-er`. Une interface nil et une interface qui contient un pointeur nil ne sont **pas** égales (piège du `return err` avec un `*MonErreur` nil).

## Génériques (chap. 16)

```go
func Map[T, U any](xs []T, f func(T) U) []U { out := make([]U, 0, len(xs)); for _, x := range xs { out = append(out, f(x)) }; return out }
func Max[T cmp.Ordered](a, b T) T { if a > b { return a }; return b }
type Pile[T any] struct{ items []T }                            // type générique
type Nombre interface{ ~int | ~float64 }                        // contrainte ; ~ = « tout type dont le sous-jacent est »
```

Avant d'en écrire : `slices`, `maps`, `cmp` de la stdlib le font déjà. Une interface suffit souvent.

## Goroutines, channels, select, context (chap. 11)

```go
go f(x)                                          // lance f dans une goroutine ; main n'attend PAS
var wg sync.WaitGroup                            // attendre : Add(n) avant go, Done() dans la goroutine, Wait()
wg.Add(1); go func() { defer wg.Done(); travail() }(); wg.Wait()
wg.Go(func() { travail() })                     // Go 1.25 : Add + Done pour toi

ch := make(chan int)                             // non bufferisé : l'envoi bloque jusqu'à réception (rendez-vous)
ch := make(chan int, 10)                         // bufferisé : bloque quand plein
ch <- v; v := <-ch; v, ok := <-ch                // ok = false si fermé et vide
close(ch); for v := range ch { }                 // range s'arrête à la fermeture ; seul l'émetteur ferme
// chan<- T (envoi seul), <-chan T (réception seule) dans les signatures

select {                                         // attend le premier prêt ; aléatoire si plusieurs
case v := <-ch:
case out <- v:
case <-time.After(time.Second):                  // délai
case <-ctx.Done(): return ctx.Err()              // annulation
default:                                         // non bloquant
}

ctx, annule := context.WithTimeout(context.Background(), 5*time.Second); defer annule()
ctx, annule := context.WithCancel(ctx); ctx = context.WithValue(ctx, cle, v)   // valeur : rare (trace id)
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
func travail(ctx context.Context, ...) error     // ctx TOUJOURS premier paramètre, jamais dans une struct

var mu sync.Mutex; mu.Lock(); defer mu.Unlock()  // protéger une map, un compteur ; sync.RWMutex si lecture >> écriture
var n atomic.Int64; n.Add(1); n.Load()           // compteur sans mutex
var once sync.Once; once.Do(init)                // une seule fois
// errgroup.Group (golang.org/x/sync) : WaitGroup + première erreur + contexte annulé
```

Patrons : worker pool (`for range n { go worker(jobs, results) }`), pipeline (`chan` entre étapes), fan-out/fan-in. Règle : « ne communiquez pas en partageant la mémoire, partagez la mémoire en communiquant ». Détecteur : `go test -race`, `go run -race`.

## Fichiers, JSON, CLI, HTTP, SQL (chap. 12 à 14)

```go
data, err := os.ReadFile("sondes.json"); os.WriteFile(p, data, 0o644)          // petits fichiers
f, err := os.Open(p); defer f.Close(); sc := bufio.NewScanner(f); for sc.Scan() { sc.Text() }   // ligne par ligne
os.Stat(p); os.MkdirAll(d, 0o755); os.Remove(p); filepath.Join(a, b); filepath.WalkDir(root, fn)
os.Getenv("DSN"); os.LookupEnv("PORT"); os.Args; os.Exit(1); os.Hostname()
flag.String("addr", ":8080", "adresse"); flag.Int; flag.Bool; flag.Parse(); flag.Args()
exec.CommandContext(ctx, "pg_dump", "-h", h).Output()                          // sous-processus
json.Marshal(v); json.MarshalIndent(v, "", "  "); json.Unmarshal(data, &v)     // tags `json:"nom,omitempty"`
json.NewDecoder(r).Decode(&v); json.NewEncoder(w).Encode(v)                    // sur un flux
time.Now(); time.Since(t); t.Format(time.RFC3339); time.Parse(layout, s); time.ParseDuration("1h30m"); 5 * time.Second
slog.Info("sonde", "cible", nom, "latence", d); slog.New(slog.NewJSONHandler(os.Stdout, nil))

//go:embed static/*                                                            // fichiers dans le binaire
var static embed.FS

client := &http.Client{Timeout: 5 * time.Second}; resp, err := client.Get(url); defer resp.Body.Close()
req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, body); req.Header.Set("Content-Type", "application/json")
mux := http.NewServeMux(); mux.HandleFunc("GET /taches/{id}", func(w http.ResponseWriter, r *http.Request) { r.PathValue("id") })
srv := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}; srv.ListenAndServe(); srv.Shutdown(ctx)
httptest.NewRecorder(); httptest.NewServer(h)

db, err := sql.Open("sqlite", "inventaire.db"); db.PingContext(ctx)           // Open ne connecte pas
db.QueryRowContext(ctx, "SELECT ip FROM hotes WHERE nom = $1", nom).Scan(&ip)   // sql.ErrNoRows si absent
rows, err := db.QueryContext(ctx, q, args...); defer rows.Close(); for rows.Next() { rows.Scan(&a, &b) }; rows.Err()
db.ExecContext(ctx, "INSERT ...", a, b); tx, err := db.BeginTx(ctx, nil); defer tx.Rollback(); tx.Commit()
```

## Paquets, modules, tests (chap. 9)

```go
package sonde                     // un dossier = un paquet ; internal/ = visible seulement par le module parent
import "cours-go/labs/16-projet/internal/sonde"   // chemin = module + dossier ; alias : import s "..."
func init() { }                   // exécuté à l'import ; à éviter
```

```go
func TestSonderTCP(t *testing.T) {                          // fichier xxx_test.go, même paquet
	cas := []struct{ nom string; port int; up bool }{ {"ouvert", 22, true}, {"fermé", 1, false} }
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {                   // sous-tests nommés
			if got := Sonder(c.port); got != c.up { t.Errorf("Sonder(%d) = %v, attendu %v", c.port, got, c.up) }
		})
	}
}
// t.Fatal (arrête), t.Error (continue), t.Helper(), t.TempDir(), t.Setenv(), t.Parallel(), t.Skip()
func BenchmarkX(b *testing.B) { for b.Loop() { X() } }     // go test -bench .
func ExampleX() { fmt.Println(X()) } // suivi du commentaire « // Output: 42 » : doc + test
```

## Bibliothèque standard à connaître

| Paquet | Pour |
|---|---|
| `fmt`, `strings`, `strconv`, `unicode/utf8`, `bytes`, `regexp` | texte |
| `os`, `io`, `bufio`, `path/filepath`, `embed`, `io/fs` | fichiers, flux |
| `encoding/json`, `encoding/csv`, `encoding/base64`, `encoding/hex` | formats |
| `flag`, `os/exec`, `os/signal`, `log`, `log/slog` | CLI, processus, journaux |
| `time`, `context`, `sync`, `sync/atomic`, `errors` | temps, annulation, concurrence, erreurs |
| `net`, `net/http`, `net/http/httptest`, `net/url`, `net/netip`, `crypto/tls` | réseau |
| `database/sql`, `slices`, `maps`, `cmp`, `sort`, `math`, `math/rand/v2` | données |
| `crypto/sha256`, `crypto/rand`, `crypto/hmac`, `hash/fnv` | empreintes, aléa sûr |
| `testing`, `runtime`, `runtime/pprof`, `net/http/pprof`, `expvar`, `unsafe` | tests, sous le capot |

Hors stdlib mais standard de fait : `golang.org/x/sync/errgroup`, `golang.org/x/sys`, `github.com/spf13/cobra` (CLI), `github.com/go-chi/chi/v5` (routeur), `github.com/jackc/pgx/v5` (PostgreSQL), `modernc.org/sqlite`, `github.com/prometheus/client_golang`, `github.com/coder/websocket`.

## Commandes

```bash
go mod init exemple.com/sondes ; go mod tidy ; go get pkg@v1.2.3 ; go get -u ./... ; go mod verify ; go list -m all
go run . ; go build -o bin/sondes ./cmd/sondes ; go install ./... ; go install github.com/x/y@latest
go build -ldflags "-s -w -X main.version=1.2.0" -trimpath .                 # chap. 15
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build .                            # compilation croisée ; go tool dist list
go test ./... ; go test -v -run TestX ./pkg ; go test -race ./... ; go test -count=1 . ; go test -cover ./... ; go test -bench . -benchmem
go vet ./... ; gofmt -l . ; gofmt -w . ; golangci-lint run ; staticcheck ./...
go doc net/http.Server ; go doc -all strings | less ; go env GOROOT GOPATH ; go version -m bin/sondes
go tool pprof http://localhost:6060/debug/pprof/profile ; go tool trace ; GODEBUG=gctrace=1
go clean -cache -modcache ; go work init ; go generate ./...
```
