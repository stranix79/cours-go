// Labo 04 : un inventaire de serveurs, rôle vers liste d'hôtes.
// Lancé par : go run .    Testé par : go test .
//
// Complète les TODO dans l'ordre. Tout compile dès maintenant.
package main

import (
	"fmt"     // affichage
	"strings" // Builder, Join
	// TODO : ajoute "slices" (Contains, Index, Delete, Sort, Compact) et
	// "maps" (Keys) au fur et à mesure : un import inutilisé ne compile pas.
)

// Inventaire est un nom pour map[string][]string : rôle vers hôtes.
type Inventaire map[string][]string

// Ajouter place hote dans la liste du rôle. Renvoie false si l'hôte y était
// déjà (et ne l'ajoute pas deux fois). Un rôle absent est créé au passage.
func Ajouter(inv Inventaire, role, hote string) bool {
	// TODO 1 : slices.Contains(inv[role], hote) pour le doublon, puis
	// inv[role] = append(inv[role], hote). Lire inv[role] sur un rôle absent
	// donne un slice nil, sur lequel Contains et append marchent.
	return false
}

// Retirer enlève hote du rôle. Renvoie false s'il n'y était pas. Si le rôle
// se retrouve vide, supprime la clé de la map.
func Retirer(inv Inventaire, role, hote string) bool {
	// TODO 2 : slices.Index pour trouver la position (-1 si absent), puis
	// inv[role] = slices.Delete(inv[role], i, i+1). Enfin delete(inv, role)
	// si len(inv[role]) == 0.
	return false
}

// Roles renvoie les rôles triés.
func Roles(inv Inventaire) []string {
	// TODO 3 : slices.Sorted(maps.Keys(inv))
	return nil
}

// Hotes renvoie tous les hôtes, triés et sans doublon, tous rôles confondus.
func Hotes(inv Inventaire) []string {
	// TODO 4 : rassemble tout dans un slice (append(tous, hotes...)),
	// slices.Sort, puis slices.Compact (qui ne retire que les doublons
	// consécutifs : d'où le tri d'abord).
	return nil
}

// CompterParRole renvoie le nombre d'hôtes de chaque rôle.
func CompterParRole(inv Inventaire) map[string]int {
	// TODO 5 : une map créée avec make, un range sur inv.
	return nil
}

// Doublons renvoie, triés, les hôtes qui apparaissent dans plus d'un rôle.
func Doublons(inv Inventaire) []string {
	// TODO 6 : compte les apparitions de chaque hôte dans une map[string]int
	// (vus[h]++ marche sans initialisation : la valeur zéro est 0), garde
	// ceux qui dépassent 1, trie.
	return nil
}

// Lister formate l'inventaire en une ligne par rôle, rôles triés, hôtes
// dans l'ordre d'ajout : "postgres: db01, db02\nweb: web01\n".
func Lister(inv Inventaire) string {
	// TODO 7 : var sb strings.Builder ; pour chaque rôle de Roles(inv),
	// fmt.Fprintf(&sb, "%s: %s\n", role, strings.Join(inv[role], ", ")).
	var sb strings.Builder
	return sb.String()
}

// Premiers renvoie les n premiers hôtes (ou toute la liste si elle est plus
// courte), dans un slice INDÉPENDANT de l'original.
//
// La version ci-dessous est la version naïve : elle compile, elle passe
// les deux premiers tests... et elle échoue aux deux suivants, parce que
// hotes[:n] partage le tableau de hotes (chapitre 4.4). Lance
// `go test -v -run TestPremiers .` et lis les messages, puis corrige avec
// make + copy (ou slices.Clone).
func Premiers(hotes []string, n int) []string {
	if n > len(hotes) {
		n = len(hotes)
	}
	// TODO 8 : remplace cette ligne par une vraie copie.
	return hotes[:n]
}

func main() {
	inv := Inventaire{}
	Ajouter(inv, "web", "web01")
	Ajouter(inv, "web", "web02")
	Ajouter(inv, "postgres", "db01")
	Ajouter(inv, "postgres", "db02")
	Ajouter(inv, "monitoring", "web01") // le même hôte dans deux rôles
	Ajouter(inv, "web", "web01")        // refusé : déjà là

	fmt.Print(Lister(inv))
	fmt.Println("Hôtes    :", Hotes(inv))
	fmt.Println("Doublons :", Doublons(inv))
	fmt.Println("Comptes  :", CompterParRole(inv))

	Retirer(inv, "monitoring", "web01") // le rôle devient vide et disparaît
	fmt.Println("Rôles    :", Roles(inv))

	// Le piège n°1 en direct : avec la version naïve de Premiers, l'append
	// ci-dessous écrase le troisième hôte de `tous`.
	tous := Hotes(inv)
	deux := Premiers(tous, 2)
	deux = append(deux, "intrus")
	fmt.Println("Premiers :", deux)
	fmt.Println("Intacts  :", tous)
}
