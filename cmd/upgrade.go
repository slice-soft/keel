package cmd

import (
	"fmt"
	"runtime"

	"github.com/charmbracelet/huh"
	"github.com/slice-soft/keel/internal/updater"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade keel to the latest version",
	Long: `Downloads and installs the latest version of keel
from the official GitHub releases.

The current binary is replaced atomically — if anything fails
the previous version is automatically restored.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		confirm := false
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Upgrade keel to the latest version?").
					Description("Current: " + version).
					Value(&confirm),
			),
		).WithTheme(keelTheme).Run(); err != nil {
			return err
		}

		if !confirm {
			fmt.Println("  Upgrade cancelled.")
			return nil
		}

		return updater.Upgrade(version)
	},
}

func init() {
	rootCmd.SetVersionTemplate(fmt.Sprintf(`
  ⚓  keel %s
  keel-go.dev
  github.com/slice-soft/keel-cli

`, "{{.Version}}"))

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show the installed keel version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("\n  ⚓  keel %s\n", version)
			fmt.Printf("  OS/Arch : %s/%s\n", runtime.GOOS, runtime.GOARCH)
			fmt.Printf("  Site    : keel-go.dev\n\n")
		},
	})
}
