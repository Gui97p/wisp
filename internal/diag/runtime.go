package diag

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var StdRoot func() string

func DisplayPath(file string) string {
	if file == "" {
		return file
	}
	if StdRoot != nil {
		if root := StdRoot(); root != "" {
			if rel, err := filepath.Rel(root, file); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return "std/" + filepath.ToSlash(rel)
			}
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(cwd, file); err == nil {
			return rel
		}
	}
	return file
}

func RuntimeBlock(header, file string, source []byte, line, col, endLine, endCol int, label string) string {
	lines := splitLines(source)
	if line-1 < 0 || line-1 >= len(lines) {
		return ""
	}

	width := len(strconv.Itoa(line))
	gutter := strings.Repeat(" ", width)

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n--> %s:%d:%d\n%s |\n", header, DisplayPath(file), line, col, gutter)
	if label != "" {
		label = " " + label
	}
	writeSnippet(&b, lines, width, line, col, endLine, endCol, "", label, palette{})
	return b.String()
}

func FrameBlock(file string, source []byte, line, col, endLine, endCol int, function string, snippets bool) string {
	head := fmt.Sprintf("%s:%d:%d: in %s\n", DisplayPath(file), line, col, function)
	if !snippets {
		return head
	}
	if source == nil {
		return head
	}
	text, pad, carets, ok := underline(splitLines(source), line, col, endLine, endCol)
	if !ok {
		return head
	}
	return head + "    " + text + "\n    " + pad + carets + "\n"
}
