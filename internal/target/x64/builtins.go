package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/target"
)

func (t *X64Target) useStreq() string {
	if !t.postlude.define("streq") {
		return "__wisp_streq"
	}
	t.postlude.print(`__wisp_streq:
	xor eax, eax
	cmp rsi, rcx
	jne .done
	mov rcx, rsi
	mov rsi, rdi
	mov rdi, rdx
	repe cmpsb
	sete al
.done:
	ret
`)
	return "__wisp_streq"
}

func (t *X64Target) useStrfind() string {
	if !t.postlude.define("strfind") {
		return "__wisp_strfind"
	}
	t.postlude.print(`__wisp_strfind:
	mov r8, rdi
	mov r9, rsi
    mov r10, rdx
    mov r11, rcx
    sub r11, r9
    jb .no
    inc r11
.loop:
    mov rdi, r8
    mov rsi, r10
    mov rcx, r9
    xor eax, eax
    repe cmpsb
    je .yes
    inc r10
    dec r11
    jnz .loop
.no:
    xor eax, eax
    ret
.yes:
    mov eax, 1
    ret
`)
	return "__wisp_strfind"
}

func (t *X64Target) useStrinstrings() string {
	if !t.postlude.define("strinstrings") {
		return "__wisp_in_strings"
	}
	t.postlude.print(`__wisp_in_strings:
    mov r8, rdi
    mov r9, rsi
    mov r10, rcx
.loop:
    test r10, r10
    jz .no
    cmp [rdx+8], r9
    jne .next
    mov rdi, r8
    mov rsi, [rdx]
    mov rcx, r9
    xor eax, eax
    repe cmpsb
    je .yes
.next:
    add rdx, 16
    dec r10
    jmp .loop
.no:
    xor eax, eax
    ret
.yes:
    mov eax, 1
    ret
`)
	return "__wisp_in_strings"
}

func (t *X64Target) compileBuiltin(name string, expr *ast.CallExpr) (Operand, error) {
	switch name {
	case "emit":
		return t.compileEmitBuiltin(expr)
	case "len":
		return t.compileLenBuiltin(expr)
	case "panic":
		return t.compilePanicBuiltin(expr)
	default:
		return nil, fmt.Errorf("x86-64: builtin %s not supported", name)
	}
}

func (t *X64Target) compileEmitBuiltin(expr *ast.CallExpr) (Operand, error) {
	mod, err := target.Lookup(t.target)
	if err != nil {
		return nil, err
	}
	if mod.RuntimeSource == "" {
		return nil, fmt.Errorf("x86-64: target %s does not provide emit", mod.Name)
	}

	if len(expr.Args) != 1 {
		return nil, fmt.Errorf("x86-64: emit only accepts 1 parameter")
	}

	_, err = t.compileCall("__wisp_emit", true, expr.Args, nil)
	return nil, err
}

func (t *X64Target) compileLenBuiltin(expr *ast.CallExpr) (Operand, error) {
	switch arg := t.info.Types[expr.Args[0]].(type) {
	case analyser.ArrayType:
		return Imm(arg.Size), nil
	case analyser.PrimitiveType:
		if arg.Name != "string" {
			return nil, fmt.Errorf("x86-64: cannot get length of %s", arg)
		}
		op, err := t.compileExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}

		return t.field(op, 1), nil
	case analyser.SpanType:
		op, err := t.compileExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}

		return t.field(op, 1), nil
	default:
		return nil, fmt.Errorf("x86-64: cannot get length of %s", arg)
	}
}

func (t *X64Target) compilePanicBuiltin(expr *ast.CallExpr) (Operand, error) {
	mod, err := target.Lookup(t.target)
	if err != nil {
		return nil, err
	}
	if mod.RuntimeSource == "" {
		return nil, fmt.Errorf("x86-64: target %s does not provide panic", mod.Name)
	}

	if len(expr.Args) != 1 {
		return nil, fmt.Errorf("x86-64: panic only accepts 1 parameter")
	}

	_, err = t.compileCall("__wisp_panic", true, expr.Args, nil)
	return nil, err
}
