package diag

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

type palette struct {
	reset, red, yellow, cyan, green string
}

func paletteFor(w io.Writer) palette {
	if os.Getenv("NO_COLOR") != "" {
		return palette{}
	}
	f, ok := w.(*os.File)
	if os.Getenv("FORCE_COLOR") == "" && (!ok || !isTerminal(f)) {
		return palette{}
	}
	return palette{reset: "\x1b[0m", red: "\x1b[1;31m", yellow: "\x1b[1;33m", cyan: "\x1b[1;36m", green: "\x1b[1;32m"}
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func Count(diags List) (errors, warnings int) {
	for _, d := range diags {
		if d.Severity == SeverityWarning {
			warnings++
		} else {
			errors++
		}
	}
	return errors, warnings
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func Summary(w io.Writer, errors, warnings int) {
	p := paletteFor(w)
	switch {
	case errors > 0 && warnings > 0:
		fmt.Fprintf(w, ">> %serror:%s aborting due to %s (and %s)\n", p.red, p.reset, plural(errors, "error"), plural(warnings, "warning"))
	case errors > 0:
		fmt.Fprintf(w, ">> %serror:%s aborting due to %s\n", p.red, p.reset, plural(errors, "error"))
	case warnings > 0:
		fmt.Fprintf(w, ">> %swarning:%s %s emitted\n", p.yellow, p.reset, plural(warnings, "warning"))
	}
}

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
	p := paletteFor(w)

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

		label, color := "error", p.red
		if d.Severity == SeverityWarning {
			label, color = "warning", p.yellow
		}

		if d.Origin != "" {
			label += "[" + d.Origin + "]"
		}

		fmt.Fprintf(w, ">> %s%s:%s %s\n", color, label, p.reset, d.Message)
		fmt.Fprintf(w, "%s--> %s%s:%d:%d\n", p.cyan, p.reset, filename, d.Line, d.Col)
		fmt.Fprintf(w, "%s%s |%s\n", p.cyan, gutter, p.reset)
		writeSnippet(w, lines, width, d.Line, d.Col, d.EndLine, d.EndCol, color, "", p)

		for _, n := range d.Notes {
			noteLines := lines
			noteFile := n.File
			if noteFile == "" {
				noteFile = filename
			}
			if noteFile != filename && buffers != nil {
				noteLines = splitLines(buffers[noteFile])
			}
			fmt.Fprintf(w, "%s%s |%s\n", p.cyan, gutter, p.reset)
			fmt.Fprintf(w, "%s%s = note:%s %s\n", p.cyan, gutter, p.reset, n.Message)
			fmt.Fprintf(w, "%s%s --> %s%s:%d:%d\n", p.cyan, gutter, p.reset, noteFile, n.Line, n.Col)
			fmt.Fprintf(w, "%s%s |%s\n", p.cyan, gutter, p.reset)
			writeSnippet(w, noteLines, width, n.Line, n.Col, n.EndLine, n.EndCol, p.green, "", p)
		}

		if d.Help != "" {
			fmt.Fprintf(w, "%s%s |%s\n", p.cyan, gutter, p.reset)
			fmt.Fprintf(w, "%s%s = help:%s %s\n", p.cyan, gutter, p.reset, d.Help)
		}
		fmt.Fprintln(w)
	}
}

func splitLines(source []byte) []string {
	return strings.Split(strings.ReplaceAll(string(source), "\r\n", "\n"), "\n")
}

func writeSnippet(w io.Writer, lines []string, width, line, col, endLine, endCol int, color, label string, p palette) {
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
	fmt.Fprintf(w, "%s%*s |%s %s\n", p.cyan, width, num, p.reset, text)
	fmt.Fprintf(w, "%s%s |%s %s%s%s%s%s\n", p.cyan, strings.Repeat(" ", width), p.reset, pad.String(), color, carets, p.reset, label)
}
