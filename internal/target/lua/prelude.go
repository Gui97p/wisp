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
	{name: "__wisp_panic", code: `local function __wisp_panic(message)
	io.stderr:write("panic: ", message, "\n")
	os.exit(1)
end
`},
	{name: "__wisp_arr", code: `local function __wisp_arr(t, n)
	return {t = t, o = 0, n = n}
end
`},
	{name: "__wisp_empty", code: `local function __wisp_empty()
	return {t = {}, o = 0, n = 0}
end
`},
	{name: "__wisp_nn", deps: []string{"__wisp_panic"}, code: `local function __wisp_nn(x)
	if x == nil then __wisp_panic("null pointer dereference") end
	return x
end
`},
	{name: "__wisp_vget", deps: []string{"__wisp_panic"}, code: `local function __wisp_vget(v, i)
	if i < 0 or i >= v.n then __wisp_panic("index out of range") end
	return v.t[v.o + i]
end
`},
	{name: "__wisp_vset", deps: []string{"__wisp_panic"}, code: `local function __wisp_vset(v, i, x)
	if i < 0 or i >= v.n then __wisp_panic("index out of range") end
	v.t[v.o + i] = x
end
`},
	{name: "__wisp_vslice", deps: []string{"__wisp_panic"}, code: `local function __wisp_vslice(v, a, b)
	a = a or 0
	b = b or v.n
	if a < 0 or b < a or b > v.n then __wisp_panic("slice out of range") end
	return {t = v.t, o = v.o + a, n = b - a}
end
`},
	{name: "__wisp_pslice", deps: []string{"__wisp_nn", "__wisp_panic"}, code: `local function __wisp_pslice(p, a, b)
	__wisp_nn(p)
	if a < 0 or b < a then __wisp_panic("slice out of range") end
	return {t = p.t, o = p.o + a, n = b - a}
end
`},
	{name: "__wisp_pget", deps: []string{"__wisp_nn"}, code: `local function __wisp_pget(p, i)
	__wisp_nn(p)
	return p.t[p.o + i]
end
`},
	{name: "__wisp_pset", deps: []string{"__wisp_nn"}, code: `local function __wisp_pset(p, i, x)
	__wisp_nn(p)
	p.t[p.o + i] = x
end
`},
	{name: "__wisp_get", deps: []string{"__wisp_nn"}, code: `local function __wisp_get(p)
	__wisp_nn(p)
	return p.t[p.o]
end
`},
	{name: "__wisp_set", deps: []string{"__wisp_nn"}, code: `local function __wisp_set(p, x)
	__wisp_nn(p)
	p.t[p.o] = x
end
`},
	{name: "__wisp_pack", code: `local function __wisp_pack(...)
	local r = {}
	local n = select("#", ...)
	for i = 1, n do
		r[i - 1] = (select(i, ...))
	end
	return {t = r, o = 0, n = n}
end
`},
	{name: "__wisp_spread", code: `local function __wisp_spread(v)
	return table.unpack(v.t, v.o, v.o + v.n - 1)
end
`},
	{name: "__wisp_contains", code: `local function __wisp_contains(v, x)
	for i = 0, v.n - 1 do
		if v.t[v.o + i] == x then return true end
	end
	return false
end
`},
	{name: "__wisp_str", code: `local function __wisp_str(v)
	local parts = {}
	local i = 0
	while i < v.n do
		local j = math.min(i + 4096, v.n)
		local chunk = {}
		for k = i, j - 1 do
			chunk[#chunk + 1] = v.t[v.o + k] % 256
		end
		parts[#parts + 1] = string.char(table.unpack(chunk))
		i = j
	end
	return table.concat(parts)
end
`},
	{name: "__wisp_pstr", deps: []string{"__wisp_str"}, code: `local function __wisp_pstr(p, n)
	return __wisp_str({t = p.t, o = p.o, n = n})
end
`},
	{name: "__wisp_sget", deps: []string{"__wisp_panic"}, code: `local function __wisp_sget(s, i)
	local c = string.byte(s, i + 1)
	if i < 0 or c == nil then __wisp_panic("index out of range") end
	return c
end
`},
	{name: "__wisp_mlen", code: `local function __wisp_mlen(m)
	local n = 0
	for _ in pairs(m) do n = n + 1 end
	return n
end
`},
	{name: "__wisp_padd", deps: []string{"__wisp_nn"}, code: `local function __wisp_padd(p, k)
	__wisp_nn(p)
	return {t = p.t, o = p.o + k}
end
`},
	{name: "__wisp_vref", deps: []string{"__wisp_panic"}, code: `local function __wisp_vref(v, i)
	if i < 0 or i >= v.n then __wisp_panic("index out of range") end
	return {t = v.t, o = v.o + i}
end
`},
	{name: "__wisp_pdiff", deps: []string{"__wisp_nn"}, code: `local function __wisp_pdiff(p, q)
	__wisp_nn(p)
	__wisp_nn(q)
	return p.o - q.o
end
`},
	{name: "__wisp_peq", code: `local function __wisp_peq(p, q)
	if p == nil or q == nil then return p == q end
	if p.t == q.t and p.o == q.o then return true end
	local a, b = p.t[p.o], q.t[q.o]
	return type(a) == "table" and a == b
end
`},
	{name: "__wisp_plt", deps: []string{"__wisp_nn"}, code: `local function __wisp_plt(p, q)
	__wisp_nn(p)
	__wisp_nn(q)
	return p.o < q.o
end
`},
	{name: "__wisp_ple", deps: []string{"__wisp_nn"}, code: `local function __wisp_ple(p, q)
	__wisp_nn(p)
	__wisp_nn(q)
	return p.o <= q.o
end
`},
	{name: "__wisp_vptr", code: `local function __wisp_vptr(v)
	return {t = v.t, o = v.o}
end
`},
	{name: "__wisp_shared", code: `local __wisp_shared = package.loaded.__wisp_shared or {next = 268435456, regs = {}, free = {}}
package.loaded.__wisp_shared = __wisp_shared
`},
	{name: "__wisp_fds", deps: []string{"__wisp_shared"}, code: `local __wisp_fds = __wisp_shared.fds or {}
__wisp_shared.fds = __wisp_fds
`},
	{name: "__wisp_cstr", code: `local function __wisp_cstr(p)
	if p == nil then return "" end
	local parts = {}
	local i = 0
	while true do
		local c = p.t[p.o + i]
		if c == nil or c == 0 then break end
		parts[#parts + 1] = c % 256
		i = i + 1
	end
	return string.char(table.unpack(parts))
end
`},
	{name: "__wisp_class", code: `local function __wisp_class(n)
	local c = 16
	while c < n do c = c * 2 end
	return c
end
`},
	{name: "__wisp_alloc", deps: []string{"__wisp_shared", "__wisp_class"}, code: `local function __wisp_alloc(n)
	if n < 0 or n > 1099511627776 then
		return nil, {domain = "runtime.AllocError", code = 1, message = "out of memory", file = "", line = 0}
	end
	local c = __wisp_class(n)
	local list = __wisp_shared.free[c]
	local reuse = list and table.remove(list)
	return {raw = n, cls = c, reuse = reuse}, nil
end
`},
	{name: "__wisp_realloc", deps: []string{"__wisp_alloc", "__wisp_class"}, code: `local function __wisp_realloc(p, n)
	if p == nil then return __wisp_alloc(n) end
	local c = __wisp_class(n)
	if p.t ~= nil and p.t.__cls == c and p.o == 0 then
		return {raw = n, cls = c, same = p}, nil
	end
	local r, err = __wisp_alloc(n)
	if err ~= nil then return nil, err end
	r.src = p
	return r, nil
end
`},
	{name: "__wisp_free", deps: []string{"__wisp_shared"}, code: `local function __wisp_free(...)
	for i = 1, select("#", ...) do
		local p = (select(i, ...))
		if type(p) == "table" and p.t ~= nil and p.o == 0 then
			local t = p.t
			local c = t.__cls
			if c ~= nil and not t.__freed then
				t.__freed = true
				local list = __wisp_shared.free[c]
				if list == nil then
					list = {}
					__wisp_shared.free[c] = list
				end
				list[#list + 1] = t
			end
		end
	end
end
`},
	{name: "__wisp_cast", code: `local function __wisp_cast(r, zero, size)
	if r == nil or r.t ~= nil then return r end
	if r.same ~= nil then return r.same end
	local count = math.floor(r.raw / size)
	local t = r.reuse
	if t ~= nil then
		for k in pairs(t) do
			if type(k) == "number" then t[k] = nil end
		end
		t.__freed = nil
	else
		t = {}
	end
	for i = 0, count - 1 do
		t[i] = zero()
	end
	t.__cls = r.cls
	local s = r.src
	if s ~= nil and s.t ~= nil then
		for i = 0, count - 1 do
			local v = s.t[s.o + i]
			if v == nil then break end
			t[i] = v
		end
	end
	return {t = t, o = 0, n = count}
end
`},
	{name: "__wisp_addr", deps: []string{"__wisp_shared"}, code: `local function __wisp_addr(p, size)
	if p == nil then return 0 end
	local t = p.t
	local base = t.__addr
	if base == nil then
		base = __wisp_shared.next
		__wisp_shared.next = base + 268435456
		t.__addr = base
		__wisp_shared.regs[base] = t
	end
	return base + p.o * size
end
`},
	{name: "__wisp_fromaddr", deps: []string{"__wisp_shared"}, code: `local function __wisp_fromaddr(a, size)
	if a == 0 then return nil end
	local base = a - a % 268435456
	local t = __wisp_shared.regs[base]
	if t == nil then return nil end
	return {t = t, o = math.floor((a - base) / size)}
end
`},
	{name: "__wisp_bytes", code: `local function __wisp_bytes(s)
	local t = {}
	for i = 1, #s do
		t[i - 1] = string.byte(s, i)
	end
	t[#s] = 0
	return {t = t, o = 0, n = #s}
end
`},
	{name: "__wisp_is", code: `local function __wisp_is(e, domain, code)
	return e ~= nil and e.domain == domain and e.code == code
end
`},
	{name: "__wisp_unwrap", deps: []string{"__wisp_panic"}, code: `local function __wisp_unwrap(e)
	__wisp_panic("unwrapped error " .. e.message .. ", raised at " .. e.file .. ":" .. e.line)
end
`},
	{name: "__wisp_idiv", deps: []string{"__wisp_panic"}, code: `local function __wisp_idiv(a, b)
	if b == 0 then __wisp_panic("integer divide by zero") end
	local q = a / b
	if q >= 0 then return math.floor(q) end
	return math.ceil(q)
end
`},
	{name: "__wisp_imod", deps: []string{"__wisp_panic"}, code: `local function __wisp_imod(a, b)
	if b == 0 then __wisp_panic("integer divide by zero") end
	return math.fmod(a, b)
end
`},
	{name: "__wisp_sar", code: `local function __wisp_sar(a, n)
	if n >= 64 then
		if a < 0 then return -1 end
		return 0
	end
	return math.floor(a / 2 ^ n)
end
`},
	{name: "__wisp_shr", code: `local function __wisp_shr(a, n)
	if n >= 64 then return 0 end
	return math.floor(a / 2 ^ n)
end
`},
	{name: "__wisp_shl", code: `local function __wisp_shl(a, n, bits)
	if n >= bits then return 0 end
	if bits < 64 then return (a % 2 ^ (bits - n)) * 2 ^ n end
	return a * 2 ^ n
end
`},
	{name: "__wisp_mulw", code: `local function __wisp_mulw(a, b)
	a = a % 4294967296
	b = b % 4294967296
	local high = math.floor(a / 65536)
	local low = a % 65536
	return (((high * b) % 65536) * 65536 + low * b) % 4294967296
end
`},
	{name: "__wisp_bitop", code: `local function __wisp_bitop(op, a, b, signed)
	local alo = a % 4294967296
	local ahi = ((a - alo) / 4294967296) % 4294967296
	local blo = b % 4294967296
	local bhi = ((b - blo) / 4294967296) % 4294967296
	local lo = op(alo, blo)
	local hi = op(ahi, bhi)
	if signed and hi >= 2147483648 then hi = hi - 4294967296 end
	return hi * 4294967296 + lo
end
`},
	{name: "__wisp_band", deps: []string{"__wisp_bitop"}, code: `local function __wisp_band(a, b, signed)
	return __wisp_bitop(bit32.band, a, b, signed)
end
`},
	{name: "__wisp_bor", deps: []string{"__wisp_bitop"}, code: `local function __wisp_bor(a, b, signed)
	return __wisp_bitop(bit32.bor, a, b, signed)
end
`},
	{name: "__wisp_bxor", deps: []string{"__wisp_bitop"}, code: `local function __wisp_bxor(a, b, signed)
	return __wisp_bitop(bit32.bxor, a, b, signed)
end
`},
	{name: "__wisp_tou64", code: `local function __wisp_tou64(x)
	if x < 0 then return x + 18446744073709551616 end
	return x
end
`},
	{name: "__wisp_toi64", code: `local function __wisp_toi64(x)
	if x >= 9223372036854775808 then return x - 18446744073709551616 end
	return x
end
`},
	{name: "__wisp_f2i", code: `local function __wisp_f2i(x)
	if x >= 0 then return math.floor(x) end
	return math.ceil(x)
end
`},
	{name: "__wisp_box", code: `local function __wisp_box(t, v, e)
	return {t = t, v = v, e = e}
end
`},
	{name: "__wisp_isint", code: `local function __wisp_isint(b)
	local t = b.t
	return t == "int" or t == "int8" or t == "int16" or t == "int32" or t == "int64"
end
`},
	{name: "__wisp_isuint", code: `local function __wisp_isuint(b)
	local t = b.t
	return t == "uint" or t == "uint8" or t == "uint16" or t == "uint32" or t == "uint64"
end
`},
	{name: "__wisp_isfloat", code: `local function __wisp_isfloat(b)
	return b.t == "float32" or b.t == "float64"
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
