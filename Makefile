.PHONY: build test lint run-daemon clean tray-app

BINARY := bin/vibeguard
PKG := ./cmd/vibeguard
VERSION ?= dev

# Systray (vibeguard tray) requires CGO for github.com/getlantern/systray.
CGO_ENABLED ?= 1

build:
	CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "-X github.com/alisaitteke/vibeguard/cmd/vibeguard/cmd.Version=$(VERSION)" -o $(BINARY) $(PKG)

# macOS only: build bin/vibeguard and package dist/VibeGuard Tray.app (LSUIElement, no Dock).
tray-app: build
	@chmod +x scripts/build-tray-macos-app.sh
	@./scripts/build-tray-macos-app.sh

test:
	go test ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else echo "golangci-lint not installed; skipping"; fi

run-daemon:
	go run $(PKG) daemon start

clean:
	rm -rf bin/ dist/
