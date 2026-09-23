// Paquet api, solution : metrics.go, le format d'exposition Prometheus.
//
// Quoi : FormatMetriques écrit trois métriques au format texte que Prometheus
// scrappe : sonde_up (gauge), sonde_latence_secondes (gauge),
// sonde_verifications_total (counter). Vingt lignes, pas de bibliothèque.
// Qui l'appelle : le handler /metrics ; les tests, avec un bytes.Buffer.
package api

import (
	"fmt"     // fmt.Fprintf vers un io.Writer
	"io"      // io.Writer : la fonction ne sait pas si c'est HTTP ou un buffer
	"strings" // strings.NewReplacer pour échapper les labels

	"cours-go/labs/16-projet/solution/internal/sonde"
)

// echapperLabel applique les trois échappements que le format exige dans une
// valeur de label : la barre oblique inverse, le guillemet et le saut de
// ligne. Un nom de cible bizarre ne doit pas casser le scrape.
var echapperLabel = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)

// FormatMetriques écrit les métriques. Le format : pour chaque métrique, une
// ligne # HELP, une ligne # TYPE, puis une ligne par série
// nom{label="valeur",...} valeur. Les labels sont toujours dans le même ordre
// et les cibles triées (Etat.Resultats les trie), donc la sortie est stable.
func FormatMetriques(w io.Writer, resultats []sonde.Resultat, total uint64) {
	fmt.Fprintln(w, "# HELP sonde_up 1 si la cible a répondu à la dernière vérification, 0 sinon.")
	fmt.Fprintln(w, "# TYPE sonde_up gauge")
	for _, r := range resultats {
		fmt.Fprintf(w, "sonde_up{cible=\"%s\",type=\"%s\"} %d\n",
			echapperLabel.Replace(r.Cible), echapperLabel.Replace(r.Type), booleen(r.Up))
	}
	fmt.Fprintln(w, "# HELP sonde_latence_secondes Durée de la dernière vérification, en secondes.")
	fmt.Fprintln(w, "# TYPE sonde_latence_secondes gauge")
	for _, r := range resultats {
		// %g : notation la plus courte, 0.012 plutôt que 0.012000 ; Prometheus
		// lit n'importe quel flottant.
		fmt.Fprintf(w, "sonde_latence_secondes{cible=\"%s\",type=\"%s\"} %g\n",
			echapperLabel.Replace(r.Cible), echapperLabel.Replace(r.Type), r.Latence.Seconds())
	}
	fmt.Fprintln(w, "# HELP sonde_verifications_total Nombre de vérifications effectuées depuis le démarrage.")
	fmt.Fprintln(w, "# TYPE sonde_verifications_total counter")
	fmt.Fprintf(w, "sonde_verifications_total %d\n", total)
}

// booleen convertit un bool en 0/1 : Prometheus n'a pas de booléens.
func booleen(b bool) int {
	if b {
		return 1
	}
	return 0
}
