package lsp

import (
	"path/filepath"
	"strings"

	"github.com/Gui97p/wisp/internal/module"
)

func resolveProjectEntry(path string) string {
	root := module.FindProjectRoot(filepath.Dir(path))

	cfg, err := module.LoadConfig(root)
	if err != nil || cfg.Entry == "" {
		return path
	}

	return filepath.Join(root, cfg.Entry)
}

func isUnderStdlib(files []string) bool {
	root, err := module.StdlibRoot()
	if err != nil || len(files) == 0 {
		return false
	}

	for _, f := range files {
		rel, err := filepath.Rel(root, f)
		if err != nil || strings.HasPrefix(rel, "..") {
			return false
		}
	}
	return true
}
