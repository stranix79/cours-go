# Labo 16 : sondes. Cibles : build, build-solution, test, lint, run, cross, docker, clean.
BIN      := bin/sondes
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)
GOFLAGS  := -trimpath

.PHONY: build build-solution test lint run run-solution cross docker clean

build:
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/sondes

build-solution:
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN)-solution ./solution/cmd/sondes

test:
	go test -race -count=1 ./...

lint:
	@test -z "$$(gofmt -l .)" || (echo "gofmt :"; gofmt -l .; exit 1)
	go vet ./...
	@if command -v golangci-lint >/dev/null; then golangci-lint run ./...; else echo "golangci-lint absent, ignoré"; fi

run: build
	./$(BIN) -config sondes.json

run-solution: build-solution
	./$(BIN)-solution -config sondes.json

# Un binaire par plateforme, sans rien installer.
cross:
	@for cible in linux/amd64 linux/arm64 darwin/arm64 windows/amd64; do \
		os=$${cible%/*}; arch=$${cible#*/}; ext=""; [ "$$os" = windows ] && ext=.exe; \
		echo "  $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
			-o bin/sondes-$$os-$$arch$$ext ./cmd/sondes || exit 1; \
	done

docker:
	docker build --build-arg VERSION=$(VERSION) -t sondes:$(VERSION) .

clean:
	rm -rf bin
