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

func intermediatePath(tmpDir, subdir, ext, entryFile, modPath string, isEntry bool) (string, error) {
	name := strings.TrimSuffix(filepath.Base(entryFile), ".wsp") + ext
	if !isEntry {
		name = strings.TrimSuffix(modPath, ".wsp") + ext
	}

	full := filepath.Join(tmpDir, subdir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return "", err
	}
	return full, nil
}

func objPath(tmpDir, entryFile, modPath string, isEntry bool) (string, error) {
	return intermediatePath(tmpDir, "obj", ".o", entryFile, modPath, isEntry)
}

func asmPath(tmpDir, entryFile, modPath string, isEntry bool) (string, error) {
	return intermediatePath(tmpDir, "asm", ".asm", entryFile, modPath, isEntry)
}

func anchorToRoot(root, dir, defaultName string) string {
	if dir == "" {
		dir = defaultName
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(root, dir)
}

func resolveEntry(args []string) (string, error) {
	if len(args) == 1 {
		inputPath := args[0]

		root := module.FindProjectRoot(filepath.Dir(inputPath))
		luaOutDir = anchorToRoot(root, luaOutDir, "dist")
		buildOutDir = anchorToRoot(root, buildOutDir, "bin")
		buildBuildDir = anchorToRoot(root, buildBuildDir, "build")
		return inputPath, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	root := module.FindProjectRoot(cwd)
	luaOutDir = anchorToRoot(root, luaOutDir, "dist")
	buildOutDir = anchorToRoot(root, buildOutDir, "bin")
	buildBuildDir = anchorToRoot(root, buildBuildDir, "build")

	cfg, err := module.LoadConfig(root)
	if err != nil || cfg.Entry == "" {
		return "", fmt.Errorf("no entry file given and no 'entry' configured in wisp.toml")
	}

	return filepath.Join(root, cfg.Entry), nil
}
