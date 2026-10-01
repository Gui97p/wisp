package module

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
	"github.com/Gui97p/wisp/internal/target"
)

type Config struct {
	Entry   string                  `toml:"entry"`
	Target  string                  `toml:"target"`
	Output  OutputConfig            `toml:"output"`
	Targets map[string]TargetConfig `toml:"targets"`
}

type TargetConfig struct {
	Arch    string   `toml:"arch"`
	OS      string   `toml:"os"`
	ABI     string   `toml:"abi"`
	Format  string   `toml:"format"`
	Entry   string   `toml:"entry"`
	Runtime string   `toml:"runtime"`
	Start   string   `toml:"start"`
	Link    []string `toml:"link"`
}

type OutputConfig struct {
	Bin string `toml:"bin"`
	Obj string `toml:"obj"`
	Asm string `toml:"asm"`
	Lua string `toml:"lua"`
}

func LoadConfig(root string) (*Config, error) {
	path := filepath.Join(root, "wisp.toml")

	if _, err := os.Stat(path); err != nil {
		return &Config{}, nil
	}

	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	if err := cfg.registerTargets(root); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) registerTargets(root string) error {
	names := make([]string, 0, len(c.Targets))
	for name := range c.Targets {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		tc := c.Targets[name]

		var start, runtimeSource string
		if tc.Start != "" {
			data, err := os.ReadFile(filepath.Join(root, tc.Start))
			if err != nil {
				return fmt.Errorf("target %q: cannot read start file: %w", name, err)
			}
			start = string(data)
		}
		if tc.Runtime != "" {
			data, err := os.ReadFile(filepath.Join(root, tc.Runtime))
			if err != nil {
				return fmt.Errorf("target %q: cannot read runtime file: %w", name, err)
			}
			runtimeSource = string(data)
		}

		err := target.Register(target.Target{
			Name:          name,
			Arch:          tc.Arch,
			OS:            tc.OS,
			ABI:           tc.ABI,
			Format:        tc.Format,
			Entry:         tc.Entry,
			StartSource:   start,
			RuntimeSource: runtimeSource,
			LinkArgs:      tc.Link,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
