package lsp

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type typeParamScope struct {
	owner  string
	params []ast.TypeParam
	uses   map[[2]int]ast.TypeParam
}

func typeParamScopes(program *ast.Program) []typeParamScope {
	var scopes []typeParamScope
	for _, decl := range program.Declarations {
		var params []ast.TypeParam
		var owner string
		switch d := decl.(type) {
		case *ast.FuncDecl:
			params = d.TypeParams
			owner = "function " + d.Name
			if d.Receiver != nil {
				owner = "method " + d.Name
				params = receiverTypeParams(program, d)
			}
		case *ast.StructDecl:
			params = d.TypeParams
			owner = "struct " + d.Name
		case *ast.TypeDecl:
			params = d.TypeParams
			owner = "type " + d.Name
			if d.Data {
				owner = "enum " + d.Name
			}
		case *ast.InterfaceDecl:
			params = d.TypeParams
			owner = "interface " + d.Name
		}
		if len(params) == 0 {
			continue
		}
		byName := map[string]ast.TypeParam{}
		for _, param := range params {
			byName[param.Name] = param
		}
		scope := typeParamScope{owner: owner, params: params, uses: map[[2]int]ast.TypeParam{}}
		walkTypeRefs(reflect.ValueOf(decl), func(ref ast.TypeRef) {
			if param, ok := byName[ref.Name]; ok && ref.Module == "" {
				scope.uses[[2]int{ref.Line, ref.Col}] = param
			}
		})
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Body != nil {
			walkStmt(fd.Body, func(expr ast.Expression) {
				if id, ok := expr.(*ast.IdentLiteral); ok {
					if param, ok := byName[id.Value]; ok {
						line, col := id.Position()
						scope.uses[[2]int{line, col}] = param
					}
				}
			})
		}
		scopes = append(scopes, scope)
	}
	return scopes
}

func typeParamUses(program *ast.Program) map[[2]int]bool {
	uses := map[[2]int]bool{}
	for _, scope := range typeParamScopes(program) {
		for at := range scope.uses {
			uses[at] = true
		}
	}
	return uses
}

type typeParamHit struct {
	param ast.TypeParam
	owner string
	rng   ast.Position
	sites []ast.Position
	taken map[string]bool
}

func (s typeParamScope) hit(param ast.TypeParam, rng ast.Position) *typeParamHit {
	hit := &typeParamHit{param: param, owner: s.owner, rng: rng, taken: map[string]bool{}}
	for _, p := range s.params {
		hit.taken[p.Name] = true
		if p.Name == param.Name {
			hit.sites = append(hit.sites, p.At)
		}
	}
	for at, used := range s.uses {
		if used.Name == param.Name && !slices.ContainsFunc(hit.sites, func(p ast.Position) bool { return p.Line == at[0] && p.Col == at[1] }) {
			hit.sites = append(hit.sites, ast.Position{Line: at[0], Col: at[1], EndLine: at[0], EndCol: at[1] + len(param.Name) - 1})
		}
	}
	return hit
}

func typeParamAt(program *ast.Program, line, col int) *typeParamHit {
	within := func(line0, col0 int, name string) bool {
		return line == line0 && col >= col0 && col < col0+len(name)
	}
	for _, scope := range typeParamScopes(program) {
		for _, param := range scope.params {
			if within(param.At.Line, param.At.Col, param.Name) {
				return scope.hit(param, param.At)
			}
		}
		for at, param := range scope.uses {
			if within(at[0], at[1], param.Name) {
				return scope.hit(param, ast.Position{Line: at[0], Col: at[1], EndLine: at[0], EndCol: at[1] + len(param.Name) - 1})
			}
		}
	}
	return nil
}

func typeParamHover(hit *typeParamHit) string {
	code := "type parameter " + hit.param.Name
	if hit.param.Const {
		code = fmt.Sprintf("const %s %s", hit.param.Type.String(), hit.param.Name)
	}
	doc := fmt.Sprintf("Type parameter of the %s: each use of it picks a type.", hit.owner)
	if hit.param.Const {
		doc = fmt.Sprintf("Value parameter of the %s: each use of it picks a constant.", hit.owner)
	}
	return renderDoc(code, doc)
}

func typeParamLocation(path string, hit *typeParamHit) []protocol.Location {
	return []protocol.Location{pointLocation(path, hit.param.At.Line, hit.param.At.Col)}
}

func receiverTypeParams(program *ast.Program, fd *ast.FuncDecl) []ast.TypeParam {
	ref := fd.Receiver.Type
	var declared []ast.TypeParam
	for _, d := range program.Declarations {
		switch decl := d.(type) {
		case *ast.StructDecl:
			if decl.Name == ref.Name {
				declared = decl.TypeParams
			}
		case *ast.TypeDecl:
			if decl.Name == ref.Name {
				declared = decl.TypeParams
			}
		}
		if declared != nil {
			break
		}
	}
	var params []ast.TypeParam
	for i, arg := range ref.Args {
		if arg.Type == nil {
			continue
		}
		param := ast.TypeParam{Name: arg.Type.Name, At: ast.Position{Line: arg.Type.Line, Col: arg.Type.Col, EndLine: arg.Type.EndLine, EndCol: arg.Type.EndCol}}
		if i < len(declared) {
			param.Const, param.Type = declared[i].Const, declared[i].Type
		}
		params = append(params, param)
	}
	return params
}

func typeParamRename(uri protocol.DocumentUri, hit *typeParamHit, newName string) (*protocol.WorkspaceEdit, error) {
	if hit.taken[newName] && newName != hit.param.Name {
		return nil, fmt.Errorf("%s is already a parameter of the %s", newName, hit.owner)
	}
	var edits []protocol.TextEdit
	for _, at := range hit.sites {
		edits = append(edits, protocol.TextEdit{Range: rangeOf(at.Line, at.Col, at.Line, at.Col+len(hit.param.Name)-1), NewText: newName})
	}
	return &protocol.WorkspaceEdit{Changes: map[protocol.DocumentUri][]protocol.TextEdit{uri: edits}}, nil
}

func typeParamsText(params []ast.TypeParam) string {
	names := make([]string, len(params))
	for i, p := range params {
		names[i] = p.Name
		if p.Constraint != nil {
			names[i] = p.Constraint.String() + " " + p.Name
		}
		if p.Const {
			names[i] = "const " + p.Type.String() + " " + p.Name
		}
	}
	return "<" + strings.Join(names, ", ") + ">"
}

func paramsText(params []ast.Param) string {
	parts := make([]string, len(params))
	for i, p := range params {
		name := p.Name
		if p.Variadic || p.Any {
			name = "..." + name
		}
		if p.Any {
			parts[i] = name
			continue
		}
		parts[i] = refText(p.Type) + " " + name
	}
	return strings.Join(parts, ", ")
}

func genericHover(sym *analyser.Symbol) string {
	g := sym.Generic
	switch {
	case g.Func != nil:
		fd := g.Func
		text := fmt.Sprintf("func %s%s(%s)", fd.Name, typeParamsText(fd.TypeParams), paramsText(fd.Params))
		results := make([]string, len(fd.ReturnTypes))
		for i := range fd.ReturnTypes {
			results[i] = refText(fd.ReturnTypes[i])
		}
		bang := ""
		if fd.Fallible {
			bang = "!"
		}
		switch len(results) {
		case 0:
			if fd.Fallible {
				text += " !"
			}
		case 1:
			text += " " + bang + results[0]
		default:
			text += " " + bang + "(" + strings.Join(results, ", ") + ")"
		}
		return renderDoc(text, joinDocs(sym.Doc, "Generic function: each use picks its types."))
	case g.Struct != nil:
		sd := g.Struct
		var b strings.Builder
		fmt.Fprintf(&b, "struct %s%s {", sd.Name, typeParamsText(sd.TypeParams))
		for i := range sd.Members {
			fmt.Fprintf(&b, "\n  %s %s", refText(sd.Members[i].Type), sd.Members[i].Name)
		}
		b.WriteString("\n}")
		return renderDoc(b.String(), joinDocs(sym.Doc, "Generic struct: each use picks its types."))
	case g.Sum != nil:
		td := g.Sum
		var b strings.Builder
		fmt.Fprintf(&b, "enum %s%s {", td.Name, typeParamsText(td.TypeParams))
		for _, m := range td.Members {
			fmt.Fprintf(&b, "\n  %s", m.Name)
			if m.Payload {
				fmt.Fprintf(&b, " { %s }", paramsText(m.Fields))
			}
		}
		b.WriteString("\n}")
		return renderDoc(b.String(), joinDocs(sym.Doc, "Generic enum: each use picks its types."))
	case g.Interface != nil:
		return renderDoc(interfaceDeclText(g.Interface), joinDocs(sym.Doc, "Generic interface: each use picks its types."))
	case g.Alias != nil:
		td := g.Alias
		return renderDoc(fmt.Sprintf("type %s%s %s", td.Name, typeParamsText(td.TypeParams), refText(td.Underlying)), joinDocs(sym.Doc, "Generic type: each use picks its types."))
	}
	return ""
}

func methodDeclHover(program *ast.Program, src []byte, line, col int) (string, bool) {
	word := wordUnderCursor(src, line, col)
	if word == "" {
		return "", false
	}
	for _, decl := range program.Declarations {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Receiver == nil || fd.Name != word || fd.Generated || fd.Instance != "" {
			continue
		}
		if l, _ := fd.Position(); l != line {
			continue
		}
		receiver := refText(fd.Receiver.Type)
		text := fmt.Sprintf("func (%s %s) %s(%s)", receiver, fd.Receiver.Name, fd.Name, paramsText(fd.Params))
		results := make([]string, len(fd.ReturnTypes))
		for i := range fd.ReturnTypes {
			results[i] = refText(fd.ReturnTypes[i])
		}
		switch len(results) {
		case 0:
		case 1:
			text += " " + results[0]
		default:
			text += " (" + strings.Join(results, ", ") + ")"
		}
		if fd.Fallible {
			text += " !"
		}
		return renderDoc(text, joinDocs(fd.Doc, "Method.")), true
	}
	return "", false
}

func interfaceDeclText(decl *ast.InterfaceDecl) string {
	var b strings.Builder
	fmt.Fprintf(&b, "interface %s", decl.Name)
	if len(decl.TypeParams) > 0 {
		b.WriteString(typeParamsText(decl.TypeParams))
	}
	if len(decl.Methods) == 0 && decl.Constraint != nil {
		fmt.Fprintf(&b, " { %s }", decl.Constraint.String())
		return b.String()
	}
	b.WriteString(" {")
	if decl.Constraint != nil {
		fmt.Fprintf(&b, "\n  %s", decl.Constraint.String())
	}
	for _, m := range decl.Methods {
		fmt.Fprintf(&b, "\n  %s", interfaceMethodText(m))
	}
	b.WriteString("\n}")
	return b.String()
}

func interfaceMethodText(m ast.InterfaceMethod) string {
	text := fmt.Sprintf("func %s(%s)", m.Name, paramsText(m.Params))
	results := make([]string, len(m.ReturnTypes))
	for i := range m.ReturnTypes {
		results[i] = refText(m.ReturnTypes[i])
	}
	switch len(results) {
	case 0:
	case 1:
		text += " " + results[0]
	default:
		text += " (" + strings.Join(results, ", ") + ")"
	}
	if m.Fallible {
		text += " !"
	}
	return text
}

func interfaceMethodDeclHover(program *ast.Program, line, col int) (string, *protocol.Range) {
	for _, decl := range program.Declarations {
		d, ok := decl.(*ast.InterfaceDecl)
		if !ok {
			continue
		}
		for _, m := range d.Methods {
			if positionContains(m.NameAt, line, col) {
				return renderDoc(interfaceMethodText(m), joinDocs(m.Doc, "Method of the interface `"+d.Name+"`.")), positionRange(m.NameAt)
			}
		}
	}
	return "", nil
}
