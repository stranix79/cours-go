# Labo 15 : image multi-étapes pour diskexporter.
# Construire : docker build --build-arg VERSION=1.0.0 -t diskexporter:1.0.0 .
# Lancer     : docker run --rm -p 9101:9101 diskexporter:1.0.0 -points /
#
# Étape 1 : la chaîne de compilation complète (environ 800 Mo), jetée ensuite.
FROM golang:1.27 AS build
WORKDIR /src
# go.mod d'abord, seul : cette couche n'est reconstruite que s'il change.
# (go.sum* : le glob tolère l'absence du fichier, ce labo n'a pas de dépendance.)
COPY go.mod go.sum* ./
RUN go mod download
# Puis le code.
COPY . .
ARG VERSION=dev
ARG SRC=.
# CGO_ENABLED=0 : binaire statique, sans libc, qui tourne sur scratch.
# -trimpath : pas de chemins de ta machine dans le binaire (reproductible).
# -s -w : sans table des symboles ni infos de débogage (plus petit).
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/diskexporter ${SRC}

# Étape 2 : l'image finale. scratch = rien du tout : pas de shell, pas de
# libc, pas de /etc/passwd. Juste notre binaire, lancé par un utilisateur
# non root (65534 = nobody, en numérique puisqu'il n'y a pas de /etc/passwd).
FROM scratch
COPY --from=build /out/diskexporter /diskexporter
USER 65534:65534
EXPOSE 9101
ENTRYPOINT ["/diskexporter"]
CMD ["-addr", ":9101", "-points", "/"]
