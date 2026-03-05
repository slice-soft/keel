package completion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceLineForShell(t *testing.T) {
	path := "/tmp/keel.zsh"
	line := sourceLineForShell("zsh", path)
	if line != "source \"/tmp/keel.zsh\"" {
		t.Fatalf("unexpected source line: %q", line)
	}
}

func TestEnsureSourceLineIsIdempotent(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".zshrc")
	line := "source \"/tmp/keel.zsh\""

	if err := ensureSourceLine(configPath, line); err != nil {
		t.Fatalf("first ensureSourceLine failed: %v", err)
	}
	if err := ensureSourceLine(configPath, line); err != nil {
		t.Fatalf("second ensureSourceLine failed: %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	text := string(content)
	if strings.Count(text, line) != 1 {
		t.Fatalf("expected source line once, got content: %q", text)
	}
}

func TestWriteCompletionScript(t *testing.T) {
	home := t.TempDir()
	path, err := writeCompletionScript(home, "zsh", "#compdef keel")
	if err != nil {
		t.Fatalf("writeCompletionScript failed: %v", err)
	}

	if !fileExists(path) {
		t.Fatalf("expected script to exist: %s", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read completion script: %v", err)
	}
	if string(content) != "#compdef keel" {
		t.Fatalf("unexpected script content: %q", string(content))
	}
}
