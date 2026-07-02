.PHONY: build test lint run-daemon clean tray-app site-dev

BINARY := bin/sideguard
PKG := ./cmd/sideguard
VERSION ?= dev

# Systray (sideguard tray) requires CGO for github.com/getlantern/systray.
CGO_ENABLED ?= 1

build:
	CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "-X github.com/alisaitteke/sideguard/cmd/sideguard/cmd.Version=$(VERSION)" -o $(BINARY) $(PKG)

# macOS only: build bin/sideguard and package dist/SideGuard Tray.app (LSUIElement, no Dock).
tray-app: build
	@chmod +x scripts/build-tray-macos-app.sh
	@./scripts/build-tray-macos-app.sh

test:
	go test ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else echo "golangci-lint not installed; skipping"; fi

run-daemon:
	go run $(PKG) daemon start

site-dev:
	cd site && npm run dev

clean:
	rm -rf bin/ dist/
