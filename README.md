# Wisp

A compiled programming language written in Go.

Wisp is a personal language project focused on simplicity, readability, and direct compilation to native code. The design takes inspiration from C, Go, and Rust while experimenting with its own control-flow constructs and syntax.

The language is still in its early stages and evolves as features are implemented and tested.

## Current Features

* Functions
* Struct declarations
* Fixed-size arrays
* Type inference with `let`
* Range-based loops
* Labels (`break` and `continue`)
* Explicit typing
* Single-expression functions with `=>`

Example:

```wsp
func add(int a, b) int => a + b;

func main() {
  let result = add(2, 3);

  for i = 1..10 {
    emit(i);
  }
}
```

## Stack

* **Go** — compiler implementation
* **x86-64** — compilation target (planned)

## Project Structure

```text
cmd/
└── main/
    └── main.go

examples/
├── syntax.wsp
└── ...
```

## Running

```bash
go run ./cmd/main
```

Or build the compiler:

```bash
go build -o wisp ./cmd/main
```

## Goals

* Keep the language simple and predictable
* Prioritize readability over clever syntax
* Build a complete compiler from scratch
* Learn compiler and language design through implementation

The language is developed incrementally, with decisions driven by real implementation experience rather than extensive upfront design.
