package module

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/target"
)

func ConfigureTarget(name string) {
	target.SetActive(name)
}

func currentTarget() string {
	return target.Active()
}

func FileDirective(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "" || strings.HasPrefix(line, "//"):
			continue
		case strings.HasPrefix(line, "#target"):
			return strings.TrimSpace(strings.TrimPrefix(line, "#target"))
		}
		break
	}
	return ""
}

func packageEntry(dir string) string {
	entry := filepath.Join(dir, filepath.Base(dir)+".wsp")
	if info, err := os.Stat(entry); err == nil && !info.IsDir() {
		return entry
	}
	return ""
}

func isVariantFile(file string) bool {
	entry := packageEntry(filepath.Dir(file))
	return entry != "" && entry != file && FileDirective(file) != ""
}

func directiveVariants(file string) []string {
	dir := filepath.Dir(file)
	if packageEntry(dir) != file {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var variants []string
	for _, entry := range entries {
		candidate := filepath.Join(dir, entry.Name())
		if entry.IsDir() || !strings.HasSuffix(candidate, ".wsp") || candidate == file {
			continue
		}
		expr := FileDirective(candidate)
		if expr == "" {
			continue
		}
		if target.ValidateSelectorText(expr) != nil || target.MatchesText(currentTarget(), expr) {
			variants = append(variants, candidate)
		}
	}
	sort.Strings(variants)
	return variants
}

func withVariants(files []string) []string {
	if len(files) == 0 {
		return files
	}
	return append(slices.Clone(files), directiveVariants(files[0])...)
}
