package format

import (
	"path/filepath"
	"slices"
	"strings"
)

func Excluded(root string, excludes []string, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	for _, pattern := range excludes {
		pattern = strings.Trim(filepath.ToSlash(pattern), "/")
		switch {
		case pattern == "":
		case rel == pattern, strings.HasPrefix(rel, pattern+"/"):
			return true
		case !strings.Contains(pattern, "/"):
			if slices.Contains(strings.Split(rel, "/"), pattern) {
				return true
			}
		}
	}
	return false
}
