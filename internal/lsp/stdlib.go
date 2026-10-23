package lsp

import "github.com/Gui97p/wisp/internal/module"

func buildProjectGraph(entry string, overrides map[string][]byte) ([]*module.Module, error) {
	s := loadSettings(entry)
	if s.configErr != nil {
		module.ConfigureStdlib("", false)
	} else {
		module.ConfigureStdlib(s.stdPath, s.stdDisabled)
	}

	return module.BuildGraphWithOverrides(entry, overrides)
}

func discoverProjectGraph(path string, overrides map[string][]byte) ([]*module.Module, error) {
	s := loadSettings(path)
	if s.configErr != nil {
		module.ConfigureStdlib("", false)
	} else {
		module.ConfigureStdlib(s.stdPath, s.stdDisabled)
	}

	return module.DiscoverProjectWithOverrides(s.root, overrides)
}
