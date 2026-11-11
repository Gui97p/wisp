package main

import (
	"fmt"
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
	checkCmd.Flags().StringVar(&osOpt, "os", "", "analyse for this system instead of the default")
	checkCmd.Flags().StringVar(&archOpt, "arch", "", "analyse for this architecture instead of the default")
	addProfileFlags(checkCmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
	start := time.Now()

	root, cfg, err := openProject(cmd)
	if err != nil {
		return err
	}

	for _, name := range cfg.ProgramNames(projectName(root)) {
		if err := validateEntry(root, cfg.ForProgram(name)); err != nil {
			return err
		}
	}

	proj, err := loadProject(root, cfg)
	if err != nil {
		return err
	}

	for _, name := range cfg.LibraryNames() {
		roots, err := libraryRoots(root, name, cfg.Libraries[name].Modules)
		if err != nil {
			return err
		}
		probe := *proj
		if err := probe.restrict(roots); err != nil {
			return fmt.Errorf("the library %s: %w", name, err)
		}
	}

	status("Checked %s in %s", count(len(proj.compiled), "module"), elapsed(start))
	return nil
}
