package x64

import (
	"fmt"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/target"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) useChars() string {
	label, ok := t.createData("__wisp_chars", "")
	if !ok {
		t.rodata.printf("%s db ", label)
		for i := range 255 {
			t.rodata.printf("%d, ", i)
		}
		t.rodata.printf("%d\n", 255)
	}
	return label
}

func (t *X64Target) useErreq() string {
	if !t.postlude.define("erreq") {
		return "__wisp_erreq"
	}
	t.postlude.print(`__wisp_erreq:
	xor eax, eax
	test rdi, rdi
	jz .done
	cmp [rdi+16], rcx
	jne .done
	cmp [rdi+8], rdx
	jne .done
	mov rdi, [rdi]
	mov rcx, rdx
	xor eax, eax
	repe cmpsb
	sete al
.done:
	ret
`)
	return "__wisp_erreq"
}

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
	cld
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
	cld
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

func (t *X64Target) useStrcmp() string {
	if !t.postlude.define("strcmp") {
		return "__wisp_strcmp"
	}
	t.postlude.print(`__wisp_strcmp:
	mov r8, rsi
	mov r9, rcx

	cmp r8, r9
	cmovb rcx, r8

	mov rsi, rdi
	mov rdi, rdx

	cld
	xor eax, eax
	repe cmpsb
	jne .diff

	cmp r8, r9
	je .equal
	jb .less

	mov rax, 1
	ret

.less:
	mov rax, -1
	ret

.diff:
	movzx eax, byte [rsi-1]
	movzx edx, byte [rdi-1]
	sub rax, rdx
	ret

.equal:
	xor eax, eax
	ret
`)
	return "__wisp_strcmp"
}

func (t *X64Target) compileBuiltin(name string, expr *ast.CallExpr) ([]Operand, error) {
	switch name {
	case "emit":
		return t.compileEmitBuiltin(expr)
	case "len":
		return t.compileLenBuiltin(expr)
	case "panic":
		return t.compilePanicBuiltin(expr)
	case "sizeof":
		tp := t.info.SizeOf[expr]
		return []Operand{Imm(sizeOf(tp))}, nil
	case "malloc", "realloc":
		return t.compileAllocatorBuiltin(name, expr)
	case "free":
		return t.compileFreeBuiltin(expr)
	default:
		return nil, fmt.Errorf("x86-64: builtin %s not supported", name)
	}
}

func (t *X64Target) compileEmitBuiltin(expr *ast.CallExpr) ([]Operand, error) {
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

	_, err = t.compileCall("__wisp_emit", true, expr.Args, nil, false)
	return nil, err
}

func (t *X64Target) compileLenBuiltin(expr *ast.CallExpr) ([]Operand, error) {
	var op Operand
	var err error

	switch arg := t.info.Types[expr.Args[0]].(type) {
	case analyser.ArrayType:
		op = Imm(arg.Size)
	case analyser.PrimitiveType:
		if arg.Name != "string" {
			return nil, fmt.Errorf("x86-64: cannot get length of %s", arg)
		}
		op, err = t.compileExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}

		op = t.field(op, 1)
	case analyser.SpanType:
		op, err = t.compileExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}

		op = t.field(op, 1)
	default:
		return nil, fmt.Errorf("x86-64: cannot get length of %s", arg)
	}

	return []Operand{op}, nil
}

func (t *X64Target) compilePanicBuiltin(expr *ast.CallExpr) ([]Operand, error) {
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

	_, err = t.compileCall("__wisp_panic", true, expr.Args, nil, false)
	return nil, err
}

func (t *X64Target) compileAllocatorBuiltin(name string, expr *ast.CallExpr) ([]Operand, error) {
	mod, err := target.Lookup(t.target)
	if err != nil {
		return nil, err
	}
	if mod.RuntimeSource == "" {
		return nil, fmt.Errorf("x86-64: target %s does not provide %s", mod.Name, name)
	}

	return t.compileCall(fmt.Sprintf("__wisp_%s", name), true, expr.Args, t.info.CallReturns[expr], false)
}

func (t *X64Target) compileFreeBuiltin(expr *ast.CallExpr) ([]Operand, error) {
	mod, err := target.Lookup(t.target)
	if err != nil {
		return nil, err
	}
	if mod.RuntimeSource == "" {
		return nil, fmt.Errorf("x86-64: target %s does not provide free", mod.Name)
	}

	if len(expr.Args) == 1 {
		return t.compileCall("__wisp_free", true, expr.Args, nil, false)
	}

	regs := t.ctx.AllocatedRegisters()
	for _, reg := range regs {
		t.pushWord(Reg{reg, 8})
	}

	n := 0
	err = t.eachValue(expr.Args, func(i int, op Operand) error {
		t.pushValue(t.field(op, 0))
		n++
		return nil
	})
	if err != nil {
		return nil, err
	}

	t.prelude.define("__wisp_free")
	for range n {
		t.pop(x64context.DI)
		t.text.printlnt("call __wisp_free")
	}

	for _, reg := range slices.Backward(regs) {
		t.pop(reg)
	}

	return nil, nil
}
