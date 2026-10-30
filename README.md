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

You need Go, plus `nasm` and `ld` for x86-64, `cc` for `--libc` and the C tests, and `lua` 5.4 for the Lua backend.

```bash
make build         # bin/wisp
make install       # package into dist/ and install to /usr/local/wisp (asks for sudo)
make test          # both backends; FILTER=name narrows it
make ext           # package the VS Code extension into dist/vscode
```

## Use

```bash
wisp init                  # a native project in this folder
wisp init lua              # a project that targets Lua
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
target = "linux_x64"          # the default target

[output]
bin = "bin/app"
```

Without `[output]`, files go under a folder named after the target: `bin/<target>/`, `build/<target>/` and `dist/<target>/`.

## Backends and targets

| Target | Status |
| --- | --- |
| `linux_x64` | The main backend: NASM assembly, linked with `ld`, System V calling convention. Floating point and function values are still missing. |
| `lua` | Secondary. It has floating point and function values, but no pointers, `extern` or `malloc`. |

A target of your own is a `[targets.<name>]` block with a start file and a runtime file. `wisp init target <name>` writes both, with comments that say what each routine must do. A block named after a built-in target adjusts it.

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
vscode/              VS Code extension
```
