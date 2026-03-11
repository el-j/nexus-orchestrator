# =============================================================================
# NexusOrchestrator — Makefile
# =============================================================================
# Usage:
#   make build           native CLI + daemon (current OS/arch)
#   make build-all       cross-compile CLI + daemon for all platforms
#   make test            run all tests
#   make vet             go vet
#   make clean           remove dist/
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
DIST          := dist
MODULE        := nexus-orchestrator

# Build tags that enable the mattn/go-sqlite3 driver
BUILD_FLAGS := -trimpath
LDFLAGS     := -s -w

# Detect host OS for zig target triple selection
UNAME_S := $(shell uname -s 2>/dev/null || echo Windows)

.PHONY: build build-gui build-all test vet clean help \
        build-linux-amd64 build-linux-arm64 \
        build-darwin-amd64 build-darwin-arm64 \
        build-windows-amd64

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
# Desktop GUI (Wails — native only, requires wails CLI)
# ---------------------------------------------------------------------------
build-gui:
	wails build -platform darwin/arm64
	@echo "Built → build/bin/"

# ---------------------------------------------------------------------------
# Cross-compile all platforms (CLI + daemon only; GUI is native-only)
# ---------------------------------------------------------------------------
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64
	@echo "All cross-platform builds complete → $(DIST)/"

build-linux-amd64:
	@mkdir -p $(DIST)/linux_amd64
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		CC="zig cc -target x86_64-linux-musl" \
		CXX="zig c++ -target x86_64-linux-musl" \
		go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST)/linux_amd64/$(BINARY_CLI) ./cmd/nexus-cli/...
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		CC="zig cc -target x86_64-linux-musl" \
		CXX="zig c++ -target x86_64-linux-musl" \
		go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST)/linux_amd64/$(BINARY_DAEMON) ./cmd/nexus-daemon/...
	@echo "Built → $(DIST)/linux_amd64/"

build-linux-arm64:
	@mkdir -p $(DIST)/linux_arm64
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
		CC="zig cc -target aarch64-linux-musl" \
		CXX="zig c++ -target aarch64-linux-musl" \
		go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST)/linux_arm64/$(BINARY_CLI) ./cmd/nexus-cli/...
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
		CC="zig cc -target aarch64-linux-musl" \
		CXX="zig c++ -target aarch64-linux-musl" \
		go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST)/linux_arm64/$(BINARY_DAEMON) ./cmd/nexus-daemon/...
	@echo "Built → $(DIST)/linux_arm64/"

build-darwin-amd64:
	@mkdir -p $(DIST)/darwin_amd64
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
		go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST)/darwin_amd64/$(BINARY_CLI) ./cmd/nexus-cli/...
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
		go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST)/darwin_amd64/$(BINARY_DAEMON) ./cmd/nexus-daemon/...
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
	CGO_ENABLED=1 go test -race -count=1 ./...

test-unit:
	CGO_ENABLED=1 go test -race -count=1 ./internal/core/...

test-cover:
	CGO_ENABLED=1 go test -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report → coverage.html"

vet:
	go vet ./...

# ---------------------------------------------------------------------------
# Housekeeping
# ---------------------------------------------------------------------------
clean:
	rm -rf $(DIST) coverage.out coverage.html

help:
	@echo ""
	@echo "  make build              Native CLI + daemon"
	@echo "  make build-gui          Desktop GUI (Wails, macOS ARM64)"
	@echo "  make build-all          Cross-compile all platforms"
	@echo "  make build-linux-amd64  Linux x86-64 (static, musl)"
	@echo "  make build-linux-arm64  Linux ARM64  (static, musl)"
	@echo "  make build-darwin-amd64 macOS x86-64"
	@echo "  make build-darwin-arm64 macOS ARM64 (Apple Silicon)"
	@echo "  make build-windows-amd64 Windows x86-64"
	@echo "  make test               Run all tests with -race"
	@echo "  make test-unit          Core service tests only"
	@echo "  make test-cover         Tests + HTML coverage report"
	@echo "  make vet                go vet ./..."
	@echo "  make clean              Remove dist/"
	@echo ""
	@echo "GUI desktop app:"
	@echo "  wails dev               Hot-reload dev mode"
	@echo "  wails build             Production Wails binary"
	@echo ""
