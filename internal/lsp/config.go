package lsp

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/target"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	configHeader = regexp.MustCompile(`^\s*\[\[?\s*([^\]]*?)\s*\]`)
	configKey    = regexp.MustCompile(`^(\s*)([A-Za-z0-9_-]+)\s*=`)
	configWord   = regexp.MustCompile(`^\s*[A-Za-z0-9_-]*$`)
)

func isConfigURI(uri protocol.DocumentUri) bool {
	return filepath.Base(uriToPath(uri)) == "wisp.toml"
}

func configLines(path string) ([]string, bool) {
	src, ok := fileBuffer[path]
	if !ok {
		return nil, false
	}
	return strings.Split(strings.ReplaceAll(string(src), "\r", ""), "\n"), true
}

func configSection(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	if parts[0] == "targets" {
		if len(parts) >= 3 && parts[2] == "output" {
			return "targets.output"
		}
		return "targets"
	}
	return parts[0]
}

func headerParts(header string) []string {
	parts := strings.Split(header, ".")
	for i, p := range parts {
		parts[i] = strings.Trim(strings.TrimSpace(p), `"`)
	}
	if len(parts) == 1 && parts[0] == "" {
		return nil
	}
	return parts
}

type configContext struct {
	section  string
	name     string
	present  map[string]bool
	declared []string
}

func configContextAt(lines []string, line int) configContext {
	ctx := configContext{present: map[string]bool{}}
	seen := map[string]bool{}
	current := ""
	for i, l := range lines {
		if m := configHeader.FindStringSubmatch(l); m != nil {
			parts := headerParts(m[1])
			if len(parts) == 2 && parts[0] == "targets" && !seen[parts[1]] {
				seen[parts[1]] = true
				ctx.declared = append(ctx.declared, parts[1])
			}
			if i < line {
				current = configSection(parts)
				ctx.section = current
				ctx.name = ""
				if len(parts) >= 2 && parts[0] == "targets" {
					ctx.name = parts[1]
				}
				ctx.present = map[string]bool{}
			}
			continue
		}
		if m := configKey.FindStringSubmatch(l); m != nil && i != line {
			if sectionOfLine(lines, i) == ctx.section {
				ctx.present[m[2]] = true
			}
		}
	}
	return ctx
}

func sectionOfLine(lines []string, line int) string {
	section := ""
	for i := 0; i < line && i < len(lines); i++ {
		if m := configHeader.FindStringSubmatch(lines[i]); m != nil {
			section = configSection(headerParts(m[1]))
		}
	}
	return section
}

func valueChoices(section, key string, declared []string) []string {
	switch {
	case section == "" && key == "target":
		names := append([]string{}, target.Names()...)
		for _, d := range declared {
			names = append(names, d)
		}
		sort.Strings(names)
		out := names[:0]
		for i, n := range names {
			if i == 0 || n != names[i-1] {
				out = append(out, n)
			}
		}
		return out
	case section == "targets" && key == "arch":
		return []string{"x64"}
	case section == "targets" && key == "abi":
		return []string{"sysv"}
	case section == "targets" && key == "format":
		return []string{"elf64"}
	}
	return nil
}

func configCompletion(uri protocol.DocumentUri, pos protocol.Position) []protocol.CompletionItem {
	path := uriToPath(uri)
	lines, ok := configLines(path)
	if !ok || int(pos.Line) >= len(lines) {
		return nil
	}

	runes := []rune(lines[pos.Line])
	col := int(pos.Character)
	if col > len(runes) {
		col = len(runes)
	}
	prefix := string(runes[:col])
	ctx := configContextAt(lines, int(pos.Line))

	trimmed := strings.TrimLeft(prefix, " \t")
	if strings.HasPrefix(trimmed, "[") && !strings.Contains(trimmed, "]") {
		return headerItems(ctx.declared, pos, prefix)
	}

	if m := configKey.FindStringSubmatch(prefix); m != nil {
		return valueItems(ctx, m[2], prefix)
	}

	if configWord.MatchString(prefix) {
		return keyItems(ctx, pos, prefix)
	}

	return nil
}

func headerItems(declared []string, pos protocol.Position, prefix string) []protocol.CompletionItem {
	start := strings.Index(prefix, "[")
	for start < len(prefix) && (prefix[start] == '[' || prefix[start] == ' ') {
		start++
	}
	rng := protocol.Range{
		Start: protocol.Position{Line: pos.Line, Character: uint32(utf16Len(prefix[:start]))},
		End:   pos,
	}

	kind := protocol.CompletionItemKindModule
	items := []protocol.CompletionItem{}
	add := func(label, snippet, detail string) {
		item := protocol.CompletionItem{
			Label:    label,
			Kind:     &kind,
			Detail:   &detail,
			TextEdit: protocol.TextEdit{Range: rng, NewText: snippet},
		}
		if strings.Contains(snippet, "$") {
			format := protocol.InsertTextFormatSnippet
			item.InsertTextFormat = &format
		}
		items = append(items, item)
	}

	add("output", "output", "table")
	add("targets.<name>", "targets.${1:name}", "custom target")

	names := append([]string{}, declared...)
	for _, n := range target.Names() {
		if target.IsBuiltin(n) && !slices.Contains(names, n) {
			names = append(names, n)
		}
	}
	for _, n := range names {
		detail := "custom target"
		if target.IsBuiltin(n) {
			detail = "built-in target"
		}
		add("targets."+n, "targets."+n, detail)
		add("targets."+n+".output", "targets."+n+".output", "output of "+n)
	}
	return items
}

func keyItems(ctx configContext, pos protocol.Position, prefix string) []protocol.CompletionItem {
	items := []protocol.CompletionItem{}
	kind := protocol.CompletionItemKindProperty
	snippet := protocol.InsertTextFormatSnippet
	word := strings.TrimLeft(prefix, " \t")
	start := uint32(utf16Len(prefix[:len(prefix)-len(word)]))
	rng := protocol.Range{Start: protocol.Position{Line: pos.Line, Character: start}, End: pos}

	builtin := ctx.section == "targets" && target.IsBuiltin(ctx.name)
	builtinLua := false
	if builtin {
		if t, err := target.Lookup(ctx.name); err == nil {
			builtinLua = t.Arch != "x64"
		}
	}

	for _, f := range module.ConfigFields(ctx.section) {
		if f.Table || ctx.present[f.Key] {
			continue
		}
		if builtin && (f.Key == "arch" || builtinLua) {
			continue
		}
		var text string
		switch f.Type {
		case "bool":
			text = f.Key + " = ${1|true,false|}"
		case "array of strings":
			text = f.Key + ` = ["$1"]`
		default:
			text = f.Key + ` = "$1"`
		}
		detail := f.Type
		doc := protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: f.Doc}
		items = append(items, protocol.CompletionItem{
			Label:            f.Key,
			Kind:             &kind,
			Detail:           &detail,
			Documentation:    doc,
			InsertTextFormat: &snippet,
			TextEdit:         protocol.TextEdit{Range: rng, NewText: text},
		})
	}
	return items
}

func valueItems(ctx configContext, key, prefix string) []protocol.CompletionItem {
	kind := protocol.CompletionItemKindValue
	items := []protocol.CompletionItem{}

	field, ok := module.ConfigFieldNamed(ctx.section, key)
	if !ok {
		return items
	}

	if field.Type == "bool" {
		for _, v := range []string{"true", "false"} {
			items = append(items, protocol.CompletionItem{Label: v, Kind: &kind})
		}
		return items
	}

	inQuote := strings.Count(prefix[strings.Index(prefix, "="):], `"`)%2 == 1
	for _, v := range valueChoices(ctx.section, key, ctx.declared) {
		text := `"` + v + `"`
		if inQuote {
			text = v
		}
		items = append(items, protocol.CompletionItem{Label: v, Kind: &kind, InsertText: &text})
	}
	return items
}

func configHover(uri protocol.DocumentUri, pos protocol.Position) *protocol.Hover {
	path := uriToPath(uri)
	lines, ok := configLines(path)
	if !ok || int(pos.Line) >= len(lines) {
		return nil
	}

	line := lines[pos.Line]
	runes := []rune(line)
	col := int(pos.Character)
	if col > len(runes) {
		col = len(runes)
	}

	result := func(text string, from, to int) *protocol.Hover {
		return &protocol.Hover{
			Contents: protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: text},
			Range: &protocol.Range{
				Start: protocol.Position{Line: pos.Line, Character: uint32(from)},
				End:   protocol.Position{Line: pos.Line, Character: uint32(to)},
			},
		}
	}

	wordAt := func() (string, int, int) {
		isWord := func(r rune) bool {
			return r == '_' || r == '-' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
		}
		start, end := col, col
		for start > 0 && isWord(runes[start-1]) {
			start--
		}
		for end < len(runes) && isWord(runes[end]) {
			end++
		}
		return string(runes[start:end]), start, end
	}

	if m := configHeader.FindStringSubmatch(line); m != nil {
		word, start, end := wordAt()
		if word == "" {
			return nil
		}
		parts := headerParts(m[1])
		if len(parts) >= 3 && parts[0] == "targets" && parts[2] == "output" && word == "output" {
			if f, ok := module.ConfigFieldNamed("targets", "output"); ok {
				return result(fmt.Sprintf("**[targets.%s.output]**\n\n%s", parts[1], f.Doc), utf16Len(string(runes[:start])), utf16Len(string(runes[:end])))
			}
		}
		if len(parts) > 0 && word == parts[0] {
			if f, ok := module.ConfigFieldNamed("", word); ok && f.Table {
				return result(fmt.Sprintf("**[%s]**\n\n%s", word, f.Doc), utf16Len(string(runes[:start])), utf16Len(string(runes[:end])))
			}
		}
		return nil
	}

	m := configKey.FindStringSubmatch(line)
	if m == nil {
		return nil
	}

	section := sectionOfLine(lines, int(pos.Line))
	keyStart := len([]rune(m[1]))
	keyEnd := keyStart + len([]rune(m[2]))

	if col >= keyStart && col <= keyEnd {
		f, ok := module.ConfigFieldNamed(section, m[2])
		if !ok {
			return nil
		}
		text := fmt.Sprintf("**%s** (%s)\n\n%s", f.Key, f.Type, f.Doc)
		return result(text, utf16Len(string(runes[:keyStart])), utf16Len(string(runes[:keyEnd])))
	}

	if section == "" && m[2] == "target" {
		word, start, end := wordAt()
		if t, err := target.Lookup(word); err == nil && word != "" {
			text := fmt.Sprintf("**%s**\n\narch: `%s`", t.Name, t.Arch)
			if t.OS != "" {
				text += fmt.Sprintf(", os: `%s`", t.OS)
			}
			if t.Builtin {
				text += "\n\nBuilt-in target."
			} else {
				text += "\n\nCustom target."
			}
			return result(text, utf16Len(string(runes[:start])), utf16Len(string(runes[:end])))
		}
	}
	return nil
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n += utf16Units(r)
	}
	return n
}
