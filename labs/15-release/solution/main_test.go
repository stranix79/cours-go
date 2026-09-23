// Labo 15 : tests du routeur et de run (démarrage, arrêt propre).
// Lancé par : go test .   (ton code)   ou   go test ./solution/   (la solution)
package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestDecouperPoints(t *testing.T) {
	cas := []struct {
		nom, entree string
		attendu     []string
	}{
		{"un seul", "/", []string{"/"}},
		{"plusieurs avec espaces", "/, /var ,/data", []string{"/", "/var", "/data"}},
		{"vides ignorés", ",/,,", []string{"/"}},
		{"chaîne vide", "", nil},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := decouperPoints(c.entree); !reflect.DeepEqual(got, c.attendu) {
				t.Errorf("decouperPoints(%q) = %#v, attendu %#v", c.entree, got, c.attendu)
			}
		})
	}
}

func TestRouteur(t *testing.T) {
	h := nouveauRouteur(sondeTest, []string{"/"})
	cas := []struct {
		nom, methode, chemin string
		statut               int
	}{
		{"metrics", http.MethodGet, "/metrics", http.StatusOK},
		{"healthz", http.MethodGet, "/healthz", http.StatusOK},
		{"route inconnue", http.MethodGet, "/", http.StatusNotFound},
		{"mauvaise méthode", http.MethodPost, "/metrics", http.StatusMethodNotAllowed},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(c.methode, c.chemin, nil))
			if rec.Code != c.statut {
				t.Errorf("%s %s : statut %d, attendu %d", c.methode, c.chemin, rec.Code, c.statut)
			}
		})
	}
}

// portLibre demande au système un port TCP libre et le rend aussitôt.
func portLibre(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().String()
}

func TestRunArretPropre(t *testing.T) {
	addr := portLibre(t)
	ctx, cancel := context.WithCancel(context.Background())
	fini := make(chan error, 1)
	go func() { fini <- run(ctx, addr, sondeTest, []string{"/"}) }()

	// On attend que le serveur réponde (au plus 2 s), puis on lit /metrics.
	client := &http.Client{Timeout: time.Second}
	var resp *http.Response
	var err error
	for debut := time.Now(); time.Since(debut) < 2*time.Second; {
		resp, err = client.Get("http://" + addr + "/healthz")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		cancel()
		t.Fatalf("le serveur ne répond pas sur %s : %v (run : %v)", addr, err, <-fini)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz : statut %d", resp.StatusCode)
	}

	// Annuler le contexte doit faire revenir run sans erreur, rapidement.
	cancel()
	select {
	case err := <-fini:
		if err != nil {
			t.Errorf("run a renvoyé une erreur après l'arrêt : %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("run ne s'est pas arrêté après l'annulation du contexte")
	}

	// Après l'arrêt, le port est fermé.
	if _, err := client.Get("http://" + addr + "/healthz"); err == nil {
		t.Error("le serveur répond encore après l'arrêt")
	}
}

func TestRunPortOccupe(t *testing.T) {
	// On occupe un port, puis on demande à run de l'utiliser : il doit
	// échouer tout de suite, pas rester bloqué.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = run(ctx, ln.Addr().String(), sondeTest, []string{"/"})
	if err == nil {
		t.Fatal("run doit échouer quand le port est déjà pris")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("run est resté bloqué au lieu de signaler l'erreur de bind")
	}
}
