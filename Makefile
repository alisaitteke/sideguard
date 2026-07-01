.PHONY: build test lint run-daemon clean

BINARY := bin/vibeguard
PKG := ./cmd/vibeguard
VERSION ?= dev

build:
	go build -ldflags "-X github.com/alisaitteke/vibeguard/cmd/vibeguard/cmd.Version=$(VERSION)" -o $(BINARY) $(PKG)

test:
	go test ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else echo "golangci-lint not installed; skipping"; fi

run-daemon:
	go run $(PKG) daemon start

clean:
	rm -rf bin/
