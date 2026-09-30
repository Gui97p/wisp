package x64

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const startAsm = `global _start
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

func Link(objPaths []string, outputPath string, keepObj bool) error {
	tmp, err := os.MkdirTemp("", "wisp-link-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	startObj := filepath.Join(tmp, "start.o")
	if err := Assemble(startAsm, startObj, filepath.Join(tmp, "start.asm"), false); err != nil {
		return err
	}

	args := []string{"-z", "noexecstack", "-o", outputPath, startObj}
	args = append(args, objPaths...)

	ldCmd := exec.Command("ld", args...)
	ldCmd.Stderr = os.Stderr
	if err := ldCmd.Run(); err != nil {
		return fmt.Errorf("ld failed: %w", err)
	}

	if !keepObj {
		for _, path := range objPaths {
			os.Remove(path)
			removeEmptyDirs(filepath.Dir(path))
		}
	}

	return nil
}
