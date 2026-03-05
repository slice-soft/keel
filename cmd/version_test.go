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
