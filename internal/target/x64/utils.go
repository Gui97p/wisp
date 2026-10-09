package x64

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func isSigned(tp analyser.Type) bool {
	switch t := underlying(tp).(type) {
	case analyser.PrimitiveType:
		switch t.Name {
		case "int", "int8", "int16", "int32", "int64", "float", "float32", "float64":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func regsNeeded(expr ast.Expression) int {
	switch e := expr.(type) {
	case *ast.ArrayLiteral:
		needed := 0
		for _, el := range e.Elements {
			needed = max(regsNeeded(el), needed)
		}
		return needed
	case *ast.StructLiteral:
		needed := 0
		for _, v := range e.Values {
			needed = max(regsNeeded(v), needed)
		}
		return needed
	case *ast.BinaryExpr:
		return max(regsNeeded(e.Left), 1+regsNeeded(e.Right), 1)
	case *ast.UnaryExpr:
		if e.Operator == "*" {
			return max(regsNeeded(e.Value), 2)
		}
		return max(regsNeeded(e.Value), 1)
	case *ast.CastExpr:
		return max(regsNeeded(e.Value), 2)
	case *ast.CallExpr:
		maxRegs := 1
		for _, arg := range e.Args {
			maxRegs = max(regsNeeded(arg), maxRegs)
		}
		return maxRegs
	case *ast.MemberExpr:
		return max(regsNeeded(e.Object), 2)
	case *ast.IndexExpr:
		return max(regsNeeded(e.Array), 1+regsNeeded(e.Index), 2)
	case *ast.SliceExpr:
		return max(regsNeeded(e.Array), 1+regsNeeded(e.Start), 2+regsNeeded(e.End), 3)
	case *ast.TernaryExpr:
		return max(regsNeeded(e.Condition), regsNeeded(e.Then), regsNeeded(e.Else))
	case *ast.InExpr:
		return max(regsNeeded(e.Left), 1+regsNeeded(e.Right), 1)
	}
	return 0
}

func decodeEscapes(s string) ([]byte, error) {
	out := make([]byte, 0, len(s))

	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			out = append(out, s[i])
			continue
		}

		i++
		if i >= len(s) {
			return nil, fmt.Errorf("unterminated escape sequence")
		}

		switch s[i] {
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'b':
			out = append(out, '\b')
		case 'f':
			out = append(out, '\f')
		case 'v':
			out = append(out, '\v')
		case 'a':
			out = append(out, '\a')
		case '\\':
			out = append(out, '\\')
		case '"':
			out = append(out, '"')
		case '\'':
			out = append(out, '\'')
		case '0':
			out = append(out, 0)

		case 'x':
			if i+2 >= len(s) {
				return nil, fmt.Errorf("invalid hexadecimal escape")
			}

			hi, ok1 := hexValue(s[i+1])
			lo, ok2 := hexValue(s[i+2])
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("invalid hexadecimal escape")
			}

			out = append(out, hi<<4|lo)
			i += 2

		default:
			return nil, fmt.Errorf("unknown escape sequence: \\%c", s[i])
		}
	}

	return out, nil
}

func hexValue(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}

func bytesToAsm(b []byte) string {
	var sb strings.Builder

	for i, v := range b {
		if i > 0 {
			sb.WriteString(", ")
		}

		sb.WriteString(strconv.Itoa(int(v)))
	}

	return sb.String()
}

func underlying(tp analyser.Type) analyser.Type {
	switch t := tp.(type) {
	case analyser.NamedType:
		return underlying(t.Underlying)
	default:
		return t
	}
}
