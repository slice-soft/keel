package run

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

	t.Run("invalid path", func(t *testing.T) {
		err := ensureKeelConfigExists("bad\x00path")
		if err == nil || !strings.Contains(err.Error(), "failed to access") {
			t.Fatalf("expected access error for invalid path, got %v", err)
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

	t.Run("invalid config path", func(t *testing.T) {
		_, err := loadScriptsFromConfig("bad\x00path")
		if err == nil || !strings.Contains(err.Error(), "failed to read keel.toml") {
			t.Fatalf("expected read config error, got %v", err)
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

	_, err = findScriptCommand(map[string]string{"empty": ""}, "empty")
	if err == nil {
		t.Fatalf("expected error for empty script command")
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

func TestNewCommand(t *testing.T) {
	cmd := NewCommand()

	if cmd.Use != "run [script]" {
		t.Fatalf("unexpected use: %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Fatalf("expected short description to be set")
	}
	if cmd.RunE == nil {
		t.Fatalf("expected RunE handler to be configured")
	}
}

func TestRunScript(t *testing.T) {
	t.Run("runs configured script", func(t *testing.T) {
		dir := t.TempDir()
		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get wd: %v", err)
		}
		defer func() { _ = os.Chdir(oldWD) }()

		if err := os.Chdir(dir); err != nil {
			t.Fatalf("failed to change directory: %v", err)
		}

		script := "printf ok > result.txt"
		if runtime.GOOS == "windows" {
			script = "echo ok > result.txt"
		}

		config := "[scripts]\nwrite=\"" + script + "\"\n"
		if err := os.WriteFile("keel.toml", []byte(config), 0644); err != nil {
			t.Fatalf("failed to create keel.toml: %v", err)
		}

		if err := runScript(nil, []string{"write"}); err != nil {
			t.Fatalf("runScript returned error: %v", err)
		}

		output, err := os.ReadFile("result.txt")
		if err != nil {
			t.Fatalf("expected script side effect file: %v", err)
		}
		if !strings.Contains(string(output), "ok") {
			t.Fatalf("unexpected script output content: %q", string(output))
		}
	})

	t.Run("fails when keel.toml does not exist", func(t *testing.T) {
		dir := t.TempDir()
		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get wd: %v", err)
		}
		defer func() { _ = os.Chdir(oldWD) }()

		if err := os.Chdir(dir); err != nil {
			t.Fatalf("failed to change directory: %v", err)
		}

		err = runScript(nil, []string{"dev"})
		if err == nil || !strings.Contains(err.Error(), "keel.toml not found") {
			t.Fatalf("expected missing keel.toml error, got %v", err)
		}
	})

	t.Run("fails when requested script is missing", func(t *testing.T) {
		dir := t.TempDir()
		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get wd: %v", err)
		}
		defer func() { _ = os.Chdir(oldWD) }()

		if err := os.Chdir(dir); err != nil {
			t.Fatalf("failed to change directory: %v", err)
		}

		config := "[scripts]\ndev=\"echo hello\"\n"
		if err := os.WriteFile("keel.toml", []byte(config), 0644); err != nil {
			t.Fatalf("failed to create keel.toml: %v", err)
		}

		err = runScript(nil, []string{"build"})
		if err == nil || !strings.Contains(err.Error(), "does not exist") {
			t.Fatalf("expected missing script error, got %v", err)
		}
	})
}
