// Labo 13 : les tests de l'API, sans réseau (httptest.NewRecorder) sauf le
// dernier, qui lance run() sur un port libre et vérifie l'arrêt propre.
// Lancé par : go test .   (ton code)   ou   go test ./solution/   (la solution)
package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// appelle envoie une requête au routeur sans passer par le réseau et renvoie
// le statut, le corps décodé en JSON (nil si vide) et les en-têtes.
func appelle(t *testing.T, h http.Handler, methode, chemin, corps string) (int, map[string]any, http.Header) {
	t.Helper()
	var lecteur io.Reader
	if corps != "" {
		lecteur = strings.NewReader(corps)
	}
	req := httptest.NewRequest(methode, chemin, lecteur)
	if corps != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var decode map[string]any
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &decode); err != nil {
			// une liste n'est pas une map : on la range sous la clé "liste" ;
			// un corps qui n'est pas du JSON (le "404 page not found" du
			// ServeMux) donne une map nil, que les tests savent lire.
			var liste []any
			if err2 := json.Unmarshal(rec.Body.Bytes(), &liste); err2 == nil {
				decode = map[string]any{"liste": liste}
			}
		}
	}
	return rec.Code, decode, rec.Header()
}

func TestMagasin(t *testing.T) {
	m := NouveauMagasin()
	if l := m.Liste(); l == nil || len(l) != 0 {
		t.Fatalf("Liste sur magasin vide = %v, attendu une slice vide non nil", l)
	}
	a := m.Ajoute("sauvegarder pg-01")
	b := m.Ajoute("renouveler le certificat")
	if a.ID != 1 || b.ID != 2 {
		t.Fatalf("IDs attribués : %d et %d, attendu 1 et 2", a.ID, b.ID)
	}
	if a.Faite {
		t.Fatal("une tâche créée doit être non faite")
	}
	if lu, ok := m.Lit(2); !ok || lu.Titre != "renouveler le certificat" {
		t.Fatalf("Lit(2) = %+v, %v", lu, ok)
	}
	if _, ok := m.Lit(42); ok {
		t.Fatal("Lit(42) doit renvoyer false")
	}
	maj, ok := m.Remplace(1, Tache{ID: 999, Titre: "sauvegarder pg-01 (fait)", Faite: true})
	if !ok || maj.ID != 1 || !maj.Faite || maj.Titre != "sauvegarder pg-01 (fait)" {
		t.Fatalf("Remplace(1) = %+v, %v : l'ID doit rester 1", maj, ok)
	}
	if _, ok := m.Remplace(42, Tache{}); ok {
		t.Fatal("Remplace(42) doit renvoyer false")
	}
	if !m.Supprime(1) || m.Supprime(1) {
		t.Fatal("Supprime(1) doit renvoyer true puis false")
	}
	if l := m.Liste(); len(l) != 1 || l[0].ID != 2 {
		t.Fatalf("après suppression, Liste = %+v", l)
	}
}

func TestMagasinConcurrent(t *testing.T) {
	m := NouveauMagasin()
	fini := make(chan bool)
	for i := 0; i < 50; i++ {
		go func() {
			m.Ajoute("tâche parallèle")
			m.Liste()
			fini <- true
		}()
	}
	for i := 0; i < 50; i++ {
		<-fini
	}
	if n := len(m.Liste()); n != 50 {
		t.Fatalf("50 ajouts en parallèle, %d tâches", n)
	}
}

func TestAPI(t *testing.T) {
	log.SetOutput(io.Discard) // le middleware journalise : on ne veut pas polluer la sortie des tests
	h := nouveauRouteur(NouveauMagasin())

	t.Run("liste vide", func(t *testing.T) {
		code, corps, entetes := appelle(t, h, "GET", "/taches", "")
		if code != 200 {
			t.Fatalf("statut %d, attendu 200", code)
		}
		if ct := entetes.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Fatalf("Content-Type = %q, attendu application/json", ct)
		}
		if l, ok := corps["liste"].([]any); !ok || len(l) != 0 {
			t.Fatalf("corps = %v, attendu []", corps)
		}
	})

	t.Run("création 201", func(t *testing.T) {
		code, corps, _ := appelle(t, h, "POST", "/taches", `{"titre":"sauvegarder pg-01"}`)
		if code != 201 {
			t.Fatalf("statut %d, attendu 201", code)
		}
		if corps["id"] != float64(1) || corps["titre"] != "sauvegarder pg-01" || corps["faite"] != false {
			t.Fatalf("corps = %v", corps)
		}
	})

	t.Run("création sans titre 400", func(t *testing.T) {
		code, corps, _ := appelle(t, h, "POST", "/taches", `{"titre":""}`)
		if code != 400 {
			t.Fatalf("statut %d, attendu 400", code)
		}
		if _, ok := corps["erreur"]; !ok {
			t.Fatalf("corps = %v, attendu une clé \"erreur\"", corps)
		}
	})

	t.Run("création JSON invalide 400", func(t *testing.T) {
		code, corps, _ := appelle(t, h, "POST", "/taches", `{"titre":`)
		if code != 400 || corps["erreur"] == nil {
			t.Fatalf("statut %d, corps %v", code, corps)
		}
	})

	t.Run("lecture 200", func(t *testing.T) {
		code, corps, _ := appelle(t, h, "GET", "/taches/1", "")
		if code != 200 || corps["titre"] != "sauvegarder pg-01" {
			t.Fatalf("statut %d, corps %v", code, corps)
		}
	})

	t.Run("lecture inconnue 404", func(t *testing.T) {
		code, corps, _ := appelle(t, h, "GET", "/taches/42", "")
		if code != 404 || corps["erreur"] == nil {
			t.Fatalf("statut %d, corps %v", code, corps)
		}
	})

	t.Run("id non numérique 400", func(t *testing.T) {
		code, corps, _ := appelle(t, h, "GET", "/taches/abc", "")
		if code != 400 || corps["erreur"] == nil {
			t.Fatalf("statut %d, corps %v", code, corps)
		}
	})

	t.Run("remplacement 200", func(t *testing.T) {
		code, corps, _ := appelle(t, h, "PUT", "/taches/1", `{"titre":"sauvegarder pg-01","faite":true}`)
		if code != 200 || corps["faite"] != true || corps["id"] != float64(1) {
			t.Fatalf("statut %d, corps %v", code, corps)
		}
	})

	t.Run("remplacement inconnu 404", func(t *testing.T) {
		code, _, _ := appelle(t, h, "PUT", "/taches/42", `{"titre":"x","faite":true}`)
		if code != 404 {
			t.Fatalf("statut %d, attendu 404", code)
		}
	})

	t.Run("liste après modification", func(t *testing.T) {
		appelle(t, h, "POST", "/taches", `{"titre":"renouveler le certificat"}`)
		code, corps, _ := appelle(t, h, "GET", "/taches", "")
		l, ok := corps["liste"].([]any)
		if code != 200 || !ok || len(l) != 2 {
			t.Fatalf("statut %d, corps %v : attendu 2 tâches", code, corps)
		}
		premiere, _ := l[0].(map[string]any)
		if premiere["id"] != float64(1) || premiere["faite"] != true {
			t.Fatalf("première tâche = %v", premiere)
		}
	})

	t.Run("suppression 204 puis 404", func(t *testing.T) {
		code, corps, _ := appelle(t, h, "DELETE", "/taches/1", "")
		if code != 204 || corps != nil {
			t.Fatalf("statut %d, corps %v : attendu 204 sans corps", code, corps)
		}
		code, _, _ = appelle(t, h, "DELETE", "/taches/1", "")
		if code != 404 {
			t.Fatalf("seconde suppression : statut %d, attendu 404", code)
		}
	})

	t.Run("méthode non permise 405", func(t *testing.T) {
		code, _, _ := appelle(t, h, "PATCH", "/taches", "")
		if code != 405 {
			t.Fatalf("statut %d, attendu 405", code)
		}
	})

	t.Run("route inconnue 404", func(t *testing.T) {
		code, _, _ := appelle(t, h, "GET", "/inconnu", "")
		if code != 404 {
			t.Fatalf("statut %d, attendu 404", code)
		}
	})
}

// TestRunArretPropre lance run() sur un port libre, vérifie qu'il répond,
// annule le context et vérifie que run rend la main sans erreur en moins
// d'une seconde.
func TestRunArretPropre(t *testing.T) {
	log.SetOutput(io.Discard)
	// On demande au système un port libre puis on le libère pour run.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()

	ctx, annule := context.WithCancel(context.Background())
	fini := make(chan error, 1)
	go func() { fini <- run(ctx, addr) }()

	// On attend que le serveur réponde (au plus 2 s).
	client := &http.Client{Timeout: 200 * time.Millisecond}
	var resp *http.Response
	for i := 0; i < 20; i++ {
		resp, err = client.Get("http://" + addr + "/taches")
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		annule()
		t.Fatalf("le serveur ne répond pas sur %s : %v (run a renvoyé : %v)", addr, err, <-fini)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("GET /taches : statut %d", resp.StatusCode)
	}

	annule()
	select {
	case err := <-fini:
		if err != nil {
			t.Fatalf("run a renvoyé une erreur à l'arrêt : %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("run n'a pas rendu la main en 1 s après l'annulation")
	}
}
