// Labo 10 : un compteur partagé cassé (course de données) et sa version sûre.
// Lancé par : RACE=1 go test -race -run Casse .   puis   go test -race .
package capot

import "sync"

// Compteur est ce qu'on attend d'un compteur : incrémenter, lire.
type Compteur interface {
	Inc()
	Valeur() int
}

// CompteurCasse incrémente sans aucune protection. Il est VOLONTAIREMENT
// faux : c.n++ est trois instructions (lire, ajouter, écrire) que deux
// goroutines peuvent entrelacer. Ne le corrige pas, il sert de témoin.
type CompteurCasse struct{ n int }

func (c *CompteurCasse) Inc()        { c.n++ }
func (c *CompteurCasse) Valeur() int { return c.n }

// CompteurSur protège n par un mutex : une seule goroutine à la fois entre
// Lock et Unlock.
type CompteurSur struct {
	mu sync.Mutex // le verrou vit À CÔTÉ de la donnée qu'il protège
	n  int
}

// Inc incrémente sous verrou.
func (c *CompteurSur) Inc() {
	// TODO 3 : c.mu.Lock(), defer c.mu.Unlock(), puis c.n++.
}

// Valeur lit sous verrou (une lecture non protégée est AUSSI une course).
func (c *CompteurSur) Valeur() int {
	// TODO 4 : même schéma, renvoie c.n.
	return 0
}

// Marteler lance `goroutines` goroutines qui appellent chacune c.Inc()
// `fois` fois, et attend qu'elles aient toutes fini. Fourni : c'est le banc
// d'essai, pas l'exercice.
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
