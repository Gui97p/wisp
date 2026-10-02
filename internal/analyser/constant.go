package analyser

import (
	"strconv"

	"github.com/Gui97p/wisp/internal/ast"
)

func intLiteralValue(expr ast.Expression) (neg bool, mag uint64, ok bool) {
	switch e := expr.(type) {
	case *ast.IntLiteral:
		return false, uint64(e.Value), true
	case *ast.UnaryExpr:
		if e.Operator != "-" {
			return false, 0, false
		}
		if lit, isLit := e.Value.(*ast.IntLiteral); isLit {
			return true, uint64(lit.Value), true
		}
	}
	return false, 0, false
}

func integerRange(t Type) (bits int, signed bool, ok bool) {
	if nt, isNamed := t.(NamedType); isNamed {
		t = nt.Underlying
	}
	p, isPrim := t.(PrimitiveType)
	if !isPrim {
		return 0, false, false
	}
	switch p.Name {
	case "int8":
		return 8, true, true
	case "int16":
		return 16, true, true
	case "int32":
		return 32, true, true
	case "int", "int64":
		return 64, true, true
	case "uint8", "char":
		return 8, false, true
	case "uint16":
		return 16, false, true
	case "uint32":
		return 32, false, true
	case "uint", "uint64":
		return 64, false, true
	}
	return 0, false, false
}

func IntegerRange(t Type) (bits int, signed bool, ok bool) {
	return integerRange(t)
}

func fitsInteger(neg bool, mag uint64, bits int, signed bool) bool {
	if signed {
		limit := uint64(1) << (bits - 1)
		if neg {
			return mag <= limit
		}
		return mag < limit
	}
	if neg {
		return mag == 0
	}
	if bits == 64 {
		return true
	}
	return mag < uint64(1)<<bits
}

func (a *Analyser) checkConstFits(expr ast.Expression, target Type) {
	neg, mag, ok := intLiteralValue(expr)
	if !ok {
		return
	}
	bits, signed, ok := integerRange(target)
	if !ok {
		return
	}
	if fitsInteger(neg, mag, bits, signed) {
		return
	}

	text := strconv.FormatUint(mag, 10)
	if neg {
		text = "-" + text
	}
	a.errorf(expr, "constant %s overflows %s", text, target.String())
}

func (a *Analyser) checkBinaryConst(op string, left, right ast.Expression, leftType, rightType Type) {
	if op == "<<" || op == ">>" {
		return
	}
	if _, untyped := leftType.(UntypedIntType); !untyped {
		a.checkConstFits(right, leftType)
	}
	if _, untyped := rightType.(UntypedIntType); !untyped {
		a.checkConstFits(left, rightType)
	}
}

func (a *Analyser) valueAt(values []ast.Expression, n, i int) ast.Expression {
	if len(values) != n || i >= len(values) {
		return nil
	}
	return values[i]
}
