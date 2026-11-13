package x64context

import (
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
)

type Context struct {
	symbolRegisters map[*analyser.Symbol]Register
	homes           map[Register]bool
	savedRegisters  map[Register]bool
	stackOffset     map[*analyser.Symbol]int
	stackSize       int

	registers map[Register]bool

	pushed    int
	optimized bool

	loopStack []LoopContext
}

func NewContext() *Context {
	return &Context{
		symbolRegisters: map[*analyser.Symbol]Register{},
		homes:           map[Register]bool{},
		savedRegisters:  map[Register]bool{},
		stackOffset:     map[*analyser.Symbol]int{},
		registers:       map[Register]bool{},
		loopStack:       []LoopContext{},
	}
}

func (c *Context) Get(ident *analyser.Symbol) (int, bool) {
	value, ok := c.stackOffset[ident]
	return value, ok
}

func (c *Context) Register(ident *analyser.Symbol) (Register, bool) {
	value, ok := c.symbolRegisters[ident]
	return value, ok
}

func (c *Context) SavedRegisters() []Register {
	regs := make([]Register, 0, len(c.savedRegisters))
	for r, b := range c.savedRegisters {
		if b {
			regs = append(regs, r)
		}
	}
	slices.Sort(regs)
	return regs
}

type Binding struct {
	Symbol   *analyser.Symbol
	Register Register
}

func (c *Context) IsHome(reg Register) bool {
	return c.homes[reg]
}

func (c *Context) XmmBindings() []Binding {
	var out []Binding
	for sym, reg := range c.symbolRegisters {
		if reg.Class() == SSE {
			out = append(out, Binding{sym, reg})
		}
	}
	slices.SortFunc(out, func(a, b Binding) int { return int(a.Register) - int(b.Register) })
	return out
}

func (c *Context) Bind(ident *analyser.Symbol, reg Register) {
	c.symbolRegisters[ident] = reg
	c.homes[reg] = true
	if slices.Contains(SavedOrder, reg) {
		c.savedRegisters[reg] = true
	}
}

func (c *Context) Unbind(ident *analyser.Symbol) {
	reg, ok := c.symbolRegisters[ident]
	if !ok {
		return
	}
	delete(c.symbolRegisters, ident)
	delete(c.homes, reg)
}

func (c *Context) Size() int {
	return c.stackSize
}

func (c *Context) Reserve(size int) int {
	align := max(min(size, 8), 1)
	c.stackSize = (c.stackSize + size + align - 1) / align * align
	return c.stackSize
}

func (c *Context) AlignTo(align int) int {
	c.stackSize = (c.stackSize + align - 1) / align * align
	return c.stackSize
}

func (c *Context) Set(ident *analyser.Symbol, size int) int {
	align := max(min(size, 8), 1)
	c.stackSize = (c.stackSize + size + align - 1) / align * align
	c.stackOffset[ident] = c.stackSize

	return c.stackSize
}

func (c *Context) AddPushed(n int) {
	c.pushed += n
}

func (c *Context) Pushed() int {
	return c.pushed
}
