# Étape 1 : compiler avec la chaîne Go complète (image ~800 Mo, jetée ensuite).
FROM golang:1.27 AS build
WORKDIR /src
# go.mod d'abord : cette couche n'est reconstruite que si les dépendances
# changent (ici aucune, mais le réflexe vaut pour tous les projets).
COPY go.mod ./
RUN go mod download
COPY . .
ARG VERSION=dev
# CGO_ENABLED=0 : binaire statique, sans libc, qui tourne dans scratch.
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/sondes ./cmd/sondes

# Étape 2 : l'image finale, vide. Pas de shell, pas de libc, pas de paquet :
# juste le binaire, les certificats racine (pour sonder du https) et la
# config. Résultat : une dizaine de méga-octets.
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/sondes /sondes
COPY sondes.json /etc/sondes/sondes.json
# Utilisateur non root (nobody). scratch n'a pas de /etc/passwd : on donne
# l'UID numérique.
USER 65534:65534
ENV SONDES_CONFIG=/etc/sondes/sondes.json
EXPOSE 8080
ENTRYPOINT ["/sondes"]
