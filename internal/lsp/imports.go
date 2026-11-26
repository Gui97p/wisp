package lsp

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/module"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	importPrefixPattern = regexp.MustCompile(`^\s*import\s+"([^"]*)$`)
	importLinePattern   = regexp.MustCompile(`(?m)^\s*import\s+"([^"]*)"`)
)

type importSegment struct {
	file     string
	module   bool
	children int
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func importItems(path string, pos protocol.Position) ([]protocol.CompletionItem, bool) {
	prefix := linePrefix(path, int(pos.Line), int(pos.Character))
	m := importPrefixPattern.FindStringSubmatch(prefix)
	if m == nil {
		return nil, false
	}

	dir, partial := "", m[1]
	if i := strings.LastIndex(m[1], "/"); i >= 0 {
		dir, partial = m[1][:i+1], m[1][i+1:]
	}

	start := pos.Character - uint32(utf16Len(partial))
	end := pos.Character
	suffix := lineSuffix(path, int(pos.Line), int(pos.Character))
	if before, _, ok := strings.Cut(suffix, `"`); ok {
		end += uint32(utf16Len(before))
	}
	rng := protocol.Range{
		Start: protocol.Position{Line: pos.Line, Character: start},
		End:   protocol.Position{Line: pos.Line, Character: end},
	}

	imported := map[string]bool{}
	if src, ok := fileBuffer[path]; ok {
		for _, im := range importLinePattern.FindAllStringSubmatch(string(src), -1) {
			imported[im[1]] = true
		}
	}

	root := module.FindProjectRoot(filepath.Dir(path))

	segments := map[string]*importSegment{}
	candidates := module.ImportablePaths(root)
	byPath := pathIndex(candidates)
	for _, c := range candidates {
		if c.Contains(path) || imported[c.Path] || !strings.HasPrefix(c.Path, dir) || exportsNothing(path, c, byPath) {
			continue
		}

		rest := strings.TrimPrefix(c.Path, dir)
		name, deeper, nested := strings.Cut(rest, "/")
		if name == "" || (nested && deeper == "") {
			continue
		}

		seg := segments[name]
		if seg == nil {
			seg = &importSegment{}
			segments[name] = seg
		}
		if nested {
			seg.children++
		} else {
			seg.module, seg.file = true, c.File
		}
	}

	names := make([]string, 0, len(segments))
	for name := range segments {
		names = append(names, name)
	}
	sort.Strings(names)

	items := []protocol.CompletionItem{}
	for _, name := range names {
		seg := segments[name]

		order := "0"
		if dir == "" && name == "std" {
			order = "1"
		}

		item := protocol.CompletionItem{FilterText: &name, SortText: ptr(order + name)}
		if seg.module {
			kind := protocol.CompletionItemKindModule
			detail := seg.file
			if rel, err := filepath.Rel(root, seg.file); err == nil && !strings.HasPrefix(rel, "..") {
				detail = filepath.ToSlash(rel)
			}
			if seg.children > 0 {
				detail += " · +" + plural(seg.children, "submodule")
			}
			item.Label, item.Kind, item.Detail = name, &kind, &detail
			item.TextEdit = protocol.TextEdit{Range: rng, NewText: name}
		} else {
			kind := protocol.CompletionItemKindFolder
			detail := plural(seg.children, "module")
			item.Label, item.Kind, item.Detail = name+"/", &kind, &detail
			item.TextEdit = protocol.TextEdit{Range: rng, NewText: name + "/"}
			item.Command = &protocol.Command{Title: "Suggest", Command: "editor.action.triggerSuggest"}
		}
		items = append(items, item)
	}
	return items, true
}

func ptr[T any](v T) *T { return &v }

func importHover(path string, pos protocol.Position) *protocol.Hover {
	lines, ok := bufferLines(path)
	if !ok || int(pos.Line) >= len(lines) {
		return nil
	}

	line := lines[pos.Line]
	loc := importLinePattern.FindStringSubmatchIndex(line)
	if loc == nil {
		return nil
	}

	start := utf16Len(line[:loc[2]])
	end := utf16Len(line[:loc[3]])
	if int(pos.Character) < start || int(pos.Character) > end {
		return nil
	}

	importPath := line[loc[2]:loc[3]]
	root := module.FindProjectRoot(filepath.Dir(path))

	text := ""
	if files, err := module.ResolveImport(root, importPath); err != nil {
		text = err.Error()
	} else {
		file := files[0]
		if rel, err := filepath.Rel(root, file); err == nil && !strings.HasPrefix(rel, "..") {
			file = filepath.ToSlash(rel)
		}
		text = renderDoc("import \""+importPath+"\"", "Module file: `"+file+"`")
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: text},
		Range: &protocol.Range{
			Start: protocol.Position{Line: pos.Line, Character: uint32(start)},
			End:   protocol.Position{Line: pos.Line, Character: uint32(end)},
		},
	}
}
