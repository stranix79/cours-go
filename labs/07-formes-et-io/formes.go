// Labo 07 : des formes géométriques derrière une interface.
// Rôle du fichier : l'interface Forme, trois types qui la satisfont, et Total.
// Appelé par main.go et io.go.
package main

import (
	"fmt"
	"math"
)

// Forme est ce que Total et EcrireRapport attendent : deux méthodes.
// Aucun type ci-dessous ne mentionne Forme : c'est la satisfaction implicite.
type Forme interface {
	Aire() float64
	Perimetre() float64
}

// Rectangle : largeur et hauteur.
type Rectangle struct {
	L, H float64
}

// TODO 1 : Aire (L*H), Perimetre (2*(L+H)) et String ("rectangle 3x4", avec %g).
func (r Rectangle) Aire() float64      { return 0 }
func (r Rectangle) Perimetre() float64 { return 0 }
func (r Rectangle) String() string     { return "TODO" }

// Cercle : un rayon.
type Cercle struct {
	R float64
}

// TODO 2 : Aire (math.Pi*R*R), Perimetre (2*math.Pi*R), String ("cercle r=1").
func (c Cercle) Aire() float64      { return 0 }
func (c Cercle) Perimetre() float64 { return 0 }
func (c Cercle) String() string     { return "TODO" }

// Triangle : trois côtés. Aire par la formule de Héron :
// s = périmètre/2, aire = sqrt(s*(s-A)*(s-B)*(s-C)) avec math.Sqrt.
type Triangle struct {
	A, B, C float64
}

// TODO 3 : Perimetre, Aire, String ("triangle 3-4-5").
func (t Triangle) Aire() float64      { return 0 }
func (t Triangle) Perimetre() float64 { return 0 }
func (t Triangle) String() string     { return "TODO" }

// Ces lignes vérifient à la compilation que chaque type satisfait Forme.
// Retire une méthode ci-dessus et regarde le message du compilateur.
var (
	_ Forme = Rectangle{}
	_ Forme = Cercle{}
	_ Forme = Triangle{}
)

// Total additionne les aires de toutes les formes.
// TODO 4 : une boucle range, sans savoir quels types concrets sont dedans.
func Total(formes []Forme) float64 {
	return 0
}

// Les deux imports ci-dessus servent dès que tu écris Aire et String ; en
// attendant, ces deux lignes empêchent « imported and not used ». Supprime-les
// quand tu n'en as plus besoin.
var _ = math.Pi
var _ = fmt.Sprintf
