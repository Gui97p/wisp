# Wisp Language Tools

Language support for [Wisp](https://github.com/Gui97p/wisp), a personal compiled language written in Go. Wisp aims to give low-level control with a high-level feel, without a garbage collector. Source files use the `.wsp` extension.

## Features

- Syntax highlighting for keywords, types, numbers, strings, characters, comments, operators, functions, struct and enum names, and enum values.
- A language server (`wisp lsp`) with diagnostics as you type, completion (members, struct fields, `import` paths), hover with doc comments, signature help, outline, go to definition, rename and quick fixes. Picking a module name or one of its members from the completion adds the `import` by itself, and saving removes the imports that nothing uses, adds the missing ones when only one module fits, and formats the file with the same style as `wisp fmt`. It needs the `wisp` binary on the `PATH`, or the `wisp.serverPath` setting.
- `wisp.toml` support: key and value completion, hover and diagnostics, with the same icon as `.wsp` files. Highlighting there comes from any installed TOML extension.
- Snippets for `main`, `func`, `if`, `ifelse`, `match`, `each`, `for`, `loop`, `struct` and `enum`.
- Bracket matching, auto-closing pairs, comment toggling and `// region` folding.
- A file icon for `.wsp` files.

## Requirements

The extension talks to the Wisp language server, so it needs the `wisp` compiler. Install it with one command, which puts it in `~/.wisp` (Linux and WSL) or `%LOCALAPPDATA%\wisp` (Windows) and adds it to the `PATH`:

```sh
curl -fsSL https://raw.githubusercontent.com/Gui97p/wisp/main/scripts/install.sh | sh
```

```powershell
irm https://raw.githubusercontent.com/Gui97p/wisp/main/scripts/install.ps1 | iex
```

Restart VS Code afterwards so it sees the new `PATH`, or set `wisp.serverPath` to the full path of the binary. `wisp doctor` says what is missing when something does not work, and `wisp update` brings the compiler to the newest release. The releases are at [github.com/Gui97p/wisp/releases](https://github.com/Gui97p/wisp/releases).

## Notes

The grammar is written by hand and follows the language as it evolves, so some constructs may be missing or colored differently than expected. Issues and suggestions are welcome in the repository.
