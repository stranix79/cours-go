// Labo 06 : le scénario de démonstration.
// Lancé par : go run .   (ou go run ./solution pour la version corrigée)
//
// main ne contient aucune logique : il enchaîne des appels et affiche. Tout ce
// qui se teste est dans compte.go et banque.go. Rien à modifier ici.
package main

import "fmt"

func main() {
	banque := NewBanque()

	alice, _ := banque.Ouvrir("alice")
	bob, _ := banque.Ouvrir("bob")
	if alice == nil || bob == nil {
		fmt.Println("Ouvrir n'est pas encore écrit (TODO 8)")
		return
	}

	if err := alice.Deposer(150000, "salaire"); err != nil {
		fmt.Println("erreur :", err)
	}
	if err := alice.Retirer(2050, "courses"); err != nil {
		fmt.Println("erreur :", err)
	}
	if err := banque.Virement("alice", "bob", 50000); err != nil {
		fmt.Println("erreur :", err)
	}
	if err := banque.Virement("bob", "alice", 100000); err != nil {
		fmt.Println("erreur :", err)
	}
	if err := banque.Virement("carol", "alice", 100); err != nil {
		fmt.Println("erreur :", err)
	}

	fmt.Println(alice)
	fmt.Println(bob)
	fmt.Println("opérations :", alice.Nombre(), "pour alice,", bob.Nombre(), "pour bob")
}
