package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Gui97p/wisp/internal/format"
	"github.com/Gui97p/wisp/internal/module"
	"github.com/spf13/cobra"
)

var (
	fmtCheck  bool
	fmtStdout bool
)

var fmtCmd = &cobra.Command{
	Use:   "fmt [path...]",
	Short: "Format the Wisp files of the project",
	Long: `Format Wisp source files. There is one style and no options: tabs for indentation, one space around operators, the line breaks and the comments the author wrote kept where they are.

Without arguments it formats every .wsp file of the project. A path (a file or a folder) restricts it. wisp.toml can leave files out:

    [fmt]
    exclude = ["generated", "third_party/api.wsp"]

A file with a syntax error is not touched, and the error is shown. With --check nothing is written: the files that would change are listed, and the exit status is 1 if there are any.`,
	Args:         cobra.ArbitraryArgs,
	RunE:         runFmt,
	SilenceUsage: true,
}

func init() {
	f := fmtCmd.Flags()
	f.BoolVar(&fmtCheck, "check", false, "list the files that need formatting instead of changing them, and exit with status 1 if there are any")
	f.BoolVar(&fmtStdout, "stdout", false, "print the formatted file instead of writing it (one file)")
	addOutputFlags(fmtCmd)
}

func collectSources(root string, excludes []string, args []string) ([]string, error) {
	if len(args) == 0 {
		args = []string{root}
	}
	var files []string
	seen := map[string]bool{}
	for _, arg := range args {
		abs, err := filepath.Abs(arg)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("%s is not a file or folder", arg)
		}
		if !info.IsDir() {
			if !strings.HasSuffix(abs, ".wsp") {
				return nil, fmt.Errorf("%s is not a Wisp file", arg)
			}
			if !seen[abs] {
				seen[abs] = true
				files = append(files, abs)
			}
			continue
		}
		err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			name := d.Name()
			if d.IsDir() {
				if path != abs && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
					return filepath.SkipDir
				}
				if path != root && path != abs {
					if _, err := os.Stat(filepath.Join(path, "wisp.toml")); err == nil {
						return filepath.SkipDir
					}
				}
				if format.Excluded(root, excludes, path) {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(name, ".wsp") && !format.Excluded(root, excludes, path) && !seen[path] {
				seen[path] = true
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

func runFmt(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root := module.FindProjectRoot(cwd)
	cfg, err := module.LoadConfig(root)
	if err != nil {
		return err
	}

	files, err := collectSources(root, cfg.Fmt.Exclude, args)
	if err != nil {
		return err
	}
	if fmtStdout && len(files) != 1 {
		return fmt.Errorf("--stdout formats one file, and %d were given", len(files))
	}

	changed, failed := 0, 0
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out, err := format.Source(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", relPath(path), err)
			failed++
			continue
		}
		if fmtStdout {
			os.Stdout.Write(out)
			return nil
		}
		if bytes.Equal(src, out) {
			continue
		}
		changed++
		if fmtCheck {
			fmt.Println(relPath(path))
			continue
		}
		if err := os.WriteFile(path, out, 0o644); err != nil {
			return err
		}
		detail("formatted %s", relPath(path))
	}

	switch {
	case failed > 0:
		status("%s could not be formatted", count(failed, "file"))
		os.Exit(1)
	case fmtCheck && changed > 0:
		status("%s would change", count(changed, "file"))
		os.Exit(1)
	case fmtCheck:
		status("%s already formatted", count(len(files), "file"))
	default:
		status("Formatted %s, %d changed", count(len(files), "file"), changed)
	}
	return nil
}
