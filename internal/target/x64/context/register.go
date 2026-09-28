package x64context

var registers = map[string]map[int]string{
	"ax":  {1: "al", 2: "ax", 4: "eax", 8: "rax"},
	"bx":  {1: "bl", 2: "bx", 4: "ebx", 8: "rbx"},
	"cx":  {1: "cl", 2: "cx", 4: "ecx", 8: "rcx"},
	"dx":  {1: "dl", 2: "dx", 4: "edx", 8: "rdx"},
	"di":  {1: "dil", 2: "di", 4: "edi", 8: "rdi"},
	"si":  {1: "sil", 2: "si", 4: "esi", 8: "rsi"},
	"r8":  {1: "r8b", 2: "r8w", 4: "r8d", 8: "r8"},
	"r9":  {1: "r9b", 2: "r9w", 4: "r9d", 8: "r9"},
	"r10": {1: "r10b", 2: "r10w", 4: "r10d", 8: "r10"},
	"r11": {1: "r11b", 2: "r11w", 4: "r11d", 8: "r11"},
	"r12": {1: "r12b", 2: "r12w", 4: "r12d", 8: "r12"},
}

func (c *Context) GetRegister(reg string, size int) string {
	return registers[reg][size]
}

func (c *Context) AllocRegister(reg string) bool {
	if _, ok := c.registers[reg]; ok {
		return false
	}
	c.registers[reg] = true
	return true
}

func (c *Context) FreeRegister(reg string) {
	delete(c.registers, reg)
}
