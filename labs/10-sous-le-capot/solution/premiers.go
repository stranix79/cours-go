// Labo 10, solution : un calcul volontairement naïf à profiler.
// Lancé par : go test -bench=Premiers -cpuprofile=cpu.prof ./solution
package capot

// EstPremier : divisions successives jusqu'à la racine carrée. On compare
// d*d <= n plutôt que d <= sqrt(n) pour rester en entiers. C'est O(√n) par
// nombre, et c'est cette boucle que pprof montrera en tête.
func EstPremier(n int) bool {
	if n < 2 {
		return false // 0 et 1 ne sont pas premiers
	}
	for d := 2; d*d <= n; d++ {
		if n%d == 0 {
			return false // un diviseur suffit
		}
	}
	return true
}

// ComptePremiers compte les premiers dans [2, max[. Rien d'astucieux : le
// but du labo est de voir où passe le temps, pas d'aller vite.
func ComptePremiers(max int) int {
	total := 0
	for n := 2; n < max; n++ {
		if EstPremier(n) {
			total++
		}
	}
	return total
}
