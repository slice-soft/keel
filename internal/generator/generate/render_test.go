package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderToFile(t *testing.T) {
	tests := []struct {
		name      string
		file      string
		data      Data
		wantError bool
		contains  string
	}{
		{
			name:     "render env file",
			file:     "templates/project/env.tmpl",
			data:     Data{AppName: "my-backend"},
			contains: "SERVICE_NAME=my-backend",
		},
		{
			name:      "template not found",
			file:      "templates/project/does-not-exist.tmpl",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			dest := filepath.Join(root, "my-backend", ".env")

			data := tt.data
			err := RenderToFile(tt.file, dest, data)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("RenderToFile returned error: %v", err)
			}

			if !FileExists(dest) {
				t.Fatalf("expected generated file %s to exist", dest)
			}

			content, err := os.ReadFile(dest)
			if err != nil {
				t.Fatalf("failed to read generated file: %v", err)
			}
			text := string(content)
			if !strings.Contains(text, tt.contains) {
				t.Fatalf("expected rendered content to include %q, got: %q", tt.contains, text)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	test := []struct {
		name      string
		path      string
		wantExist bool
	}{
		{name: "file does not exist", path: "nonexistent.txt", wantExist: false},
		{name: "file exists", path: "existing.txt", wantExist: true},
	}

	for _, tt := range test {
		root := t.TempDir()
		path := filepath.Join(root, tt.path)
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantExist {
				if err := os.WriteFile(path, []byte("ok"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				defer os.Remove(path)
			}

			exists := FileExists(path)
			if exists != tt.wantExist {
				t.Fatalf("expected FileExists(%q) to be %v, got %v", path, tt.wantExist, exists)
			}
		})
	}
}

func TestRenderReadmeTemplateByStarterFlag(t *testing.T) {
	tests := []struct {
		name             string
		useStarterModule bool
		wantContains     string
	}{
		{
			name:             "with starter module",
			useStarterModule: true,
			wantContains:     "internal/modules/",
		},
		{
			name:             "without starter module",
			useStarterModule: false,
			wantContains:     "keel g module users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			dest := filepath.Join(root, "README.md")

			data := Data{
				AppName:          "my-backend",
				UseStarterModule: tt.useStarterModule,
				UseEnv:           true,
				UseAirConfig:     true,
			}

			if err := RenderToFile("templates/project/readme.tmpl", dest, data); err != nil {
				t.Fatalf("RenderToFile returned error: %v", err)
			}

			content, err := os.ReadFile(dest)
			if err != nil {
				t.Fatalf("failed to read generated README: %v", err)
			}
			text := string(content)

			if !strings.Contains(text, "https://docs.keel-go.dev") {
				t.Fatalf("expected README to include docs URL")
			}
			if !strings.Contains(text, tt.wantContains) {
				t.Fatalf("expected README to include %q, got: %q", tt.wantContains, text)
			}
		})
	}
}

func TestRenderKeelTemplateForInitMode(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "keel.toml")

	data := NewInitData("my-backend", true, false)

	if err := RenderToFile("templates/project/keel.toml.tmpl", dest, data); err != nil {
		t.Fatalf("RenderToFile returned error: %v", err)
	}

	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read generated keel.toml: %v", err)
	}
	text := string(content)

	if !strings.Contains(text, "name    = \"\"") || !strings.Contains(text, "version = \"\"") {
		t.Fatalf("expected init keel.toml to keep [app] section with empty values, got: %s", text)
	}

	if !strings.Contains(text, "dev   = \"air\"") {
		t.Fatalf("expected init keel.toml to include air script, got: %s", text)
	}
}

func TestRenderKeelTemplateForInitModeWithExistingAirConfig(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "keel.toml")

	data := NewInitData("my-backend", true, true)

	if err := RenderToFile("templates/project/keel.toml.tmpl", dest, data); err != nil {
		t.Fatalf("RenderToFile returned error: %v", err)
	}

	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read generated keel.toml: %v", err)
	}
	text := string(content)

	if !strings.Contains(text, "dev   = \"air -c .air.toml\"") {
		t.Fatalf("expected init keel.toml to include air config script, got: %s", text)
	}
}

func TestRenderToFileGoFormattingAndWriteErrors(t *testing.T) {
	t.Run("formats generated go file", func(t *testing.T) {
		root := t.TempDir()
		dest := filepath.Join(root, "service.go")
		data := NewData("users")

		if err := RenderToFile("templates/service/service.go.tmpl", dest, data); err != nil {
			t.Fatalf("RenderToFile returned error: %v", err)
		}

		content, err := os.ReadFile(dest)
		if err != nil {
			t.Fatalf("failed to read generated file: %v", err)
		}
		text := string(content)
		if !strings.Contains(text, "package users") {
			t.Fatalf("expected go output to contain package declaration, got: %s", text)
		}
	})

	t.Run("returns error when destination path is invalid", func(t *testing.T) {
		err := RenderToFile("templates/project/env.tmpl", "bad\x00path/.env", Data{AppName: "demo"})
		if err == nil {
			t.Fatalf("expected error for invalid destination path")
		}
	})

	t.Run("returns error when destination is a directory", func(t *testing.T) {
		root := t.TempDir()
		dest := filepath.Join(root, "already-a-dir")
		if err := os.MkdirAll(dest, 0755); err != nil {
			t.Fatalf("failed to create destination directory: %v", err)
		}

		err := RenderToFile("templates/project/env.tmpl", dest, Data{AppName: "demo"})
		if err == nil {
			t.Fatalf("expected write error when destination is a directory")
		}
	})
}
