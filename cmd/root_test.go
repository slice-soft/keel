package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecuteSuccessPath(t *testing.T) {
	previousPreRun := rootCmd.PersistentPreRun
	previousPostRun := rootCmd.PersistentPostRun
	previousUpdateCh := updateCh
	previousExitFn := exitFn
	previousStderrWriter := stderrWriter
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.PersistentPreRun = previousPreRun
		rootCmd.PersistentPostRun = previousPostRun
		updateCh = previousUpdateCh
		exitFn = previousExitFn
		stderrWriter = previousStderrWriter
	})

	rootCmd.SetArgs([]string{"--help"})
	rootCmd.PersistentPreRun = nil
	rootCmd.PersistentPostRun = nil
	updateCh = nil
	exitFn = func(code int) {}
	stderrWriter = &bytes.Buffer{}

	Execute()
}

func TestExecuteErrorPath(t *testing.T) {
	previousPreRun := rootCmd.PersistentPreRun
	previousPostRun := rootCmd.PersistentPostRun
	previousUpdateCh := updateCh
	previousExitFn := exitFn
	previousStderrWriter := stderrWriter
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.PersistentPreRun = previousPreRun
		rootCmd.PersistentPostRun = previousPostRun
		updateCh = previousUpdateCh
		exitFn = previousExitFn
		stderrWriter = previousStderrWriter
	})

	rootCmd.SetArgs([]string{"unknown-subcommand"})
	rootCmd.PersistentPreRun = nil
	rootCmd.PersistentPostRun = nil
	updateCh = nil

	exitCode := 0
	exitFn = func(code int) {
		exitCode = code
	}
	var stderr bytes.Buffer
	stderrWriter = &stderr

	Execute()

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("expected error output to include unknown command, got %q", stderr.String())
	}
}

func TestRootRegistersUpdateRelatedCommands(t *testing.T) {
	for _, name := range []string{"upgrade", "version"} {
		found, _, err := rootCmd.Find([]string{name})
		if err != nil {
			t.Fatalf("expected to find %q command, got error %v", name, err)
		}
		if found == nil || found.Name() != name {
			t.Fatalf("expected to find %q command, got %#v", name, found)
		}
	}
}
