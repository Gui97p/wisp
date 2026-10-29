package lsp

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/Gui97p/wisp/internal/module"
)

type projectSettings struct {
	root         string
	tomlPath     string
	hasConfig    bool
	entry        string
	stdPath      string
	stdDisabled  bool
	configErr    error
	configSource string
	target       string
}

var (
	projectMu    sync.Mutex
	projectCache *projectSettings
)

func loadSettings(path string) *projectSettings {
	root := module.FindProjectRoot(filepath.Dir(path))
	tomlPath := filepath.Join(root, "wisp.toml")

	source, buffered := fileBuffer[tomlPath]
	text := string(source)
	if !buffered {
		if data, err := os.ReadFile(tomlPath); err == nil {
			text = string(data)
			buffered = true
		}
	}

	projectMu.Lock()
	defer projectMu.Unlock()

	if c := projectCache; c != nil && c.root == root && c.hasConfig == buffered && c.configSource == text {
		return c
	}

	s := &projectSettings{root: root, tomlPath: tomlPath, hasConfig: buffered, configSource: text}

	var cfg *module.Config
	if buffered {
		cfg, s.configErr = module.LoadConfigSource(root, text)
	} else {
		cfg, s.configErr = module.LoadConfig(root)
	}

	if s.configErr == nil {
		s.target = cfg.Target
		if cfg.Entry != "" {
			s.entry = filepath.Join(root, cfg.Entry)
		}
		s.stdPath, s.stdDisabled, s.configErr = cfg.StdlibSettings(root)
	}

	projectCache = s
	return s
}

func resolveProjectEntry(path string) string {
	if s := loadSettings(path); s.entry != "" {
		return s.entry
	}
	return path
}
