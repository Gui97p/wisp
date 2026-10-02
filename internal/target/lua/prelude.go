package lua

import (
	"regexp"
	"strings"
)

type helper struct {
	name string
	deps []string
	code string
}

var preludeHelpers = []helper{
	{name: "__wisp_len", code: `local function __wisp_len(t)
	local n = 0
	while t[n] ~= nil do n = n + 1 end
	return n
end
`},
	{name: "__wisp_slice", code: `local function __wisp_slice(t, a, b)
	local r = {}
	local j = 0
	for i = a, b - 1 do
		r[j] = t[i]
		j = j + 1
	end
	return r
end
`},
	{name: "__wisp_idiv", code: `local function __wisp_idiv(a, b)
	local q = a // b
	if q < 0 and q * b ~= a then q = q + 1 end
	return q
end
`},
	{name: "__wisp_imod", code: `local function __wisp_imod(a, b)
	return math.fmod(a, b)
end
`},
	{name: "__wisp_udiv", code: `local function __wisp_udiv(a, b)
	if b < 0 then
		if math.ult(a, b) then return 0 end
		return 1
	end
	if a >= 0 then return a // b end
	local q = ((a >> 1) // b) << 1
	local r = a - q * b
	if not math.ult(r, b) then q = q + 1 end
	return q
end
`},
	{name: "__wisp_umod", deps: []string{"__wisp_udiv"}, code: `local function __wisp_umod(a, b)
	return a - __wisp_udiv(a, b) * b
end
`},
	{name: "__wisp_sar", code: `local function __wisp_sar(a, n)
	if n >= 63 then
		if a < 0 then return -1 end
		return 0
	end
	return a // (1 << n)
end
`},
	{name: "__wisp_ule", code: `local function __wisp_ule(a, b)
	return not math.ult(b, a)
end
`},
	{name: "__wisp_ugt", code: `local function __wisp_ugt(a, b)
	return math.ult(b, a)
end
`},
	{name: "__wisp_uge", code: `local function __wisp_uge(a, b)
	return not math.ult(a, b)
end
`},
	{name: "__wisp_f2i", code: `local function __wisp_f2i(x)
	if x >= 0 then return math.floor(x) end
	return math.ceil(x)
end
`},
	{name: "__wisp_pack", code: `local function __wisp_pack(...)
	local r = {}
	for i = 1, select("#", ...) do
		r[i - 1] = (select(i, ...))
	end
	return r
end
`},
	{name: "__wisp_contains", deps: []string{"__wisp_len"}, code: `local function __wisp_contains(t, v)
	local n = __wisp_len(t)
	for i = 0, n - 1 do
		if t[i] == v then return true end
	end
	return false
end
`},
}

var helperRef = regexp.MustCompile(`__wisp_[A-Za-z0-9_]+`)

func preludeFor(body string) string {
	used := map[string]bool{}
	for _, name := range helperRef.FindAllString(body, -1) {
		used[name] = true
	}

	byName := make(map[string]helper, len(preludeHelpers))
	for _, h := range preludeHelpers {
		byName[h.name] = h
	}

	var need func(name string)
	need = func(name string) {
		h, ok := byName[name]
		if !ok {
			return
		}
		for _, dep := range h.deps {
			if !used[dep] {
				used[dep] = true
				need(dep)
			}
		}
	}
	for name := range used {
		need(name)
	}

	var b strings.Builder
	for _, h := range preludeHelpers {
		if used[h.name] {
			b.WriteString(h.code)
			b.WriteByte('\n')
		}
	}
	return b.String()
}
