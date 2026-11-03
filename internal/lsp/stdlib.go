package lsp

import (
	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/target"
)

func analysisTarget(path string) string {
	s := loadSettings(path)
	selector, variant := module.VariantSelector(path)
	if !variant || module.FileAppliesTo(path, s.target) {
		return s.target
	}
	for _, t := range target.List() {
		if t.Matches(selector) {
			return t.Name
		}
	}
	return s.target
}

func buildProjectGraph(entry string, overrides map[string][]byte) ([]*module.Module, error) {
	s := loadSettings(entry)
	if s.configErr != nil {
		module.ConfigureStdlib("", false)
	} else {
		module.ConfigureStdlib(s.stdPath, s.stdDisabled)
	}
	module.ConfigureTarget(analysisTarget(entry))

	return module.BuildGraphWithOverrides(entry, overrides)
}

func discoverProjectGraph(path string, overrides map[string][]byte) ([]*module.Module, error) {
	s := loadSettings(path)
	if s.configErr != nil {
		module.ConfigureStdlib("", false)
	} else {
		module.ConfigureStdlib(s.stdPath, s.stdDisabled)
	}
	module.ConfigureTarget(analysisTarget(path))

	return module.DiscoverProjectWithOverrides(s.root, overrides)
}
