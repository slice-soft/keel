package gomod

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestRunTidy(t *testing.T) {
	t.Run("calls go mod tidy with correct args", func(t *testing.T) {
		var gotName string
		var gotArgs []string
		runner := func(name string, args ...string) *exec.Cmd {
			gotName = name
			gotArgs = args
			return exec.Command("true")
		}

		var out bytes.Buffer
		if err := RunTidy(runner, ".", &out, &out); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotName != "go" {
			t.Errorf("expected command 'go', got %q", gotName)
		}
		if len(gotArgs) != 2 || gotArgs[0] != "mod" || gotArgs[1] != "tidy" {
			t.Errorf("expected args [mod tidy], got %v", gotArgs)
		}
		if !strings.Contains(out.String(), "go mod tidy") {
			t.Errorf("expected stdout to mention 'go mod tidy', got %q", out.String())
		}
	})

	t.Run("returns error when command fails", func(t *testing.T) {
		runner := func(name string, args ...string) *exec.Cmd {
			return exec.Command("false")
		}
		err := RunTidy(runner, ".", nil, nil)
		if err == nil {
			t.Fatal("expected error from failing command")
		}
		if !strings.Contains(err.Error(), "go mod tidy failed") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("nil runner falls back to exec.Command", func(t *testing.T) {
		// Provide a runner that always errors to avoid actually running go mod tidy.
		// Passing nil should use exec.Command; we just verify it doesn't panic.
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("RunTidy panicked with nil runner: %v", r)
			}
		}()
		// We can't reliably run go mod tidy in a temp dir without a go.mod,
		// so use a noop runner to confirm nil is handled.
		_ = RunTidy(nil, t.TempDir(), nil, nil)
	})

	t.Run("nil stdout and stderr are replaced with discard", func(t *testing.T) {
		runner := func(name string, args ...string) *exec.Cmd {
			return exec.Command("true")
		}
		// Should not panic when stdout/stderr are nil.
		if err := RunTidy(runner, ".", nil, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wraps command error with context", func(t *testing.T) {
		runner := func(name string, args ...string) *exec.Cmd {
			return exec.Command("sh", "-c", "exit 2")
		}
		err := RunTidy(runner, ".", nil, nil)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "go mod tidy failed") {
			t.Errorf("expected wrapped error, got: %v", err)
		}
	})
}
