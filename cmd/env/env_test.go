package env_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	envCmd "github.com/slice-soft/keel/cmd/env"
	"github.com/slice-soft/keel/internal/keeltoml"
)

// setupDir changes the working directory to a temp dir for the duration of the test.
func setupDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

func assertContains(t *testing.T, content, sub string) {
	t.Helper()
	if !strings.Contains(content, sub) {
		t.Errorf("expected content to contain %q\n\nfull content:\n%s", sub, content)
	}
}

func assertNotContains(t *testing.T, content, sub string) {
	t.Helper()
	if strings.Contains(content, sub) {
		t.Errorf("content should NOT contain %q\n\nfull content:\n%s", sub, content)
	}
}

// ---- keel env sync --------------------------------------------------------

func TestEnvSync_GeneratesEnvExample(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[env]]
key         = "DB_DSN"
source      = "gorm"
required    = true
secret      = true
description = "PostgreSQL DSN"

[[env]]
key      = "REDIS_URL"
source   = "redis"
required = false
default  = "redis://localhost:6379"
`)

	cmd := envCmd.NewCommand()
	cmd.SetArgs([]string{"sync"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := readFile(t, ".env.example")
	assertContains(t, content, "DB_DSN=your-secret-here")
	assertContains(t, content, "REDIS_URL=redis://localhost:6379")
	// Must NOT contain real values pattern (placeholder only)
	assertNotContains(t, content, "postgres://")
}

func TestEnvSync_SecretGetsPlaceholder(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[env]]
key    = "JWT_SECRET"
secret = true
`)

	cmd := envCmd.NewCommand()
	cmd.SetArgs([]string{"sync"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, ".env.example")
	assertContains(t, content, "JWT_SECRET=your-secret-here")
}

func TestEnvSync_PreservesExistingManualEntries(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[env]]
key = "NEW_KEY"
`)
	// Pre-existing manual entry in .env.example.
	writeFile(t, ".env.example", "MANUAL_KEY=some-value\n")

	cmd := envCmd.NewCommand()
	cmd.SetArgs([]string{"sync"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, ".env.example")
	assertContains(t, content, "MANUAL_KEY=some-value")
	assertContains(t, content, "NEW_KEY=")
}

func TestEnvSync_NoDuplicateKeys(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[env]]
key = "DB_HOST"
`)
	// Key already in .env.example — should not be duplicated.
	writeFile(t, ".env.example", "DB_HOST=localhost\n")

	cmd := envCmd.NewCommand()
	cmd.SetArgs([]string{"sync"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, ".env.example")
	if strings.Count(content, "DB_HOST=") != 1 {
		t.Errorf("expected exactly 1 DB_HOST entry, got:\n%s", content)
	}
}

func TestEnvSync_NoEnvEntries(t *testing.T) {
	setupDir(t)
	writeFile(t, keeltoml.DefaultPath, "[app]\nname = \"myapp\"\n")

	cmd := envCmd.NewCommand()
	cmd.SetArgs([]string{"sync"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// .env.example should not be created.
	if _, err := os.Stat(".env.example"); err == nil {
		t.Error("expected .env.example to not exist when there are no env entries")
	}
}

// ---- keel env generate ----------------------------------------------------

func TestEnvGenerate_RequiredKeyIsEmpty(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[env]]
key      = "DB_DSN"
required = true
`)

	cmd := envCmd.NewCommand()
	cmd.SetArgs([]string{"generate"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, ".env")
	assertContains(t, content, "DB_DSN=")
	// Required key should NOT be commented.
	assertNotContains(t, content, "# DB_DSN")
}

func TestEnvGenerate_OptionalKeyWithDefaultIsCommented(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[env]]
key      = "LOG_LEVEL"
required = false
default  = "info"
`)

	cmd := envCmd.NewCommand()
	cmd.SetArgs([]string{"generate"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, ".env")
	assertContains(t, content, "# LOG_LEVEL=info")
}

func TestEnvGenerate_DoesNotOverwriteExistingKeys(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[env]]
key      = "DB_HOST"
required = true

[[env]]
key      = "NEW_KEY"
required = true
`)
	// DB_HOST already in .env with a real value.
	writeFile(t, ".env", "DB_HOST=production-db.example.com\n")

	cmd := envCmd.NewCommand()
	cmd.SetArgs([]string{"generate"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, ".env")
	// Original value must be preserved.
	assertContains(t, content, "DB_HOST=production-db.example.com")
	// New key must be added.
	assertContains(t, content, "NEW_KEY=")
	// DB_HOST must appear exactly once.
	if strings.Count(content, "DB_HOST=") != 1 {
		t.Errorf("DB_HOST should appear once, got:\n%s", content)
	}
}

func TestEnvGenerate_NoEnvEntries(t *testing.T) {
	setupDir(t)
	writeFile(t, keeltoml.DefaultPath, "[app]\nname = \"myapp\"\n")

	cmd := envCmd.NewCommand()
	cmd.SetArgs([]string{"generate"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// .env should not be created.
	if _, err := os.Stat(".env"); err == nil {
		t.Error("expected .env to not exist when there are no env entries")
	}
}

// ---- keel env check -------------------------------------------------------

func TestEnvCheck_AllRequiredPresent(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[env]]
key      = "DB_DSN"
required = true

[[env]]
key      = "APP_PORT"
required = true
`)
	writeFile(t, ".env", "DB_DSN=postgres://localhost/db\nAPP_PORT=8080\n")

	cmd := envCmd.NewCommand()
	cmd.SetArgs([]string{"check"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnvCheck_RequiredMissing(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[env]]
key      = "JWT_SECRET"
required = true
`)
	writeFile(t, ".env", "APP_ENV=development\n")

	cmd := envCmd.NewCommand()
	cmd.SetArgs([]string{"check"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing required var")
	}
	assertContains(t, err.Error(), "missing required")
}

func TestEnvCheck_RequiredFromOSEnv(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[env]]
key      = "JWT_SECRET"
required = true
`)
	writeFile(t, ".env", "# empty\n")
	t.Setenv("JWT_SECRET", "supersecret")

	cmd := envCmd.NewCommand()
	cmd.SetArgs([]string{"check"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error when var is in OS env, got: %v", err)
	}
}

func TestEnvCheck_OptionalMissingIsWarningNotError(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[env]]
key      = "CACHE_TTL"
required = false
default  = "300"
`)
	writeFile(t, ".env", "# empty\n")

	cmd := envCmd.NewCommand()
	cmd.SetArgs([]string{"check"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("optional missing var should not return error, got: %v", err)
	}
}

func TestEnvCheck_MixedRequiredAndOptional(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[env]]
key      = "DB_DSN"
required = true

[[env]]
key      = "LOG_LEVEL"
required = false
default  = "info"
`)
	// DB_DSN is set, LOG_LEVEL is not.
	writeFile(t, ".env", "DB_DSN=postgres://localhost/db\n")

	cmd := envCmd.NewCommand()
	cmd.SetArgs([]string{"check"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("all required vars are present, optional missing is ok: %v", err)
	}
}

func TestEnvCheck_NoEnvEntries(t *testing.T) {
	setupDir(t)
	writeFile(t, keeltoml.DefaultPath, "[app]\nname = \"myapp\"\n")

	cmd := envCmd.NewCommand()
	cmd.SetArgs([]string{"check"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
