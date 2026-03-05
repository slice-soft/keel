package new

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

var runPromptForm = func(form *huh.Form) error {
	return form.WithTheme(keelTheme).Run()
}

var airInstalledFn = airInstalled
var installAirBinaryFn = installAirBinary

func resolveProjectName(args []string) (string, error) {
	if len(args) > 0 {
		if err := validateProjectName(args[0]); err != nil {
			return "", err
		}
		return strings.TrimSpace(args[0]), nil
	}

	if yesFlag {
		return "", fmt.Errorf("project name is required when using --yes/-y")
	}

	appName := ""
	if err := runPromptForm(huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project name?").
				Placeholder("my-backend").
				Validate(validateProjectName).
				Value(&appName),
		),
	)); err != nil {
		return "", err
	}

	return strings.TrimSpace(appName), nil
}

func promptModulePath(appName string) (string, error) {
	host := "github"
	if err := runPromptForm(huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Where will this module be hosted?").
				Options(
					huh.NewOption("GitHub", "github"),
					huh.NewOption("GitLab", "gitlab"),
					huh.NewOption("Custom domain", "custom"),
					huh.NewOption("Local module", "local"),
				).
				Value(&host),
		),
	)); err != nil {
		return "", err
	}

	var preview string
	var allowLocal bool

	switch host {
	case "github":
		owner := ""
		if err := runPromptForm(huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("GitHub username or organization?").
					Placeholder("slice-soft").
					Validate(validateNonEmpty("GitHub username or organization")).
					Value(&owner),
			),
		)); err != nil {
			return "", err
		}
		preview = fmt.Sprintf("github.com/%s/%s", strings.TrimSpace(owner), appName)

	case "gitlab":
		owner := ""
		if err := runPromptForm(huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("GitLab username or group?").
					Placeholder("slice-soft").
					Validate(validateNonEmpty("GitLab username or group")).
					Value(&owner),
			),
		)); err != nil {
			return "", err
		}
		preview = fmt.Sprintf("gitlab.com/%s/%s", strings.TrimSpace(owner), appName)

	case "custom":
		domain := ""
		if err := runPromptForm(huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Custom domain?").
					Placeholder("code.example.com").
					Validate(validateCustomDomain).
					Value(&domain),
			),
		)); err != nil {
			return "", err
		}
		cleanDomain := strings.Trim(strings.TrimSpace(domain), "/")
		preview = fmt.Sprintf("%s/%s", cleanDomain, appName)

	case "local":
		allowLocal = true
		preview = appName

	default:
		return "", fmt.Errorf("unsupported host option: %s", host)
	}

	return confirmOrEditModulePath(preview, allowLocal)
}

func confirmOrEditModulePath(preview string, allowLocal bool) (string, error) {
	fmt.Println()
	fmt.Println("Module path preview")
	fmt.Println(preview)
	fmt.Println()

	usePreview := true
	if err := runPromptForm(huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Use this module path?").
				Affirmative("Yes").
				Negative("Edit").
				Value(&usePreview),
		),
	)); err != nil {
		return "", err
	}

	if usePreview {
		return preview, nil
	}

	modulePath := preview
	if err := runPromptForm(huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Module path?").
				Placeholder(preview).
				Validate(func(s string) error {
					return validateModulePath(s, allowLocal)
				}).
				Value(&modulePath),
		),
	)); err != nil {
		return "", err
	}

	return strings.TrimSpace(modulePath), nil
}

func promptYesNo(title string, defaultValue bool) (bool, error) {
	choice := defaultValue
	if err := runPromptForm(huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Value(&choice).
				Affirmative("yes").
				Negative("No"),
		),
	)); err != nil {
		return false, err
	}
	return choice, nil
}

func promptAirSetup() (bool, bool, error) {
	useAir, err := promptYesNo("Use Air for hot reload?", true)
	if err != nil {
		return false, false, err
	}
	if !useAir {
		return false, false, nil
	}

	includeAirConfig, err := promptYesNo("Include Air config file (.air.toml)?", true)
	if err != nil {
		return false, false, err
	}

	fmt.Println()
	if airInstalledFn() {
		fmt.Println("  ✓ Air is already installed")
		return true, includeAirConfig, nil
	}

	fmt.Println("  ⚠  Air is not installed on your PATH.")
	installAir, err := promptYesNo("Install Air now? (go install github.com/air-verse/air@latest)", true)
	if err != nil {
		return false, false, err
	}
	if !installAir {
		fmt.Println("  ⚠  Skipping Air installation")
		return true, includeAirConfig, nil
	}

	fmt.Println()
	fmt.Println("  Installing Air...")
	if err := installAirBinaryFn(); err != nil {
		fmt.Printf("  ⚠  failed to install Air: %v\n", err)
		return true, includeAirConfig, nil
	}

	if airInstalledFn() {
		fmt.Println("  ✓ Air installed")
		return true, includeAirConfig, nil
	}

	fmt.Println("  ✓ Air installed (restart your shell if 'air' is not available yet)")
	return true, includeAirConfig, nil
}
