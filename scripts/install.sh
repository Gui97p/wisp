#!/usr/bin/env sh
set -eu

BASE="${WISP_UPDATE_URL:-https://github.com/Gui97p/wisp}"
HOME_DIR="${WISP_HOME:-$HOME/.wisp}"

fail() {
    echo "install: $*" >&2
    exit 1
}

[ "$(uname -s)" = "Linux" ] || fail "this installer is for Linux and WSL; on Windows use install.ps1"
case "$(uname -m)" in
x86_64 | amd64) ;;
*) fail "there is no release for $(uname -m) yet" ;;
esac
command -v curl >/dev/null 2>&1 || fail "curl is needed"
command -v tar >/dev/null 2>&1 || fail "tar is needed"
command -v sha256sum >/dev/null 2>&1 || fail "sha256sum is needed"

TAG="${WISP_VERSION:-}"
if [ -z "$TAG" ]; then
    TAG="$(curl -sSLI -o /dev/null -w '%{url_effective}' "$BASE/releases/latest" | sed 's#.*/##')"
fi
case "$TAG" in
v[0-9]*.[0-9]*.[0-9]*) ;;
*) fail "no release has been published yet" ;;
esac
VERSION="${TAG#v}"
NAME="wisp-$VERSION-linux-amd64.tar.gz"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

echo "downloading wisp $VERSION"
curl -fsSL -o "$WORK/$NAME" "$BASE/releases/download/$TAG/$NAME" || fail "could not download $NAME"
curl -fsSL -o "$WORK/SHA256SUMS" "$BASE/releases/download/$TAG/SHA256SUMS" || fail "could not download SHA256SUMS"
(cd "$WORK" && grep " $NAME\$" SHA256SUMS | sha256sum -c - >/dev/null) || fail "the checksum of $NAME does not match"

mkdir -p "$WORK/out"
tar -xzf "$WORK/$NAME" -C "$WORK/out"
rm -rf "$HOME_DIR.new"
mv "$WORK/out/wisp" "$HOME_DIR.new"
rm -rf "$HOME_DIR"
mv "$HOME_DIR.new" "$HOME_DIR"

echo "wisp $VERSION installed in $HOME_DIR"

case ":$PATH:" in
*":$HOME_DIR/bin:"*) ;;
*)
    case "${SHELL:-}" in
    */zsh) RC="$HOME/.zshrc" ;;
    *) RC="$HOME/.bashrc" ;;
    esac
    LINE="export PATH=\"$HOME_DIR/bin:\$PATH\""
    if ! grep -qsF "$LINE" "$RC"; then
        printf '\n%s\n' "$LINE" >>"$RC"
        echo "added $HOME_DIR/bin to PATH in $RC: open a new terminal to use it"
    fi
    ;;
esac

"$HOME_DIR/bin/wisp" doctor || true
