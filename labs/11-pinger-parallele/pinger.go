// Labo 11 : sonder des ports TCP en parallèle, avec délai et annulation.
// Ce fichier contient toute la logique testable ; main.go ne fait qu'afficher.
package main

import (
	"context"
	"errors"
	"net"
	"strconv"
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

// String rend "hote:port", avec les crochets IPv6 si besoin ("[::1]:22").
func (c Cible) String() string {
	return net.JoinHostPort(c.Hote, strconv.Itoa(c.Port))
}

// Resultat est l'issue d'une sonde : la cible, succès ou non, le temps mis,
// et l'erreur (nil si OK).
type Resultat struct {
	Cible Cible
	OK    bool
	Duree time.Duration
	Err   error
}

// ParseCible lit "hote:port" (ou "[ipv6]:port") et valide le port (1 à 65535).
func ParseCible(s string) (Cible, error) {
	// TODO 1 : net.SplitHostPort(s) rend (hote, port string, err). Convertis
	// le port avec strconv.Atoi ; hors de 1..65535 → errors.New("port hors
	// plage"). Renvoie Cible{Hote: hote, Port: port}.
	return Cible{}, errors.New("TODO")
}

// Sonder tente une connexion TCP vers hote:port, bornée par timeout ET par
// ctx (le premier des deux qui expire gagne). La connexion est refermée
// aussitôt : on veut savoir si le port répond, pas parler au service.
func Sonder(ctx context.Context, hote string, port int, timeout time.Duration) Resultat {
	// TODO 2 : note l'heure de départ ; construis un net.Dialer{Timeout:
	// timeout} et appelle d.DialContext(ctx, "tcp", adresse) où adresse est
	// Cible{hote, port}.String(). Remplis Resultat (Duree = time.Since(debut),
	// Err = l'erreur, OK = err == nil) et ferme conn si elle est ouverte.
	return Resultat{}
}

// SonderTous sonde toutes les cibles avec au plus `parallelisme` sondes en
// même temps (worker pool) et rend les résultats DANS L'ORDRE des cibles.
// parallelisme <= 0 vaut 1. Une annulation de ctx fait échouer les sondes
// restantes avec ctx.Err(), sans bloquer.
func SonderTous(ctx context.Context, cibles []Cible, parallelisme int) []Resultat {
	// TODO 3 : un channel de tâches (les index i), un channel de résultats
	// (index + Resultat), `parallelisme` workers lancés avec wg.Go qui lisent
	// les tâches et sondent (Timeout de la cible, ou TimeoutDefaut si 0), une
	// goroutine qui alimente les tâches puis ferme le channel, une goroutine
	// qui attend wg puis ferme les résultats, et la boucle qui range chaque
	// résultat à sa place dans la slice.
	return nil
}

// FormatTableau met en forme les résultats, une ligne par cible, précédée
// d'un en-tête. Format : "%-22s %-4s %8s  %s\n" avec la cible, OK ou KO,
// la durée arrondie à la milliseconde, et l'erreur (vide si OK).
func FormatTableau(res []Resultat) string {
	// TODO 4 : un strings.Builder, l'en-tête
	// fmt.Sprintf("%-22s %-4s %8s  %s\n", "CIBLE", "ETAT", "DUREE", "DETAIL"),
	// puis une ligne par résultat.
	return ""
}
