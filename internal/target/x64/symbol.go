package x64

import (
	"fmt"
	"regexp"
	"strings"
)

var re = regexp.MustCompile(`[^A-Za-z0-9_]`)

func symbolName(mod string, parts ...string) string {
	if len(parts) == 0 {
		return re.ReplaceAllString(mod, "_")
	}
	return fmt.Sprintf("%s_%s", re.ReplaceAllString(mod, "_"), strings.Join(parts, "_"))
}

func (t *X64Target) funcSymbol(mod, name string) string {
	if t.isEntry && name == "main" {
		return "main"
	}
	return symbolName(mod, name)
}
