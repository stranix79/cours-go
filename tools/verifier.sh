#!/usr/bin/env bash
# tools/verifier.sh : vérifie que le cours est cohérent :
#   1. tous les liens relatifs des .md pointent vers un fichier existant ;
#   2. chaque labo est formaté (gofmt), passe go vet (squelette ET solution),
#      et sa solution passe ses tests (ou se lance, s'il n'y a pas de tests).
# Lancé par : ./tools/verifier.sh depuis la racine du dépôt (Go 1.25+ requis).
set -u
cd "$(dirname "$0")/.."
erreurs=0

echo "== Liens markdown"
# Extrait chaque [texte](cible) des .md hors blocs de code, ignore les URLs,
# vérifie que la cible existe.
while IFS=: read -r fichier cible; do
    cible="${cible%%#*}"                       # enlève l'ancre #section
    [ -z "$cible" ] && continue
    chemin="$(dirname "$fichier")/$cible"
    if [ ! -e "$chemin" ]; then
        echo "  CASSÉ  $fichier → $cible"
        erreurs=$((erreurs + 1))
    fi
done < <(grep -r --include='*.md' -H '' README.md cours labs tools \
         | awk -F: '{ if ($2 ~ /^```/) { fence[$1] = !fence[$1]; next } if (!fence[$1]) print }' \
         | sed -E 's/`[^`]*`//g' \
         | grep -o -E '^[^:]+:.*\]\([^)]+\)' \
         | sed -E 's/^([^:]+):.*\]\(([^)]+)\)$/\1:\2/' | grep -v -E ':(https?:|mailto:)')
echo "  liens vérifiés"

echo "== Labos"
for labo in labs/*/; do
    nom=$(basename "$labo")
    log=/tmp/verif_go_$nom.log
    if [ ! -f "$labo/go.mod" ]; then
        echo "  —      $nom (pas de go.mod)"
        continue
    fi
    (
        cd "$labo" || exit 1
        mal=$(gofmt -l . 2>&1)
        if [ -n "$mal" ]; then echo "gofmt : $mal"; exit 1; fi
        go vet ./... || exit 1
        if ls solution/*_test.go >/dev/null 2>&1; then
            go test -count=1 ./solution/... || exit 1
        elif [ -f solution/verifier.sh ]; then
            (cd solution && bash verifier.sh) || exit 1
        else
            go build -o /dev/null ./solution || exit 1
        fi
    ) >"$log" 2>&1
    if [ $? -eq 0 ]; then
        echo "  OK     $nom"
    else
        echo "  ÉCHEC  $nom → $log"
        erreurs=$((erreurs + 1))
    fi
done

echo
if [ "$erreurs" -eq 0 ]; then echo "Tout est bon."; else echo "$erreurs problème(s)."; exit 1; fi
