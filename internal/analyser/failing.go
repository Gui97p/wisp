package analyser

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/parser"
)

const externRawPrefix = "__wisp_c_"

func (a *Analyser) expandFailing() {
	changed := false
	out := make([]ast.Declaration, 0, len(a.program.Declarations))
	for _, d := range a.program.Declarations {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || !needsExternWrapper(fd) {
			out = append(out, d)
			continue
		}
		raw, wrapper := a.splitExtern(fd)
		if wrapper == nil {
			out = append(out, d)
			continue
		}
		if file, known := a.declFiles[d]; known {
			a.declFiles[raw] = file
			a.declFiles[wrapper] = file
		}
		out = append(out, raw, wrapper)
		changed = true
	}
	if changed {
		a.program.Declarations = out
	}
}

func hasSliceParam(fd *ast.FuncDecl) bool {
	for _, p := range fd.Params {
		if len(p.Type.Layers) > 0 && p.Type.Layers[0].IsSpan && !p.Type.IsFunc {
			return true
		}
	}
	return false
}

func failingShape(fd *ast.FuncDecl) bool {
	if !fd.Fallible || len(fd.ReturnTypes) != 1 {
		return false
	}
	tag := fd.TagNamed("fails")
	return tag != nil && len(tag.Args) == 2
}

func needsExternWrapper(fd *ast.FuncDecl) bool {
	if !fd.Extern || fd.CVariadic || fd.Receiver != nil || len(fd.TypeParams) > 0 {
		return false
	}
	if failingShape(fd) {
		return true
	}
	return hasSliceParam(fd) && !fd.Fallible && fd.TagNamed("fails") == nil && len(fd.ReturnTypes) <= 1
}

func (a *Analyser) splitExtern(fd *ast.FuncDecl) (*ast.FuncDecl, *ast.FuncDecl) {
	tag := fd.TagNamed("fails")
	fails := failingShape(fd)
	onNull := false
	if fails {
		_, onNull = tag.Args[0].(*ast.NullLiteral)
	}

	raw := ast.Clone(fd, ast.CloneOptions{})
	raw.Name = externRawPrefix + fd.Name
	raw.Fallible = false
	raw.Exported = false
	raw.Reach = ast.ReachNone
	raw.Doc = ""
	raw.Tags = nil
	raw.Params = nil
	if onNull {
		raw.ReturnTypes[0].Layers = append([]ast.TypeLayer{{Optional: true}}, raw.ReturnTypes[0].Layers...)
	}

	var params, args []string
	for i, p := range fd.Params {
		name := p.Name
		if name == "" || name == "_" {
			name = fmt.Sprintf("__wisp_arg%d", i)
		}
		params = append(params, p.Type.String()+" "+name)
		if len(p.Type.Layers) > 0 && p.Type.Layers[0].IsSpan && !p.Type.IsFunc {
			element := p.Type
			element.Layers = append([]ast.TypeLayer{}, p.Type.Layers[1:]...)
			pointer := p.Type
			pointer.Layers = append([]ast.TypeLayer{{Pointer: true}}, p.Type.Layers[1:]...)
			raw.Params = append(raw.Params, ast.Param{Name: name, Type: pointer, Line: p.Line, Col: p.Col, EndLine: p.EndLine, EndCol: p.EndCol})
			raw.Params = append(raw.Params, ast.Param{Name: name + "_len", Type: ast.TypeRef{Name: "c_size"}, Line: p.Line, Col: p.Col, EndLine: p.EndLine, EndCol: p.EndCol})
			args = append(args, fmt.Sprintf("(*%s)(%s)", element.String(), name), fmt.Sprintf("c_size(len(%s))", name))
			continue
		}
		next := p
		next.Name = name
		raw.Params = append(raw.Params, next)
		args = append(args, name)
	}
	call := fmt.Sprintf("%s(%s)", raw.Name, strings.Join(args, ", "))

	var source strings.Builder
	returns := ""
	if len(fd.ReturnTypes) == 1 {
		returns = " " + fd.ReturnTypes[0].String()
	}
	switch {
	case fails && onNull:
		fmt.Fprintf(&source, "func %s(%s) !%s {\n%s result = %s ?? { return __wisp_failure; };\nreturn result;\n}\n", fd.Name, strings.Join(params, ", "), returns[1:], returns[1:], call)
	case fails:
		fmt.Fprintf(&source, "func %s(%s) !%s {\n%s result = %s;\nif __wisp_cond { return __wisp_failure; }\nreturn result;\n}\n", fd.Name, strings.Join(params, ", "), returns[1:], returns[1:], call)
	case returns == "":
		fmt.Fprintf(&source, "func %s(%s) {\n%s;\n}\n", fd.Name, strings.Join(params, ", "), call)
	default:
		fmt.Fprintf(&source, "func %s(%s)%s {\nreturn %s;\n}\n", fd.Name, strings.Join(params, ", "), returns, call)
	}

	program := parser.NewParser(lexer.NewLexer([]byte(source.String()))).ParseProgram()
	if len(program.Declarations) != 1 {
		return nil, nil
	}
	built, ok := program.Declarations[0].(*ast.FuncDecl)
	if !ok {
		return nil, nil
	}

	at := ast.Position{}
	at.Line, at.Col = fd.Position()
	at.EndLine, at.EndCol = fd.EndPosition()
	wrapper := ast.Clone(built, ast.CloneOptions{At: &at, Replace: func(n ast.Node) (ast.Node, bool) {
		ident, isIdent := n.(*ast.IdentLiteral)
		if !isIdent || !fails {
			return nil, false
		}
		switch ident.Value {
		case "__wisp_cond":
			return ast.Clone(tag.Args[0], ast.CloneOptions{}), true
		case "__wisp_failure":
			if _, isCall := tag.Args[1].(*ast.CallExpr); isCall {
				return ast.Clone(tag.Args[1], ast.CloneOptions{}), true
			}
			name := &ast.IdentLiteral{Value: "Error", Synthetic: true}
			call := &ast.CallExpr{Name: name, Args: []ast.Expression{ast.Clone(tag.Args[1], ast.CloneOptions{})}}
			name.SetPos(at.Line, at.Col)
			name.SetEndPos(at.EndLine, at.EndCol)
			call.SetPos(at.Line, at.Col)
			call.SetEndPos(at.EndLine, at.EndCol)
			return call, true
		}
		return nil, false
	}})
	wrapper.Doc = fd.Doc
	wrapper.Exported = fd.Exported
	wrapper.Reach = fd.Reach
	wrapper.Selector, wrapper.SelectorAt = fd.Selector, fd.SelectorAt
	wrapper.NameAt = fd.NameAt
	wrapper.Params = append([]ast.Param{}, wrapper.Params...)
	for i := range wrapper.Params {
		if i < len(fd.Params) {
			wrapper.Params[i].Line, wrapper.Params[i].Col = fd.Params[i].Line, fd.Params[i].Col
			wrapper.Params[i].EndLine, wrapper.Params[i].EndCol = fd.Params[i].EndLine, fd.Params[i].EndCol
		}
	}
	for _, t := range fd.Tags {
		if t.Name != "fails" && t.Name != "symbol" {
			wrapper.Tags = append(wrapper.Tags, t)
		}
	}
	return raw, wrapper
}
