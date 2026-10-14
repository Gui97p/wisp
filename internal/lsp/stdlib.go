package lsp

import (
	"path/filepath"

	"github.com/Gui97p/wisp/internal/module"
)

func buildProjectGraph(entry string, overrides map[string][]byte) ([]*module.Module, error) {
	root := module.FindProjectRoot(filepath.Dir(entry))
	path, disabled, err := module.LoadStdlibSettings(root)
	if err != nil {
		path, disabled = "", false
	}
	module.ConfigureStdlib(path, disabled)

	return module.BuildGraphWithOverrides(entry, overrides)
}
