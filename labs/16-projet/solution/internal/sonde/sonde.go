// Paquet sonde, solution : vérifier qu'une cible répond, et garder le résultat.
//
// Ce fichier : le type Resultat, l'interface Sondeur, et l'implémentation
// réseau (SonderHTTP, SonderTCP). Les autres fichiers du paquet : etat.go
// (la mémoire des derniers résultats) et moteur.go (le worker pool et la
// boucle périodique).
// Qui l'appelle : le Moteur appelle Sondeur.Sonder ; main construit le
// SondeurReseau ; les tests appellent SonderHTTP et SonderTCP directement.
package sonde

import (
	"context"  // annulation et délais propagés jusqu'à la connexion réseau
	"fmt"      // messages d'erreur
	"io"       // io.Copy / io.Discard pour vider le corps HTTP
	"net"      // net.Dialer pour la sonde TCP
	"net/http" // client HTTP
	"time"     // latences et timeouts

	"cours-go/labs/16-projet/solution/internal/config"
)

// Resultat est ce qu'une sonde produit : la cible, up ou down, en combien de
// temps, avec quelle erreur, et quand. C'est la valeur stockée dans Etat et
// exposée (transformée) par l'API.
type Resultat struct {
	Cible   string        // le nom de la cible (clé de l'état, label Prometheus)
	Type    string        // "http" ou "tcp"
	Up      bool          // la cible a répondu correctement
	Latence time.Duration // durée de la vérification
	Erreur  string        // vide si Up ; sinon la cause, en clair
	Quand   time.Time     // début de la vérification
}

// Sondeur est ce dont le Moteur a besoin : quelque chose qui sonde une cible.
// Une interface d'une méthode (chapitre 7) : en production c'est SondeurReseau,
// dans les tests un faux qui répond instantanément et compte les appels.
type Sondeur interface {
	Sonder(ctx context.Context, c config.Cible) Resultat
}

// SondeurReseau est l'implémentation réelle. Le client HTTP est partagé entre
// toutes les sondes (il réutilise les connexions) et porte le timeout global :
// un http.Client sans Timeout attend indéfiniment un serveur muet.
type SondeurReseau struct {
	Client  *http.Client
	Timeout time.Duration
}

// NouveauSondeur construit un SondeurReseau avec un client borné par timeout.
func NouveauSondeur(timeout time.Duration) *SondeurReseau {
	return &SondeurReseau{
		Client:  &http.Client{Timeout: timeout},
		Timeout: timeout,
	}
}

// Sonder aiguille selon le type, mesure le temps, et transforme l'erreur en
// Resultat. Une cible injoignable est un résultat down, pas une erreur du
// programme : c'est la règle de tout superviseur.
func (s *SondeurReseau) Sonder(ctx context.Context, c config.Cible) Resultat {
	debut := time.Now()
	var err error
	switch c.Type {
	case config.TypeHTTP:
		err = SonderHTTP(ctx, s.Client, c.URL)
	case config.TypeTCP:
		err = SonderTCP(ctx, c.Adresse, s.Timeout)
	default:
		// La config a été validée, on ne devrait jamais passer ici ; mais un
		// switch sans default qui laisse err à nil dirait « up » à tort.
		err = fmt.Errorf("type %q inconnu", c.Type)
	}
	r := Resultat{
		Cible:   c.Nom,
		Type:    c.Type,
		Up:      err == nil,
		Latence: time.Since(debut),
		Quand:   debut,
	}
	if err != nil {
		r.Erreur = err.Error()
	}
	return r
}

// SonderHTTP fait un GET et considère la cible up si le statut est < 400.
// Le contexte est attaché à la requête : si le programme s'arrête pendant une
// sonde, la connexion est coupée immédiatement.
func SonderHTTP(ctx context.Context, client *http.Client, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "sondes/1") // poli, et visible dans les logs du serveur sondé
	resp, err := client.Do(req)
	if err != nil {
		return err // refus de connexion, DNS, timeout : tout arrive ici
	}
	defer resp.Body.Close() // sinon la connexion n'est jamais rendue au pool
	// Vider le corps (borné à 64 Kio) permet de réutiliser la connexion
	// keep-alive ; on ignore le contenu, seul le statut compte.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("statut HTTP %d", resp.StatusCode)
	}
	return nil
}

// SonderTCP ouvre une connexion et la referme aussitôt : si le port accepte,
// le service écoute. C'est ce que fait « nc -z ». Le Dialer porte le timeout,
// le contexte porte l'annulation ; les deux s'appliquent.
func SonderTCP(ctx context.Context, adresse string, timeout time.Duration) error {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", adresse)
	if err != nil {
		return err
	}
	return conn.Close()
}
