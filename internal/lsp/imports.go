package lsp

import (
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/module"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	importPrefixPattern = regexp.MustCompile(`^\s*import\s+"([^"]*)$`)
	importLinePattern   = regexp.MustCompile(`(?m)^\s*import\s+"([^"]*)"`)
)

func importItems(path string, pos protocol.Position) ([]protocol.CompletionItem, bool) {
	prefix := linePrefix(path, int(pos.Line), int(pos.Character))
	m := importPrefixPattern.FindStringSubmatch(prefix)
	if m == nil {
		return nil, false
	}

	start := pos.Character - uint32(utf16Len(m[1]))
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
	cyclic := dependentFiles(path)
	candidates := module.ImportablePaths(root)
	sort.SliceStable(candidates, func(i, j int) bool {
		si, sj := strings.HasPrefix(candidates[i].Path, "std/"), strings.HasPrefix(candidates[j].Path, "std/")
		if si != sj {
			return sj
		}
		return candidates[i].Path < candidates[j].Path
	})

	kind := protocol.CompletionItemKindModule
	items := []protocol.CompletionItem{}
	for _, c := range candidates {
		if c.File == path || imported[c.Path] || cyclic[c.File] {
			continue
		}

		detail := c.File
		if rel, err := filepath.Rel(root, c.File); err == nil && !strings.HasPrefix(rel, "..") {
			detail = filepath.ToSlash(rel)
		}
		items = append(items, protocol.CompletionItem{
			Label:    c.Path,
			Kind:     &kind,
			Detail:   &detail,
			TextEdit: protocol.TextEdit{Range: rng, NewText: c.Path},
		})
	}
	return items, true
}

func importHover(path string, pos protocol.Position) *protocol.Hover {
	lines, ok := configLines(path)
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

func dependentFiles(path string) map[string]bool {
	files := map[string]bool{}

	modules, err := buildProjectGraph(resolveProjectEntry(path), fileBuffer)
	if err != nil {
		return files
	}

	dependents := map[string]bool{}
	for _, mod := range modules {
		if slices.Contains(mod.FilePaths, path) {
			dependents[mod.Path] = true
		}
	}

	for changed := true; changed; {
		changed = false
		for _, mod := range modules {
			if dependents[mod.Path] {
				continue
			}
			for _, imp := range mod.Imports {
				if dependents[imp.Path] {
					dependents[mod.Path] = true
					changed = true
					break
				}
			}
		}
	}

	for _, mod := range modules {
		if dependents[mod.Path] {
			for _, f := range mod.FilePaths {
				files[f] = true
			}
		}
	}
	return files
}
