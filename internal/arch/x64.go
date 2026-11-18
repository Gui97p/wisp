package arch

import (
	"fmt"
	"sort"
	"strings"
)

type RegClass int

const (
	General RegClass = iota
	Vector
	Segment
	Pointer
)

type Register struct {
	Name      string
	Canonical string
	Width     int
	Class     RegClass
}

var x64Registers = buildX64Registers()

func buildX64Registers() map[string]Register {
	regs := make(map[string]Register)
	add := func(name, canonical string, width int, class RegClass) {
		regs[name] = Register{Name: name, Canonical: canonical, Width: width, Class: class}
	}

	legacy := []struct{ q, d, w, l, h string }{
		{"rax", "eax", "ax", "al", "ah"},
		{"rbx", "ebx", "bx", "bl", "bh"},
		{"rcx", "ecx", "cx", "cl", "ch"},
		{"rdx", "edx", "dx", "dl", "dh"},
	}
	for _, r := range legacy {
		add(r.q, r.q, 64, General)
		add(r.d, r.q, 32, General)
		add(r.w, r.q, 16, General)
		add(r.l, r.q, 8, General)
		add(r.h, r.q, 8, General)
	}
	index := []struct{ q, d, w, l string }{
		{"rsi", "esi", "si", "sil"},
		{"rdi", "edi", "di", "dil"},
		{"rbp", "ebp", "bp", "bpl"},
		{"rsp", "esp", "sp", "spl"},
	}
	for _, r := range index {
		add(r.q, r.q, 64, General)
		add(r.d, r.q, 32, General)
		add(r.w, r.q, 16, General)
		add(r.l, r.q, 8, General)
	}
	for n := 8; n <= 15; n++ {
		q := fmt.Sprintf("r%d", n)
		add(q, q, 64, General)
		add(q+"d", q, 32, General)
		add(q+"w", q, 16, General)
		add(q+"b", q, 8, General)
	}
	for n := 0; n <= 15; n++ {
		x := fmt.Sprintf("xmm%d", n)
		add(x, x, 128, Vector)
		add(fmt.Sprintf("ymm%d", n), x, 256, Vector)
	}
	add("fs", "fs", 16, Segment)
	add("gs", "gs", 16, Segment)
	add("rip", "rip", 64, Pointer)
	return regs
}

func LookupRegister(name string) (Register, bool) {
	r, ok := x64Registers[strings.ToLower(name)]
	return r, ok
}

var sizeKeywords = map[string]bool{
	"byte": true, "word": true, "dword": true, "qword": true, "tword": true,
	"oword": true, "yword": true, "zword": true, "rel": true,
}

func IsSizeKeyword(name string) bool {
	return sizeKeywords[strings.ToLower(name)]
}

var CallerSaved = []string{
	"rax", "rcx", "rdx", "rsi", "rdi", "r8", "r9", "r10", "r11",
	"xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5", "xmm6", "xmm7",
	"xmm8", "xmm9", "xmm10", "xmm11", "xmm12", "xmm13", "xmm14", "xmm15",
}

type Effects struct {
	Touches []string
	Stack   bool
}

func InstrEffects(prefixes []string, mnemonic string, operands int) Effects {
	m := strings.ToLower(mnemonic)
	repeated := len(prefixes) > 0
	var e Effects
	touch := func(names ...string) { e.Touches = append(e.Touches, names...) }

	switch m {
	case "div", "idiv":
		touch("rax", "rdx")
	case "mul", "imul":
		if operands == 1 {
			touch("rax", "rdx")
		}
	case "cqo", "cdq", "cwd", "cwde", "cdqe", "cbw":
		touch("rax", "rdx")
	case "syscall":
		touch("rax", "rcx", "r11")
	case "sysenter", "int":
		touch("rax", "rcx", "r11")
	case "cpuid":
		touch("rax", "rbx", "rcx", "rdx")
	case "rdtsc", "rdtscp":
		touch("rax", "rdx", "rcx")
	case "xgetbv":
		touch("rax", "rdx", "rcx")
	case "movsb", "movsw", "movsd", "movsq", "cmpsb", "cmpsw", "cmpsq":
		if m == "movsd" && operands > 0 {
			break
		}
		touch("rsi", "rdi")
		if repeated {
			touch("rcx")
		}
	case "stosb", "stosw", "stosd", "stosq", "scasb", "scasw", "scasq":
		touch("rdi", "rax")
		if repeated {
			touch("rcx")
		}
	case "lodsb", "lodsw", "lodsd", "lodsq":
		touch("rsi", "rax")
		if repeated {
			touch("rcx")
		}
	case "loop", "loope", "loopne", "loopz", "loopnz", "jrcxz", "jecxz":
		touch("rcx")
	case "cmpxchg":
		touch("rax")
	case "cmpxchg16b":
		touch("rax", "rdx", "rbx", "rcx")
	case "cmpxchg8b":
		touch("rax", "rdx", "rbx", "rcx")
	case "call":
		touch(CallerSaved...)
		e.Stack = true
	case "push", "pop", "pushf", "popf", "pushfq", "popfq", "ret", "retn", "leave", "enter":
		e.Stack = true
		touch("rsp")
	}
	return e
}

func Canonical(names []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, n := range names {
		if r, ok := LookupRegister(n); ok {
			n = r.Canonical
		}
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

func RegisterNames() []string {
	names := make([]string, 0, len(x64Registers))
	for name, r := range x64Registers {
		if r.Class == General || r.Class == Vector {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
