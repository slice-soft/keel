package run

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "run [script]",
		Short:         "Run a script defined in keel.toml",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runScript,
	}

	return cmd
}

func runScript(cmd *cobra.Command, args []string) error {
	const keelConfig = "keel.toml"
	scriptName := args[0]

	if err := ensureKeelConfigExists(keelConfig); err != nil {
		return err
	}

	scripts, err := loadScriptsFromConfig(keelConfig)
	if err != nil {
		return err
	}

	scriptCmd, err := findScriptCommand(scripts, scriptName)
	if err != nil {
		return err
	}

	return executeScript(scriptCmd)
}
