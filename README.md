# Wisp

A compiled programming language written in Go.

Wisp is a personal language project focused on simplicity, readability and direct compilation to native code, with low-level control and no garbage collector. The design takes inspiration from C, Go and Rust while experimenting with its own control-flow constructs and syntax.

The language is still in its early stages and evolves as features are implemented and tested.

```wsp
func add(int a, b) int => a + b;

func main() {
  let total = add(2, 3);

  for i in 1..3 {
    emitf("%d + %d = %d", i, total, add(i, total));
  }
}
```

## The language

* Functions, multiple return values, variadic parameters, methods and function literals
* Structs, enums and nominal type aliases
* Fixed-size arrays, slices, maps and pointers
* `let` inference next to explicit types, and zero-value declarations
* `if`, `switch` (statement and expression), `for` ranges with steps, and `loop` in its while, do-while and infinite forms
* Labels for `break` and `continue`
* Error handling with `!T`, `!expr` and `??`
* Modules with `import`, `export` and re-exports
* `native` blocks for embedding backend-specific code

The [`example/`](example) folder is a guided tour: one small file per topic, written as code and comments, tied together by `main.wsp`.

## Backends

| Backend | Command | Status |
| --- | --- | --- |
| Lua | `wisp lua` | The most complete. Does not cover pointers or `malloc` yet. |
| x86-64 (NASM, ELF) | `wisp build`, `wisp run` | In progress: integer arithmetic, comparisons, `if`, locals and `native` so far. |

The x86-64 backend emits NASM assembly, assembles it with `nasm` and links with `gcc`, following the System V calling convention so the object files can be linked against C.

## Usage

```bash
wisp init                  # create wisp.toml and a main.wsp
wisp lua --run             # transpile the project to Lua and run it
wisp build                 # compile to a native binary
wisp run                   # build, then run the binary
wisp debug <lexer|parser|analyser> <file.wsp>
wisp lsp                   # start the language server
```

There is no entry file to name: the compiler reads every `.wsp` file under the project root and looks for a function called `main`.

* No `main`: the project is a library. Nothing is linked, and asking for a binary is an error.
* One `main`: it becomes the entry point and is exported automatically.
* Several: set `entry` in `wisp.toml` to pick one.

Directories starting with `.` or `_` are skipped when looking for source files.

### Build outputs

Each stage of the x86-64 pipeline is produced only when asked for, and each takes the path it should be written to:

```bash
wisp build --asm build/asm --obj build/obj --bin bin/app
```

With none of the three given, `wisp build` produces just the binary. Asking only for `--asm` or `--obj` never needs a `main`, which is how you produce `.o` files to link into a C project.

`wisp build` and `wisp run` also take `--target` (for example `linux_x64`), which selects the matching `native` blocks. `wisp lua` takes `--dir` for the output folder.

### `wisp.toml`

Optional. Its presence marks the project root, and every key has a command line counterpart that takes priority:

```toml
entry = "src/main.wsp"
target = "linux_x64"

[output]
bin = "bin/app"
obj = "build/obj"
asm = "build/asm"
lua = "dist"
```

## Modules

Every `.wsp` file is its own module, and its outputs mirror its path: `libs/math.wsp` becomes `libs/math.asm`, `libs/math.o` or `libs/math.lua`.

```wsp
import "libs/math" as m;    // the alias is optional; it defaults to the last segment
```

An import always names a file, with one shortcut: if the path is a directory, it resolves to the file inside that has the same name, so `import "libs/math"` finds `libs/math/math.wsp`. That file can gather its siblings for the outside world:

```wsp
import "libs/math/constants";
export constants;              // m.PI, flattened into this module
export constants as consts;    // m.consts.PI, nested instead
```

Importing cycles are an error, and two exports with the same name in one module are too.

## Standard library

The standard library lives in [`std/`](std) and is imported with the `std/` prefix.

```wsp
import "std/math" as math;
import "std/strings" as strings;
```

The compiler looks for it in a `std` folder next to the `bin` folder that holds the `wisp` executable, which is the layout `make prod` packages.

## Native blocks

```wsp
func abs(int x) int {
  int r;
  native lua(out r, in x) "r = x < 0 and -x or x";
  return r;
}
```

The name after `native` picks the backend or target; blocks for other targets are skipped, so one function can carry an implementation per target. The bindings list the variables the block reads (`in`) and writes (`out`). For x86-64 the code refers to them as `{name}`, which becomes a sized memory operand, and `;` separates instructions:

```wsp
native linux_x64(in x, out r) "mov rax, {x}; add rax, rax; mov {r}, rax";
```

## Building

Requirements:

* Go, to build the compiler
* `nasm` and `gcc`, for the x86-64 backend
* `lua` 5.4, for `wisp lua --run`

```bash
make build    # writes bin/wisp
make prod     # packages bin/ and std/ into dist/
```

## Project structure

```text
cmd/main/            command line interface
internal/
├── lexer/           tokens
├── parser/          syntax tree
├── ast/             node definitions
├── analyser/        scopes, types and checks
├── module/          project discovery, imports and wisp.toml
├── diag/            error rendering
├── lsp/             language server
└── target/
    ├── lua/         Lua backend
    └── x64/         x86-64 backend
std/                 standard library
example/             guided tour of the language
vscode/              VS Code extension (syntax, snippets, icon)
```

## Goals

* Keep the language simple and predictable
* Prioritize readability over clever syntax
* Build a complete compiler from scratch
* Learn compiler and language design through implementation

The language is developed incrementally, with decisions driven by real implementation experience rather than extensive upfront design.
