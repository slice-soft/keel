package completion

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
)

var shellConfigCandidates = map[string][]string{
	"zsh":  {".zshrc", ".zprofile"},
	"bash": {".bashrc", ".bash_profile", ".profile"},
	"fish": {".config/fish/config.fish"},
}

func detectShellFromEnv(shellPath string) string {
	name := strings.ToLower(filepath.Base(shellPath))
	switch name {
	case "zsh", "bash", "fish":
		return name
	default:
		return ""
	}
}

func detectAvailableShells(homeDir string) []string {
	ordered := []string{"zsh", "bash", "fish"}
	available := make([]string, 0, len(ordered))

	for _, shell := range ordered {
		if shellConfigExists(shell, homeDir) {
			available = append(available, shell)
		}
	}

	return available
}

func shellConfigExists(shell, homeDir string) bool {
	for _, rel := range shellConfigCandidates[shell] {
		if fileExists(filepath.Join(homeDir, rel)) {
			return true
		}
	}
	return false
}

func resolveShell(homeDir string) (string, error) {
	detected := detectShellFromEnv(os.Getenv("SHELL"))
	available := detectAvailableShells(homeDir)

	options := mergeShellOptions(detected, available)
	if len(options) == 0 {
		return "zsh", nil
	}
	if len(options) == 1 {
		return options[0], nil
	}

	if !isInteractiveTerminal() && detected != "" {
		return detected, nil
	}

	return promptSelectShell(options)
}

func mergeShellOptions(detected string, available []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 3)

	if detected != "" {
		seen[detected] = true
		out = append(out, detected)
	}

	for _, item := range available {
		if !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}

	return out
}

func promptSelectShell(options []string) (string, error) {
	selected := options[0]
	huhOptions := make([]huh.Option[string], 0, len(options))
	for _, option := range options {
		huhOptions = append(huhOptions, huh.NewOption(strings.ToUpper(option), option))
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Multiple shells detected. Which shell should Keel configure?").
				Options(huhOptions...).
				Value(&selected),
		),
	).Run()
	if err != nil {
		return "", err
	}

	return selected, nil
}

func resolveConfigFile(shell, homeDir string) (string, error) {
	relativeCandidates := shellConfigCandidates[shell]
	if len(relativeCandidates) == 0 {
		return "", nil
	}

	absCandidates := make([]string, 0, len(relativeCandidates))
	existing := make([]string, 0, len(relativeCandidates))
	for _, rel := range relativeCandidates {
		abs := filepath.Join(homeDir, rel)
		absCandidates = append(absCandidates, abs)
		if fileExists(abs) {
			existing = append(existing, abs)
		}
	}

	if len(existing) == 1 {
		return existing[0], nil
	}

	if len(existing) > 1 {
		if !isInteractiveTerminal() {
			return existing[0], nil
		}
		return promptSelectConfigFile(shell, existing)
	}

	return absCandidates[0], nil
}

func promptSelectConfigFile(shell string, files []string) (string, error) {
	selected := files[0]
	huhOptions := make([]huh.Option[string], 0, len(files))
	for _, file := range files {
		huhOptions = append(huhOptions, huh.NewOption(file, file))
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Multiple " + strings.ToUpper(shell) + " config files found. Which one should be updated?").
				Options(huhOptions...).
				Value(&selected),
		),
	).Run()
	if err != nil {
		return "", err
	}

	return selected, nil
}

func isInteractiveTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
