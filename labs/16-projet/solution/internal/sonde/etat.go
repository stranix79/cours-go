// Paquet sonde, solution : etat.go, la mémoire des derniers résultats.
//
// Quoi : Etat garde le dernier Resultat de chaque cible et un compteur de
// vérifications, derrière un mutex, parce que les workers écrivent pendant
// que les handlers HTTP lisent.
// Qui l'appelle : Moteur.Tour écrit (Enregistrer) ; le paquet api lit
// (Resultats, Total).
package sonde

import (
	"sort" // pour rendre les résultats dans un ordre stable
	"sync" // sync.RWMutex
)

// Etat est partagé entre goroutines : toute lecture ou écriture des champs
// passe par le mutex. Le mutex est à l'intérieur de la struct et la struct
// n'est jamais copiée (on manipule un *Etat), sinon go vet râle à raison.
type Etat struct {
	mu       sync.RWMutex        // RW : plusieurs lecteurs à la fois, un seul écrivain
	derniers map[string]Resultat // clé : nom de la cible
	total    uint64              // vérifications depuis le démarrage (compteur Prometheus)
}

// NouvelEtat construit un Etat prêt à l'emploi. Une map nil ne se remplit pas :
// il faut la créer ici, d'où le constructeur.
func NouvelEtat() *Etat {
	return &Etat{derniers: make(map[string]Resultat)}
}

// Enregistrer stocke le résultat et dit si l'état up/down de la cible a changé
// (ou si c'est la première fois qu'on la voit) : c'est ce que le Moteur
// journalise, plutôt que chaque vérification.
func (e *Etat) Enregistrer(r Resultat) (change bool) {
	e.mu.Lock()
	defer e.mu.Unlock() // libéré même si quelque chose panique plus bas
	precedent, existe := e.derniers[r.Cible]
	e.derniers[r.Cible] = r
	e.total++
	return !existe || precedent.Up != r.Up
}

// Resultats renvoie une COPIE des derniers résultats, triée par nom de cible.
// Une copie : l'appelant peut la parcourir sans tenir le verrou, et la map
// interne n'est jamais exposée. Un tri : l'API et /metrics sortent toujours
// dans le même ordre, ce qui rend les tests et les diffs lisibles.
func (e *Etat) Resultats() []Resultat {
	e.mu.RLock()
	defer e.mu.RUnlock()
	liste := make([]Resultat, 0, len(e.derniers))
	for _, r := range e.derniers {
		liste = append(liste, r)
	}
	sort.Slice(liste, func(i, j int) bool { return liste[i].Cible < liste[j].Cible })
	return liste
}

// Total renvoie le nombre de vérifications enregistrées.
func (e *Etat) Total() uint64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.total
}
