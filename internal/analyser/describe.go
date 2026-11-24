package analyser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func TypeText(t Type) string {
	switch v := t.(type) {
	case *StructType:
		return v.Name
	case PointerType:
		return "*" + TypeText(v.Element)
	case ArrayType:
		return fmt.Sprintf("[%d]%s", v.Size, TypeText(v.Element))
	case SpanType:
		return "[]" + TypeText(v.Element)
	case UntypedIntType:
		return "int"
	case UntypedFloatType:
		return "float"
	}
	return t.String()
}

func FuncSignature(name string, ft *FuncType, decl *ast.FuncDecl) string {
	if text, ok := genericMethodSignature(name, ft); ok {
		return text
	}
	receiver := ""
	if ft.Receiver != nil {
		receiver = "(" + TypeText(ft.Receiver)
		switch {
		case decl != nil && decl.Receiver != nil && decl.Receiver.Name != "":
			receiver += " " + decl.Receiver.Name
		case ft.ReceiverName != "":
			receiver += " " + ft.ReceiverName
		}
		receiver += ") "
	}

	return fmt.Sprintf("func %s%s(%s)%s", receiver, name, strings.Join(ParamLabels(ft, decl), ", "), returnsText(ft))
}

func ParamLabels(ft *FuncType, decl *ast.FuncDecl) []string {
	if fd := genericMethodDecl(ft); fd != nil {
		return genericMethodParams(fd)
	}
	labels := make([]string, 0, len(ft.Params)+1)
	first := 0
	if ft.Decl != nil && ft.Decl.TagNamed("caller") != nil && isHereParam(ft.Decl) {
		first = 1
	}
	for i, p := range ft.Params {
		if i < first {
			continue
		}
		label := TypeText(p)
		paramName := ""
		switch {
		case decl != nil && i < len(decl.Params):
			paramName = decl.Params[i].Name
		case i < len(ft.ParamNames):
			paramName = ft.ParamNames[i]
		}
		if _, boxed := p.(BoxedType); boxed && ft.Variadic && i == len(ft.Params)-1 {
			labels = append(labels, "..."+paramName)
			continue
		}
		if ft.Variadic && i == len(ft.Params)-1 {
			label += " ..."
		}
		if paramName != "" {
			label += " " + paramName
		}
		labels = append(labels, label)
	}
	if ft.CVariadic {
		labels = append(labels, "...")
	}
	return labels
}

func returnsText(ft *FuncType) string {
	var b strings.Builder
	if ft.Fallible {
		b.WriteString(" !")
	}
	switch len(ft.Returns) {
	case 0:
	case 1:
		if !ft.Fallible {
			b.WriteString(" ")
		}
		b.WriteString(TypeText(ft.Returns[0]))
	default:
		if !ft.Fallible {
			b.WriteString(" ")
		}
		parts := make([]string, len(ft.Returns))
		for i, r := range ft.Returns {
			parts[i] = TypeText(r)
		}
		b.WriteString("(")
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString(")")
	}
	return b.String()
}

func EnumText(t NamedType) string {
	var b strings.Builder
	fmt.Fprintf(&b, "enum %s {\n", t.Name)
	for _, name := range t.MemberOrder {
		fmt.Fprintf(&b, "  %s = %d,\n", name, t.Members[name])
	}
	b.WriteString("}")
	return b.String()
}

func variantFields(variant SumVariant) string {
	if variant.Payload == nil || len(variant.Payload.Order) == 0 {
		return ""
	}
	parts := make([]string, 0, len(variant.Payload.Order))
	for _, field := range variant.Payload.Order {
		parts = append(parts, TypeText(variant.Payload.Fields[field])+" "+field)
	}
	return " { " + strings.Join(parts, "; ") + " }"
}

func VariantText(sum *StructType, variant SumVariant) string {
	return sum.Name + "." + variant.Name + variantFields(variant)
}

func StructText(st *StructType) string {
	var b strings.Builder
	if st.Optional != nil {
		return st.Name
	}
	if st.Sum != nil {
		fmt.Fprintf(&b, "enum %s {\n", st.Name)
		for i, variant := range st.Sum.Variants {
			comma := ","
			if i == len(st.Sum.Variants)-1 {
				comma = ""
			}
			fmt.Fprintf(&b, "  %s%s%s\n", variant.Name, variantFields(variant), comma)
		}
		b.WriteString("}")
	} else {
		fmt.Fprintf(&b, "struct %s {\n", st.Name)
		for _, field := range st.Order {
			if strings.HasPrefix(field, "$") {
				continue
			}
			fmt.Fprintf(&b, "  %s %s\n", TypeText(st.Fields[field]), field)
		}
		b.WriteString("}")
	}

	names := make([]string, 0, len(st.Methods))
	for name := range st.Methods {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) > 0 {
		b.WriteString("\n")
		for _, name := range names {
			fmt.Fprintf(&b, "\n%s", FuncSignature(name, st.Methods[name], nil))
		}
	}

	return b.String()
}

func genericMethodDecl(ft *FuncType) *ast.FuncDecl {
	if ft == nil || ft.GenericMethod == nil || ft.GenericMethod.Generic == nil {
		return nil
	}
	fd := ft.GenericMethod.Generic.Func
	if fd == nil || len(fd.Params) == 0 {
		return nil
	}
	return fd
}

func genericMethodParams(fd *ast.FuncDecl) []string {
	labels := make([]string, 0, len(fd.Params)-1)
	for _, p := range fd.Params[1:] {
		label := p.Type.String()
		switch {
		case p.Any:
			label = "..." + p.Name
		case p.Variadic:
			label += " ..." + p.Name
		default:
			label += " " + p.Name
		}
		labels = append(labels, label)
	}
	return labels
}

func genericMethodSignature(name string, ft *FuncType) (string, bool) {
	fd := genericMethodDecl(ft)
	if fd == nil {
		return "", false
	}
	return GenericMethodText(name, fd, TypeText(ft.Receiver)), true
}

func GenericMethodText(name string, fd *ast.FuncDecl, receiverText string) string {
	own := len(fd.TypeParams)
	for _, arg := range fd.Params[0].Type.Args {
		if arg.Type != nil {
			own--
		}
	}
	if own < 0 {
		own = 0
	}
	params := make([]string, own)
	for i, p := range fd.TypeParams[:own] {
		params[i] = p.Name
		if p.Constraint != nil {
			params[i] = p.Constraint.String() + " " + p.Name
		}
	}
	generics := ""
	if own > 0 {
		generics = "<" + strings.Join(params, ", ") + ">"
	}
	receiver := receiverText
	if fd.Params[0].Name != "" && fd.Params[0].Name != hiddenSelf {
		receiver += " " + fd.Params[0].Name
	}
	results := make([]string, len(fd.ReturnTypes))
	for i := range fd.ReturnTypes {
		results[i] = fd.ReturnTypes[i].String()
	}
	text := fmt.Sprintf("func (%s) %s%s(%s)", receiver, name, generics, strings.Join(genericMethodParams(fd), ", "))
	switch {
	case fd.Fallible && len(results) == 0:
		text += " !"
	case fd.Fallible:
		text += " !" + strings.Join(results, ", ")
	case len(results) > 0:
		text += " " + strings.Join(results, ", ")
	}
	return text
}
