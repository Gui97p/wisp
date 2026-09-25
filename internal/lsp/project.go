package lsp

import (
	"path/filepath"

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
