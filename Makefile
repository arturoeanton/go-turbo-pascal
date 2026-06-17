# BPGo / go-turbo-pascal

BIN := bin

.PHONY: all build test tools pasrun pls pdap fmt clean

all: build test

build:
	go build ./...

test:
	go test ./...

fmt:
	gofmt -w internal pkg cmd

# Comparative benchmarks (vmpas vs goja) live in a separate module so goja
# never enters the main module's dependencies.
bench:
	cd internal/bench && go test ./... -bench . -benchmem -run '^$$'

# Build the user-facing tools into ./bin.
tools: pasrun pls pdap

pasrun:
	go build -o $(BIN)/pasrun ./cmd/pasrun

pls:
	go build -o $(BIN)/pls ./cmd/pls

pdap:
	go build -o $(BIN)/pdap ./cmd/pdap

clean:
	rm -rf $(BIN)
