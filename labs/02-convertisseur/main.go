// Labo 02 : un convertisseur d'unités (températures, tailles, durées).
// Lancé par : go run .    Testé par : go test .
//
// Complète les fonctions marquées TODO dans l'ordre. Chacune renvoie pour
// l'instant une valeur zéro (ou une erreur "TODO") pour que le programme
// compile ; `go test .` te dira lesquelles restent à faire.
package main

import (
	"errors"
	"fmt"
	"time"
)

// Fahrenheit convertit des degrés Celsius en degrés Fahrenheit : c × 9/5 + 32.
func Fahrenheit(celsius float64) float64 {
	// TODO 1 : la formule. Les constantes 9, 5, 32 se combinent avec un
	// float64 sans conversion.
	return 0
}

// Celsius fait la conversion inverse : (f - 32) × 5/9.
func Celsius(fahrenheit float64) float64 {
	// TODO 2
	return 0
}

// unites est le tableau des suffixes, du plus petit au plus grand.
var unites = [...]string{"o", "Ko", "Mo", "Go", "To"}

// TailleLisible convertit un nombre d'octets en texte lisible, base 1024 :
// 1536 donne "1.5 Ko", 10485760 donne "10.0 Mo". En dessous de 1024, on
// affiche l'entier sans décimale ("512 o"). Au-delà du To, on reste en To
// ("2048.0 To").
func TailleLisible(octets int64) string {
	// TODO 3 : si octets < 1024, fmt.Sprintf("%d o", octets).
	// Sinon : convertis en float64 (explicitement !), puis divise par 1024
	// tant que la valeur est >= 1024 et qu'il reste une unité plus grande.
	// Termine par fmt.Sprintf("%.1f %s", valeur, unites[indice]).
	_ = unites
	return ""
}

// ParseDuree lit une durée écrite à la façon de Go ("1h30m", "45s") avec
// time.ParseDuration et renvoie (durée, nil), ou (0, erreur) si le texte
// est invalide. Enveloppe l'erreur : fmt.Errorf("durée %q invalide : %w", texte, err).
func ParseDuree(texte string) (time.Duration, error) {
	// TODO 4
	return 0, errors.New("TODO")
}

// Secondes renvoie le nombre entier de secondes d'une durée.
// Indice : d / time.Second est un Duration ; il faut le convertir en int64.
func Secondes(d time.Duration) int64 {
	// TODO 5
	return 0
}

func main() {
	fmt.Println("Températures")
	for _, c := range []float64{100, 37, -40} {
		fmt.Printf("  %6.1f °C = %6.1f °F\n", c, Fahrenheit(c))
	}
	fmt.Printf("  %6.1f °F = %6.1f °C\n", 98.6, Celsius(98.6))

	fmt.Println("Tailles")
	for _, o := range []int64{512, 1536, 10 * 1024 * 1024, 3 * 1024 * 1024 * 1024 * 1024} {
		fmt.Printf("  %16d o = %s\n", o, TailleLisible(o))
	}

	fmt.Println("Durées")
	for _, texte := range []string{"1h30m", "45s", "2h", "abc"} {
		d, err := ParseDuree(texte)
		if err != nil {
			fmt.Printf("  %-6s : %v\n", texte, err)
			continue
		}
		fmt.Printf("  %-6s = %6d s\n", texte, Secondes(d))
	}
}
