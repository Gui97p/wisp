package x64context

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

	BP
)

var ParamOrder = []Register{DI, SI, DX, CX, R8, R9}
var RegisterOrder = []Register{AX, CX, DX, DI, SI, R8, R9, R10, R11}

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

	BP: {1: "bpl", 2: "bp", 4: "ebp", 8: "rbp"},
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

func (c *Context) AllocatedRegisters() []Register {
	regs := []Register{}
	for reg := range c.registers {
		regs = append(regs, reg)
	}
	return regs
}

func (c *Context) FreeRegister(reg Register) {
	delete(c.registers, reg)
}

func (c *Context) AllocFreeRegister() Register {
	for _, reg := range RegisterOrder {
		if _, ok := c.registers[reg]; !ok {
			c.AllocRegister(reg)
			return reg
		}
	}
	panic("no free registers")
}
