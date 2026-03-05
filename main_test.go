package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	previousArgs := os.Args
	t.Cleanup(func() {
		os.Args = previousArgs
	})

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if err := os.MkdirAll(filepath.Join(home, ".keel"), 0755); err != nil {
		t.Fatalf("failed creating ~/.keel: %v", err)
	}
	data, _ := time.Now().MarshalText()
	if err := os.WriteFile(filepath.Join(home, ".keel", "last_check"), data, 0644); err != nil {
		t.Fatalf("failed writing last_check: %v", err)
	}

	os.Args = []string{"keel", "--help"}
	main()
}
