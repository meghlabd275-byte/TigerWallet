# TigerSmartChain build and test entry points.
#
# Toolchain note: this module depends on go-ethereum v1.13.15, the last release
# that still supports proof-of-authority block sealing. That release uses a
# linkname into runtime.stopTheWorld, which Go 1.23 and later reject by default,
# so every binary link needs -checklinkname=0. Test binaries link the same code
# and need the same flag.

GO        ?= go
GOFLAGS   ?=
LDFLAGS   := -checklinkname=0
BIN       := bin/tscnode

.PHONY: all build test test-short vet fmt clean run-local

all: build

build:
	$(GO) build $(GOFLAGS) -ldflags='$(LDFLAGS)' -o $(BIN) ./cmd/tscnode

test:
	$(GO) test $(GOFLAGS) -ldflags='$(LDFLAGS)' -count=1 ./...

test-short:
	$(GO) test $(GOFLAGS) -ldflags='$(LDFLAGS)' -count=1 -short ./...

vet:
	$(GO) vet $(GOFLAGS) ./...

fmt:
	gofmt -w chain cmd

# Bring up a single-validator development chain on the default ports.
run-local: build
	./$(BIN) init -config devnet.json -chain-id 7711 -validators 1 -supply 1000000000 -force
	./$(BIN) run -config devnet.json -datadir ./devnet-data -mine \
		-validator-key "$$(./$(BIN) account -index 1000 | awk '/private key/{print $$3}')"

clean:
	rm -rf bin devnet-data devnet.json
