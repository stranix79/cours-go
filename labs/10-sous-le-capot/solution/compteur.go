// Labo 10, solution : un compteur partagé cassé et sa version sûre.
// Lancé par : RACE=1 go test -race -run Casse ./solution   puis   go test -race ./solution
package capot

import "sync"

// Compteur : incrémenter, lire. Les deux implémentations ci-dessous la
// satisfont, ce qui permet à Marteler de tester l'une ou l'autre.
type Compteur interface {
	Inc()
	Valeur() int
}

// CompteurCasse est le témoin : aucune protection. On le garde tel quel
// pour que RACE=1 go test -race montre à quoi ressemble un rapport de course.
type CompteurCasse struct{ n int }

func (c *CompteurCasse) Inc()        { c.n++ }      // lire, ajouter, écrire : entrelaçable
func (c *CompteurCasse) Valeur() int { return c.n } // lecture non protégée : course aussi

// CompteurSur : le mutex est un champ non exporté, juste à côté de n. Le
// type est toujours utilisé par pointeur (méthodes sur *CompteurSur) : copier
// un mutex verrouillé serait une faute, et go vet le signale.
type CompteurSur struct {
	mu sync.Mutex
	n  int
}

// Inc prend le verrou, incrémente, rend le verrou. Le defer garantit
// l'Unlock même si le code entre les deux paniquait un jour.
func (c *CompteurSur) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
}

// Valeur lit sous verrou. Sans ça, une lecture pendant une écriture est une
// course, même si « ça ne fait que lire ».
func (c *CompteurSur) Valeur() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// Marteler lance `goroutines` goroutines de `fois` incréments et attend.
// wg.Go (Go 1.25) fait Add, go et Done. `for range n` (Go 1.22) itère n fois
// sans variable.
func Marteler(c Compteur, goroutines, fois int) {
	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			for range fois {
				c.Inc()
			}
		})
	}
	wg.Wait()
}
