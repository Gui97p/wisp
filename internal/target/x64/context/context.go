package x64context

type Context struct {
	stackOffset map[string]int
	stackSize   int

	registers map[Reg]bool
}

func NewContext() *Context {
	return &Context{stackOffset: map[string]int{}, registers: map[Reg]bool{}}
}

func (c *Context) Get(ident string) (int, bool) {
	value, ok := c.stackOffset[ident]
	return value, ok
}

func (c *Context) Size() int {
	return c.stackSize
}

func (c *Context) AlignTo(align int) int {
	c.stackSize = (c.stackSize + align - 1) / align * align
	return c.stackSize
}

func (c *Context) Set(ident string, size int) int {
	c.stackSize += size
	c.stackOffset[ident] = c.stackSize

	return c.stackSize
}
