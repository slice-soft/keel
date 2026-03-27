package generator

import (
	"strings"
	"testing"
)

func TestReadTemplate(t *testing.T) {
	t.Run("reads existing template", func(t *testing.T) {
		content, err := ReadTemplate("templates/project/keel.toml.tmpl")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(content) == 0 {
			t.Fatal("expected non-empty template content")
		}
	})

	t.Run("content is valid text", func(t *testing.T) {
		content, err := ReadTemplate("templates/project/application.properties.tmpl")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(string(content), "application") && !strings.Contains(string(content), "{{") {
			t.Errorf("expected template content, got: %q", string(content))
		}
	})

	t.Run("returns error for missing template", func(t *testing.T) {
		_, err := ReadTemplate("templates/does-not-exist.tmpl")
		if err == nil {
			t.Fatal("expected error for missing template")
		}
	})
}
