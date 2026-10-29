package lua

import (
	"fmt"
	"strings"
)

var luaKeywords = map[string]bool{
	"and": true, "break": true, "do": true, "else": true, "elseif": true, "end": true,
	"false": true, "for": true, "function": true, "goto": true, "if": true, "in": true,
	"local": true, "nil": true, "not": true, "or": true, "repeat": true, "return": true,
	"then": true, "true": true, "until": true, "while": true,
}

func luaIdent(name string) string {
	if luaKeywords[name] {
		return "__wisp_kw_" + name
	}
	return name
}

func fieldAccess(field string) string {
	if luaKeywords[field] {
		return fmt.Sprintf("[%q]", field)
	}
	return "." + field
}

func fieldKey(field string) string {
	if luaKeywords[field] {
		return fmt.Sprintf("[%q]=", field)
	}
	return field + "="
}

func luaIdents(names []string) string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = luaIdent(n)
	}
	return strings.Join(out, ", ")
}

const luaGlobals = "local __lua_math, __lua_string, __lua_table, __lua_io, __lua_os, __lua_package = math, string, table, io, os, package\n"
