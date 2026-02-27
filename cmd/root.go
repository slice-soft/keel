package cmd

import (
	"fmt"
	"os"

	"github.com/slice-soft/ss-keel-cli/internal/updater"
	"github.com/spf13/cobra"
)

// version is injected at build time via ldflags:
//
//	go build -ldflags "-X github.com/slice-soft/ss-keel-cli/cmd.version=$(jq -r '."."' .release-please-manifest.json)"
//
// Defaults to "dev" for local development.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "keel",
	Version: version,
	Short:   "⚓ Keel CLI — Opinionated Go framework by slice-soft",
	Long: `
  ⚓  K E E L  C L I
  ────────────────────────────────
  Opinionated Go framework by slice-soft
  keel.slice-soft.dev
  ────────────────────────────────`,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Name() == "upgrade" {
			return
		}
		updateCh = updater.CheckAndNotify(version)
	},

	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if updateCh == nil {
			return
		}
		if msg := <-updateCh; msg != "" {
			fmt.Print(msg)
		}
	},
}

var updateCh chan string

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(upgradeCmd)
}
