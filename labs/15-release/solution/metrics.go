// Labo 15, solution : collecte des mesures et format d'exposition Prometheus.
// Utilisé par : main.go (nouveauRouteur), les tests (formaterMetriques, handlers).
//
// Le format texte de Prometheus est simple à écrire à la main : pour chaque
// métrique, une ligne "# HELP", une ligne "# TYPE", puis une ligne par jeu
// de labels : nom{label="valeur"} nombre. C'est ce que fait ce fichier, sans
// bibliothèque. Le paquet client_golang fait la même chose en mieux (types,
// histogrammes, métriques du runtime) ; le chapitre 15 montre les deux.
package main

import (
	"bytes"    // tampon pour construire la réponse avant de l'envoyer
	"errors"   // errors.Join : plusieurs erreurs en une
	"fmt"      // Fprintf vers un io.Writer
	"io"       // l'interface Writer : formaterMetriques ne connaît pas HTTP
	"log/slog" // journaliser les collectes partielles
	"net/http" // les handlers
	"strconv"  // formatage des nombres sans passer par %v
	"strings"  // Replacer pour échapper les labels
)

// Mesure est une ligne de métriques : un point de montage et son Espace.
// L'embarquement (Espace sans nom de champ) donne m.Total et m.Libre
// directement, comme si Mesure avait ces champs.
type Mesure struct {
	Point string
	Espace
}

// collecter interroge la sonde pour chaque point et renvoie les mesures
// obtenues. Un point qui échoue n'empêche pas les autres : on continue, et
// on renvoie toutes les erreurs jointes. L'appelant reçoit donc à la fois
// des mesures ET une erreur, ce qui est inhabituel mais voulu ici : un
// exporter doit exposer ce qu'il sait, pas tout cacher pour un disque absent.
func collecter(s Sonde, points []string) ([]Mesure, error) {
	var mesures []Mesure
	var erreurs []error
	for _, p := range points {
		e, err := s.Statfs(p)
		if err != nil {
			erreurs = append(erreurs, err)
			continue
		}
		mesures = append(mesures, Mesure{Point: p, Espace: e})
	}
	// errors.Join(nil...) renvoie nil : pas d'erreur si tout a réussi.
	return mesures, errors.Join(erreurs...)
}

// echapperLabel rend une valeur de label sûre pour le format texte : la
// barre oblique inverse, le guillemet et le saut de ligne sont échappés,
// exactement comme la spécification l'exige. Un point de montage nommé
// /Volumes/Disque "externe" ne casse ainsi pas la ligne.
func echapperLabel(v string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return r.Replace(v)
}

// ratioUtilise calcule la fraction utilisée, entre 0 et 1. Total à zéro
// (un pseudo-système de fichiers comme /proc) donnerait une division par
// zéro : on renvoie 0 dans ce cas.
func ratioUtilise(e Espace) float64 {
	if e.Total == 0 {
		return 0
	}
	return float64(e.Total-e.Libre) / float64(e.Total)
}

// formaterMetriques écrit la page /metrics complète dans w. Elle ne sait rien
// de HTTP : un io.Writer suffit, ce qui permet de la tester avec un
// bytes.Buffer et de comparer la chaîne obtenue à la chaîne attendue.
func formaterMetriques(w io.Writer, version string, mesures []Mesure, nbErreurs int) {
	// build_info : la convention Prometheus pour exposer une version. La
	// valeur est toujours 1, l'information est dans le label.
	fmt.Fprintln(w, "# HELP diskexporter_build_info Version du binaire ; la valeur vaut toujours 1.")
	fmt.Fprintln(w, "# TYPE diskexporter_build_info gauge")
	fmt.Fprintf(w, "diskexporter_build_info{version=\"%s\"} 1\n", echapperLabel(version))

	// scrape_errors : combien de points n'ont pas pu être mesurés lors de
	// cette collecte. Une alerte "> 0" vaut mieux qu'une métrique absente.
	fmt.Fprintln(w, "# HELP diskexporter_scrape_errors Points de montage non mesurés lors de cette collecte.")
	fmt.Fprintln(w, "# TYPE diskexporter_scrape_errors gauge")
	fmt.Fprintf(w, "diskexporter_scrape_errors %d\n", nbErreurs)

	// Une famille de métriques = HELP + TYPE + une ligne par mesure. La
	// closure évite d'écrire trois fois la même boucle ; elle reçoit la
	// fonction qui extrait la valeur, déjà formatée en texte.
	famille := func(nom, aide string, valeur func(Mesure) string) {
		fmt.Fprintf(w, "# HELP %s %s\n", nom, aide)
		fmt.Fprintf(w, "# TYPE %s gauge\n", nom)
		for _, m := range mesures {
			fmt.Fprintf(w, "%s{mountpoint=\"%s\"} %s\n", nom, echapperLabel(m.Point), valeur(m))
		}
	}
	famille("disk_total_bytes", "Taille du point de montage, en octets.",
		func(m Mesure) string { return strconv.FormatUint(m.Total, 10) })
	famille("disk_free_bytes", "Espace libre pour un utilisateur non root, en octets.",
		func(m Mesure) string { return strconv.FormatUint(m.Libre, 10) })
	famille("disk_used_ratio", "Fraction utilisée, entre 0 et 1.",
		// 'f' et 4 décimales : 0.7500 plutôt que 0.75 ou 7.5e-01, lisible et stable.
		func(m Mesure) string { return strconv.FormatFloat(ratioUtilise(m.Espace), 'f', 4, 64) })
}

// handlerMetrics répond à GET /metrics. On construit toute la réponse dans
// un tampon AVANT d'écrire : le premier appel à w.Write envoie le statut 200
// et les en-têtes, il serait trop tard pour changer d'avis ensuite.
func handlerMetrics(s Sonde, points []string, version string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mesures, err := collecter(s, points)
		nbErreurs := 0
		if err != nil {
			// On journalise, mais on répond quand même : collecte partielle.
			slog.Warn("collecte partielle", "err", err)
			nbErreurs = len(points) - len(mesures)
		}
		var buf bytes.Buffer
		formaterMetriques(&buf, version, mesures, nbErreurs)

		// Le Content-Type exact que Prometheus attend pour le format texte.
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		// WriteTo renvoie (n, err) ; une erreur ici veut dire que le client a
		// raccroché, on n'a personne à qui la signaler : on l'ignore explicitement.
		_, _ = buf.WriteTo(w)
	})
}

// handlerHealthz répond "ok" : le processus tourne et sert HTTP. C'est une
// sonde de vivacité (liveness) : Docker ou Kubernetes redémarrent le
// conteneur si elle ne répond plus. Elle ne mesure rien, exprès : un disque
// injoignable ne doit pas faire redémarrer l'exporter en boucle.
func handlerHealthz() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "ok\n")
	})
}
