package initcmd

import "github.com/charmbracelet/huh"

var keelTheme = huh.ThemeCharm()

func promptUseAir() (bool, bool, error) {
	useAir := true
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Use Air for hot reload?").
				Value(&useAir).
				Affirmative("Yes").
				Negative("No"),
		),
	).WithTheme(keelTheme).Run(); err != nil {
		return false, false, err
	}

	return useAir, fileExists(".air.toml"), nil
}
