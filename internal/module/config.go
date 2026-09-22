package module

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Entry string `toml:"entry"`
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
	return &cfg, nil
}
