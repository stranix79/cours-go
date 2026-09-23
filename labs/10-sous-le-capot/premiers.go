// Labo 10 : un calcul volontairement naïf à profiler avec pprof.
// Lancé par : go test -bench=Premiers -cpuprofile=cpu.prof .
package capot

// EstPremier teste la primalité par divisions successives jusqu'à √n.
// 0 et 1 ne sont pas premiers.
func EstPremier(n int) bool {
	// TODO 7 : n < 2 → false ; pour d de 2 tant que d*d <= n, si n%d == 0
	// → false ; sinon true.
	return false
}

// ComptePremiers compte les nombres premiers strictement inférieurs à max.
func ComptePremiers(max int) int {
	// TODO 8 : une boucle de 2 à max-1 et un compteur.
	return 0
}
