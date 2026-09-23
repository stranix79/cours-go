// Labo 03 : le jeu du nombre à deviner.
// Lancé par : go run . [graine]    Testé par : go test .
//
// Complète les TODO dans l'ordre : NouveauSecret, Evaluer, Jouer, puis la
// boucle de main. Tout compile dès maintenant ; go test . te guide.
package main

import (
	"fmt"       // affichage
	"math/rand" // générateur pseudo-aléatoire, avec graine
	"os"        // os.Args (arguments), os.Stdin (entrée standard)
	"strconv"   // Atoi, ParseInt : texte vers entier
	"time"      // time.Now() pour une graine différente à chaque partie
	// TODO : ajoute "bufio" quand tu écris la boucle de lecture (TODO 4).
)

// Maximum est la borne haute du secret : le nombre est entre 1 et Maximum.
const Maximum = 100

// NouveauSecret tire un nombre entre 1 et max inclus, à partir d'une graine.
// Même graine, même nombre : c'est ce qui rend le jeu testable.
func NouveauSecret(graine int64, max int) int {
	// TODO 1 : generateur := rand.New(rand.NewSource(graine)), puis
	// generateur.Intn(max) donne un entier dans [0, max[ ; ajoute 1.
	_ = rand.New
	return 0
}

// Evaluer compare une proposition au secret et renvoie "plus grand" si le
// secret est plus grand que la proposition, "plus petit" s'il est plus
// petit, "gagné" si c'est le bon nombre.
func Evaluer(secret, proposition int) string {
	// TODO 2 : un switch sans expression (chapitre 3.3), trois cas.
	return ""
}

// Jouer rejoue une partie à partir d'une liste de propositions et renvoie
// le nombre d'essais qu'il a fallu pour trouver, ou 0 si la liste est
// épuisée sans succès. Une proposition après la victoire ne compte pas.
func Jouer(secret int, entrees []int) (essais int) {
	// TODO 3 : range sur entrees, essais++ à chaque tour, return dès que
	// Evaluer répond "gagné". Après la boucle, return 0.
	return 0
}

func main() {
	// La graine vient du premier argument s'il y en a un, sinon de l'horloge.
	graine := time.Now().UnixNano()
	if len(os.Args) > 1 {
		g, err := strconv.ParseInt(os.Args[1], 10, 64)
		if err != nil {
			fmt.Println("graine invalide :", os.Args[1])
			os.Exit(1)
		}
		graine = g
	}
	secret := NouveauSecret(graine, Maximum)
	fmt.Printf("Je pense à un nombre entre 1 et %d (graine %d).\n", Maximum, graine)

	essais := 0
	// TODO 4a : un defer qui affiche "Essais :" et le nombre d'essais à la
	// sortie de main. Attention : `defer fmt.Println("Essais :", essais)`
	// figerait essais à 0 (les arguments sont évalués tout de suite).
	// Il faut une fonction anonyme : defer func() { ... }()

	// TODO 4b : scanner := bufio.NewScanner(os.Stdin) puis
	// for scanner.Scan() { ... } : à chaque ligne, strconv.Atoi(scanner.Text()) ;
	// si ce n'est pas un nombre, redemande sans compter d'essai ;
	// sinon essais++, affiche Evaluer(secret, proposition), et return si "gagné".
	// Affiche "Proposition ? " (fmt.Print, sans retour à la ligne) avant
	// chaque lecture.

	// On arrive ici si l'entrée est terminée sans victoire.
	fmt.Println("\nAbandon. Le nombre était", secret)
	_ = essais
}
