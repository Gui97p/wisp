package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/parser"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func typedWord(prefix string) string {
	end := len(prefix)
	start := end
	for start > 0 && isIdentByte(prefix[start-1]) {
		start--
	}
	return prefix[start:end]
}

func matchesWord(name, word string) bool {
	return strings.HasPrefix(strings.ToLower(name), strings.ToLower(word))
}

func shortcutItems(path string, scope *analyser.Scope, word, suffix string) []protocol.CompletionItem {
	if word == "" {
		return nil
	}
	var items []protocol.CompletionItem
	items = append(items, receiverMembers(scope, word, suffix)...)
	if len(word) >= 2 {
		items = append(items, importedMembers(scope, word, suffix)...)
		items = append(items, unimportedMembers(path, scope, word, suffix)...)
	}
	return items
}

func receiverMembers(scope *analyser.Scope, word, suffix string) []protocol.CompletionItem {
	var items []protocol.CompletionItem
	for _, sym := range scope.Visible() {
		if sym.Kind != analyser.PARAM || sym.Name == "" || strings.HasPrefix(sym.Name, "_") {
			continue
		}
		tp := sym.Type
		if pt, ok := tp.(analyser.PointerType); ok {
			tp = pt.Element
		}

		if st := nativeStructOf(tp); st != nil {
			for _, field := range st.Order {
				if !matchesWord(field, word) {
					continue
				}
				it := item(sym.Name+"."+field, protocol.CompletionItemKindField, analyser.TypeText(st.Fields[field]))
				it.FilterText = ptr(field)
				it.SortText = ptr("8" + field)
				items = append(items, it)
			}
		}

		methods := analyser.MethodsOf(tp)
		names := make([]string, 0, len(methods))
		for name := range methods {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if !matchesWord(name, word) {
				continue
			}
			ft := methods[name]
			it := item(sym.Name+"."+name, protocol.CompletionItemKindMethod, analyser.FuncSignature(name, ft, nil))
			it.FilterText = ptr(name)
			it.SortText = ptr("8" + name)
			items = append(items, callable(it, len(ft.Params), suffix))
		}
	}
	return items
}

func importedMembers(scope *analyser.Scope, word, suffix string) []protocol.CompletionItem {
	var items []protocol.CompletionItem
	for _, sym := range scope.Visible() {
		mt, ok := sym.Type.(*analyser.ModuleType)
		if sym.Kind != analyser.MODULE || !ok || sym.Name == "" {
			continue
		}
		names := make([]string, 0, len(mt.Exports))
		for name := range mt.Exports {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if !matchesWord(name, word) {
				continue
			}
			qualified := *mt.Exports[name]
			qualified.Name = sym.Name + "." + name
			it := symbolItem(&qualified, suffix)
			it.FilterText = ptr(name)
			it.SortText = ptr("9" + name)
			items = append(items, it)
		}
	}
	return items
}

type exportEntry struct {
	name     string
	kind     protocol.CompletionItemKind
	detail   string
	callable bool
	params   int
}

type exportCacheEntry struct {
	modified time.Time
	size     int64
	entries  []exportEntry
}

var (
	exportCache   = map[string]exportCacheEntry{}
	exportCacheMu sync.Mutex
)

func unimportedMembers(path string, scope *analyser.Scope, word, suffix string) []protocol.CompletionItem {
	root := module.FindProjectRoot(filepath.Dir(path))
	imported := importedPaths(path)
	cyclic := dependentFiles(path)

	byPath := map[string]string{}
	candidates := module.ImportablePaths(root)
	for _, c := range candidates {
		byPath[c.Path] = c.File
	}

	var items []protocol.CompletionItem
	for _, c := range candidates {
		alias := moduleAlias(c.Path)
		if c.File == path || imported[c.Path] || cyclic[c.File] || alias == "" {
			continue
		}
		if _, taken := scope.Resolve(alias); taken {
			continue
		}

		edit, ok := importEdit(path, c.Path)
		if !ok {
			continue
		}
		for _, e := range moduleExports(c.File, byPath, map[string]bool{}, 0) {
			if !matchesWord(e.name, word) {
				continue
			}
			it := item(alias+"."+e.name, e.kind, "auto import "+c.Path+"  "+e.detail)
			it.FilterText = ptr(e.name)
			it.SortText = ptr("z" + e.name)
			it.AdditionalTextEdits = []protocol.TextEdit{edit}
			if e.callable {
				it = callable(it, e.params, suffix)
			}
			items = append(items, it)
		}
	}
	return items
}

func moduleAlias(importPath string) string {
	return importPath[strings.LastIndex(importPath, "/")+1:]
}

func importedPaths(path string) map[string]bool {
	imported := map[string]bool{}
	if src, ok := fileBuffer[path]; ok {
		for _, im := range importLinePattern.FindAllStringSubmatch(string(src), -1) {
			imported[im[1]] = true
		}
	}
	return imported
}

func importEdit(path, importPath string) (protocol.TextEdit, bool) {
	src, ok := fileBuffer[path]
	if !ok {
		return protocol.TextEdit{}, false
	}
	lines := splitLines(src)

	last := -1
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "import ") {
			last = i
		}
	}

	line := fmt.Sprintf("import %q;\n", importPath)
	at := uint32(last + 1)
	if last < 0 {
		first := 0
		inBlock := false
		for i, l := range lines {
			t := strings.TrimSpace(l)
			if inBlock {
				if strings.Contains(t, "*/") {
					inBlock = false
				}
				first = i + 1
				continue
			}
			if strings.HasPrefix(t, "/*") {
				inBlock = !strings.Contains(t, "*/")
				first = i + 1
				continue
			}
			if strings.HasPrefix(t, "//") || t == "" {
				first = i + 1
				continue
			}
			break
		}
		at = uint32(first)
		line += "\n"
	}

	pos := protocol.Position{Line: at, Character: 0}
	return protocol.TextEdit{Range: protocol.Range{Start: pos, End: pos}, NewText: line}, true
}

func moduleExports(file string, byPath map[string]string, seen map[string]bool, depth int) []exportEntry {
	if depth > 4 || seen[file] {
		return nil
	}
	seen[file] = true

	var src []byte
	var modified time.Time
	var size int64
	if open, ok := fileBuffer[file]; ok {
		src = open
	} else {
		info, err := os.Stat(file)
		if err != nil {
			return nil
		}
		modified, size = info.ModTime(), info.Size()

		exportCacheMu.Lock()
		cached, ok := exportCache[file]
		exportCacheMu.Unlock()
		if ok && cached.modified.Equal(modified) && cached.size == size {
			return cached.entries
		}

		data, err := os.ReadFile(file)
		if err != nil {
			return nil
		}
		src = data
	}

	program := parser.NewParser(lexer.NewLexer(src)).ParseProgram()

	var entries []exportEntry
	imports := map[string]string{}
	for _, d := range program.Declarations {
		if imp, ok := d.(*ast.ImportDecl); ok {
			imports[imp.Alias] = imp.Path
		}
	}

	for _, d := range program.Declarations {
		switch decl := d.(type) {
		case *ast.FuncDecl:
			if decl.Exported && decl.Receiver == nil {
				entries = append(entries, exportEntry{name: decl.Name, kind: protocol.CompletionItemKindFunction, detail: funcDeclText(decl), callable: true, params: len(decl.Params)})
			}
		case *ast.StructDecl:
			if decl.Exported {
				entries = append(entries, exportEntry{name: decl.Name, kind: protocol.CompletionItemKindStruct, detail: "struct"})
			}
		case *ast.TypeDecl:
			if decl.Exported {
				kind, detail := protocol.CompletionItemKindClass, "type"
				if decl.IsEnum {
					kind, detail = protocol.CompletionItemKindEnum, "enum"
				}
				entries = append(entries, exportEntry{name: decl.Name, kind: kind, detail: detail})
			}
		case *ast.ConstDecl:
			if decl.Exported {
				for _, v := range decl.Vars {
					entries = append(entries, exportEntry{name: v.Name, kind: protocol.CompletionItemKindConstant, detail: "const"})
				}
			}
		case *ast.VarDecl:
			if decl.Exported {
				for _, v := range decl.Vars {
					entries = append(entries, exportEntry{name: v.Name, kind: protocol.CompletionItemKindVariable, detail: "var"})
				}
			}
		case *ast.ReexportDecl:
			if decl.Alias != "" {
				continue
			}
			if importPath, ok := imports[decl.Name]; ok {
				if target, found := byPath[importPath]; found {
					entries = append(entries, moduleExports(target, byPath, seen, depth+1)...)
				}
			}
		}
	}

	if _, open := fileBuffer[file]; !open && depth == 0 {
		exportCacheMu.Lock()
		exportCache[file] = exportCacheEntry{modified: modified, size: size, entries: entries}
		exportCacheMu.Unlock()
	}
	return entries
}

func funcDeclText(decl *ast.FuncDecl) string {
	params := make([]string, 0, len(decl.Params))
	for _, p := range decl.Params {
		switch {
		case p.Any:
			params = append(params, "..."+p.Name)
		case p.Variadic:
			params = append(params, refText(p.Type)+" ..."+p.Name)
		default:
			params = append(params, refText(p.Type)+" "+p.Name)
		}
	}
	text := "func " + decl.Name + "(" + strings.Join(params, ", ") + ")"
	if decl.Fallible {
		text += " !"
	}
	if len(decl.ReturnTypes) > 0 {
		rets := make([]string, len(decl.ReturnTypes))
		for i, r := range decl.ReturnTypes {
			rets[i] = refText(r)
		}
		text += " " + strings.Join(rets, ", ")
	}
	return text
}
