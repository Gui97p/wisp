# Stdlib core: contract

Modules: `sys`, `os`, `fmt`. There is no `io`. The stdlib is written by hand, per target (`linux_x64` and `lua`).

## sys

Raw, per target, Linux x64 values. Thin: no retry, no buffering, no portability. `os` builds on it, and `os` is where names become portable. `sys` mirrors the Linux kernel ABI, so its constants and numbers are the kernel's. Another target ships its own `sys`, with `std = "path"` in `wisp.toml`.

### Results

`sys` has no `!T` and no error enum.

- Every function returns what the kernel returned: `>= 0` on success, `-errno` in `-4095..-1` on failure.
- `errno(int raw) int` returns `0` for a success and the positive errno number for a failure. Callers that need the exact number use it.
- Turning failures into `Error` values is the job of `os`.
- An `Error` is a static record, one per `Error(...)` site, so its first argument must be a constant enum member. A runtime errno cannot become an `Error` directly, which is why `os` maps it with a `switch`.

### Functions

Raw bytes go in as a pointer and a length, like the syscall itself. `read` and `write` take `*char`, `mmap` and `munmap` use `*void`. A caller holding a string passes `(*char)(s)` and `len(s)`, and a caller holding an array or span passes `(*char)(buf)` and `len(buf)`. A string is a view and is never assumed to end in a zero byte, so paths are `*char` to zero-terminated memory.

| Function | Syscall | Notes |
| --- | --- | --- |
| `read(int fd, *char buf, int n) int` | `read` 0 | bytes read, 0 at end of file |
| `write(int fd, *char buf, int n) int` | `write` 1 | may write fewer than `n` bytes |
| `open(*char path, int flags, int mode) int` | `openat` 257 with `AT_FDCWD` | the mode is the permission of a new file. `0` creates a file nobody can read |
| `close(int fd) int` | `close` 3 | |
| `seek(int fd, int offset, int whence) int` | `lseek` 8 | new offset |
| `sync(int fd) int` | `fsync` 74 | |
| `size(int fd) int` | `fstat` 5 | reads `st_size` (offset 48), or the negative error |
| `remove(*char path, int flags) int` | `unlinkat` 263 | `0` for a file, `AT_REMOVEDIR` for an empty directory |
| `mkdir(*char path, int mode) int` | `mkdirat` 258 with `AT_FDCWD` | the same `*at` family as `open` and `remove`; plain `mkdir` 83 does the same job |
| `rename(*char from, *char to) int` | `renameat` 264, both directory descriptors `AT_FDCWD` | |
| `stat(*char path, *char buf) int` | `newfstatat` 262 with `AT_FDCWD`, flags 0 | follows links. `buf` is a `Stat` |
| `fstat(int fd, *char buf) int` | `fstat` 5 | same buffer. `size` is `fstat` plus the `ST_SIZE` read |
| `getdents(int fd, *char buf, int n) int` | `getdents64` 217 | fills `buf` with records, returns the bytes filled, `0` at the end of the directory |
| `getcwd(*char buf, int n) int` | `getcwd` 79 | the kernel returns the length including the zero byte |
| `mmap(int len) *void` | `mmap` 9 | anonymous, private, read and write. A failure is a negative address |
| `munmap(*void p, int len) int` | `munmap` 11 | |
| `time(int clock) (int, int)` | `clock_gettime` 228 | seconds and nanoseconds within the second |
| `sleep(int sec, int nsec) int` | `nanosleep` 35 | returns `-4` if interrupted |
| `random(*void buf, int n) int` | `getrandom` 318 | flags 0 |
| `isatty(int fd) bool` | `ioctl` 16 with `TCGETS` | needs a termios buffer for the third argument |
| `exit(int code)` | `exit_group` 231 | does not return |

### Constants

- Open flags: `O_RDONLY = 0`, `O_WRONLY = 1`, `O_RDWR = 2`, `O_CREAT = 64`, `O_EXCL = 128`, `O_TRUNC = 512`, `O_APPEND = 1024`, `O_CLOEXEC = 524288`.
- `O_DIRECTORY = 65536`: `open` fails with `ENOTDIR` unless the path is a directory.
- `struct Stat` mirrors the kernel's `struct stat` (144 bytes: `st_mode` at 24, `st_size` at 48, `st_mtim` at 88), so `os` reads the fields by name instead of by offset. Mode type bits: `S_IFMT = 0o170000`, `S_IFREG = 0o100000`, `S_IFDIR = 0o040000`, `S_IFLNK = 0o120000`.
- `getdents64` record: `d_reclen` is 2 bytes at offset 16, `d_type` is 1 byte at 18, the name starts at 19 and ends at a zero byte. `DT_UNKNOWN = 0`, `DT_DIR = 4`, `DT_REG = 8`, `DT_LNK = 10`. A record with `DT_UNKNOWN` needs a `stat` to know its kind.
- `SEEK_SET = 0`, `SEEK_CUR = 1`, `SEEK_END = 2`.
- `CLOCK_REALTIME = 0`, `CLOCK_MONOTONIC = 1`.
- `AT_FDCWD = -100`, `AT_REMOVEDIR = 512`.

### Not syscalls

The environment block is not reached through a syscall: the kernel puts `envp` on the initial stack, so the start stub keeps it, as it keeps `argc` and `argv`, and `os.getenv` walks it. `sys` only needs a way to read that pointer.

### Not in sys yet

`fork`, `execve`, `wait4`, `clone`, signals, sockets. They come with a process API in `os`.

### Lua

Only `write`, `time` and `exit` have a `native lua` body. Lua has no pointers, and a `*char` there is the string itself, so `read` cannot fill a buffer. Every other function fails loudly with `error("sys.<name> is not available on lua")`. A function with no `native` for the active target compiles to a body that does nothing and returns zero, which is why the loud error is needed.

## os

The portable layer over `sys`. Every function that can fail is `!T`. It is a package: what `os.wsp`, `global.wsp` (constants, `IOError`, the error helper), `files.wsp` and `time.wsp` make `public` is the API of `os`, so callers write `os.open`, `os.READ`, `os.now`.

### Constants

- `STDIN = 0`, `STDOUT = 1`, `STDERR = 2`: plain fd constants. There is no global buffer, so nothing to flush at exit.
- Open flags, re-exported from `sys` with neutral names: `READ`, `WRITE`, `READ_WRITE`, `CREATE`, `EXCLUSIVE`, `TRUNCATE`, `APPEND`. They combine with `|`.
- Seek origins: `SEEK_START`, `SEEK_CURRENT`, `SEEK_END`.

### Errors

`public enum IOError { NotFound = 1, Denied, Exists, Interrupted, IO, NotDir, IsDir, NotEmpty }`. A path or a buffer that does not fit is `errors.Kind.Invalid`, shared with the other modules.

- One private helper turns the raw result of a `sys` call into an `Error`. It receives the positive errno, which is `sys.errno(raw)`, never the raw negative value.
- `2` is `NotFound`, `1` and `13` are `Denied`, `17` is `Exists`, `4` is `Interrupted`, `22`, `34` and `36` are `errors.Kind.Invalid`, `20` is `NotDir`, `21` is `IsDir`, `39` is `NotEmpty`, the rest is `IO`.

### Descriptors

| Function | Behaviour |
| --- | --- |
| `write(int fd, string s) !int` | writes the string through the same loop as `writeBytes` |
| `writeBytes(int fd, char[] data) !int` | loops until every byte is written, retries when the kernel returns `-4`, treats a zero result as `IO`, returns the number of bytes |
| `read(int fd, char[] buf) !int` | one read of at most `len(buf)` bytes, retried on `-4`, returns the bytes read, `0` at end of file. It returns as soon as the descriptor has data, so a pipe or a terminal never waits to fill the buffer. A smaller read is `buf[0:k]` |
| `close(int fd) !void` | an interrupted close counts as done, the descriptor is gone either way |
| `seek(int fd, int offset, int whence) !int` | the new offset |
| `size(int fd) !int` | the file size in bytes |
| `sync(int fd) !void` | flushes the file to disk |
| `isatty(int fd) bool` | true for a terminal |

A private `writeRaw(int fd, *char buf, int n)` holds the loop, so it exists in one place.

### Files

| Function | Behaviour |
| --- | --- |
| `open(string path, int flags) !int` | copies the path to a 4096-byte stack buffer and adds the zero byte, because a string is not guaranteed to end in one. A longer path is `errors.Kind.Invalid`. Creates with mode `0o644`. Adds the close-on-exec flag |
| `create(string path) !int` | `open(path, WRITE \| CREATE \| TRUNCATE)` |
| `remove(string path) !void` | deletes a file, same path rule |
| `removeDir(string path) !void` | deletes an empty directory, `sys.remove` with `AT_REMOVEDIR` |

### Time and process

| Function | Behaviour |
| --- | --- |
| `now() int` | nanoseconds since the epoch, from the real-time clock |
| `ticks() int` | nanoseconds from the monotonic clock, for measuring durations |
| `sleep(int ms) !void` | sleeps for milliseconds, `Interrupted` if a signal stops it |
| `exit(int code)` | ends the program, never returns |

### Directories and the environment (proposed 2026-10-27)

| Function | Behaviour |
| --- | --- |
| `mkdir(string path) !void` | one directory, mode `0o755`, `Exists` if it is there |
| `mkdirAll(string path) !void` | every missing parent too, an existing directory is not an error. Walks `path.dir` upward |
| `rename(string from, to) !void` | atomic on the same filesystem |
| `stat(string path) !Stat` | `Stat{int size, FileKind kind, int modified, int mode}`, `modified` in nanoseconds like `now()` |
| `exists(string path) bool`, `isDir(string path) bool` | `stat` that folds every error into `false` |
| `cwd(char[] buf) !string` | the working directory, a view of `buf`, `errors.Kind.Invalid` when it does not fit (the kernel's `ERANGE`, 34) |
| `getenv(string name) (string, bool)` | a view into the environment block, `false` when unset. Needs the start stub to keep `envp`, as it already does for `args` |

`public enum FileKind { File, Dir, Link, Other }`. `IOError` grows `NotDir` (20), `IsDir` (21) and `NotEmpty` (39).

Listing a directory has no result size, and there is no growable list. So it is an iterator, the same shape as `Splitter` and `Cursor`:

- `openDir(string path) !Dir`, `d.next() (Entry, bool)`, `d.close()`.
- `Entry{string name, FileKind kind}`. `name` is a view into the `Dir`'s own buffer and is valid until the next call to `next`.
- The `Dir` holds the descriptor and a `getdents64` buffer in `internal` fields. `.` and `..` are skipped.
- Paths are copied to a zero-terminated stack buffer exactly as in `open`. That copy is the one private helper all path calls share.

### Later

- A buffered `File` (the `fopen` idea) is its own layer on top of this one: a `Reader` with a buffer plus the `fmt.Writer`, with a buffer the caller owns and `defer f.close()`.
- Processes (`spawn`, pipes), sockets and signals.

## path

Pure functions on strings, no system calls, so it never fails and never allocates. The separator is `/`; a second separator arrives with a second target. It follows the rules of `strings`: a `string` result is a view of the argument, and what has to be built is written into a caller buffer.

### Views

| Function | Behaviour |
| --- | --- |
| `base(string p) string` | the last element, trailing separators ignored. `base("a/b/")` is `b`, `base("/")` is `/`, `base("")` is `.` |
| `dir(string p) string` | everything before the last element. `dir("a/b")` is `a`, `dir("a")` is `.`, `dir("/a")` is `/` |
| `ext(string p) string` | the suffix from the last dot of the base, dot included. `ext("a.tar.gz")` is `.gz`. A leading dot is part of the name: `ext(".gitignore")` is empty |
| `stem(string p) string` | the base without its extension |
| `isAbs(string p) bool` | starts with `/` |
| `elements(string p) strings.Splitter` | the pieces between separators, for walking a path without a list |

### Producers

| Function | Behaviour |
| --- | --- |
| `join(char[] buf, string ...parts) !string` | joins with a single separator, skipping empty parts. An absolute later part does not reset the path (that is `Clean`'s business in Go, and surprising) |
| `clean(char[] buf, string p) !string` | lexical only: collapses repeated separators and `.`, resolves `..` against the previous element, never touches the disk. An empty result is `.` |

Both return `errors.Kind.NoSpace` when `buf` is too small, the same value `strings` returns.

### Not here

Symbolic links, `rel`, glob matching and the current directory (`os.cwd`).

## math

Integer first. The float functions stay as they are and wait for floats on x64.

### Naming

An unprefixed name is the integer function when an integer version exists (`abs`, `min`, `max`) and the float one carries an `f` (`fabs`, `fmin`, `fmax`). Functions with no integer meaning keep the plain name (`sqrt`, `sin`, `floor`). The float `pow` stays, so the integer power is `ipow`.

### Constants

`MAX_INT`, `MIN_INT`, `MAX_UINT` and the sized forms `MAX_INT8` to `MAX_UINT64` and `MIN_INT8` to `MIN_INT64`. Together with `PI`.

### Integer functions

| Function | Behaviour |
| --- | --- |
| `abs`, `min`, `max` | exist. `abs(MIN_INT)` is `MIN_INT`, as in every two's complement language; `checkedAbs` exists for the careful case |
| `sign(int x) int` | -1, 0 or 1 |
| `clamp(int x, lo, hi) int` | `lo` when below, `hi` when above. `lo > hi` is a bug of the caller and returns `lo` |
| `mod(int a, b) int` | the Euclidean remainder, never negative for positive `b`, for wrap-around indices. `b == 0` panics like `%` does |
| `gcd(int a, b) int`, `lcm(int a, b) !int` | `lcm` can overflow |
| `ipow(int base, exp) !int` | `exp < 0` is `errors.Kind.Invalid`, overflow is `errors.Kind.Overflow` |
| `isqrt(int n) int` | the integer square root, `0` for negatives |
| `isPowerOf2(int n) bool` | |
| `popCount`, `leadingZeros`, `trailingZeros` | on `uint64`. Plain loops until the compiler has intrinsics |

### Checked arithmetic

`checkedAdd`, `checkedSub`, `checkedMul` and `checkedAbs`, all `(int a, b) !int` (`checkedAbs` takes one). Precedents are Rust's `checked_*` and Zig's `@addWithOverflow`; without a builtin they are plain comparisons against `MAX_INT` and `MIN_INT`. They exist for code that must not wrap, first of all `conv.parseInt` and the lexer's number reading.

`math` has no error enum of its own: `Overflow` and `Invalid` come from `errors.Kind`.

### Existing bug to fix with this

`min` and `max` loop with `for i in 1..l-1`. Ranges are inclusive and a range whose start is greater than its end counts down, so a single argument visits indexes 1 and 0 and panics. `for _, v in args[1:]` has no such case.

## conv

Numbers to text and back, with buffers the caller owns. Nothing allocates, and nothing is a language feature.

### Errors

`conv` has no error enum of its own: its three failures are `errors.Kind.Invalid`, `errors.Kind.Overflow` and `errors.Kind.NoSpace`.

- `Invalid`: the text is empty or holds a character that is not a digit of the base.
- `Overflow`: the value does not fit in the result type.
- `NoSpace`: the buffer is too small for the digits.

### To text

Each function writes the digits into `buf` and returns a view of the part it used, so the result is a string that points into `buf`. The digits are written from the end of the buffer backwards, which means nothing has to move at the end.

| Function | Behaviour |
| --- | --- |
| `itoa(char[] buf, int n) !string` | decimal, with a leading `-` for a negative value. `MaxDigits` bytes always fit |
| `utoa(char[] buf, uint n) !string` | decimal for an unsigned value |
| `formatUint(char[] buf, uint n, int base, bool upper) !string` | any base from 2 to 36. `upper` picks the case of the letters. No prefix, `fmt` adds `0x` |
| `formatInt(char[] buf, int n, int base, bool upper) !string` | the signed version, `-` before the digits |

- `IntMax`, `IntMin` and `UintMax` are public constants.
- `public const MaxDigits = 65;` is the size that fits any value in any base: 64 binary digits and a sign.
- A base outside 2 to 36 is `Invalid`.
- The most negative `int` cannot be negated, so the magnitude is computed as an unsigned value: `uint(0) - uint(n)`.

### From text

| Function | Behaviour |
| --- | --- |
| `atoi(string s) !int` | a decimal value with an optional `+` or `-`. Spaces are not trimmed |
| `parseInt(string s, int base) !int` | the same, in a given base from 2 to 36 |
| `parseUint(string s, int base) !uint` | no sign allowed |
| `atou(string s) !uint` | decimal, no sign |

- Underscores and prefixes (`0x`) are not accepted. A caller that wants them strips the prefix and picks the base.
- A value past the range is `Overflow`, never a silent wrap.

### Not here

Floats wait for floats on x64. Character tests (`isDigit`, `isSpace`) live in `strings`.

## errors

Conditions that mean the same thing in every module live in one enum, the way Rust has one `io::ErrorKind` and Go one `io.EOF`.

`public enum Kind { NoSpace = 1, Invalid, Overflow }`.

- `NoSpace`: a buffer the caller owns is too small.
- `Invalid`: the argument is outside what the function accepts (text that is not a number, a base out of range, a path that is too long).
- `Overflow`: the value does not fit in the result type.

A rule for the next one: the same word with the same meaning in two modules belongs here, and what only one module can say stays in that module (`os.IOError`, `fmt.FormatError`). `AllocError` is a runtime builtin and stays there.

Because an `Error` compares by domain and by code, `e == errors.Kind.NoSpace` is true for a `NoSpace` from `strings`, `conv`, `fmt` or `path`, and the caller does not need to know which one raised it. The module has no imports, so every other module can use it.

### Error codes start at 1

The enums that raise errors have no `Ok` member: success is a null `Error`, not code `0`. The first member is written `= 1`, as in Zig, where `0` is reserved for "no error". It matters beyond taste: a fallible `main` that ends with an error leaves with that code as its exit status, and `0` would read as success. The analyser warns when `Error(X)` is raised with a member whose value is `0`.

## strings

Text over bytes: a `string` is a read-only view of bytes, and `len` counts bytes. Nothing here decodes UTF-8 or knows about letters beyond ASCII; that is a later `utf8` module. Number to text lives in `conv`, formatting in `fmt`.

### Rules

1. **The return type says who owns the memory.** A function that returns `string` returns a view of one of its arguments (or of a literal). It never allocates, never fails and never panics: an out-of-range request is clamped.
2. **A function that makes new bytes writes into a destination the caller owns**, the way `conv.formatInt` already does: `f(char[] buf, ...) !string`. The result is a view of the part of `buf` that was written, and `NoSpace` is the only failure. The caller keeps `buf`.
3. **A function that changes bytes in place takes `char[]` and returns nothing.**
4. **Only `clone` allocates.** It returns a `char[]` the caller frees, and it is the only way from a `string` to bytes you own. There is no conversion from `string` to `char[]`: that would hand out a writable view of read-only memory, and a literal lives in `.rodata`.
5. Anything that needs an unknown amount of output (joining a whole file, building a message piece by piece) writes through a `Writer`, which is the one thing that grows.

### Errors

The only failure is `errors.Kind.NoSpace`, which means the destination is too small. A negative count behaves as zero, and an empty pattern has the meaning written next to each function.

### Search (views in, numbers out)

- `indexOf(string s, string sub) int`: first byte offset of `sub`, or -1. An empty `sub` is found at 0.
- `lastIndexOf(string s, string sub) int`: last offset, or -1. An empty `sub` is found at `len(s)`.
- `indexOfByte(string s, char c) int` and `lastIndexOfByte(string s, char c) int`.
- `count(string s, string sub) int`: non-overlapping occurrences. An empty `sub` counts `len(s) + 1`, as in Go.
- `startsWith(string s, string prefix) bool` and `endsWith(string s, string suffix) bool`: false when the prefix or suffix is longer than `s`, true when it is empty.
- `compare(string a, string b) int`: -1, 0 or 1, byte by byte. `==`, `<` and `in` already exist on strings, so there is no `equals` and no `contains`.
- `equalFold(string a, string b) bool`: equal ignoring ASCII case.

### Views (no allocation)

- `trim(string s, string charset) string`, `trimLeft` and `trimRight`: remove the bytes that are in `charset` from the ends.
- `trimSpace(string s) string`: the same with the whitespace of `isSpace`.
- `trimPrefix(string s, string prefix) string` and `trimSuffix(string s, string suffix) string`: unchanged when it does not match.
- `cut(string s, string sep) (string, string, bool)`: the text before the first `sep`, the text after it, and whether `sep` was found. When it is not found the result is `s`, `""`, `false`. An empty `sep` is found at 0.
- `Splitter`: walks the pieces of a string without a list. `splitter(string s, string sep) Splitter`, then `next() (string, bool)` gives each piece (an empty `sep` splits into single bytes) and `rest() string` what has not been read. Its fields are `internal`.
- `fields(string s) Splitter`: the same, splitting on runs of whitespace and skipping empty pieces.

### Producers (write into `buf`, return `!string`)

- `copy(char[] buf, string s) !string`: the bytes of `s` in `buf`.
- `concat(char[] buf, string ...parts) !string`.
- `join(char[] buf, string[] parts, string sep) !string`.
- `rep(char[] buf, string s, int n) !string`: `s` repeated `n` times, empty when `n <= 0`.
- `replace(char[] buf, string s, string old, string new, int n) !string`: replaces the first `n` occurrences, all of them when `n < 0`. An empty `old` makes no replacement.
- `upper(char[] buf, string s) !string` and `lower(char[] buf, string s) !string`: ASCII only.

### In place (`char[]`, no result)

- `toUpper(char[] s)`, `toLower(char[] s)`: ASCII only.
- `reverse(char[] s)`: reverses the bytes.

### Allocating

- `clone(string s) !char[]`: a copy the caller owns and frees. An empty string gives `[]` without allocating. The error is `AllocError`.

### Characters

Tests and conversions on one byte, ASCII only. They live here because the character tests belong with text (see `conv`).

- `isDigit(char c) bool`, `isHexDigit(char c) bool`, `isAlpha(char c) bool`, `isAlnum(char c) bool`, `isUpper(char c) bool`, `isLower(char c) bool`, `isSpace(char c) bool` (space, tab, newline, carriage return, vertical tab, form feed), `isPrint(char c) bool`.
- `upperByte(char c) char` and `lowerByte(char c) char`: unchanged when `c` is not a letter.
- `digitValue(char c, int base) (int, bool)`: the value of a digit in `base`, and whether it is one (letters of either case up to base 36).

### Not here

Anything that needs a growing buffer beyond `clone` waits for `Writer` to grow. A list of pieces (`split` returning `string[]`) is not provided: the `Splitter` walks them, and a list needs somewhere to live, which is the same question as the growable vector. Unicode (`runes`, `utf8`) is a later module.

## fmt

### Writer

```
struct Writer {
  int fd;
  char[] buf;
  int len;
  BufMode mode;
  int total;
}

enum BufMode { None, Line, Full }
```

- `fd >= 0`: a real descriptor. A full buffer is flushed with `write`.
- `fd == -1`: memory. There is nothing to flush, so filling the buffer is an error.
- `None`: every write goes straight out.
- `Line`: flush when a `\n` is written.
- `Full`: flush only when the buffer fills, or on an explicit `flush`.

The operations that change a `Writer` are methods with a pointer receiver, so `w.flush()` works on a variable without taking its address. Methods are the unit an interface can abstract over later, which is why they are not plain functions.

- `(*Writer w) flush() !void`: does nothing for a memory writer (`fd < 0`) or an empty buffer.
- `(*Writer w) format(string format, ...args) !int`: the one real formatter. Returns the bytes written.
- `(*Writer w) write(string s) !void`, `writeBytes(char[] data) !void`, `writeByte(char c) !void` and `fill(char c, int n) !void`: the primitives `format` is built on. All of them go through one private routine that holds the buffering rules.

The rules of that routine, by mode:

- `Full`: copy into the buffer. When it is full, flush it and go on. A text larger than the whole buffer is written straight out after flushing what is there.
- `Line`: the same as `Full`, and after the text is in, flush if it held a `\n`.
- `None`: write straight out and never touch the buffer.
- A buffer of length 0 with `fd >= 0` writes straight out in every mode.
- A memory writer (`fd < 0`) never flushes. A piece that does not fit is `errors.Kind.NoSpace`, and nothing of that piece is written.
- `Writer` also keeps a count of the bytes it accepted, which is how `format` knows what to return.
- Errors of the descriptor come out of `flush` and `write` as the `IOError` of `os`.
- Methods live in the module of their type, so all of these are in `writer.wsp`. The long logic of `format` sits in a function in another file and the method is a thin call to it.

Functions that make a temporary `Writer`:

- `fprintf(int fd, string format, ...args) !int`: builds a `Writer` on the stack in `Full` mode, calls `format`, flushes before returning. One `write` per call.
- `printf(string format, ...args)`: best effort on standard output, like `emit`. It cannot fail: a failed write to the terminal is ignored, since the program has nothing to do about it, and a problem in the format stays visible in the output. A program that must know uses `fprintf(os.STDOUT, ...)`.
- `sprintf(char[] buf, string format, ...args) !int`: a memory `Writer` over `buf`.
- A persistent `Writer` with a mode is opt-in. Whoever creates it calls `flush`.

### Format string

`{` [[fill] align] [`#`] [`0`] [width] [`.` precision] [type] `}`

- `{}` formats the argument by its own type. A placeholder never selects a type, only how to show it.
- `{{` and `}}` are literal braces.
- No `:` and no positional index. `{...}` is never an index.
- Align: an optional fill byte, then `<`, `>` or `^`. When the second byte of the spec is one of those three, the first is the fill. Default `<` for text, `>` for numbers.
- `0`: zero padding, numbers only. It wins over an explicit alignment: zeros always go after the sign and the prefix.
- Width: digits, at most 65535.
- Precision: `.N`, at most 65535, and at least one digit after the dot. Truncates a string. Number of decimals for floats, once they exist.
- `#`: only with `x`, `X`, `b` or `o`, and it writes the `0x`, `0X`, `0b` or `0o` prefix. It comes before the `0` and the width: `{#06x}`.
- Type letters: `d` decimal, `x` lower hex, `X` upper hex, `b` binary, `o` octal, `c` integer as character, `p` pointer.

### Per type

| Type | `{}` | Valid |
| --- | --- | --- |
| int, uint (any width) | decimal | `d x X b o c`, width, `0`, align, `#` |
| char | the character | `c d x`, width, align |
| bool | `true` / `false` | width, align |
| string | the text | width, precision, align |
| pointer | `0x...` | `p`, width |
| enum | the number, as an int | `d x X b o`, width, align |

- To print an enum by name, pass `c.name()`. Every enum has a generated `name()` method (`"?"` when the value has no member), and a hand-written one replaces it.
- A struct by value cannot be an argument (already an error at the call).

### Layout rules

- Width counts bytes. It pads, it never cuts: a text longer than the width is written whole.
- Centered text puts the extra space on the right: `{^5}` of `42` is ` 42  `.
- Zero padding goes after the sign and after the prefix: `{05}` of `-42` is `-0042`, and `{#06x}` of `255` is `0x00ff`.
- A signed value prints as `-` and its magnitude in every base, the same as `conv.formatInt`: `{x}` of `-255` is `-ff`.
- The fill is the character before `<`, `>` or `^`: `{*^7}` of `ab` is `**ab***`, and `{-<6}` of `7` is `7-----`. When the second character of the spec is not one of the three, there is no fill.
- `{c}` writes the integer as one byte. `{d}`, `{x}` and the others on a `char` use its code.
- An enum prints its number. `{p}` and `{}` on a pointer both print `0x` and the address in hex.

### Errors

`format` never panics because of its format string. A problem in the text or in the arguments is left visible in the output, and `format` reports the **first** one when it is done.

`public enum FormatError { Unmatched = 1, BadSpec, WrongType, UnsupportedType, ExtraArgument }`. A writer that is full is `errors.Kind.NoSpace`. With no `Ok` member the writer keeps its first problem as an `Error` (null when there is none).

| Problem | In the output | Error at the end |
| --- | --- | --- |
| `{` with no `}`, or a lone `}` | the text is written as it is | `Unmatched` |
| a spec that does not parse (`{5.}`, `{z}`, `{x5}`) | the placeholder is written as it was | `BadSpec` |
| a spec that does not fit the type (`{x}` on a string) | the placeholder is written as it was | `WrongType` |
| an argument of a type `fmt` cannot show (a named type) | the placeholder is written as it was | `UnsupportedType` |
| more arguments than placeholders | nothing for the extras | `ExtraArgument` |
| fewer arguments than placeholders | the placeholder is written as it was | none |

- A placeholder always takes one argument, even when it is written back as it was, so the placeholders after it still line up with their arguments. The one exception is a placeholder with no argument left.
- Only the first problem is reported, and formatting goes on to the end of the string.
- Problems of the writer are different and stop `format` at once: `errors.Kind.NoSpace` from a memory writer that is full, and the `IOError` of `os` from a descriptor that fails.
- A caller that does not care about a problem writes `fprintf(...) ?? 0`, and the text comes out either way. `printf` has no error to handle.

## Box ABI the formatter reads (x64)

An untyped variadic `...args` is a span of boxes. Element is 24 bytes: `[tag, w0, w1]`.

- Primitive: `tag = BoxCode`, `w0` = value widened to 64 bits.
- String: `w0` = pointer, `w1` = length.
- Enum: `tag` = code of `int`, `w0` = value, `w1` = descriptor address.
- Nominal type: `tag` = descriptor address.
- Pointer: `tag` = code of pointer, `w0` = address, `w1` = pointee identity.

Enums need nothing more in the box: no name table, the descriptor stays `dq 1`.

## How fmt is laid out

- `cursor.wsp`: a `Cursor` over a string with a position (`peek`, `peekAt`, `advance`, `advanceUntil`, `mark`/`since`, `number`). `peek` gives a zero byte at the end, so nothing indexes past the text.
- `spec.wsp`: `Spec`, `parse` of what sits between the braces, and `validate` of a `Spec` against the type of the argument.
- `render.wsp`: turns a value into a `Piece { sign, radix, body }`. The three parts exist because there is no `+` on strings, so `-0xff` cannot be one string. The body points into a buffer that `format` owns.
- `writer.wsp`: `Writer`, `BufMode`, `FormatError`, the buffering rules, `writePiece` (width, alignment, zeros) and `format`. Methods live in the module of their type, so they cannot be split off, and what does not need a `Writer` lives in the other files.
- `fmt.wsp`: `printf`, `fprintf`, `sprintf`. `fprintf` flushes even when `format` reports a problem, so the text still comes out. `printf` calls `fprintf` on standard output and drops the result.

## Reading files and arguments (proposed 2026-10-26)

- `os.read` returns 0 at the end of the file. That is not an error: end of file is not a failure, and a loop that has to catch an `EndOfFile` error for every read is the worse shape.
- `os.readFile(string path) !char[]` opens, sizes, allocates, reads to the end and closes. The caller owns the result and frees it; a `string` is only a view (`string(data)`) and is never freed.
- `Reader` mirrors `Writer`: `Reader{fd, buf, pos, len}` with `next() (char, bool)`, `peek()` and `readLine()`, for reading without holding the whole file. It is the open-file counterpart of `fmt`'s `Cursor`.
- The program arguments reach `main` as a `string[]`: `func main(string[] args) !int`, or `main()` with none. `args[0]` is the program name. The language checks the signature, and the start stub of each target builds the slice (pointer in the first argument register, length in the second) from `argc` and `argv`.

## Decisions on record

- Shared conditions (`NoSpace`, `Invalid`, `Overflow`) are one enum in `std/errors`. Error enums have no `Ok`, and their first member is `= 1` (2026-10-28).

- `printf` is not fallible: best-effort output to the terminal, like `emit`. `fprintf` and `sprintf` stay fallible because for a file or a buffer the error matters.
- `Writer` operations are methods and the entry points that make a temporary `Writer` are functions. Methods are what interfaces will abstract over.
- `format` does not panic on a bad format string: the problem stays visible in the output and is reported as an error at the end.
- `case enum` was tried and reverted as too magic. Enums get a generated `name()` method instead.
- Integer to string is not a language feature. It is the stdlib `conv` module, with buffers the caller owns.
- Dynamic `Error` codes were considered (a record table per enum) and rejected: there is no way to test enum membership, so a value outside the set could only panic.
- `malloc`, `realloc` and `free` should move out of the builtins into a `mem` module, with the runtime symbols kept for the compiler's own allocations (maps). The open point is how a map is freed.

## Open

- Printing an enum by name through `{}` waits for interfaces (a `Named` interface that `name()` already satisfies).
- Float formatting waits for floats on x64.
- Per-target files inside one module (`sys_linux_x64.wsp`, the Go style) are the plan for when a third target appears. Until then `sys` stays the Linux module.
