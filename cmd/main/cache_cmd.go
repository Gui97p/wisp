package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/spf13/cobra"
)

var cacheOld bool

var cacheCmd = &cobra.Command{
	Use:          "cache",
	Short:        "Manage the standard library cache",
	SilenceUsage: true,
}

var cacheDirCmd = &cobra.Command{
	Use:          "dir",
	Short:        "Print the cache directory",
	Args:         cobra.NoArgs,
	RunE:         runCacheDir,
	SilenceUsage: true,
}

var cacheCleanCmd = &cobra.Command{
	Use:          "clean",
	Short:        "Remove cached standard library objects",
	Args:         cobra.NoArgs,
	RunE:         runCacheClean,
	SilenceUsage: true,
}

var cacheKeyDir = regexp.MustCompile(`^[0-9a-f]{16}$`)

func init() {
	cacheCleanCmd.Flags().BoolVar(&cacheOld, "old", false, "keep the entry the current compiler would use")
	cacheCmd.AddCommand(cacheDirCmd, cacheCleanCmd)
}

func runCacheDir(cmd *cobra.Command, args []string) error {
	base, err := cacheBase()
	if err != nil {
		return err
	}

	fmt.Println(base)
	return nil
}

func runCacheClean(cmd *cobra.Command, args []string) error {
	base, err := cacheBase()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		fmt.Printf("nothing to clean in %s\n", base)
		return nil
	}
	if err != nil {
		return err
	}

	keep := ""
	if cacheOld {
		if keep, err = cacheKey(); err != nil {
			return err
		}
	}

	removed, freed := 0, int64(0)
	for _, e := range entries {
		if !e.IsDir() || !cacheKeyDir.MatchString(e.Name()) || e.Name() == keep {
			continue
		}

		path := filepath.Join(base, e.Name())
		freed += dirSize(path)
		if err := os.RemoveAll(path); err != nil {
			return err
		}
		removed++
	}

	fmt.Printf("removed %d cache entries (%s) from %s\n", removed, humanSize(freed), base)
	return nil
}

func dirSize(dir string) int64 {
	var size int64
	filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			size += info.Size()
		}
		return nil
	})

	return size
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}
