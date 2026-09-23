// Labo 10, solution : deux constructeurs à comparer avec l'analyse
// d'échappement. Lancé par : go build -gcflags='-m -l' ./solution
//
// Les deux fonctions font strictement la même chose ; seule la façon de
// RENDRE le résultat change, et c'est elle qui décide pile ou tas.
package capot

// Serveur décrit une machine à sonder.
type Serveur struct {
	Nom  string
	Port int
	Tags []string
}

// NouveauParValeur renvoie une copie : la variable locale s est copiée dans
// la valeur de retour, puis oubliée. Rien ne survit à la fonction, donc s
// reste sur la pile. Le compilateur dit seulement « leaking param: nom to
// result » : la string nom (son en-tête) ressort dans le résultat, ce n'est
// pas une allocation.
func NouveauParValeur(nom string) Serveur {
	s := Serveur{Nom: nom, Port: 5432} // sur la pile
	return s                           // copie de 40 octets vers l'appelant
}

// NouveauParPointeur renvoie l'ADRESSE de s. L'appelant va lire cette
// adresse après le retour de la fonction, donc s ne peut pas être sur une
// pile qui n'existera plus : le compilateur la déplace sur le tas
// (« moved to heap: s »). Légal en Go, bug en C, et une allocation ici.
func NouveauParPointeur(nom string) *Serveur {
	s := Serveur{Nom: nom, Port: 5432} // sur le tas, décidé à la compilation
	return &s                          // 8 octets copiés : l'adresse
}
