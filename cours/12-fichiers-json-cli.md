# 12. Fichiers, JSON, ligne de commande DevOps

*Go de zéro à la prod : chapitre 12 sur 16.* ← [11. Concurrence : goroutines, channels, select, context](11-concurrence.md) · [Sommaire](../README.md) · [13. HTTP : client, serveur, API JSON](13-http.md) →

Ce que tu écriras le plus souvent en Go, ce n'est pas un serveur, c'est un outil en ligne de commande : le script bash devenu illisible, le script Python qu'il faut installer sur cinq machines, le petit binaire qui lit un fichier de log, appelle `kubectl`, sort du JSON pour `jq` et rend un code de sortie que `set -e` comprend. Ce chapitre est la boîte à outils de ce quotidien, entièrement dans la bibliothèque standard : fichiers, JSON, arguments, processus, signaux, journalisation. Il se termine par un outil complet de quatre-vingts lignes qui assemble tout.

Le fil rouge est le contrat Unix, le même qu'en C et qu'en Python : le résultat sur stdout, tout le reste sur stderr, un code de sortie honnête, et stdin quand on ne donne pas de fichier.

### 12.1 Fichiers : d'un coup ou ligne par ligne

Deux façons de lire, selon la taille. `os.ReadFile` charge tout en mémoire dans un `[]byte` : parfait pour une configuration, mauvais pour un log de 4 Go. `os.Open` puis `bufio.Scanner` lit ligne par ligne avec un tampon de 64 Ko, quelle que soit la taille du fichier :

```go
os.WriteFile("app.conf", []byte("port=8080\nlog=/var/log/app.log\n"), 0o644) // rend une erreur, à tester
contenu, err := os.ReadFile("app.conf")
if err != nil {
	fmt.Fprintln(os.Stderr, "lecture :", err)
	os.Exit(1)
}
fmt.Printf("%d octets, type %T\n", len(contenu), contenu)

f, err := os.Open("app.conf")
if err != nil {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
defer f.Close()
sc := bufio.NewScanner(f)
n := 0
for sc.Scan() {
	n++
	fmt.Printf("ligne %d : %q\n", n, sc.Text())
}
if err := sc.Err(); err != nil {
	fmt.Fprintln(os.Stderr, "scan :", err)
}
```

```
31 octets, type []uint8
ligne 1 : "port=8080"
ligne 2 : "log=/var/log/app.log"
```

Le `0o644` de `WriteFile` est le mode Unix habituel (`rw-r--r--`), en octal préfixé `0o` comme en Python ; il ne s'applique qu'à la création. `defer f.Close()` juste après l'ouverture réussie, c'est le réflexe : le fichier sera fermé quelle que soit la sortie de la fonction. Et `sc.Err()` après la boucle : `Scan()` rend `false` à la fin du fichier *et* en cas d'erreur, seule cette vérification les distingue.

**Piège :** le `Scanner` refuse une ligne de plus de 64 Ko (`bufio.Scanner: token too long`). Pour des logs JSON d'une ligne par événement, ça arrive. `sc.Buffer(make([]byte, 1024*1024), 1024*1024)` avant la boucle relève la limite à 1 Mo.

**Venant de Python :** `open(...).read()` est `os.ReadFile`, `for ligne in f:` est le `Scanner`, `with` est `defer f.Close()`. Il manque le `.strip()` : `sc.Text()` rend la ligne sans son `\n`.

### 12.2 `os.Stat`, `filepath`, permissions, fichiers temporaires

```go
info, err := os.Stat("app.conf")
fmt.Println(info.Name(), info.Size(), info.Mode(), info.IsDir())

_, err = os.Stat("/etc/nexiste.conf")
fmt.Println(err, "|", errors.Is(err, fs.ErrNotExist))

fmt.Println(filepath.Join("/var", "log", "..", "lib", "postgresql"))
logs, _ := filepath.Glob(filepath.Join(dir, "*.log"))
fmt.Println(len(logs), "fichiers *.log au premier niveau")
filepath.WalkDir(dir, func(chemin string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	rel, _ := filepath.Rel(dir, chemin)
	fmt.Printf("  %-10s dossier=%v\n", rel, d.IsDir())
	return nil
})
```

```
app.conf 31 -rw-r--r-- false
stat /etc/nexiste.conf: no such file or directory | true
/var/lib/postgresql
2 fichiers *.log au premier niveau
  .          dossier=true
  a.log      dossier=false
  b.log      dossier=false
  sub        dossier=true
  sub/c.log  dossier=false
```

`os.Stat` rend un `fs.FileInfo` : nom, taille, mode (les droits, imprimés comme `ls -l`), date de modification, dossier ou non. Un fichier absent se teste avec `errors.Is(err, fs.ErrNotExist)`, pas en comparant le message. `filepath.Join` nettoie les `..` et met le bon séparateur (`\` sous Windows). `Glob` est le motif du shell, non récursif ; `WalkDir` parcourt récursivement et appelle ta fonction pour chaque entrée, avec `fs.SkipDir` en retour pour sauter un dossier. `os.MkdirAll` est `mkdir -p`, `os.Remove`/`os.RemoveAll` sont `rm` et `rm -rf`, `os.Chmod(chemin, 0o600)` change les droits. `os.CreateTemp` et `os.MkdirTemp` créent un fichier ou un dossier au nom unique dans `$TMPDIR` ; à toi de les supprimer (`defer os.RemoveAll(dir)`).

### 12.3 JSON

`encoding/json` convertit entre structs Go et JSON. Le mécanisme repose sur les **tags** : une chaîne accolée au champ qui dit son nom JSON et ses options. Les champs non exportés (minuscule) ne sont jamais sérialisés :

```go
type Sonde struct {
	Hote       string   `json:"hote"`
	Port       int      `json:"port"`
	Timeout    string   `json:"timeout,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	motDePasse string   // non exporté : jamais sérialisé
}

type Config struct {
	Version int     `json:"version"`
	Sondes  []Sonde `json:"sondes"`
	Alerte  *struct {
		Email string `json:"email"`
	} `json:"alerte,omitempty"`
}

cfg := Config{Version: 1, Sondes: []Sonde{
	{Hote: "db1", Port: 5432, Tags: []string{"prod", "pg"}},
	{Hote: "web1", Port: 443, Timeout: "2s"},
}}
b, _ := json.Marshal(cfg)
fmt.Println(string(b))
b, _ = json.MarshalIndent(cfg, "", "  ")
```

```
{"version":1,"sondes":[{"hote":"db1","port":5432,"tags":["prod","pg"]},{"hote":"web1","port":443,"timeout":"2s"}]}
{
  "version": 1,
  "sondes": [
    {
      "hote": "db1",
      "port": 5432,
      "tags": [
    ...
```

`omitempty` omet le champ s'il vaut la valeur zéro (`""`, `0`, `nil`, slice vide) : `timeout` n'apparaît pas pour db1, `tags` pas pour web1, `alerte` (un pointeur `nil`) pas du tout. `MarshalIndent` produit la version lisible. Dans l'autre sens, `Unmarshal` prend un pointeur vers la struct à remplir ; les clés inconnues sont ignorées, les clés absentes laissent la valeur zéro :

```go
texte := `{"version":2,"sondes":[{"hote":"cache","port":6379,"inconnu":true}],"alerte":{"email":"ops@code79.com"}}`
var lu Config
if err := json.Unmarshal([]byte(texte), &lu); err != nil {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
fmt.Printf("%+v alerte=%s\n", lu.Sondes[0], lu.Alerte.Email)

var libre map[string]any // quand on ne connaît pas la forme
json.Unmarshal([]byte(texte), &libre)
fmt.Printf("version=%v (%T)\n", libre["version"], libre["version"])
for _, s := range libre["sondes"].([]any) {
	m := s.(map[string]any)
	fmt.Println("hote :", m["hote"], "port :", m["port"])
}

dec := json.NewDecoder(flux) // un flux : plusieurs documents à la suite
for {
	var s Sonde
	if err := dec.Decode(&s); err != nil {
		break // io.EOF en fin de flux
	}
	fmt.Print(s.Hote, ":", s.Port, " ")
}
```

```
{Hote:cache Port:6379 Timeout: Tags:[] motDePasse:} alerte=ops@code79.com
version=2 (float64)
hote : cache port : 6379
a:1 b:2 c:3
```

Trois choses à retenir de cette sortie. `map[string]any` décode n'importe quoi, mais tous les nombres deviennent des `float64` et il faut des assertions de type (`.([]any)`, `.(map[string]any)`) à chaque niveau : c'est la voie de secours, pas la voie normale ; dès que tu connais la forme, écris la struct. `json.NewDecoder` lit depuis un `io.Reader` (un fichier, une réponse HTTP, stdin) sans tout charger, et sait enchaîner plusieurs documents : c'est le format JSON Lines des logs. Et une erreur de type est précise (`json: cannot unmarshal string into Go struct field Config.version of type int`) : lis-la avant de chercher.

**Venant de Python :** `json.dumps`/`json.loads` rendent des dicts ; ici on décode dans un type, et le type fait la validation. Le `sort_keys` n'existe pas parce que les champs d'une struct sortent dans l'ordre de déclaration, et les clés d'une map sont triées automatiquement. Pour YAML, il n'y a rien dans la bibliothèque standard : `gopkg.in/yaml.v3` fonctionne exactement pareil, avec des tags `yaml:"nom"`, et c'est ce qu'utilisent Kubernetes et Docker Compose.

### 12.4 Les arguments : le paquet `flag`

`os.Args` est la slice des arguments, `os.Args[0]` le nom du programme, comme `argv` en C. Pour les options, le paquet `flag` : on déclare chaque option avec son nom, sa valeur par défaut et son aide, on appelle `Parse`, et on lit les pointeurs :

```go
hote := flag.String("hote", "localhost", "hôte à sonder")
port := flag.Int("port", 5432, "port TCP")
verbeux := flag.Bool("v", false, "détails sur stderr")
timeout := flag.Duration("timeout", 2*time.Second, "délai de connexion")
flag.Usage = func() {
	fmt.Fprintf(os.Stderr, "usage : %s [options] [fichier...]\n", os.Args[0])
	flag.PrintDefaults()
}
flag.Parse()
fmt.Println(*hote, *port, *verbeux, *timeout, "reste :", flag.Args())
```

```
$ ./sonde -hote db1 -port 5433 -v -timeout 500ms a.log b.log
db1 5433 true 500ms reste : [a.log b.log]
$ ./sonde -port abc
invalid value "abc" for flag -port: parse error
usage : ./sonde [options] [fichier...]
  -hote string
    	hôte à sonder (default "localhost")
  -port int
    	port TCP (default 5432)
  -timeout duration
    	délai de connexion (default 2s)
  -v	détails sur stderr
$ echo $?
2
```

Ce que tu obtiens gratuitement : la conversion (`-port abc` est refusé), les durées lisibles (`500ms`, `2s`, `1h30m`), l'aide générée par `-h` ou `-help`, et le code de sortie 2 sur une erreur d'usage. `flag.Args()` rend ce qui reste après les options (les fichiers), `flag.NArg()` leur nombre. Deux différences avec `getopt` et argparse : un seul tiret (`-port`, pas `--port`, même si `--port` est accepté), et **les options viennent avant les positionnels** : `./sonde a.log -v` laisse `-v` dans `flag.Args()`.

Pour des sous-commandes à la `git` (`outil ping`, `outil list`), on regarde `os.Args[1]` et on donne à chaque sous-commande son propre jeu d'options avec `flag.NewFlagSet` :

```go
switch os.Args[1] {
case "ping":
	fs := flag.NewFlagSet("ping", flag.ExitOnError)
	n := fs.Int("n", 3, "nombre d'essais")
	fs.Parse(os.Args[2:])
	fmt.Println("ping", fs.Args(), "essais :", *n)
case "list":
	// même schéma, avec ses propres options (-json...)
default:
	fmt.Fprintln(os.Stderr, "sous-commande inconnue :", os.Args[1])
	os.Exit(2)
}
```

```
$ ./outil ping -n 5 db1
ping [db1] essais : 5
$ ./outil rm
sous-commande inconnue : rm
$ echo $?
2
```

`flag.ExitOnError` fait sortir avec 2 sur une option inconnue ; `flag.ContinueOnError` rend l'erreur, ce qui permet de tester sans que le test se termine (le labo s'en sert). Au-delà de trois sous-commandes avec des options imbriquées, de la complétion shell et une aide colorée, il y a **cobra** (`github.com/spf13/cobra`), la bibliothèque derrière `kubectl`, `docker`, `gh`, `hugo`. Elle mérite sa dépendance à partir d'un certain volume ; en dessous, `flag` suffit et n'a rien à installer.

**Venant de Python :** `flag` est entre `sys.argv` et `argparse` ; cobra est typer. Le `main(argv)` testable d'argparse a son équivalent ici : une fonction `run(args []string, stdin io.Reader, stdout, stderr io.Writer) int` que `main` appelle avec les vrais flux, et que les tests appellent avec des tampons. C'est la structure du labo.

### 12.5 Lancer un processus : `os/exec`

```go
out, err := exec.Command("uname", "-sm").Output()
fmt.Printf("%q %v\n", out, err)

out, err = exec.Command("ls", "/nexiste").CombinedOutput() // stdout et stderr mélangés
fmt.Printf("%q\n", out)
var ee *exec.ExitError
if errors.As(err, &ee) {
	fmt.Println("code de sortie :", ee.ExitCode())
}

ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
defer cancel()
debut := time.Now()
err = exec.CommandContext(ctx, "sleep", "5").Run()
fmt.Println(err, "après", time.Since(debut).Round(10*time.Millisecond), "|", ctx.Err())
```

```
"Darwin arm64\n" <nil>
"ls: /nexiste: No such file or directory\n"
code de sortie : 1
signal: killed après 200ms | context deadline exceeded
```

`exec.Command(nom, args...)` construit la commande, **un argument par élément**, sans passer par un shell : pas de globbing, pas de pipe, pas d'injection possible même si un argument contient `; rm -rf /`. `Output()` rend stdout, `CombinedOutput()` stdout et stderr mélangés (pour un diagnostic), `Run()` ne rend que l'erreur. Un code de sortie non nul est une erreur de type `*exec.ExitError`, à extraire avec `errors.As` pour lire `ExitCode()` ; une commande introuvable est une autre erreur (`executable file not found in $PATH`), rendue avant même le lancement. Et `CommandContext` tue le processus (SIGKILL) quand le contexte expire : c'est le timeout de `subprocess.run`, mais qui se combine avec le Ctrl-C de la section suivante. Pour lire la sortie au fil de l'eau (`docker logs -f`), `cmd.StdoutPipe()` rend un `io.Reader` à donner à un `bufio.Scanner` ; `cmd.Stdin`, `cmd.Dir`, `cmd.Env` règlent l'entrée, le dossier et l'environnement de l'enfant.

**Venant du C :** c'est `fork` + `execvp` + `waitpid` + les `pipe` pour récupérer la sortie, en une ligne. Quand tu veux vraiment un pipeline shell, écris-le en Go (un `Scanner` sur la sortie du premier, `cmd.Stdin` du second), ou, en dernier recours, `exec.Command("sh", "-c", ligne)` avec une ligne que tu as construite toi-même et qui ne contient rien venant de l'utilisateur.

### 12.6 Signaux, environnement, codes de sortie, stderr

**Signaux.** Un `Ctrl-C` envoie SIGINT, `docker stop` et `systemctl stop` envoient SIGTERM. Sans rien faire, le programme meurt. `signal.NotifyContext` transforme ces signaux en annulation d'un contexte : toutes les goroutines qui le surveillent s'arrêtent, tu ranges, tu sors avec le code convenu (128 + numéro du signal : 130 pour SIGINT, 143 pour SIGTERM) :

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

fmt.Println("je travaille, Ctrl-C pour arrêter (pid", os.Getpid(), ")")
for i := 1; ; i++ {
	select {
	case <-ctx.Done():
		fmt.Println("signal reçu, je range mes affaires :", ctx.Err())
		os.Exit(130)
	case <-time.After(300 * time.Millisecond):
		fmt.Println("tour", i)
	}
}
```

```
$ ./sig &  sleep 0.7 ; kill -INT %1
je travaille, Ctrl-C pour arrêter (pid 61234 )
tour 1
signal reçu, je range mes affaires : context canceled
$ echo $?
130
```

**Variables d'environnement.** `os.Getenv("PORT")` rend `""` si absente ; `os.LookupEnv` rend aussi un booléen pour distinguer « absente » de « vide ». C'est la façon normale de configurer un binaire dans un conteneur, et la conversion est à ta charge : `strconv.Atoi(os.Getenv("PORT"))`, et si l'erreur n'est pas nulle, un message sur stderr et `os.Exit(2)` (`PORT=abc ./outil` doit refuser, pas deviner).

**Codes de sortie et flux.** `os.Exit(n)` termine immédiatement, **sans exécuter les `defer`** : appelle-le depuis `main` seulement, après que les fonctions qui ont des fichiers à fermer ont rendu la main. La convention que `set -e`, `&&` et les pipelines CI comprennent : 0 tout va bien, 1 erreur d'exécution (fichier illisible, hôte injoignable), 2 erreur d'usage (mauvais argument, celle de `flag`). Et les trois flux sont des variables : `os.Stdin` (un `io.Reader`), `os.Stdout` et `os.Stderr` (des `io.Writer`). `fmt.Println` écrit sur stdout ; `fmt.Fprintln(os.Stderr, ...)` sur stderr. Celui qui fait `outil -json | jq` ne veut pas ton « connexion en cours… » dans son JSON.

### 12.7 Logs structurés : `log/slog`

Le paquet `log` historique écrit des lignes de texte. `log/slog` (Go 1.21) écrit des **paires clé-valeur**, en texte ou en JSON, avec des niveaux : c'est ce que Loki, Elasticsearch ou CloudWatch indexent sans regex.

```go
slog.Info("démarrage", "port", 8080, "env", "prod") // Debug serait caché : niveau Info par défaut

logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
slog.SetDefault(logger)
slog.Debug("connexion", "hote", "db1", "port", 5432)
slog.Warn("réponse lente", "hote", "db1", "ms", 1832)
slog.Error("échec", "err", os.ErrNotExist, "tentatives", 3)
```

```
2026/09/23 12:30:00 INFO démarrage port=8080 env=prod
{"time":"2026-09-23T12:30:00.000000+02:00","level":"DEBUG","msg":"connexion","hote":"db1","port":5432}
{"time":"2026-09-23T12:30:00.000000+02:00","level":"WARN","msg":"réponse lente","hote":"db1","ms":1832}
{"time":"2026-09-23T12:30:00.000000+02:00","level":"ERROR","msg":"échec","err":"file does not exist","tentatives":3}
```

Le message est une constante courte, les variables vont dans les attributs : `slog.Info("connexion refusée", "hote", h)` et pas `slog.Info("connexion refusée par " + h)`. Un `-v` sur ton outil se traduit par `Level: slog.LevelDebug` dans les options du handler ; `TextHandler` pour un humain, `JSONHandler` pour une machine. Le handler par défaut est un texte sur stderr, niveau Info, et `logger.With("composant", "sondeur")` rend un logger qui ajoute cet attribut à chaque ligne.

### 12.8 Un outil complet : `dirstat`

Tout ce qui précède, dans un programme de quatre-vingts lignes qui donne la taille cumulée par extension d'une arborescence, avec `-n`, `-json`, `-v`, un Ctrl-C propre et des codes de sortie honnêtes :

```go
type Stat struct {
	Extension string `json:"extension"`
	Fichiers  int    `json:"fichiers"`
	Octets    int64  `json:"octets"`
}

// parcourir agrège les tailles par extension, et s'arrête si ctx est annulé.
func parcourir(ctx context.Context, racine string) ([]Stat, error) {
	parExt := map[string]*Stat{}
	err := filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err() // Ctrl-C : on sort proprement du parcours
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		ext := filepath.Ext(chemin)
		if parExt[ext] == nil {
			parExt[ext] = &Stat{Extension: ext}
		}
		parExt[ext].Fichiers++
		parExt[ext].Octets += info.Size()
		slog.Debug("fichier", "chemin", chemin, "octets", info.Size())
		return nil
	})
	if err != nil {
		return nil, err
	}
	stats := make([]Stat, 0, len(parExt))
	for _, s := range parExt {
		stats = append(stats, *s)
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].Octets > stats[j].Octets })
	return stats, nil
}

func main() {
	n := flag.Int("n", 5, "nombre d'extensions affichées")
	enJSON := flag.Bool("json", false, "sortie JSON sur stdout")
	verbeux := flag.Bool("v", false, "journal détaillé sur stderr")
	flag.Parse()
	racine := "."
	if flag.NArg() > 0 {
		racine = flag.Arg(0)
	}
	niveau := slog.LevelInfo
	if *verbeux {
		niveau = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: niveau})))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	stats, err := parcourir(ctx, racine)
	if err != nil {
		slog.Error("parcours impossible", "racine", racine, "err", err)
		os.Exit(1)
	}
	if len(stats) > *n {
		stats = stats[:*n]
	}
	if *enJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(stats)
		return
	}
	fmt.Printf("%-10s %8s %12s\n", "EXT", "FICHIERS", "OCTETS")
	for _, s := range stats {
		fmt.Printf("%-10s %8d %12d\n", s.Extension, s.Fichiers, s.Octets)
	}
}
```

```
$ ./dirstat -n 3 $(go env GOROOT)/src/encoding/json
EXT        FICHIERS       OCTETS
.go              97      1730807
.zst              7       282490
$ ./dirstat -json -n 1 $(go env GOROOT)/src/encoding/json
[
  {
    "extension": ".go",
    "fichiers": 97,
    "octets": 1730807
  }
]
$ ./dirstat /nexiste ; echo $?
time=2026-09-23T12:25:01.196+02:00 level=ERROR msg="parcours impossible" racine=/nexiste err="lstat /nexiste: no such file or directory"
1
$ ./dirstat -v $(go env GOROOT)/src/encoding/json 2>&1 >/dev/null | head -1
time=2026-09-23T12:30:00.000+02:00 level=DEBUG msg=fichier chemin=$GOROOT/src/encoding/json/bench_test.go octets=13230
```

Relis-le avec la grille du chapitre : la logique (`parcourir`) est une fonction pure qui prend un contexte et rend des valeurs, testable sans processus ; `main` ne fait que lire les options, régler les logs, brancher le signal, appeler, afficher, sortir. Le résultat (tableau ou JSON) va sur stdout, les logs sur stderr, l'erreur sort avec 1, et `flag` sortirait avec 2 sur une mauvaise option. Un Ctrl-C au milieu d'un `/` de 2 To rend `context canceled` et le code 1, au lieu de laisser un processus mourir n'importe où.

### 12.9 Pour le labo

Le [labo 12](../labs/12-cli-devops/README.md) te fait écrire `logstat`, un outil à sous-commandes (`count`, `top`, `json`) qui lit un fichier de log ou stdin, avec `flag.NewFlagSet`, `slog` en mode `-v`, les erreurs sur stderr et les bons codes de sortie. Les tests appellent `run(args, stdin, stdout, stderr)` avec des tampons : pas de processus, pas de fichier, une milliseconde.

### À retenir

- `os.ReadFile`/`os.WriteFile` pour les petits fichiers ; `os.Open` + `bufio.Scanner` pour les gros, ligne par ligne, avec `sc.Err()` après la boucle et `defer f.Close()` juste après l'ouverture.
- `os.Stat` + `errors.Is(err, fs.ErrNotExist)` ; `filepath.Join`, `Glob`, `WalkDir` ; `os.MkdirAll`, `os.CreateTemp`, modes en `0o644`.
- JSON : des structs avec des tags `json:"nom,omitempty"`, `Marshal`/`Unmarshal`, `MarshalIndent` pour l'humain, `NewDecoder`/`NewEncoder` pour les flux, `map[string]any` en dernier recours. YAML : `gopkg.in/yaml.v3`, même modèle.
- `flag` : `String`/`Int`/`Bool`/`Duration`, `Parse`, `Args`, aide gratuite, code 2 sur erreur ; `flag.NewFlagSet` par sous-commande ; cobra quand ça grossit.
- `exec.Command` sans shell, `Output`/`CombinedOutput`/`Run`, `*exec.ExitError` pour le code, `CommandContext` pour le délai.
- `signal.NotifyContext` pour un Ctrl-C propre ; `os.Getenv`/`LookupEnv` ; codes 0/1/2 ; résultat sur stdout, tout le reste sur stderr ; `os.Exit` n'exécute pas les `defer`.
- `log/slog` : message constant, attributs clé-valeur, `JSONHandler` pour les machines, `LevelDebug` derrière `-v`, stderr par défaut (comme `log`). Structure d'un outil : des fonctions pures testables, un `run(args, stdin, stdout, stderr) int`, un `main` de cinq lignes.

---

← [11. Concurrence : goroutines, channels, select, context](11-concurrence.md) · [Sommaire](../README.md) · [13. HTTP : client, serveur, API JSON](13-http.md) →
