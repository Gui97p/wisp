package x64context

import "slices"

type Register int

const (
	NoReg Register = iota
	AX
	BX
	CX
	DX
	DI
	SI
	R8
	R9
	R10
	R11
	R12
	R13
	R14
	R15

	BP
	SP

	X0
	X1
	X2
	X3
	X4
	X5
	X6
	X7
	X8
	X9
	X10
	X11
	X12
	X13
	X14
	X15
)

type Class int

const (
	Int Class = iota
	SSE
	Memory
)

func (r Register) Class() Class {
	switch r {
	case X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X12, X13, X14, X15:
		return SSE
	}
	return Int
}

var ParamOrder = []Register{DI, SI, DX, CX, R8, R9}
var Win64ParamOrder = []Register{CX, DX, R8, R9}
var ReturnOrder = []Register{AX, DX}
var SavedOrder = []Register{BX, R12, R13, R14, R15}
var RegisterOrder = []Register{AX, CX, DX, DI, SI, R8, R9, R10, R11}
var XMMOrder = []Register{X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X12, X13, X14, X15}
var XMMParamOrder = []Register{X0, X1, X2, X3, X4, X5, X6, X7}
var XMMWin64ParamOrder = []Register{X0, X1, X2, X3}
var XMMSavedOrder = []Register{X8, X9, X10, X11, X12, X13, X14, X15}
var XMMWin64SavedOrder = []Register{X6, X7, X8, X9, X10, X11, X12, X13, X14, X15}
var XMMReturnOrder = []Register{X0, X1}

var registers = map[Register]map[int]string{
	AX:  {1: "al", 2: "ax", 4: "eax", 8: "rax"},
	BX:  {1: "bl", 2: "bx", 4: "ebx", 8: "rbx"},
	CX:  {1: "cl", 2: "cx", 4: "ecx", 8: "rcx"},
	DX:  {1: "dl", 2: "dx", 4: "edx", 8: "rdx"},
	DI:  {1: "dil", 2: "di", 4: "edi", 8: "rdi"},
	SI:  {1: "sil", 2: "si", 4: "esi", 8: "rsi"},
	R8:  {1: "r8b", 2: "r8w", 4: "r8d", 8: "r8"},
	R9:  {1: "r9b", 2: "r9w", 4: "r9d", 8: "r9"},
	R10: {1: "r10b", 2: "r10w", 4: "r10d", 8: "r10"},
	R11: {1: "r11b", 2: "r11w", 4: "r11d", 8: "r11"},
	R12: {1: "r12b", 2: "r12w", 4: "r12d", 8: "r12"},
	R13: {1: "r13b", 2: "r13w", 4: "r13d", 8: "r13"},
	R14: {1: "r14b", 2: "r14w", 4: "r14d", 8: "r14"},
	R15: {1: "r15b", 2: "r15w", 4: "r15d", 8: "r15"},

	BP: {1: "bpl", 2: "bp", 4: "ebp", 8: "rbp"},
	SP: {1: "spl", 2: "sp", 4: "esp", 8: "rsp"},

	X0:  {4: "xmm0", 8: "xmm0"},
	X1:  {4: "xmm1", 8: "xmm1"},
	X2:  {4: "xmm2", 8: "xmm2"},
	X3:  {4: "xmm3", 8: "xmm3"},
	X4:  {4: "xmm4", 8: "xmm4"},
	X5:  {4: "xmm5", 8: "xmm5"},
	X6:  {4: "xmm6", 8: "xmm6"},
	X7:  {4: "xmm7", 8: "xmm7"},
	X8:  {4: "xmm8", 8: "xmm8"},
	X9:  {4: "xmm9", 8: "xmm9"},
	X10: {4: "xmm10", 8: "xmm10"},
	X11: {4: "xmm11", 8: "xmm11"},
	X12: {4: "xmm12", 8: "xmm12"},
	X13: {4: "xmm13", 8: "xmm13"},
	X14: {4: "xmm14", 8: "xmm14"},
	X15: {4: "xmm15", 8: "xmm15"},
}

func (c *Context) GetRegister(reg Register, size int) string {
	return registers[reg][size]
}

func (c *Context) AllocRegister(reg Register) bool {
	if _, ok := c.registers[reg]; ok {
		return false
	}
	c.registers[reg] = true
	return true
}

func (c *Context) IsAllocated(reg Register) bool {
	_, ok := c.registers[reg]
	return ok
}

func (c *Context) AllocatedRegisters() []Register {
	regs := []Register{}
	for reg := range c.registers {
		regs = append(regs, reg)
	}
	return regs
}

func (c *Context) AllocatedOrderRegisters() []Register {
	regs := []Register{}
	for reg := range c.registers {
		if slices.Contains(RegisterOrder, reg) {
			regs = append(regs, reg)
		}
	}
	return regs
}

func (c *Context) FreeRegister(reg Register) {
	if c.homes[reg] {
		return
	}
	delete(c.registers, reg)
}

func (c *Context) AllocFreeRegister() Register {
	for _, reg := range RegisterOrder {
		if _, ok := c.registers[reg]; !ok {
			c.AllocRegister(reg)
			return reg
		}
	}
	return NoReg
}

func (c *Context) AllocFreeRegisterAmong(regs ...Register) Register {
	for _, reg := range regs {
		if _, ok := c.registers[reg]; !ok {
			c.AllocRegister(reg)
			return reg
		}
	}
	return NoReg
}

func (c *Context) AllocFreeXmm() Register {
	xmms := XMMOrder
	if c.optimized {
		xmms = XMMParamOrder
	}

	for _, reg := range xmms {
		if _, ok := c.registers[reg]; !ok {
			c.AllocRegister(reg)
			return reg
		}
	}
	return NoReg
}

type ArgLoc struct {
	Regs  []Register
	Stack int
}
