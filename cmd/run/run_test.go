package run

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnsureKeelConfigExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "keel.toml")
		if err := os.WriteFile(path, []byte("[scripts]\ndev=\"go test ./...\"\n"), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		if err := ensureKeelConfigExists(path); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("missing", func(t *testing.T) {
		err := ensureKeelConfigExists(filepath.Join(t.TempDir(), "keel.toml"))
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestLoadScriptsFromConfig(t *testing.T) {
	t.Run("loads scripts", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "keel.toml")
		content := "[scripts]\ndev=\"go run ./cmd/main.go\"\ntest=\"go test ./...\"\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		scripts, err := loadScriptsFromConfig(path)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if scripts["dev"] != "go run ./cmd/main.go" {
			t.Fatalf("unexpected script value: %q", scripts["dev"])
		}
	})

	t.Run("no scripts", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "keel.toml")
		if err := os.WriteFile(path, []byte("[app]\nname=\"x\"\n"), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		_, err := loadScriptsFromConfig(path)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestFindScriptCommand(t *testing.T) {
	scripts := map[string]string{"dev": "go run ./cmd/main.go"}

	cmd, err := findScriptCommand(scripts, "dev")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cmd != "go run ./cmd/main.go" {
		t.Fatalf("unexpected script command: %q", cmd)
	}

	_, err = findScriptCommand(scripts, "build")
	if err == nil {
		t.Fatalf("expected error for missing script")
	}
}

func TestShellCommand(t *testing.T) {
	cmd := shellCommand("echo hello")
	if runtime.GOOS == "windows" {
		if cmd.Path == "" || len(cmd.Args) < 3 || cmd.Args[1] != "/C" {
			t.Fatalf("unexpected windows shell command: %#v", cmd.Args)
		}
		return
	}

	if cmd.Path == "" || len(cmd.Args) < 3 || cmd.Args[1] != "-c" {
		t.Fatalf("unexpected unix shell command: %#v", cmd.Args)
	}
}
