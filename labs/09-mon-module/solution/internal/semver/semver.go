// Labo 09, solution. Rôle du fichier : toute la logique du module ; importé
// par cmd/semver (la ligne de commande) et testé par semver_test.go.

// Package semver analyse, compare et fait évoluer des versions sémantiques
// de la forme MAJEUR.MINEUR.CORRECTIF (https://semver.org), et vérifie des
// contraintes à la npm / cargo : « ^1.2.0 », « ~1.2.0 », « >=1.2.0,<2.0.0 ».
package semver

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalide est renvoyée (emballée) par Parse et Compatible quand le texte
// n'est pas une version ou une contrainte valide.
var ErrInvalide = errors.New("version invalide")

// Version est un triplet. Type « valeur » : petit, comparable avec ==,
// jamais modifié (Bump renvoie une nouvelle Version). Receveurs valeur.
type Version struct {
	Majeur, Mineur, Correctif int
}

// String rend Version un fmt.Stringer : « 1.2.3 ».
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Majeur, v.Mineur, v.Correctif)
}

// Parse lit « 1.2.3 » ou « v1.2.3 » (espaces autour tolérés, comme dans un
// tag git). Toute autre forme renvoie une erreur qui emballe ErrInvalide.
func Parse(s string) (Version, error) {
	texte := strings.TrimSpace(s)
	texte = strings.TrimPrefix(texte, "v")
	parties := strings.Split(texte, ".")
	if len(parties) != 3 {
		return Version{}, fmt.Errorf("parse %q: attendu MAJEUR.MINEUR.CORRECTIF: %w", s, ErrInvalide)
	}
	var nombres [3]int
	for i, p := range parties {
		n, err := strconv.Atoi(p)
		// Atoi accepte « -2 » et « +2 » : on refuse tout ce qui n'est pas
		// un entier positif écrit simplement.
		if err != nil || n < 0 || strings.HasPrefix(p, "+") {
			return Version{}, fmt.Errorf("parse %q: partie %q n'est pas un entier positif: %w", s, p, ErrInvalide)
		}
		nombres[i] = n
	}
	return Version{Majeur: nombres[0], Mineur: nombres[1], Correctif: nombres[2]}, nil
}

// Compare renvoie -1 si a < b, 0 si égales, 1 si a > b, champ par champ dans
// l'ordre majeur, mineur, correctif. C'est la convention de cmp.Compare et de
// slices.SortFunc : Compare peut leur être passée telle quelle.
func Compare(a, b Version) int {
	switch {
	case a.Majeur != b.Majeur:
		return signe(a.Majeur - b.Majeur)
	case a.Mineur != b.Mineur:
		return signe(a.Mineur - b.Mineur)
	default:
		return signe(a.Correctif - b.Correctif)
	}
}

// signe est un helper privé (minuscule) : invisible hors du paquet.
func signe(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

// Bump incrémente une partie et remet à zéro celles de droite, comme le veut
// semver : Bump(1.2.3, "minor") = 1.3.0. La partie est « major », « minor »
// ou « patch » (les mots anglais, ceux de npm version et cargo release).
func Bump(v Version, partie string) (Version, error) {
	switch partie {
	case "major":
		return Version{Majeur: v.Majeur + 1}, nil
	case "minor":
		return Version{Majeur: v.Majeur, Mineur: v.Mineur + 1}, nil
	case "patch":
		return Version{Majeur: v.Majeur, Mineur: v.Mineur, Correctif: v.Correctif + 1}, nil
	}
	return Version{}, fmt.Errorf("bump: partie inconnue %q (major, minor ou patch)", partie)
}

// Compatible dit si v satisfait la contrainte :
//
//	^1.2.0          même majeur et >= 1.2.0 (l'API ne casse pas)
//	~1.2.0          même majeur.mineur et >= 1.2.0 (correctifs seulement)
//	1.2.0           exactement cette version
//	>=1.2.0 >1.2.0 <2.0.0 <=2.0.0
//	>=1.2.0,<2.0.0  plusieurs contraintes séparées par des virgules : toutes
//
// Une contrainte vide ou dont la version ne se lit pas renvoie une erreur
// emballant ErrInvalide.
func Compatible(v Version, contrainte string) (bool, error) {
	// Plusieurs contraintes : ET logique. On les évalue TOUTES, même après
	// une fausse, pour qu'une contrainte mal écrite soit toujours signalée.
	if strings.Contains(contrainte, ",") {
		toutes := true
		for _, c := range strings.Split(contrainte, ",") {
			ok, err := Compatible(v, c)
			if err != nil {
				return false, err
			}
			toutes = toutes && ok
		}
		return toutes, nil
	}

	contrainte = strings.TrimSpace(contrainte)
	if contrainte == "" {
		return false, fmt.Errorf("contrainte vide: %w", ErrInvalide)
	}
	// Les opérateurs à deux caractères AVANT ceux à un seul, sinon « >=1.2.0 »
	// serait lu comme « > » suivi de « =1.2.0 ». L'ordre de la slice compte.
	operateur := ""
	for _, op := range []string{"^", "~", ">=", "<=", ">", "<"} {
		if strings.HasPrefix(contrainte, op) {
			operateur = op
			break
		}
	}
	borne, err := Parse(strings.TrimPrefix(contrainte, operateur))
	if err != nil {
		return false, fmt.Errorf("contrainte %q: %w", contrainte, err)
	}

	cmp := Compare(v, borne)
	switch operateur {
	case "^":
		return v.Majeur == borne.Majeur && cmp >= 0, nil
	case "~":
		return v.Majeur == borne.Majeur && v.Mineur == borne.Mineur && cmp >= 0, nil
	case ">=":
		return cmp >= 0, nil
	case "<=":
		return cmp <= 0, nil
	case ">":
		return cmp > 0, nil
	case "<":
		return cmp < 0, nil
	}
	return cmp == 0, nil
}
