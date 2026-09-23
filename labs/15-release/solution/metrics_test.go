// Labo 15 : tests du format d'exposition et des handlers HTTP.
// Lancé par : go test .   (ton code)   ou   go test ./solution/   (la solution)
//
// Aucun port n'est ouvert : formaterMetriques reçoit un bytes.Buffer, et les
// handlers reçoivent un httptest.ResponseRecorder. Les chiffres viennent
// d'une fausseSonde, donc la sortie attendue est la même sur toute machine.
package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sondeTest est la sonde utilisée par tous les tests : deux points connus.
var sondeTest = fausseSonde{
	"/":     {Total: 1000, Libre: 250},
	"/data": {Total: 4000, Libre: 4000},
}

func TestEchapperLabel(t *testing.T) {
	cas := []struct {
		nom, entree, attendu string
	}{
		{"chemin simple", "/var/lib", "/var/lib"},
		{"guillemet", `/Volumes/Disque "externe"`, `/Volumes/Disque \"externe\"`},
		{"barre oblique inverse", `C:\data`, `C:\\data`},
		{"saut de ligne", "a\nb", `a\nb`},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := echapperLabel(c.entree); got != c.attendu {
				t.Errorf("echapperLabel(%q) = %q, attendu %q", c.entree, got, c.attendu)
			}
		})
	}
}

func TestRatioUtilise(t *testing.T) {
	cas := []struct {
		nom     string
		espace  Espace
		attendu float64
	}{
		{"trois quarts", Espace{Total: 1000, Libre: 250}, 0.75},
		{"vide", Espace{Total: 4000, Libre: 4000}, 0},
		{"plein", Espace{Total: 10, Libre: 0}, 1},
		{"total nul", Espace{}, 0},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := ratioUtilise(c.espace); got != c.attendu {
				t.Errorf("ratioUtilise(%+v) = %v, attendu %v", c.espace, got, c.attendu)
			}
		})
	}
}

func TestCollecter(t *testing.T) {
	t.Run("tous les points connus", func(t *testing.T) {
		mesures, err := collecter(sondeTest, []string{"/", "/data"})
		if err != nil {
			t.Fatalf("erreur inattendue : %v", err)
		}
		if len(mesures) != 2 || mesures[0].Point != "/" || mesures[1].Total != 4000 {
			t.Errorf("mesures = %+v", mesures)
		}
	})
	t.Run("un point inconnu n'empêche pas les autres", func(t *testing.T) {
		mesures, err := collecter(sondeTest, []string{"/", "/absent", "/data"})
		if err == nil {
			t.Fatal("erreur attendue pour /absent")
		}
		if !strings.Contains(err.Error(), "/absent") {
			t.Errorf("l'erreur doit nommer le point : %v", err)
		}
		if len(mesures) != 2 {
			t.Errorf("2 mesures attendues malgré l'erreur, obtenu %d", len(mesures))
		}
	})
}

func TestFormaterMetriques(t *testing.T) {
	mesures, _ := collecter(sondeTest, []string{"/", "/data"})
	var buf bytes.Buffer
	formaterMetriques(&buf, "1.2.3", mesures, 1)

	attendu := `# HELP diskexporter_build_info Version du binaire ; la valeur vaut toujours 1.
# TYPE diskexporter_build_info gauge
diskexporter_build_info{version="1.2.3"} 1
# HELP diskexporter_scrape_errors Points de montage non mesurés lors de cette collecte.
# TYPE diskexporter_scrape_errors gauge
diskexporter_scrape_errors 1
# HELP disk_total_bytes Taille du point de montage, en octets.
# TYPE disk_total_bytes gauge
disk_total_bytes{mountpoint="/"} 1000
disk_total_bytes{mountpoint="/data"} 4000
# HELP disk_free_bytes Espace libre pour un utilisateur non root, en octets.
# TYPE disk_free_bytes gauge
disk_free_bytes{mountpoint="/"} 250
disk_free_bytes{mountpoint="/data"} 4000
# HELP disk_used_ratio Fraction utilisée, entre 0 et 1.
# TYPE disk_used_ratio gauge
disk_used_ratio{mountpoint="/"} 0.7500
disk_used_ratio{mountpoint="/data"} 0.0000
`
	if buf.String() != attendu {
		t.Errorf("sortie inattendue :\n--- obtenu ---\n%s--- attendu ---\n%s", buf.String(), attendu)
	}
}

func TestHandlerMetrics(t *testing.T) {
	h := handlerMetrics(sondeTest, []string{"/", "/absent"}, "test")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("statut = %d, attendu 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain; version=0.0.4; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	corps := rec.Body.String()
	for _, ligne := range []string{
		`diskexporter_build_info{version="test"} 1`,
		`diskexporter_scrape_errors 1`,
		`disk_total_bytes{mountpoint="/"} 1000`,
		`disk_used_ratio{mountpoint="/"} 0.7500`,
	} {
		if !strings.Contains(corps, ligne+"\n") {
			t.Errorf("ligne absente : %s\ncorps :\n%s", ligne, corps)
		}
	}
	if strings.Contains(corps, "/absent") {
		t.Errorf("un point non mesuré ne doit pas apparaître :\n%s", corps)
	}
}

func TestHandlerHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	handlerHealthz().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "ok\n" {
		t.Errorf("healthz : statut %d, corps %q", rec.Code, rec.Body.String())
	}
}
