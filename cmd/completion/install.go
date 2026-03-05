package completion

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func runInstall(root *cobra.Command) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to resolve home directory: %w", err)
	}

	shell, err := resolveShell(homeDir)
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ Detected shell: %s\n", shell)

	script, err := generateCompletionScript(root, shell)
	if err != nil {
		return err
	}

	scriptPath, err := writeCompletionScript(homeDir, shell, script)
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ Installed script: %s\n", scriptPath)

	configPath, err := resolveConfigFile(shell, homeDir)
	if err != nil {
		return err
	}

	if configPath == "" {
		fmt.Println("  ✓ No shell config update required")
		return nil
	}

	if err := ensureSourceLine(configPath, sourceLineForShell(shell, scriptPath)); err != nil {
		return err
	}
	fmt.Printf("  ✓ Updated shell config: %s\n", configPath)

	return nil
}

func generateCompletionScript(root *cobra.Command, shell string) (string, error) {
	var out bytes.Buffer

	switch shell {
	case "zsh":
		if err := root.GenZshCompletion(&out); err != nil {
			return "", fmt.Errorf("failed to generate zsh completion: %w", err)
		}
	case "bash":
		if err := root.GenBashCompletionV2(&out, true); err != nil {
			return "", fmt.Errorf("failed to generate bash completion: %w", err)
		}
	case "fish":
		if err := root.GenFishCompletion(&out, true); err != nil {
			return "", fmt.Errorf("failed to generate fish completion: %w", err)
		}
	case "powershell":
		if err := root.GenPowerShellCompletionWithDesc(&out); err != nil {
			return "", fmt.Errorf("failed to generate powershell completion: %w", err)
		}
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}

	return out.String(), nil
}

func writeCompletionScript(homeDir, shell, content string) (string, error) {
	path := filepath.Join(homeDir, ".config", "keel", "completion", "keel."+shell)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("failed to create completion directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write completion script: %w", err)
	}
	return path, nil
}

func sourceLineForShell(shell, scriptPath string) string {
	escapedPath := strings.ReplaceAll(scriptPath, "\"", "\\\"")
	if shell == "fish" {
		return "source \"" + escapedPath + "\""
	}
	return "source \"" + escapedPath + "\""
}

func ensureSourceLine(configPath, sourceLine string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to ensure shell config directory: %w", err)
	}

	existing := ""
	if fileExists(configPath) {
		content, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read shell config %s: %w", configPath, err)
		}
		existing = string(content)
		if strings.Contains(existing, sourceLine) {
			return nil
		}
	}

	f, err := os.OpenFile(configPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open shell config %s: %w", configPath, err)
	}
	defer f.Close()

	if existing != "" && !strings.HasSuffix(existing, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write newline to %s: %w", configPath, err)
		}
	}

	entry := "# Keel CLI completion\n" + sourceLine + "\n"
	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("failed to update shell config %s: %w", configPath, err)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
