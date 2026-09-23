// Labo 09. Rôle du fichier : toute la logique du module ; importé par
// cmd/semver (la ligne de commande) et testé par semver_test.go.

// Package semver analyse, compare et fait évoluer des versions sémantiques
// de la forme MAJEUR.MINEUR.CORRECTIF (https://semver.org), et vérifie des
// contraintes à la npm / cargo : « ^1.2.0 », « ~1.2.0 », « >=1.2.0,<2.0.0 ».
package semver

import (
	"errors"
	"fmt"
)

// ErrInvalide est renvoyée (emballée avec %w) par Parse et Compatible quand
// le texte n'est pas une version ou une contrainte valide.
var ErrInvalide = errors.New("version invalide")

// Version est un triplet. Type « valeur » : petit, comparable avec ==, jamais
// modifié (Bump renvoie une nouvelle Version). Receveurs valeur.
type Version struct {
	Majeur, Mineur, Correctif int
}

// String rend Version un fmt.Stringer : « 1.2.3 ».
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Majeur, v.Mineur, v.Correctif)
}

// Parse lit « 1.2.3 » ou « v1.2.3 » (espaces autour tolérés, comme dans un
// tag git). Toute autre forme (pas trois parties, partie non entière ou
// négative, vide) renvoie une erreur qui emballe ErrInvalide.
// TODO 1 : strings.TrimSpace, strings.TrimPrefix "v", strings.Split sur ".",
// strconv.Atoi sur chaque partie.
func Parse(s string) (Version, error) {
	return Version{}, fmt.Errorf("parse %q: TODO: %w", s, ErrInvalide)
}

// Compare renvoie -1 si a < b, 0 si égales, 1 si a > b, champ par champ dans
// l'ordre majeur, mineur, correctif (1.2.0 < 1.10.0 : numérique, pas
// alphabétique). Même convention que cmp.Compare.
// TODO 2.
func Compare(a, b Version) int {
	return 0
}

// Bump incrémente une partie et remet à zéro celles de droite :
// Bump(1.2.3, "minor") = 1.3.0, Bump(1.2.3, "major") = 2.0.0. La partie est
// « major », « minor » ou « patch » ; toute autre valeur est une erreur.
// TODO 3 : un switch.
func Bump(v Version, partie string) (Version, error) {
	return Version{}, errors.New("TODO")
}

// Compatible dit si v satisfait la contrainte :
//
//	^1.2.0          même majeur et >= 1.2.0
//	~1.2.0          même majeur.mineur et >= 1.2.0
//	1.2.0           exactement cette version
//	>=1.2.0 >1.2.0 <2.0.0 <=2.0.0
//	>=1.2.0,<2.0.0  plusieurs contraintes séparées par des virgules : toutes
//	                doivent être vraies, et toutes doivent être valides
//
// Espaces tolérés autour de chaque contrainte. Une contrainte vide ou dont la
// version ne se lit pas renvoie une erreur emballant ErrInvalide.
// TODO 4 : si « , » dans la contrainte, appelle Compatible sur chaque morceau ;
// sinon détecte l'opérateur (teste ">=" et "<=" AVANT ">" et "<"), Parse le
// reste, Compare, et conclus selon l'opérateur.
func Compatible(v Version, contrainte string) (bool, error) {
	return false, errors.New("TODO")
}
