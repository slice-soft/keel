package initcmd

import "github.com/spf13/cobra"

var validateKeelConfigDoesNotExistFn = validateKeelConfigDoesNotExist
var promptUseAirFn = promptUseAir
var ensureAirReadyFn = ensureAirReady
var generateKeelConfigFn = generateKeelConfig

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

	if err := validateKeelConfigDoesNotExistFn(keelConfigPath); err != nil {
		return err
	}

	useAir, airConfigExists, err := promptUseAirFn()
	if err != nil {
		return err
	}

	if err := ensureAirReadyFn(useAir); err != nil {
		return err
	}

	return generateKeelConfigFn(keelConfigPath, useAir, airConfigExists)
}
