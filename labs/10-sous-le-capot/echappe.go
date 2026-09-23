// Labo 10 : deux constructeurs à comparer avec l'analyse d'échappement.
// Lancé par : go build -gcflags='-m -l' .   (puis go test .)
package capot

// Serveur décrit une machine à sonder. Trois champs, dont une slice : la
// struct fait 40 octets (16 pour Nom, 8 pour Port, 24 pour Tags, arrondis).
type Serveur struct {
	Nom  string
	Port int
	Tags []string
}

// NouveauParValeur construit un Serveur et le renvoie PAR VALEUR (une copie).
// Port vaut 5432 par défaut, Tags est nil.
func NouveauParValeur(nom string) Serveur {
	// TODO 1 : déclare une variable locale s de type Serveur avec Nom et
	// Port: 5432, et renvoie s. Puis regarde ce que dit -gcflags='-m -l'.
	return Serveur{}
}

// NouveauParPointeur construit le même Serveur mais renvoie son ADRESSE.
func NouveauParPointeur(nom string) *Serveur {
	// TODO 2 : même variable locale s, mais renvoie &s. Le compilateur va
	// dire « moved to heap: s » : explique pourquoi dans ta tête avant de
	// lire le README.
	return nil
}
