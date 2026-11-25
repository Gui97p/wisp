package analyser

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

type MethodSig struct {
	Name       string
	Params     []Type
	ParamNames []string
	Returns    []Type
	Fallible   bool
	Doc        string
	File       string
	Line       int
	Col        int
}

func (m MethodSig) String() string {
	params := make([]string, len(m.Params))
	for i, p := range m.Params {
		params[i] = p.String()
	}
	text := m.Name + "(" + strings.Join(params, ", ") + ")"
	results := make([]string, len(m.Returns))
	for i, r := range m.Returns {
		results[i] = r.String()
	}
	switch {
	case m.Fallible && len(results) == 0:
		text += " !"
	case m.Fallible && len(results) == 1:
		text += " !" + results[0]
	case m.Fallible:
		text += " !(" + strings.Join(results, ", ") + ")"
	case len(results) == 1:
		text += " " + results[0]
	case len(results) > 1:
		text += " (" + strings.Join(results, ", ") + ")"
	}
	return text
}

type InterfaceType struct {
	Name     string
	Module   string
	Methods  []MethodSig
	Expr     *ConstraintExpr
	Exported bool
	Dynamic  bool
	Doc      string
	Instance *InstanceInfo

	key     string
	builtin string
	fill    func()
	filling bool
	table   map[string]*FuncType
}

func (i *InterfaceType) String() string {
	return i.Name
}

func (i *InterfaceType) Equals(other Type) bool {
	o, ok := other.(*InterfaceType)
	if !ok {
		return false
	}
	return i == o || (i.key != "" && i.key == o.key)
}

func (i *InterfaceType) Key() string {
	return i.key
}

func (i *InterfaceType) ensure() {
	if i.fill != nil {
		fill := i.fill
		i.fill = nil
		i.filling = true
		fill()
		i.filling = false
	}
}

func (i *InterfaceType) isTypeSet() bool {
	i.ensure()
	return i.builtin == "" && len(i.Methods) == 0 && i.Expr != nil && i.Expr.typeSet()
}

func (i *InterfaceType) ConstraintOnly() bool {
	i.ensure()
	return i.builtin != "" || (i.Expr != nil && i.Expr.constraintOnly())
}

func (i *InterfaceType) AllMethods() []MethodSig {
	i.ensure()
	methods := append([]MethodSig(nil), i.Methods...)
	if i.Expr != nil {
		for _, m := range i.Expr.requiredMethods() {
			if !hasMethodNamed(methods, m.Name) {
				methods = append(methods, m)
			}
		}
	}
	return methods
}

func (i *InterfaceType) methodTable() map[string]*FuncType {
	if i.table != nil {
		return i.table
	}
	i.table = map[string]*FuncType{}
	for slot, m := range i.AllMethods() {
		i.table[m.Name] = &FuncType{
			Name: m.Name, Params: m.Params, ParamNames: m.ParamNames, Returns: m.Returns, Fallible: m.Fallible,
			Receiver: i, Exported: true, Module: i.Module, Iface: i, Slot: slot,
		}
	}
	return i.table
}

func hasMethodNamed(methods []MethodSig, name string) bool {
	for _, m := range methods {
		if m.Name == name {
			return true
		}
	}
	return false
}

type ConstraintExpr struct {
	Kind  ast.ConstraintKind
	Type  Type
	Iface *InterfaceType
	Left  *ConstraintExpr
	Right *ConstraintExpr
}

func (e *ConstraintExpr) String() string {
	switch e.Kind {
	case ast.ConstraintOr:
		return e.Left.operand(ast.ConstraintOr) + " | " + e.Right.operand(ast.ConstraintOr)
	case ast.ConstraintAnd:
		return e.Left.operand(ast.ConstraintAnd) + " & " + e.Right.operand(ast.ConstraintAnd)
	case ast.ConstraintNot:
		return "!" + e.Left.operand(ast.ConstraintNot)
	}
	if e.Iface != nil {
		return e.Iface.String()
	}
	return e.Type.String()
}

func (e *ConstraintExpr) operand(parent ast.ConstraintKind) string {
	text := e.String()
	switch {
	case e.Kind == ast.ConstraintType:
		return text
	case parent == ast.ConstraintNot, e.Kind == ast.ConstraintOr && parent == ast.ConstraintAnd:
		return "(" + text + ")"
	}
	return text
}

func (e *ConstraintExpr) typeSet() bool {
	switch e.Kind {
	case ast.ConstraintType:
		return e.Iface == nil || e.Iface.isTypeSet()
	case ast.ConstraintNot:
		return e.Left.typeSet()
	}
	return e.Left.typeSet() && e.Right.typeSet()
}

func (e *ConstraintExpr) constraintOnly() bool {
	switch e.Kind {
	case ast.ConstraintType:
		return e.Iface == nil || e.Iface.ConstraintOnly()
	case ast.ConstraintAnd:
		return e.Left.constraintOnly() || e.Right.constraintOnly()
	}
	return true
}

func (e *ConstraintExpr) requiredMethods() []MethodSig {
	switch e.Kind {
	case ast.ConstraintType:
		if e.Iface != nil {
			return e.Iface.AllMethods()
		}
	case ast.ConstraintAnd:
		methods := e.Left.requiredMethods()
		for _, m := range e.Right.requiredMethods() {
			if !hasMethodNamed(methods, m.Name) {
				methods = append(methods, m)
			}
		}
		return methods
	}
	return nil
}

type ImplSet struct {
	declared []*InterfaceType
	explicit []implEntry
	init     func()
	typ      Type
	refs     []ast.TypeRef
	node     ast.Node
	file     string
	skip     bool
	verified bool
}

func (s *ImplSet) ensure() {
	if s != nil && s.init != nil {
		init := s.init
		s.init = nil
		init()
	}
}

func (s *ImplSet) Has(i *InterfaceType) bool {
	if s == nil {
		return false
	}
	s.ensure()
	for _, d := range s.declared {
		if d.Equals(i) {
			return true
		}
	}
	return false
}

func (s *ImplSet) add(i *InterfaceType) {
	if i == nil {
		return
	}
	for _, d := range s.declared {
		if d.Equals(i) {
			return
		}
	}
	s.declared = append(s.declared, i)
	i.ensure()
	if i.Expr != nil {
		s.addAtoms(i.Expr)
	}
}

func (s *ImplSet) addAtoms(e *ConstraintExpr) {
	switch e.Kind {
	case ast.ConstraintType:
		if e.Iface != nil && e.Iface.builtin == "" {
			s.add(e.Iface)
		}
	case ast.ConstraintAnd:
		s.addAtoms(e.Left)
		s.addAtoms(e.Right)
	}
}

func implSetOf(t Type) *ImplSet {
	if p, ok := t.(PointerType); ok {
		t = p.Element
	}
	switch v := t.(type) {
	case *StructType:
		return v.Impl
	case NamedType:
		return v.Impl
	}
	return nil
}

func sameBase(t, atom Type) bool {
	return definedBase(t).String() == definedBase(atom).String()
}

func comparableType(t Type) bool {
	switch v := definedBase(t).(type) {
	case PrimitiveType, PointerType:
		return true
	case NamedType:
		return v.Enum
	case *StructType:
		return hasOperatorMethod(t, "eq", returnsBool)
	}
	return false
}

func orderableType(t Type) bool {
	switch v := definedBase(t).(type) {
	case PrimitiveType:
		return v.Name != "bool"
	case *StructType:
		return hasOperatorMethod(t, "less", returnsBool)
	}
	return false
}

func (a *Analyser) methodProblems(t Type, i *InterfaceType) []string {
	methods := MethodsOf(t)
	var problems []string
	for _, want := range i.AllMethods() {
		have, ok := methods[want.Name]
		if !ok {
			problems = append(problems, "missing "+want.String())
			continue
		}
		if !sameSignature(want, have) {
			problems = append(problems, fmt.Sprintf("%s has the signature %s, and %s wants %s", want.Name, have.signatureText(), i.Name, want.String()))
		}
	}
	return problems
}

func (f *FuncType) signatureText() string {
	return MethodSig{Name: f.Name, Params: f.Params, Returns: f.Returns, Fallible: f.Fallible}.String()
}

func sameSignature(want MethodSig, have *FuncType) bool {
	if len(want.Params) != len(have.Params) || len(want.Returns) != len(have.Returns) || want.Fallible != have.Fallible || have.Variadic {
		return false
	}
	for i := range want.Params {
		if !want.Params[i].Equals(have.Params[i]) {
			return false
		}
	}
	for i := range want.Returns {
		if !want.Returns[i].Equals(have.Returns[i]) {
			return false
		}
	}
	return true
}

func (a *Analyser) satisfies(t Type, e *ConstraintExpr) string {
	switch e.Kind {
	case ast.ConstraintOr:
		if a.satisfies(t, e.Left) == "" || a.satisfies(t, e.Right) == "" {
			return ""
		}
		return fmt.Sprintf("%s is not in %s", t, e)
	case ast.ConstraintAnd:
		if reason := a.satisfies(t, e.Left); reason != "" {
			return reason
		}
		return a.satisfies(t, e.Right)
	case ast.ConstraintNot:
		if a.satisfies(t, e.Left) == "" {
			return fmt.Sprintf("%s is excluded by %s", t, e)
		}
		return ""
	}
	if e.Iface != nil {
		return a.satisfiesInterface(t, e.Iface)
	}
	if sameBase(t, e.Type) {
		return ""
	}
	return fmt.Sprintf("%s is not %s", t, e.Type)
}

func (a *Analyser) satisfiesInterface(t Type, i *InterfaceType) string {
	i.ensure()
	if i.builtin == "comparable" {
		if comparableType(t) {
			return ""
		}
		return fmt.Sprintf("%s cannot be compared with ==", t)
	}
	if i.builtin == "orderable" {
		if orderableType(t) {
			return ""
		}
		return fmt.Sprintf("%s cannot be ordered with <", t)
	}
	if held, isInterface := t.(*InterfaceType); isInterface {
		return coverProblem(held, i)
	}
	if len(i.Methods) > 0 && !implSetOf(t).Has(i) {
		problems := a.methodProblems(t, i)
		if len(problems) == 0 {
			return fmt.Sprintf("%s has the methods of %s but does not declare it: add `implements %s` to its declaration", elementName(t), i.Name, i.Name)
		}
		return fmt.Sprintf("%s does not implement %s: %s", elementName(t), i.Name, strings.Join(problems, "; "))
	}
	if i.Expr != nil {
		return a.satisfies(t, i.Expr)
	}
	return ""
}

func elementName(t Type) string {
	if p, ok := t.(PointerType); ok {
		return p.Element.String()
	}
	return t.String()
}

func coverProblem(held, want *InterfaceType) string {
	if held.Equals(want) {
		return ""
	}
	have := held.AllMethods()
	for _, m := range want.AllMethods() {
		found := false
		for _, h := range have {
			if h.Name == m.Name && sameMethodSig(h, m) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Sprintf("%s does not have %s", held.Name, m.String())
		}
	}
	if want.Expr != nil && want.Expr.constraintOnly() {
		return fmt.Sprintf("%s is a constraint on types, and %s holds any type", want.Name, held.Name)
	}
	return ""
}

func sameMethodSig(a, b MethodSig) bool {
	if len(a.Params) != len(b.Params) || len(a.Returns) != len(b.Returns) || a.Fallible != b.Fallible {
		return false
	}
	for i := range a.Params {
		if !a.Params[i].Equals(b.Params[i]) {
			return false
		}
	}
	for i := range a.Returns {
		if !a.Returns[i].Equals(b.Returns[i]) {
			return false
		}
	}
	return true
}

func Implements(t Type, i *InterfaceType) bool {
	return implSetOf(t).Has(i)
}

func (i *InterfaceType) Declared(t Type) bool {
	return Implements(t, i)
}
