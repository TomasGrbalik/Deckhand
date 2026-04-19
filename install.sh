#!/bin/sh
# Deckhand install script.
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/TomasGrbalik/Deckhand/main/install.sh | sh
#   curl -sSL https://raw.githubusercontent.com/TomasGrbalik/Deckhand/main/install.sh | VERSION=v0.2.0 sh

set -eu

REPO="TomasGrbalik/Deckhand"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-}"

err() {
	printf 'error: %s\n' "$1" >&2
	exit 1
}

info() {
	printf '%s\n' "$1"
}

need_cmd() {
	command -v "$1" >/dev/null 2>&1 || err "required command not found: $1"
}

need_cmd curl
need_cmd tar
need_cmd sha256sum
need_cmd uname

OS=$(uname -s)
[ "$OS" = "Linux" ] || err "unsupported OS: $OS (only Linux is supported; use 'go install' for other platforms)"

RAW_ARCH=$(uname -m)
case "$RAW_ARCH" in
	x86_64|amd64) ARCH="amd64" ;;
	aarch64|arm64) ARCH="arm64" ;;
	*) err "unsupported architecture: $RAW_ARCH (use 'go install github.com/TomasGrbalik/Deckhand/cmd/deckhand@latest' or download manually)" ;;
esac

if [ -z "$VERSION" ]; then
	info "Resolving latest release..."
	VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
		| grep '"tag_name"' \
		| head -n 1 \
		| cut -d'"' -f4)
	[ -n "$VERSION" ] || err "failed to resolve latest version"
fi

# Strip leading 'v' for artifact filenames (GoReleaser names files without 'v').
VERSION_NUM="${VERSION#v}"
TARBALL="deckhand_${VERSION_NUM}_linux_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

TMP=$(mktemp -d 2>/dev/null || mktemp -d -t deckhand)
trap 'rm -rf "$TMP"' EXIT INT TERM

info "Downloading ${TARBALL}..."
curl -fsSL -o "${TMP}/${TARBALL}" "${BASE_URL}/${TARBALL}" \
	|| err "failed to download ${TARBALL} (is ${VERSION} a valid release?)"

info "Downloading checksums.txt..."
curl -fsSL -o "${TMP}/checksums.txt" "${BASE_URL}/checksums.txt" \
	|| err "failed to download checksums.txt"

info "Verifying SHA256..."
(
	cd "$TMP"
	grep " ${TARBALL}\$" checksums.txt > checksum.expected \
		|| { printf 'error: no checksum entry for %s\n' "$TARBALL" >&2; exit 1; }
	sha256sum -c checksum.expected >/dev/null \
		|| { printf 'error: checksum mismatch for %s\n' "$TARBALL" >&2; exit 1; }
)

info "Extracting..."
tar -xzf "${TMP}/${TARBALL}" -C "$TMP" deckhand

TARGET="${INSTALL_DIR}/deckhand"
if [ -w "$INSTALL_DIR" ] || ([ ! -e "$INSTALL_DIR" ] && mkdir -p "$INSTALL_DIR" 2>/dev/null); then
	mv "${TMP}/deckhand" "$TARGET"
	chmod +x "$TARGET"
else
	info "Installing to ${TARGET} (requires sudo)..."
	need_cmd sudo
	sudo mv "${TMP}/deckhand" "$TARGET"
	sudo chmod +x "$TARGET"
fi

info "Installed: $("$TARGET" --version 2>/dev/null || printf 'deckhand %s' "$VERSION")"
info "Location:  $TARGET"
