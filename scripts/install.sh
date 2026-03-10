#!/bin/sh
# scripts/install.sh — Install nexus-cli and nexus-daemon binaries.
# Usage: curl -sSfL https://raw.githubusercontent.com/el-j/nexus-orchestrator/main/scripts/install.sh | sh
#
# Environment variables:
#   NEXUS_INSTALL_DIR  — Override install directory (default: /usr/local/bin or ~/.local/bin)
#   NEXUS_VERSION      — Install a specific version tag (default: "latest")

set -eu

GITHUB_REPO="el-j/nexus-orchestrator"
BINARIES="nexus-cli nexus-daemon"
OPTIONAL_BINARIES="nexus-submit"
TMPDIR_PREFIX="nexus-install"

# --- helpers ----------------------------------------------------------------

log()   { printf '[nexus] %s\n' "$*"; }
warn()  { printf '[nexus] WARNING: %s\n' "$*" >&2; }
fatal() { printf '[nexus] ERROR: %s\n' "$*" >&2; exit 1; }

cleanup() {
    if [ -n "${TMPDIR_INSTALL:-}" ] && [ -d "${TMPDIR_INSTALL}" ]; then
        rm -rf "${TMPDIR_INSTALL}"
    fi
}
trap cleanup EXIT

# --- platform detection -----------------------------------------------------

detect_os() {
    case "$(uname -s)" in
        Linux|linux)   echo "linux" ;;
        Darwin|darwin) echo "darwin" ;;
        CYGWIN*|MINGW*|MSYS*|Windows_NT)
            fatal "Windows is not supported by this installer. Download binaries manually from https://github.com/${GITHUB_REPO}/releases" ;;
        *) fatal "Unsupported operating system: $(uname -s)" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64)  echo "arm64" ;;
        *) fatal "Unsupported architecture: $(uname -m)" ;;
    esac
}

# --- download helper --------------------------------------------------------

has_cmd() { command -v "$1" >/dev/null 2>&1; }

download() {
    # $1 = url, $2 = output path
    if has_cmd curl; then
        curl -sSfL -o "$2" "$1"
    elif has_cmd wget; then
        wget -q -O "$2" "$1"
    else
        fatal "Neither curl nor wget found. Please install one of them."
    fi
}

# --- checksum verification --------------------------------------------------

verify_checksum() {
    # $1 = expected checksum, $2 = file path
    if has_cmd sha256sum; then
        actual=$(sha256sum "$2" | awk '{print $1}')
    elif has_cmd shasum; then
        actual=$(shasum -a 256 "$2" | awk '{print $1}')
    else
        warn "Neither sha256sum nor shasum found — skipping checksum verification."
        warn "*** UNVERIFIED INSTALL — integrity of the downloaded archive was NOT checked ***"
        return 0
    fi

    if [ "$actual" != "$1" ]; then
        fatal "Checksum mismatch for $(basename "$2")\n  expected: $1\n  actual:   $actual"
    fi
    log "Checksum verified: $(basename "$2")"
}

# --- resolve version --------------------------------------------------------

resolve_version() {
    version="${NEXUS_VERSION:-latest}"
    if [ "$version" = "latest" ]; then
        # GitHub redirects /latest to the actual tag; follow the redirect and extract.
        if has_cmd curl; then
            version=$(curl -sSfI "https://github.com/${GITHUB_REPO}/releases/latest" 2>/dev/null \
                | grep -i '^location:' | sed 's|.*/tag/||;s/[[:space:]]*$//')
        elif has_cmd wget; then
            version=$(wget --spider -S "https://github.com/${GITHUB_REPO}/releases/latest" 2>&1 \
                | grep -i '^ *location:' | tail -1 | sed 's|.*/tag/||;s/[[:space:]]*$//')
        fi
        if [ -z "$version" ]; then
            fatal "Unable to determine latest release version."
        fi
    fi
    echo "$version"
}

# --- install dir ------------------------------------------------------------

resolve_install_dir() {
    dir="${NEXUS_INSTALL_DIR:-}"
    if [ -n "$dir" ]; then
        echo "$dir"
        return
    fi
    if [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
    else
        echo "${HOME}/.local/bin"
    fi
}

ensure_install_dir() {
    dir="$1"
    if [ ! -d "$dir" ]; then
        mkdir -p "$dir" 2>/dev/null || fatal "Cannot create install directory: $dir\nRun with sudo or set NEXUS_INSTALL_DIR to a writable path."
    fi
    if [ ! -w "$dir" ]; then
        fatal "Install directory is not writable: $dir\nRun with sudo or set NEXUS_INSTALL_DIR to a writable path."
    fi
}

# --- main -------------------------------------------------------------------

main() {
    OS=$(detect_os)
    ARCH=$(detect_arch)
    VERSION=$(resolve_version)
    INSTALL_DIR=$(resolve_install_dir)

    log "Installing nexus-orchestrator ${VERSION} (${OS}/${ARCH})"

    ensure_install_dir "$INSTALL_DIR"

    ARCHIVE="nexus-orchestrator-${OS}-${ARCH}.tar.gz"
    BASE_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}"
    ARCHIVE_URL="${BASE_URL}/${ARCHIVE}"
    CHECKSUM_URL="${BASE_URL}/SHA256SUMS.txt"

    TMPDIR_INSTALL=$(mktemp -d "${TMPDIR:-/tmp}/${TMPDIR_PREFIX}.XXXXXX")

    # Download archive
    log "Downloading ${ARCHIVE_URL}"
    download "$ARCHIVE_URL" "${TMPDIR_INSTALL}/${ARCHIVE}"

    # Download and verify checksum
    log "Downloading checksums"
    if download "$CHECKSUM_URL" "${TMPDIR_INSTALL}/SHA256SUMS.txt" 2>/dev/null; then
        expected=$(grep "${ARCHIVE}" "${TMPDIR_INSTALL}/SHA256SUMS.txt" | awk '{print $1}')
        if [ -n "$expected" ]; then
            verify_checksum "$expected" "${TMPDIR_INSTALL}/${ARCHIVE}"
        else
            fatal "Archive not found in SHA256SUMS.txt — cannot verify integrity of ${ARCHIVE}."
        fi
    else
        fatal "Could not download SHA256SUMS.txt — cannot verify integrity of ${ARCHIVE}."
    fi

    # Extract
    log "Extracting archive"
    tar -xzf "${TMPDIR_INSTALL}/${ARCHIVE}" -C "${TMPDIR_INSTALL}"

    # Install required binaries
    for bin in ${BINARIES}; do
        src="${TMPDIR_INSTALL}/${bin}"
        if [ ! -f "$src" ]; then
            fatal "Expected binary not found in archive: ${bin}"
        fi
        chmod +x "$src"
        mv "$src" "${INSTALL_DIR}/${bin}"
        log "Installed ${INSTALL_DIR}/${bin}"
    done

    # Install optional binaries
    for bin in ${OPTIONAL_BINARIES}; do
        src="${TMPDIR_INSTALL}/${bin}"
        if [ -f "$src" ]; then
            chmod +x "$src"
            mv "$src" "${INSTALL_DIR}/${bin}"
            log "Installed ${INSTALL_DIR}/${bin}"
        fi
    done

    # Verify binaries work
    "$INSTALL_DIR/nexus-cli" --version >/dev/null 2>&1 || fatal "Binary verification failed: nexus-cli is not executable or corrupted"
    "$INSTALL_DIR/nexus-daemon" --version >/dev/null 2>&1 || warn "nexus-daemon could not self-verify (may require CGO runtime)"
    if [ -f "$INSTALL_DIR/nexus-submit" ]; then
        "$INSTALL_DIR/nexus-submit" --version >/dev/null 2>&1 || warn "nexus-submit could not self-verify"
    fi

    # Done
    log ""
    log "Successfully installed nexus-orchestrator ${VERSION}"
    log "  nexus-cli    → ${INSTALL_DIR}/nexus-cli"
    log "  nexus-daemon → ${INSTALL_DIR}/nexus-daemon"
    if [ -f "$INSTALL_DIR/nexus-submit" ]; then
        log "  nexus-submit → ${INSTALL_DIR}/nexus-submit"
    fi
    log ""

    # Check if install dir is in PATH
    case ":${PATH}:" in
        *":${INSTALL_DIR}:"*) ;;
        *)
            warn "${INSTALL_DIR} is not in your PATH."
            log "Add it with:"
            log "  export PATH=\"${INSTALL_DIR}:\$PATH\""
            ;;
    esac
}

main
