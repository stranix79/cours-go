// Labo 04, solution : un inventaire de serveurs, rôle vers liste d'hôtes.
// Lancé par : go run ./solution   (depuis labs/04-inventaire)
// Testé par : go test ./solution/
//
// L'inventaire est une map[string][]string : la clé est le rôle ("web",
// "postgres"), la valeur la liste des hôtes qui le portent. Toutes les
// fonctions reçoivent la map en paramètre : une map est un pointeur déguisé,
// les modifications faites dans la fonction sont vues par l'appelant.
package main

import (
	"fmt"     // affichage
	"maps"    // maps.Keys : itérateur sur les clés (Go 1.23)
	"slices"  // Sort, Sorted, Contains, Index, Compact, Clone (Go 1.21)
	"strings" // Builder, Join
)

// Inventaire est un nom pour map[string][]string : plus lisible dans les
// signatures, et le chapitre 6 lui ajoutera des méthodes.
type Inventaire map[string][]string

// Ajouter place hote dans la liste du rôle. Renvoie false si l'hôte y était
// déjà (et ne l'ajoute pas deux fois).
func Ajouter(inv Inventaire, role, hote string) bool {
	// inv[role] sur un rôle absent renvoie un slice nil : Contains sur nil
	// vaut false, append sur nil marche. Pas de cas particulier à écrire.
	if slices.Contains(inv[role], hote) {
		return false
	}
	// Réaffectation obligatoire : append peut réallouer, et de toute façon
	// la map doit connaître le nouveau slice (nouvelle longueur).
	inv[role] = append(inv[role], hote)
	return true
}

// Retirer enlève hote du rôle. Renvoie false s'il n'y était pas. Si le rôle
// se retrouve vide, on supprime la clé pour ne pas garder de rôle fantôme.
func Retirer(inv Inventaire, role, hote string) bool {
	i := slices.Index(inv[role], hote)
	if i < 0 {
		return false
	}
	// slices.Delete(s, i, j) retire les éléments d'index [i, j[ et renvoie
	// le slice raccourci (à réaffecter, comme append).
	inv[role] = slices.Delete(inv[role], i, i+1)
	if len(inv[role]) == 0 {
		delete(inv, role)
	}
	return true
}

// Roles renvoie les rôles triés. maps.Keys donne un itérateur (ordre
// aléatoire), slices.Sorted le consomme et renvoie un slice trié.
func Roles(inv Inventaire) []string {
	return slices.Sorted(maps.Keys(inv))
}

// Hotes renvoie tous les hôtes, triés et sans doublon, tous rôles confondus.
func Hotes(inv Inventaire) []string {
	// make avec une capacité estimée évite les réallocations dans la boucle.
	tous := make([]string, 0, len(inv))
	for _, hotes := range inv {
		// append accepte un slice étalé avec ... (chapitre 5.3).
		tous = append(tous, hotes...)
	}
	slices.Sort(tous)
	// Compact ne retire que les doublons CONSÉCUTIFS : d'où le tri avant.
	return slices.Compact(tous)
}

// CompterParRole renvoie le nombre d'hôtes de chaque rôle.
func CompterParRole(inv Inventaire) map[string]int {
	comptes := make(map[string]int, len(inv))
	for role, hotes := range inv {
		comptes[role] = len(hotes)
	}
	return comptes
}

// Doublons renvoie, triés, les hôtes qui apparaissent dans plus d'un rôle.
func Doublons(inv Inventaire) []string {
	// Compter les apparitions : la valeur zéro d'un int est 0, donc
	// vus[h]++ marche dès la première fois sans initialisation.
	vus := make(map[string]int)
	for _, hotes := range inv {
		for _, h := range hotes {
			vus[h]++
		}
	}
	// Un slice nil (var) plutôt que make : si aucun doublon, on renvoie nil,
	// ce que les tests comparent avec len() == 0.
	var doublons []string
	for h, n := range vus {
		if n > 1 {
			doublons = append(doublons, h)
		}
	}
	slices.Sort(doublons)
	return doublons
}

// Lister formate l'inventaire en une ligne par rôle, rôles triés, hôtes
// dans l'ordre d'ajout : "postgres: db01, db02\nweb: web01\n".
func Lister(inv Inventaire) string {
	// strings.Builder accumule les morceaux ; sa valeur zéro est prête.
	var sb strings.Builder
	for _, role := range Roles(inv) {
		// Fprintf écrit dans n'importe quoi qui sait recevoir des octets ;
		// &sb parce que la méthode Write de Builder a un récepteur pointeur
		// (chapitre 6). strings.Join recolle les hôtes avec ", ".
		fmt.Fprintf(&sb, "%s: %s\n", role, strings.Join(inv[role], ", "))
	}
	return sb.String()
}

// Premiers renvoie les n premiers hôtes d'une liste (ou toute la liste si
// elle est plus courte), dans un slice INDÉPENDANT de l'original.
//
// La version naïve `return hotes[:n]` partage le tableau sous-jacent :
// l'appelant qui modifie ou étend le résultat modifie la liste d'origine
// (chapitre 4.4). D'où la copie explicite.
func Premiers(hotes []string, n int) []string {
	if n > len(hotes) {
		n = len(hotes)
	}
	// make(len n) puis copy : le nouveau slice a son propre tableau, avec
	// len == cap == n, donc un append dessus réalloue sans toucher hotes.
	resultat := make([]string, n)
	copy(resultat, hotes[:n])
	return resultat
	// Équivalent en une ligne : return slices.Clone(hotes[:n])
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

	// Le piège n°1 en direct : une vue sur les deux premiers, puis un append
	// qui écrase le troisième élément de l'original si on n'a pas copié.
	tous := Hotes(inv)
	deux := Premiers(tous, 2)
	deux = append(deux, "intrus")
	fmt.Println("Premiers :", deux)
	fmt.Println("Intacts  :", tous)
}
