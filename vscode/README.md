# Wisp Language Support

Language support for [Wisp](https://github.com/Gui97p/wisp), a personal compiled language written in Go. Wisp aims to give low-level control with a high-level feel, without a garbage collector. Source files use the `.wsp` extension.

## Features

- Syntax highlighting for keywords, types, numbers, strings, characters, comments, operators, functions, struct and enum names, and enum values.
- A language server (`wisp lsp`) with diagnostics as you type, completion (members, struct fields, `import` paths), hover with doc comments, signature help, outline, go to definition, rename and quick fixes. It needs the `wisp` binary on the `PATH`, or the `wisp.serverPath` setting.
- `wisp.toml` support: key and value completion, hover and diagnostics, with the same icon as `.wsp` files. Highlighting there comes from any installed TOML extension.
- Snippets for `main`, `func`, `if`, `ifelse`, `switch`, `for`, `loop`, `struct` and `enum`.
- Bracket matching, auto-closing pairs, comment toggling and `// region` folding.
- A file icon for `.wsp` files.

## Notes

The grammar is written by hand and follows the language as it evolves, so some constructs may be missing or colored differently than expected. Issues and suggestions are welcome in the repository.
