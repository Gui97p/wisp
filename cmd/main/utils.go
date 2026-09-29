package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Gui97p/wisp/internal/module"
)

func resolveModules(inputPath string) ([]*module.Module, bool) {
	modules, err := module.BuildGraph(inputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, false
	}
	return modules, true
}

func resolveEntry(args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	root := module.FindProjectRoot(cwd)

	cfg, err := module.LoadConfig(root)
	if err != nil || cfg.Entry == "" {
		return "", fmt.Errorf("no entry file given and no 'entry' configured in wisp.toml")
	}

	return filepath.Join(root, cfg.Entry), nil
}
