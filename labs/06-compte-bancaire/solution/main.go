// Labo 06, solution : le scénario de démonstration.
// Lancé par : go run ./solution   (depuis labs/06-compte-bancaire)
//
// main ne contient aucune logique : il enchaîne des appels et affiche. Tout ce
// qui se teste est dans compte.go et banque.go.
package main

import "fmt"

func main() {
	banque := NewBanque()

	// Les deux valeurs de retour sont ignorées avec _ uniquement parce que ce
	// scénario ne peut pas échouer (titulaires distincts). Dans un vrai
	// programme, on teste err.
	alice, _ := banque.Ouvrir("alice")
	bob, _ := banque.Ouvrir("bob")

	// Les montants sont en centimes : 150000 = 1500,00 €.
	if err := alice.Deposer(150000, "salaire"); err != nil {
		fmt.Println("erreur :", err)
	}
	if err := alice.Retirer(2050, "courses"); err != nil {
		fmt.Println("erreur :", err)
	}
	if err := banque.Virement("alice", "bob", 50000); err != nil {
		fmt.Println("erreur :", err)
	}

	// Ce virement échoue : bob n'a que 500,00 €. L'erreur remonte avec son
	// contexte, du plus général au plus précis.
	if err := banque.Virement("bob", "alice", 100000); err != nil {
		fmt.Println("erreur :", err)
	}
	// Et celui-ci échoue avant même de toucher aux soldes.
	if err := banque.Virement("carol", "alice", 100); err != nil {
		fmt.Println("erreur :", err)
	}

	// Println appelle String() : alice et bob sont des fmt.Stringer.
	fmt.Println(alice)
	fmt.Println(bob)
	fmt.Println("opérations :", alice.Nombre(), "pour alice,", bob.Nombre(), "pour bob")
}
