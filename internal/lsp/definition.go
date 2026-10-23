package lsp

import (
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func definition(uri protocol.DocumentUri, pos protocol.Position) []protocol.Location {
	path := uriToPath(uri)
	line := int(pos.Line) + 1
	col := int(pos.Character) + 1

	pa, err := analyseProject(path)
	if err != nil || pa == nil {
		return nil
	}
	m := pa.moduleOf(path)
	if m == nil {
		return nil
	}

	for _, d := range m.merged.Declarations {
		if imp, ok := d.(*ast.ImportDecl); ok && m.declFiles[d] == path && contains(imp, line, col) {
			return pa.moduleFile(imp.Path)
		}
	}

	if expr := exprInProgram(m.merged, line, col); expr != nil {
		if locations := exprDefinition(pa, m, expr, line, col); len(locations) > 0 {
			return locations
		}
	}

	if ref := typeRefAt(m.merged, line, col); ref != nil {
		if ref.Module != "" && line == ref.Line && col >= ref.Col && col < ref.Col+len(ref.Module) {
			return pa.moduleLocations(m, scopeFor(m.info, line, col), ref.Module)
		}
		if loc := symbolLocation(typeRefSymbol(ref, m.info, line, col)); loc != nil {
			return []protocol.Location{*loc}
		}
	}
	return nil
}

func (pa *projectAnalysis) moduleOf(path string) *moduleAnalysis {
	for i := range pa.modules {
		if slices.Contains(pa.modules[i].mod.FilePaths, path) {
			return &pa.modules[i]
		}
	}
	return nil
}

func (pa *projectAnalysis) moduleByPath(modulePath string) *moduleAnalysis {
	for i := range pa.modules {
		if pa.modules[i].mod.Path == modulePath {
			return &pa.modules[i]
		}
	}
	return nil
}

func (pa *projectAnalysis) moduleFile(modulePath string) []protocol.Location {
	target := pa.moduleByPath(modulePath)
	if target == nil || len(target.mod.FilePaths) == 0 {
		return nil
	}
	return []protocol.Location{pointLocation(target.mod.FilePaths[0], 1, 1)}
}

func (pa *projectAnalysis) moduleLocations(m *moduleAnalysis, scope *analyser.Scope, name string) []protocol.Location {
	var locations []protocol.Location
	sym, ok := scope.Resolve(name)
	if !ok || sym.Kind != analyser.MODULE {
		return nil
	}
	if sym.File != "" && sym.Line > 0 {
		locations = append(locations, pointLocation(sym.File, sym.Line, sym.Col))
	}
	for _, d := range m.merged.Declarations {
		imp, isImport := d.(*ast.ImportDecl)
		if !isImport || imp.Alias != name {
			continue
		}
		locations = append(locations, pa.moduleFile(imp.Path)...)
		break
	}
	return locations
}

func exprDefinition(pa *projectAnalysis, m *moduleAnalysis, expr ast.Expression, line, col int) []protocol.Location {
	info := m.info

	switch e := expr.(type) {
	case *ast.IdentLiteral:
		sym := info.Idents[e]
		if sym == nil {
			return nil
		}
		if sym.Kind == analyser.MODULE {
			return pa.moduleLocations(m, scopeFor(info, line, col), sym.Name)
		}
		if loc := symbolLocation(sym); loc != nil {
			return []protocol.Location{*loc}
		}
	case *ast.DotIdent:
		if value, ok := info.EnumValues[e]; ok {
			return pa.enumMember(value)
		}
	case *ast.MemberExpr:
		if value, ok := info.EnumValues[e]; ok {
			return pa.enumMember(value)
		}
		if sym := moduleMember(e, info, scopeFor(info, line, col)); sym != nil {
			if loc := symbolLocation(sym); loc != nil {
				return []protocol.Location{*loc}
			}
		}
		if member := info.Members[e]; member != nil && member.Struct != nil && member.Kind == analyser.MemberField {
			return pa.structField(member.Struct, e.Field)
		}
		for _, call := range info.MethodCalls {
			if name, ok := methodName(call, info, e); ok {
				return pa.method(call, name)
			}
		}
	case *ast.StructLiteral:
		st, ok := info.Types[e].(*analyser.StructType)
		if !ok {
			return nil
		}
		for i, at := range e.KeyAt {
			if i < len(e.Keys) && positionContains(at, line, col) {
				return pa.structField(st, e.Keys[i])
			}
		}
	}
	return nil
}

func moduleMember(e *ast.MemberExpr, info *analyser.Info, scope *analyser.Scope) *analyser.Symbol {
	if mt, ok := info.Types[e.Object].(*analyser.ModuleType); ok {
		return mt.Exports[e.Field]
	}
	if ident, ok := e.Object.(*ast.IdentLiteral); ok {
		if sym, found := scope.Resolve(ident.Value); found && sym.Kind == analyser.MODULE {
			if mt, isModule := sym.Type.(*analyser.ModuleType); isModule {
				return mt.Exports[e.Field]
			}
		}
	}
	return nil
}

func methodName(call *analyser.MethodCall, info *analyser.Info, member *ast.MemberExpr) (string, bool) {
	for expr, candidate := range info.MethodCalls {
		if candidate != call {
			continue
		}
		if m, ok := expr.Name.(*ast.MemberExpr); ok && m == member {
			return m.Field, true
		}
	}
	return "", false
}

func (pa *projectAnalysis) structField(st *analyser.StructType, field string) []protocol.Location {
	target := pa.moduleByPath(st.Module)
	if target == nil {
		return nil
	}
	for _, d := range target.merged.Declarations {
		decl, ok := d.(*ast.StructDecl)
		if !ok || decl.Name != st.Name {
			continue
		}
		for _, member := range decl.Members {
			if member.Name == field && member.Line > 0 {
				return []protocol.Location{pointLocation(target.declFiles[d], member.Line, member.Col)}
			}
		}
	}
	return nil
}

func (pa *projectAnalysis) enumMember(value analyser.EnumValue) []protocol.Location {
	target := pa.moduleByPath(value.Type.Module)
	if target == nil {
		return nil
	}
	for _, d := range target.merged.Declarations {
		decl, ok := d.(*ast.TypeDecl)
		if !ok || !decl.IsEnum || decl.Name != value.Type.Name {
			continue
		}
		for _, member := range decl.Members {
			if member.Name == value.Name && member.Line > 0 {
				return []protocol.Location{pointLocation(target.declFiles[d], member.Line, member.Col)}
			}
		}
	}
	return nil
}

func (pa *projectAnalysis) method(call *analyser.MethodCall, name string) []protocol.Location {
	target := pa.moduleByPath(call.Module)
	if target == nil {
		return nil
	}
	for _, d := range target.merged.Declarations {
		decl, ok := d.(*ast.FuncDecl)
		if !ok || decl.Receiver == nil || decl.Name != name || decl.Receiver.Type.Name != call.Owner {
			continue
		}
		line, col := decl.Position()
		return []protocol.Location{pointLocation(target.declFiles[d], line, col)}
	}
	return nil
}

func pointLocation(file string, line, col int) protocol.Location {
	target := protocol.Position{Line: uint32(line - 1), Character: uint32(col - 1)}
	return protocol.Location{URI: pathToURI(file), Range: protocol.Range{Start: target, End: target}}
}

func resolveSymbol(expr ast.Expression, info *analyser.Info) *analyser.Symbol {
	switch e := expr.(type) {
	case *ast.IdentLiteral:
		return info.Idents[e]
	case *ast.MemberExpr:
		if mt, ok := info.Types[e.Object].(*analyser.ModuleType); ok {
			return mt.Exports[e.Field]
		}
	}
	return nil
}

func symbolLocation(sym *analyser.Symbol) *protocol.Location {
	if sym == nil || sym.File == "" {
		return nil
	}
	loc := pointLocation(sym.File, sym.Line, sym.Col)
	return &loc
}
