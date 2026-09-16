package diag

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

func Render(w io.Writer, filename string, source []byte, diags List) {
	lines := strings.Split(string(source), "\n")

	for _, d := range diags {
		lineNum := strconv.Itoa(d.Line)
		gutter := strings.Repeat(" ", len(lineNum))

		fmt.Fprintf(w, ">> \x1b[1;31merror:\x1b[0m %s\n", d.Message)
		fmt.Fprintf(w, "\x1b[1;36m--> \x1b[0m%s:%d:%d\n", filename, d.Line, d.Col)
		fmt.Fprintf(w, "\x1b[1;36m%s |\x1b[0m\n", gutter)
		if d.Line-1 >= 0 && d.Line-1 < len(lines) {
			fmt.Fprintf(w, "\x1b[1;36m%s |\x1b[0m %s\n", lineNum, lines[d.Line-1])
			fmt.Fprintf(w, "\x1b[1;36m%s |\x1b[0m %s\x1b[1;31m^\x1b[0m\n", gutter, strings.Repeat(" ", d.Col-1))
		}
		fmt.Fprintln(w)
	}
}
