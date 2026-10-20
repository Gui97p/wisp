package analyser

import (
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

var BoxCodes = []string{
	"int", "int8", "int16", "int32", "int64",
	"uint", "uint8", "uint16", "uint32", "uint64",
	"float32", "float64",
	"bool", "char", "string", "pointer",
}

func BoxCode(name string) int {
	for i, code := range BoxCodes {
		if code == name {
			return i + 1
		}
	}
	return 0
}

type BoxLayout struct {
	Code     string
	Identity string
}

func boxPrimitiveName(name string) string {
	if name == "float" {
		return "float64"
	}
	return name
}

func BoxKey(t Type) string {
	switch v := t.(type) {
	case PrimitiveType:
		return boxPrimitiveName(v.Name)
	case NamedType:
		return qualifiedName(v.Module, v.Name)
	case *StructType:
		return qualifiedName(v.Module, v.Name)
	case PointerType:
		return "*" + BoxKey(v.Element)
	case VoidType:
		return "void"
	}
	return t.String()
}

func qualifiedName(module, name string) string {
	if module == "" {
		return name
	}
	return strings.ReplaceAll(module, "/", ".") + "." + name
}

func Layout(t Type) BoxLayout {
	switch v := t.(type) {
	case PrimitiveType:
		return BoxLayout{Code: boxPrimitiveName(v.Name)}
	case NamedType:
		if v.Enum {
			return BoxLayout{Code: "int", Identity: BoxKey(v)}
		}
		return BoxLayout{Identity: BoxKey(v)}
	case PointerType:
		pointee := "void"
		switch elem := v.Element.(type) {
		case PrimitiveType, NamedType, *StructType:
			pointee = BoxKey(elem)
		case PointerType:
			pointee = "pointer"
		}
		return BoxLayout{Code: "pointer", Identity: pointee}
	}
	return BoxLayout{}
}

func (l BoxLayout) Named() bool {
	return l.Code == "" && l.Identity != ""
}

func boxable(t Type) (bool, string) {
	switch v := t.(type) {
	case PrimitiveType:
		return true, ""
	case UntypedIntType, UntypedFloatType:
		return true, ""
	case NamedType:
		return boxable(v.Underlying)
	case PointerType:
		return true, ""
	case *StructType:
		return false, "pass a pointer to it instead"
	case ArrayType, SpanType:
		return false, "pass its address or a string instead"
	case BoxedType:
		return false, "an any value cannot be passed on, only the variadic itself with ..."
	}
	return false, ""
}

func (ft *FuncType) acceptsBoxes() bool {
	if !ft.Variadic || len(ft.Params) == 0 {
		return false
	}
	_, boxed := ft.Params[len(ft.Params)-1].(BoxedType)
	return boxed
}

func (a *Analyser) checkBoxedArgs(expr *ast.CallExpr, argTypes []Type, fixed int) {
	if len(argTypes) != len(expr.Args) {
		a.error(expr, "a call with several results cannot be passed to an untyped variadic")
		return
	}

	boxedEnd := len(argTypes)
	if expr.Spread {
		boxedEnd--
		span, isSpan := argTypes[boxedEnd].(SpanType)
		if boxedEnd < fixed || !isSpan {
			a.error(expr, "... passes on the untyped variadic of the calling function, and has to be the last argument")
			return
		}
		if _, boxed := span.Element.(BoxedType); !boxed {
			a.errorf(expr, "... passes on an untyped variadic, got %s", span.String())
			return
		}
	}

	for i := fixed; i < boxedEnd; i++ {
		arg := expr.Args[i]
		t := argTypes[i]
		if _, invalid := t.(InvalidType); invalid {
			continue
		}

		switch t.(type) {
		case UntypedIntType:
			t = PrimitiveType{Name: "int"}
			a.fixUntyped(arg, t)
		case UntypedFloatType:
			t = PrimitiveType{Name: "float64"}
			a.fixUntyped(arg, t)
		}

		ok, hint := boxable(t)
		if !ok {
			d := a.errorf(arg, "cannot pass %s as an untyped variadic argument", t.String())
			if hint != "" {
				d.WithHelp("%s", hint)
			}
			continue
		}
		a.info.Boxes[arg] = t
	}
}

func (a *Analyser) checkTypeCase(expr *ast.BinaryExpr) Type {
	if expr.Operator != "==" {
		a.errorf(expr, "operator %s does not apply to an any value", expr.Operator).
			WithHelp("inspect it with a switch on types, as in `switch let v = a { case int: ... }`")
		return InvalidType{}
	}

	t, ok := a.labelType(expr.Right)
	if !ok {
		return InvalidType{}
	}

	if okType, hint := boxable(t); !okType {
		d := a.errorf(expr.Right, "%s cannot be held by an untyped variadic", t.String())
		if hint != "" {
			d.WithHelp("%s", hint)
		}
		return InvalidType{}
	}

	group := false
	if _, isLabel := expr.Right.(*ast.TypeLabel); isLabel {
		if p, isPrimitive := t.(PrimitiveType); isPrimitive {
			group = p.Name == "int" || p.Name == "uint" || p.Name == "float"
		}
	}

	a.info.TypeCases[expr] = &TypeCase{Type: t, Group: group}
	return PrimitiveType{Name: "bool"}
}

func (a *Analyser) labelType(label ast.Expression) (Type, bool) {
	switch l := label.(type) {
	case *ast.TypeLabel:
		t := a.resolveTypeRef(l, a.scope, l.Type)
		if _, invalid := t.(InvalidType); invalid || t == nil {
			return nil, false
		}
		return t, true
	case *ast.IdentLiteral:
		sym, found := a.scope.Resolve(l.Value)
		if !found || !isTypeSymbol(sym) {
			a.errorf(l, "%s is not a type", l.Value).WithHelp("a case of a switch on an any value names a type")
			return nil, false
		}
		sym.Used = true
		return sym.Type, true
	case *ast.MemberExpr:
		if module, isIdent := l.Object.(*ast.IdentLiteral); isIdent {
			if sym, found := a.lookupType(a.scope, module.Value, l.Field); found && isTypeSymbol(sym) {
				return sym.Type, true
			}
		}
	}

	a.error(label, "an any value is only compared with a type, in a switch").
		WithHelp("write `switch let v = a { case int: ... }`")
	return nil, false
}

func isTypeSymbol(sym *Symbol) bool {
	return sym.Kind == TYPE || sym.Kind == STRUCT
}

func narrowedType(tc *TypeCase) Type {
	if tc.Group {
		if p, ok := tc.Type.(PrimitiveType); ok && p.Name == "float" {
			return PrimitiveType{Name: "float64"}
		}
	}
	return tc.Type
}

func (a *Analyser) narrowSwitchVariable(condition ast.Expression) {
	match, ok := condition.(*ast.BinaryExpr)
	if !ok {
		return
	}
	tc := a.info.TypeCases[match]
	if tc == nil {
		return
	}
	id, ok := match.Left.(*ast.IdentLiteral)
	if !ok {
		return
	}
	original := a.info.Idents[id]
	if original == nil {
		return
	}

	narrowed := &Symbol{Name: original.Name, Kind: VAR, Type: narrowedType(tc), File: original.File, Used: true}
	a.scope.Define(narrowed)
	a.info.Narrowed[narrowed] = original
}

func (a *Analyser) assignsSwitchVariable(target ast.Expression) bool {
	id, ok := target.(*ast.IdentLiteral)
	if !ok {
		return false
	}
	sym, found := a.scope.Resolve(id.Value)
	if !found {
		return false
	}
	if _, narrowed := a.info.Narrowed[sym]; !narrowed {
		return false
	}
	a.errorf(target, "cannot assign to %s, the variable of a type switch", id.Value)
	return true
}

func DescriptorOf(t Type) (module, key string, ok bool) {
	switch v := t.(type) {
	case NamedType:
		return v.Module, BoxKey(v), true
	case *StructType:
		return v.Module, BoxKey(v), true
	case PointerType:
		switch elem := v.Element.(type) {
		case NamedType:
			return elem.Module, BoxKey(elem), true
		case *StructType:
			return elem.Module, BoxKey(elem), true
		}
	}
	return "", "", false
}

func TypeKey(module, name string) string {
	return qualifiedName(module, name)
}
