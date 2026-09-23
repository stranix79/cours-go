package main

import (
	"slices"
	"testing"
)

// exemple construit un inventaire de départ pour les tests. Chaque test
// reçoit sa propre map : les modifications d'un test n'affectent pas l'autre.
func exemple() Inventaire {
	return Inventaire{
		"web":        {"web01", "web02"},
		"postgres":   {"db01", "db02"},
		"monitoring": {"web01"},
	}
}

func TestAjouter(t *testing.T) {
	t.Run("nouveau rôle", func(t *testing.T) {
		inv := Inventaire{}
		if !Ajouter(inv, "cache", "redis01") {
			t.Fatal("Ajouter a renvoyé false pour un hôte nouveau")
		}
		if !slices.Equal(inv["cache"], []string{"redis01"}) {
			t.Errorf("inv[cache] = %v, attendu [redis01]", inv["cache"])
		}
	})
	t.Run("rôle existant, ordre d'ajout conservé", func(t *testing.T) {
		inv := exemple()
		Ajouter(inv, "web", "web03")
		if !slices.Equal(inv["web"], []string{"web01", "web02", "web03"}) {
			t.Errorf("inv[web] = %v", inv["web"])
		}
	})
	t.Run("doublon refusé", func(t *testing.T) {
		inv := exemple()
		if Ajouter(inv, "web", "web01") {
			t.Error("Ajouter a accepté un doublon")
		}
		if len(inv["web"]) != 2 {
			t.Errorf("inv[web] = %v, l'hôte a été ajouté deux fois", inv["web"])
		}
	})
}

func TestRetirer(t *testing.T) {
	t.Run("présent", func(t *testing.T) {
		inv := exemple()
		if !Retirer(inv, "web", "web01") {
			t.Fatal("Retirer a renvoyé false pour un hôte présent")
		}
		if !slices.Equal(inv["web"], []string{"web02"}) {
			t.Errorf("inv[web] = %v, attendu [web02]", inv["web"])
		}
	})
	t.Run("absent", func(t *testing.T) {
		inv := exemple()
		if Retirer(inv, "web", "web99") {
			t.Error("Retirer a renvoyé true pour un hôte absent")
		}
	})
	t.Run("rôle inconnu", func(t *testing.T) {
		inv := exemple()
		if Retirer(inv, "dns", "ns01") {
			t.Error("Retirer a renvoyé true pour un rôle inconnu")
		}
	})
	t.Run("le rôle vide disparaît", func(t *testing.T) {
		inv := exemple()
		Retirer(inv, "monitoring", "web01")
		if _, existe := inv["monitoring"]; existe {
			t.Error("le rôle monitoring devrait avoir été supprimé")
		}
	})
}

func TestRoles(t *testing.T) {
	t.Run("triés", func(t *testing.T) {
		obtenu := Roles(exemple())
		attendu := []string{"monitoring", "postgres", "web"}
		if !slices.Equal(obtenu, attendu) {
			t.Errorf("Roles = %v, attendu %v", obtenu, attendu)
		}
	})
	t.Run("inventaire vide", func(t *testing.T) {
		if r := Roles(Inventaire{}); len(r) != 0 {
			t.Errorf("Roles(vide) = %v", r)
		}
	})
}

func TestHotes(t *testing.T) {
	t.Run("triés et sans doublon", func(t *testing.T) {
		obtenu := Hotes(exemple())
		attendu := []string{"db01", "db02", "web01", "web02"}
		if !slices.Equal(obtenu, attendu) {
			t.Errorf("Hotes = %v, attendu %v", obtenu, attendu)
		}
	})
	t.Run("inventaire vide", func(t *testing.T) {
		if h := Hotes(Inventaire{}); len(h) != 0 {
			t.Errorf("Hotes(vide) = %v", h)
		}
	})
}

func TestCompterParRole(t *testing.T) {
	obtenu := CompterParRole(exemple())
	attendu := map[string]int{"web": 2, "postgres": 2, "monitoring": 1}
	if len(obtenu) != len(attendu) {
		t.Fatalf("CompterParRole = %v, attendu %v", obtenu, attendu)
	}
	for role, n := range attendu {
		if obtenu[role] != n {
			t.Errorf("CompterParRole[%s] = %d, attendu %d", role, obtenu[role], n)
		}
	}
}

func TestDoublons(t *testing.T) {
	cas := []struct {
		nom     string
		inv     Inventaire
		attendu []string
	}{
		{"un hôte dans deux rôles", exemple(), []string{"web01"}},
		{"aucun doublon", Inventaire{"web": {"web01"}, "db": {"db01"}}, nil},
		{"plusieurs, triés", Inventaire{
			"a": {"zed", "alpha"},
			"b": {"alpha", "zed"},
			"c": {"zed"},
		}, []string{"alpha", "zed"}},
		{"vide", Inventaire{}, nil},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			obtenu := Doublons(c.inv)
			if len(obtenu) == 0 && len(c.attendu) == 0 {
				return
			}
			if !slices.Equal(obtenu, c.attendu) {
				t.Errorf("Doublons = %v, attendu %v", obtenu, c.attendu)
			}
		})
	}
}

func TestLister(t *testing.T) {
	obtenu := Lister(exemple())
	attendu := "monitoring: web01\npostgres: db01, db02\nweb: web01, web02\n"
	if obtenu != attendu {
		t.Errorf("Lister =\n%q\nattendu\n%q", obtenu, attendu)
	}
}

func TestPremiers(t *testing.T) {
	t.Run("les n premiers", func(t *testing.T) {
		hotes := []string{"a", "b", "c", "d"}
		obtenu := Premiers(hotes, 2)
		if !slices.Equal(obtenu, []string{"a", "b"}) {
			t.Errorf("Premiers = %v, attendu [a b]", obtenu)
		}
	})
	t.Run("n plus grand que la liste", func(t *testing.T) {
		obtenu := Premiers([]string{"a", "b"}, 5)
		if !slices.Equal(obtenu, []string{"a", "b"}) {
			t.Errorf("Premiers = %v, attendu [a b]", obtenu)
		}
	})
	t.Run("modifier le résultat ne touche pas l'original", func(t *testing.T) {
		hotes := []string{"a", "b", "c", "d"}
		p := Premiers(hotes, 2)
		p[0] = "modifié"
		if hotes[0] != "a" {
			t.Errorf("hotes[0] = %q : le résultat partage la mémoire de l'original", hotes[0])
		}
	})
	t.Run("append sur le résultat ne touche pas l'original", func(t *testing.T) {
		hotes := []string{"a", "b", "c", "d"}
		p := Premiers(hotes, 2)
		p = append(p, "intrus")
		if hotes[2] != "c" {
			t.Errorf("hotes = %v : l'append a écrasé l'original (capacité partagée)", hotes)
		}
		if len(p) != 3 {
			t.Errorf("len(p) = %d, attendu 3", len(p))
		}
	})
}
