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

func Dirs() []string {
	var dirs []string
	if dir := os.Getenv(EnvDir); dir != "" {
		dirs = append(dirs, dir)
	}
	if exe, err := os.Executable(); err == nil {
		if exe, err = filepath.EvalSymlinks(exe); err == nil {
			dirs = append(dirs, filepath.Join(filepath.Dir(exe), "..", "toolchain", "bin"))
		}
	}
	return dirs
}

func Find(names ...string) (string, error) {
	dirs := Dirs()
	for _, dir := range dirs {
		for _, name := range names {
			path := filepath.Join(dir, executable(name))
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				return path, nil
			}
		}
	}
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("%s was not found (looked in %s and in PATH)", strings.Join(names, " or "), strings.Join(dirs, ", "))
}

func LibDir(system string) (string, error) {
	dirs := Dirs()
	for _, dir := range dirs {
		path := filepath.Join(dir, "..", system, "lib")
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return filepath.Clean(path), nil
		}
	}
	return "", fmt.Errorf("the %s libraries were not found (looked next to %s): run scripts/fetch-toolchain.sh", system, strings.Join(dirs, ", "))
}

func executable(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
