package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

var keelTheme = huh.ThemeCharm()

// promptName shows an interactive input for the given label/placeholder
// and writes the result into dest.
func promptName(label, placeholder string, dest *string) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(label).
				Placeholder(placeholder).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("name cannot be empty")
					}
					return nil
				}).
				Value(dest),
		),
	).WithTheme(keelTheme).Run()
}

// resolveName returns args[0] if provided, otherwise prompts the user.
// Returns an error if no name can be determined.
func resolveName(args []string, label, placeholder string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	if yesFlag {
		return "", fmt.Errorf("name is required (use: keel g <type> <name>)")
	}
	name := ""
	if err := promptName(label, placeholder, &name); err != nil {
		return "", err
	}
	return name, nil
}
