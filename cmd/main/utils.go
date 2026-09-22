package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

func resolveOutput(inputPath, output, outDir, ext string) (string, error) {
	if output == "" {
		base := strings.TrimSuffix(filepath.Base(inputPath), ".wsp")
		output = base + ext
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(outDir, output), nil
}

func modulePath(outDir, entryFile, output, modPath string, isEntry bool) (string, error) {
	if isEntry {
		return resolveOutput(entryFile, output, outDir, ".lua")
	}

	name := strings.TrimSuffix(modPath, ".wsp") + ".lua"
	full := filepath.Join(outDir, "__wisp_modules", name)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return "", err
	}
	return full, nil
}

func resolveEntry(args []string) (string, error) {
	if len(args) == 1 {
		inputPath := args[0]

		root := module.FindProjectRoot(filepath.Dir(inputPath))
		if luaOutDir == "" {
			luaOutDir = filepath.Join(root, "dist")
		}
		if buildOutDir == "" {
			buildOutDir = filepath.Join(root, "bin")
		}
		return inputPath, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	root := module.FindProjectRoot(cwd)
	if luaOutDir == "" {
		luaOutDir = filepath.Join(root, "dist")
	}
	if buildOutDir == "" {
		buildOutDir = filepath.Join(root, "bin")
	}

	cfg, err := module.LoadConfig(root)
	if err != nil || cfg.Entry == "" {
		return "", fmt.Errorf("no entry file given and no 'entry' configured in wisp.toml")
	}

	return filepath.Join(root, cfg.Entry), nil
}
