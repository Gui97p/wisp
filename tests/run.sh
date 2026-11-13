#!/usr/bin/env bash
set -u

WISP="${WISP:-wisp}"
case "$WISP" in
*/*) WISP="$(realpath "$WISP")" ;;
esac
PROFILE="${PROFILE:-debug}"
TARGET="${1:-linux}"
FILTER="${2:-}"
ROOT="$(cd "$(dirname "$0")/numeric" && pwd)"
OUT="$(mktemp -d)"
RESULTS="$OUT/results.txt"
export LUA_INIT="@$(cd "$(dirname "$0")/shim" && pwd)/cc.lua"
: >"$RESULTS"

clean() {
    sed 's/\x1b\[[0-9;]*m//g'
}

for dir in "$ROOT"/*/; do
    name="$(basename "$dir")"
    if [[ -n "$FILTER" && "$name" != *"$FILTER"* ]]; then
        continue
    fi

    if [[ -f "$dir/profile-only" && "$(cat "$dir/profile-only")" != "$PROFILE" ]]; then
        continue
    fi

    if [[ -f "$dir/os-only" ]]; then
        effective="$TARGET"
        if [[ "$TARGET" == "cc" ]]; then
            effective="linux"
        fi
        if [[ "$(cat "$dir/os-only")" != "$effective" ]]; then
            continue
        fi
    fi

    if [[ "$TARGET" == "cc" && -f "$dir/x64-only" ]]; then
        continue
    fi

    if [[ "$TARGET" != "cc" && -f "$dir/lua-only" ]]; then
        continue
    fi

    out="$OUT/$name.out"
    err="$OUT/$name.err"
    if [[ "$TARGET" == "cc" ]]; then
        flags=(--profile "$PROFILE" --bin "$OUT/lua/$name")
    else
        flags=(--profile "$PROFILE" --bin "$OUT/bin/$name" --obj "$OUT/obj/$name" --asm "$OUT/asm/$name")
    fi
    envs=()
    if [[ -f "$dir/env.txt" ]]; then
        read -r -a envs <"$dir/env.txt"
    fi

    if [[ "$TARGET" != "cc" ]]; then
        if [[ -f "$dir/lib.c" ]]; then
            mkdir -p "$OUT/c"
            if ! cc -c -O1 -fno-stack-protector -fno-pic -ffreestanding "$dir/lib.c" -o "$OUT/c/$name.o" 2>"$err"; then
                printf 'BUILD %s  cc failed\n' "$name" >>"$RESULTS"
                continue
            fi
            flags+=(--link "$OUT/c/$name.o")
        fi
        if [[ -f "$dir/libc" ]]; then
            flags+=(--libc)
        fi
    fi

    if [[ -f "$dir/library" ]]; then
        lib="$(cat "$dir/library")"
        mkdir -p "$OUT/bin"
        (
            cd "$dir" &&
                timeout 120 "$WISP" build --os "$TARGET" --profile "$PROFILE" --bin "$OUT/bin/$name" >"$out.raw" &&
                cc -no-pie main.c "$OUT/bin/$name/$lib.a" -o "$OUT/bin/$name/prog" &&
                "$OUT/bin/$name/prog" >>"$out.raw"
        ) 2>"$err"
    elif [[ -f "$dir/main.c" ]]; then
        mkdir -p "$OUT/bin"
        (
            cd "$dir" &&
                timeout 120 "$WISP" build --os "$TARGET" --obj "$OUT/obj/$name" --asm "$OUT/asm/$name" >"$out.raw" &&
                cc -no-pie main.c "$OUT/obj/$name"/*.o -o "$OUT/bin/$name" &&
                "$OUT/bin/$name" >>"$out.raw"
        ) 2>"$err"
    else
        (cd "$dir" && env "${envs[@]}" timeout 120 "$WISP" run --os "$TARGET" "${flags[@]}") >"$out.raw" 2>"$err"
    fi
    code=$?

    clean <"$out.raw" | grep -v '^\[\(lua\|x64\)\]' >"$out"
    clean <"$err" >"$err.clean"

    if grep -q -E '>> error|^Error:|nasm|ld:|collect2|goroutine [0-9]+' "$err.clean" && ! grep -q '^FAIL$' "$out"; then
        detail="$(grep -m1 -E '>> error|^Error:|nasm|goroutine|panic:' "$err.clean" | sed 's/^Error: //; s/^>> error\(\[[^]]*\]\)\?: //' | cut -c1-110)"
        printf 'BUILD %s  %s\n' "$name" "$detail" >>"$RESULTS"
        continue
    fi

    if [[ -f "$dir/abort.txt" ]]; then
        want="$(cat "$dir/abort.txt")"
        want_code=1
        [[ -f "$dir/exit.txt" ]] && want_code="$(cat "$dir/exit.txt")"
        unwanted=""
        if [[ -f "$dir/stderr.txt" ]]; then
            while IFS= read -r line; do
                [[ "$line" == '!'* ]] && grep -qF -- "${line:1}" "$err.clean" && unwanted="$line"
            done <"$dir/stderr.txt"
        fi
        if diff -q "$out" "$dir/expected.txt" >/dev/null && [[ "$code" -eq "$want_code" ]] && grep -qF -- "$want" "$err.clean" && [[ -z "$unwanted" ]]; then
            printf 'PASS  %s\n' "$name" >>"$RESULTS"
        else
            printf 'WRONG %s  abort expected (exit %s, "%s"), got exit %s\n' "$name" "$want_code" "$want" "$code" >>"$RESULTS"
        fi
        continue
    fi

    if diff -q "$out" "$dir/expected.txt" >/dev/null; then
        if [[ -f "$dir/stderr.txt" ]]; then
            missing=""
            while IFS= read -r want; do
                [[ -z "$want" ]] && continue
                if [[ "$want" == '!'* ]]; then
                    grep -qF -- "${want:1}" "$err.clean" && missing="unexpected: ${want:1}"
                else
                    grep -qF -- "$want" "$err.clean" || missing="missing: $want"
                fi
            done <"$dir/stderr.txt"
            if [[ -n "$missing" ]]; then
                printf 'WRONG %s  stderr %s\n' "$name" "$missing" >>"$RESULTS"
                continue
            fi
        fi
        if [[ -f "$dir/max-size" ]]; then
            limit="$(cat "$dir/max-size")"
            if [[ "$TARGET" == "windows" && -f "$dir/max-size-windows" ]]; then
                limit="$(cat "$dir/max-size-windows")"
            fi
            exe="$name"
            if [[ "$TARGET" == "windows" ]]; then
                exe="$name.exe"
            fi
            actual="$(stat -c %s "$OUT/bin/$name/$exe" 2>/dev/null || echo 0)"
            if [[ "$actual" -eq 0 || "$actual" -gt "$limit" ]]; then
                printf 'WRONG %s  binary is %s bytes, the limit is %s\n' "$name" "$actual" "$limit" >>"$RESULTS"
                continue
            fi
        fi
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
