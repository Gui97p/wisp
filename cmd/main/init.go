package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/target"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:          "init",
	Short:        "Initialize a new Wisp project in the current directory",
	Long:         "Initialize a new Wisp project in the current directory.\nWithout a subcommand it creates a native project for the host target.",
	Args:         cobra.NoArgs,
	RunE:         runInit,
	SilenceUsage: true,
}

var initLuaCmd = &cobra.Command{
	Use:          "lua",
	Short:        "Initialize a project that targets Lua",
	Args:         cobra.NoArgs,
	RunE:         runInitLua,
	SilenceUsage: true,
}

var initLibCmd = &cobra.Command{
	Use:          "lib",
	Short:        "Initialize a library project (object files, no main)",
	Args:         cobra.NoArgs,
	RunE:         runInitLib,
	SilenceUsage: true,
}

var initTargetCmd = &cobra.Command{
	Use:   "target <name>",
	Short: "Add a target of your own (kernel userspace), creating the project if there is none",
	Long: `Adds a target of your own (kernel userspace) to the project and writes its
start and runtime files. Without a wisp.toml it creates the project too, with
the new target as its default. In an existing project the default target is
kept, unless --default is given.`,
	Args:         cobra.ExactArgs(1),
	RunE:         runInitTarget,
	SilenceUsage: true,
}

var makeDefault bool

func init() {
	initTargetCmd.Flags().BoolVar(&makeDefault, "default", false, "make the new target the default of the project")
	initCmd.AddCommand(initLuaCmd, initLibCmd, initTargetCmd)
}

const nativeTomlContent = `target = "%s"

# entry = "main.wsp"

[output]
bin = "bin/main"
# obj = "build/obj"
# asm = "build/asm"
`

const luaTomlContent = `target = "lua"

# entry = "main.wsp"

[output]
lua = "dist"
`

const libTomlContent = `[output]
obj = "build/obj"
asm = "build/asm"
`

const nativeMainContent = `func main() int {
    return 0;
}
`

const luaMainContent = `func main() {
    emit("Hello, Wisp!");
}
`

const libContent = `export func add(int a, b) int => a + b;
`

const targetTomlContent = `
[targets.%[1]s]
arch = "x64"
os = "%[2]s"
entry = "_start"
runtime = "targets/%[1]s/runtime.asm"
start = "targets/%[1]s/start.asm"
link = ["-z", "noexecstack"]
`

const targetStartContent = `; Entry point of the %[1]s target. It runs when the kernel starts the program:
; it calls main and then has to leave the program the way this kernel requires.
; The exit below is the Linux convention and only a placeholder, replace it.

global _start
extern main

section .text
_start:
    xor ebp, ebp
    call main
    mov rdi, rax
    mov eax, 60
    syscall

section .note.GNU-stack noalloc noexec nowrite progbits
`

const targetRuntimeContent = `; Runtime of the %[1]s target: routines the compiler calls on its own. Only the
; ones a program uses are needed, so start with the first two.
;
; __wisp_emit backs the emit builtin: rdi holds the address of the text and
; rsi its length. It writes the text and a line feed to the standard output.
; __wisp_panic backs the panic builtin and the checks the compiler inserts, such
; as division by zero: rdi holds the address of the message and rsi its length.
; It writes "panic: ", the message and a line feed to the error output, then
; ends the program with a failure status. It never returns.
;
; Needed only when the program uses them:
; __wisp_error_emit, for a main that can fail and ends in an error: rdi holds the
; address of an error record. It writes the error to the error output and returns.
; __wisp_malloc backs malloc: rdi holds the size in bytes. It returns the address
; in rax and 0 in rdx, or 0 in rax and the address of an error record in rdx
; when it fails (a negative size fails too).
; __wisp_realloc backs realloc: rdi holds the address (or 0) and rsi the new
; size. It returns like malloc and keeps the old block when it fails.
; __wisp_free backs free: rdi holds the address (or 0). It returns nothing.
;
; They follow the System V calling convention and must keep rbx, rbp and r12
; to r15.
;
; An error record is 64 bytes: the domain as address (offset 0) and length (8),
; the code (16), the message as address (24) and length (32), the file as
; address (40) and length (48), and the line (56).
; The Linux calls below are only placeholders, replace them with your kernel's.

global __wisp_emit
global __wisp_panic

section .rodata
__wisp_panic_prefix: db "panic: "

section .text
__wisp_emit:
    mov rdx, rsi
    mov rsi, rdi
    mov edi, 1
    mov eax, 1
    syscall
    push 10
    mov rsi, rsp
    mov edx, 1
    mov edi, 1
    mov eax, 1
    syscall
    pop rax
    ret

__wisp_panic:
    push rdi
    push rsi
    mov eax, 1
    mov edi, 2
    lea rsi, [rel __wisp_panic_prefix]
    mov edx, 7
    syscall
    pop rdx
    pop rsi
    mov eax, 1
    mov edi, 2
    syscall
    push 10
    mov rsi, rsp
    mov edx, 1
    mov edi, 2
    mov eax, 1
    syscall
    mov edi, 1
    mov eax, 60
    syscall

section .note.GNU-stack noalloc noexec nowrite progbits
`

func runInit(cmd *cobra.Command, args []string) error {
	return scaffold(fmt.Sprintf(nativeTomlContent, runtime.GOOS+"_x64"), map[string]string{"main.wsp": nativeMainContent})
}

func runInitLua(cmd *cobra.Command, args []string) error {
	return scaffold(luaTomlContent, map[string]string{"main.wsp": luaMainContent})
}

func runInitLib(cmd *cobra.Command, args []string) error {
	return scaffold(libTomlContent, map[string]string{"lib.wsp": libContent})
}

func scaffold(toml string, files map[string]string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	tomlPath := filepath.Join(cwd, "wisp.toml")
	if _, err := os.Stat(tomlPath); err == nil {
		return fmt.Errorf("wisp.toml already exists in %s", cwd)
	}
	if err := os.WriteFile(tomlPath, []byte(toml), 0644); err != nil {
		return err
	}

	for name, content := range files {
		path := filepath.Join(cwd, name)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	fmt.Printf("initialized wisp project in %s\n", cwd)
	return nil
}

func runInitTarget(cmd *cobra.Command, args []string) error {
	name := args[0]
	if err := target.CheckName(name); err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root := module.FindProjectRoot(cwd)

	tomlPath := filepath.Join(root, "wisp.toml")
	existing, err := os.ReadFile(tomlPath)
	fresh := err != nil
	if fresh {
		root = cwd
		tomlPath = filepath.Join(root, "wisp.toml")
		existing = fmt.Appendf(nil, nativeTomlContent, name)
	} else {
		cfg, err := module.LoadConfig(root)
		if err != nil {
			return err
		}
		if _, ok := cfg.Targets[name]; ok {
			return fmt.Errorf("target %q is already defined in wisp.toml", name)
		}
	}

	dir := filepath.Join(root, "targets", name)
	startPath := filepath.Join(dir, "start.asm")
	runtimePath := filepath.Join(dir, "runtime.asm")
	for _, p := range []string{startPath, runtimePath} {
		if _, err := os.Stat(p); err == nil {
			return fmt.Errorf("%s already exists", p)
		}
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(startPath, fmt.Appendf(nil, targetStartContent, name), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(runtimePath, fmt.Appendf(nil, targetRuntimeContent, name), 0644); err != nil {
		return err
	}

	toml := string(existing)
	if !strings.HasSuffix(toml, "\n") {
		toml += "\n"
	}
	if makeDefault && !fresh {
		toml = setDefaultTarget(toml, name)
	}
	toml += fmt.Sprintf(targetTomlContent, name, strings.TrimSuffix(name, "_x64"))
	if err := os.WriteFile(tomlPath, []byte(toml), 0644); err != nil {
		return err
	}

	if fresh {
		mainPath := filepath.Join(root, "main.wsp")
		if _, err := os.Stat(mainPath); err != nil {
			if err := os.WriteFile(mainPath, []byte(nativeMainContent), 0644); err != nil {
				return err
			}
		}
		fmt.Printf("created a project in %s, with %q as its target\n", root, name)
	} else {
		fmt.Printf("added target %q to %s\n", name, tomlPath)
	}
	fmt.Printf("  %s\n  %s\n", startPath, runtimePath)
	fmt.Println("both files use Linux syscalls as a placeholder, adjust them to your kernel")
	fmt.Printf("then: wisp build --target %s\n", name)
	return nil
}

func setDefaultTarget(toml, name string) string {
	line := fmt.Sprintf("target = %q", name)
	lines := strings.Split(toml, "\n")
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "[") {
			break
		}
		if strings.HasPrefix(trimmed, "target") && strings.Contains(trimmed, "=") && !strings.HasPrefix(trimmed, "#") {
			lines[i] = line
			return strings.Join(lines, "\n")
		}
	}
	return line + "\n" + toml
}
