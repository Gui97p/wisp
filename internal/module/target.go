package module

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/Gui97p/wisp/internal/target"
)

var (
	activeTargetMu sync.RWMutex
	activeTarget   string
)

func ConfigureTarget(name string) {
	activeTargetMu.Lock()
	defer activeTargetMu.Unlock()
	activeTarget = name
}

func currentTarget() string {
	activeTargetMu.RLock()
	defer activeTargetMu.RUnlock()
	if activeTarget == "" {
		return runtime.GOOS + "_x64"
	}
	return activeTarget
}

func FileApplies(path string) bool {
	base := strings.TrimSuffix(filepath.Base(path), ".wsp")

	selectors := target.Selectors()
	slices.SortFunc(selectors, func(a, b string) int { return len(b) - len(a) })

	for _, selector := range selectors {
		suffix := "_" + selector
		if len(base) > len(suffix) && strings.HasSuffix(base, suffix) {
			return target.MatchesName(currentTarget(), selector)
		}
	}
	return true
}

func targetSuffix(file string) (stem, selector string, ok bool) {
	base := strings.TrimSuffix(filepath.Base(file), ".wsp")

	selectors := target.Selectors()
	slices.SortFunc(selectors, func(a, b string) int { return len(b) - len(a) })

	for _, s := range selectors {
		suffix := "_" + s
		if len(base) > len(suffix) && strings.HasSuffix(base, suffix) {
			return strings.TrimSuffix(base, suffix), s, true
		}
	}
	return "", "", false
}

func isVariantFile(file string) bool {
	stem, _, ok := targetSuffix(file)
	if !ok {
		return false
	}
	info, err := os.Stat(filepath.Join(filepath.Dir(file), stem+".wsp"))
	return err == nil && !info.IsDir()
}

func variantFiles(file string) []string {
	dir := filepath.Dir(file)
	stem := strings.TrimSuffix(filepath.Base(file), ".wsp")

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var variants []string
	for _, entry := range entries {
		candidate := filepath.Join(dir, entry.Name())
		if entry.IsDir() || !strings.HasSuffix(candidate, ".wsp") || !isVariantFile(candidate) {
			continue
		}
		if s, _, _ := targetSuffix(candidate); s == stem && FileApplies(candidate) {
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
	return append(slices.Clone(files), variantFiles(files[0])...)
}
