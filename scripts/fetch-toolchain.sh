#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-linux}"
case "$MODE" in
linux | windows | all) ;;
*)
    echo "usage: $0 [linux|windows|all]" >&2
    exit 2
    ;;
esac

MINGW_VERSION=20261104
MINGW_LINUX_FILE="llvm-mingw-${MINGW_VERSION}-ucrt-ubuntu-22.04-x86_64.tar.xz"
MINGW_LINUX_SHA256=919380403de87dca3c581c2053230cd298af5e891286628ba0613b7fed370926
MINGW_WINDOWS_FILE="llvm-mingw-${MINGW_VERSION}-ucrt-x86_64.zip"
MINGW_WINDOWS_SHA256=552191b5754c49a5a20723276a6fc56541ea1922c4c50b7b669022333fc1b40f
MINGW_BASE="https://github.com/mstorsjo/llvm-mingw/releases/download/${MINGW_VERSION}"

NASM_VERSION=3.02
NASM_FILE="nasm-${NASM_VERSION}.tar.gz"
NASM_SHA256=f504227b2f529e658d41629075f0503b38d67d790af345f34eba4af60c6a5998
NASM_WINDOWS_FILE="nasm-${NASM_VERSION}-win64.zip"
NASM_WINDOWS_SHA256=161d0bfaff53c2f9e9f3e69fd0672323ebabafd1268976a5cec11be92a19aee7
NASM_BASE="https://www.nasm.us/pub/nasm/releasebuilds/${NASM_VERSION}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
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

linux_toolchain() {
    local out="$ROOT/dependencies/toolchain/linux-amd64"
    rm -rf "$out"
    mkdir -p "$out/bin" "$out/lib" "$out/windows-x64/lib"

    fetch "$MINGW_BASE/$MINGW_LINUX_FILE" "$MINGW_LINUX_FILE" "$MINGW_LINUX_SHA256"
    python3 -I - "$CACHE/$MINGW_LINUX_FILE" "$out" <<'PY'
import sys, tarfile

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

    fetch "$NASM_BASE/$NASM_FILE" "$NASM_FILE" "$NASM_SHA256"
    local cc_command
    cc_command="$(pick_cc)"
    tar -C "$WORK" -xzf "$CACHE/$NASM_FILE"
    (
        cd "$WORK/nasm-${NASM_VERSION}"
        CC="$cc_command" CFLAGS="-O2 -Wno-date-time" ./configure LDFLAGS=-static >/dev/null
        make -j"$(nproc)" nasm >/dev/null
        strip nasm 2>/dev/null || true
        cp nasm "$out/bin/nasm"
        cp LICENSE "$out/LICENSE-nasm.txt"
    )

    echo "toolchain ready in $out"
    ls -l "$out/bin"
}

windows_toolchain() {
    local out="$ROOT/dependencies/toolchain/windows-amd64"
    rm -rf "$out"
    mkdir -p "$out/bin" "$out/windows-x64/lib"

    fetch "$MINGW_BASE/$MINGW_WINDOWS_FILE" "$MINGW_WINDOWS_FILE" "$MINGW_WINDOWS_SHA256"
    fetch "$NASM_BASE/win64/$NASM_WINDOWS_FILE" "$NASM_WINDOWS_FILE" "$NASM_WINDOWS_SHA256"

    python3 -I - "$CACHE/$MINGW_WINDOWS_FILE" "$CACHE/$NASM_WINDOWS_FILE" "$out" <<'PY'
import os, sys, zipfile

mingw, nasm, out = sys.argv[1], sys.argv[2], sys.argv[3]
wanted = {
    "bin/ld.lld.exe": "bin/ld.lld.exe",
    "bin/llvm-ar.exe": "bin/llvm-ar.exe",
    "bin/libLLVM-23.dll": "bin/libLLVM-23.dll",
    "bin/libc++.dll": "bin/libc++.dll",
    "bin/libunwind.dll": "bin/libunwind.dll",
    "bin/libwinpthread-1.dll": "bin/libwinpthread-1.dll",
    "LICENSE.TXT": "LICENSE-llvm-mingw.txt",
}
for lib in ("kernel32", "ntdll", "advapi32", "bcrypt"):
    wanted["x86_64-w64-mingw32/lib/lib%s.a" % lib] = "windows-x64/lib/lib%s.a" % lib

def write(zf, info, dest):
    path = os.path.join(out, dest)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "wb") as f:
        f.write(zf.read(info))

z = zipfile.ZipFile(mingw)
found = set()
for info in z.infolist():
    rel = info.filename.split("/", 1)[1] if "/" in info.filename else ""
    if rel in wanted:
        write(z, info, wanted[rel])
        found.add(wanted[rel])
missing = set(wanted.values()) - found
if missing:
    sys.exit("missing in the llvm-mingw archive: " + ", ".join(sorted(missing)))

n = zipfile.ZipFile(nasm)
want_nasm = {"nasm.exe": "bin/nasm.exe", "LICENSE": "LICENSE-nasm.txt"}
found = set()
for info in n.infolist():
    rel = info.filename.split("/", 1)[1] if "/" in info.filename else ""
    if rel in want_nasm:
        write(n, info, want_nasm[rel])
        found.add(want_nasm[rel])
missing = set(want_nasm.values()) - found
if missing:
    sys.exit("missing in the nasm archive: " + ", ".join(sorted(missing)))
PY

    chmod +x "$out"/bin/*.exe
    echo "toolchain ready in $out"
    ls -l "$out/bin"
}

case "$MODE" in
linux) linux_toolchain ;;
windows) windows_toolchain ;;
all)
    linux_toolchain
    windows_toolchain
    ;;
esac
