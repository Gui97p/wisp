package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Gui97p/wisp/internal/module"
	"github.com/spf13/cobra"
)

func configureStdlib(cmd *cobra.Command, root string, cfg *module.Config) error {
	path, disabled, err := cfg.StdlibSettings(root)
	if err != nil {
		return err
	}

	stdFlag := cmd.Flags().Changed("std")
	noStdFlag := cmd.Flags().Changed("no-std") && noStdOpt

	if stdFlag && noStdFlag {
		return fmt.Errorf("--std and --no-std cannot be used together")
	}

	switch {
	case noStdFlag:
		path, disabled = "", true
	case stdFlag:
		abs, err := filepath.Abs(stdOpt)
		if err != nil {
			return err
		}
		path, disabled = abs, false
	}

	if path != "" {
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("standard library path %q is not a folder", path)
		}
	}

	module.ConfigureStdlib(path, disabled)
	return nil
}
