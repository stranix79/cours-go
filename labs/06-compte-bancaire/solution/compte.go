// Labo 06, solution : un compte bancaire avec historique.
//
// Rôle du fichier : le type Compte (données + méthodes) et le type Journal
// qu'il embarque. Appelé par main.go (le scénario) et banque.go (le virement).
// Les montants sont en centimes (int64) : jamais de float pour de l'argent,
// 0,1 + 0,2 ne fait pas 0,3 en binaire.
package main

import (
	"errors"
	"fmt"
	"strings"
)

// Les erreurs sentinelles du paquet : une seule valeur chacune, créée au
// démarrage, que l'appelant compare avec errors.Is (chapitre 8).
var (
	ErrMontantInvalide  = errors.New("montant invalide")
	ErrSoldeInsuffisant = errors.New("solde insuffisant")
)

// Operation est une ligne d'historique. C'est un type « valeur » : petit,
// jamais modifié après création, comparable avec ==.
type Operation struct {
	Type    string // "dépôt", "retrait", "virement reçu", "virement émis"
	Montant int64  // toujours positif, en centimes
	Libelle string
}

// Journal est l'historique des opérations. Il est embarqué dans Compte :
// ses champs et méthodes sont promus, donc c.Enregistrer(...) et c.Operations
// marchent directement sur un Compte.
type Journal struct {
	Operations []Operation
}

// Enregistrer ajoute une opération. Receveur pointeur obligatoire : append
// modifie la slice (longueur, et parfois adresse du tableau). Avec un
// receveur valeur, on ajouterait à une copie du Journal, perdue au retour.
func (j *Journal) Enregistrer(op Operation) {
	j.Operations = append(j.Operations, op)
}

// Nombre renvoie le nombre d'opérations. Lecture seule, mais on garde un
// receveur pointeur pour rester cohérent avec Enregistrer (règle du chapitre 6).
func (j *Journal) Nombre() int {
	return len(j.Operations)
}

// Compte est un compte bancaire. Le solde est en minuscule : privé au paquet,
// donc modifiable uniquement par Deposer et Retirer, qui valident.
type Compte struct {
	Titulaire string
	solde     int64
	Journal   // embedding : un Compte « a » un Journal, et ses méthodes sont promues
}

// NewCompte est le constructeur par convention. Il renvoie un pointeur parce
// que toutes les méthodes de Compte ont un receveur pointeur, et parce qu'un
// compte a une identité : deux variables doivent désigner LE même compte.
func NewCompte(titulaire string) *Compte {
	return &Compte{Titulaire: titulaire}
}

// Deposer crédite le compte. Receveur pointeur : la méthode modifie le solde.
// Un montant nul ou négatif est refusé avec ErrMontantInvalide.
func (c *Compte) Deposer(centimes int64, libelle string) error {
	if centimes <= 0 {
		return fmt.Errorf("déposer %d: %w", centimes, ErrMontantInvalide)
	}
	c.solde += centimes
	c.Enregistrer(Operation{Type: "dépôt", Montant: centimes, Libelle: libelle})
	return nil
}

// Retirer débite le compte. Deux échecs possibles, chacun avec sa sentinelle,
// emballée dans un message qui donne le contexte (chapitre 8).
func (c *Compte) Retirer(centimes int64, libelle string) error {
	if centimes <= 0 {
		return fmt.Errorf("retirer %d: %w", centimes, ErrMontantInvalide)
	}
	if centimes > c.solde {
		return fmt.Errorf("retirer %s (solde %s): %w",
			FormatEuros(centimes), FormatEuros(c.solde), ErrSoldeInsuffisant)
	}
	c.solde -= centimes
	c.Enregistrer(Operation{Type: "retrait", Montant: centimes, Libelle: libelle})
	return nil
}

// Solde est l'accesseur en lecture. Pas de « GetSolde » : en Go, le getter
// porte le nom du champ, sans préfixe.
func (c *Compte) Solde() int64 {
	return c.solde
}

// String rend Compte un fmt.Stringer (chapitre 7) : fmt.Println(c) affiche
// le relevé. On construit le texte avec strings.Builder plutôt que par
// concaténations répétées : une seule allocation à la fin.
func (c *Compte) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Relevé de %s\n", c.Titulaire)
	for _, op := range c.Operations {
		signe := "+"
		if op.Type == "retrait" || op.Type == "virement émis" {
			signe = "-"
		}
		fmt.Fprintf(&b, "  %-14s %s%9s  %s\n", op.Type, signe, FormatEuros(op.Montant), op.Libelle)
	}
	fmt.Fprintf(&b, "Solde : %s", FormatEuros(c.solde))
	return b.String()
}

// FormatEuros écrit 12345 centimes comme « 123,45 € ». Fonction et pas
// méthode : elle ne dépend d'aucun compte, et la Banque s'en sert aussi.
func FormatEuros(centimes int64) string {
	return fmt.Sprintf("%d,%02d €", centimes/100, centimes%100)
}
