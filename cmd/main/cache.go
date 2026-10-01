package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/target"
)

type stdCache struct {
	dir string
}

func cacheBase() (string, error) {
	if base := os.Getenv("WISP_CACHE_DIR"); base != "" {
		return base, nil
	}

	userCache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("cannot locate cache directory: %w", err)
	}

	return filepath.Join(userCache, "wisp"), nil
}

func openStdCache(t target.Target) (*stdCache, error) {
	base, err := cacheBase()
	if err != nil {
		return nil, err
	}

	key, err := cacheKey()
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256([]byte(t.CompileKey()))
	dir := fmt.Sprintf("%s-%s", t.Name, hex.EncodeToString(sum[:])[:8])

	return &stdCache{dir: filepath.Join(base, key, dir)}, nil
}

func cacheKey() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exeData, err := os.ReadFile(exe)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write(exeData)

	stdRoot, err := module.StdlibRoot()
	if err != nil {
		return "", err
	}

	var files []string
	err = filepath.WalkDir(stdRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(p) == ".wsp" {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(files)

	for _, f := range files {
		rel, _ := filepath.Rel(stdRoot, f)
		data, err := os.ReadFile(f)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%s\x00%d\x00", filepath.ToSlash(rel), len(data))
		h.Write(data)
	}

	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

func (c *stdCache) object(modPath string, build func(objPath string) error) (string, error) {
	objPath := filepath.Join(c.dir, filepath.FromSlash(modPath)+".o")
	if _, err := os.Stat(objPath); err == nil {
		return objPath, nil
	}

	if err := os.MkdirAll(filepath.Dir(objPath), 0755); err != nil {
		return "", err
	}

	tmp := fmt.Sprintf("%s.tmp%d", objPath, os.Getpid())
	defer os.Remove(tmp)

	if err := build(tmp); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, objPath); err != nil {
		return "", err
	}

	return objPath, nil
}
