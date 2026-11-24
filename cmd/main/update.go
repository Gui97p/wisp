package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Gui97p/wisp/internal/update"
	"github.com/Gui97p/wisp/internal/version"
	"github.com/spf13/cobra"
)

var (
	updateCheck   bool
	updateVersion string
)

var updateCmd = &cobra.Command{
	Use:          "update",
	Short:        "Install the latest release of wisp",
	Args:         cobra.NoArgs,
	RunE:         runUpdate,
	SilenceUsage: true,
}

func init() {
	if base := os.Getenv("WISP_UPDATE_URL"); base != "" {
		update.BaseURL = base
	}
	updateCmd.Flags().BoolVar(&updateCheck, "check", false, "only say whether a newer release exists")
	updateCmd.Flags().StringVar(&updateVersion, "version", "", "install this version instead of the latest one, even if it is older")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	root, err := update.Root(exe)
	if err != nil && !updateCheck {
		return err
	}
	if err == nil {
		update.Cleanup(root)
	}

	var release update.Release
	if updateVersion != "" {
		release = update.ForTag(updateVersion)
	} else {
		release, err = update.Latest()
		if err != nil {
			return err
		}
	}

	current := version.Version()
	cmp := update.Compare(release.Version, current)
	if updateCheck {
		switch {
		case cmp > 0:
			fmt.Printf("wisp %s is available (this is %s): run `wisp update`\n", release.Version, current)
		default:
			fmt.Printf("wisp %s is up to date\n", current)
		}
		return nil
	}
	if updateVersion == "" && cmp <= 0 {
		fmt.Printf("wisp %s is up to date\n", current)
		return nil
	}

	name, err := update.AssetName(release.Version, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	staged, err := update.Stage(root)
	if err != nil {
		return err
	}
	defer os.RemoveAll(staged)

	archive := filepath.Join(staged, name)
	sums := filepath.Join(staged, update.SumsName)

	fmt.Printf("updating wisp %s -> %s\n", current, release.Version)
	if err := update.Download(update.AssetURL(release, update.SumsName), sums, nil); err != nil {
		return err
	}
	last := -1
	progress := func(done, total int64) {
		if total <= 0 {
			return
		}
		if percent := int(done * 100 / total); percent/10 != last/10 {
			last = percent
			fmt.Printf("\rdownloading %s  %3d%%", name, percent)
		}
	}
	if err := update.Download(update.AssetURL(release, name), archive, progress); err != nil {
		return err
	}
	fmt.Println()
	if err := update.Verify(sums, name, archive); err != nil {
		return err
	}

	fresh := filepath.Join(staged, "files")
	if err := update.Extract(archive, fresh); err != nil {
		return err
	}
	if err := update.Install(root, fresh); err != nil {
		return fmt.Errorf("the installation was left as it was (%w)", err)
	}
	fmt.Printf("wisp %s installed in %s\n", release.Version, root)
	return nil
}
