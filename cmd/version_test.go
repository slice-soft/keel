package cmd

import (
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

func TestSyncRootVersionOutputUsesSharedRenderer(t *testing.T) {
	previousVersion := version
	previousCommit := commit
	previousBuildDate := buildDate
	previousRootVersion := rootCmd.Version
	previousRootLong := rootCmd.Long
	t.Cleanup(func() {
		version = previousVersion
		commit = previousCommit
		buildDate = previousBuildDate
		rootCmd.Version = previousRootVersion
		rootCmd.Long = previousRootLong
	})

	version = "v9.9.9"
	commit = "c0ffee"
	buildDate = "2026-03-05T13:00:00Z"

	syncRootVersionOutput()

	want := renderVersionOutput(version, commit, buildDate)
	if rootCmd.Version != want {
		t.Fatalf("expected root version output to match renderer")
	}
	if rootCmd.Long != want {
		t.Fatalf("expected root long help header to match renderer")
	}
}

func TestColorHelpers(t *testing.T) {
	t.Run("colors disabled with NO_COLOR", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		t.Setenv("TERM", "xterm-256color")
		if colorsEnabled() {
			t.Fatalf("expected colors to be disabled with NO_COLOR")
		}
	})

	t.Run("colors disabled with dumb term", func(t *testing.T) {
		t.Setenv("NO_COLOR", "")
		t.Setenv("TERM", "dumb")
		if colorsEnabled() {
			t.Fatalf("expected colors to be disabled for TERM=dumb")
		}
	})

	t.Run("colors enabled otherwise", func(t *testing.T) {
		t.Setenv("NO_COLOR", "")
		t.Setenv("TERM", "xterm")
		if !colorsEnabled() {
			t.Fatalf("expected colors to be enabled")
		}
	})

	t.Run("colorize applies escape codes when enabled", func(t *testing.T) {
		colored := colorize("hello", colorCyan, true)
		if !strings.Contains(colored, colorCyan) || !strings.Contains(colored, colorReset) {
			t.Fatalf("expected colored output with ANSI wrappers, got %q", colored)
		}
	})

	t.Run("colorize returns plain text when disabled", func(t *testing.T) {
		if got := colorize("hello", colorCyan, false); got != "hello" {
			t.Fatalf("expected plain text, got %q", got)
		}
	})
}
