package main

import (
	"errors"
	"testing"
	"time"
)

func TestSlugify(t *testing.T) {
	cas := []struct {
		nom     string
		entree  string
		attendu string
	}{
		{"exemple du README", "Serveur Web #1 (prod)", "serveur-web-1-prod"},
		{"déjà propre", "web01", "web01"},
		{"majuscules", "PostgreSQL 16", "postgresql-16"},
		{"séparateurs multiples", "a  --  b", "a-b"},
		{"blancs aux extrémités", "  hello world  ", "hello-world"},
		{"accents conservés", "Été 2026", "été-2026"},
		{"vide", "", ""},
		{"que des séparateurs", "###", ""},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if obtenu := Slugify(c.entree); obtenu != c.attendu {
				t.Errorf("Slugify(%q) = %q, attendu %q", c.entree, obtenu, c.attendu)
			}
		})
	}
}

func TestParseKV(t *testing.T) {
	cas := []struct {
		nom     string
		entree  string
		attendu map[string]string
	}{
		{"simple", "host=web01 port=22", map[string]string{"host": "web01", "port": "22"}},
		{"blancs multiples et tabulation", "  host=web01\tport=22  ", map[string]string{"host": "web01", "port": "22"}},
		{"morceau sans égal ignoré", "host=web01 debug port=22", map[string]string{"host": "web01", "port": "22"}},
		{"valeur contenant un égal", "url=a=b", map[string]string{"url": "a=b"}},
		{"valeur vide", "host=", map[string]string{"host": ""}},
		{"clé vide ignorée", "=x host=y", map[string]string{"host": "y"}},
		{"dernière clé gagne", "a=1 a=2", map[string]string{"a": "2"}},
		{"texte vide", "", map[string]string{}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			obtenu := ParseKV(c.entree)
			if len(obtenu) != len(c.attendu) {
				t.Fatalf("ParseKV(%q) = %v, attendu %v", c.entree, obtenu, c.attendu)
			}
			for k, v := range c.attendu {
				if obtenu[k] != v {
					t.Errorf("ParseKV(%q)[%q] = %q, attendu %q", c.entree, k, obtenu[k], v)
				}
			}
		})
	}
}

func TestSomme(t *testing.T) {
	cas := []struct {
		nom     string
		nombres []int
		attendu int
	}{
		{"trois nombres", []int{1, 2, 3}, 6},
		{"un seul", []int{42}, 42},
		{"aucun", nil, 0},
		{"négatifs", []int{-5, 5, -1}, -1},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if obtenu := Somme(c.nombres...); obtenu != c.attendu {
				t.Errorf("Somme(%v) = %d, attendu %d", c.nombres, obtenu, c.attendu)
			}
		})
	}
}

func TestNouveauCompteur(t *testing.T) {
	t.Run("compte à partir de 1", func(t *testing.T) {
		c := NouveauCompteur()
		for attendu := 1; attendu <= 3; attendu++ {
			if obtenu := c(); obtenu != attendu {
				t.Fatalf("appel %d : obtenu %d", attendu, obtenu)
			}
		}
	})
	t.Run("deux compteurs sont indépendants", func(t *testing.T) {
		a := NouveauCompteur()
		b := NouveauCompteur()
		a()
		a()
		if obtenu := b(); obtenu != 1 {
			t.Errorf("b() = %d après deux appels de a, attendu 1", obtenu)
		}
		if obtenu := a(); obtenu != 3 {
			t.Errorf("a() = %d, attendu 3", obtenu)
		}
	})
}

func TestRetry(t *testing.T) {
	errTest := errors.New("pas encore prêt")

	t.Run("réussit du premier coup", func(t *testing.T) {
		appels := 0
		err := Retry(3, 0, func() error {
			appels++
			return nil
		})
		if err != nil {
			t.Fatalf("erreur inattendue : %v", err)
		}
		if appels != 1 {
			t.Errorf("f appelée %d fois, attendu 1", appels)
		}
	})

	t.Run("échoue deux fois puis réussit", func(t *testing.T) {
		appels := 0
		err := Retry(5, 0, func() error {
			appels++
			if appels < 3 {
				return errTest
			}
			return nil
		})
		if err != nil {
			t.Fatalf("erreur inattendue : %v", err)
		}
		if appels != 3 {
			t.Errorf("f appelée %d fois, attendu 3", appels)
		}
	})

	t.Run("échoue toujours : dernière erreur enveloppée", func(t *testing.T) {
		appels := 0
		err := Retry(3, 0, func() error {
			appels++
			return errTest
		})
		if err == nil {
			t.Fatal("erreur attendue, obtenu nil")
		}
		if appels != 3 {
			t.Errorf("f appelée %d fois, attendu 3", appels)
		}
		if !errors.Is(err, errTest) {
			t.Errorf("l'erreur renvoyée %q n'enveloppe pas l'erreur d'origine (%%w)", err)
		}
	})

	t.Run("attend entre les essais", func(t *testing.T) {
		debut := time.Now()
		_ = Retry(3, 20*time.Millisecond, func() error { return errTest })
		if ecoule := time.Since(debut); ecoule < 40*time.Millisecond {
			t.Errorf("3 essais avec 20ms d'attente ont pris %v, attendu au moins 40ms", ecoule)
		}
	})

	t.Run("zéro essai est une erreur", func(t *testing.T) {
		err := Retry(0, 0, func() error { return nil })
		if !errors.Is(err, ErrEssais) {
			t.Errorf("Retry(0, ...) = %v, attendu ErrEssais", err)
		}
	})
}
