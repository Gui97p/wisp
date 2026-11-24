package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

type exportedFunc struct {
	decl *ast.FuncDecl
	info *analyser.Info
}

func cTypeName(t analyser.Type) string {
	switch v := t.(type) {
	case analyser.PrimitiveType:
		switch v.Name {
		case "int", "int64":
			return "int64_t"
		case "int8":
			return "int8_t"
		case "int16":
			return "int16_t"
		case "int32":
			return "int32_t"
		case "uint", "uint64":
			return "uint64_t"
		case "uint8":
			return "uint8_t"
		case "uint16":
			return "uint16_t"
		case "uint32":
			return "uint32_t"
		case "float32":
			return "float"
		case "float", "float64":
			return "double"
		case "bool":
			return "bool"
		}
	case analyser.PointerType:
		if prim, ok := v.Element.(analyser.PrimitiveType); ok && prim.Name == "int8" {
			return "char*"
		}
		if _, ok := v.Element.(analyser.VoidType); ok {
			return "void*"
		}
		if prim, ok := v.Element.(analyser.PrimitiveType); ok {
			return cTypeName(prim) + "*"
		}
		return "void*"
	}
	return "void*"
}

func cHeader(name string, funcs []exportedFunc) string {
	guard := "WISP_" + strings.ToUpper(sanitizeIdent(name)) + "_H"
	var b strings.Builder
	fmt.Fprintf(&b, "#ifndef %s\n#define %s\n\n#include <stdbool.h>\n#include <stdint.h>\n\n#ifdef __cplusplus\nextern \"C\" {\n#endif\n\n", guard, guard)
	for _, f := range funcs {
		ret := "void"
		if returns := f.info.FuncReturns[f.decl]; len(returns) > 0 {
			ret = cTypeName(returns[0])
		}
		params := make([]string, 0, len(f.decl.Params))
		for i, t := range f.info.VarTypes[f.decl] {
			text := cTypeName(t)
			if i < len(f.decl.Params) && f.decl.Params[i].Name != "" {
				text += " " + f.decl.Params[i].Name
			}
			params = append(params, text)
		}
		if len(params) == 0 {
			params = []string{"void"}
		}
		fmt.Fprintf(&b, "%s %s(%s);\n", ret, f.decl.CExport, strings.Join(params, ", "))
	}
	b.WriteString("\n#ifdef __cplusplus\n}\n#endif\n\n#endif\n")
	return b.String()
}

func sanitizeIdent(name string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return '_'
	}, name)
}

func writeCHeader(path, name string, funcs []exportedFunc) error {
	return os.WriteFile(path, []byte(cHeader(name, funcs)), 0644)
}
