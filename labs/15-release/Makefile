# Labo 15 : Makefile de diskexporter.
# Usage : make build | test | lint | cross | docker | release | clean
#         make build SRC=./solution   (construit la solution au lieu de ton code)
#         make build VERSION=1.0.0    (force la version injectée)

BIN      := diskexporter
SRC      ?= .
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)
GOFLAGS  := -trimpath
export CGO_ENABLED := 0

.PHONY: build test lint cross docker release clean run

## build : binaire statique pour la machine courante, dans bin/
build:
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o bin/$(BIN) $(SRC)

## run : lance ton code sur :9101
run:
	go run $(SRC) -addr :9101 -points /

## test : tests avec le détecteur de course, sans cache
test:
	go test -race -count=1 ./...

## lint : gofmt, go vet, golangci-lint
lint:
	@test -z "$$(gofmt -l .)" || (echo "gofmt : fichiers à formater :"; gofmt -l .; exit 1)
	go vet ./...
	golangci-lint run ./...

## cross : un binaire par cible dans bin/, sans rien installer
cross:
	@for cible in linux/amd64 linux/arm64 darwin/arm64 windows/amd64; do \
		os=$${cible%/*}; arch=$${cible#*/}; ext=""; \
		[ "$$os" = windows ] && ext=.exe; \
		echo "  $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
			-o bin/$(BIN)-$$os-$$arch$$ext $(SRC) || exit 1; \
	done

## docker : image multi-étapes, binaire seul sur scratch
docker:
	docker build --build-arg VERSION=$(VERSION) --build-arg SRC=$(SRC) -t $(BIN):$(VERSION) .

## release : répétition locale de goreleaser, sans tag ni GitHub
release:
	goreleaser release --snapshot --clean

clean:
	rm -rf bin dist
