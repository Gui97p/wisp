package module

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func isStandaloneSource(buffer []byte) bool {
	for _, line := range strings.Split(string(buffer), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "" || strings.HasPrefix(line, "//"):
			continue
		case strings.HasPrefix(line, "#standalone"):
			return true
		case strings.HasPrefix(line, "#"):
			continue
		}
		return false
	}
	return false
}

func isStandalone(file string, overrides map[string][]byte) bool {
	buffer, ok := overrides[file]
	if !ok {
		var err error
		if buffer, err = os.ReadFile(file); err != nil {
			return false
		}
	}
	return isStandaloneSource(buffer)
}

func folderFiles(dir string, overrides map[string][]byte) (files, standalone []string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".wsp") && !skipFile(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		file := filepath.Join(dir, name)
		if isStandalone(file, overrides) {
			standalone = append(standalone, file)
		} else {
			files = append(files, file)
		}
	}
	return files, standalone
}

const rootModule = "main"

const RootModule = rootModule

func folderKey(root, dir string) string {
	if stdRoot, err := StdlibRoot(); err == nil {
		if rel, err := filepath.Rel(stdRoot, dir); err == nil && !strings.HasPrefix(rel, "..") {
			if rel == "." {
				return "std"
			}
			return "std/" + filepath.ToSlash(rel)
		}
	}
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." {
		return rootModule
	}
	return filepath.ToSlash(rel)
}

func fileKey(root, file string) string {
	if key, ok := stdModulePath(file); ok {
		return key
	}
	key, err := canonicalModulePath(root, file)
	if err != nil {
		return file
	}
	return key
}
