package new

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/huh"
)

func stubPromptInputs(t *testing.T, inputs ...string) {
	t.Helper()
	previousRunPromptForm := runPromptForm
	index := 0
	runPromptForm = func(form *huh.Form) error {
		if index >= len(inputs) {
			t.Fatalf("missing stubbed prompt input for form call #%d", index+1)
		}
		in := strings.NewReader(inputs[index])
		index++
		return form.WithTheme(keelTheme).WithAccessible(true).WithInput(in).WithOutput(io.Discard).Run()
	}
	t.Cleanup(func() {
		runPromptForm = previousRunPromptForm
	})
}

func TestResolveProjectNameInteractive(t *testing.T) {
	previousYesFlag := yesFlag
	t.Cleanup(func() { yesFlag = previousYesFlag })
	yesFlag = false

	stubPromptInputs(t, "my-backend\n")

	got, err := resolveProjectName(nil)
	if err != nil {
		t.Fatalf("resolveProjectName returned error: %v", err)
	}
	if got != "my-backend" {
		t.Fatalf("expected project name my-backend, got %q", got)
	}
}

func TestPromptYesNo(t *testing.T) {
	stubPromptInputs(t, "n\n")
	got, err := promptYesNo("Install dependencies?", true)
	if err != nil {
		t.Fatalf("promptYesNo returned error: %v", err)
	}
	if got {
		t.Fatalf("expected false choice")
	}

	t.Run("returns error when form fails", func(t *testing.T) {
		previousRunPromptForm := runPromptForm
		runPromptForm = func(form *huh.Form) error {
			return errors.New("form failed")
		}
		t.Cleanup(func() {
			runPromptForm = previousRunPromptForm
		})

		_, err := promptYesNo("Install dependencies?", true)
		if err == nil || err.Error() != "form failed" {
			t.Fatalf("expected prompt error, got %v", err)
		}
	})
}

func TestConfirmOrEditModulePath(t *testing.T) {
	t.Run("uses preview path", func(t *testing.T) {
		stubPromptInputs(t, "y\n")
		got, err := confirmOrEditModulePath("github.com/acme/demo", false)
		if err != nil {
			t.Fatalf("confirmOrEditModulePath returned error: %v", err)
		}
		if got != "github.com/acme/demo" {
			t.Fatalf("expected preview path, got %q", got)
		}
	})

	t.Run("edits module path", func(t *testing.T) {
		stubPromptInputs(t, "n\n", "code.example.com/acme/demo\n")
		got, err := confirmOrEditModulePath("github.com/acme/demo", false)
		if err != nil {
			t.Fatalf("confirmOrEditModulePath returned error: %v", err)
		}
		if got != "code.example.com/acme/demo" {
			t.Fatalf("expected edited path, got %q", got)
		}
	})
}

func TestPromptModulePath(t *testing.T) {
	t.Run("github", func(t *testing.T) {
		stubPromptInputs(t, "1\n", "slice-soft\n", "y\n")
		got, err := promptModulePath("my-backend")
		if err != nil {
			t.Fatalf("promptModulePath returned error: %v", err)
		}
		if got != "github.com/slice-soft/my-backend" {
			t.Fatalf("unexpected module path: %q", got)
		}
	})

	t.Run("gitlab", func(t *testing.T) {
		stubPromptInputs(t, "2\n", "acme-group\n", "y\n")
		got, err := promptModulePath("my-backend")
		if err != nil {
			t.Fatalf("promptModulePath returned error: %v", err)
		}
		if got != "gitlab.com/acme-group/my-backend" {
			t.Fatalf("unexpected module path: %q", got)
		}
	})

	t.Run("custom domain", func(t *testing.T) {
		stubPromptInputs(t, "3\n", "code.example.com\n", "y\n")
		got, err := promptModulePath("my-backend")
		if err != nil {
			t.Fatalf("promptModulePath returned error: %v", err)
		}
		if got != "code.example.com/my-backend" {
			t.Fatalf("unexpected module path: %q", got)
		}
	})

	t.Run("local module", func(t *testing.T) {
		stubPromptInputs(t, "4\n", "y\n")
		got, err := promptModulePath("my-backend")
		if err != nil {
			t.Fatalf("promptModulePath returned error: %v", err)
		}
		if got != "my-backend" {
			t.Fatalf("unexpected module path: %q", got)
		}
	})
}

func TestPromptAirSetup(t *testing.T) {
	previousAirInstalledFn := airInstalledFn
	previousInstallAirBinaryFn := installAirBinaryFn
	t.Cleanup(func() {
		airInstalledFn = previousAirInstalledFn
		installAirBinaryFn = previousInstallAirBinaryFn
	})

	t.Run("use air disabled", func(t *testing.T) {
		stubPromptInputs(t, "n\n")
		useAir, includeAirConfig, err := promptAirSetup()
		if err != nil {
			t.Fatalf("promptAirSetup returned error: %v", err)
		}
		if useAir || includeAirConfig {
			t.Fatalf("expected both values false when air is disabled")
		}
	})

	t.Run("returns error when first prompt fails", func(t *testing.T) {
		previousRunPromptForm := runPromptForm
		runPromptForm = func(form *huh.Form) error {
			return errors.New("prompt failed")
		}
		t.Cleanup(func() {
			runPromptForm = previousRunPromptForm
		})

		_, _, err := promptAirSetup()
		if err == nil || err.Error() != "prompt failed" {
			t.Fatalf("expected prompt error, got %v", err)
		}
	})

	t.Run("air already installed", func(t *testing.T) {
		airInstalledFn = func() bool { return true }
		stubPromptInputs(t, "y\n", "y\n")
		useAir, includeAirConfig, err := promptAirSetup()
		if err != nil {
			t.Fatalf("promptAirSetup returned error: %v", err)
		}
		if !useAir || !includeAirConfig {
			t.Fatalf("expected both values true")
		}
	})

	t.Run("skip installation when air missing", func(t *testing.T) {
		airInstalledFn = func() bool { return false }
		stubPromptInputs(t, "y\n", "y\n", "n\n")
		useAir, includeAirConfig, err := promptAirSetup()
		if err != nil {
			t.Fatalf("promptAirSetup returned error: %v", err)
		}
		if !useAir || !includeAirConfig {
			t.Fatalf("expected air to stay enabled with config included")
		}
	})

	t.Run("attempt install and continue on error", func(t *testing.T) {
		airInstalledFn = func() bool { return false }
		installAirBinaryFn = func() error { return errors.New("install failed") }
		stubPromptInputs(t, "y\n", "y\n", "y\n")
		useAir, includeAirConfig, err := promptAirSetup()
		if err != nil {
			t.Fatalf("promptAirSetup returned error: %v", err)
		}
		if !useAir || !includeAirConfig {
			t.Fatalf("expected air setup choices to remain enabled")
		}
	})

	t.Run("attempt install and detect success after install", func(t *testing.T) {
		checks := 0
		airInstalledFn = func() bool {
			checks++
			return checks > 1
		}
		installCalled := false
		installAirBinaryFn = func() error {
			installCalled = true
			return nil
		}
		stubPromptInputs(t, "y\n", "y\n", "y\n")
		useAir, includeAirConfig, err := promptAirSetup()
		if err != nil {
			t.Fatalf("promptAirSetup returned error: %v", err)
		}
		if !installCalled {
			t.Fatalf("expected installAirBinary to be called")
		}
		if !useAir || !includeAirConfig {
			t.Fatalf("expected air setup choices to remain enabled")
		}
	})
}
