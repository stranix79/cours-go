// Labo 13 : le stockage en mémoire des tâches.
// Utilisé par : api.go (les handlers) et api_test.go.
package main

import "sync"

// Tache est ce que l'API renvoie et reçoit en JSON. Les tags `json:"..."`
// fixent le nom des clés (minuscules), sinon encoding/json prend le nom du
// champ tel quel ("ID", "Titre").
type Tache struct {
	ID    int    `json:"id"`
	Titre string `json:"titre"`
	Faite bool   `json:"faite"`
}

// Magasin garde les tâches en mémoire. Plusieurs requêtes HTTP sont servies
// en parallèle (une goroutine par connexion), donc la map doit être protégée
// par un mutex : une map Go n'est pas sûre en écriture concurrente.
type Magasin struct {
	mu      sync.Mutex
	suivant int
	taches  map[int]Tache
}

// NouveauMagasin renvoie un magasin vide, prêt à servir, dont le premier
// identifiant sera 1.
func NouveauMagasin() *Magasin {
	// TODO 1 : initialise la map (une map nil plante à l'écriture) et
	// suivant à 1.
	return &Magasin{}
}

// Liste renvoie toutes les tâches, triées par ID croissant. Une slice vide
// (pas nil) s'il n'y en a aucune, pour que le JSON soit [] et pas null.
func (m *Magasin) Liste() []Tache {
	// TODO 2 : verrouille, copie les valeurs de la map dans une slice,
	// déverrouille, trie par ID (slices.SortFunc ou sort.Slice).
	return nil
}

// Ajoute crée une tâche non faite avec le titre donné, lui attribue le
// prochain ID et la renvoie.
func (m *Magasin) Ajoute(titre string) Tache {
	// TODO 3
	return Tache{}
}

// Lit renvoie la tâche d'identifiant id et true, ou la valeur zéro et false
// si elle n'existe pas.
func (m *Magasin) Lit(id int) (Tache, bool) {
	// TODO 4
	return Tache{}, false
}

// Remplace met à jour le titre et l'état de la tâche id. Renvoie la tâche
// mise à jour et true, ou false si id n'existe pas. L'ID ne change jamais.
func (m *Magasin) Remplace(id int, t Tache) (Tache, bool) {
	// TODO 5
	return Tache{}, false
}

// Supprime retire la tâche id et renvoie true, ou false si elle n'existait pas.
func (m *Magasin) Supprime(id int) bool {
	// TODO 6
	return false
}
