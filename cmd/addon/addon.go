package addon

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addon",
		Short: "Manage Keel addons in the current project",
	}
	cmd.AddCommand(newRemoveCommand())
	cmd.AddCommand(newUpgradeCommand())
	return cmd
}
