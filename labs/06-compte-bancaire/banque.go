// Labo 06 : la banque, qui tient les comptes et fait les virements.
// Rôle du fichier : le type Banque et sa méthode Virement. Appelé par main.go.
package main

import (
	"errors"
)

var (
	ErrCompteInconnu  = errors.New("compte inconnu")
	ErrCompteExistant = errors.New("compte déjà ouvert")
)

// Banque range ses comptes dans une map indexée par titulaire.
// Pourquoi des POINTEURS dans la map ? Relis la section 6.4 du chapitre.
type Banque struct {
	comptes map[string]*Compte
}

// NewBanque crée la map (une map nil ne peut pas recevoir d'écriture).
func NewBanque() *Banque {
	return &Banque{comptes: make(map[string]*Compte)}
}

// Ouvrir crée un compte pour titulaire et le range dans la map.
// Renvoie ErrCompteExistant (emballée) si le titulaire a déjà un compte.
// TODO 8 : forme « valeur, ok » de la map pour détecter le doublon.
func (b *Banque) Ouvrir(titulaire string) (*Compte, error) {
	return nil, errors.New("TODO")
}

// Compte retrouve un compte. Renvoie ErrCompteInconnu (emballée) s'il n'existe pas.
// TODO 9.
func (b *Banque) Compte(titulaire string) (*Compte, error) {
	return nil, errors.New("TODO")
}

// Virement déplace centimes du compte « de » vers le compte « vers ».
// Si une étape échoue (compte inconnu, solde insuffisant, montant invalide),
// aucun solde ne doit avoir changé et aucune opération ne doit avoir été
// enregistrée. Chaque erreur est emballée : fmt.Errorf("virement: %w", err).
// Après succès, la dernière opération de « de » a le type "virement émis" et
// celle de « vers » le type "virement reçu".
// TODO 10 : Compte(de), Compte(vers), Retirer, Deposer, puis corriger les types.
func (b *Banque) Virement(de, vers string, centimes int64) error {
	return errors.New("TODO")
}
