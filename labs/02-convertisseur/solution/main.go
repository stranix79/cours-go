// Labo 02, solution : un convertisseur d'unités (températures, tailles, durées).
// Lancé par : go run ./solution   (depuis labs/02-convertisseur)
// Testé par : go test ./solution/
//
// Toute la logique est dans des fonctions exportées (majuscule) pour que les
// tests puissent les appeler. main() ne fait qu'afficher un tableau de démo.
package main

import (
	"fmt"  // Printf, Sprintf : formatage et affichage
	"time" // time.Duration et time.ParseDuration
)

// Fahrenheit convertit des degrés Celsius en degrés Fahrenheit.
// Les deux valeurs sont des float64 : pas de conversion à faire, et les
// constantes 9, 5 et 32 sont non typées, donc compatibles avec float64.
func Fahrenheit(celsius float64) float64 {
	return celsius*9/5 + 32
}

// Celsius fait la conversion inverse.
func Celsius(fahrenheit float64) float64 {
	return (fahrenheit - 32) * 5 / 9
}

// unites est le tableau des suffixes, du plus petit au plus grand. Un tableau
// (taille fixe dans le type) plutôt qu'un slice : on ne le modifiera jamais.
var unites = [...]string{"o", "Ko", "Mo", "Go", "To"}

// TailleLisible convertit un nombre d'octets en texte lisible, base 1024,
// comme `ls -h` ou `df -h` : 1536 donne "1.5 Ko". En dessous de 1024, on
// affiche l'entier sans décimale ("512 o"). Au-delà du To, on reste en To.
func TailleLisible(octets int64) string {
	// Cas simple d'abord : un nombre d'octets s'affiche tel quel, en entier.
	// %d marche pour un int64 comme pour un int (chapitre 2).
	if octets < 1024 {
		return fmt.Sprintf("%d o", octets)
	}

	// On travaille en flottant à partir d'ici : la conversion est explicite,
	// Go ne fait jamais int64 -> float64 tout seul.
	valeur := float64(octets)
	indice := 0 // position dans unites : 0 = "o"

	// Tant qu'on peut diviser par 1024 ET qu'il reste une unité plus grande,
	// on monte d'un cran. La seconde condition évite de dépasser le tableau
	// (indice hors bornes = panic).
	for valeur >= 1024 && indice < len(unites)-1 {
		valeur /= 1024
		indice++
	}

	// %.1f : une décimale. Sprintf renvoie la chaîne au lieu de l'afficher.
	return fmt.Sprintf("%.1f %s", valeur, unites[indice])
}

// ParseDuree lit une durée écrite à la façon de Go ("1h30m", "45s", "250ms")
// et renvoie un time.Duration, c'est-à-dire un int64 de nanosecondes.
// C'est la convention (résultat, erreur) du chapitre 5 : err vaut nil quand
// tout va bien. On enveloppe l'erreur de la bibliothèque avec du contexte.
func ParseDuree(texte string) (time.Duration, error) {
	d, err := time.ParseDuration(texte)
	if err != nil {
		// %q met le texte entre guillemets, %w garde l'erreur d'origine.
		return 0, fmt.Errorf("durée %q invalide : %w", texte, err)
	}
	return d, nil
}

// Secondes renvoie le nombre entier de secondes d'une durée. Duration est un
// int64 en nanosecondes ; la division par time.Second (une constante de type
// Duration) donne un Duration, qu'on convertit explicitement en int64.
func Secondes(d time.Duration) int64 {
	return int64(d / time.Second)
}

func main() {
	// Un tableau aligné avec des largeurs fixes dans Printf :
	// %6.1f  = flottant sur 6 colonnes, 1 décimale, aligné à droite
	// %10d   = entier sur 10 colonnes
	// %-6s   = chaîne sur 6 colonnes, alignée à gauche
	fmt.Println("Températures")
	for _, c := range []float64{100, 37, -40} {
		fmt.Printf("  %6.1f °C = %6.1f °F\n", c, Fahrenheit(c))
	}
	fmt.Printf("  %6.1f °F = %6.1f °C\n", 98.6, Celsius(98.6))

	fmt.Println("Tailles")
	// int64 explicite : 10 * 1024 * 1024 * 1024 * 1024 tient dans un int
	// (64 bits) mais TailleLisible attend un int64, et Go ne convertit pas.
	for _, o := range []int64{512, 1536, 10 * 1024 * 1024, 3 * 1024 * 1024 * 1024 * 1024} {
		fmt.Printf("  %16d o = %s\n", o, TailleLisible(o))
	}

	fmt.Println("Durées")
	for _, texte := range []string{"1h30m", "45s", "2h", "abc"} {
		d, err := ParseDuree(texte)
		if err != nil {
			// On traite l'erreur ici (affichage) et on passe au suivant :
			// on ne la remonte pas en plus, c'est l'un ou l'autre.
			fmt.Printf("  %-6s : %v\n", texte, err)
			continue
		}
		fmt.Printf("  %-6s = %6d s\n", texte, Secondes(d))
	}
}
