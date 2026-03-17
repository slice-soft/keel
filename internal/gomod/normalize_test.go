package gomod

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeDirective(t *testing.T) {
	t.Run("strips patch version", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "go.mod")
		if err := os.WriteFile(path, []byte("module example.com/demo\n\ngo 1.25.7\n"), 0644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		if err := NormalizeDirective(path); err != nil {
			t.Fatalf("NormalizeDirective returned error: %v", err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read go.mod: %v", err)
		}
		if !strings.Contains(string(content), "go 1.25\n") {
			t.Fatalf("expected normalized go directive, got %q", string(content))
		}
	})

	t.Run("keeps major minor directive", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "go.mod")
		original := "module example.com/demo\n\ngo 1.25\n"
		if err := os.WriteFile(path, []byte(original), 0644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		if err := NormalizeDirective(path); err != nil {
			t.Fatalf("NormalizeDirective returned error: %v", err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read go.mod: %v", err)
		}
		if string(content) != original {
			t.Fatalf("expected go.mod to remain unchanged, got %q", string(content))
		}
	})

	t.Run("returns read error", func(t *testing.T) {
		err := NormalizeDirective(filepath.Join(t.TempDir(), "missing.go.mod"))
		if err == nil {
			t.Fatal("expected read error")
		}
	})
}
