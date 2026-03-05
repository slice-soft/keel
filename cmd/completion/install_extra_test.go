package completion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestGenerateCompletionScript(t *testing.T) {
	root := &cobra.Command{Use: "keel"}

	for _, shell := range []string{"zsh", "bash", "fish", "powershell"} {
		script, err := generateCompletionScript(root, shell)
		if err != nil {
			t.Fatalf("generateCompletionScript(%s) returned error: %v", shell, err)
		}
		if strings.TrimSpace(script) == "" {
			t.Fatalf("expected non-empty script for shell %s", shell)
		}
	}

	_, err := generateCompletionScript(root, "unsupported")
	if err == nil || !strings.Contains(err.Error(), "unsupported shell") {
		t.Fatalf("expected unsupported shell error, got %v", err)
	}
}

func TestRunInstallWritesScriptAndConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SHELL", "/bin/zsh")

	root := &cobra.Command{Use: "keel"}
	if err := runInstall(root); err != nil {
		t.Fatalf("runInstall returned error: %v", err)
	}

	scriptPath := filepath.Join(home, ".config", "keel", "completion", "keel.zsh")
	if !fileExists(scriptPath) {
		t.Fatalf("expected completion script at %s", scriptPath)
	}

	configPath := filepath.Join(home, ".zshrc")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected shell config at %s: %v", configPath, err)
	}

	sourceLine := sourceLineForShell("zsh", scriptPath)
	if !strings.Contains(string(content), sourceLine) {
		t.Fatalf("expected shell config to include source line %q", sourceLine)
	}
}

func TestDetectAvailableShells(t *testing.T) {
	home := t.TempDir()

	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("# zsh"), 0644); err != nil {
		t.Fatalf("failed writing .zshrc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("# bash"), 0644); err != nil {
		t.Fatalf("failed writing .bashrc: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".config", "fish"), 0755); err != nil {
		t.Fatalf("failed creating fish dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "fish", "config.fish"), []byte("# fish"), 0644); err != nil {
		t.Fatalf("failed writing fish config: %v", err)
	}

	available := detectAvailableShells(home)
	got := strings.Join(available, ",")
	if got != "zsh,bash,fish" {
		t.Fatalf("unexpected available shells order: %s", got)
	}
}

func TestResolveShellDefaultsToZshWithoutHints(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SHELL", "/bin/unknown")

	shell, err := resolveShell(home)
	if err != nil {
		t.Fatalf("resolveShell returned error: %v", err)
	}
	if shell != "zsh" {
		t.Fatalf("expected default shell zsh, got %q", shell)
	}
}
