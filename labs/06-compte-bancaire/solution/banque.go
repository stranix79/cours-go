// Labo 06, solution : la banque, qui tient les comptes et fait les virements.
//
// Rôle du fichier : le type Banque et sa méthode Virement. Appelé par main.go.
package main

import (
	"errors"
	"fmt"
)

// Sentinelles propres à la banque.
var (
	ErrCompteInconnu  = errors.New("compte inconnu")
	ErrCompteExistant = errors.New("compte déjà ouvert")
)

// Banque range ses comptes dans une map indexée par titulaire. La map contient
// des POINTEURS : les éléments d'une map ne sont pas adressables, donc
// map[string]Compte interdirait d'appeler une méthode à receveur pointeur
// dessus (chapitre 6). Avec *Compte, on modifie le compte en place.
type Banque struct {
	comptes map[string]*Compte
}

// NewBanque crée la map : la valeur zéro d'une map est nil, et écrire dans
// une map nil panique. C'est le cas typique où un constructeur est nécessaire.
func NewBanque() *Banque {
	return &Banque{comptes: make(map[string]*Compte)}
}

// Ouvrir crée un compte et le range. Le pointeur renvoyé est le même que celui
// stocké dans la map : l'appelant et la banque voient le même solde.
func (b *Banque) Ouvrir(titulaire string) (*Compte, error) {
	if _, existe := b.comptes[titulaire]; existe {
		return nil, fmt.Errorf("ouvrir %q: %w", titulaire, ErrCompteExistant)
	}
	c := NewCompte(titulaire)
	b.comptes[titulaire] = c
	return c, nil
}

// Compte retrouve un compte par titulaire. La forme « valeur, ok » de la map
// distingue « absent » de « présent mais nil ».
func (b *Banque) Compte(titulaire string) (*Compte, error) {
	c, ok := b.comptes[titulaire]
	if !ok {
		return nil, fmt.Errorf("compte %q: %w", titulaire, ErrCompteInconnu)
	}
	return c, nil
}

// Virement déplace des centimes d'un compte à l'autre. Il est atomique au sens
// simple : si le retrait échoue (solde insuffisant, montant invalide), rien
// n'est modifié, parce que Retirer valide AVANT de toucher au solde et que
// Deposer n'est appelé qu'après. Chaque étape emballe l'erreur du dessous.
func (b *Banque) Virement(de, vers string, centimes int64) error {
	source, err := b.Compte(de)
	if err != nil {
		return fmt.Errorf("virement: %w", err)
	}
	cible, err := b.Compte(vers)
	if err != nil {
		return fmt.Errorf("virement: %w", err)
	}
	if err := source.Retirer(centimes, "vers "+vers); err != nil {
		return fmt.Errorf("virement: %w", err)
	}
	// Retirer a réussi, donc centimes > 0 : Deposer ne peut pas échouer. On
	// vérifie quand même : ignorer une erreur est le seul vrai péché en Go.
	if err := cible.Deposer(centimes, "de "+de); err != nil {
		return fmt.Errorf("virement: %w", err)
	}
	// Retirer et Deposer ont enregistré « retrait » et « dépôt » ; on corrige
	// le type des deux dernières opérations pour que le relevé dise « virement ».
	source.Operations[len(source.Operations)-1].Type = "virement émis"
	cible.Operations[len(cible.Operations)-1].Type = "virement reçu"
	return nil
}
