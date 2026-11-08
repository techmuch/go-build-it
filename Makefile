# go-build-it build (P0-09, P1-12)
VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)
PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64

.PHONY: all build web go gen test vet e2e release run

all: build

build: web go

web:
	cd web && npm run build && touch dist/.keep

go:
	go build -ldflags "$(LDFLAGS)" -o bin/gbi ./cmd/gbi

gen:
	go run ./cmd/gbi dev gen-client web/src/api.gen.ts

vet:
	go vet ./...
	cd web && npx tsc --noEmit -p .

test: vet
	go test ./...

e2e: build
	cd web && npx playwright test

run: build
	./bin/gbi serve

# Cross-compile signed release assets into release/ (ADR-004).
#   VERSION=YYYY.MM.DD.N make release
release: web
	mkdir -p release
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; ext=""; [ $$os = windows ] && ext=".exe"; \
		echo "building $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o release/gbi_$${os}_$${arch}$$ext ./cmd/gbi || exit 1; \
	done
	go run ./cmd/gbi dev sign release/gbi_*
	@ls -la release
