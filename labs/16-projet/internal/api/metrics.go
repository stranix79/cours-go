// Paquet api : metrics.go, le format d'exposition Prometheus écrit à la main.
package api

import (
	"io"
	"strings"

	"cours-go/labs/16-projet/internal/sonde"
)

// echapperLabel applique les trois échappements du format dans une valeur de
// label : \ devient \\, " devient \", saut de ligne devient \n.
var echapperLabel = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)

// FormatMetriques écrit, dans cet ordre, exactement :
//
//	# HELP sonde_up 1 si la cible a répondu à la dernière vérification, 0 sinon.
//	# TYPE sonde_up gauge
//	sonde_up{cible="NOM",type="TYPE"} 0|1          (une ligne par résultat)
//	# HELP sonde_latence_secondes Durée de la dernière vérification, en secondes.
//	# TYPE sonde_latence_secondes gauge
//	sonde_latence_secondes{cible="NOM",type="TYPE"} SECONDES   (%g, une par résultat)
//	# HELP sonde_verifications_total Nombre de vérifications effectuées depuis le démarrage.
//	# TYPE sonde_verifications_total counter
//	sonde_verifications_total N
//
// Les valeurs de label passent par echapperLabel.Replace.
func FormatMetriques(w io.Writer, resultats []sonde.Resultat, total uint64) {
	// TODO 19
	_, _, _ = w, resultats, total
	_ = echapperLabel
}
