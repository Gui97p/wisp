package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:          "init",
	Short:        "Initialize a new Wisp project in the current directory",
	Args:         cobra.NoArgs,
	RunE:         runInit,
	SilenceUsage: true,
}

const initTomlContent = `entry = "main.wsp"
`

const initMainContent = `func main() {
    emit("Hello, Wisp!");
}
`

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	tomlPath := filepath.Join(cwd, "wisp.toml")
	if _, err := os.Stat(tomlPath); err == nil {
		return fmt.Errorf("wisp.toml already exists in %s", cwd)
	}

	if err := os.WriteFile(tomlPath, []byte(initTomlContent), 0644); err != nil {
		return err
	}

	mainPath := filepath.Join(cwd, "main.wsp")
	if _, err := os.Stat(mainPath); err != nil {
		if err := os.WriteFile(mainPath, []byte(initMainContent), 0644); err != nil {
			return err
		}
	}

	fmt.Printf("initialized wisp project in %s\n", cwd)
	return nil
}
