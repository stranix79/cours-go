// Paquet sonde : vérifier qu'une cible répond, et garder le résultat.
// Ce fichier : Resultat, l'interface Sondeur, l'implémentation réseau.
// Voir aussi etat.go (mémoire des résultats) et moteur.go (worker pool, boucle).
package sonde

import (
	"context"
	"errors"
	"net/http"
	"time"

	"cours-go/labs/16-projet/internal/config"
)

// Resultat est ce qu'une sonde produit.
type Resultat struct {
	Cible   string        // nom de la cible
	Type    string        // "http" ou "tcp"
	Up      bool          // la cible a répondu correctement
	Latence time.Duration // durée de la vérification
	Erreur  string        // vide si Up ; sinon la cause
	Quand   time.Time     // début de la vérification
}

// Sondeur est ce dont le Moteur a besoin. En production : SondeurReseau ;
// dans les tests : un faux qui compte les appels.
type Sondeur interface {
	Sonder(ctx context.Context, c config.Cible) Resultat
}

// SondeurReseau est l'implémentation réelle. Le client HTTP est partagé et
// porte le timeout.
type SondeurReseau struct {
	Client  *http.Client
	Timeout time.Duration
}

// NouveauSondeur construit un SondeurReseau dont le client est borné par
// timeout (http.Client{Timeout: timeout}).
func NouveauSondeur(timeout time.Duration) *SondeurReseau {
	// TODO 4 : renvoyer un SondeurReseau avec un Client non nil.
	return &SondeurReseau{Timeout: timeout}
}

// Sonder aiguille selon c.Type vers SonderHTTP ou SonderTCP, mesure la
// latence, et remplit le Resultat (Up = pas d'erreur, Erreur = err.Error()).
// Un type inconnu est un résultat down avec une erreur, jamais un panic.
func (s *SondeurReseau) Sonder(ctx context.Context, c config.Cible) Resultat {
	// TODO 5 : debut := time.Now() ; switch c.Type ; construire le Resultat.
	return Resultat{Cible: c.Nom, Type: c.Type}
}

// SonderHTTP fait un GET (http.NewRequestWithContext) et renvoie nil si le
// statut est < 400, sinon une erreur "statut HTTP NNN". Ferme le corps
// (defer resp.Body.Close()) et vide-le (io.Copy vers io.Discard) pour que la
// connexion soit réutilisable.
func SonderHTTP(ctx context.Context, client *http.Client, url string) error {
	// TODO 6
	_, _, _ = ctx, client, url
	return errors.New("TODO")
}

// SonderTCP ouvre une connexion avec net.Dialer{Timeout: timeout}.DialContext
// et la referme aussitôt. nil si le port accepte, l'erreur du Dial sinon.
func SonderTCP(ctx context.Context, adresse string, timeout time.Duration) error {
	// TODO 7
	_, _, _ = ctx, adresse, timeout
	return errors.New("TODO")
}
