package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/module"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "wisp",
	Short:        "Wisp compiler toolchain",
	SilenceUsage: true,
}

func Execute() {
	rootCmd.SilenceErrors = true

	if err := rootCmd.Execute(); err != nil {
		if configErr, ok := errors.AsType[*module.ConfigError](err); ok {
			list := configErr.Diagnostics()
			diag.Render(os.Stderr, configErr.File, []byte(configErr.Source), list)
			errs, warns := diag.Count(list)
			diag.Summary(os.Stderr, errs, warns)
			os.Exit(1)
		}

		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(buildCmd, runCmd, checkCmd, cleanCmd, envCmd, targetsCmd, docCmd, debugCmd, initCmd, cacheCmd, versionCmd, lspCmd)
}
