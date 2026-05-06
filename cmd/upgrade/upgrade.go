package upgrade

import (
	"github.com/slice-soft/keel/internal/updater"
	"github.com/spf13/cobra"
)

var upgradeFn = updater.Upgrade

func NewCommand(currentVersion func() string) *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Update Keel CLI",
		Long: `Update Keel CLI for the detected installation source.

If Keel was installed with Homebrew or go install, this command runs the
package-manager update command. If the installation source cannot be detected,
this command asks you to update manually.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return upgradeFn(currentVersion())
		},
	}
}
