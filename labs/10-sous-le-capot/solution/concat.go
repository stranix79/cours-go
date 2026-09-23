// Labo 10, solution : concaténer avec + ou avec strings.Builder.
// Lancé par : go test -bench=Concat -benchmem ./solution
package capot

import "strings"

// ConcatPlus : à chaque +=, Go alloue une nouvelle string de la taille
// totale et y copie l'ancienne plus le morceau. Pour n mots, on copie
// 1 + 2 + ... + n fois la taille moyenne : quadratique, et n allocations
// que le ramasse-miettes devra balayer.
func ConcatPlus(mots []string) string {
	resultat := ""
	for i, mot := range mots {
		if i > 0 {
			resultat += " " // nouvelle string
		}
		resultat += mot // encore une
	}
	return resultat
}

// ConcatBuilder : le Builder garde un []byte interne qui double quand il
// déborde (comme append). WriteString copie le morceau à la fin, sans
// réallouer tant qu'il y a de la place. String() rend le résultat sans copie
// supplémentaire. Quelques allocations au total au lieu de 2n.
func ConcatBuilder(mots []string) string {
	var b strings.Builder // la valeur zéro est prête à l'emploi
	for i, mot := range mots {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(mot)
	}
	return b.String()
}
