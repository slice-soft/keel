package cmd

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
)

func TestRenderVersionOutputIncludesMetadata(t *testing.T) {
	output := renderVersionOutput("1.2.3", "abc1234", "2026-03-05T12:00:00Z")

	requiredFragments := []string{
		"keel-cli: 1.2.3",
		"commit: abc1234",
		"build date: 2026-03-05T12:00:00Z",
		"go: " + runtime.Version(),
		"operating system: " + runtime.GOOS + "/" + runtime.GOARCH,
		"framework: Keel Framework (https://keel-go.dev)",
		"repository: Keel CLI Repository (https://github.com/slice-soft/keel)",
	}

	for _, fragment := range requiredFragments {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected output to contain %q", fragment)
		}
	}
}

func TestVersionCommandWritesOutput(t *testing.T) {
	previousVersion := version
	previousCommit := commit
	previousBuildDate := buildDate
	version = "test-version"
	commit = "test-commit"
	buildDate = "test-date"
	t.Cleanup(func() {
		version = previousVersion
		commit = previousCommit
		buildDate = previousBuildDate
	})

	command := newVersionCommand()
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)

	command.Run(command, nil)

	if !strings.Contains(stdout.String(), "keel-cli: test-version") {
		t.Fatalf("expected command output to include CLI version, got: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "commit: test-commit") {
		t.Fatalf("expected command output to include commit, got: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "build date: test-date") {
		t.Fatalf("expected command output to include build date, got: %q", stdout.String())
	}
}

func TestSyncRootVersionOutputUsesSharedRenderer(t *testing.T) {
	previousVersion := version
	previousCommit := commit
	previousBuildDate := buildDate
	previousRootVersion := rootCmd.Version
	t.Cleanup(func() {
		version = previousVersion
		commit = previousCommit
		buildDate = previousBuildDate
		rootCmd.Version = previousRootVersion
	})

	version = "v9.9.9"
	commit = "c0ffee"
	buildDate = "2026-03-05T13:00:00Z"

	syncRootVersionOutput()

	want := renderVersionOutput(version, commit, buildDate)
	if rootCmd.Version != want {
		t.Fatalf("expected root version output to match renderer")
	}
}
