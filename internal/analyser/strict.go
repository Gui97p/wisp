package analyser

import (
	"fmt"
	"math/big"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func (a *Analyser) funcCache() map[string]*Symbol {
	if a.phantomBuilding {
		return a.checkFuncInstances
	}
	return a.funcInstances
}

func (a *Analyser) methodCache() map[string]*FuncType {
	if a.phantomBuilding {
		return a.checkMethodInstances
	}
	return a.methodInstances
}

type representative struct {
	value TypeValue
	note  string
}

type combination struct {
	args []TypeValue
	note string
}

func (a *Analyser) instantiatePhantoms() {
	a.phantomBuilding = true
	defer func() { a.phantomBuilding, a.phantomNote = false, "" }()

	declarations := append([]ast.Declaration(nil), a.program.Declarations...)
	for _, d := range declarations {
		a.currentFile = a.declFiles[d]
		switch decl := d.(type) {
		case *ast.FuncDecl:
			if len(decl.TypeParams) == 0 {
				continue
			}
			if decl.Receiver != nil {
				if sym := a.genericMethodSyms[decl]; sym != nil && sym.Generic != nil {
					for _, combo := range a.combinations(sym.Generic) {
						a.phantomNote = combo.note
						a.instantiateFunc(decl, sym.Generic, combo.args)
					}
				}
				continue
			}
			if g := a.genericOf(decl.Name, decl); g != nil {
				for _, combo := range a.combinations(g) {
					a.phantomNote = combo.note
					a.instantiateFunc(decl, g, combo.args)
				}
			}
		case *ast.StructDecl:
			if len(decl.TypeParams) > 0 {
				a.phantomType(decl, decl.Name)
			}
		case *ast.TypeDecl:
			switch {
			case len(decl.TypeParams) == 0:
			case decl.Data:
				a.phantomType(decl, decl.Name)
			default:
				if g := a.genericOf(decl.Name, decl); g != nil {
					for _, combo := range a.combinations(g) {
						a.phantomNote = combo.note
						a.instantiateAlias(decl, g, combo.args)
					}
				}
			}
		case *ast.InterfaceDecl:
			if len(decl.TypeParams) == 0 {
				continue
			}
			if g := a.genericOf(decl.Name, decl); g != nil {
				for _, combo := range a.combinations(g) {
					if it, ok := a.instantiateInterface(decl, g, combo.args).(*InterfaceType); ok {
						it.ensure()
					}
				}
			}
		}
	}
}

func (a *Analyser) phantomType(decl ast.Node, name string) {
	g := a.genericOf(name, decl)
	if g == nil {
		return
	}
	for _, combo := range a.combinations(g) {
		a.phantomNote = combo.note
		var created Type
		if g.Sum != nil {
			created = a.instantiateSum(decl, g, combo.args)
		} else {
			created = a.instantiateStruct(decl, g, combo.args)
		}
		st, isStruct := created.(*StructType)
		if !isStruct {
			continue
		}
		methods := MethodsOf(st)
		names := make([]string, 0, len(methods))
		for name := range methods {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			a.instanceOfMethod(decl, methods[name])
		}
	}
}

func (a *Analyser) combinations(g *GenericDecl) []combination {
	reps, ok := a.representatives(g)
	if !ok {
		return nil
	}

	total := 1
	for _, r := range reps {
		total *= len(r)
		if total > 16 {
			break
		}
	}

	build := func(pick []int) combination {
		args := make([]TypeValue, len(reps))
		var notes []string
		for i, r := range reps {
			args[i] = r[pick[i]].value
			if r[pick[i]].note != "" {
				notes = append(notes, r[pick[i]].note)
			}
		}
		return combination{args: args, note: strings.Join(notes, ", ")}
	}

	var result []combination
	if total <= 16 {
		pick := make([]int, len(reps))
		for {
			result = append(result, build(pick))
			i := len(reps) - 1
			for i >= 0 {
				pick[i]++
				if pick[i] < len(reps[i]) {
					break
				}
				pick[i] = 0
				i--
			}
			if i < 0 {
				return result
			}
		}
	}

	result = append(result, build(make([]int, len(reps))))
	for i, r := range reps {
		for k := 1; k < len(r); k++ {
			pick := make([]int, len(reps))
			pick[i] = k
			result = append(result, build(pick))
		}
	}
	return result
}

func (a *Analyser) representatives(g *GenericDecl) ([][]representative, bool) {
	names := paramNames(g.Params)
	current := make([]TypeValue, len(g.Params))
	reps := make([][]representative, len(g.Params))

	for i, param := range g.Params {
		if param.Const {
			value, ok := a.constRepresentative(g, param)
			if !ok {
				return nil, false
			}
			current[i] = value
			reps[i] = []representative{{value: value}}
			continue
		}
		current[i] = TypeValue{Type: a.opaqueFor(g, param, nil)}
	}

	for i, param := range g.Params {
		if param.Const {
			continue
		}
		if param.Constraint == nil {
			reps[i] = []representative{{value: current[i]}}
			continue
		}

		scope := a.bindScope(g.Scope, names, g.Params, current, 0)
		savedFile := a.currentFile
		a.currentFile = g.File
		expr := a.resolveConstraint(g.node(), param.Constraint, scope, false)
		a.currentFile = savedFile
		if expr == nil {
			return nil, false
		}

		if members, known := a.constraintMembers(expr); known && len(members) > 0 {
			if len(members) > 8 {
				members = members[:8]
			}
			for _, m := range members {
				note := ""
				if len(members) > 1 {
					note = fmt.Sprintf("%s = %s, one of the types of %s", param.Name, m, expr)
				}
				reps[i] = append(reps[i], representative{value: TypeValue{Type: m}, note: note})
			}
		} else {
			reps[i] = []representative{{value: TypeValue{Type: a.opaqueFor(g, param, expr)}}}
		}
		current[i] = reps[i][0].value
	}
	return reps, true
}

func (a *Analyser) constRepresentative(g *GenericDecl, param ast.TypeParam) (TypeValue, bool) {
	paramType := a.resolveTypeRef(g.node(), g.Scope, param.Type)
	if paramType == nil {
		return TypeValue{}, false
	}
	value := ConstValue{Kind: ConstInt, Int: big.NewInt(1), Type: paramType}
	if prim, isPrim := paramType.(PrimitiveType); isPrim && prim.Name == "bool" {
		value = ConstValue{Kind: ConstBool, Bool: true, Type: paramType}
	}
	return TypeValue{Type: paramType, Value: &value, Label: param.Name}, true
}

func (a *Analyser) constraintMembers(e *ConstraintExpr) ([]Type, bool) {
	switch e.Kind {
	case ast.ConstraintOr:
		left, lok := a.constraintMembers(e.Left)
		right, rok := a.constraintMembers(e.Right)
		if !lok || !rok {
			return nil, false
		}
		return unionTypes(left, right), true
	case ast.ConstraintAnd:
		if e.Right.Kind == ast.ConstraintNot {
			return a.excluding(e.Left, e.Right.Left)
		}
		if e.Left.Kind == ast.ConstraintNot {
			return a.excluding(e.Right, e.Left.Left)
		}
		left, lok := a.constraintMembers(e.Left)
		right, rok := a.constraintMembers(e.Right)
		switch {
		case lok && rok:
			return intersectTypes(left, right), true
		case lok && isComparableAtom(e.Right):
			return filterTypes(left, comparableType), true
		case rok && isComparableAtom(e.Left):
			return filterTypes(right, comparableType), true
		}
		return nil, false
	case ast.ConstraintNot:
		return nil, false
	}

	if e.Iface != nil {
		e.Iface.ensure()
		if e.Iface.builtin != "" || len(e.Iface.Methods) > 0 || e.Iface.Expr == nil {
			return nil, false
		}
		return a.constraintMembers(e.Iface.Expr)
	}
	return []Type{e.Type}, true
}

func (a *Analyser) excluding(from, excluded *ConstraintExpr) ([]Type, bool) {
	members, ok := a.constraintMembers(from)
	if !ok {
		return nil, false
	}
	out, known := a.constraintMembers(excluded)
	if !known {
		return members, true
	}
	return filterTypes(members, func(t Type) bool {
		for _, o := range out {
			if sameBase(t, o) {
				return false
			}
		}
		return true
	}), true
}

func isComparableAtom(e *ConstraintExpr) bool {
	return e.Kind == ast.ConstraintType && e.Iface != nil && e.Iface.builtin == "comparable"
}

func unionTypes(left, right []Type) []Type {
	out := append([]Type(nil), left...)
	for _, t := range right {
		found := false
		for _, o := range out {
			if sameBase(t, o) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, t)
		}
	}
	return out
}

func intersectTypes(left, right []Type) []Type {
	return filterTypes(left, func(t Type) bool {
		for _, o := range right {
			if sameBase(t, o) {
				return true
			}
		}
		return false
	})
}

func filterTypes(types []Type, keep func(Type) bool) []Type {
	var out []Type
	for _, t := range types {
		if keep(t) {
			out = append(out, t)
		}
	}
	return out
}

func (a *Analyser) opaqueFor(g *GenericDecl, param ast.TypeParam, expr *ConstraintExpr) Type {
	key := "opaque:" + qualifiedName(g.Module, g.Name) + "." + param.Name
	impl := &ImplSet{}
	methods := map[string]*FuncType{}
	comparable := false
	orderable := false

	if expr != nil {
		walkConstraintAtoms(expr, func(atom *ConstraintExpr) {
			if atom.Iface == nil {
				return
			}
			if atom.Iface.builtin == "comparable" {
				comparable = true
				return
			}
			if atom.Iface.builtin == "orderable" {
				orderable = true
				return
			}
			impl.add(atom.Iface)
		})
	}

	var result Type
	if orderable {
		nt := NamedType{Name: param.Name, Underlying: PrimitiveType{Name: "int"}, Methods: methods, Module: g.Module, Dir: filepath.Dir(g.File), Exported: true, Impl: impl}
		impl.typ = nt
		result = nt
	} else if comparable {
		nt := NamedType{Name: param.Name, Underlying: PrimitiveType{Name: "int"}, Enum: true, Methods: methods, Module: g.Module, Dir: filepath.Dir(g.File), Exported: true, Members: map[string]int64{}, MemberDocs: map[string]string{}, Impl: impl}
		impl.typ = nt
		result = nt
	} else {
		st := &StructType{
			Name:      param.Name,
			Fields:    map[string]Type{},
			Methods:   methods,
			FieldDocs: map[string]string{},
			FieldDirs: map[string]string{},
			Module:    g.Module,
			Dir:       filepath.Dir(g.File),
			Exported:  true,
			key:       key,
			Impl:      impl,
		}
		st.complete = true
		impl.typ = st
		result = st
	}

	if expr != nil {
		for _, m := range expr.requiredMethods() {
			methods[m.Name] = &FuncType{
				Name: m.Name, Params: m.Params, ParamNames: m.ParamNames, Returns: m.Returns, Fallible: m.Fallible,
				Receiver: PointerType{Element: result}, Exported: true, Module: g.Module,
			}
		}
	}
	return result
}

func walkConstraintAtoms(e *ConstraintExpr, fn func(*ConstraintExpr)) {
	switch e.Kind {
	case ast.ConstraintType:
		fn(e)
	case ast.ConstraintAnd:
		walkConstraintAtoms(e.Left, fn)
		walkConstraintAtoms(e.Right, fn)
	}
}
