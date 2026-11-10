package folders

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func Inside(file, dir string) bool {
	if file == "" || dir == "" {
		return true
	}
	fileDir := filepath.Dir(file)
	if within(fileDir, dir) {
		return true
	}
	fileTail, ok := stdTail(fileDir)
	if !ok {
		return false
	}
	dirTail, ok := stdTail(dir)
	return ok && within(fileTail, dirTail)
}

func within(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func stdTail(path string) (string, bool) {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, part := range slices.Backward(parts) {
		if part == "std" {
			return filepath.Join(append([]string{"/"}, parts[i:]...)...), true
		}
	}
	return "", false
}

func UnitRoot(file string) string {
	if file == "" {
		return ""
	}
	dir := filepath.Dir(file)
	if parts := strings.Split(filepath.ToSlash(dir), "/"); slices.Contains(parts, "std") {
		for i, part := range slices.Backward(parts) {
			if part == "std" {
				return filepath.FromSlash(strings.Join(parts[:i+1], "/"))
			}
		}
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "wisp.toml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
