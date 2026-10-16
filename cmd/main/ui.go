package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	quietOpt   bool
	verboseOpt bool
)

func addOutputFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.BoolVarP(&quietOpt, "quiet", "q", false, "print nothing but diagnostics and errors")
	f.BoolVarP(&verboseOpt, "verbose", "v", false, "print every step: outputs, timings and the assembler and linker commands")
}

func status(format string, args ...any) {
	if quietOpt {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func detail(format string, args ...any) {
	if !verboseOpt {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func relPath(p string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return p
	}
	rel, err := filepath.Rel(cwd, p)
	if err != nil || strings.HasPrefix(rel, "..") {
		return p
	}
	return rel
}

func elapsed(since time.Time) string {
	d := time.Since(since)
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func count(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
