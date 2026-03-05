package initcmd

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/huh"
)

func stubInitPromptInputs(t *testing.T, inputs ...string) {
	t.Helper()
	previous := runInitPromptForm
	index := 0
	runInitPromptForm = func(form *huh.Form) error {
		if index >= len(inputs) {
			t.Fatalf("missing prompt input for call #%d", index+1)
		}
		in := strings.NewReader(inputs[index])
		index++
		return form.WithTheme(keelTheme).WithAccessible(true).WithInput(in).WithOutput(io.Discard).Run()
	}
	t.Cleanup(func() {
		runInitPromptForm = previous
	})
}

func TestPromptUseAir(t *testing.T) {
	t.Run("selects yes and detects existing air config", func(t *testing.T) {
		dir := t.TempDir()
		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get wd: %v", err)
		}
		defer func() { _ = os.Chdir(oldWD) }()
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}

		if err := os.WriteFile(filepath.Join(dir, ".air.toml"), []byte("seed"), 0644); err != nil {
			t.Fatalf("failed to seed .air.toml: %v", err)
		}

		stubInitPromptInputs(t, "y\n")
		useAir, airConfigExists, err := promptUseAir()
		if err != nil {
			t.Fatalf("promptUseAir returned error: %v", err)
		}
		if !useAir || !airConfigExists {
			t.Fatalf("expected useAir=true and airConfigExists=true")
		}
	})

	t.Run("selects no and reports missing air config", func(t *testing.T) {
		dir := t.TempDir()
		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get wd: %v", err)
		}
		defer func() { _ = os.Chdir(oldWD) }()
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}

		stubInitPromptInputs(t, "n\n")
		useAir, airConfigExists, err := promptUseAir()
		if err != nil {
			t.Fatalf("promptUseAir returned error: %v", err)
		}
		if useAir || airConfigExists {
			t.Fatalf("expected useAir=false and airConfigExists=false")
		}
	})

	t.Run("returns error when prompt form fails", func(t *testing.T) {
		previous := runInitPromptForm
		runInitPromptForm = func(form *huh.Form) error {
			return errors.New("prompt failed")
		}
		t.Cleanup(func() {
			runInitPromptForm = previous
		})

		_, _, err := promptUseAir()
		if err == nil || err.Error() != "prompt failed" {
			t.Fatalf("expected prompt error, got %v", err)
		}
	})
}
