package doctor_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slice-soft/keel/cmd/doctor"
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

// writeFile writes content to path (relative to current dir).
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// ---- tests ----------------------------------------------------------------

func TestDoctor_HealthyProject(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[addons]]
id   = "gorm"
repo = "github.com/slice-soft/ss-keel-gorm"

[[env]]
key      = "DB_DSN"
source   = "gorm"
required = true
`)
	writeFile(t, "go.mod", `module myapp

go 1.23

require github.com/slice-soft/ss-keel-gorm v1.0.0
`)
	writeFile(t, ".env", "DB_DSN=postgres://localhost/mydb\n")

	cmd := doctor.NewCommand()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestDoctor_MissingKeelToml(t *testing.T) {
	setupDir(t)
	// No keel.toml — should warn (not error) and succeed.
	cmd := doctor.NewCommand()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error for missing keel.toml, got: %v", err)
	}
}

func TestDoctor_InvalidKeelToml(t *testing.T) {
	setupDir(t)
	writeFile(t, keeltoml.DefaultPath, "[[this is not valid toml\n")

	cmd := doctor.NewCommand()
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid keel.toml, got nil")
	}
}

func TestDoctor_AddonMissingFromGoMod(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[addons]]
id   = "gorm"
repo = "github.com/slice-soft/ss-keel-gorm"
`)
	writeFile(t, "go.mod", "module myapp\n\ngo 1.23\n")

	cmd := doctor.NewCommand()
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for addon missing from go.mod")
	}
}

func TestDoctor_AddonFoundByHeuristic(t *testing.T) {
	setupDir(t)

	// No repo field — falls back to ss-keel-<id> heuristic.
	writeFile(t, keeltoml.DefaultPath, `
[[addons]]
id = "redis"
`)
	writeFile(t, "go.mod", `module myapp

go 1.23

require github.com/slice-soft/ss-keel-redis v2.0.0
`)

	cmd := doctor.NewCommand()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestDoctor_RequiredEnvVarMissing(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[env]]
key      = "JWT_SECRET"
required = true
`)
	writeFile(t, ".env", "APP_ENV=development\n")

	cmd := doctor.NewCommand()
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing required env var")
	}
	if !strings.Contains(err.Error(), "doctor found issues") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDoctor_RequiredEnvVarFromOSEnv(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[env]]
key      = "JWT_SECRET"
required = true
`)
	writeFile(t, ".env", "# empty\n")

	t.Setenv("JWT_SECRET", "supersecret")

	cmd := doctor.NewCommand()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error when var is in OS env, got: %v", err)
	}
}

func TestDoctor_OptionalVarMissingIsNotError(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[env]]
key      = "OPTIONAL_VAR"
required = false
default  = "default-value"
`)
	writeFile(t, ".env", "# empty\n")

	cmd := doctor.NewCommand()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("optional missing var should not be an error, got: %v", err)
	}
}

func TestDoctor_NoAddons(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[app]
name = "myapp"
`)
	// Healthy: keel.toml valid, no addons to check.
	cmd := doctor.NewCommand()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error for project with no addons, got: %v", err)
	}
}

// Ensure unused import of errors doesn't happen
var _ = errors.New
