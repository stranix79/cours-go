// Labo 01 : un programme qui dit sur quelle machine il tourne.
// Lancé par : go run .   (ou go build -o hello . puis ./hello)
package main

import (
	"fmt"
	// TODO : ajoute "os" et "runtime" ici quand tu en as besoin (un import
	// inutilisé est une erreur de compilation, donc ajoute-les au fur et à mesure).
)

func main() {
	fmt.Println("Bonjour depuis Go.")

	// TODO 1 : le nom de la machine. os.Hostname() renvoie (string, error) ;
	// ignore l'erreur avec le blanc _ pour ce labo :  nom, _ := os.Hostname()

	// TODO 2 : le système et l'architecture : runtime.GOOS et runtime.GOARCH.

	// TODO 3 : la version de Go (runtime.Version()) et le nombre de cœurs
	// (runtime.NumCPU()).

	// Affiche chaque ligne avec fmt.Println, le libellé aligné sur 10 colonnes
	// comme dans la sortie attendue du README.
}
