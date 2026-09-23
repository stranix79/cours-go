// Labo 10 : concaténer des mots avec + ou avec strings.Builder, et mesurer.
// Lancé par : go test -bench=Concat -benchmem .
package capot

// ConcatPlus assemble les mots séparés par un espace (sans espace final)
// avec l'opérateur +. Chaque += crée une NOUVELLE string : n mots = n copies.
func ConcatPlus(mots []string) string {
	// TODO 5 : une variable resultat, une boucle avec l'index i : si i > 0,
	// ajoute " " ; puis ajoute le mot. Renvoie resultat.
	return ""
}

// ConcatBuilder fait la même chose avec un strings.Builder, qui garde un
// tampon et ne copie que quand il déborde (import "strings" à ajouter).
func ConcatBuilder(mots []string) string {
	// TODO 6 : var b strings.Builder ; b.WriteString(...) ; b.String().
	return ""
}
