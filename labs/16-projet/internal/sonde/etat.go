// Paquet sonde : etat.go, la mémoire des derniers résultats, partagée entre
// les workers (qui écrivent) et les handlers HTTP (qui lisent).
package sonde

import (
	"sync"
)

// Etat garde le dernier Resultat de chaque cible et un compteur de
// vérifications. Tout accès aux champs passe par le mutex. Ne jamais copier
// un Etat : on manipule toujours un *Etat.
type Etat struct {
	mu       sync.RWMutex
	derniers map[string]Resultat // clé : nom de la cible
	total    uint64              // vérifications depuis le démarrage
}

// NouvelEtat construit un Etat prêt à l'emploi (la map doit être créée).
func NouvelEtat() *Etat {
	// TODO 8 : initialiser la map.
	return &Etat{}
}

// Enregistrer stocke r, incrémente le total, et renvoie vrai si l'état up/down
// de la cible a changé ou si c'est la première fois qu'on la voit.
func (e *Etat) Enregistrer(r Resultat) (change bool) {
	// TODO 9 : Lock / defer Unlock ; comparer avec l'ancien résultat.
	_ = r
	return false
}

// Resultats renvoie une COPIE des derniers résultats, triée par nom de cible
// (sort.Slice ou slices.SortFunc). Jamais la map elle-même.
func (e *Etat) Resultats() []Resultat {
	// TODO 10 : RLock / defer RUnlock ; copier dans une slice ; trier.
	return nil
}

// Total renvoie le nombre de vérifications enregistrées.
func (e *Etat) Total() uint64 {
	// TODO 11 : RLock / defer RUnlock.
	return 0
}
