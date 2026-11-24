package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Gui97p/wisp/internal/module"
)

func resolveModules(inputPath string) ([]*module.Module, bool) {
	root := module.FindProjectRoot(filepath.Dir(inputPath))
	if path, disabled, err := module.LoadStdlibSettings(root); err == nil {
		module.ConfigureStdlib(path, disabled)
	}

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

func processEnv(vars []module.EnvVar, libDirs []string, osName string) []string {
	env := os.Environ()
	for _, v := range vars {
		if _, set := os.LookupEnv(v.Name); !set {
			env = append(env, v.Name+"="+v.Value)
		}
	}
	if len(libDirs) == 0 {
		return env
	}

	name := "LD_LIBRARY_PATH"
	if osName == "windows" {
		name = "PATH"
	}
	value := strings.Join(libDirs, string(os.PathListSeparator))
	if current := os.Getenv(name); current != "" {
		value += string(os.PathListSeparator) + current
	}
	return append(env, name+"="+value)
}
