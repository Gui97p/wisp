package lua

var nativeFuncs = map[string]map[string]string{
	"std/math": {
		"abs":   "function(x) return x < 0 and -x or x end",
		"fabs":  "math.abs",
		"min":   "math.min",
		"fmin":  "math.min",
		"max":   "math.max",
		"fmax":  "math.max",
		"floor": "math.floor",
		"ceil":  "math.ceil",
		"round": "function(x) return math.floor(x + 0.5) end",
		"sqrt":  "math.sqrt",
		"pow":   "function(base, exp) return base ^ exp end",
		"sin":   "math.sin",
		"cos":   "math.cos",
		"tan":   "math.tan",
		"log":   "math.log",
		"exp":   "math.exp",
	},
}

func (t *LuaTarget) nativeAlias(name string) string {
	return nativeFuncs[t.modPath][name]
}
