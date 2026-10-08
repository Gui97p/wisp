package analyser

import (
	"sort"

	"github.com/Gui97p/wisp/internal/ast"
)

func (a *Analyser) enumTypeExpr(expr ast.Expression) (NamedType, bool) {
	switch e := expr.(type) {
	case *ast.IdentLiteral:
		sym, ok := a.scope.Resolve(e.Value)
		if !ok || sym.Kind != TYPE {
			return NamedType{}, false
		}
		enum, ok := sym.Type.(NamedType)
		return enum, ok && enum.Enum
	case *ast.MemberExpr:
		module, ok := e.Object.(*ast.IdentLiteral)
		if !ok {
			return NamedType{}, false
		}
		sym, ok := a.lookupType(a.scope, module.Value, e.Field)
		if !ok || sym.Kind != TYPE {
			return NamedType{}, false
		}
		enum, ok := sym.Type.(NamedType)
		return enum, ok && enum.Enum
	}
	return NamedType{}, false
}

func (a *Analyser) checkEnumMember(expr *ast.MemberExpr, enum NamedType) Type {
	value, ok := enum.Members[expr.Field]
	if !ok {
		a.errorf(expr, "enum %s has no member %s", enum.Name, expr.Field)
		return InvalidType{}
	}
	a.info.Members[expr] = &MemberInfo{Kind: MemberEnum}
	a.info.EnumValues[expr] = EnumValue{Type: enum, Name: expr.Field, Value: value}
	return enum
}

func (a *Analyser) checkDotIdent(expr *ast.DotIdent) Type {
	a.pendingDots[expr] = true
	return ImplicitEnumType{Name: expr.Name}
}

func (a *Analyser) resolveDot(expr ast.Expression, to Type) bool {
	node, ok := expr.(*ast.DotIdent)
	if !ok {
		return false
	}
	if _, stillImplicit := to.(ImplicitEnumType); stillImplicit {
		return true
	}

	delete(a.pendingDots, node)

	enum, isNamed := to.(NamedType)
	if !isNamed || !enum.Enum {
		a.errorf(node, "cannot infer the enum of .%s here: %s is not an enum", node.Name, to.String())
		return true
	}
	value, found := enum.Members[node.Name]
	if !found {
		a.errorf(node, "enum %s has no member %s", enum.Name, node.Name)
		return true
	}

	a.info.EnumValues[node] = EnumValue{Type: enum, Name: node.Name, Value: value}
	a.info.Types[node] = enum
	return true
}

func (a *Analyser) dropDot(expr ast.Expression) {
	if node, ok := expr.(*ast.DotIdent); ok {
		delete(a.pendingDots, node)
	}
}

func (a *Analyser) reportUnresolvedDots() {
	nodes := make([]*ast.DotIdent, 0, len(a.pendingDots))
	for node := range a.pendingDots {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		li, ci := nodes[i].Position()
		lj, cj := nodes[j].Position()
		if li != lj {
			return li < lj
		}
		return ci < cj
	})
	for _, node := range nodes {
		a.errorf(node, "cannot infer the enum of .%s here: write Enum.%s", node.Name, node.Name)
	}
}

func isImplicitEnum(t Type) bool {
	_, ok := t.(ImplicitEnumType)
	return ok
}

func isEnumType(t Type) bool {
	nt, ok := t.(NamedType)
	return ok && nt.Enum
}
