package initcmd

import "github.com/charmbracelet/huh"

var keelTheme = huh.ThemeCharm()
var runInitPromptForm = func(form *huh.Form) error {
	return form.WithTheme(keelTheme).Run()
}

func promptUseAir() (bool, bool, error) {
	useAir := true
	if err := runInitPromptForm(huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Use Air for hot reload?").
				Value(&useAir).
				Affirmative("Yes").
				Negative("No"),
		),
	)); err != nil {
		return false, false, err
	}

	return useAir, fileExists(".air.toml"), nil
}
