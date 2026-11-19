package lsp

import (
	"fmt"
	"reflect"
	"slices"

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
		if sd, ok := d.(*ast.StructDecl); ok && sd.Name == ref.Name {
			declared = sd.TypeParams
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
