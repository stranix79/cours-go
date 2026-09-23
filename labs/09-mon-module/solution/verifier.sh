#!/usr/bin/env bash
# Labo 09 : vérification de la solution, appelée par tools/verifier.sh
# (les tests ne sont pas directement dans solution/ mais dans ses sous-paquets).
# Lancé par : bash solution/verifier.sh   (depuis labs/09-mon-module)
set -eu
cd "$(dirname "$0")"
go test -count=1 ./...
