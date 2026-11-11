package lsp

import (
	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/target"
)

func analysisTarget(path string) string {
	s := loadSettings(path)
	if expr := module.FileDirective(path); expr != "" {
		if target.MatchesText(s.target, expr) {
			return s.target
		}
		if name, ok := target.FirstMatching(expr); ok {
			return name
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
	configureProfile(s)

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
	configureProfile(s)

	return module.DiscoverProjectWithOverrides(s.root, overrides)
}

func configureProfile(s *projectSettings) {
	if s.profile.Name == "" {
		facts, _ := target.BuiltinProfile(target.DefaultProfile)
		target.SetActiveProfile(facts)
		return
	}
	target.SetActiveProfile(s.profile)
}
