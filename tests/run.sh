#!/usr/bin/env bash
set -u

WISP="${WISP:-wisp}"
case "$WISP" in
*/*) WISP="$(realpath "$WISP")" ;;
esac
TARGET="${1:-linux_x64}"
FILTER="${2:-}"
ROOT="$(cd "$(dirname "$0")/numeric" && pwd)"
OUT="$(mktemp -d)"
RESULTS="$OUT/results.txt"
: >"$RESULTS"

clean() {
    sed 's/\x1b\[[0-9;]*m//g'
}

for dir in "$ROOT"/*/; do
    name="$(basename "$dir")"
    if [[ -n "$FILTER" && "$name" != *"$FILTER"* ]]; then
        continue
    fi

    if [[ "$TARGET" == "lua" && -f "$dir/x64-only" ]]; then
        continue
    fi

    out="$OUT/$name.out"
    err="$OUT/$name.err"
    if [[ "$TARGET" == "lua" ]]; then
        flags=(--lua "$OUT/lua/$name")
    else
        flags=(--bin "$OUT/bin/$name" --obj "$OUT/obj/$name" --asm "$OUT/asm/$name")
    fi
    (cd "$dir" && timeout 120 "$WISP" run --target "$TARGET" "${flags[@]}") >"$out.raw" 2>"$err"
    code=$?

    clean <"$out.raw" | grep -v '^\[\(lua\|x64\)\]' >"$out"
    clean <"$err" >"$err.clean"

    if grep -q -E '>> error|^Error:|nasm|ld:|goroutine [0-9]+' "$err.clean" && ! grep -q '^FAIL$' "$out"; then
        detail="$(grep -m1 -E '>> error|^Error:|nasm|goroutine|panic:' "$err.clean" | sed 's/^Error: //; s/^>> error: //' | cut -c1-110)"
        printf 'BUILD %s  %s\n' "$name" "$detail" >>"$RESULTS"
        continue
    fi

    if [[ -f "$dir/abort.txt" ]]; then
        want="$(cat "$dir/abort.txt")"
        if diff -q "$out" "$dir/expected.txt" >/dev/null && [[ "$code" -eq 1 ]] && grep -qF -- "$want" "$err.clean"; then
            printf 'PASS  %s\n' "$name" >>"$RESULTS"
        else
            printf 'WRONG %s  abort expected (exit 1, "%s"), got exit %s\n' "$name" "$want" "$code" >>"$RESULTS"
        fi
        continue
    fi

    if diff -q "$out" "$dir/expected.txt" >/dev/null; then
        printf 'PASS  %s\n' "$name" >>"$RESULTS"
        continue
    fi

    fails="$(grep -c '^FAIL$' "$out")"
    missing="$(comm -13 <(sort "$out") <(sort "$dir/expected.txt") | wc -l)"
    if [[ "$code" -ne 0 && "$missing" -gt 0 && "$fails" -eq 0 ]]; then
        printf 'CRASH %s  exit=%s ran %s of %s checks\n' "$name" "$code" "$(wc -l <"$out")" "$(wc -l <"$dir/expected.txt")" >>"$RESULTS"
    else
        printf 'WRONG %s  %s failed, %s missing\n' "$name" "$fails" "$missing" >>"$RESULTS"
    fi
done

cat "$RESULTS"
echo
awk '
BEGIN {
    n = split("int8 int16 int32 int64 uint8 uint16 uint32 uint64", cols, " ")
    for (i = 1; i <= n; i++) isType[cols[i]] = 1
}
{
    status = $1
    dir = $2
    tot[status]++
    k = split(dir, p, "_")
    if (k >= 3 && (p[k] in isType)) {
        row = p[1]
        for (i = 2; i < k; i++) row = row "_" p[i]
        cell[row, p[k]] = (status == "PASS") ? "." : substr(status, 1, 1)
        rows[row] = 1
    } else {
        manual[++m] = status " " dir
    }
}
END {
    printf "1%-18s", "operation"
    for (i = 1; i <= n; i++) printf " %6s", cols[i]
    printf "\n"
    for (r in rows) {
        printf "2%-18s", r
        for (i = 1; i <= n; i++) printf " %6s", ((r, cols[i]) in cell) ? cell[r, cols[i]] : "-"
        printf "\n"
    }
    print "3"
    print "3cells: . pass   W wrong output   B build error   C crash"
    print "3"
    for (i = 1; i <= m; i++) print "4" manual[i]
    print "5"
    printf "5TOTAL  PASS %d  WRONG %d  BUILD %d  CRASH %d\n", tot["PASS"], tot["WRONG"], tot["BUILD"], tot["CRASH"]
}' "$RESULTS" | sort | cut -c2-
echo "details: $OUT"
[[ "$(grep -vc '^PASS' "$RESULTS")" -eq 0 ]]
