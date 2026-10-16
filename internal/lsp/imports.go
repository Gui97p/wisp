package lsp

import (
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
		if c.File == path || imported[c.Path] {
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
