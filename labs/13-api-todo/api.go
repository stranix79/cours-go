// Labo 13 : les handlers HTTP, le routeur et le middleware de journalisation.
// Utilisé par : main.go (run) et api_test.go.
package main

import (
	"log"
	"net/http"
)

// nouveauRouteur construit le ServeMux avec les cinq routes de l'API, chacune
// avec sa méthode et son paramètre de chemin (Go 1.22+), et l'enveloppe dans
// le middleware de journalisation :
//
//	GET    /taches        liste           200
//	POST   /taches        crée            201 + la tâche créée
//	GET    /taches/{id}   lit             200 ou 404
//	PUT    /taches/{id}   remplace        200 ou 400/404
//	DELETE /taches/{id}   supprime        204 ou 404
func nouveauRouteur(m *Magasin) http.Handler {
	mux := http.NewServeMux()
	// TODO 7 : enregistre les routes avec mux.HandleFunc("GET /taches", ...).
	// Chaque handler a besoin du magasin : écris des closures, ou des méthodes
	// sur un type qui porte le magasin.
	return journalise(mux)
}

// repondJSON écrit l'en-tête Content-Type, le code de statut, puis v encodé
// en JSON. Le statut doit être écrit AVANT le corps (WriteHeader puis Encode).
func repondJSON(w http.ResponseWriter, statut int, v any) {
	// TODO 8
}

// repondErreur écrit une erreur au format {"erreur": "message"} avec le
// statut donné. C'est le SEUL format d'erreur de l'API : un client JSON ne
// doit jamais recevoir du texte brut.
func repondErreur(w http.ResponseWriter, statut int, message string) {
	// TODO 9 : réutilise repondJSON avec une map[string]string.
}

// journalise est un middleware : il reçoit le handler suivant et renvoie un
// handler qui journalise "MÉTHODE chemin -> statut durée" après chaque
// requête. Pour connaître le statut, il faut envelopper le ResponseWriter
// (voir le type enregistreur ci-dessous).
func journalise(suivant http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO 10 : note l'heure, enveloppe w dans un enregistreur, appelle
		// suivant.ServeHTTP, puis log.Printf le résultat.
		log.Printf("%s %s", r.Method, r.URL.Path)
		suivant.ServeHTTP(w, r)
	})
}

// enregistreur enveloppe un http.ResponseWriter pour retenir le code de
// statut écrit par le handler (le ResponseWriter d'origine ne le rend pas).
type enregistreur struct {
	http.ResponseWriter
	statut int
}

// WriteHeader retient le statut puis le transmet au vrai ResponseWriter.
func (e *enregistreur) WriteHeader(statut int) {
	// TODO 11
	e.ResponseWriter.WriteHeader(statut)
}
