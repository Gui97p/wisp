package diag

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

const (
	reset  = "\x1b[0m"
	bold   = "\x1b[1m"
	red    = "\x1b[1;31m"
	yellow = "\x1b[1;33m"
	cyan   = "\x1b[1;36m"
	green  = "\x1b[1;32m"
)

func RenderGrouped(w io.Writer, buffers map[string][]byte, diags List) {
	var order []string
	grouped := map[string]List{}

	for _, d := range diags {
		if _, ok := grouped[d.File]; !ok {
			order = append(order, d.File)
		}
		grouped[d.File] = append(grouped[d.File], d)
	}

	for _, file := range order {
		list := grouped[file]
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].Line != list[j].Line {
				return list[i].Line < list[j].Line
			}
			return list[i].Col < list[j].Col
		})
		Render(w, file, buffers[file], list, buffers)
	}
}

func RenderError(w io.Writer, buffers map[string][]byte, e *Error) {
	Render(w, e.File, buffers[e.File], List{e.Diagnostic}, buffers)
}

func Render(w io.Writer, filename string, source []byte, diags List, others ...map[string][]byte) {
	lines := splitLines(source)

	var buffers map[string][]byte
	if len(others) > 0 {
		buffers = others[0]
	}

	for _, d := range diags {
		width := len(strconv.Itoa(d.Line))
		for _, n := range d.Notes {
			width = max(width, len(strconv.Itoa(n.Line)))
		}
		gutter := strings.Repeat(" ", width)

		label, color := "error", red
		if d.Severity == SeverityWarning {
			label, color = "warning", yellow
		}

		if d.Origin != "" {
			label += "[" + d.Origin + "]"
		}

		fmt.Fprintf(w, ">> %s%s:%s %s\n", color, label, reset, d.Message)
		fmt.Fprintf(w, "%s--> %s%s:%d:%d\n", cyan, reset, filename, d.Line, d.Col)
		fmt.Fprintf(w, "%s%s |%s\n", cyan, gutter, reset)
		writeSnippet(w, lines, width, d.Line, d.Col, d.EndLine, d.EndCol, color, "")

		for _, n := range d.Notes {
			noteLines := lines
			noteFile := n.File
			if noteFile == "" {
				noteFile = filename
			}
			if noteFile != filename && buffers != nil {
				noteLines = splitLines(buffers[noteFile])
			}
			fmt.Fprintf(w, "%s%s |%s\n", cyan, gutter, reset)
			fmt.Fprintf(w, "%s%s = note:%s %s\n", cyan, gutter, reset, n.Message)
			fmt.Fprintf(w, "%s%s --> %s%s:%d:%d\n", cyan, gutter, reset, noteFile, n.Line, n.Col)
			fmt.Fprintf(w, "%s%s |%s\n", cyan, gutter, reset)
			writeSnippet(w, noteLines, width, n.Line, n.Col, n.EndLine, n.EndCol, green, "")
		}

		if d.Help != "" {
			fmt.Fprintf(w, "%s%s |%s\n", cyan, gutter, reset)
			fmt.Fprintf(w, "%s%s = help:%s %s\n", cyan, gutter, reset, d.Help)
		}
		fmt.Fprintln(w)
	}
}

func splitLines(source []byte) []string {
	return strings.Split(strings.ReplaceAll(string(source), "\r\n", "\n"), "\n")
}

func writeSnippet(w io.Writer, lines []string, width, line, col, endLine, endCol int, color, label string) {
	if line-1 < 0 || line-1 >= len(lines) {
		return
	}

	text := lines[line-1]
	runes := []rune(text)
	start := max(col-1, 0)
	if start > len(runes) {
		start = len(runes)
	}

	last := start
	switch {
	case endLine == line && endCol >= col:
		last = endCol - 1
	case endLine > line:
		last = len(runes) - 1
	default:
		last = start
	}
	last = min(max(last, start), max(len(runes)-1, start))

	var pad strings.Builder
	for _, r := range runes[:start] {
		if r == '\t' {
			pad.WriteRune('\t')
		} else {
			pad.WriteRune(' ')
		}
	}
	carets := strings.Repeat("^", last-start+1)

	num := strconv.Itoa(line)
	fmt.Fprintf(w, "%s%*s |%s %s\n", cyan, width, num, reset, text)
	fmt.Fprintf(w, "%s%s |%s %s%s%s%s%s\n", cyan, strings.Repeat(" ", width), reset, pad.String(), color, carets, reset, label)
}
