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
| Lua | `wisp build --target lua`, `wisp run --target lua` | The most complete. Does not cover pointers or `malloc` yet. |
| x86-64 (NASM, ELF) | `wisp build`, `wisp run` | In progress: integer arithmetic, comparisons, `if`, locals and `native` so far. |

The x86-64 backend emits NASM assembly, assembles it with `nasm` and links with `ld`, following the System V calling convention so the object files can be linked against C. Binaries are static and do not use libc: a small start stub calls `main` and exits with its return value, and it is only added when `wisp build --bin` links, so the `.o` files stay linkable by any toolchain.

## Usage

```bash
wisp init                  # native project for the host target
wisp init lua              # project that targets Lua
wisp init lib              # library project: object files, no main
wisp init target <name>    # add a target of your own (kernel userspace)
wisp build                 # compile for the target (x86-64 for the host by default)
wisp run                   # build, then run the result
wisp run --target lua      # transpile to Lua and run it
wisp debug <lexer|parser|analyser> <file.wsp>
wisp lsp                   # start the language server
```

There is no entry file to name: the compiler reads every `.wsp` file under the project root and looks for a function called `main`.

* No `main`: the project is a library. Nothing is linked, and asking for a binary is an error.
* One `main`: it becomes the entry point and is exported automatically.
* Several: set `entry` in `wisp.toml` to pick one.

Directories starting with `.` or `_` are skipped when looking for source files.

### Targets and pipelines

`wisp build` and `wisp run` work for every target; the target picks the pipeline. It comes from `--target`, then `target = "..."` in `wisp.toml`, then the host x86-64 target. An x86-64 target goes through assembly, object files and linking, a `lua` target writes one `.lua` file per module. Setting `target = "lua"` in `wisp.toml` makes plain `wisp run` run a Lua project.

### Build outputs

Each stage of the x86-64 pipeline is produced only when asked for, and each takes the path it should be written to:

```bash
wisp build --asm build/asm --obj build/obj --bin bin/app
```

With none of the three given, `wisp build` produces just the binary.

Those folders only ever contain your own modules. Standard library modules are built on demand into a cache (`~/.cache/wisp`, or the folder in `WISP_CACHE_DIR`), keyed by the compiler and the contents of `std/`, and `--bin` links them from there. That is the same role `libc` plays for a C program: it is an input of the toolchain, not a file of your project. `wisp cache dir` prints the cache folder, and `wisp cache clean` empties it (`--old` keeps only what the current compiler would use). Asking only for `--asm` or `--obj` never needs a `main`, which is how you produce `.o` files to link into a C project.

`wisp build` and `wisp run` also take `--target` (for example `linux_x64`), which selects the matching `native` blocks. Targets are checked: an unknown or misspelled name, in the flag, in `wisp.toml` or in a `native` block, is an error that suggests the closest known one. A `native` block can name a full target (`linux_x64`), an architecture (`x64`) or an operating system (`linux`). A flag for another pipeline is an error, never silently ignored: `--asm`, `--obj` and `--bin` do not apply to `lua`, and `--lua <dir>`, the output folder of the Lua pipeline (default `dist`), does not apply to x86-64 targets.

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

## Targets

Besides the built-in targets (`linux_x64`, `sine_x64`, `lua`), a project can define its own, for instance to write a userspace for a kernel of its own. A target is data in `wisp.toml`:

```toml
[targets.mykernel]
arch = "x64"
os = "mykernel"
entry = "kstart"
runtime = "mylib/runtime"
start = "targets/start.asm"
link = ["-T", "targets/link.ld"]
```

* `arch` is required and only `x64` exists for now; `abi` (`sysv`) and `format` (`elf64`) default to the only supported values.
* `start` is required: the assembly that calls `main` and leaves the program on this kernel. It is assembled only when linking, and `entry` is the symbol the linker starts at (default `_start`).
* `link` holds extra `ld` arguments. They run from the project root, so relative paths resolve there.
* `runtime` is the module providing what the compiler calls on its own, such as `emit` (default `std/runtime`).

Then `wisp build --target mykernel` uses it, and `native mykernel "..."` blocks apply to it. Names are lowercase letters, digits and underscores, and cannot reuse a built-in target, architecture or operating system name.

A function that has `native` blocks but none for the target being built is an error, so a missing port is never a silently empty function. This applies to every module in the project, used or not.

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

## Builtins

`emit`, `emitf`, `len`, `malloc`, `realloc` and `free` are builtins: the compiler knows them, and they need no import. Declaring a function with the same name in your own module shadows the builtin. The Lua backend translates `emit`, `emitf` and `len`; the x86-64 backend refuses a builtin it cannot translate yet, with an error naming it.

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
* `nasm` and `ld` (binutils), for the x86-64 backend
* `lua` 5.4, for `wisp run --target lua`

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
