// Labo 13, solution : le stockage en mémoire des tâches.
// Utilisé par : api.go (les handlers) et api_test.go.
//
// Ce fichier ne sait rien de HTTP : il pourrait servir tel quel à une
// interface en ligne de commande. Séparer le stockage des handlers est ce qui
// rend les deux testables indépendamment.
package main

import (
	"slices" // slices.SortFunc : tri générique d'une slice (Go 1.21+)
	"sync"   // sync.Mutex : exclusion mutuelle entre goroutines
)

// Tache est ce que l'API renvoie et reçoit en JSON. Les tags `json:"..."`
// fixent le nom des clés (minuscules), sinon encoding/json prend le nom du
// champ tel quel ("ID", "Titre"). Les champs sont exportés (majuscule) parce
// que encoding/json, qui est dans un autre paquet, ne voit pas les autres.
type Tache struct {
	ID    int    `json:"id"`
	Titre string `json:"titre"`
	Faite bool   `json:"faite"`
}

// Magasin garde les tâches en mémoire. Le serveur HTTP lance une goroutine
// par connexion, donc plusieurs handlers peuvent appeler ces méthodes en
// même temps : la map est protégée par un mutex. Le mutex est un champ
// (pas un pointeur) et Magasin ne se manipule que par pointeur, sinon on
// copierait le verrou (go vet le signale : "copylocks").
type Magasin struct {
	mu      sync.Mutex
	suivant int           // prochain identifiant à attribuer
	taches  map[int]Tache // les tâches, indexées par ID
}

// NouveauMagasin renvoie un magasin vide, prêt à servir. C'est le
// « constructeur » à la Go : une fonction Nouveau... qui renvoie un pointeur.
func NouveauMagasin() *Magasin {
	return &Magasin{
		suivant: 1,
		taches:  make(map[int]Tache), // une map nil se lit mais ne s'écrit pas
	}
}

// Liste renvoie toutes les tâches, triées par ID croissant. On renvoie une
// slice vide et non nil pour que le JSON soit [] et pas null : un client
// JavaScript ou Python préfère une liste vide à une absence de liste.
func (m *Magasin) Liste() []Tache {
	m.mu.Lock()
	// defer Unlock : le verrou est rendu quoi qu'il arrive, même en cas de
	// panic. La section critique est courte : on copie, on ne trie pas sous
	// le verrou... mais ici trier est si rapide que ça ne change rien.
	defer m.mu.Unlock()
	liste := make([]Tache, 0, len(m.taches))
	for _, t := range m.taches {
		liste = append(liste, t) // Tache est une valeur : append copie
	}
	// L'ordre d'itération d'une map est aléatoire (chapitre 4) : on trie.
	slices.SortFunc(liste, func(a, b Tache) int { return a.ID - b.ID })
	return liste
}

// Ajoute crée une tâche non faite avec le titre donné, lui attribue le
// prochain ID et la renvoie (par valeur : l'appelant a sa copie).
func (m *Magasin) Ajoute(titre string) Tache {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := Tache{ID: m.suivant, Titre: titre} // Faite vaut false par défaut
	m.taches[t.ID] = t
	m.suivant++
	return t
}

// Lit renvoie la tâche id et true, ou la valeur zéro et false. C'est la
// même convention « valeur, ok » que la lecture d'une map.
func (m *Magasin) Lit(id int) (Tache, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.taches[id]
	return t, ok
}

// Remplace met à jour le titre et l'état de la tâche id. L'ID reçu dans t
// est ignoré : c'est celui de l'URL qui fait foi, un client ne peut pas
// déplacer une tâche en envoyant un autre id dans le corps.
func (m *Magasin) Remplace(id int, t Tache) (Tache, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.taches[id]; !ok {
		return Tache{}, false
	}
	t.ID = id
	m.taches[id] = t
	return t, true
}

// Supprime retire la tâche id et dit si elle existait. delete sur une clé
// absente ne fait rien et ne se plaint pas, d'où le test préalable.
func (m *Magasin) Supprime(id int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.taches[id]; !ok {
		return false
	}
	delete(m.taches, id)
	return true
}
