#!/usr/bin/env bash
# Lancé par tools/verifier.sh depuis solution/ : les tests de la solution sont
# dans les sous-paquets internal/..., pas à la racine de solution/, d'où ce
# relais. Depuis labs/16-projet, l'équivalent est : go test ./solution/...
set -e
cd "$(dirname "$0")/.."
go test -count=1 ./solution/...
