// Labo 05, solution : une boîte à outils de fonctions DevOps.
// Lancé par : go run ./solution   (depuis labs/05-boite-a-outils)
// Testé par : go test ./solution/
//
// Cinq fonctions indépendantes : Slugify, ParseKV, Somme (variadique),
// NouveauCompteur (closure) et Retry (qui reçoit une fonction et gère les
// erreurs). main() fait une démo de chacune. Ce fichier deviendra un paquet
// importable au chapitre 9.
package main

import (
	"errors"  // errors.New : une erreur fixe
	"fmt"     // affichage, Errorf
	"strings" // Builder, Fields, Cut, TrimSpace, ToLower
	"time"    // time.Duration, time.Sleep
	"unicode" // IsLetter, IsDigit : tests sur une rune
)

// Slugify transforme "Serveur Web #1 (prod)" en "serveur-web-1-prod" :
// minuscules, tout ce qui n'est pas lettre ou chiffre devient un tiret,
// pas de tirets doubles ni aux extrémités. Utile pour des noms de fichiers,
// d'URL, de conteneurs.
func Slugify(s string) string {
	var sb strings.Builder
	// dernierTiret évite d'écrire deux tirets de suite : on ne pose un tiret
	// que si le précédent caractère écrit n'en était pas un. Initialisé à
	// true pour ne jamais commencer par un tiret.
	dernierTiret := true
	// range sur une chaîne décode l'UTF-8 et donne des runes : "é" est une
	// seule rune, et unicode.IsLetter la reconnaît comme une lettre.
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			sb.WriteRune(r)
			dernierTiret = false
		case !dernierTiret:
			sb.WriteByte('-')
			dernierTiret = true
		}
	}
	// Il peut rester un tiret final si la chaîne se terminait par un
	// séparateur ("prod)") : TrimSuffix l'enlève, sans effet sinon.
	return strings.TrimSuffix(sb.String(), "-")
}

// ParseKV transforme "host=web01 port=22" en map[host:web01 port:22].
// Les morceaux sont séparés par des blancs ; un morceau sans "=" est ignoré ;
// la valeur peut contenir un "=" (on coupe au premier).
func ParseKV(texte string) map[string]string {
	resultat := make(map[string]string)
	// Fields coupe sur n'importe quelle suite de blancs et ignore les vides.
	for _, morceau := range strings.Fields(texte) {
		// strings.Cut coupe au premier séparateur et renvoie trois valeurs :
		// avant, après, et un booléen "trouvé". C'est le partition() de Python.
		cle, valeur, ok := strings.Cut(morceau, "=")
		if !ok || cle == "" {
			continue
		}
		resultat[cle] = valeur
	}
	return resultat
}

// Somme additionne un nombre quelconque d'entiers. Sans argument, elle
// renvoie 0 : `nombres` est alors un slice vide, et range ne fait rien.
func Somme(nombres ...int) int {
	total := 0
	for _, n := range nombres {
		total += n
	}
	return total
}

// NouveauCompteur renvoie une fonction qui renvoie 1, puis 2, puis 3...
// La variable n est capturée par la closure : elle survit au retour de
// NouveauCompteur, et chaque compteur créé a la sienne.
func NouveauCompteur() func() int {
	n := 0
	return func() int {
		n++
		return n
	}
}

// ErrEssais est renvoyée (enveloppée) par Retry quand essais < 1.
var ErrEssais = errors.New("le nombre d'essais doit être au moins 1")

// Retry appelle f jusqu'à ce qu'elle renvoie nil, au plus `essais` fois, en
// attendant `attente` entre deux tentatives. Renvoie nil au premier succès,
// ou, si tout a échoué, la dernière erreur enveloppée avec le nombre
// d'essais. f est passée SANS parenthèses par l'appelant : c'est Retry qui
// l'appelle.
func Retry(essais int, attente time.Duration, f func() error) error {
	if essais < 1 {
		return fmt.Errorf("retry : %w", ErrEssais)
	}
	var derniere error
	for i := 1; i <= essais; i++ {
		derniere = f()
		if derniere == nil {
			return nil // succès : on sort tout de suite
		}
		// Pas d'attente après le dernier échec : il n'y aura pas d'essai suivant.
		if i < essais {
			time.Sleep(attente)
		}
	}
	// %w garde l'erreur d'origine : l'appelant peut faire errors.Is dessus.
	return fmt.Errorf("échec après %d essais : %w", essais, derniere)
}

func main() {
	fmt.Println(Slugify("Serveur Web #1 (prod)"))
	fmt.Println(ParseKV("host=web01 port=22 user=root"))
	fmt.Println(Somme(1, 2, 3), Somme())

	compteur := NouveauCompteur()
	fmt.Println(compteur(), compteur(), compteur())

	// Une fonction instable : échoue deux fois, puis réussit. Le compteur
	// d'appels est une variable locale de main capturée par la closure,
	// exactement comme n dans NouveauCompteur.
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
