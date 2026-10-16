# Wisp

A compiled programming language written in Go.

Wisp is a personal language project focused on simplicity, readability and direct compilation to native code, with low-level control and no garbage collector. The design takes inspiration from C, Go and Rust while experimenting with its own control-flow constructs and syntax.

The language is still in its early stages and evolves as features are implemented and tested.

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

## The language

* Functions, multiple return values, variadic parameters, methods and function literals
* Structs, enums and nominal type aliases. An enum member is written `Color.Green`, or `.Green` where the enum is already known (assignment, argument, return, comparison, `case`)
* Integer types that wrap at their width. `int` and `uint` are 64 bits wide, and `float` is another name for `float64`, in the same way. A literal with a decimal point is a `float`, and `%` is only defined for integers. Lossless widening is implicit (`int8` to `int64`, an enum to `int`); every other conversion names the type, as in `int8(x)`
* Fixed-size arrays, slices (views that share the elements), strings and maps. Indexing arrays, slices and strings is bounds-checked and panics on a violation
* Pointers with address-of, dereference, arithmetic, indexing and slicing, and manual memory with `malloc`, `realloc` and `free`
* Global variables with constant initializers
* `let` inference next to explicit types, and zero-value declarations
* `if`, `switch` (statement and expression), `for` ranges with steps, and `loop` in its while, do-while and infinite forms
* Labels for `break` and `continue`
* Error handling: a `!` before the return list marks a function that can fail, `Error(Member)` raises, a postfix `!` passes an error up, `??` supplies a fallback, and `let v, e = f()` destructures (a `switch` can sort the failures out)
* Modules with `import`, `export` and re-exports
* `native` blocks for embedding backend-specific code, and `extern` functions for calling C (and `export ... as` for being called from C)

The [`example/`](example) folder is a guided tour: one small file per topic, written as code and comments, tied together by `main.wsp`. Run it with `wisp run` from inside that folder. Features a backend does not generate yet live in [`example/_pending/`](example/_pending), and calling C has a project of its own in [`example/_c/`](example/_c).

## Backends

| Backend | Command | Status |
| --- | --- | --- |
| x86-64 (NASM, ELF) | `wisp build`, `wisp run` | The main backend. Covers the language except floating point, maps, function values and variadic parameters, which are still missing. |
| Lua | `wisp build --target lua`, `wisp run --target lua` | Secondary, kept alongside the main one. It covers floating point, maps, function values and variadic parameters, but has no pointers, `malloc`, `free`, `sizeof`, `extern`, or conversions between bytes and strings. |

The x86-64 backend emits NASM assembly, assembles it with `nasm` and links with `ld`, following the System V calling convention so the object files can be linked against C. By default binaries are static and do not use libc: a small start stub calls `main` and exits with its return value, and it is only added when `wisp build --bin` links, so the `.o` files stay linkable by any toolchain. [C interop](#c-interop) covers the cases where you want the C library.

## Usage

```bash
wisp init                  # native project for the host target
wisp init lua              # project that targets Lua
wisp init lib              # library project: object files, no main
wisp init target <name>    # add a target of your own (kernel userspace), creating the project if needed
wisp build                 # compile for the target (x86-64 for the host by default)
wisp run                   # build, then run the result
wisp run --target lua      # transpile to Lua and run it
wisp cache dir             # where the standard library objects are cached
wisp cache clean           # empty that cache (--old keeps what this compiler uses)
wisp version               # compiler version, codename and build info
wisp debug <lexer|parser|analyser> <file.wsp>
wisp lsp                   # start the language server
```

There is no entry file to name: the compiler reads every `.wsp` file under the project root and looks for a function called `main`.

* No `main`: the project is a library. Nothing is linked, and asking for a binary is an error.
* One `main`: it becomes the entry point and is exported automatically.
* Several: set `entry` in `wisp.toml` to pick one.

Directories starting with `.` or `_` are skipped when looking for source files.

`main` can return nothing or an `int`, which becomes the exit code. It can also be fallible (`func main() !` or `func main() !int`): if it ends in an error, the program prints it as `error[domain]: message` with the file and line, and exits with the error's code. A warning tells you when that code is 0, since an unhandled error would then exit as a success.

### Targets

The built-in targets are `linux_x64` and `lua`. A project can define its own, for instance to write a userspace for a kernel of its own, and a single `wisp.toml` can hold as many as you like. A target is data:

```toml
target = "hakura_x64"

[targets.hakura_x64]
arch = "x64"
os = "hakura"
start = "targets/hakura_x64/start.asm"
runtime = "targets/hakura_x64/runtime.asm"
entry = "_start"
link = ["-T", "targets/hakura_x64/link.ld", "-z", "noexecstack"]
```

`wisp init target <name>` writes this block and starter files for you (the files use Linux system calls as placeholders to replace with your kernel's). It works in an empty folder, where it creates the whole project with the new target as its default, and in an existing project, where it adds the target and keeps the default unless you pass `--default`. Then `wisp build --target hakura_x64` uses it.

| Field | Required | Default | What goes in it |
| --- | --- | --- | --- |
| *the table name* | yes | | the target's name: lowercase letters, digits and underscores, starting with a letter. It cannot be `linux`, `x64` or `lua`, which are taken by the built-in targets |
| `arch` | yes | | the instruction set. Only `"x64"` exists for now |
| `os` | no | none | the operating system, a second name that `native` blocks can select (see below). Same character rules as the name |
| `start` | yes | | a path to an assembly file: the program's entry point. It defines the `entry` symbol, calls `main`, and then leaves the program the way this kernel requires, with the value `main` left in `rax` |
| `runtime` | no | none | a path to an assembly file with the routines the compiler calls on its own (the table below) |
| `entry` | no | `_start` | the symbol the linker starts at. When it is not `_start`, it is passed to `ld` as `-e` |
| `link` | no | none | extra arguments for `ld`, such as a linker script (`-T`), placed before the output and the objects |
| `abi` | no | `"sysv"` | the calling convention. Only `"sysv"` exists |
| `format` | no | `"elf64"` | the object format. Only `"elf64"` exists |

Paths are relative to the folder that holds `wisp.toml`, and the linker runs from there. `start` and `runtime` are assembled with `nasm` only when linking, so `--asm` and `--obj` never need them.

### Naming a target

The built-in target `linux_x64` is called after its operating system and architecture, and has `os = "linux"`. Do the same for yours: a kernel called Hakura on x86-64 is `hakura_x64` with `os = "hakura"`. The name then picks the exact target and the `os` is shared by every architecture of that kernel, so a later `hakura_arm64` would reuse the same `native hakura` blocks.

A `native` block can name any of the three selectors of a target, from the narrowest to the widest:

```wsp
native hakura_x64(...) "...";   // this target only
native hakura(...) "...";       // every target whose os is hakura
native x64(...) "...";          // every x86-64 target, including linux_x64
```

A function that carries `native` blocks but none that matches the target being built is an error, so a port that was not done yet is never a silently empty function.

### The runtime

The compiler calls these routines on its own. Only the ones a program uses are linked in, and a missing one is an undefined-symbol error from the linker, so a new kernel can start with the first two. They follow the System V convention and must keep `rbx`, `rbp` and `r12` to `r15`.

| Routine | Used by | Input | What it does |
| --- | --- | --- | --- |
| `__wisp_emit` | `emit` | `rdi` text address, `rsi` length | writes the text and a line feed to the standard output |
| `__wisp_panic` | `panic` and the compiler's checks (index, division, null) | `rdi` message address, `rsi` length | writes `panic: `, the message and a line feed to the error output, then ends the program with a failure status. Never returns |
| `__wisp_error_emit` | a `main` that can fail and ends in an error | `rdi` address of an error record | writes `error[domain]: message`, a line feed and `  at file:line` to the error output, then returns |
| `__wisp_malloc` | `malloc` | `rdi` size in bytes | returns the address in `rax` and `0` in `rdx`; on failure `0` in `rax` and the address of an error record in `rdx`. A negative size fails |
| `__wisp_realloc` | `realloc` | `rdi` address or `0`, `rsi` new size | returns like `malloc`, and keeps the old block when it fails |
| `__wisp_free` | `free` | `rdi` address or `0` | releases the block |

An error record is 64 bytes: the domain as address (offset 0) and length (8), the code (16), the message as address (24) and length (32), the file as address (40) and length (48), and the line (56). The built-in `linux_x64` runtime is a complete example, and its allocator asks the system for memory in 1 MiB chunks, which a kernel would replace with its own page allocation.

### Several kernels in one project

Put one `[targets.name]` block per kernel in the same `wisp.toml`. The top-level `target = "..."` is the default, and `--target` picks another for a single build:

```toml
target = "hakura_x64"

[targets.hakura_x64]
arch = "x64"
os = "hakura"
start = "targets/hakura/start.asm"
runtime = "targets/hakura/runtime.asm"

[targets.kinoko_x64]
arch = "x64"
os = "kinoko"
start = "targets/kinoko/start.asm"
runtime = "targets/kinoko/runtime.asm"
link = ["-T", "targets/kinoko/link.ld"]
```

```bash
wisp build                          # hakura_x64, from `target`
wisp build --target kinoko_x64      # the other kernel
wisp build --target linux_x64       # and the host, from the same sources
```

The source code is shared, and what differs between kernels is written in `native` blocks, one per target or per operating system. The standard library is built once per target (the cache is keyed by it), and if it has no port for your kernel, `--freestanding` leaves it out, and `--std <folder>` points `std/` at a library of your own.

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

Importing cycles are an error, and two exports with the same name in one module are too. Functions, constants, global variables, structs, enums and types can all be exported.

## Standard library

The standard library lives in [`std/`](std) and is imported with the `std/` prefix.

```wsp
import "std/math" as math;
import "std/strings" as strings;
```

The compiler looks for it in a `std` folder next to the `bin` folder that holds the `wisp` executable, which is the layout `make prod` packages.

The `std/` prefix always means the current standard library, and two options change which one that is:

* `--std <path>` (or `std = "path"` in `wisp.toml`) makes `std/` point at another folder. Its contents are treated exactly like the bundled library, so a kernel or another operating system can ship its own.
* `--freestanding` (or `freestanding = true`) builds without one: any `import "std/..."` is an error. Your own modules are unaffected, and can use any other prefix.

The two cannot be combined in the same place. A flag takes priority over `wisp.toml`.

The library is meant to stay small. The compiler only knows the builtins below; things like joining strings or turning a number into text are library code written on top of them.

## Builtins

The compiler knows these, and they need no import. Declaring a function with the same name in your own module shadows the builtin.

| Builtin | What it does |
| --- | --- |
| `emit(text)` | writes a string and a newline to standard output |
| `len(x)` | the length of an array, slice, string, or variadic parameter |
| `panic(message)` | stops the program with `panic: message` and the file, line and column, exit code 1 |
| `malloc(n)` | `n` bytes of heap, as a `*void`; fallible |
| `realloc(p, n)` | resizes a block, keeping its contents; fallible. `realloc(null, n)` is `malloc(n)`. If it fails, the original block is untouched |
| `free(a, b, ...)` | releases any number of pointers, strings or maps. `free(null)` does nothing. Every argument is evaluated before anything is released |
| `sizeof(x)` | the size in bytes of a type or an expression |

`malloc` and `realloc` can fail with `AllocError.OutOfMemory`, which is a built-in enum, so they are used as `let p = malloc(16)!`, with `??`, or destructured like any fallible call. There is no garbage collector, and strings made from `malloc`ed memory are yours to `free`. Freeing something that is not a heap block (a string literal, a slice of a string, a stack variable) is undefined, as in C.

The x86-64 backend implements `emit` as a call to `__wisp_emit`, which the target provides; a target without a runtime reports `target X does not provide emit`. A builtin the x86-64 backend cannot translate yet is an error naming it. The Lua backend translates `emit` and `len` only.

## Pointers and memory

The star is always a prefix of the type (`*int`, `**int`, `*Point`), never written after it.

```wsp
int x = 1;
*int p = &x;
*p = 7;

*int heap = malloc(sizeof(int) * 4)!;
heap[2] = 30;
*int next = heap + 1;       // arithmetic steps in elements, not bytes
int[] view = heap[0:4];     // slicing a pointer gives a span
free(heap);
```

* A pointer to a struct reaches its fields directly (`pp.x`), and `null` is the empty pointer. Dereferencing `null` stops the program with a panic.
* `p + n`, `n + p`, `p - n`, `p += n` and `p++` move by whole elements, `p - q` counts elements between two pointers of the same type, and pointers compare by address. None of this works on a `*void`: convert it to a typed pointer first (the conversion is implicit when you assign it).
* Indexing and slicing a pointer is not bounds-checked, and `p[a:b]` needs both bounds, because a pointer does not know where it ends.
* `*T(x)` converts an integer or another pointer into a `*T`, and `int(p)` or `uint64(p)` turns a pointer into an integer. `*T(x)` reads like a dereference, so writing through a converted address takes two stars: `**int8(address) = 1`.
* `string(bytes)` views a span of `char`, `int8` or `uint8` as a string without copying, and `*int8(s)` is the address of a string's bytes. The latter is only safe to hand to C when the string ends in a zero byte, which string literals do and slices of them do not.

## Global variables

A declaration outside any function is a global, alive for the whole program:

```wsp
int counter = 0;
string name = "wisp";
Settings settings = Settings{retries: 3, verbose: true};
export int shared = 100;
```

The initializer has to be a constant expression (literals, constants, enum members, and arrays or structs built from them), so nothing runs before `main` and the order of declarations does not matter. Without one the global starts zeroed. `export` makes it visible to other modules, which can read and write it.

A global, variable or constant that nothing uses gets a warning unless its name starts with an underscore. Exported globals are never reported.

## C interop

`extern` declares a function defined in C. It has no body, and the linker finds it among the objects you give it:

```wsp
extern func add(int32 a, int32 b) int32;
extern func puts(*int8 text) int32 as c_puts;   // the C name is `puts`, Wisp calls it c_puts
export func scale(int32 v, int32 k) int32 as wisp_scale => v * k;   // C sees `wisp_scale`
```

* `as` follows the same rule as in `import` and `export`: what is on the left exists already and the name on the right is the new one. In an `extern` the left is the C symbol and the right is the name Wisp uses; in an `export func` the left is the Wisp function and the right is the name C sees, which no longer depends on the module's path.
* C only understands integers, `bool`, `char`, floats and pointers, so those are the only types allowed in the signature of an `extern` or of an exported function with a C name, with at most one return value and no `!`. A Wisp `int` is 64 bits wide: use `int32` and the other sized types to match what C declares.
* A function without a body is an error unless it is `extern`.
* A C variadic ends its parameter list with `...`, as in `extern func printf(*int8 format, ...) int32;`. The extra arguments must be integers, `bool`, `char`, floats or pointers, and a string goes in as `*int8(text)`.
* Extern functions are not available on the Lua backend.
* Give the linker your objects with `link = ["lib.o"]` in `wisp.toml` or `--link ./lib.o` (repeatable). They are passed as written, from the project root, after the project's own objects, so archives, `-L` and `-l` work too.

By default the program is static and the C library is not set up, which is enough for C code that does not need it (freestanding code, or functions that only compute or make system calls). To call the C library itself (`printf`, `strlen`, `malloc`, ...), build with `--libc` (or `libc = true`): the program is then linked through the `cc` driver and started by the system's own startup code, which initialises the library and flushes its buffers when `main` returns. A panic still ends the program straight away without flushing C's buffers, the way C's `abort()` does.

[`example/_c/`](example/_c) is a small project that calls C, is called back by C, and shows the build steps.

## Native blocks

```wsp
func abs(int x) int {
  int r;
  native lua(out r, in x) "r = x < 0 and -x or x";
  return r;
}
```

The name after `native` picks the backend or target; blocks for other targets are skipped, so one function can carry an implementation per target. The bindings list the variables the block reads (`in`) and writes (`out`). For x86-64 the code refers to them as `{name}`, which becomes a sized memory operand (`{s.ptr}` and `{s.len}` reach the parts of a string), and `;` separates instructions:

```wsp
native linux_x64(in x, out r) "mov rax, {x}; add rax, rax; mov {r}, rax";
```

A function that has `native` blocks but none for the target being built is an error, so a missing port is never a silently empty function. This applies to every module in the project, used or not.

## Diagnostics

Errors, warnings, notes and help texts are rendered with the source line under them. Warnings never stop a build: a variable, constant or import that is never used (silenced by a leading underscore), a code that exits as success, and unreachable code. A panic prints `panic: message` followed by `at file:line:col`.

## Editor support

`wisp lsp` starts the language server, and [`vscode/`](vscode) is a VS Code extension that uses it, with syntax highlighting, snippets and an icon. The server reports diagnostics as you type and offers completion (including members after a dot), signature help, an outline, rename, and quick fixes for the "did you mean" suggestions. The extension has its own version, separate from the compiler's.

## Building

Requirements:

* Go, to build the compiler
* `nasm` and `ld` (binutils), for the x86-64 backend
* a C compiler driver (`cc`), for `--libc` and the tests that build C code
* `lua` 5.4, for `wisp run --target lua`

```bash
make build         # writes bin/wisp
make prod          # packages bin/ and std/ into dist/
make install       # make prod, then installs it into /usr/local/wisp (asks for sudo)
make ext           # packages the VS Code extension into dist/vscode
make test-x64      # runs the generated test programs on the x86-64 backend, including the C interop ones
make test-lua      # the same programs on the Lua backend
make test          # test-lua and test-x64
```

The test programs are generated by `tests/gen`, so they are rebuilt on every run. `FILTER=name` narrows any of the test targets to the programs whose name contains it.

## Versioning

The compiler version lives in one file, [`internal/version/VERSION`](internal/version/VERSION), and everything on the compiler side reads it: `wisp version` and `wisp --version`, the language server's `serverInfo`, and the Makefile (it asks the compiler, `wisp version --short`). The VS Code extension is versioned on its own, in `vscode/package.json`, since it does not change every time the compiler does.

Releases are `major.minor.patch`. While the major is 0, a new minor brings new language or CLI features (and may break things) and a patch is a fix. Each minor has a codename, a will-o'-the-wisp from a different culture, and a patch keeps the codename of its minor. The commit and build date come from the Go build itself, so they never need to be written down.

To bump the compiler: edit `internal/version/VERSION` and tag.

## Project structure

```text
cmd/main/            command line interface
internal/
├── lexer/           tokens
├── parser/          syntax tree
├── ast/             node definitions
├── analyser/        scopes, types and checks
├── module/          project discovery, imports and wisp.toml
├── version/         the compiler version (a single VERSION file)
├── diag/            error rendering
├── lsp/             language server
└── target/
    ├── asm/         the x86-64 start stub and runtime, as assembly
    ├── lua/         Lua backend
    └── x64/         x86-64 backend
std/                 standard library
example/             guided tour of the language
tests/               test generator (gen) and the runner (run.sh)
vscode/              VS Code extension (syntax, snippets, icon)
```

## Goals

* Keep the language simple and predictable
* Prioritize readability over clever syntax
* Build a complete compiler from scratch
* Learn compiler and language design through implementation

The language is developed incrementally, with decisions driven by real implementation experience rather than extensive upfront design.
