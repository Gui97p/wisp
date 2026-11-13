#!/usr/bin/env bash
set -euo pipefail

MINGW_VERSION=20261104
MINGW_FILE="llvm-mingw-${MINGW_VERSION}-ucrt-ubuntu-22.04-x86_64.tar.xz"
MINGW_URL="https://github.com/mstorsjo/llvm-mingw/releases/download/${MINGW_VERSION}/${MINGW_FILE}"
MINGW_SHA256=919380403de87dca3c581c2053230cd298af5e891286628ba0613b7fed370926

NASM_VERSION=3.02
NASM_FILE="nasm-${NASM_VERSION}.tar.gz"
NASM_URL="https://www.nasm.us/pub/nasm/releasebuilds/${NASM_VERSION}/${NASM_FILE}"
NASM_SHA256=f504227b2f529e658d41629075f0503b38d67d790af345f34eba4af60c6a5998

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/dependencies/toolchain/linux-amd64"
CACHE="$ROOT/dependencies/cache"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
mkdir -p "$CACHE"

fetch() {
    local url="$1" file="$2" sha="$3"
    if [ -f "$CACHE/$file" ] && echo "$sha  $CACHE/$file" | sha256sum -c - >/dev/null 2>&1; then
        echo "using cached $file"
    else
        echo "downloading $file"
        curl -fsSL -o "$CACHE/$file" "$url"
        echo "$sha  $CACHE/$file" | sha256sum -c - >/dev/null || {
            rm -f "$CACHE/$file"
            echo "checksum mismatch for $file" >&2
            exit 1
        }
    fi
}

pick_cc() {
    if echo 'int main(void){return 0;}' | cc -static -x c - -o "$WORK/probe" 2>/dev/null; then
        echo cc
    elif command -v zig >/dev/null 2>&1; then
        echo "zig cc -target x86_64-linux-musl"
    else
        echo "no way to link statically: install glibc-static or zig" >&2
        exit 1
    fi
}

rm -rf "$OUT"
mkdir -p "$OUT/bin" "$OUT/lib" "$OUT/windows-x64/lib"

fetch "$MINGW_URL" "$MINGW_FILE" "$MINGW_SHA256"
python3 -I - "$CACHE/$MINGW_FILE" "$OUT" <<'PY'
import os, sys, tarfile

archive, out = sys.argv[1], sys.argv[2]
wanted = {
    "bin/lld": "bin/lld",
    "bin/ld.lld": "bin/ld.lld",
    "bin/lld-link": "bin/lld-link",
    "bin/llvm-ar": "bin/llvm-ar",
    "bin/llvm-ranlib": "bin/llvm-ranlib",
    "lib/libLLVM.so.23.1": "lib/libLLVM.so.23.1",
    "LICENSE.TXT": "LICENSE-llvm-mingw.txt",
}
for lib in ("kernel32", "ntdll", "advapi32", "bcrypt"):
    wanted["x86_64-w64-mingw32/lib/lib%s.a" % lib] = "windows-x64/lib/lib%s.a" % lib
tar = tarfile.open(archive, "r:xz")
members = []
for m in tar:
    rel = m.name.split("/", 1)[1] if "/" in m.name else ""
    if rel in wanted:
        members.append((m, wanted[rel]))
found = {dest for _, dest in members}
missing = set(wanted.values()) - found
if missing:
    sys.exit("missing in the archive: " + ", ".join(sorted(missing)))
for m, dest in members:
    m.name = dest
    tar.extract(m, out, filter="data")
PY

fetch "$NASM_URL" "$NASM_FILE" "$NASM_SHA256"
CC_CMD="$(pick_cc)"
tar -C "$WORK" -xzf "$CACHE/$NASM_FILE"
(
    cd "$WORK/nasm-${NASM_VERSION}"
    CC="$CC_CMD" CFLAGS="-O2 -Wno-date-time" ./configure LDFLAGS=-static >/dev/null
    make -j"$(nproc)" nasm >/dev/null
    strip nasm 2>/dev/null || true
    cp nasm "$OUT/bin/nasm"
    cp LICENSE "$OUT/LICENSE-nasm.txt"
)

echo "toolchain ready in $OUT"
ls -l "$OUT/bin"
