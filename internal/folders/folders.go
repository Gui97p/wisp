package folders

import (
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
