package native

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Spec struct {
	Libs        []string
	Search      []string
	Interpreter string
}

type Result struct {
	Args        []string
	Interpreter string
}

var systemDirs = []string{
	"/usr/local/lib",
	"/usr/lib",
	"/usr/lib64",
	"/lib64",
	"/lib",
	"/usr/lib/x86_64-linux-gnu",
	"/lib/x86_64-linux-gnu",
}

var interpreters = []string{
	"/lib64/ld-linux-x86-64.so.2",
	"/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2",
	"/usr/lib64/ld-linux-x86-64.so.2",
	"/lib/ld-musl-x86_64.so.1",
}

func Resolve(format string, spec Spec, root string, libc bool) (Result, error) {
	if len(spec.Libs) == 0 {
		return Result{}, nil
	}

	windows := format == "win64"
	var dirs []string
	for _, dir := range spec.Search {
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(root, dir)
		}
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			return Result{}, fmt.Errorf("the native search folder %s does not exist", dir)
		}
		dirs = append(dirs, dir)
	}
	if !windows {
		for _, dir := range systemDirs {
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				dirs = append(dirs, dir)
			}
		}
	}

	var res Result
	for _, dir := range dirs {
		res.Args = append(res.Args, "-L"+dir)
	}
	for _, lib := range spec.Libs {
		arg, err := find(lib, dirs, windows)
		if err != nil {
			return Result{}, err
		}
		res.Args = append(res.Args, arg)
	}

	if !windows && !libc {
		interp, err := interpreter(spec.Interpreter)
		if err != nil {
			return Result{}, err
		}
		res.Interpreter = interp
		res.Args = append(res.Args, "--dynamic-linker", interp)
	}
	return res, nil
}

func candidates(lib string, windows bool) []string {
	if windows {
		return []string{"lib" + lib + ".dll.a", lib + ".lib", "lib" + lib + ".a"}
	}
	return []string{"lib" + lib + ".so", "lib" + lib + ".a"}
}

func find(lib string, dirs []string, windows bool) (string, error) {
	names := candidates(lib, windows)
	for _, dir := range dirs {
		for _, name := range names {
			if info, err := os.Stat(filepath.Join(dir, name)); err == nil && !info.IsDir() {
				return "-l" + lib, nil
			}
		}
	}
	if !windows {
		for _, dir := range dirs {
			versions, _ := filepath.Glob(filepath.Join(dir, "lib"+lib+".so.*"))
			if len(versions) == 0 {
				continue
			}
			sort.Strings(versions)
			return "-l:" + filepath.Base(versions[len(versions)-1]), nil
		}
	}
	return "", fmt.Errorf("the native library %s was not found (looked for %s in %s): install it, or add its folder to search in [native]", lib, strings.Join(names, ", "), strings.Join(dirs, ", "))
}

func interpreter(chosen string) (string, error) {
	if chosen != "" {
		if _, err := os.Stat(chosen); err != nil {
			return "", fmt.Errorf("the dynamic linker %s does not exist", chosen)
		}
		return chosen, nil
	}
	for _, path := range interpreters {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no dynamic linker was found (looked for %s): write interpreter = \"...\" in [native]", strings.Join(interpreters, ", "))
}
