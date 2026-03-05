package new

import (
	"fmt"

	"github.com/spf13/cobra"
)

var withoutStarterModule bool
var withFolderStructure bool
var yesFlag bool

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "new [project-name]",
		Aliases: []string{"n"},
		Short:   "Create a new Keel project",
		Args:    cobra.MaximumNArgs(1),
		RunE:    runNew,
	}

	cmd.Flags().BoolVar(
		&withoutStarterModule,
		"without-starter-module",
		false,
		"Skip creating the default 'starter' module (for advanced users)",
	)

	cmd.Flags().BoolVar(
		&withFolderStructure,
		"with-folder-structure",
		false,
		"Use a more opinionated folder structure with separate directories for middleware, guards, scheduler, checkers, events, and hooks (instead of a flat 'internal' directory)",
	)

	cmd.Flags().BoolVarP(
		&yesFlag,
		"yes",
		"y",
		false,
		"Skip interactive prompts and use defaults",
	)
	return cmd
}

func runNew(cmd *cobra.Command, args []string) error {
	printWelcome()

	setup, err := collectProjectSetup(args)
	if err != nil {
		return err
	}

	if err := scaffoldProject(setup); err != nil {
		return err
	}

	runPostSetup(setup)
	printProjectReady(setup.appName)
	return nil
}

func printWelcome() {
	fmt.Println()
	fmt.Println("Welcome to Keel!")
	fmt.Println()
}

func printProjectReady(appName string) {
	fmt.Printf(`
  ✅ Project '%s' ready!

  Next steps:
    cd %s
    keel run dev

`, appName, appName)
}
