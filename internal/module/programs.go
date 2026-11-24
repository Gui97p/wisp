package module

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Gui97p/wisp/internal/target"
)

type Program struct {
	Name string
	Key  string
	Dir  string
	File string
}

var mainDeclaration = regexp.MustCompile(`(?m)^(?:(?:public|internal|private)\s+)?func\s+main\s*\(`)

func fileHeaderTarget(buffer []byte) string {
	for _, line := range strings.Split(string(buffer), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "" || strings.HasPrefix(line, "//"):
			continue
		case strings.HasPrefix(line, "#target"):
			return strings.TrimSpace(strings.TrimPrefix(line, "#target"))
		case strings.HasPrefix(line, "#"):
			continue
		}
		return ""
	}
	return ""
}

func definesMain(file string) bool {
	buffer, err := os.ReadFile(file)
	if err != nil {
		return false
	}
	if expr := fileHeaderTarget(buffer); expr != "" && !target.MatchesText(currentTarget(), expr) {
		return false
	}
	return mainDeclaration.Match(buffer)
}

func programName(root, key string) string {
	if key == rootModule {
		return filepath.Base(root)
	}
	base := path.Base(key)
	if base == "main" {
		dir := path.Dir(key)
		if dir == "." {
			return filepath.Base(root)
		}
		return path.Base(dir)
	}
	return base
}

func DiscoverPrograms(root string) []Program {
	roots, err := discoverModuleRoots(root, nil)
	if err != nil {
		return nil
	}

	var found []Program
	for _, r := range roots {
		for _, file := range r.files {
			if definesMain(file) {
				found = append(found, Program{Name: programName(root, r.path), Key: r.path, Dir: r.dir, File: file})
				break
			}
		}
	}

	count := map[string]int{}
	for _, p := range found {
		count[p.Name]++
	}
	for i, p := range found {
		if count[p.Name] > 1 {
			found[i].Name = strings.ReplaceAll(p.Key, "/", "_")
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].Name < found[j].Name })
	return found
}

func (c *Config) ApplyDiscoveredPrograms(root string) error {
	file := filepath.Join(root, "wisp.toml")
	text := ""
	if data, err := os.ReadFile(file); err == nil {
		text = string(data)
	}
	return c.ApplyDiscoveredProgramsSource(root, file, text)
}

func (c *Config) ApplyDiscoveredProgramsSource(root, file, text string) error {
	found := DiscoverPrograms(root)

	declaredEntries := map[string]bool{}
	for _, program := range c.Programs {
		if program.Entry != "" {
			declaredEntries[filepath.Clean(filepath.Join(root, program.Entry))] = true
		}
	}

	names := make([]string, len(found))
	byName := map[string]bool{}
	for i, p := range found {
		names[i] = p.Name
		byName[p.Name] = true
	}

	for _, p := range found {
		rel, err := filepath.Rel(root, p.File)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		declared, isDeclared := c.Programs[p.Name]
		switch {
		case isDeclared && declared.Entry == "":
			declared.Entry = rel
			c.Programs[p.Name] = declared
		case isDeclared:
		case declaredEntries[filepath.Clean(p.File)] || declaredEntries[filepath.Clean(p.Dir)]:
		default:
			if c.Programs == nil {
				c.Programs = map[string]ProgramConfig{}
			}
			c.Programs[p.Name] = ProgramConfig{Entry: rel}
		}
	}

	known := "no folder or file defines main"
	if len(names) > 0 {
		known = "the programs are " + strings.Join(names, ", ")
	}

	var issues []ConfigIssue
	var undeclared []string
	for name, program := range c.Programs {
		if program.Entry == "" && !byName[name] {
			undeclared = append(undeclared, name)
		}
	}
	sort.Strings(undeclared)
	for _, name := range undeclared {
		issues = append(issues, ConfigIssue{Line: locateKey(text, toml.Key{"program", name}), Message: fmt.Sprintf("%q is not a program of this project", name), Help: known})
	}
	if c.Default != "" {
		if _, ok := c.Programs[c.Default]; !ok {
			issues = append(issues, ConfigIssue{Line: locateKey(text, toml.Key{"default"}), Message: fmt.Sprintf("default names %q, which is not a program of this project", c.Default), Help: known})
		}
	}
	sort.Slice(issues, func(i, j int) bool { return issues[i].Line < issues[j].Line })
	if len(issues) > 0 {
		return &ConfigError{File: file, Source: text, Issues: issues}
	}
	return nil
}
