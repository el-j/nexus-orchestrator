# =============================================================================
# NexusOrchestrator — Makefile
# =============================================================================
# Usage:
#   make build           native CLI + daemon (current OS/arch)
#   make build-all       cross-compile CLI + daemon for all platforms
#   make test            run all tests
#   make vet             go vet
#   make clean           remove build/ output subdirectories
#   make help            list targets
#
# Cross-compilation notes:
#   go-sqlite3 requires CGO.  For non-native targets this Makefile uses the
#   zig C compiler (https://ziglang.org) as a zero-install cross-toolchain.
#   Install once:  brew install zig   (macOS)  |  apt install zig  (Debian/Ubuntu)
#
#   The desktop GUI (main.go) requires Wails and can only be compiled natively.
#   Run:  wails build   or   go run main.go
# =============================================================================

BINARY_CLI    := nexus-cli
BINARY_DAEMON := nexus-daemon
DIST          := build
DIST_DESKTOP  := $(DIST)/desktop
DIST_VSCODE   := $(DIST)/vscode
MODULE        := nexus-orchestrator
IMAGE         := ghcr.io/el-j/nexus-orchestrator

# ---------------------------------------------------------------------------
# Version stamping — computed from git at build time
# ---------------------------------------------------------------------------
GIT_TAG    := $(shell git describe --tags --match 'v[0-9]*' --abbrev=0 2>/dev/null || echo "v0.0.0")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_DIRTY  := $(shell git diff --quiet 2>/dev/null || echo "-dirty")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION    := $(shell echo "$(GIT_TAG)" | sed 's/^v//')$(GIT_DIRTY)
VSIX_VERSION := $(shell echo "$(GIT_TAG)" | sed 's/^v//')

VERSION_FLAGS := -X 'main.version=$(VERSION)' \
                 -X 'main.commit=$(GIT_COMMIT)' \
                 -X 'main.buildDate=$(BUILD_DATE)'

# Build tags that enable the mattn/go-sqlite3 driver
BUILD_FLAGS := -trimpath
LDFLAGS     := -s -w $(VERSION_FLAGS)
# Windows GUI binary requires -H windowsgui to suppress the console window.
LDFLAGS_WIN_GUI := -s -w -H windowsgui $(VERSION_FLAGS)

# zig 0.15.x musl: pure-Go net/user avoids musl libc symbol issues; -static links sqlite3 statically
LINUX_BUILD_FLAGS := -trimpath -tags netgo,osusergo
LINUX_LDFLAGS     := -s -w -extldflags='-static' $(VERSION_FLAGS)

# Detect host OS for zig target triple selection
UNAME_S := $(shell uname -s 2>/dev/null || echo Windows)

.PHONY: build build-gui build-gui-windows-amd64 build-all test vet lint clean help \
        build-linux-amd64 build-linux-arm64 \
        build-darwin-amd64 build-darwin-arm64 \
        build-windows-amd64 \
        build-frontend build-vscode build-dev dev dev-daemon check-air \
        version version-sync release-alpha release-beta release-rc release \
        docker-build docker-push docker-run \
        nice nice-go nice-frontend

# ---------------------------------------------------------------------------
# Default: native build (CLI + daemon)
# ---------------------------------------------------------------------------
build: vet
	@mkdir -p $(DIST)/native
	CGO_ENABLED=1 go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST)/native/$(BINARY_CLI) ./cmd/nexus-cli/...
	CGO_ENABLED=1 go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST)/native/$(BINARY_DAEMON) ./cmd/nexus-daemon/...
	@echo "Built → $(DIST)/native/"

# ---------------------------------------------------------------------------
# Frontend GUI assets (Vite → build/frontend/) and VS Code extension
# ---------------------------------------------------------------------------

# Build the Vite frontend into build/frontend/  (embedded by Wails at build time)
build-frontend:
	@echo "Building Vite frontend…"
	cd frontend && npm install --prefer-offline --silent && npm run build
	@echo "Built → build/frontend/"

# Compile the VS Code extension bundle and package it as a .vsix
build-vscode:
	@echo "Building VS Code extension…"
	@mkdir -p $(DIST_VSCODE)
	cd vscode-extension && npm install --prefer-offline --silent && \
		node -e "const fs=require('fs'),p='package.json',pkg=JSON.parse(fs.readFileSync(p,'utf8'));pkg.version='$(VSIX_VERSION)';fs.writeFileSync(p,JSON.stringify(pkg,null,2)+'\n');" && \
		npm run build && \
		npx @vscode/vsce package --no-dependencies --out ../$(DIST_VSCODE)/nexus-orchestrator.vsix
	@echo "Built → $(DIST_VSCODE)/nexus-orchestrator.vsix"

# Convenience target: build frontend + VS Code extension (quick pre-release check)
build-dev: build-frontend build-vscode
	@echo ""
	@echo "┌─────────────────────────────────────────┐"
	@echo "│  build-dev complete                     │"
	@echo "│  build/frontend/  — Vite GUI assets     │"
	@echo "│  build/vscode/    — .vsix ready to test │"
	@echo "└─────────────────────────────────────────┘"

# ---------------------------------------------------------------------------
# Hot-reload dev mode — daemon (air) + Vite frontend (HMR) in parallel
# ---------------------------------------------------------------------------

# Ensure air is installed; install it automatically if missing.
check-air:
	@command -v air >/dev/null 2>&1 || { \
		echo "→ Installing air (Go hot-reload watcher)…"; \
		go install github.com/air-verse/air@latest; \
	}

# Start daemon with hot-reload + Vite HMR frontend in parallel.
# Press Ctrl+C once to stop both.
#
#   Daemon   → http://127.0.0.1:63987  (air rebuilds on *.go changes)
#   MCP      → http://127.0.0.1:63988  (JSON-RPC 2.0)
#   Frontend → http://127.0.0.1:63989  (Vite HMR, proxies /api → :63987, /mcp → :63988)
dev: check-air
	@cd frontend && npm install --prefer-offline --silent 2>/dev/null
	@echo ""
	@echo "┌──────────────────────────────────────────────────────────┐"
	@echo "│  nexusOrchestrator — dev mode                            │"
	@echo "│                                                          │"
	@echo "│  Starting daemon via air (hot-reload)…                   │"
	@echo "└──────────────────────────────────────────────────────────┘"
	@echo ""
	@trap 'kill 0' INT; \
	  air & \
	  AIR_PID=$$!; \
	  echo "⏳ Waiting for daemon to become healthy…"; \
	  for i in $$(seq 1 30); do \
	    if curl -sf http://127.0.0.1:63987/api/health >/dev/null 2>&1; then \
	      echo "✓ Daemon healthy on :63987"; \
	      break; \
	    fi; \
	    if [ "$$i" = "30" ]; then \
	      echo "⚠  Daemon did not become healthy after 30s — starting frontend anyway"; \
	    fi; \
	    sleep 1; \
	  done; \
	  echo ""; \
	  echo "┌──────────────────────────────────────────────────────────┐"; \
	  echo "│  All services starting                                   │"; \
	  echo "│                                                          │"; \
	  echo "│  Daemon   → http://127.0.0.1:63987  (HTTP API)           │"; \
	  echo "│  MCP      → http://127.0.0.1:63988  (JSON-RPC 2.0)      │"; \
	  echo "│  Frontend → http://127.0.0.1:63989  (Vite HMR)          │"; \
	  echo "│  Discovery→ http://127.0.0.1:63987/.well-known/nexus.json│"; \
	  echo "│  How-to   → http://127.0.0.1:63987/api/howto             │"; \
	  echo "│                                                          │"; \
	  echo "│  Ctrl+C to stop all                                      │"; \
	  echo "└──────────────────────────────────────────────────────────┘"; \
	  echo ""; \
	  (cd frontend && npm run dev) & \
	  wait

# Backend only — daemon hot-reload without the frontend.
dev-daemon: check-air
	air

# ---------------------------------------------------------------------------
# Desktop GUI (Wails — native only, requires wails CLI)
# ---------------------------------------------------------------------------
build-gui: build-frontend
	@echo "Building Wails desktop application..."
	@mkdir -p $(DIST_DESKTOP)
	@if command -v wails >/dev/null 2>&1; then \
		wails build -clean -ldflags "-s -w -X 'main.version=$(VERSION)' -X 'main.commit=$(GIT_COMMIT)' -X 'main.buildDate=$(BUILD_DATE)'"; \
		cp -r build/bin/* $(DIST_DESKTOP)/; \
		echo "  → $(DIST_DESKTOP)/"; \
	else \
		echo "  ⚠  wails not installed, skipping GUI build"; \
	fi
# NOTE: build/bin/ is used by Wails for its raw output; build/desktop/ is the
# final packaged artifact for distribution.

# Windows GUI build — uses -H windowsgui to suppress the console window.
# Requires wails CLI and a Windows-capable cross-compilation environment.
build-gui-windows-amd64:
	GOOS=windows GOARCH=amd64 \
		wails build -platform windows/amd64 \
		-ldflags "-s -w -H windowsgui -X 'main.version=$(VERSION)' -X 'main.commit=$(GIT_COMMIT)' -X 'main.buildDate=$(BUILD_DATE)'"
	@echo "Built → build/bin/"

# ---------------------------------------------------------------------------
# Cross-compile all platforms (CLI + daemon only; GUI is native-only)
# ---------------------------------------------------------------------------
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 build-frontend build-vscode build-gui
	@echo "All cross-platform builds complete → $(DIST)/"

build-linux-amd64:
	@mkdir -p $(DIST)/linux_amd64
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		CC="zig cc -target x86_64-linux-musl" \
		CXX="zig c++ -target x86_64-linux-musl" \
		go build $(LINUX_BUILD_FLAGS) -ldflags "$(LINUX_LDFLAGS)" \
		-o $(DIST)/linux_amd64/$(BINARY_CLI) ./cmd/nexus-cli/...
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		CC="zig cc -target x86_64-linux-musl" \
		CXX="zig c++ -target x86_64-linux-musl" \
		go build $(LINUX_BUILD_FLAGS) -ldflags "$(LINUX_LDFLAGS)" \
		-o $(DIST)/linux_amd64/$(BINARY_DAEMON) ./cmd/nexus-daemon/...
	@echo "Built → $(DIST)/linux_amd64/"

build-linux-arm64:
	@mkdir -p $(DIST)/linux_arm64
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
		CC="zig cc -target aarch64-linux-musl" \
		CXX="zig c++ -target aarch64-linux-musl" \
		go build $(LINUX_BUILD_FLAGS) -ldflags "$(LINUX_LDFLAGS)" \
		-o $(DIST)/linux_arm64/$(BINARY_CLI) ./cmd/nexus-cli/...
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
		CC="zig cc -target aarch64-linux-musl" \
		CXX="zig c++ -target aarch64-linux-musl" \
		go build $(LINUX_BUILD_FLAGS) -ldflags "$(LINUX_LDFLAGS)" \
		-o $(DIST)/linux_arm64/$(BINARY_DAEMON) ./cmd/nexus-daemon/...
	@echo "Built → $(DIST)/linux_arm64/"

build-darwin-amd64:
	@mkdir -p $(DIST)/darwin_amd64
	# macOS cross-arch: pass -arch x86_64 to the native clang via CGO_CFLAGS/CGO_LDFLAGS.
	# Requires Xcode Command Line Tools (xcrun sdk present). Skips gracefully if SDK absent.
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
		CGO_CFLAGS="-arch x86_64" \
		CGO_LDFLAGS="-arch x86_64" \
		go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST)/darwin_amd64/$(BINARY_CLI) ./cmd/nexus-cli/... || \
		(echo "NOTE: build-darwin-amd64 requires Xcode SDK with x86_64 support — skipped."; exit 0)
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
		CGO_CFLAGS="-arch x86_64" \
		CGO_LDFLAGS="-arch x86_64" \
		go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST)/darwin_amd64/$(BINARY_DAEMON) ./cmd/nexus-daemon/... || \
		(echo "NOTE: build-darwin-amd64 requires Xcode SDK with x86_64 support — skipped."; exit 0)
	@echo "Built → $(DIST)/darwin_amd64/"

build-darwin-arm64:
	@mkdir -p $(DIST)/darwin_arm64
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
		go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST)/darwin_arm64/$(BINARY_CLI) ./cmd/nexus-cli/...
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
		go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST)/darwin_arm64/$(BINARY_DAEMON) ./cmd/nexus-daemon/...
	@echo "Built → $(DIST)/darwin_arm64/"

build-windows-amd64:
	@mkdir -p $(DIST)/windows_amd64
	# NOTE: for the GUI binary (main.go), use LDFLAGS_WIN_GUI (-H windowsgui) to suppress the console window.
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
		CC="zig cc -target x86_64-windows-gnu" \
		CXX="zig c++ -target x86_64-windows-gnu" \
		go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST)/windows_amd64/$(BINARY_CLI).exe ./cmd/nexus-cli/...
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
		CC="zig cc -target x86_64-windows-gnu" \
		CXX="zig c++ -target x86_64-windows-gnu" \
		go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST)/windows_amd64/$(BINARY_DAEMON).exe ./cmd/nexus-daemon/...
	@echo "Built → $(DIST)/windows_amd64/"

# ---------------------------------------------------------------------------
# Test & quality
# ---------------------------------------------------------------------------
test:
	CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 ./...

test-unit:
	CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 ./internal/core/...

test-cover:
	CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report → coverage.html"

vet:
	go vet ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		LINT_VER=$$(golangci-lint version 2>&1 | grep -oE 'v[0-9]+' | head -1); \
		if [ "$$LINT_VER" = "v1" ]; then \
			echo "  ⚠  golangci-lint v1 found but config requires v2 — upgrading…"; \
			go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
		fi; \
	fi
	golangci-lint run ./...

# ---------------------------------------------------------------------------
# Code quality — format, fix, lint everything in one go
# ---------------------------------------------------------------------------

# Go: auto-format + vet + lint with auto-fix
nice-go:
	@echo "── Go: formatting ──"
	@gofmt -w .
	@echo "── Go: vet ──"
	@go vet ./...
	@echo "── Go: lint + auto-fix ──"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		LINT_VER=$$(golangci-lint version 2>&1 | grep -oE 'v[0-9]+' | head -1); \
		if [ "$$LINT_VER" = "v1" ]; then \
			echo "  ⚠  golangci-lint v1 found but config requires v2 — upgrading…"; \
			go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
		fi; \
		golangci-lint run --fix ./...; \
	else \
		echo "  ⚠  golangci-lint not installed — installing v2…"; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
		golangci-lint run --fix ./...; \
	fi
	@echo "── Go: clean ✓ ──"

# Frontend: type-check
nice-frontend:
	@echo "── Frontend: type-check ──"
	@cd frontend && npx vue-tsc --noEmit
	@echo "── Frontend: clean ✓ ──"

# Everything: Go + Frontend
nice: nice-go nice-frontend
	@echo ""
	@echo "┌──────────────────────────────────┐"
	@echo "│  All nice — code is clean ✓      │"
	@echo "└──────────────────────────────────┘"

# ---------------------------------------------------------------------------
# Housekeeping
# ---------------------------------------------------------------------------
clean:
	# Remove generated build outputs — preserves build/darwin/ and build/windows/ (Wails resources)
	rm -rf build/bin build/native build/desktop build/vscode build/docs build/frontend
	rm -rf build/linux_amd64 build/linux_arm64
	rm -rf build/darwin_amd64 build/darwin_arm64
	rm -rf build/windows_amd64
	rm -f coverage.out coverage.html
	rm -f vscode-extension/*.vsix


# ---------------------------------------------------------------------------
# Upgrade all dependencies (Node + Go)
# ---------------------------------------------------------------------------

upgrade-latest:
	@echo "── Upgrading all Node.js packages (ncu) ──"
	@for pkg in package.json frontend/package.json github-action/package.json vscode-extension/package.json docs/package.json; do \
	  if [ -f "$$pkg" ]; then \
	    dir=$$(dirname "$$pkg"); \
	    echo "→ $$pkg"; \
	    (cd "$$dir" && npx -y npm-check-updates && npx -y npm-check-updates -u --doctor); \
	  fi; \
	done
	@echo "── Upgrading Go modules to latest 1.26+ ──"
	@if [ -f go.mod ]; then \
	  go get -u=patch ./...; \
	  go get -u ./...; \
	  go mod tidy; \
	  go mod verify; \
	  echo "✓ Go modules upgraded"; \
	fi
	@echo "── All dependencies checked for latest versions ──"

# ---------------------------------------------------------------------------
# Version display and package sync
# ---------------------------------------------------------------------------
version:
	@echo "$(VERSION) (commit: $(GIT_COMMIT), built: $(BUILD_DATE))"

version-sync:
	@bash scripts/version-sync.sh "$(VERSION)"

# ---------------------------------------------------------------------------
# Release tagging (local) — pushes an annotated git tag
# ---------------------------------------------------------------------------
# Usage:
#   make release-alpha VER=0.10.0    → tags v0.10.0-alpha.1 (next alpha number)
#   make release-beta  VER=0.10.0    → tags v0.10.0-beta.1
#   make release-rc    VER=0.10.0    → tags v0.10.0-rc.1
#   make release       VER=0.10.0    → tags v0.10.0
#
# If VER is omitted the current computed VERSION is used.

release-alpha:
	@bash scripts/release.sh alpha "$(or $(VER),$(VERSION))"

release-beta:
	@bash scripts/release.sh beta "$(or $(VER),$(VERSION))"

release-rc:
	@bash scripts/release.sh rc "$(or $(VER),$(VERSION))"

release:
	@bash scripts/release.sh stable "$(or $(VER),$(VERSION))"

# ---------------------------------------------------------------------------
# Docker — build image locally and push to ghcr.io
# ---------------------------------------------------------------------------
docker-build:
	docker build \
		--build-arg VERSION=$(VSIX_VERSION) \
		--build-arg COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(IMAGE):$(VSIX_VERSION) \
		-t $(IMAGE):latest \
		.
	@echo "Built → $(IMAGE):$(VSIX_VERSION)"

docker-push:
	docker push $(IMAGE):$(VSIX_VERSION)
	docker push $(IMAGE):latest
	@echo "Pushed → $(IMAGE):$(VSIX_VERSION)"

# Quick local smoke-run: mounts ~/.nexus as /data, exposes both ports
docker-run:
	docker run --rm \
		-p 63987:63987 \
		-p 63988:63988 \
		-v "$(HOME)/.nexus:/data" \
		$(IMAGE):$(VSIX_VERSION)

help:
	@echo ""
	@echo "Build:"
	@echo "  make build                  Native CLI + daemon (current OS/arch)"
	@echo "  make build-gui              Desktop GUI (Wails, native)"
	@echo "  make build-gui-windows-amd64 Desktop GUI (Wails, Windows AMD64)"
	@echo "  make build-frontend         Vite GUI assets → build/frontend/"
	@echo "  make build-vscode           VS Code extension bundle + VSIX (version-stamped)"
	@echo "  make build-dev              build-frontend + build-vscode"
	@echo "  make build-all              Cross-compile all platforms"
	@echo "  make build-linux-amd64      Linux x86-64 (static, musl)"
	@echo "  make build-linux-arm64      Linux ARM64  (static, musl)"
	@echo "  make build-darwin-amd64     macOS x86-64"
	@echo "  make build-darwin-arm64     macOS ARM64 (Apple Silicon)"
	@echo "  make build-windows-amd64    Windows x86-64"
	@echo ""
	@echo "Dev:"
	@echo "  make dev                    daemon (air, health-wait) + Vite HMR"
	@echo "  make dev-daemon             daemon hot-reload only"
	@echo ""
	@echo "Test & quality:"
	@echo "  make test                   All tests with -race"
	@echo "  make test-unit              Core service tests only"
	@echo "  make test-cover             Tests + HTML coverage report"
	@echo "  make vet                    go vet ./..."
	@echo "  make lint                   golangci-lint run ./..."
	@echo "  make nice                   Format + vet + lint (Go + Frontend)"
	@echo "  make nice-go                Format + vet + lint (Go only)"
	@echo "  make nice-frontend          Type-check frontend (vue-tsc)"
	@echo ""
	@echo "Versioning:"
	@echo "  make version                Show current version + commit + build date"
	@echo "  make version-sync           Stamp version into all package manifests"
	@echo "  make release-alpha VER=X.Y.Z  Tag v X.Y.Z-alpha.N (auto-increment N)"
	@echo "  make release-beta  VER=X.Y.Z  Tag vX.Y.Z-beta.N"
	@echo "  make release-rc    VER=X.Y.Z  Tag vX.Y.Z-rc.N"
	@echo "  make release       VER=X.Y.Z  Tag vX.Y.Z (stable)"
	@echo ""
	@echo "  Branch → tag workchain:"
	@echo "    feature/** or alpha/**  →  vX.Y.Z-alpha.N"
	@echo "    beta/**                 →  vX.Y.Z-beta.N"
	@echo "    release/**              →  vX.Y.Z-rc.N"
	@echo "    main                    →  vX.Y.Z"
	@echo "    hotfix/**               →  vX.Y.(Z+1)"
	@echo ""
	@echo "Docker (ghcr.io/el-j/nexus-orchestrator):"
	@echo "  make docker-build           Build image locally (nexus-daemon)"
	@echo "  make docker-push            Push to ghcr.io"
	@echo "  make docker-run             Run locally on :63987/:63988 with ~/.nexus as /data"
	@echo "  (CI: .github/workflows/docker.yml auto-builds on every vX.Y.Z tag)"
	@echo ""
	@echo "Other:"
	@echo "  make upgrade-latest         Upgrade all Node.js and Go dependencies to latest"
	@echo "  make clean                  Remove build/ output subdirs"
	@echo "  wails dev                   Wails hot-reload dev mode"
	@echo "  wails build                 Production Wails binary"
	@echo ""
