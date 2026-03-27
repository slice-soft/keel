package appproperties

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	t.Run("empty content", func(t *testing.T) {
		doc, err := Parse([]byte{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(doc.EnvVars) != 0 {
			t.Fatalf("expected 0 env vars, got %d", len(doc.EnvVars))
		}
	})

	t.Run("skips comments and blank lines", func(t *testing.T) {
		doc, err := Parse([]byte("# comment\n! also comment\n\napp.name=demo\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(doc.EnvVars) != 0 {
			t.Fatalf("expected 0 env vars from non-placeholder line, got %d", len(doc.EnvVars))
		}
	})

	t.Run("extracts placeholder without default", func(t *testing.T) {
		doc, err := Parse([]byte("jwt.secret=${JWT_SECRET}\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(doc.EnvVars) != 1 {
			t.Fatalf("expected 1 env var, got %d", len(doc.EnvVars))
		}
		v := doc.EnvVars[0]
		if v.Key != "JWT_SECRET" {
			t.Errorf("expected key JWT_SECRET, got %q", v.Key)
		}
		if v.HasDefault {
			t.Error("expected HasDefault=false")
		}
		if v.Default != "" {
			t.Errorf("expected empty default, got %q", v.Default)
		}
	})

	t.Run("extracts placeholder with default", func(t *testing.T) {
		doc, err := Parse([]byte("app.name=${SERVICE_NAME:my-service}\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(doc.EnvVars) != 1 {
			t.Fatalf("expected 1 env var, got %d", len(doc.EnvVars))
		}
		v := doc.EnvVars[0]
		if v.Key != "SERVICE_NAME" {
			t.Errorf("expected key SERVICE_NAME, got %q", v.Key)
		}
		if !v.HasDefault {
			t.Error("expected HasDefault=true")
		}
		if v.Default != "my-service" {
			t.Errorf("expected default 'my-service', got %q", v.Default)
		}
	})

	t.Run("extracts placeholder with empty default", func(t *testing.T) {
		doc, err := Parse([]byte("x=${KEY:}\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(doc.EnvVars) != 1 {
			t.Fatalf("expected 1 env var, got %d", len(doc.EnvVars))
		}
		v := doc.EnvVars[0]
		if !v.HasDefault {
			t.Error("expected HasDefault=true for empty default")
		}
		if v.Default != "" {
			t.Errorf("expected empty default string, got %q", v.Default)
		}
	})

	t.Run("deduplicates repeated keys", func(t *testing.T) {
		doc, err := Parse([]byte("a=${KEY:val1}\nb=${KEY:val2}\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(doc.EnvVars) != 1 {
			t.Fatalf("expected 1 unique key, got %d", len(doc.EnvVars))
		}
	})

	t.Run("multiple placeholders on one line", func(t *testing.T) {
		doc, err := Parse([]byte("x=${FOO:a}/${BAR:b}\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(doc.EnvVars) != 2 {
			t.Fatalf("expected 2 env vars, got %d", len(doc.EnvVars))
		}
	})

	t.Run("colon separator on key=value line", func(t *testing.T) {
		doc, err := Parse([]byte("app.url:${BASE_URL:http://localhost}\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(doc.EnvVars) != 1 {
			t.Fatalf("expected 1 env var, got %d", len(doc.EnvVars))
		}
	})

	t.Run("skips placeholders with invalid keys", func(t *testing.T) {
		doc, err := Parse([]byte("x=${invalid-key:val}\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(doc.EnvVars) != 0 {
			t.Fatalf("expected 0 env vars for invalid key, got %d", len(doc.EnvVars))
		}
	})

	t.Run("returns error for line without separator", func(t *testing.T) {
		_, err := Parse([]byte("no-separator\n"))
		if err == nil {
			t.Fatal("expected error for line without separator")
		}
	})

	t.Run("preserves declaration order", func(t *testing.T) {
		doc, err := Parse([]byte("a=${FIRST}\nb=${SECOND}\nc=${THIRD}\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(doc.EnvVars) != 3 {
			t.Fatalf("expected 3 env vars, got %d", len(doc.EnvVars))
		}
		keys := []string{"FIRST", "SECOND", "THIRD"}
		for i, k := range keys {
			if doc.EnvVars[i].Key != k {
				t.Errorf("index %d: expected %q, got %q", i, k, doc.EnvVars[i].Key)
			}
		}
	})
}

func TestLoad(t *testing.T) {
	t.Run("missing file returns empty document", func(t *testing.T) {
		doc, err := Load(filepath.Join(t.TempDir(), "missing.properties"))
		if err != nil {
			t.Fatalf("expected no error for missing file, got: %v", err)
		}
		if len(doc.EnvVars) != 0 {
			t.Fatalf("expected empty document, got %d env vars", len(doc.EnvVars))
		}
	})

	t.Run("reads and parses file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "application.properties")
		if err := os.WriteFile(path, []byte("x=${API_KEY}\n"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
		doc, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(doc.EnvVars) != 1 || doc.EnvVars[0].Key != "API_KEY" {
			t.Fatalf("unexpected env vars: %+v", doc.EnvVars)
		}
	})

	t.Run("returns error for unreadable file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "unreadable.properties")
		if err := os.WriteFile(path, []byte("x=${KEY}\n"), 0644); err != nil {
			t.Fatalf("failed to write: %v", err)
		}
		if err := os.Chmod(path, 0000); err != nil {
			t.Skip("cannot change file permissions on this system")
		}
		t.Cleanup(func() { _ = os.Chmod(path, 0644) })

		_, err := Load(path)
		if err == nil {
			t.Fatal("expected error for unreadable file")
		}
	})
}

func TestLooksLikeEnvKey(t *testing.T) {
	valid := []string{"KEY", "MY_VAR", "VAR123", "A", "ABC_DEF_123"}
	for _, k := range valid {
		if !looksLikeEnvKey(k) {
			t.Errorf("expected %q to be a valid env key", k)
		}
	}

	invalid := []string{"lowercase", "with-dash", "with space", "MiXeD"}
	for _, k := range invalid {
		if looksLikeEnvKey(k) {
			t.Errorf("expected %q to be invalid", k)
		}
	}
}
