// Labo 15 : collecte des mesures et format d'exposition Prometheus, à la main.
// Utilisé par : main.go (nouveauRouteur), metrics_test.go.
//
// Le format texte de Prometheus, pour une jauge avec un label :
//
//	# HELP disk_total_bytes Taille du point de montage, en octets.
//	# TYPE disk_total_bytes gauge
//	disk_total_bytes{mountpoint="/"} 1000
//
// Le test TestFormaterMetriques contient la page complète attendue : lis-le
// avant d'écrire formaterMetriques, c'est la spécification.
package main

import (
	"io"
	"net/http"
)

// Mesure est une ligne de métriques : un point de montage et son Espace.
// Espace est embarqué : m.Total et m.Libre marchent directement.
type Mesure struct {
	Point string
	Espace
}

// collecter interroge la sonde pour chaque point. Un point qui échoue
// n'empêche pas les autres : on continue, et on renvoie les mesures obtenues
// ET les erreurs jointes (errors.Join). Sans erreur, le second retour est nil.
// TODO 2 : la boucle, l'append des mesures, l'accumulation des erreurs.
func collecter(s Sonde, points []string) ([]Mesure, error) {
	return nil, nil
}

// echapperLabel rend une valeur de label sûre : `\` devient `\\`, `"` devient
// `\"`, le saut de ligne devient `\n` (deux caractères).
// TODO 3 : strings.NewReplacer avec les trois paires, puis Replace.
func echapperLabel(v string) string {
	return v
}

// ratioUtilise calcule (Total - Libre) / Total, entre 0 et 1 ; 0 si Total vaut 0.
// TODO 3 bis : attention à la division entière, convertis en float64 avant.
func ratioUtilise(e Espace) float64 {
	return 0
}

// formaterMetriques écrit la page /metrics complète dans w, dans cet ordre :
// diskexporter_build_info{version="..."} 1, diskexporter_scrape_errors N,
// puis pour chaque famille disk_total_bytes, disk_free_bytes, disk_used_ratio :
// une ligne HELP, une ligne TYPE gauge, une ligne par mesure avec le label
// mountpoint. Les textes HELP exacts sont dans le test. Le ratio est formaté
// avec 4 décimales (strconv.FormatFloat(x, 'f', 4, 64)).
// TODO 4 : fmt.Fprintf vers w. Une closure "famille(nom, aide, valeur)" évite
// d'écrire trois fois la même boucle.
func formaterMetriques(w io.Writer, version string, mesures []Mesure, nbErreurs int) {
}

// handlerMetrics répond à GET /metrics : collecter, compter les erreurs
// (len(points) - len(mesures)), formater dans un bytes.Buffer, poser
// Content-Type: text/plain; version=0.0.4; charset=utf-8, puis écrire.
// TODO 5 : http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { ... }).
func handlerMetrics(s Sonde, points []string, version string) http.Handler {
	return http.NotFoundHandler()
}

// handlerHealthz répond 200 avec le corps "ok\n" (sonde de vivacité).
// TODO 5 : idem, avec io.WriteString(w, "ok\n").
func handlerHealthz() http.Handler {
	return http.NotFoundHandler()
}
