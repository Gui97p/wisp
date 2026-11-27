package analyser

import (
	"sort"

	"github.com/Gui97p/wisp/internal/ast"
)

func isInferredSize(expr ast.Expression) bool {
	id, ok := expr.(*ast.IdentLiteral)
	return ok && id.Value == "_"
}

func (a *Analyser) checkAnonLiteral(expr *ast.AnonLiteral) Type {
	a.pendingAnon[expr] = true
	return AnonType{}
}

func anonTarget(to Type) Type {
	if st, ok := isOptionalStruct(to); ok {
		return st.Optional
	}
	if element, ok := optionalElement(to); ok {
		return element
	}
	return to
}

func (a *Analyser) resolveAnonTernary(ternary *ast.TernaryExpr, to Type) (Type, bool) {
	arm := func(slot *ast.Expression) bool {
		switch (*slot).(type) {
		case *ast.AnonLiteral, *ast.TernaryExpr:
			built, ok := a.resolveAnon(slot, to)
			if !ok {
				return false
			}
			if built.Equals(to) {
				return true
			}
			return a.coerce(slot, built, to)
		}
		from, known := a.info.Types[*slot]
		if !known {
			return false
		}
		if _, isNull := from.(NullType); isNull {
			return a.coerce(slot, from, to)
		}
		if from.Equals(to) {
			return true
		}
		return a.coerce(slot, from, to)
	}
	thenOk := arm(&ternary.Then)
	elseOk := arm(&ternary.Else)
	if !thenOk || !elseOk {
		return nil, false
	}
	a.info.Types[ternary] = to
	return to, true
}

func buildsAnon(t Type) bool {
	switch underlying(anonTarget(t)).(type) {
	case *StructType, ArrayType, SpanType:
		return true
	}
	return false
}

func isPendingAnon(t Type) bool {
	_, ok := t.(AnonType)
	return ok
}

func (a *Analyser) resolveAnon(slot *ast.Expression, to Type) (Type, bool) {
	if ternary, isTernary := (*slot).(*ast.TernaryExpr); isTernary {
		return a.resolveAnonTernary(ternary, to)
	}
	lit, ok := (*slot).(*ast.AnonLiteral)
	if !ok {
		return nil, false
	}
	delete(a.pendingAnon, lit)

	target := anonTarget(to)
	switch t := underlying(target).(type) {
	case *StructType:
		if t.Sum != nil {
			a.errorf(lit, "%s has variants: write .Name{...} with the variant you want", t.Name)
			return nil, false
		}
		if len(lit.Elements) > 0 {
			a.errorf(lit, "%s is a struct: write name: value pairs", t.Name)
			return nil, false
		}
		node := &ast.StructLiteral{Name: t.Name, Keys: lit.Keys, KeyAt: lit.KeyAt, Values: lit.Values}
		node.NodePos = lit.NodePos
		a.checkFieldInits(node, t)
		a.info.Types[node] = target
		*slot = node
		return target, true
	case ArrayType:
		if len(lit.Keys) > 0 {
			a.errorf(lit, "an array takes plain values, not name: value pairs")
			return nil, false
		}
		node := &ast.ArrayLiteral{Elements: lit.Elements}
		node.NodePos = lit.NodePos
		result, ok := a.fillArray(node, t)
		if !ok {
			return nil, false
		}
		a.info.Types[node] = result
		*slot = node
		return result, true
	case SpanType:
		if len(lit.Keys) > 0 {
			a.errorf(lit, "a slice takes plain values, not name: value pairs")
			return nil, false
		}
		node := &ast.ArrayLiteral{Elements: lit.Elements}
		node.NodePos = lit.NodePos
		if len(lit.Elements) == 0 {
			a.info.Types[node] = target
			*slot = node
			return target, true
		}
		result, ok := a.fillArray(node, ArrayType{Element: t.Element, Size: -1})
		if !ok {
			return nil, false
		}
		a.info.Types[node] = result
		*slot = node
		a.insertSlice(slot, t)
		return t, true
	}

	a.errorf(lit, "cannot build %s from .{...}", target.String())
	return nil, false
}

func (a *Analyser) fillArray(node *ast.ArrayLiteral, at ArrayType) (ArrayType, bool) {
	count := int64(len(node.Elements))
	if at.Size < 0 {
		at.Size = count
	}
	if count > at.Size {
		a.errorf(node, "an array of %d takes at most %d values, got %d", at.Size, at.Size, count)
		return at, false
	}
	for i := range node.Elements {
		elementType := a.checkExpr(node.Elements[i])
		if _, invalid := elementType.(InvalidType); invalid {
			continue
		}
		if !a.coerce(&node.Elements[i], elementType, at.Element) {
			a.helpMismatch(a.errorf(node.Elements[i], "array element %d expects %s, got %s", i+1, at.Element.String(), elementType.String()), elementType, at.Element)
			continue
		}
		a.checkConstFits(node.Elements[i], at.Element)
	}
	if count < at.Size {
		a.requireZeroValue(node, "the elements left out", at.Element)
	}
	return at, true
}

func (a *Analyser) checkArrayLiteral(expr *ast.ArrayLiteral) Type {
	saved := a.inferSize
	a.inferSize = true
	declared := a.resolveTypeRef(expr, a.scope, *expr.Type)
	a.inferSize = saved
	if declared == nil {
		return InvalidType{}
	}

	switch t := declared.(type) {
	case ArrayType:
		result, ok := a.fillArray(expr, t)
		if !ok {
			return InvalidType{}
		}
		return result
	case SpanType:
		if len(expr.Elements) > 0 {
			a.errorf(expr, "a slice literal has no elements: write [%d]%s{...} for an array, which becomes a slice where one is expected", len(expr.Elements), t.Element.String())
			return InvalidType{}
		}
		return t
	case InvalidType:
		return t
	}
	a.errorf(expr, "%s is not an array or slice type", declared.String())
	return InvalidType{}
}

func (a *Analyser) reportUnresolvedAnon() {
	nodes := make([]*ast.AnonLiteral, 0, len(a.pendingAnon))
	for node := range a.pendingAnon {
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
		a.errorf(node, "cannot infer the type of .{...} here").
			WithHelp("it takes the type of what receives it: write it where a type is expected, or name the type, as in Point{x: 1} or [3]int{1, 2, 3}")
	}
}

func (a *Analyser) inferredSize(declared, value Type) (Type, bool) {
	at, ok := declared.(ArrayType)
	if !ok || at.Size >= 0 {
		return declared, true
	}
	va, ok := value.(ArrayType)
	if !ok {
		return declared, false
	}
	at.Size = va.Size
	return at, true
}
