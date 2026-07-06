GO ?= go
GOCACHE ?= /tmp/jire-go-build
GOMODCACHE ?= /tmp/jire-gomodcache
GOENV := GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE)

.PHONY: run test build fmt tidy

run:
	$(GOENV) $(GO) run ./cmd/jire

test:
	$(GOENV) $(GO) test ./...

build:
	$(GOENV) $(GO) build -o ./jire ./cmd/jire

fmt:
	$(GOENV) $(GO) fmt ./...

tidy:
	$(GOENV) $(GO) mod tidy
