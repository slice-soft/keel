package cmd

import "testing"

func TestApplyBuildInfoDoesNotPanic(t *testing.T) {
	originalVersion := version
	originalCommit := commit
	originalBuildDate := buildDate
	t.Cleanup(func() {
		version = originalVersion
		commit = originalCommit
		buildDate = originalBuildDate
	})

	applyBuildInfo()
}

func TestApplyBuildInfoDoesNotSetDevilVersion(t *testing.T) {
	originalVersion := version
	t.Cleanup(func() { version = originalVersion })

	version = "dev"
	applyBuildInfo()

	if version == "(devel)" {
		t.Fatal("applyBuildInfo must not set version to '(devel)'")
	}
}
