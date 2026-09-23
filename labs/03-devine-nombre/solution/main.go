// Labo 03, solution : le jeu du nombre à deviner.
// Lancé par : go run ./solution [graine]   (depuis labs/03-devine-nombre)
// Testé par : go test ./solution/
//
// Trois fonctions testables (NouveauSecret, Evaluer, Jouer) et un main qui
// lit les propositions au clavier. Le générateur aléatoire reçoit une graine
// explicite : avec la même graine, la même partie, donc des tests possibles.
package main

import (
	"bufio"     // Scanner : lecture ligne par ligne de l'entrée standard
	"fmt"       // affichage
	"math/rand" // générateur pseudo-aléatoire, avec graine
	"os"        // os.Args (arguments), os.Stdin (entrée standard)
	"strconv"   // Atoi : texte vers entier
	"time"      // time.Now() pour une graine différente à chaque partie
)

// Maximum est la borne haute du secret : le nombre est entre 1 et Maximum.
const Maximum = 100

// NouveauSecret tire un nombre entre 1 et max inclus. rand.New crée un
// générateur indépendant du générateur global, initialisé avec la graine :
// deux appels avec la même graine donnent le même nombre. C'est ce qui rend
// la fonction testable, et c'est le pattern à retenir pour tout ce qui est
// aléatoire dans du code qu'on veut tester.
func NouveauSecret(graine int64, max int) int {
	generateur := rand.New(rand.NewSource(graine))
	// Intn(max) renvoie un entier dans [0, max[ ; +1 décale dans [1, max].
	return generateur.Intn(max) + 1
}

// Evaluer compare une proposition au secret et renvoie l'indication à
// afficher. Un switch sans expression : chaque case est une condition,
// la première vraie gagne, pas de fallthrough (chapitre 3).
func Evaluer(secret, proposition int) string {
	switch {
	case proposition < secret:
		return "plus grand"
	case proposition > secret:
		return "plus petit"
	default:
		return "gagné"
	}
}

// Jouer rejoue une partie complète à partir d'une liste de propositions et
// renvoie le nombre d'essais qu'il a fallu pour trouver, ou 0 si la liste
// est épuisée sans succès. C'est la version « pure » de la boucle de main :
// pas de clavier, pas d'affichage, donc testable.
// Le retour nommé `essais` est initialisé à 0 et renvoyé par le `return` nu.
func Jouer(secret int, entrees []int) (essais int) {
	for _, proposition := range entrees {
		essais++
		if Evaluer(secret, proposition) == "gagné" {
			return
		}
	}
	return 0
}

func main() {
	// La graine vient du premier argument si on en donne un (partie
	// reproductible, utile pour la sortie attendue du README), sinon de
	// l'horloge (partie différente à chaque lancement).
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
	// Ce defer s'exécute à la sortie de main, quelle que soit la façon dont
	// on en sort (fin de l'entrée, ou victoire). C'est une closure : elle lit
	// la valeur de `essais` AU MOMENT où elle s'exécute, pas au moment du
	// defer. Avec `defer fmt.Println("Essais :", essais)`, on afficherait 0.
	defer func() {
		fmt.Println("Essais :", essais)
	}()

	// bufio.Scanner découpe l'entrée standard ligne par ligne. Scan() renvoie
	// false quand il n'y a plus rien à lire (Ctrl-D, ou fin du tube).
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Proposition ? ")
	for scanner.Scan() {
		proposition, err := strconv.Atoi(scanner.Text())
		if err != nil {
			// Une ligne qui n'est pas un nombre ne compte pas comme un essai.
			fmt.Print("Ce n'est pas un nombre. Proposition ? ")
			continue
		}
		essais++
		verdict := Evaluer(secret, proposition)
		fmt.Println(verdict)
		if verdict == "gagné" {
			// return sort de main : le defer s'exécute juste après.
			return
		}
		fmt.Print("Proposition ? ")
	}
	// On arrive ici si l'entrée est terminée sans victoire.
	fmt.Println("\nAbandon. Le nombre était", secret)
}
