package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Gui97p/wisp/internal/module"
	"github.com/spf13/cobra"
)

var perMember = map[string]bool{"check": true, "build": true, "test": true, "fmt": true, "clean": true}

func init() {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return workspaceHook(cmd)
	}
}

func workspaceHook(cmd *cobra.Command) error {
	name := cmd.Name()
	if !perMember[name] && name != "run" {
		return nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	root := module.FindProjectRoot(cwd)
	cfg, err := module.LoadConfig(root)
	if err != nil || !cfg.IsWorkspace() || module.HasOwnSources(root) {
		return nil
	}
	members := cfg.WorkspaceMembers(root)

	if name == "run" {
		names := make([]string, len(members))
		for i, m := range members {
			names[i] = m.Name
		}
		return fmt.Errorf("this folder is a workspace with no program of its own: run a program from one of its members (%s), for instance cd %s && wisp run", strings.Join(names, ", "), relPath(members[0].Dir))
	}

	failed := 0
	for _, member := range members {
		fmt.Fprintf(os.Stderr, "== %s (%s) ==\n", member.Name, relPath(member.Dir))
		run := exec.Command(os.Args[0], os.Args[1:]...)
		run.Dir = member.Dir
		run.Stdin, run.Stdout, run.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := run.Run(); err != nil {
			failed++
		}
	}
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "%d of %d projects failed\n", failed, len(members))
		os.Exit(1)
	}
	os.Exit(0)
	return nil
}

func workspaceDescription(cfg *module.Config, root string) string {
	if !cfg.IsWorkspace() {
		if wsRoot, ok := module.WorkspaceRootOf(root); ok {
			return "member of " + wsRoot
		}
		return "(none)"
	}
	members := cfg.WorkspaceMembers(root)
	parts := make([]string, 0, len(members))
	for _, m := range members {
		parts = append(parts, m.Name+" = "+filepath.ToSlash(relPath(m.Dir)))
	}
	return strings.Join(parts, ", ")
}
