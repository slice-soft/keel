package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/slice-soft/keel/cmd/add"
	"github.com/slice-soft/keel/cmd/completion"
	"github.com/slice-soft/keel/cmd/doctor"
	envCmd "github.com/slice-soft/keel/cmd/env"
	"github.com/slice-soft/keel/cmd/generate"
	initcmd "github.com/slice-soft/keel/cmd/init"
	"github.com/slice-soft/keel/cmd/new"
	"github.com/slice-soft/keel/cmd/run"
	telemetrycmd "github.com/slice-soft/keel/cmd/telemetry"
	"github.com/slice-soft/keel/internal/telemetry"
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
		telemetry.Send(cmd.Name(), version)
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
var stderrWriter io.Writer = os.Stderr
var exitFn = os.Exit

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(stderrWriter, err)
		exitFn(1)
	}
}

func init() {
	syncRootVersionOutput()
	rootCmd.AddCommand(add.NewCommand())
	rootCmd.AddCommand(new.NewCommand())
	rootCmd.AddCommand(initcmd.NewCommand())
	rootCmd.AddCommand(generate.NewCommand())
	rootCmd.AddCommand(completion.NewCommand(rootCmd))
	rootCmd.AddCommand(run.NewCommand())
	rootCmd.AddCommand(doctor.NewCommand())
	rootCmd.AddCommand(envCmd.NewCommand())
	rootCmd.AddCommand(telemetrycmd.NewCommand())
}

func syncRootVersionOutput() {
	versionOutput := renderVersionOutput(version, commit, buildDate)
	rootCmd.Version = versionOutput
	rootCmd.Long = versionOutput
	rootCmd.SetVersionTemplate("{{.Version}}")
}
