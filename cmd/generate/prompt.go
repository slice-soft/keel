package generate

import "github.com/charmbracelet/huh"

var runGeneratePromptForm = func(form *huh.Form) error {
	return form.WithTheme(huh.ThemeCharm()).Run()
}

func promptRepositoryBackend() (repositoryBackend, error) {
	choice := string(repositoryBackendGorm)
	if err := runGeneratePromptForm(huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Which repository backend do you want to generate?").
				Options(
					huh.NewOption("GORM (SQL)", string(repositoryBackendGorm)),
					huh.NewOption("MongoDB", string(repositoryBackendMongo)),
				).
				Value(&choice),
		),
	)); err != nil {
		return "", err
	}

	return parseRepositoryBackend(choice)
}
