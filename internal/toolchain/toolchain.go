package toolchain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const EnvDir = "WISP_TOOLCHAIN"

const (
	SourcePackage = "package"
	SourceEnv     = EnvDir
	SourcePath    = "PATH"
)

type Tool struct {
	Name   string
	Path   string
	Source string
}

func PackageDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), "..", "toolchain", "bin")
}

type location struct {
	dir    string
	source string
}

func locations() []location {
	var locs []location
	if dir := PackageDir(); dir != "" {
		locs = append(locs, location{dir, SourcePackage})
	}
	if dir := os.Getenv(EnvDir); dir != "" {
		locs = append(locs, location{dir, SourceEnv})
	}
	return locs
}

func Dirs() []string {
	var dirs []string
	for _, loc := range locations() {
		dirs = append(dirs, loc.dir)
	}
	return dirs
}

func Locate(names ...string) (Tool, error) {
	locs := locations()
	for _, loc := range locs {
		for _, name := range names {
			path := filepath.Join(loc.dir, executable(name))
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				return Tool{Name: name, Path: path, Source: loc.source}, nil
			}
		}
	}
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return Tool{Name: name, Path: path, Source: SourcePath}, nil
		}
	}
	var looked []string
	for _, loc := range locs {
		looked = append(looked, loc.dir)
	}
	looked = append(looked, "PATH")
	return Tool{}, fmt.Errorf("%s was not found (looked in %s): run `wisp doctor` to see how to fix it", strings.Join(names, " or "), strings.Join(looked, ", "))
}

func Find(names ...string) (string, error) {
	tool, err := Locate(names...)
	return tool.Path, err
}

func LibDir(system string) (string, error) {
	path, ok := FindLibDir(system)
	if ok {
		return path, nil
	}
	return "", fmt.Errorf("the %s libraries were not found: they come with the wisp release package, so install one or run `wisp doctor`", system)
}

func FindLibDir(system string) (string, bool) {
	for _, dir := range Dirs() {
		path := filepath.Join(dir, "..", system, "lib")
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return filepath.Clean(path), true
		}
	}
	return "", false
}

func Version(tool Tool) string {
	flag := "--version"
	if tool.Name == "nasm" {
		flag = "-v"
	}
	out, err := exec.Command(tool.Path, flag).Output()
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	return strings.TrimSpace(line)
}

func executable(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
