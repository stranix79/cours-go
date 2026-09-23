package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Une config valide minimale, réutilisée par plusieurs tests.
const jsonMinimal = `{"cibles":[{"nom":"ssh","type":"tcp","adresse":"127.0.0.1:22"}]}`

func TestParser(t *testing.T) {
	cas := []struct {
		nom      string
		json     string
		veutErr  bool // une erreur est attendue
		invalide bool // et elle doit envelopper ErrInvalide
	}{
		{"config valide complète", `{"ecoute":":9090","intervalle":"10s","timeout":"2s","workers":2,
			"cibles":[{"nom":"web","type":"http","url":"http://127.0.0.1:8000/"},
			          {"nom":"pg","type":"tcp","adresse":"127.0.0.1:5432"}]}`, false, false},
		{"config minimale avec défauts", jsonMinimal, false, false},
		{"JSON illisible", `{"cibles":[`, true, false},
		{"intervalle illisible", `{"intervalle":"cinq secondes","cibles":[]}`, true, false},
		{"champ inconnu refusé", `{"intervale":"5s","cibles":[]}`, true, false},
		{"aucune cible", `{"cibles":[]}`, true, true},
		{"nom vide", `{"cibles":[{"nom":"  ","type":"tcp","adresse":"127.0.0.1:22"}]}`, true, true},
		{"type inconnu", `{"cibles":[{"nom":"x","type":"icmp","adresse":"127.0.0.1:22"}]}`, true, true},
		{"url sans schéma", `{"cibles":[{"nom":"x","type":"http","url":"127.0.0.1:8000"}]}`, true, true},
		{"adresse sans port", `{"cibles":[{"nom":"x","type":"tcp","adresse":"127.0.0.1"}]}`, true, true},
		{"nom en double", `{"cibles":[{"nom":"x","type":"tcp","adresse":"127.0.0.1:1"},
			{"nom":"x","type":"tcp","adresse":"127.0.0.1:2"}]}`, true, true},
		{"timeout supérieur à l'intervalle", `{"intervalle":"1s","timeout":"2s",
			"cibles":[{"nom":"x","type":"tcp","adresse":"127.0.0.1:1"}]}`, true, true},
		{"workers à zéro", `{"workers":0,"cibles":[{"nom":"x","type":"tcp","adresse":"127.0.0.1:1"}]}`, true, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			cfg, err := Parser(strings.NewReader(c.json))
			if c.veutErr {
				if err == nil {
					t.Fatalf("attendu une erreur, obtenu nil (cfg=%+v)", cfg)
				}
				if c.invalide && !errors.Is(err, ErrInvalide) {
					t.Fatalf("l'erreur doit envelopper ErrInvalide : %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("erreur inattendue : %v", err)
			}
			if cfg == nil || len(cfg.Cibles) == 0 {
				t.Fatalf("config vide : %+v", cfg)
			}
		})
	}
}

func TestDefauts(t *testing.T) {
	cfg, err := Parser(strings.NewReader(jsonMinimal))
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if cfg.Ecoute != ":8080" {
		t.Errorf("Ecoute = %q, attendu :8080", cfg.Ecoute)
	}
	if cfg.Intervalle.Duration() != 30*time.Second {
		t.Errorf("Intervalle = %s, attendu 30s", cfg.Intervalle.Duration())
	}
	if cfg.Timeout.Duration() != 5*time.Second {
		t.Errorf("Timeout = %s, attendu 5s", cfg.Timeout.Duration())
	}
	if cfg.Workers != 4 {
		t.Errorf("Workers = %d, attendu 4", cfg.Workers)
	}
}

func TestDuree(t *testing.T) {
	cas := []struct {
		nom   string
		texte string
		veut  time.Duration
		erreu bool
	}{
		{"secondes", "5s", 5 * time.Second, false},
		{"millisecondes", "250ms", 250 * time.Millisecond, false},
		{"minutes et secondes", "1m30s", 90 * time.Second, false},
		{"sans unité", "5", 0, true},
		{"texte", "vite", 0, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			var d Duree
			err := d.UnmarshalText([]byte(c.texte))
			if c.erreu {
				if err == nil {
					t.Fatalf("attendu une erreur pour %q", c.texte)
				}
				return
			}
			if err != nil {
				t.Fatalf("erreur inattendue : %v", err)
			}
			if d.Duration() != c.veut {
				t.Errorf("obtenu %s, attendu %s", d.Duration(), c.veut)
			}
			b, err := d.MarshalText()
			if err != nil || string(b) != c.texte {
				t.Errorf("MarshalText = %q (%v), attendu %q", b, err, c.texte)
			}
		})
	}
}

func TestCharger(t *testing.T) {
	t.Run("fichier valide", func(t *testing.T) {
		chemin := filepath.Join(t.TempDir(), "sondes.json")
		if err := os.WriteFile(chemin, []byte(jsonMinimal), 0o644); err != nil {
			t.Fatal(err)
		}
		cfg, err := Charger(chemin)
		if err != nil {
			t.Fatalf("erreur inattendue : %v", err)
		}
		if len(cfg.Cibles) != 1 || cfg.Cibles[0].Nom != "ssh" {
			t.Errorf("cibles = %+v", cfg.Cibles)
		}
	})
	t.Run("fichier absent", func(t *testing.T) {
		if _, err := Charger(filepath.Join(t.TempDir(), "absent.json")); err == nil {
			t.Fatal("attendu une erreur pour un fichier absent")
		}
	})
}
