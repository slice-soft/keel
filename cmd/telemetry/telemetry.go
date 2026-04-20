package telemetry

import (
	"fmt"

	"github.com/slice-soft/keel/internal/telemetry"
	"github.com/spf13/cobra"
)

// NewCommand builds the `keel telemetry` parent command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Manage anonymous usage data collection",
		Long: `Commands to check and control whether Keel sends anonymous usage data.

  keel telemetry status   — show current setting
  keel telemetry enable   — turn on data collection
  keel telemetry disable  — turn off data collection

You can also set KEEL_TELEMETRY=off in your environment to disable permanently,
or edit ~/.keel/config.json directly.`,
	}
	cmd.AddCommand(newStatusCommand())
	cmd.AddCommand(newEnableCommand())
	cmd.AddCommand(newDisableCommand())
	return cmd
}

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show whether telemetry is enabled",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if telemetry.IsEnabled() {
				fmt.Println("  ✓  telemetry is enabled")
				fmt.Println("     Opt out: keel telemetry disable  or  KEEL_TELEMETRY=off")
			} else {
				fmt.Println("  ✗  telemetry is disabled")
				fmt.Println("     Opt in:  keel telemetry enable")
			}
			return nil
		},
	}
}

func newEnableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable anonymous usage data collection",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := telemetry.Enable(); err != nil {
				return err
			}
			fmt.Println("  ✓  telemetry enabled — thank you!")
			return nil
		},
	}
}

func newDisableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable anonymous usage data collection",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := telemetry.Disable(); err != nil {
				return err
			}
			fmt.Println("  ✓  telemetry disabled")
			fmt.Println("     You can also set KEEL_TELEMETRY=off in your shell profile.")
			return nil
		},
	}
}
