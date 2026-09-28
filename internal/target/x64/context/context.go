package x64context

type Context struct {
	stackOffset map[string]int
	stackSize   int

	registers map[string]bool
}

func NewContext() *Context {
	return &Context{stackOffset: map[string]int{}, registers: map[string]bool{}}
}

func (c *Context) Get(ident string) (int, bool) {
	value, ok := c.stackOffset[ident]
	return value, ok
}

func (c *Context) AlignTo(align int) {
	c.stackSize = (c.stackSize + align - 1) / align * align
}

func (c *Context) Set(ident string, size int) int {
	c.stackSize += size
	c.stackOffset[ident] = c.stackSize

	return c.stackSize
}
