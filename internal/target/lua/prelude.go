package lua

const runtimePrelude = `local function __wisp_len(t)
	local n = 0
	while t[n] ~= nil do n = n + 1 end
	return n
end

local function __wisp_slice(t, a, b)
	local r = {}
	local j = 0
	for i = a, b - 1 do
		r[j] = t[i]
		j = j + 1
	end
	return r
end

local function __wisp_idiv(a, b)
	local q = a // b
	if q < 0 and q * b ~= a then q = q + 1 end
	return q
end

local function __wisp_imod(a, b)
	return math.fmod(a, b)
end

local function __wisp_pack(...)
	local r = {}
	for i = 1, select("#", ...) do
		r[i - 1] = (select(i, ...))
	end
	return r
end

local function __wisp_contains(t, v)
	local n = __wisp_len(t)
	for i = 0, n - 1 do
		if t[i] == v then return true end
	end
	return false
end

`
