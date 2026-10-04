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
	Use:          "target <name>",
	Short:        "Add a target of your own (kernel userspace) to the current project",
	Args:         cobra.ExactArgs(1),
	RunE:         runInitTarget,
	SilenceUsage: true,
}

func init() {
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
os = "%[1]s"
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

const targetRuntimeContent = `; Runtime of the %[1]s target: routines the compiler calls on its own.
; __wisp_emit backs the emit builtin: rdi holds the address of the text and
; rsi its length. It writes the text and a line feed to the standard output.
; The Linux write below is only a placeholder, replace it with your kernel's.

global __wisp_emit

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
	if err != nil {
		return fmt.Errorf("no wisp.toml found, run `wisp init` first")
	}

	cfg, err := module.LoadConfig(root)
	if err != nil {
		return err
	}
	if _, ok := cfg.Targets[name]; ok {
		return fmt.Errorf("target %q is already defined in wisp.toml", name)
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
	toml += fmt.Sprintf(targetTomlContent, name)
	if err := os.WriteFile(tomlPath, []byte(toml), 0644); err != nil {
		return err
	}

	fmt.Printf("added target %q to %s\n", name, tomlPath)
	fmt.Printf("  %s\n  %s\n", startPath, runtimePath)
	fmt.Println("both files use Linux syscalls as a placeholder, adjust them to your kernel")
	fmt.Printf("then: wisp build --target %s\n", name)
	return nil
}
