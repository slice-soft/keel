package initcmd

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a keel.toml in the current project",
		Args:  cobra.NoArgs,
		RunE:  runInit,
	}

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	const keelConfigPath = "keel.toml"

	if err := validateKeelConfigDoesNotExist(keelConfigPath); err != nil {
		return err
	}

	useAir, airConfigExists, err := promptUseAir()
	if err != nil {
		return err
	}

	if err := ensureAirReady(useAir); err != nil {
		return err
	}

	return generateKeelConfig(keelConfigPath, useAir, airConfigExists)
}
