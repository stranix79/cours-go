// Labo 13, solution : les handlers HTTP, le routeur et le middleware.
// Utilisé par : main.go (run) et api_test.go.
//
// Tout est en bibliothèque standard : le ServeMux de Go 1.22 sait router par
// méthode ("GET /taches") et extraire un paramètre de chemin ("{id}"), ce qui
// suffit à une API de cette taille sans chi ni gin.
package main

import (
	"encoding/json" // encoder/décoder le JSON
	"errors"        // errors.New pour les erreurs de validation
	"log"           // journalisation simple sur stderr
	"net/http"      // le serveur HTTP et ses types
	"strconv"       // strconv.Atoi pour l'id du chemin
	"strings"       // strings.TrimSpace pour valider le titre
	"time"          // mesurer la durée d'une requête
)

// api regroupe ce dont les handlers ont besoin : le magasin. Les handlers
// sont des méthodes sur *api, ce qui évite les closures et se lit mieux
// quand il y en a cinq. C'est le patron qu'on retrouve dans Deckhand
// (un type Hub avec ses méthodes handlers).
type api struct {
	magasin *Magasin
}

// nouveauRouteur construit le ServeMux avec les cinq routes de l'API et
// l'enveloppe dans le middleware de journalisation. Il renvoie http.Handler,
// l'interface, pas *http.ServeMux : l'appelant n'a pas à savoir ce qu'il y a
// dedans, et les tests peuvent l'appeler directement avec ServeHTTP.
func nouveauRouteur(m *Magasin) http.Handler {
	a := &api{magasin: m}
	mux := http.NewServeMux()
	// Le motif "MÉTHODE /chemin" (Go 1.22) : le mux répond 405 tout seul si
	// la méthode ne correspond à aucune route du chemin, et 404 si le chemin
	// est inconnu. "{id}" capture un segment, lu par r.PathValue("id").
	mux.HandleFunc("GET /taches", a.liste)
	mux.HandleFunc("POST /taches", a.cree)
	mux.HandleFunc("GET /taches/{id}", a.lit)
	mux.HandleFunc("PUT /taches/{id}", a.remplace)
	mux.HandleFunc("DELETE /taches/{id}", a.supprime)
	return journalise(mux)
}

// liste : GET /taches -> 200 et la liste JSON (vide : []).
func (a *api) liste(w http.ResponseWriter, r *http.Request) {
	repondJSON(w, http.StatusOK, a.magasin.Liste())
}

// cree : POST /taches avec {"titre": "..."} -> 201 et la tâche créée.
// 400 si le JSON est invalide ou le titre vide.
func (a *api) cree(w http.ResponseWriter, r *http.Request) {
	t, err := decodeTache(r)
	if err != nil {
		repondErreur(w, http.StatusBadRequest, err.Error())
		return
	}
	creee := a.magasin.Ajoute(t.Titre)
	repondJSON(w, http.StatusCreated, creee)
}

// lit : GET /taches/{id} -> 200 et la tâche, 400 si id n'est pas un
// entier, 404 si elle n'existe pas.
func (a *api) lit(w http.ResponseWriter, r *http.Request) {
	id, ok := idDuChemin(w, r)
	if !ok {
		return // idDuChemin a déjà répondu 400
	}
	t, trouvee := a.magasin.Lit(id)
	if !trouvee {
		repondErreur(w, http.StatusNotFound, "tâche introuvable")
		return
	}
	repondJSON(w, http.StatusOK, t)
}

// remplace : PUT /taches/{id} avec {"titre": "...", "faite": bool} -> 200
// et la tâche mise à jour ; 400 si le corps est invalide, 404 si inconnue.
func (a *api) remplace(w http.ResponseWriter, r *http.Request) {
	id, ok := idDuChemin(w, r)
	if !ok {
		return
	}
	t, err := decodeTache(r)
	if err != nil {
		repondErreur(w, http.StatusBadRequest, err.Error())
		return
	}
	maj, trouvee := a.magasin.Remplace(id, t)
	if !trouvee {
		repondErreur(w, http.StatusNotFound, "tâche introuvable")
		return
	}
	repondJSON(w, http.StatusOK, maj)
}

// supprime : DELETE /taches/{id} -> 204 sans corps, ou 404.
func (a *api) supprime(w http.ResponseWriter, r *http.Request) {
	id, ok := idDuChemin(w, r)
	if !ok {
		return
	}
	if !a.magasin.Supprime(id) {
		repondErreur(w, http.StatusNotFound, "tâche introuvable")
		return
	}
	// 204 No Content : le statut suffit, pas de corps, pas de Content-Type.
	w.WriteHeader(http.StatusNoContent)
}

// idDuChemin lit {id} dans l'URL et le convertit en entier. En cas d'échec
// il répond 400 lui-même et renvoie false : le handler n'a plus qu'à return.
func idDuChemin(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		repondErreur(w, http.StatusBadRequest, "id invalide : "+r.PathValue("id"))
		return 0, false
	}
	return id, true
}

// decodeTache lit le corps JSON de la requête dans une Tache et valide le
// titre. Le décodeur est limité à 1 Mo (http.MaxBytesReader) : un client
// ne doit pas pouvoir remplir la mémoire du serveur avec un corps énorme.
func decodeTache(r *http.Request) (Tache, error) {
	var t Tache
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // {"title": ...} (faute de frappe) devient une erreur
	if err := dec.Decode(&t); err != nil {
		return Tache{}, errors.New("JSON invalide : " + err.Error())
	}
	if strings.TrimSpace(t.Titre) == "" {
		return Tache{}, errors.New("le titre est obligatoire")
	}
	return t, nil
}

// repondJSON écrit l'en-tête Content-Type, le code de statut, puis v encodé
// en JSON. L'ordre compte : les en-têtes doivent être posés AVANT
// WriteHeader, et WriteHeader AVANT le corps ; après le premier octet écrit,
// le statut est parti sur le réseau et ne peut plus changer.
func repondJSON(w http.ResponseWriter, statut int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statut)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Trop tard pour changer le statut : on ne peut que journaliser.
		log.Println("encodage JSON :", err)
	}
}

// repondErreur écrit {"erreur": "message"} avec le statut donné. Un client
// JSON ne reçoit jamais de texte brut de nos handlers.
func repondErreur(w http.ResponseWriter, statut int, message string) {
	repondJSON(w, statut, map[string]string{"erreur": message})
}

// journalise est un middleware : une fonction qui prend un handler et en
// renvoie un autre qui fait quelque chose avant/après. http.HandlerFunc
// convertit la closure en http.Handler (chapitre 7 : une fonction qui
// satisfait une interface).
func journalise(suivant http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		debut := time.Now()
		// Sans enveloppe on ne connaîtrait pas le statut : ResponseWriter
		// ne l'expose pas. Le statut par défaut est 200 (celui que net/http
		// envoie si le handler écrit sans appeler WriteHeader).
		e := &enregistreur{ResponseWriter: w, statut: http.StatusOK}
		suivant.ServeHTTP(e, r)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, e.statut, time.Since(debut).Round(time.Microsecond))
	})
}

// enregistreur enveloppe un http.ResponseWriter par composition (le champ
// anonyme promeut Header, Write et WriteHeader) et redéfinit WriteHeader
// pour retenir le statut. C'est le patron « décorateur » à la Go.
type enregistreur struct {
	http.ResponseWriter
	statut int
}

// WriteHeader retient le statut puis le transmet au vrai ResponseWriter.
func (e *enregistreur) WriteHeader(statut int) {
	e.statut = statut
	e.ResponseWriter.WriteHeader(statut)
}
