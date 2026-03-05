package initcmd

import (
	"errors"
	"testing"
)

func TestNewCommandConfiguration(t *testing.T) {
	cmd := NewCommand()
	if cmd.Use != "init" {
		t.Fatalf("unexpected command use: %q", cmd.Use)
	}
	if cmd.RunE == nil {
		t.Fatalf("expected RunE to be configured")
	}
}

func TestRunInit(t *testing.T) {
	previousValidate := validateKeelConfigDoesNotExistFn
	previousPrompt := promptUseAirFn
	previousEnsureAirReady := ensureAirReadyFn
	previousGenerate := generateKeelConfigFn
	t.Cleanup(func() {
		validateKeelConfigDoesNotExistFn = previousValidate
		promptUseAirFn = previousPrompt
		ensureAirReadyFn = previousEnsureAirReady
		generateKeelConfigFn = previousGenerate
	})

	t.Run("returns validation error", func(t *testing.T) {
		validateKeelConfigDoesNotExistFn = func(path string) error { return errors.New("already exists") }
		promptUseAirFn = promptUseAir
		ensureAirReadyFn = ensureAirReady
		generateKeelConfigFn = generateKeelConfig

		err := runInit(nil, nil)
		if err == nil || err.Error() != "already exists" {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("returns prompt error", func(t *testing.T) {
		validateKeelConfigDoesNotExistFn = func(path string) error { return nil }
		promptUseAirFn = func() (bool, bool, error) { return false, false, errors.New("prompt failed") }
		ensureAirReadyFn = ensureAirReady
		generateKeelConfigFn = generateKeelConfig

		err := runInit(nil, nil)
		if err == nil || err.Error() != "prompt failed" {
			t.Fatalf("expected prompt error, got %v", err)
		}
	})

	t.Run("returns ensureAirReady error", func(t *testing.T) {
		validateKeelConfigDoesNotExistFn = func(path string) error { return nil }
		promptUseAirFn = func() (bool, bool, error) { return true, false, nil }
		ensureAirReadyFn = func(useAir bool) error { return errors.New("air failed") }
		generateKeelConfigFn = generateKeelConfig

		err := runInit(nil, nil)
		if err == nil || err.Error() != "air failed" {
			t.Fatalf("expected ensureAirReady error, got %v", err)
		}
	})

	t.Run("returns generate error", func(t *testing.T) {
		validateKeelConfigDoesNotExistFn = func(path string) error { return nil }
		promptUseAirFn = func() (bool, bool, error) { return true, true, nil }
		ensureAirReadyFn = func(useAir bool) error { return nil }
		generateKeelConfigFn = func(destPath string, useAir, airConfigExists bool) error {
			return errors.New("generate failed")
		}

		err := runInit(nil, nil)
		if err == nil || err.Error() != "generate failed" {
			t.Fatalf("expected generate error, got %v", err)
		}
	})

	t.Run("success passes prompt values to generator", func(t *testing.T) {
		validateKeelConfigDoesNotExistFn = func(path string) error { return nil }
		promptUseAirFn = func() (bool, bool, error) { return true, false, nil }

		receivedUseAir := false
		receivedAirConfigExists := true
		ensureAirReadyFn = func(useAir bool) error {
			receivedUseAir = useAir
			return nil
		}
		generateKeelConfigFn = func(destPath string, useAir, airConfigExists bool) error {
			if destPath != "keel.toml" {
				t.Fatalf("expected keel.toml destination, got %q", destPath)
			}
			receivedUseAir = useAir
			receivedAirConfigExists = airConfigExists
			return nil
		}

		if err := runInit(nil, nil); err != nil {
			t.Fatalf("runInit returned error: %v", err)
		}
		if !receivedUseAir || receivedAirConfigExists {
			t.Fatalf("expected useAir=true and airConfigExists=false")
		}
	})
}
