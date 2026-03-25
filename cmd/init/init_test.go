package initcmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateKeelConfigDoesNotExist(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "keel.toml")
		if err := validateKeelConfigDoesNotExist(path); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "keel.toml")
		if err := os.WriteFile(path, []byte("[scripts]\n"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		err := validateKeelConfigDoesNotExist(path)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")

	if fileExists(path) {
		t.Fatalf("expected file to not exist")
	}

	if err := os.WriteFile(path, []byte("ok"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	if !fileExists(path) {
		t.Fatalf("expected file to exist")
	}
}

func TestGenerateKeelConfig(t *testing.T) {
	t.Run("with air and no existing air config creates keel.toml, application.properties and .air.toml", func(t *testing.T) {
		dir := t.TempDir()
		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get wd: %v", err)
		}
		defer func() { _ = os.Chdir(oldWD) }()

		if err := os.Chdir(dir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}

		if err := generateKeelConfig("keel.toml", true, false); err != nil {
			t.Fatalf("generateKeelConfig returned error: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(dir, "keel.toml"))
		if err != nil {
			t.Fatalf("failed to read generated keel.toml: %v", err)
		}
		text := string(content)

		if !strings.Contains(text, "[project]") || !strings.Contains(text, "config  = \"application.properties\"") {
			t.Fatalf("expected [project] section in init mode, got: %s", text)
		}
		if !strings.Contains(text, "dev   = \"air\"") {
			t.Fatalf("expected air dev script in generated file, got: %s", text)
		}

		if _, err := os.Stat(filepath.Join(dir, "application.properties")); err != nil {
			t.Fatalf("expected application.properties to be generated: %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, ".air.toml")); err != nil {
			t.Fatalf("expected .air.toml to be generated: %v", err)
		}
	})

	t.Run("with air and existing air config does not regenerate .air.toml", func(t *testing.T) {
		dir := t.TempDir()
		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get wd: %v", err)
		}
		defer func() { _ = os.Chdir(oldWD) }()

		if err := os.Chdir(dir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}

		airPath := filepath.Join(dir, ".air.toml")
		if err := os.WriteFile(airPath, []byte("custom-air-config"), 0644); err != nil {
			t.Fatalf("failed to seed .air.toml: %v", err)
		}

		if err := generateKeelConfig("keel.toml", true, true); err != nil {
			t.Fatalf("generateKeelConfig returned error: %v", err)
		}

		airContent, err := os.ReadFile(airPath)
		if err != nil {
			t.Fatalf("failed to read .air.toml: %v", err)
		}
		if string(airContent) != "custom-air-config" {
			t.Fatalf("expected existing .air.toml to remain untouched")
		}
	})

	t.Run("without air creates keel.toml and application.properties only", func(t *testing.T) {
		dir := t.TempDir()
		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get wd: %v", err)
		}
		defer func() { _ = os.Chdir(oldWD) }()

		if err := os.Chdir(dir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}

		if err := generateKeelConfig("keel.toml", false, false); err != nil {
			t.Fatalf("generateKeelConfig returned error: %v", err)
		}

		if _, err := os.Stat(filepath.Join(dir, ".air.toml")); !os.IsNotExist(err) {
			t.Fatalf("did not expect .air.toml to be generated")
		}
		if _, err := os.Stat(filepath.Join(dir, "application.properties")); err != nil {
			t.Fatalf("expected application.properties to be generated: %v", err)
		}
	})
}

func TestBuildInitFiles(t *testing.T) {
	t.Run("with air and no existing config", func(t *testing.T) {
		files := buildInitFiles("keel.toml", true, false, false)
		if len(files) != 3 {
			t.Fatalf("expected 3 files with air and no config, got %d", len(files))
		}
	})

	t.Run("with air and existing config", func(t *testing.T) {
		files := buildInitFiles("keel.toml", true, true, true)
		if len(files) != 1 {
			t.Fatalf("expected 1 file when .air.toml and application.properties already exist, got %d", len(files))
		}
	})

	t.Run("without air", func(t *testing.T) {
		files := buildInitFiles("keel.toml", false, false, false)
		if len(files) != 2 {
			t.Fatalf("expected 2 files without air, got %d", len(files))
		}
	})

	t.Run("without air and existing application properties", func(t *testing.T) {
		files := buildInitFiles("keel.toml", false, false, true)
		if len(files) != 1 {
			t.Fatalf("expected 1 file when application.properties already exists, got %d", len(files))
		}
	})
}

func TestCurrentDirName(t *testing.T) {
	t.Run("returns app when working directory is filesystem root", func(t *testing.T) {
		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get wd: %v", err)
		}
		defer func() { _ = os.Chdir(oldWD) }()

		if err := os.Chdir(string(filepath.Separator)); err != nil {
			t.Fatalf("failed to chdir to filesystem root: %v", err)
		}

		got, err := currentDirName()
		if err != nil {
			t.Fatalf("currentDirName returned error: %v", err)
		}
		if got != "app" {
			t.Fatalf("expected app for root directory, got %q", got)
		}
	})

	t.Run("returns error when cwd cannot be resolved", func(t *testing.T) {
		previousGetwdFn := getwdFn
		getwdFn = func() (string, error) {
			return "", errors.New("wd failure")
		}
		t.Cleanup(func() {
			getwdFn = previousGetwdFn
		})

		_, err := currentDirName()
		if err == nil {
			t.Fatalf("expected error when cwd was removed")
		}
	})
}

func TestGenerateKeelConfigErrors(t *testing.T) {
	t.Run("returns error for invalid destination path", func(t *testing.T) {
		dir := t.TempDir()
		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get wd: %v", err)
		}
		defer func() { _ = os.Chdir(oldWD) }()

		if err := os.Chdir(dir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}

		err = generateKeelConfig("bad\x00path", true, false)
		if err == nil {
			t.Fatalf("expected generateKeelConfig error for invalid destination")
		}
	})
}
