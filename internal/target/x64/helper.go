package x64

import (
	"math"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

func (t *X64Target) framelessExpr(expr ast.Expression, params map[*analyser.Symbol]bool) bool {
	switch e := expr.(type) {
	case *ast.IdentLiteral:
		if sym, ok := t.info.Idents[e]; ok && params[sym] {
			return true
		}
		_, ok := t.info.Folded(e)
		return ok
	case *ast.BoolLiteral, *ast.CharLiteral, *ast.IntLiteral, *ast.NullLiteral:
		return true
	case *ast.BinaryExpr:
		if e.Operator == ">>" || e.Operator == "<<" {
			_, ok := t.info.Folded(e.Right)
			return ok
		}
		if e.Operator != "/" && e.Operator != "%" {
			return t.framelessExpr(e.Left, params) && t.framelessExpr(e.Right, params)
		}
	case *ast.UnaryExpr:
		switch e.Operator {
		case "-", "!", "~":
			return t.framelessExpr(e.Value, params)
		}
	case *ast.CastExpr, *ast.ConvExpr:
		if isInteger(t.info.Types[e]) {
			return true
		}
	}
	return false
}

func (t *X64Target) framelessStmt(stmt ast.Statement, params map[*analyser.Symbol]bool) bool {
	switch s := stmt.(type) {
	case *ast.IfStmt:
		ok := t.framelessExpr(s.Condition, params) && t.framelessStmt(s.Then, params)
		if s.Else != nil {
			ok = ok && t.framelessStmt(s.Else, params)
		}
		return ok
	case *ast.ReturnStmt:
		for _, e := range s.Values {
			if !t.framelessExpr(e, params) {
				return false
			}
		}
		return true
	case *ast.BlockStmt:
		for _, st := range s.Statements {
			if !t.framelessStmt(st, params) {
				return false
			}
		}
		return true
	}
	return false
}

func (t *X64Target) framelessPrefix(stmt *ast.BlockStmt, params map[*analyser.Symbol]bool) (cont int) {
	for _, s := range stmt.Statements {
		if !t.framelessStmt(s, params) {
			return
		}
		cont++
	}
	return
}

func (t *X64Target) remainderIsZero(expr *ast.BinaryExpr) (e ast.Expression, mask int64, ok bool) {
	if expr.Operator != "==" && expr.Operator != "!=" {
		return
	}
	b, fok := expr.Left.(*ast.BinaryExpr)
	if !fok || b.Operator != "%" {
		return
	}
	f, fok := t.info.Folded(b.Right)
	if !fok {
		return
	}
	n := f.Bits()
	if !(n > 0 && (n&(n-1)) == 0) || n > 0x7fffffff {
		return
	}
	fr, fok := t.info.Folded(expr.Right)
	if !fok || fr.Bits() != 0 {
		return
	}
	return b.Left, n - 1, true
}

func (t *X64Target) scaledPlusConst(expr *ast.BinaryExpr) (x ast.Expression, k, disp int64, ok bool) {
	if expr.Operator != "+" && expr.Operator != "-" {
		return
	}
	tp := t.info.Types[expr]
	size := sizeOf(tp)
	if !isInteger(tp) || size < 4 {
		return
	}
	f, fok := t.info.Folded(expr.Right)
	var mul *ast.BinaryExpr
	if fok {
		mul, fok = expr.Left.(*ast.BinaryExpr)
		if !fok {
			return
		}
	} else {
		if expr.Operator == "-" {
			return
		}
		f, fok = t.info.Folded(expr.Left)
		if !fok {
			return
		}
		mul, fok = expr.Right.(*ast.BinaryExpr)
		if !fok {
			return
		}
	}
	if mul.Operator != "*" {
		return
	}
	var tf analyser.ConstValue
	x = mul.Right
	if tf, fok = t.info.Folded(mul.Left); !fok {
		tf, fok = t.info.Folded(mul.Right)
		if !fok {
			return
		}
		x = mul.Left
	}
	k = tf.Bits()
	if !slices.Contains([]int64{2, 3, 4, 5, 8, 9}, k) {
		return
	}
	disp = f.Bits()
	if expr.Operator == "-" {
		disp = -disp
	}
	if disp < math.MinInt32 || disp > math.MaxInt32 {
		return
	}

	ok = true
	return
}
