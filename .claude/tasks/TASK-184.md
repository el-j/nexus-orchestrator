---
id: TASK-184
title: Fix make build-all zig 0.15.x musl cross-compilation
role: devops
planId: PLAN-025
status: todo
dependencies: []
createdAt: 2026-03-12T10:00:00.000Z
---

## Context
`make build-all` is failing with zig 0.15.2 because the musl linker cannot resolve `__errno_location`, `getaddrinfo`, `pthread_*`, and other glibc/musl symbols. This is a known zig 0.15.x breaking change. The fix is to add `-tags netgo,osusergo` (pure-Go DNS/user without libc) and `-extldflags='-static'` (static musl link) to all Linux cross-compilation targets in the Makefile.

## Files to Read
- `Makefile` — all build targets, especially `build-linux-amd64`, `build-linux-arm64`, `build-windows-amd64`, `build-darwin-amd64`

## Implementation Steps

1. **Test baseline failure**: Run `make build-linux-amd64` — confirm current error.

2. **Fix `BUILD_FLAGS` or add a `LINUX_BUILD_FLAGS`** variable at the top of the Makefile:
   ```makefile
   # For zig cc musl cross-compilation: pure-Go net/user avoids musl libc symbol issues
   LINUX_BUILD_FLAGS := -trimpath -tags netgo,osusergo
   LINUX_LDFLAGS     := -s -w -extldflags='-static'
   ```

3. **Update `build-linux-amd64`** to use `LINUX_BUILD_FLAGS` and `LINUX_LDFLAGS`:
   ```makefile
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
   ```

4. **Update `build-linux-arm64`** identically (target `aarch64-linux-musl`).

5. **Update `build-windows-amd64`** to use zig for Windows cross-compilation:
   - Target: `x86_64-windows-gnu`
   - Use `-tags netgo,osusergo` and `CGO_LDFLAGS="-static"` for Windows
   - Windows doesn't need `-extldflags='-static'` in the same way; use `CGO_LDFLAGS="-static-libgcc"` if needed

6. **Darwin amd64 from arm64**: On Apple Silicon building for x86_64 darwin, CGO cross-compilation requires the x86_64 macOS SDK (not available without Rosetta/Xcode). Add a comment documenting this limitation and use `CGO_ENABLED=0` with a fallback for darwin-amd64 cross-compilation, OR conditionally skip if cross-arch darwin build tools are not available. If `CGO_ENABLED=0` is used, sqlite3 won't work — so document this correctly. Best approach: print a warning and skip `build-darwin-amd64` if `GOOS != $(UNAME_S_lower)`.

   Alternative that works: use `-tags netgo,osusergo` with the macOS clang and `-arch x86_64`:
   ```makefile
   build-darwin-amd64:
       @mkdir -p $(DIST)/darwin_amd64
       CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
           CGO_CFLAGS="-arch x86_64" \
           CGO_LDFLAGS="-arch x86_64" \
           go build $(LINUX_BUILD_FLAGS) -ldflags "$(LDFLAGS)" \
           -o $(DIST)/darwin_amd64/$(BINARY_CLI) ./cmd/nexus-cli/...
   ```

7. **Verify**: Run `make build-linux-amd64` then `make build-linux-arm64` — both should exit 0.

8. **Run tests**: `CGO_ENABLED=1 go test -race -count=1 ./...` to confirm no test regressions.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0 (native)
- [ ] `make build-linux-amd64` exits 0 with zig 0.15.x
- [ ] `make build-linux-arm64` exits 0 with zig 0.15.x
- [ ] `make build-windows-amd64` exits 0 OR is documented as requiring Windows toolchain
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0

## Anti-patterns to Avoid
- NEVER remove CGO_ENABLED=1 from the native build — sqlite3 requires it
- NEVER add global `-extldflags='-static'` to native darwin build — it will break dynamic framework linking
- NEVER use CGO_ENABLED=0 silently — document any targets that fall back to pure Go
