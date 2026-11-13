# Wisp

A compiled programming language written in Go. It compiles straight to native code, with low-level control and no garbage collector. The syntax borrows from C, Go and Rust. It is early work and changes as features land.

```wsp
func add(int a, b) int => a + b;

func main() int {
  let total = add(2, 3);

  for i in 1..3 {
    if add(i, total) > 7 {
      emit("big");
    }
  }

  return 0;
}
```

## Learn the language

The [`tour/`](tour) folder is the tutorial: one small commented file per topic, tied together by `main.wsp`. Run it from inside the folder:

```bash
cd tour
wisp run
```

Features a backend cannot generate yet live in [`tour/_pending/`](tour/_pending), and [`tour/_c/`](tour/_c) is a project that calls C.

## Install

You need Go. The assembler and the linker (`nasm`, `lld`, and for Windows the import libraries) come bundled: `make toolchain` runs `scripts/fetch-toolchain.sh`, which downloads them with fixed versions and checksums into `dependencies/` (ignored by git), and `make prod` copies them into the package. Without a bundle `wisp` falls back to `nasm` and `ld.lld` (or `ld`) in the `PATH`. `cc` is only for `--libc` and the C tests on Linux, and `lua` 5.4 for the Lua backend.

```bash
make build         # bin/wisp
make toolchain     # fetch nasm, lld and the Windows import libraries into dependencies/
make prod          # package for Linux into dist/ (a .tar.gz)
make prod-windows  # package for Windows into dist/ (a .zip with wisp.exe)
make install       # package into dist/ and install to /usr/local/wisp (asks for sudo)
make test          # both backends; FILTER=name narrows it
make test-windows  # the Windows executables, run through WSL
make ext           # package the VS Code extension into dist/vscode
```

## Use

```bash
wisp init                  # a native project in this folder
wisp init cc               # a project for CC: Tweaked
wisp init lib              # a library: object files, no main
wisp init target <name>    # add a target of your own, for instance a kernel userspace
wisp build                 # compile
wisp run                   # compile and run
wisp check                 # parse and analyse only
wisp clean                 # remove what a build writes (-n lists only)
wisp env [key]             # what the compiler resolved for this project
wisp targets               # built-in and project targets
wisp doc [module [symbol]] # the public API of a module, with its doc comments
wisp lsp                   # the language server
```

`build` and `check` print one line when they finish. `-q` hides it and `-v` shows every step. `run` prints nothing of its own, so its output can be piped.

The compiler reads every `.wsp` file under the project root, the folder with `wisp.toml`, and looks for `main`. Folders starting with `.` or `_` are skipped.

## Configure

`wisp.toml` is optional, and every key has a flag. [`tour/wisp.toml`](tour/wisp.toml) lists them all with comments.

```toml
os = "linux"                  # the system to build for (and arch = "x64", if the system has several)

[output]
bin = "bin"
```

A project can declare libraries too, `[library.<name>]` with the `modules` they are made of: `wisp build` writes `<name>.a` for them, and a program or library only contains what its entry or modules import. A project can declare several programs, `[program.<name>]`, each with its own `entry`, `name` (of the binary), `link`, `libc`, `os`, `arch`, `output` and `run.env`. `wisp build` builds all of them, `wisp build <name>` one, and `wisp run <name>` runs one; `default` in `wisp.toml` names the one `wisp run` starts without a name. A program can adjust `name`, `link`, `libc`, `output` and `run.env` for one system with `[program.<name>.<os>]` (or `[program.<name>.<os>.<arch>]`). Output paths accept `{profile}`, `{target}` (the system and architecture, as in `linux-x64`), `{os}`, `{arch}` and `{name}`. A build uses the `debug`, `release` or `fast` profile (`--release`, `--profile`, or `profile` in `wisp.toml`), which decide whether the stack trace and the source snippets of errors are kept, whether the binary is stripped and optimised, and whether the run-time checks stay (index bounds, division by zero, null pointers): `release` is optimised and keeps them, `fast` drops them. The three names can also be used in `#target` and the declaration selectors.

`bin`, `obj` and `asm` are folders: the binary of each program is written to `bin` under its name (the `name` of the program, which defaults to the program's own name). Without `[output]`, files go under folders named after the profile, the system and the architecture: `bin/<profile>/<os>-<arch>/` and `build/<profile>/<os>-<arch>/<name>/`. For Lua the entry script is `<name>.lua` in the same place, with the other modules beside it.

## Backends and targets

| System | Status |
| --- | --- |
| `linux` (x64) | The main backend: NASM assembly, linked with `lld` (or `ld`), System V calling convention. |
| `windows` (x64) | The same programs as a console `.exe`: NASM `win64` objects linked with `lld`, over `kernel32`. See below. |
| `cc` (lua) | Secondary: Lua for CC: Tweaked computers, the stdlib uses its `fs` API. It has floating point and function values, but no `extern`. Numbers are doubles there: integers are exact up to 2^53 and `int64` and `uint64` do not wrap around. |

The `windows` system keeps the Wisp calling convention for Wisp functions and uses Win64 where C is involved: `extern` functions and the ones exported with `as` take their arguments the Windows way, and a function value taken from one of them goes through an adapter. The standard library talks to `kernel32` from plain Wisp (`std/sys`), `path` follows the system (drive letters, UNC paths, both separators on input and `\` on output), paths longer than 260 characters work, the console gets UTF-16 and a Windows console needs no setup. The program has 8 MB of stack and no `--libc`. The compiler runs on Linux, where `--os windows` cross-compiles, and on Windows, from the zip made by `make prod-windows`. The test suite runs on Linux and executes the Windows programs through WSL (`make test-windows`, or `tests/run.sh windows`).

A system of your own is a `[targets.<name>]` block with an architecture, a start file and a runtime file; the name is the system's name, the one `#target` and `--os` use. A system says how to build for it; where the result goes belongs to the program (`[output]` and `[program.<name>.output]`). `wisp init target <name>` writes both, with comments that say what each routine must do.

## Editor support

`wisp lsp` is the language server, and [`vscode/`](vscode) is a VS Code extension that uses it, with highlighting, snippets and an icon. The server also covers `wisp.toml`.

## Versioning

The compiler version lives in [`internal/version/VERSION`](internal/version/VERSION) and everything reads it. While the major is 0, a minor adds features and may break things. Each minor has a codename, a will-o'-the-wisp from a different culture.

## Layout

```text
cmd/main/            command line
internal/            lexer, parser, ast, analyser, module, diag, lsp, version
internal/target/     the x64 and Lua backends, and the x64 start stub and runtime
std/                 standard library
tour/                the tutorial
tests/               test generator (gen) and runner (run.sh)
scripts/             fetch-toolchain.sh, which builds dependencies/ (nasm, lld, import libraries)
vscode/              VS Code extension
```
