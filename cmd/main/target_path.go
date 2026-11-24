package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Gui97p/wisp/internal/module"
)

var pathPrograms = map[string]string{}

func looksLikePath(arg string) bool {
	return strings.HasSuffix(arg, ".wsp") || strings.ContainsAny(arg, `/\`) || arg == "." || arg == ".."
}

func entryOf(abs, shown string) (string, error) {
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("%s does not exist", shown)
	}
	if !info.IsDir() {
		if !strings.HasSuffix(abs, ".wsp") {
			return "", fmt.Errorf("%s is not a Wisp file (.wsp)", shown)
		}
		return abs, nil
	}
	for _, name := range []string{"main.wsp", filepath.Base(abs) + ".wsp"} {
		candidate := filepath.Join(abs, name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("the folder %s has no main.wsp or %s.wsp to start from: name the file", shown, filepath.Base(abs))
}

func rootForEntry(entry, cwd string) string {
	root := module.FindProjectRoot(filepath.Dir(entry))
	if _, err := os.Stat(filepath.Join(root, "wisp.toml")); err == nil {
		return root
	}
	if rel, err := filepath.Rel(cwd, entry); err == nil && !strings.HasPrefix(rel, "..") {
		return cwd
	}
	return root
}

func resolvePathTargets(names []string) ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return names, nil
	}

	resolved := make([]string, 0, len(names))
	changedRoot := ""
	for _, name := range names {
		if !looksLikePath(name) {
			resolved = append(resolved, name)
			continue
		}
		abs, err := filepath.Abs(name)
		if err != nil {
			return nil, err
		}
		entry, err := entryOf(abs, name)
		if err != nil {
			return nil, err
		}
		root := rootForEntry(entry, cwd)
		if changedRoot != "" && root != changedRoot {
			return nil, fmt.Errorf("%s belongs to another project than the previous path: name files of one project at a time", name)
		}
		changedRoot = root

		rel, err := filepath.Rel(root, entry)
		if err != nil {
			return nil, err
		}
		program := strings.TrimSuffix(filepath.Base(entry), ".wsp")
		if program == "main" {
			program = filepath.Base(root)
		}
		pathPrograms[program] = filepath.ToSlash(rel)
		resolved = append(resolved, program)
	}

	if changedRoot != "" && changedRoot != cwd {
		if err := os.Chdir(changedRoot); err != nil {
			return nil, err
		}
	}
	return resolved, nil
}

func applyPathPrograms(cfg *module.Config) {
	if len(pathPrograms) == 0 {
		return
	}
	if cfg.Programs == nil {
		cfg.Programs = map[string]module.ProgramConfig{}
	}
	for name, entry := range pathPrograms {
		program := cfg.Programs[name]
		program.Entry = entry
		cfg.Programs[name] = program
	}
}
