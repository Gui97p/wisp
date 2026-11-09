# Benchmarks

The same six programs written in Wisp, C, Go, Zig and Rust, to follow how fast the compiler's code is as optimizations land.

    make bench                      everything, 5 runs plus a warmup
    make bench ARGS="--quick"       3 runs, no warmup
    make bench ARGS="--only fib,sieve --variants wisp,c-O0"
    make bench ARGS="--compare last --label inline"
    python3 benchmark/bench.py --list

| program | what it stresses |
|---|---|
| call_loop | a call per iteration, with a dependency that stops the C optimizer from folding the loop |
| fib | recursion, many calls and returns |
| sieve | byte array in memory, strided writes |
| matmul | nested loops, float arithmetic, indexing |
| collatz | integer division and modulo by constants |
| bytes_hash | byte loads, 64 bit multiply, xorshift |
| std_chars | `strings.isDigit`, `isAlpha`, `isSpace`, `isUpper`, `isHex` over a 4 MB buffer |
| std_fmt | `conv.itoa`, `conv.atoi` and `fmt.sprintf` in a loop |
| std_mem | `mem.alloc`, `fill`, `copy` and `free` of small blocks |
| std_math | `math.abs`, `clamp`, `fabs` and `isNaN` in a hot loop |

The four `std_*` programs exist only in Wisp: they follow what the standard library costs from one commit to the next, so the other languages are skipped. Run them with `--only std_chars,std_fmt,std_mem,std_math --variants wisp`.

Every program prints one line, and `expected.txt` holds what it must print: a variant that prints something else is reported as `WRONG OUTPUT`.

Variants: `wisp`, `c-O0`, `c-O2` (the `CC` variable picks the compiler), `go`, `zig-debug`, `zig-fast`, `rust-O0`, `rust-O2`. A variant whose tool is not installed is skipped. The Zig and Rust sources were written without a compiler at hand: fix them if they do not build.

For each program and variant the runner reports build time, binary size, min / median / stdev of the wall time, CPU time, peak memory (from a separate run, so it does not disturb the timing) and the ratio against `--ref` (default `c-O0`) and `--ref2` (default `c-O2`). The summary is the geometric mean of those ratios.

Each run is saved in `results/` as JSON with the git revision, the machine and the tool versions, and copied to `results/latest.json`. `--compare last` (or a file name) prints the change in median time against a saved run. Work in this order when optimizing: run with `--label before`, change the compiler, run with `--compare last --label after`.

To add a program, create `programs/<name>/` with a `main.<ext>` per language, a `wisp.toml` for the Wisp one, and an `expected.txt`. To add a variant, add a `Variant` to `VARIANTS` in `bench.py` (a future `wisp -O` mode goes there).
