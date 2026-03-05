package initcmd

import (
	"errors"
	"os/exec"
	"testing"
)

func TestEnsureAirReady(t *testing.T) {
	t.Run("useAir disabled", func(t *testing.T) {
		if err := ensureAirReady(false); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("already installed", func(t *testing.T) {
		reset := stubAirDeps(
			func(file string) (string, error) { return "/usr/local/bin/air", nil },
			func() error { return errors.New("should not run install") },
		)
		defer reset()

		if err := ensureAirReady(true); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("install success", func(t *testing.T) {
		calls := 0
		reset := stubAirDeps(
			func(file string) (string, error) {
				calls++
				if calls == 1 {
					return "", exec.ErrNotFound
				}
				return "/usr/local/bin/air", nil
			},
			func() error { return nil },
		)
		defer reset()

		if err := ensureAirReady(true); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("install failure", func(t *testing.T) {
		reset := stubAirDeps(
			func(file string) (string, error) { return "", exec.ErrNotFound },
			func() error { return errors.New("install failed") },
		)
		defer reset()

		err := ensureAirReady(true)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("install succeeds but binary still not found", func(t *testing.T) {
		reset := stubAirDeps(
			func(file string) (string, error) { return "", exec.ErrNotFound },
			func() error { return nil },
		)
		defer reset()

		if err := ensureAirReady(true); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})
}

func stubAirDeps(stubLookPath func(string) (string, error), stubInstall func() error) func() {
	oldLookPath := lookPath
	oldRunAirInstall := runAirInstall

	lookPath = stubLookPath
	runAirInstall = stubInstall

	return func() {
		lookPath = oldLookPath
		runAirInstall = oldRunAirInstall
	}
}
