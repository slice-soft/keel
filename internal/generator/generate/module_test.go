package generate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadModuleName(t *testing.T) {
	tests := []struct {
		name        string
		withGoMod   bool
		content     string
		expectEmpty bool
	}{
		{name: "with go.mod ", withGoMod: true, content: "module github.com/slice-soft/my-backend\n\ngo 1.21\n", expectEmpty: false},
		{name: "without go.mod", withGoMod: false, expectEmpty: true},
		{name: "without module line", withGoMod: true, content: "go 1.21\n", expectEmpty: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if tt.withGoMod {
				path := filepath.Join(root, "go.mod")
				if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
					t.Fatalf("failed to write go.mod: %v", err)
				}
			}

			wd, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get cwd: %v", err)
			}
			t.Cleanup(func() {
				_ = os.Chdir(wd)
			})

			if err := os.Chdir(root); err != nil {
				t.Fatalf("failed to chdir: %v", err)
			}

			got := ReadModuleName()
			if tt.expectEmpty && got != "" {
				t.Fatalf("expected empty module name, got %q", got)
			}
			if !tt.expectEmpty && got != "github.com/slice-soft/my-backend" {
				t.Fatalf("expected %q, got %q", "github.com/slice-soft/my-backend", got)
			}
		})
	}
}
