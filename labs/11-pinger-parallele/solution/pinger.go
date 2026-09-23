// Labo 11, solution : sonder des ports TCP en parallèle, avec délai et
// annulation. Toute la logique est ici, testée par pinger_test.go ;
// main.go (le même que dans le squelette) ne fait que lire les arguments
// et afficher.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TimeoutDefaut est le délai de connexion utilisé quand une Cible n'en a pas.
const TimeoutDefaut = 2 * time.Second

// Cible est un hôte et un port à sonder, avec un délai optionnel (0 = défaut).
type Cible struct {
	Hote    string
	Port    int
	Timeout time.Duration
}

// String rend "hote:port". net.JoinHostPort ajoute les crochets autour
// d'une adresse IPv6 ("[::1]:22"), ce qu'une concaténation naïve oublierait.
func (c Cible) String() string {
	return net.JoinHostPort(c.Hote, strconv.Itoa(c.Port))
}

// Resultat est l'issue d'une sonde.
type Resultat struct {
	Cible Cible
	OK    bool
	Duree time.Duration
	Err   error
}

// ParseCible lit "hote:port" et valide le port.
func ParseCible(s string) (Cible, error) {
	// SplitHostPort comprend "hote:port" et "[ipv6]:port", et rend une
	// erreur claire s'il manque le port : on la laisse remonter telle quelle.
	hote, portTexte, err := net.SplitHostPort(s)
	if err != nil {
		return Cible{}, err
	}
	port, err := strconv.Atoi(portTexte)
	if err != nil {
		return Cible{}, fmt.Errorf("port %q : %w", portTexte, err) // %w garde l'erreur d'origine
	}
	if port < 1 || port > 65535 {
		return Cible{}, errors.New("port hors plage (1 à 65535)")
	}
	return Cible{Hote: hote, Port: port}, nil
}

// Sonder tente une connexion TCP, bornée par timeout ET par ctx.
func Sonder(ctx context.Context, hote string, port int, timeout time.Duration) Resultat {
	cible := Cible{Hote: hote, Port: port}
	debut := time.Now()
	// Un Dialer avec Timeout borne la connexion elle-même ; DialContext la
	// borne AUSSI par le contexte. Le premier des deux qui expire gagne, et
	// un contexte déjà annulé rend la main immédiatement avec ctx.Err().
	// (net.DialTimeout ferait le délai, mais ignorerait le contexte.)
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", cible.String())
	r := Resultat{Cible: cible, Duree: time.Since(debut), Err: err}
	if err == nil {
		r.OK = true
		conn.Close() // le port répond, c'est tout ce qu'on voulait savoir
	}
	return r
}

// SonderTous : un worker pool. Les tâches sont les index des cibles, les
// résultats reviennent avec leur index, et la slice finale est remplie à la
// bonne place : l'ordre de sortie est celui de l'entrée quel que soit l'ordre
// d'arrivée.
func SonderTous(ctx context.Context, cibles []Cible, parallelisme int) []Resultat {
	if parallelisme < 1 {
		parallelisme = 1 // zéro worker ne finirait jamais
	}
	if parallelisme > len(cibles) {
		parallelisme = max(len(cibles), 1) // inutile de lancer plus de workers que de cibles
	}

	// indexe transporte un résultat avec sa position d'origine.
	type indexe struct {
		i int
		r Resultat
	}
	taches := make(chan int)       // les index à sonder
	resultats := make(chan indexe) // les résultats, dans l'ordre d'arrivée

	// Les workers : chacun lit des index jusqu'à la fermeture de `taches`.
	var wg sync.WaitGroup
	for range parallelisme {
		wg.Go(func() {
			for i := range taches {
				c := cibles[i]
				timeout := c.Timeout
				if timeout == 0 {
					timeout = TimeoutDefaut
				}
				r := Sonder(ctx, c.Hote, c.Port, timeout)
				r.Cible = c // on garde le Timeout de la cible d'origine dans le résultat
				resultats <- indexe{i, r}
			}
		})
	}

	// Le producteur : alimente les tâches puis ferme, ce qui fait sortir les
	// workers de leur range. Une seule goroutine ferme ce channel.
	go func() {
		for i := range cibles {
			taches <- i
		}
		close(taches)
	}()

	// Quand TOUS les workers ont fini, plus aucun résultat ne viendra : on
	// ferme `resultats` pour que la boucle de collecte ci-dessous se termine.
	go func() {
		wg.Wait()
		close(resultats)
	}()

	// La collecte : chaque résultat va à sa place. C'est la seule goroutine
	// qui écrit dans la slice, donc pas de course.
	res := make([]Resultat, len(cibles))
	for x := range resultats {
		res[x.i] = x.r
	}
	return res
}

// FormatTableau met en forme les résultats, une ligne par cible.
func FormatTableau(res []Resultat) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-22s %-4s %8s  %s\n", "CIBLE", "ETAT", "DUREE", "DETAIL")
	for _, r := range res {
		etat, detail := "OK", ""
		if !r.OK {
			etat = "KO"
			if r.Err != nil {
				detail = r.Err.Error()
			}
		}
		fmt.Fprintf(&b, "%-22s %-4s %8s  %s\n", r.Cible, etat, r.Duree.Round(time.Millisecond), detail)
	}
	return b.String()
}
