// Labo 05 : une boîte à outils de fonctions DevOps.
// Lancé par : go run .    Testé par : go test .
//
// Cinq fonctions indépendantes à compléter dans l'ordre. Tout compile.
package main

import (
	"errors"  // errors.New : une erreur fixe
	"fmt"     // affichage, Errorf
	"strings" // Builder, Fields, Cut, TrimSpace, ToLower
	"time"    // time.Duration, time.Sleep
	// TODO : ajoute "unicode" (IsLetter, IsDigit) pour Slugify.
)

// Slugify transforme "Serveur Web #1 (prod)" en "serveur-web-1-prod" :
// minuscules, tout ce qui n'est pas lettre ou chiffre devient un tiret,
// pas de tirets doubles ni aux extrémités. Les lettres accentuées sont des
// lettres ("Été" donne "été").
func Slugify(s string) string {
	// TODO 1 : range sur strings.ToLower(s) donne des runes ; garde celles
	// pour lesquelles unicode.IsLetter(r) || unicode.IsDigit(r) avec
	// sb.WriteRune(r), sinon écris un tiret... sauf si le dernier caractère
	// écrit en était déjà un (un booléen suffit). Termine par
	// strings.TrimSuffix(sb.String(), "-").
	var sb strings.Builder
	return sb.String()
}

// ParseKV transforme "host=web01 port=22" en map[host:web01 port:22].
// Les morceaux sont séparés par des blancs ; un morceau sans "=" ou avec
// une clé vide est ignoré ; la valeur peut contenir un "=" (coupe au premier).
func ParseKV(texte string) map[string]string {
	// TODO 2 : strings.Fields(texte), puis pour chaque morceau
	// cle, valeur, ok := strings.Cut(morceau, "="). N'oublie pas make.
	return nil
}

// Somme additionne un nombre quelconque d'entiers. Sans argument : 0.
func Somme(nombres ...int) int {
	// TODO 3 : nombres est un []int, un range suffit.
	return 0
}

// NouveauCompteur renvoie une fonction qui renvoie 1, puis 2, puis 3...
// Chaque compteur créé a son propre état.
func NouveauCompteur() func() int {
	// TODO 4 : une variable locale n, et une fonction anonyme qui
	// l'incrémente et la renvoie (chapitre 5.5).
	return func() int { return 0 }
}

// ErrEssais est renvoyée (enveloppée) par Retry quand essais < 1.
var ErrEssais = errors.New("le nombre d'essais doit être au moins 1")

// Retry appelle f jusqu'à ce qu'elle renvoie nil, au plus `essais` fois, en
// attendant `attente` entre deux tentatives (pas après la dernière).
// Renvoie nil au premier succès. Si tout a échoué, renvoie la dernière
// erreur enveloppée : fmt.Errorf("échec après %d essais : %w", essais, derniere).
// Si essais < 1, renvoie fmt.Errorf("retry : %w", ErrEssais).
func Retry(essais int, attente time.Duration, f func() error) error {
	// TODO 5 : une boucle for de 1 à essais ; derniere = f() ; return nil
	// si derniere == nil ; time.Sleep(attente) entre deux essais.
	return errors.New("TODO")
}

func main() {
	fmt.Println(Slugify("Serveur Web #1 (prod)"))
	fmt.Println(ParseKV("host=web01 port=22 user=root"))
	fmt.Println(Somme(1, 2, 3), Somme())

	compteur := NouveauCompteur()
	fmt.Println(compteur(), compteur(), compteur())

	// Une fonction instable : échoue deux fois, puis réussit.
	appels := 0
	instable := func() error {
		appels++
		if appels < 3 {
			fmt.Printf("instable : échec %d\n", appels)
			return errors.New("pas encore prêt")
		}
		fmt.Println("instable : ok")
		return nil
	}
	fmt.Println("retry :", Retry(5, 0, instable))

	// Une fonction qui échoue toujours : Retry abandonne et enveloppe.
	toujours := func() error { return errors.New("service injoignable") }
	fmt.Println("retry :", Retry(3, 0, toujours))
}
