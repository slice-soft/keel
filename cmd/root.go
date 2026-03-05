package cmd

import (
	"fmt"
	"os"

	"github.com/slice-soft/keel/cmd/completion"
	"github.com/slice-soft/keel/cmd/generate"
	initcmd "github.com/slice-soft/keel/cmd/init"
	"github.com/slice-soft/keel/cmd/new"
	"github.com/slice-soft/keel/cmd/run"
	"github.com/slice-soft/keel/internal/updater"
	"github.com/spf13/cobra"
)

// Build metadata is injected at build time via ldflags.
//
// Defaults are used for local development.
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "keel",
	Short: "⚓ Keel CLI — Opinionated Go framework by slice-soft",

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
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
	syncRootVersionOutput()
	rootCmd.AddCommand(new.NewCommand())
	rootCmd.AddCommand(initcmd.NewCommand())
	rootCmd.AddCommand(generate.NewCommand())
	rootCmd.AddCommand(completion.NewCommand(rootCmd))
	rootCmd.AddCommand(run.NewCommand())
}

func syncRootVersionOutput() {
	versionOutput := renderVersionOutput(version, commit, buildDate)
	rootCmd.Version = versionOutput
	rootCmd.Long = versionOutput
	rootCmd.SetVersionTemplate("{{.Version}}")
}
