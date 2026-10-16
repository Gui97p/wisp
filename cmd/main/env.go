package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/target"
	"github.com/Gui97p/wisp/internal/version"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:          "env [key...]",
	Short:        "Print what the compiler resolved for this project",
	Args:         cobra.ArbitraryArgs,
	RunE:         runEnv,
	SilenceUsage: true,
}

var targetsCmd = &cobra.Command{
	Use:          "targets",
	Short:        "List the built-in and project targets",
	Args:         cobra.NoArgs,
	RunE:         runTargets,
	SilenceUsage: true,
}

func init() {
	addProjectFlags(envCmd)
	envCmd.Flags().StringVar(&targetOpt, "target", "", "resolve for this target instead of the default")
}

type envEntry struct {
	key   string
	value string
}

func runEnv(cmd *cobra.Command, args []string) error {
	root, cfg, err := openProject(cmd)
	if err != nil {
		return err
	}

	tgt, err := resolveTarget(cfg)
	if err != nil {
		return err
	}

	entries := []envEntry{
		{"version", version.Version()},
		{"root", root},
		{"config", configPath(root)},
		{"entry", orDefault(cfg.Entry, "(the only main)")},
		{"target", tgt.Name},
		{"arch", tgt.Arch},
		{"os", orDefault(tgt.OS, "(none)")},
		{"std", stdDescription()},
		{"cache", cacheDescription()},
	}

	switch tgt.Arch {
	case "x64":
		out := resolveBuildTargets(root, cfg, tgt, false)
		entries = append(entries,
			envEntry{"bin", out.binPath},
			envEntry{"obj", out.objDir},
			envEntry{"asm", out.asmDir},
			envEntry{"libc", fmt.Sprint(useLibc(cfg, tgt))},
			envEntry{"link", strings.Join(append(append([]string{}, tgt.LinkArgs...), cfg.Link...), " ")},
		)
	case "lua":
		entries = append(entries, envEntry{"lua", resolveLuaOutDir(root, cfg, tgt)})
	}

	if len(args) > 0 {
		for _, key := range args {
			found := false
			for _, e := range entries {
				if e.key == key {
					fmt.Println(e.value)
					found = true
				}
			}
			if !found {
				return fmt.Errorf("unknown key %q", key)
			}
		}
		return nil
	}

	width := 0
	for _, e := range entries {
		width = max(width, len(e.key))
	}
	for _, e := range entries {
		fmt.Printf("%-*s  %s\n", width, e.key, e.value)
	}
	return nil
}

func configPath(root string) string {
	path := filepath.Join(root, "wisp.toml")
	if _, err := os.Stat(path); err != nil {
		return "(none)"
	}
	return path
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func stdDescription() string {
	if freestandingOpt {
		return "(freestanding)"
	}
	path, err := module.StdlibRoot()
	if err != nil {
		return "(not found)"
	}
	return path
}

func cacheDescription() string {
	base, err := cacheBase()
	if err != nil {
		return "(unavailable)"
	}
	return base
}

func runTargets(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root := module.FindProjectRoot(cwd)

	cfg, err := module.LoadConfig(root)
	if err != nil {
		return err
	}

	current := runtime.GOOS + "_x64"
	if cfg.Target != "" {
		current = cfg.Target
	}

	rows := [][]string{{"", "name", "arch", "os", "kind"}}
	for _, t := range target.List() {
		kind := "custom"
		switch {
		case t.Builtin && t.Modified:
			kind = "built-in, adjusted by wisp.toml"
		case t.Builtin:
			kind = "built-in"
		}
		mark := ""
		if t.Name == current {
			mark = "*"
		}
		rows = append(rows, []string{mark, t.Name, t.Arch, orDefault(t.OS, "-"), kind})
	}

	widths := make([]int, 5)
	for _, r := range rows {
		for i, c := range r {
			widths[i] = max(widths[i], len(c))
		}
	}
	for _, r := range rows {
		fmt.Printf("%-*s  %-*s  %-*s  %-*s  %s\n", widths[0], r[0], widths[1], r[1], widths[2], r[2], widths[3], r[3], r[4])
	}
	fmt.Println("\n* default for this project")
	return nil
}
