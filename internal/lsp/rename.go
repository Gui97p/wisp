package lsp

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var identPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type site struct{ line, col int }

func allSymbols(info *analyser.Info) map[*analyser.Symbol]bool {
	set := map[*analyser.Symbol]bool{}
	for _, sym := range info.Idents {
		set[sym] = true
	}
	for _, syms := range info.VarSymbols {
		for _, sym := range syms {
			set[sym] = true
		}
	}
	for _, scope := range info.Scopes {
		for _, sym := range scope.Visible() {
			set[sym] = true
		}
	}
	return set
}

func declSite(sym *analyser.Symbol, info *analyser.Info, src []byte) (site, bool) {
	for node, syms := range info.VarSymbols {
		for i, s := range syms {
			if s != sym {
				continue
			}
			var params []ast.Param
			switch n := node.(type) {
			case *ast.VarStmt:
				params = n.Vars
			case *ast.ConstDecl:
				params = n.Vars
			case *ast.VarDecl:
				params = n.Vars
			case *ast.ConstStmt:
				params = n.Vars
			case *ast.FuncDecl:
				params = n.Params
				if n.Receiver != nil {
					if i == 0 && n.Receiver.Name == sym.Name {
						return site{n.Receiver.Line, n.Receiver.Col}, true
					}
					i--
				}
			}
			if i >= 0 && i < len(params) && params[i].Name == sym.Name && params[i].Line > 0 {
				return site{params[i].Line, params[i].Col}, true
			}
		}
	}

	lines := strings.Split(string(src), "\n")
	for l := sym.Line - 1; l >= 0 && l < len(lines) && l < sym.Line+3; l++ {
		start := 0
		if l == sym.Line-1 {
			start = byteOffset(lines[l], max(sym.Col-1, 0))
		}
		if start > len(lines[l]) {
			continue
		}
		if idx := wordIndex(lines[l][start:], sym.Name); idx >= 0 {
			return site{l + 1, columnOf(lines[l], start+idx) + 1}, true
		}
	}
	return site{}, false
}

func byteOffset(line string, column int) int {
	units := 0
	for offset, r := range line {
		if units >= column {
			return offset
		}
		units += utf16Units(r)
	}
	return len(line)
}

func columnOf(line string, offset int) int {
	units := 0
	for position, r := range line {
		if position >= offset {
			break
		}
		units += utf16Units(r)
	}
	return units
}

func utf16Units(r rune) int {
	if r >= 0x10000 {
		return 2
	}
	return 1
}

func wordIndex(text, word string) int {
	from := 0
	for {
		idx := strings.Index(text[from:], word)
		if idx < 0 {
			return -1
		}
		idx += from
		end := idx + len(word)
		before := idx == 0 || !isIdentByte(text[idx-1])
		after := end >= len(text) || !isIdentByte(text[end])
		if before && after {
			return idx
		}
		from = idx + 1
	}
}

func symbolAt(program *ast.Program, info *analyser.Info, src []byte, line, col int) *analyser.Symbol {
	if expr := exprInProgram(program, line, col); expr != nil {
		if id, ok := expr.(*ast.IdentLiteral); ok {
			if sym := info.Idents[id]; sym != nil {
				return sym
			}
		}
	}
	for sym := range allSymbols(info) {
		s, ok := declSite(sym, info, src)
		if ok && s.line == line && col >= s.col && col < s.col+len(sym.Name) {
			return sym
		}
	}
	return nil
}

func renamable(path string, sym *analyser.Symbol, program *ast.Program) error {
	if sym.File != path {
		return fmt.Errorf("cannot rename %s: it is declared in another file or is built in", sym.Name)
	}
	for _, d := range program.Declarations {
		switch decl := d.(type) {
		case *ast.FuncDecl:
			if decl.Name == sym.Name && decl.Exported {
				return fmt.Errorf("cannot rename exported %s: other modules may use it", sym.Name)
			}
		case *ast.StructDecl:
			if decl.Name == sym.Name && decl.Exported {
				return fmt.Errorf("cannot rename exported %s: other modules may use it", sym.Name)
			}
		case *ast.TypeDecl:
			if decl.Name == sym.Name && decl.Exported {
				return fmt.Errorf("cannot rename exported %s: other modules may use it", sym.Name)
			}
		case *ast.VarDecl:
			for _, v := range decl.Vars {
				if v.Name == sym.Name && decl.Exported {
					return fmt.Errorf("cannot rename exported %s: other modules may use it", sym.Name)
				}
			}
		}
	}
	if sym.Kind == analyser.STRUCT || sym.Kind == analyser.TYPE || sym.Kind == analyser.MODULE {
		return fmt.Errorf("renaming %s %s is not supported yet", sym.Kind, sym.Name)
	}
	return nil
}

func singleFileModule(path string) bool {
	modules, err := buildProjectGraph(resolveProjectEntry(path), fileBuffer)
	if err != nil {
		return false
	}
	for _, mod := range modules {
		for _, f := range mod.FilePaths {
			if f == path {
				return len(mod.FilePaths) == 1
			}
		}
	}
	return false
}

func prepareRename(uri protocol.DocumentUri, pos protocol.Position) *protocol.Range {
	path := uriToPath(uri)
	program, info, _ := analyseEntryChecked(path)
	if program == nil || info == nil {
		return nil
	}
	line, col := int(pos.Line)+1, int(pos.Character)+1
	sym := symbolAt(program, info, fileBuffer[path], line, col)
	if sym == nil || renamable(path, sym, program) != nil {
		return nil
	}
	if expr := exprInProgram(program, line, col); expr != nil {
		if id, ok := expr.(*ast.IdentLiteral); ok && info.Idents[id] == sym {
			sl, sc := id.Position()
			r := rangeOf(sl, sc, sl, sc+len(sym.Name)-1)
			return &r
		}
	}
	s, _ := declSite(sym, info, fileBuffer[path])
	r := rangeOf(s.line, s.col, s.line, s.col+len(sym.Name)-1)
	return &r
}

func rename(uri protocol.DocumentUri, pos protocol.Position, newName string) (*protocol.WorkspaceEdit, error) {
	path := uriToPath(uri)
	if !identPattern.MatchString(newName) {
		return nil, fmt.Errorf("%q is not a valid identifier", newName)
	}
	for _, k := range keywords {
		if k == newName {
			return nil, fmt.Errorf("%q is a keyword", newName)
		}
	}

	program, info, clean := analyseEntryChecked(path)
	if program == nil || info == nil || !clean {
		return nil, fmt.Errorf("fix the syntax errors before renaming")
	}

	src := fileBuffer[path]
	sym := symbolAt(program, info, src, int(pos.Line)+1, int(pos.Character)+1)
	if sym == nil {
		return nil, fmt.Errorf("nothing to rename here")
	}
	if err := renamable(path, sym, program); err != nil {
		return nil, err
	}
	if (sym.Kind == analyser.FUNC || sym.Global) && !singleFileModule(path) {
		return nil, fmt.Errorf("cannot rename %s: the module has several files", sym.Name)
	}
	if existing, ok := scopeForSymbol(info, sym).Resolve(newName); ok && existing != sym {
		return nil, fmt.Errorf("%s is already declared in this scope", newName)
	}

	var edits []protocol.TextEdit
	add := func(line, col int) {
		r := rangeOf(line, col, line, col+len(sym.Name)-1)
		edits = append(edits, protocol.TextEdit{Range: r, NewText: newName})
	}

	if s, ok := declSite(sym, info, src); ok {
		add(s.line, s.col)
	}
	for id, s := range info.Idents {
		if s == sym {
			l, c := id.Position()
			add(l, c)
		}
	}

	return &protocol.WorkspaceEdit{Changes: map[protocol.DocumentUri][]protocol.TextEdit{uri: edits}}, nil
}

func scopeForSymbol(info *analyser.Info, sym *analyser.Symbol) *analyser.Scope {
	for _, scope := range info.Scopes {
		if s, ok := scope.Resolve(sym.Name); ok && s == sym {
			return scope
		}
	}
	return analyser.NewScope(nil)
}
