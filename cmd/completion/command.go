package completion

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var runInstallCommandFn = runInstall

func NewCommand(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Manage shell completion",
	}

	cmd.AddCommand(newGenerateCommand(root, "bash"))
	cmd.AddCommand(newGenerateCommand(root, "fish"))
	cmd.AddCommand(newGenerateCommand(root, "powershell"))
	cmd.AddCommand(newGenerateCommand(root, "zsh"))
	cmd.AddCommand(newInstallCommand(root))
	return cmd
}

func newGenerateCommand(root *cobra.Command, shell string) *cobra.Command {
	return &cobra.Command{
		Use:   shell,
		Short: fmt.Sprintf("Generate the autocompletion script for %s", shell),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			script, err := generateCompletionScript(root, shell)
			if err != nil {
				return err
			}
			_, err = os.Stdout.WriteString(script)
			return err
		},
	}
}

func newInstallCommand(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:          "install",
		Short:        "Detect shell and install completion",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstallCommandFn(root)
		},
	}
}
