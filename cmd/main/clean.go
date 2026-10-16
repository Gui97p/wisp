package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/target"
	"github.com/spf13/cobra"
)

var cleanDryRun bool

var cleanCmd = &cobra.Command{
	Use:          "clean",
	Short:        "Remove the files a build writes (binaries, objects, assembly, Lua)",
	Args:         cobra.NoArgs,
	RunE:         runClean,
	SilenceUsage: true,
}

func init() {
	cleanCmd.Flags().BoolVarP(&cleanDryRun, "dry-run", "n", false, "list what would be removed without removing it")
	cleanCmd.Flags().BoolVarP(&quietOpt, "quiet", "q", false, "print nothing")
}

func runClean(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root := module.FindProjectRoot(cwd)

	cfg, err := module.LoadConfig(root)
	if err != nil {
		return err
	}

	var paths []string
	seen := map[string]bool{}
	add := func(p string) {
		p = filepath.Clean(p)
		if p != "" && !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}

	for _, t := range target.List() {
		switch t.Arch {
		case "x64":
			out := resolveBuildTargets(root, cfg, t, false)
			add(out.binPath)
			add(out.objDir)
			add(out.asmDir)
		case "lua":
			add(resolveLuaOutDir(root, cfg, t))
		}
	}

	removed := 0
	for _, p := range paths {
		info, err := os.Lstat(p)
		if err != nil {
			continue
		}
		if err := safeToRemove(root, p, info); err != nil {
			fmt.Fprintf(os.Stderr, "skipped %s: %v\n", relPath(p), err)
			continue
		}
		if !cleanDryRun {
			if err := os.RemoveAll(p); err != nil {
				return err
			}
			removeEmptyParents(root, filepath.Dir(p))
		}
		status("Removed %s", relPath(p))
		removed++
	}

	if removed == 0 {
		status("Nothing to clean")
	}
	return nil
}

func safeToRemove(root, path string, info os.FileInfo) error {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("it is not inside the project")
	}
	if !info.IsDir() {
		return nil
	}

	var blocker string
	filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil || blocker != "" {
			return fs.SkipAll
		}
		name := d.Name()
		if name == ".git" || name == "wisp.toml" || strings.HasSuffix(name, ".wsp") {
			blocker = name
			return fs.SkipAll
		}
		return nil
	})
	if blocker != "" {
		return fmt.Errorf("it holds %s, which is not build output", blocker)
	}
	return nil
}

func removeEmptyParents(root, dir string) {
	for dir != root && strings.HasPrefix(dir, root) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		if os.Remove(dir) != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}
