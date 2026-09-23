// Labo 06 : un compte bancaire avec historique.
// Rôle du fichier : le type Compte (données + méthodes) et le type Journal
// qu'il embarque. Appelé par main.go (le scénario) et banque.go (le virement).
// Les montants sont en centimes (int64) : jamais de float pour de l'argent.
package main

import (
	"errors"
	"fmt"
)

// Les erreurs sentinelles : à renvoyer (emballées avec %w) par Deposer et Retirer.
var (
	ErrMontantInvalide  = errors.New("montant invalide")
	ErrSoldeInsuffisant = errors.New("solde insuffisant")
)

// Operation est une ligne d'historique.
type Operation struct {
	Type    string // "dépôt", "retrait", "virement reçu", "virement émis"
	Montant int64  // toujours positif, en centimes
	Libelle string
}

// Journal est l'historique des opérations. Il est embarqué dans Compte.
type Journal struct {
	Operations []Operation
}

// Enregistrer ajoute une opération à la fin de la slice.
// TODO 1 : choisis le receveur (valeur ou pointeur ?) et fais l'append.
func (j *Journal) Enregistrer(op Operation) {
}

// Nombre renvoie le nombre d'opérations enregistrées.
// TODO 2 : renvoie la longueur de la slice.
func (j *Journal) Nombre() int {
	return 0
}

// Compte est un compte bancaire. Le solde est privé (minuscule) : seuls
// Deposer et Retirer peuvent le modifier.
type Compte struct {
	Titulaire string
	solde     int64
	Journal   // embedding : c.Enregistrer et c.Operations sont promus
}

// NewCompte est le constructeur. Il renvoie un pointeur : un compte a une identité.
func NewCompte(titulaire string) *Compte {
	return &Compte{Titulaire: titulaire}
}

// Deposer crédite le compte de centimes et enregistre une opération "dépôt".
// Renvoie ErrMontantInvalide (emballée) si centimes <= 0.
// TODO 3 : valide, modifie le solde, enregistre. Attention au receveur.
func (c *Compte) Deposer(centimes int64, libelle string) error {
	return errors.New("TODO")
}

// Retirer débite le compte et enregistre une opération "retrait".
// Renvoie ErrMontantInvalide si centimes <= 0, ErrSoldeInsuffisant si le
// solde ne suffit pas. Dans les deux cas, rien ne change.
// TODO 4 : même structure que Deposer, avec la vérification du solde en plus.
func (c *Compte) Retirer(centimes int64, libelle string) error {
	return errors.New("TODO")
}

// Solde renvoie le solde en centimes.
// TODO 5 : une ligne.
func (c *Compte) Solde() int64 {
	return 0
}

// String rend Compte un fmt.Stringer : fmt.Println(c) affiche le relevé.
// Format exact attendu par les tests (largeurs : %-14s pour le type, %9s
// pour le montant formaté, précédé du signe + ou -) :
//
//	Relevé de alice
//	  dépôt          +1500,00 €  salaire
//	  retrait        -  20,50 €  courses
//	Solde : 1479,50 €
//
// Pas de retour à la ligne après le solde. Le signe est - pour "retrait" et
// "virement émis", + sinon.
// TODO 6 : utilise strings.Builder et fmt.Fprintf(&b, ...).
func (c *Compte) String() string {
	return fmt.Sprintf("Relevé de %s (TODO)", c.Titulaire)
}

// FormatEuros écrit 12345 centimes comme « 123,45 € » (virgule, deux
// décimales, espace, symbole).
// TODO 7 : division et modulo par 100, et le verbe %02d.
func FormatEuros(centimes int64) string {
	return "TODO"
}
