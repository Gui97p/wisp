package main

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/version"
	"github.com/spf13/cobra"
)

var versionShort bool

var versionCmd = &cobra.Command{
	Use:          "version",
	Short:        "Print the compiler version",
	Args:         cobra.NoArgs,
	RunE:         runVersion,
	SilenceUsage: true,
}

func init() {
	versionCmd.Flags().BoolVar(&versionShort, "short", false, "print only the version number")
	rootCmd.Version = version.String()
	rootCmd.SetVersionTemplate("wisp {{.Version}}\n")
}

func runVersion(cmd *cobra.Command, args []string) error {
	if versionShort {
		fmt.Println(version.Version())
		return nil
	}

	fmt.Printf("wisp %s\n", version.String())

	b := version.BuildInfo()
	if b.Commit != "" {
		commit := b.Commit
		if len(commit) > 7 {
			commit = commit[:7]
		}
		if b.Modified {
			commit += " (modified)"
		}
		fmt.Printf("  commit   %s\n", commit)
	}
	if b.Time != "" {
		fmt.Printf("  built    %s\n", strings.TrimSpace(b.Time))
	}
	fmt.Printf("  go       %s %s\n", b.Go, b.Platform)

	return nil
}
