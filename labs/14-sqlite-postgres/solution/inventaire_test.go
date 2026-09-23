// Labo 14 : les tests du dépôt Inventaire.
// Par défaut ils tournent sur SQLite en mémoire (rien à installer, quelques
// millisecondes). Avec INVENTAIRE_TEST_DSN=postgres://... ils tournent, sans
// changement, sur un PostgreSQL : la table est vidée avant chaque test.
// Lancés par : go test .   (ton code)   go test ./solution/   (la solution)
package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ouvrirTest ouvre un dépôt propre et programme sa fermeture.
func ouvrirTest(t *testing.T) *SQLStore {
	t.Helper()
	dsn := os.Getenv("INVENTAIRE_TEST_DSN")
	if dsn == "" {
		dsn = ":memory:"
	}
	s, err := Ouvrir(context.Background(), dsn)
	if err != nil {
		t.Fatalf("Ouvrir(%q) : %v", dsn, err)
	}
	t.Cleanup(func() { s.Close() })
	if dsn != ":memory:" {
		if _, err := s.db.ExecContext(context.Background(), `DELETE FROM hotes`); err != nil {
			t.Fatalf("vider la table : %v", err)
		}
	}
	return s
}

var (
	web01   = Hote{Nom: "web-01", IP: "10.0.0.11", Role: "web"}
	web02   = Hote{Nom: "web-02", IP: "10.0.0.12", Role: "web"}
	db01    = Hote{Nom: "db-01", IP: "10.0.0.21", Role: "db"}
	dateVue = time.Date(2026, 9, 23, 10, 30, 0, 0, time.UTC)
)

func TestMigrationsAppliquees(t *testing.T) {
	s := ouvrirTest(t)
	v, err := Version(context.Background(), s.db)
	if err != nil {
		t.Fatalf("Version : %v", err)
	}
	if v != 2 {
		t.Errorf("version du schéma = %d, attendu 2 (deux fichiers dans migrations/)", v)
	}
}

func TestMigrationsIdempotentes(t *testing.T) {
	if os.Getenv("INVENTAIRE_TEST_DSN") != "" {
		t.Skip("ce test ouvre un fichier SQLite ; sans objet sur PostgreSQL")
	}
	chemin := filepath.Join(t.TempDir(), "inv.db")
	ctx := context.Background()
	s1, err := Ouvrir(ctx, chemin)
	if err != nil {
		t.Fatalf("première ouverture : %v", err)
	}
	if err := s1.Ajouter(ctx, web01); err != nil {
		t.Fatalf("Ajouter : %v", err)
	}
	s1.Close()

	// Deuxième ouverture du même fichier : les migrations ne doivent pas être
	// rejouées (CREATE TABLE hotes échouerait), et les données sont toujours là.
	s2, err := Ouvrir(ctx, chemin)
	if err != nil {
		t.Fatalf("deuxième ouverture : %v", err)
	}
	defer s2.Close()
	v, _ := Version(ctx, s2.db)
	if v != 2 {
		t.Errorf("version après réouverture = %d, attendu 2", v)
	}
	if _, err := s2.Trouver(ctx, "web-01"); err != nil {
		t.Errorf("web-01 a disparu après réouverture : %v", err)
	}
}

func TestAjouterEtTrouver(t *testing.T) {
	s := ouvrirTest(t)
	ctx := context.Background()
	if err := s.Ajouter(ctx, web01); err != nil {
		t.Fatalf("Ajouter : %v", err)
	}
	h, err := s.Trouver(ctx, "web-01")
	if err != nil {
		t.Fatalf("Trouver : %v", err)
	}
	if h.IP != "10.0.0.11" || h.Role != "web" {
		t.Errorf("Trouver = %+v, attendu %+v", h, web01)
	}
	if h.DernierVu != nil {
		t.Errorf("DernierVu = %v, attendu nil (NULL) pour un hôte jamais vu", *h.DernierVu)
	}
}

func TestAjouterDoublon(t *testing.T) {
	s := ouvrirTest(t)
	ctx := context.Background()
	if err := s.Ajouter(ctx, web01); err != nil {
		t.Fatalf("Ajouter : %v", err)
	}
	err := s.Ajouter(ctx, Hote{Nom: "web-01", IP: "10.9.9.9", Role: "autre"})
	if !errors.Is(err, ErrExiste) {
		t.Fatalf("doublon : err = %v, attendu ErrExiste", err)
	}
	h, _ := s.Trouver(ctx, "web-01")
	if h.IP != "10.0.0.11" {
		t.Errorf("le doublon a écrasé l'original : IP = %s", h.IP)
	}
}

func TestTrouverIntrouvable(t *testing.T) {
	s := ouvrirTest(t)
	_, err := s.Trouver(context.Background(), "fantome")
	if !errors.Is(err, ErrIntrouvable) {
		t.Errorf("err = %v, attendu ErrIntrouvable", err)
	}
}

func TestLister(t *testing.T) {
	s := ouvrirTest(t)
	ctx := context.Background()
	for _, h := range []Hote{web02, db01, web01} { // volontairement dans le désordre
		if err := s.Ajouter(ctx, h); err != nil {
			t.Fatalf("Ajouter %s : %v", h.Nom, err)
		}
	}
	cas := []struct {
		nom     string
		role    string
		attendu []string
	}{
		{"tous, triés par nom", "", []string{"db-01", "web-01", "web-02"}},
		{"rôle web", "web", []string{"web-01", "web-02"}},
		{"rôle db", "db", []string{"db-01"}},
		{"rôle inconnu : liste vide", "gpu", nil},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			hotes, err := s.Lister(ctx, c.role)
			if err != nil {
				t.Fatalf("Lister(%q) : %v", c.role, err)
			}
			if len(hotes) != len(c.attendu) {
				t.Fatalf("Lister(%q) = %d hôtes, attendu %d", c.role, len(hotes), len(c.attendu))
			}
			for i, h := range hotes {
				if h.Nom != c.attendu[i] {
					t.Errorf("position %d : %s, attendu %s", i, h.Nom, c.attendu[i])
				}
			}
		})
	}
}

func TestMarquerVu(t *testing.T) {
	s := ouvrirTest(t)
	ctx := context.Background()
	if err := s.Ajouter(ctx, db01); err != nil {
		t.Fatalf("Ajouter : %v", err)
	}
	// Une heure locale avec des nanosecondes : le dépôt doit la ramener en UTC
	// à la microseconde, et la relire égale.
	locale := dateVue.In(time.FixedZone("CEST", 2*3600)).Add(700 * time.Nanosecond)
	if err := s.MarquerVu(ctx, "db-01", locale); err != nil {
		t.Fatalf("MarquerVu : %v", err)
	}
	h, err := s.Trouver(ctx, "db-01")
	if err != nil {
		t.Fatalf("Trouver : %v", err)
	}
	if h.DernierVu == nil {
		t.Fatal("DernierVu est resté nil après MarquerVu")
	}
	if !h.DernierVu.Equal(dateVue) {
		t.Errorf("DernierVu = %v, attendu %v", *h.DernierVu, dateVue)
	}
	if h.DernierVu.Location() != time.UTC {
		t.Errorf("DernierVu relu en %v, attendu UTC", h.DernierVu.Location())
	}
	if err := s.MarquerVu(ctx, "fantome", dateVue); !errors.Is(err, ErrIntrouvable) {
		t.Errorf("MarquerVu(fantome) : err = %v, attendu ErrIntrouvable", err)
	}
}

func TestSupprimer(t *testing.T) {
	s := ouvrirTest(t)
	ctx := context.Background()
	if err := s.Ajouter(ctx, web01); err != nil {
		t.Fatalf("Ajouter : %v", err)
	}
	if err := s.Supprimer(ctx, "web-01"); err != nil {
		t.Fatalf("Supprimer : %v", err)
	}
	if _, err := s.Trouver(ctx, "web-01"); !errors.Is(err, ErrIntrouvable) {
		t.Errorf("après Supprimer, Trouver : err = %v, attendu ErrIntrouvable", err)
	}
	if err := s.Supprimer(ctx, "web-01"); !errors.Is(err, ErrIntrouvable) {
		t.Errorf("Supprimer deux fois : err = %v, attendu ErrIntrouvable", err)
	}
}

func TestImporter(t *testing.T) {
	t.Run("tout passe", func(t *testing.T) {
		s := ouvrirTest(t)
		ctx := context.Background()
		vu := dateVue
		n, err := s.Importer(ctx, []Hote{web01, web02, {Nom: "db-01", IP: "10.0.0.21", Role: "db", DernierVu: &vu}})
		if err != nil {
			t.Fatalf("Importer : %v", err)
		}
		if n != 3 {
			t.Errorf("n = %d, attendu 3", n)
		}
		h, err := s.Trouver(ctx, "db-01")
		if err != nil {
			t.Fatalf("Trouver : %v", err)
		}
		if h.DernierVu == nil || !h.DernierVu.Equal(dateVue) {
			t.Errorf("DernierVu importé = %v, attendu %v", h.DernierVu, dateVue)
		}
	})

	t.Run("tout ou rien", func(t *testing.T) {
		s := ouvrirTest(t)
		ctx := context.Background()
		if err := s.Ajouter(ctx, db01); err != nil {
			t.Fatalf("Ajouter : %v", err)
		}
		// db-01 existe déjà, en troisième position : les deux premiers doivent
		// être annulés par le Rollback.
		n, err := s.Importer(ctx, []Hote{web01, web02, db01})
		if !errors.Is(err, ErrExiste) {
			t.Fatalf("Importer avec doublon : err = %v, attendu ErrExiste", err)
		}
		if n != 0 {
			t.Errorf("n = %d, attendu 0", n)
		}
		hotes, _ := s.Lister(ctx, "")
		if len(hotes) != 1 || hotes[0].Nom != "db-01" {
			t.Errorf("après l'échec, la table contient %v, attendu seulement db-01 (transaction non annulée ?)", hotes)
		}
	})

	t.Run("liste vide", func(t *testing.T) {
		s := ouvrirTest(t)
		n, err := s.Importer(context.Background(), nil)
		if err != nil || n != 0 {
			t.Errorf("Importer(nil) = %d, %v ; attendu 0, nil", n, err)
		}
	})
}

func TestExecuter(t *testing.T) {
	s := ouvrirTest(t)
	ctx := context.Background()
	lancer := func(args ...string) (string, error) {
		var buf bytes.Buffer
		err := executer(ctx, s, args, &buf, dateVue)
		return buf.String(), err
	}

	if _, err := lancer("ajouter", "web-01", "10.0.0.11", "web"); err != nil {
		t.Fatalf("ajouter : %v", err)
	}
	if _, err := lancer("ajouter", "db-01", "10.0.0.21", "db"); err != nil {
		t.Fatalf("ajouter : %v", err)
	}
	if _, err := lancer("vu", "db-01"); err != nil {
		t.Fatalf("vu : %v", err)
	}

	sortie, err := lancer("lister")
	if err != nil {
		t.Fatalf("lister : %v", err)
	}
	attendu := "NOM        IP              ROLE     DERNIER VU\n" +
		"db-01      10.0.0.21       db       2026-09-23 10:30:00\n" +
		"web-01     10.0.0.11       web      jamais\n"
	if sortie != attendu {
		t.Errorf("lister :\n%s\nattendu :\n%s", sortie, attendu)
	}

	if _, err := lancer("supprimer", "fantome"); !errors.Is(err, ErrIntrouvable) {
		t.Errorf("supprimer fantome : err = %v, attendu ErrIntrouvable", err)
	}
	if _, err := lancer("teleporter"); err == nil {
		t.Error("commande inconnue : erreur attendue")
	}

	// importer depuis un fichier JSON écrit dans un dossier temporaire
	chemin := filepath.Join(t.TempDir(), "hotes.json")
	contenu := `[{"nom":"web-02","ip":"10.0.0.12","role":"web"},{"nom":"bastion","ip":"10.0.0.2","role":"admin"}]`
	if err := os.WriteFile(chemin, []byte(contenu), 0o644); err != nil {
		t.Fatal(err)
	}
	sortie, err = lancer("importer", chemin)
	if err != nil {
		t.Fatalf("importer : %v", err)
	}
	if sortie != "2 hôte(s) importé(s)\n" {
		t.Errorf("importer : sortie %q", sortie)
	}
}
