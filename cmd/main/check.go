package main

import (
	"time"

	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:          "check",
	Short:        "Parse and analyse the project without generating code",
	Args:         cobra.NoArgs,
	RunE:         runCheck,
	SilenceUsage: true,
}

func init() {
	addProjectFlags(checkCmd)
	addOutputFlags(checkCmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
	start := time.Now()

	root, cfg, err := openProject(cmd)
	if err != nil {
		return err
	}

	proj, err := loadProject(root, cfg)
	if err != nil {
		return err
	}

	status("Checked %s in %s", count(len(proj.compiled), "module"), elapsed(start))
	return nil
}
