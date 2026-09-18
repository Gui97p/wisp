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

`
