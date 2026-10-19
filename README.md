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

The [`example/`](example) folder is a guided tour, one small file per topic. Run it with `wisp run` from inside that folder. Features a backend cannot generate yet live in [`example/_pending/`](example/_pending), and [`example/_c/`](example/_c) is a project that calls C.

## The language

* Functions, multiple return values, variadic parameters, methods and function literals
* Structs, enums and nominal type aliases. `Color.Green` can be written `.Green` where the enum is already known
* Integers wrap at their width. `int` and `uint` are 64 bits, and `float` is `float64`. Widening is implicit (`int8` to `int64`, an enum to `int`), and every other conversion names the type: `int8(x)`
* Arrays, slices, strings and maps. Indexing is bounds-checked and panics on a violation
* Pointers with arithmetic and slicing, and manual memory with `malloc`, `realloc` and `free`
* Globals with constant initializers, `let` inference, zero-value declarations
* `if`, `switch` (statement and expression), `for` ranges with steps, `loop`, labels for `break` and `continue`
* Errors: a `!` before the return list marks a function that can fail, `Error(Member)` raises, a postfix `!` passes an error up, `??` gives a fallback, and `let v, e = f()` destructures
* Modules with `import`, `export` and re-exports
* `native` blocks for backend-specific code, and `extern` for calling C

## Backends

| Backend | Command | Status |
| --- | --- | --- |
| x86-64 | `wisp build`, `wisp run` | The main backend. Floating point, maps and function values are still missing. |
| Lua | add `--target lua` | Secondary. It has floating point, maps and function values, but no pointers, `malloc`, `free`, `sizeof`, `extern` or byte-to-string conversions. |

The x86-64 backend writes NASM assembly, assembles it with `nasm` and links with `ld`, using the System V calling convention so the objects link against C. Binaries are static and skip libc by default. A small start stub calls `main` and exits with its return value, and it is added only when `--bin` links.

## Usage

```bash
wisp init                  # native project for the host
wisp init lua              # project that targets Lua
wisp init lib              # library: object files, no main
wisp init target <name>    # add a target of your own, creating the project if needed
wisp build                 # compile
wisp run                   # compile and run
wisp check                 # parse and analyse only, no assembler or linker
wisp clean                 # remove binaries, objects, assembly and Lua output (-n lists only)
wisp env [key]             # what the compiler resolved: root, target, std, output paths
wisp targets               # built-in and project targets, the default marked
wisp doc [module [symbol]] # exported API with doc comments; no argument lists the modules
wisp cache dir|clean       # the standard library object cache
wisp version
wisp debug lexer|parser|analyser [file.wsp]
wisp lsp                   # language server
```

`run` prints nothing of its own, so its output can be piped. `build` and `check` end with one line on stderr, such as `Finished linux_x64 -> bin/linux_x64/app in 0.4s`. `-q` hides it, and `-v` shows every step with timings and the `nasm` and `ld` commands. Diagnostics are colored only on a terminal, and `NO_COLOR` turns the color off.

The compiler reads every `.wsp` file under the project root (the folder with `wisp.toml`) and looks for `main`. With no `main` the project is a library and asking for a binary is an error. With several, set `entry` in `wisp.toml`. Folders starting with `.` or `_` are skipped.

`main` can return nothing or an `int`, which becomes the exit code. It can also be fallible (`func main() !int`): if it ends in an error, the program prints `error[domain]: message` with the file and line, and exits with the error's code.

## wisp.toml

Every key is optional, and each has a command line flag that wins over it. Keys at the top must come before any `[table]`.

```toml
target = "linux_x64"      # default target; --target picks another
entry = "main.wsp"
std = "kernel/std"        # another standard library folder (--std)
freestanding = true       # no standard library at all (--freestanding)
link = ["lib.o"]          # extra linker arguments, in order (--link)
libc = true               # link the C library (--libc)

[output]
bin = "bin/app"
# obj = "build/obj"
# asm = "build/asm"
# lua = "dist"
```

Paths are relative to the folder with `wisp.toml`. Without `[output]`, files go under a folder named after the target, so builds for different targets never overwrite each other: `bin/<target>/<project>`, `build/<target>/asm`, `build/<target>/obj`, `dist/<target>`. A target can set its own `[targets.<name>.output]`, which beats the top-level `[output]`.

## Targets

`linux_x64` and `lua` are built in. A project can add its own, for example a userspace for a kernel. `wisp init target <name>` writes the block and starter files (they use Linux system calls as placeholders to replace). Then `wisp build --target <name>` uses it.

```toml
[targets.hakura_x64]
arch = "x64"
os = "hakura"
start = "targets/hakura/start.asm"
runtime = "targets/hakura/runtime.asm"
link = ["-T", "targets/hakura/link.ld"]
```

| Key | Required | Default | Meaning |
| --- | --- | --- | --- |
| table name | yes | | lowercase letters, digits and underscores, starting with a letter. Not `linux`, `x64` or `lua` |
| `arch` | yes | | only `"x64"` for now |
| `start` | yes | | assembly file that defines the entry symbol, calls `main` and leaves the program the way the kernel needs, with `main`'s value in `rax` |
| `os` | no | none | a second name that `native` blocks can select |
| `runtime` | no | none | assembly file with the routines below |
| `entry` | no | `_start` | the symbol the linker starts at (passed as `-e` when different) |
| `link` | no | none | extra `ld` arguments |
| `abi`, `format` | no | `sysv`, `elf64` | the only values that exist |
| `libc`, `output` | no | top level | per-target override |

Name a target after its system and architecture: a kernel called Hakura on x86-64 is `hakura_x64` with `os = "hakura"`. A `native` block can then pick a selector, from narrowest to widest:

```wsp
native hakura_x64(...) "...";   // this target only
native hakura(...) "...";       // every target with os = hakura
native x64(...) "...";          // every x86-64 target, linux_x64 included
```

A function with `native` blocks and none that matches the target being built is an error, so an unfinished port is never a silently empty function.

A block named after a built-in target adjusts it. `link` is added to the built-in arguments, `os`, `abi`, `format`, `entry`, `start` and `runtime` replace theirs, `arch` cannot change, and `lua` only takes `output`.

```toml
[targets.linux_x64.output]
bin = "dist/app"
```

### The runtime

The compiler calls these routines itself. Only the ones a program uses are linked, and a missing one is an undefined-symbol error, so a new kernel can start with the first two. They follow System V and must preserve `rbx`, `rbp` and `r12` to `r15`.

| Routine | Used by | Input | What it does |
| --- | --- | --- | --- |
| `__wisp_emit` | `emit` | `rdi` text, `rsi` length | writes the text and a line feed to standard output |
| `__wisp_panic` | `panic`, bounds, division and null checks | `rdi` message, `rsi` length | writes `panic: `, the message and a line feed to standard error, then exits with a failure. Never returns |
| `__wisp_error_emit` | a `main` that ends in an error | `rdi` error record | writes `error[domain]: message` and `  at file:line` to standard error, then returns |
| `__wisp_malloc` | `malloc` | `rdi` size | `rax` address and `rdx` 0, or `rax` 0 and `rdx` an error record. A negative size fails |
| `__wisp_realloc` | `realloc` | `rdi` address or 0, `rsi` size | like `malloc`, and keeps the old block on failure |
| `__wisp_free` | `free` | `rdi` address or 0 | releases the block |

An error record is 64 bytes: domain address (offset 0) and length (8), code (16), message address (24) and length (32), file address (40) and length (48), line (56). The built-in `linux_x64` runtime is a complete example.

## Modules

Every `.wsp` file is a module. An import names a file, and a directory resolves to the file inside it with the same name, so `import "libs/math"` finds `libs/math/math.wsp`.

```wsp
import "libs/math" as m;        // the alias defaults to the last segment

import "libs/math/constants";
export constants;               // m.PI, flattened into this module
export constants as consts;     // m.consts.PI, nested instead
```

Import cycles and duplicate exports are errors.

## Standard library

[`std/`](std) is imported with the `std/` prefix and found in a `std` folder next to the `bin` folder that holds `wisp` (the layout `make prod` builds). `--std <path>` points the prefix at another folder, and `--freestanding` makes any `import "std/..."` an error. They cannot be combined.

```wsp
import "std/math" as math;
```

The library stays small. The compiler knows only the builtins below, and things like joining strings are library code on top of them.

## Builtins

No import needed. A function with the same name in your module shadows the builtin.

| Builtin | What it does |
| --- | --- |
| `emit(text)` | writes a string and a newline to standard output |
| `len(x)` | length of an array, slice, string or variadic parameter |
| `panic(message)` | stops with `panic: message` and the location, exit code 1 |
| `malloc(n)` | `n` bytes of heap as a `*void`; fallible |
| `realloc(p, n)` | resizes a block; fallible. `realloc(null, n)` is `malloc(n)`. On failure the original stays valid |
| `free(a, b, ...)` | releases pointers, strings or maps. `free(null)` does nothing |
| `sizeof(x)` | size in bytes of a type or an expression |

`malloc` and `realloc` fail with `AllocError.OutOfMemory`, so you write `let p = malloc(16)!`, use `??`, or destructure. There is no garbage collector. Strings built from `malloc`ed memory are yours to free, and freeing anything that is not a heap block is undefined, as in C.

## Pointers

The star is a prefix of the type: `*int`, `**int`, `*Point`.

```wsp
int x = 1;
*int p = &x;
*p = 7;

*int heap = malloc(sizeof(int) * 4)!;
heap[2] = 30;
*int next = heap + 1;       // steps in elements, not bytes
int[] view = heap[0:4];     // slicing a pointer gives a span
free(heap);
```

* A pointer to a struct reaches fields directly (`pp.x`). Dereferencing `null` panics.
* `p + n`, `p - n`, `p += n` and `p++` move by elements, `p - q` counts elements between pointers of the same type, and pointers compare by address. None of this works on `*void`, so convert it first (assignment does it implicitly).
* Indexing and slicing a pointer is not bounds-checked, and `p[a:b]` needs both bounds.
* `(*T)(x)` converts an integer, a string or another pointer to a `*T`, at any depth: `(**int)(&p)`. A type that is not a plain name goes in parentheses. `int(p)` goes the other way. Writing through a converted address is `*(*int8)(address) = 1`.
* `string(bytes)` views `char`, `int8` or `uint8` bytes as a string without copying. `(*int8)(s)` is the address of a string's bytes, safe for C only when the string ends in a zero byte (literals do, slices of them do not).

## Globals

A declaration outside any function is a global. The initializer must be constant (literals, constants, enum members, and arrays or structs built from them), so nothing runs before `main`. Without one the global starts zeroed. `export` makes it visible and writable from other modules. An unused global gets a warning unless its name starts with `_`.

```wsp
int counter = 0;
Settings settings = Settings{retries: 3, verbose: true};
export int shared = 100;
```

## C interop

`extern` declares a function defined in C, and the linker finds it among your objects.

```wsp
extern func add(int32 a, int32 b) int32;
extern func puts(*int8 text) int32 as c_puts;                      // C name on the left, Wisp name on the right
export func scale(int32 v, int32 k) int32 as wisp_scale => v * k;  // Wisp name on the left, C name on the right
extern func printf(*int8 format, ...) int32;                       // C variadic
```

* Signatures use only integers, `bool`, `char`, floats and pointers, with at most one return value and no `!`. A Wisp `int` is 64 bits, so use `int32` and the other sized types to match C.
* A function without a body is an error unless it is `extern`. `extern` does not exist on the Lua backend.
* Pass objects with `link = ["lib.o"]` or `--link ./lib.o`. They go to the linker as written, after the project's objects, so archives, `-L` and `-l` work.
* `--libc` (or `libc = true`) links through `cc` and starts the program with the system's startup code, so `printf` and the rest of the C library work. A panic still ends the program at once, without flushing C's buffers.

## Native blocks

```wsp
func abs(int x) int {
  int r;
  native lua(out r, in x) "r = x < 0 and -x or x";
  return r;
}
```

The name after `native` picks a backend or target, and other blocks are skipped. `in` lists variables the block reads, `out` those it writes and `inout` both. A mode covers the names after it until the next mode: `in a, b, out c`. On x86-64 the code refers to them as `{name}`, a sized memory operand, and `;` separates instructions:

```wsp
native linux_x64(in x, out r) "mov rax, {x}; add rax, rax; mov {r}, rax";
```

A binding can reach inside a variable:

| Form | Meaning |
| --- | --- |
| `{x}` | the variable, up to 8 bytes |
| `{s.ptr}`, `{s.len}` | the parts of a string or slice |
| `{p.x}`, `{o.inner.y}` | a field of a struct, nested fields included |
| `{g}` | a global works like a local, fields and `&` included |
| `{&x}`, `{&p.y}` | the address of any variable or field, of any size, as `[rbp-24]`. Use it with `lea`: `lea rax, {&values}` |

A string in backticks is a raw block. It can span lines, a newline separates instructions, and `;` starts an assembly comment:

```wsp
native linux_x64(out r) `
    mov rcx, 3
    xor rax, rax
.again:
    add rax, 14   ; comments work here
    dec rcx
    jnz .again
    mov {r}, rax
`;
```

The compiler does not save registers for you. A block that changes `rbx`, `rbp`, `rsp` or `r12` to `r15` must restore them. In Lua the code is copied as written and uses the variable names directly, so `{...}` and backticks mean nothing there.

A function that has `native` blocks but none for the target being built is an error, so a missing port is never a silently empty function. This applies to every module in the project, used or not.

## Diagnostics

Errors, warnings, notes and help texts are shown with the source line under them. Warnings never stop a build: unused variables, constants and imports (silenced by a leading `_`), an exit code of 0 from an unhandled error, and unreachable code. A panic prints `panic: message` and `at file:line:col`.

## Editor support

`wisp lsp` starts the language server. [`vscode/`](vscode) is a VS Code extension with highlighting, snippets and an icon. The server gives diagnostics as you type, completion (members, struct fields, and `import` paths one segment at a time, so `std/` offers `math`, `strings` and `sys`, and `constants` only shows after `std/math/`), hover with doc comments, signature help, outline, rename, go to definition and quick fixes. It also serves `wisp.toml`: key completion, hover and diagnostics. The extension is versioned apart from the compiler.

## Building

You need Go, plus `nasm` and `ld` for x86-64, `cc` for `--libc` and the C tests, and `lua` 5.4 for the Lua backend.

```bash
make build         # bin/wisp
make prod          # package bin/ and std/ into dist/
make install       # make prod, then install into /usr/local/wisp (asks for sudo)
make ext           # package the VS Code extension into dist/vscode
make test          # test-lua and test-x64; FILTER=name narrows them
```

`tests/gen` generates the test programs on every run.

## Versioning

The compiler version lives in [`internal/version/VERSION`](internal/version/VERSION), and `wisp version`, the language server and the Makefile read it. Releases are `major.minor.patch`. While the major is 0, a minor adds features and may break things, and a patch is a fix. Each minor has a codename, a will-o'-the-wisp from a different culture. To bump: edit the file and tag.

## Layout

```text
cmd/main/            command line
internal/            lexer, parser, ast, analyser, module, diag, lsp, version
internal/target/     asm (start stub and runtime), lua and x64 backends
std/                 standard library
example/             guided tour
tests/               generator (gen) and runner (run.sh)
vscode/              VS Code extension
```
