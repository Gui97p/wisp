package ast

import (
	"fmt"
	"strings"
	"unicode"
)

func Describe(node Node) string {
	name := fmt.Sprintf("%T", node)
	name = strings.TrimPrefix(name, "*ast.")

	for _, suffix := range []string{"Literal", "Expr", "Stmt", "Decl"} {
		if base, ok := strings.CutSuffix(name, suffix); ok && base != "" {
			name = base + map[string]string{"Literal": " literal", "Expr": " expression", "Stmt": " statement", "Decl": " declaration"}[suffix]
			break
		}
	}

	var b strings.Builder
	for i, r := range name {
		if unicode.IsUpper(r) && i > 0 && name[i-1] != ' ' {
			b.WriteByte(' ')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
