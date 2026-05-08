package addon

import (
	"fmt"
	"os"
	"strings"

	"github.com/slice-soft/keel/internal/keeltoml"
)

// Uninstall reverses the installation steps defined in the manifest and removes
// the addon entry from keel.toml.
func Uninstall(m *Manifest) error {
	for i := len(m.Steps) - 1; i >= 0; i-- {
		s := m.Steps[i]
		switch s.Type {
		case "create_provider_file":
			if err := unstepCreateProviderFile(s); err != nil {
				fmt.Printf("  ⚠  could not remove %s: %v\n", s.Filename, err)
			}
		case "main_code":
			if err := unstepMainCode(s); err != nil {
				fmt.Printf("  ⚠  could not remove wiring for %q: %v\n", s.Guard, err)
			}
		case "main_import":
			if err := unstepMainImport(s); err != nil {
				fmt.Printf("  ⚠  could not remove import %s: %v\n", s.Path, err)
			}
		case "env":
			if err := unstepEnv(s); err != nil {
				fmt.Printf("  ⚠  could not remove env key %s: %v\n", s.Key, err)
			}
		case "property":
			if err := unstepProperty(s); err != nil {
				fmt.Printf("  ⚠  could not remove property %s: %v\n", s.Key, err)
			}
		case "go_get":
			if err := unstepGoGet(m.Repo); err != nil {
				fmt.Printf("  ⚠  could not remove Go dependency: %v\n", err)
			}
		case "note":
			// nothing to undo
		}
	}

	id := addonIDForManifest(m)
	removed, err := keeltoml.RemoveAddon(keeltoml.DefaultPath, id)
	if err != nil {
		fmt.Printf("  ⚠  could not update keel.toml: %v\n", err)
	} else if removed {
		fmt.Printf("  → updated keel.toml\n")
	}

	if err := runGoModTidy(); err != nil {
		fmt.Printf("  ⚠  %v\n", err)
	}

	return nil
}

// InstalledVersion reads the version recorded in go.mod for repo.
// Returns "" when the module is not found.
func InstalledVersion(repo string) string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch {
		case len(fields) >= 3 && fields[0] == "require" && fields[1] == repo:
			return fields[2]
		case fields[0] == repo:
			return fields[1]
		}
	}
	return ""
}

func unstepCreateProviderFile(s Step) error {
	if s.Filename == "" {
		return nil
	}
	if s.Guard != "" {
		existing, err := os.ReadFile(s.Filename)
		if err != nil {
			return nil // file doesn't exist; nothing to remove
		}
		if !strings.Contains(string(existing), s.Guard) {
			return nil // guard not found; skip to avoid removing an unrelated file
		}
	}
	if err := os.Remove(s.Filename); err != nil && !os.IsNotExist(err) {
		return err
	}
	fmt.Printf("  → removed %s\n", s.Filename)
	return nil
}

func unstepMainCode(s Step) error {
	if s.Guard == "" {
		return nil
	}
	return updateMainGo(func(content string) string {
		if !strings.Contains(content, s.Guard) {
			return content
		}
		lines := strings.Split(content, "\n")
		var result []string
		for _, line := range lines {
			if strings.Contains(line, s.Guard) {
				fmt.Printf("  → removed %q wiring from cmd/main.go\n", s.Guard)
				continue
			}
			result = append(result, line)
		}
		joined := strings.Join(result, "\n")
		// Collapse triple newlines left by removal.
		for strings.Contains(joined, "\n\n\n") {
			joined = strings.ReplaceAll(joined, "\n\n\n", "\n\n")
		}
		return joined
	})
}

func unstepMainImport(s Step) error {
	if s.Path == "" {
		return nil
	}
	return updateMainGo(func(content string) string {
		importLine := fmt.Sprintf("%q", s.Path)
		lines := strings.Split(content, "\n")
		var result []string
		for _, line := range lines {
			if strings.Contains(line, importLine) {
				fmt.Printf("  → removed import %s from cmd/main.go\n", s.Path)
				continue
			}
			result = append(result, line)
		}
		return strings.Join(result, "\n")
	})
}

func unstepEnv(s Step) error {
	if s.Key == "" {
		return nil
	}
	removed1, err := removeEnvKey(".env", s.Key)
	if err != nil {
		return err
	}
	removed2, err := removeEnvKey(".env.example", s.Key)
	if err != nil {
		return err
	}
	if removed1 || removed2 {
		fmt.Printf("  → removed %s from env files\n", s.Key)
	}
	return nil
}

func unstepProperty(s Step) error {
	if s.Key == "" {
		return nil
	}
	removed, err := removePropertyKey("application.properties", s.Key)
	if err != nil {
		return err
	}
	if removed {
		fmt.Printf("  → removed %s from application.properties\n", s.Key)
	}
	return nil
}

func unstepGoGet(repo string) error {
	if repo == "" {
		return nil
	}
	fmt.Printf("  → go get %s@none\n", repo)
	cmd := execCommand("go", "get", repo+"@none")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go get %s@none: %w", repo, err)
	}
	return nil
}

func removeEnvKey(filename, key string) (bool, error) {
	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(data), "\n")
	var result []string
	removed := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") {
			removed = true
			continue
		}
		result = append(result, line)
	}

	if !removed {
		return false, nil
	}

	return true, os.WriteFile(filename, []byte(strings.Join(result, "\n")), 0644)
}

func removePropertyKey(filename, key string) (bool, error) {
	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(data), "\n")
	var result []string
	removed := false
	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Check if this is a comment followed by our key.
		if strings.HasPrefix(trimmed, "#") && i+1 < len(lines) {
			nextTrimmed := strings.TrimSpace(lines[i+1])
			sep := strings.IndexAny(nextTrimmed, "=:")
			if sep != -1 && strings.TrimSpace(nextTrimmed[:sep]) == key {
				removed = true
				i += 2 // skip comment + key line
				continue
			}
		}

		sep := strings.IndexAny(trimmed, "=:")
		if sep != -1 && strings.TrimSpace(trimmed[:sep]) == key {
			removed = true
			i++
			continue
		}

		result = append(result, line)
		i++
	}

	if !removed {
		return false, nil
	}

	return true, os.WriteFile(filename, []byte(strings.Join(result, "\n")), 0644)
}
