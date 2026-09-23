// Labo 01, solution : un programme qui dit sur quelle machine il tourne.
// Lancé par : go run ./solution   (depuis labs/01-hello)
//
// Ce fichier est le point d'entrée d'un exécutable : package main + func main.
// Il n'y a aucune logique métier, on ne fait que lire des informations que
// l'exécutif Go et le système nous donnent, puis les afficher.
package main

import (
	"fmt"     // formatage et affichage : Println, Printf...
	"os"      // dialogue avec le système d'exploitation : Hostname, fichiers, variables d'environnement
	"runtime" // informations sur l'exécutif Go : système, architecture, version, cœurs
)

func main() {
	fmt.Println("Bonjour depuis Go.")

	// os.Hostname renvoie DEUX valeurs : le nom et une erreur. C'est la
	// convention Go pour tout ce qui peut échouer (chapitre 5). Ici on ignore
	// l'erreur avec l'identifiant blanc `_` ; dans un vrai programme on la
	// testerait. Le `:=` déclare ET initialise la variable, le type (string)
	// est déduit par le compilateur.
	nom, _ := os.Hostname()
	fmt.Println("Machine   :", nom)

	// runtime.GOOS et runtime.GOARCH sont des constantes fixées à la
	// compilation : si tu compiles avec GOOS=linux, le binaire dira "linux"
	// même si tu le lances depuis un Mac (il ne se lancera pas, d'ailleurs).
	// Println sépare ses arguments par un espace, d'où le "/" collé.
	fmt.Println("Système   :", runtime.GOOS+"/"+runtime.GOARCH)

	// La version du compilateur qui a produit CE binaire, pas celle installée
	// sur la machine qui le lance : le binaire embarque son exécutif.
	fmt.Println("Go        :", runtime.Version())

	// Le nombre de processeurs logiques que l'exécutif peut utiliser : c'est
	// sur autant de cœurs que les goroutines (chapitre 11) seront réparties.
	fmt.Println("Cœurs     :", runtime.NumCPU())
}
