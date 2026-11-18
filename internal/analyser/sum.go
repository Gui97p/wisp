package analyser

import (
	"path/filepath"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

const sumTag = "$tag"

func (a *Analyser) registerSumName(td *ast.TypeDecl) {
	st := &StructType{
		Name:      td.Name,
		Fields:    make(map[string]Type),
		Methods:   make(map[string]*FuncType),
		FieldDocs: make(map[string]string),
		FieldDirs: make(map[string]string),
		Module:    a.module,
		Dir:       filepath.Dir(a.currentFile),
		Exported:  td.Exported,
		Sum:       &SumInfo{ByName: make(map[string]int, len(td.Members))},
	}
	for i, m := range td.Members {
		variant := SumVariant{Name: m.Name, Tag: int64(i)}
		if m.Payload {
			variant.Field = "$" + m.Name
			variant.Payload = &StructType{
				Name:      td.Name + "." + m.Name,
				Fields:    make(map[string]Type),
				Methods:   make(map[string]*FuncType),
				FieldDocs: make(map[string]string),
				FieldDirs: make(map[string]string),
				Module:    a.module,
				Dir:       filepath.Dir(a.currentFile),
				Exported:  td.Exported,
			}
		}
		st.Sum.ByName[m.Name] = len(st.Sum.Variants)
		st.Sum.Variants = append(st.Sum.Variants, variant)
	}

	line, col := td.Position()
	symbol := &Symbol{Name: td.Name, Kind: STRUCT, Type: st, Line: line, Col: col, File: a.currentFile, Doc: td.Doc}
	if !a.scope.Define(symbol) {
		a.errorAlreadyDeclared(td, STRUCT, td.Name)
	}
}

func (a *Analyser) registerSumFields(td *ast.TypeDecl) {
	symbol, ok := a.scope.Resolve(td.Name)
	if !ok {
		return
	}
	st, ok := symbol.Type.(*StructType)
	if !ok || st.Sum == nil {
		return
	}

	st.Fields[sumTag] = PrimitiveType{Name: "int"}
	st.Order = append(st.Order, sumTag)
	for i, m := range td.Members {
		variant := st.Sum.Variants[i]
		if variant.Payload == nil {
			continue
		}
		for _, field := range m.Fields {
			fieldType := a.resolveTypeRef(td, a.scope, field.Type)
			if fieldType == nil {
				continue
			}
			if _, dup := variant.Payload.Fields[field.Name]; dup {
				a.errorf(td, "duplicated field %s in variant %s", field.Name, m.Name)
				continue
			}
			variant.Payload.Fields[field.Name] = fieldType
			variant.Payload.FieldDocs[field.Name] = field.Doc
			variant.Payload.Order = append(variant.Payload.Order, field.Name)
		}
		st.Fields[variant.Field] = variant.Payload
		st.Order = append(st.Order, variant.Field)
	}
	st.complete = true
}

func sumOf(t Type) (*StructType, bool) {
	if ptr, ok := t.(PointerType); ok && !ptr.Nullable {
		t = ptr.Element
	}
	st, ok := t.(*StructType)
	if !ok || st.Sum == nil {
		return nil, false
	}
	return st, true
}

func (a *Analyser) sumTypeExpr(expr ast.Expression) (*StructType, bool) {
	var sym *Symbol
	var found bool
	switch e := expr.(type) {
	case *ast.IdentLiteral:
		sym, found = a.scope.Resolve(e.Value)
	case *ast.MemberExpr:
		module, ok := e.Object.(*ast.IdentLiteral)
		if !ok {
			return nil, false
		}
		sym, found = a.lookupType(a.scope, module.Value, e.Field)
	}
	if !found || sym.Kind != STRUCT {
		return nil, false
	}
	st, ok := sym.Type.(*StructType)
	if !ok || st.Sum == nil {
		return nil, false
	}
	return st, true
}

func (a *Analyser) variantRef(expr ast.Expression, sum *StructType) (SumVariant, bool) {
	switch e := expr.(type) {
	case *ast.DotIdent:
		return sum.Sum.Find(e.Name)
	case *ast.MemberExpr:
		owner, ok := a.sumTypeExpr(e.Object)
		if !ok || !owner.Equals(sum) {
			return SumVariant{}, false
		}
		return sum.Sum.Find(e.Field)
	}
	return SumVariant{}, false
}

func (a *Analyser) intLiteral(near ast.Node, value int64) *ast.IntLiteral {
	lit := &ast.IntLiteral{Value: value}
	a.at(near, lit)
	return lit
}

func (a *Analyser) checkSumTest(expr *ast.BinaryExpr, left Type) (Type, bool) {
	sum, ok := sumOf(left)
	if !ok {
		return nil, false
	}
	variant, ok := a.variantRef(expr.Right, sum)
	if !ok {
		a.errorf(expr, "%s has data, so it is compared by variant only: write `x == .Name`, or a switch to see the data", sum.Name).
			WithHelp("the variants are %s", sumNames(sum))
		a.dropDot(expr.Right)
		return PrimitiveType{Name: "bool"}, true
	}
	a.dropDot(expr.Right)
	if expr.Operator != "==" && expr.Operator != "!=" {
		a.errorf(expr, "a value of %s can only be compared with == or !=", sum.Name)
		return PrimitiveType{Name: "bool"}, true
	}

	subject := expr.Left
	test := &SumTest{Sum: sum, Variant: variant, Subject: subject}
	expr.Left = a.optionalMember(subject, sumTag)
	expr.Right = a.intLiteral(expr.Right, variant.Tag)
	a.info.Variants[expr.Right] = &VariantInfo{Sum: sum, Variant: variant}
	t := a.checkBinaryExpr(expr)
	a.info.SumTests[expr] = test
	return t, true
}

func sumNames(sum *StructType) string {
	names := make([]string, len(sum.Sum.Variants))
	for i, v := range sum.Sum.Variants {
		names[i] = "." + v.Name
	}
	return strings.Join(names, ", ")
}

func (a *Analyser) checkVariantLiteral(expr *ast.StructLiteral, sum *StructType, name string) Type {
	variant, ok := sum.Sum.Find(name)
	if !ok {
		a.suggest(a.errorf(expr, "%s has no variant %s", sum.Name, name), name, variantNames(sum))
		return InvalidType{}
	}
	a.info.Variants[expr] = &VariantInfo{Sum: sum, Variant: variant}
	if variant.Payload == nil {
		if len(expr.Keys) > 0 {
			a.errorf(expr, "variant %s has no data, so it takes no fields", name)
		}
		return sum
	}
	a.checkFieldInits(expr, variant.Payload)
	return sum
}

func variantNames(sum *StructType) []string {
	names := make([]string, len(sum.Sum.Variants))
	for i, v := range sum.Sum.Variants {
		names[i] = v.Name
	}
	return names
}

func (a *Analyser) resolveVariantValue(node ast.Expression, name string, to Type) bool {
	sum, ok := to.(*StructType)
	if !ok || sum.Sum == nil {
		return false
	}
	variant, found := sum.Sum.Find(name)
	if !found {
		a.errorf(node, "%s has no variant %s", sum.Name, name)
		return true
	}
	if variant.Payload != nil {
		a.errorf(node, "variant %s has data: write .%s{...} with its fields", name, name)
		return true
	}
	a.info.Variants[node] = &VariantInfo{Sum: sum, Variant: variant}
	a.info.Types[node] = sum
	return true
}

func (a *Analyser) resolveVariantLiteral(lit *ast.StructLiteral, to Type) bool {
	sum, ok := to.(*StructType)
	if !ok || sum.Sum == nil {
		a.errorf(lit, "cannot infer the type of .%s{...} here: %s is not an enum with data", lit.Name, to.String())
		return true
	}
	delete(a.pendingDotLits, lit)
	a.info.Types[lit] = a.checkVariantLiteral(lit, sum, lit.Name)
	return true
}
