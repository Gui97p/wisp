package x64context

import "github.com/Gui97p/wisp/internal/analyser"

type Context struct {
	stackOffset map[*analyser.Symbol]int
	stackSize   int

	registers map[Register]bool

	pushed int
}

func NewContext() *Context {
	return &Context{stackOffset: map[*analyser.Symbol]int{}, registers: map[Register]bool{}}
}

func (c *Context) Get(ident *analyser.Symbol) (int, bool) {
	value, ok := c.stackOffset[ident]
	return value, ok
}

func (c *Context) Size() int {
	return c.stackSize
}

func (c *Context) Reserve(size int) int {
	c.stackSize += size
	return c.stackSize
}

func (c *Context) AlignTo(align int) int {
	c.stackSize = (c.stackSize + align - 1) / align * align
	return c.stackSize
}

func (c *Context) Set(ident *analyser.Symbol, size int) int {
	c.stackSize += size
	c.stackOffset[ident] = c.stackSize

	return c.stackSize
}

func (c *Context) AddPushed(n int) {
	c.pushed += n
}

func (c *Context) Pushed() int {
	return c.pushed
}
