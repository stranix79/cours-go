// Labo 07, solution : des formes géométriques derrière une interface.
//
// Rôle du fichier : l'interface Forme, trois types qui la satisfont sans le
// déclarer, et Total qui ne connaît que l'interface. Appelé par main.go et io.go.
package main

import (
	"fmt"
	"math"
)

// Forme est ce qu'une fonction comme Total attend : deux méthodes, rien de
// plus. Aucun type ci-dessous ne mentionne Forme, et pourtant tous en sont
// (chapitre 7 : satisfaction implicite).
type Forme interface {
	Aire() float64
	Perimetre() float64
}

// Rectangle : largeur et hauteur. Receveurs valeur partout : petite struct,
// jamais modifiée, c'est un type « valeur ».
type Rectangle struct {
	L, H float64
}

func (r Rectangle) Aire() float64      { return r.L * r.H }
func (r Rectangle) Perimetre() float64 { return 2 * (r.L + r.H) }

// String rend Rectangle un fmt.Stringer : %v et %s affichent ce texte.
// %g affiche le float « au plus court » : 3 et pas 3.000000.
func (r Rectangle) String() string { return fmt.Sprintf("rectangle %gx%g", r.L, r.H) }

// Cercle : un rayon.
type Cercle struct {
	R float64
}

func (c Cercle) Aire() float64      { return math.Pi * c.R * c.R }
func (c Cercle) Perimetre() float64 { return 2 * math.Pi * c.R }
func (c Cercle) String() string     { return fmt.Sprintf("cercle r=%g", c.R) }

// Triangle : trois côtés. L'aire vient de la formule de Héron, qui n'a besoin
// que des côtés : s = demi-périmètre, aire = sqrt(s(s-a)(s-b)(s-c)).
type Triangle struct {
	A, B, C float64
}

func (t Triangle) Perimetre() float64 { return t.A + t.B + t.C }

func (t Triangle) Aire() float64 {
	s := t.Perimetre() / 2
	return math.Sqrt(s * (s - t.A) * (s - t.B) * (s - t.C))
}

func (t Triangle) String() string { return fmt.Sprintf("triangle %g-%g-%g", t.A, t.B, t.C) }

// Ces trois lignes ne produisent aucun code : elles demandent au compilateur
// de vérifier, ici et maintenant, que chaque type satisfait bien Forme. Si
// une méthode disparaît ou change de signature, l'erreur pointe cette ligne.
var (
	_ Forme = Rectangle{}
	_ Forme = Cercle{}
	_ Forme = Triangle{}
)

// Total additionne les aires. La fonction ne sait pas quelles formes elle
// reçoit ; ajoute demain un Hexagone avec Aire et Perimetre, elle marchera.
func Total(formes []Forme) float64 {
	total := 0.0
	for _, f := range formes {
		total += f.Aire()
	}
	return total
}
