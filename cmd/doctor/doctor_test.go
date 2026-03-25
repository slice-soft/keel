package doctor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slice-soft/keel/cmd/doctor"
	"github.com/slice-soft/keel/internal/keeltoml"
)

func TestDoctor_HealthyProject(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[addons]]
id   = "gorm"
repo = "github.com/slice-soft/ss-keel-gorm"
`)
	writeFile(t, "application.properties", "jwt.secret=${JWT_SECRET}\n")
	writeFile(t, "go.mod", `module myapp

go 1.25

require github.com/slice-soft/ss-keel-gorm v1.0.0
`)
	writeFile(t, ".env", "JWT_SECRET=supersecret\n")

	cmd := doctor.NewCommand()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestDoctor_MissingKeelToml(t *testing.T) {
	setupDir(t)
	cmd := doctor.NewCommand()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error for missing keel.toml, got: %v", err)
	}
}

func TestDoctor_InvalidKeelToml(t *testing.T) {
	setupDir(t)
	writeFile(t, keeltoml.DefaultPath, "[[this is not valid toml\n")

	cmd := doctor.NewCommand()
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for invalid keel.toml")
	}
}

func TestDoctor_InvalidApplicationProperties(t *testing.T) {
	setupDir(t)
	writeFile(t, keeltoml.DefaultPath, "[project]\nname = \"demo\"\n")
	writeFile(t, "application.properties", "invalid-line")

	cmd := doctor.NewCommand()
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for invalid application.properties")
	}
}

func TestDoctor_AddonMissingFromGoMod(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[addons]]
id   = "gorm"
repo = "github.com/slice-soft/ss-keel-gorm"
`)
	writeFile(t, "application.properties", "app.name=demo\n")
	writeFile(t, "go.mod", "module myapp\n\ngo 1.25\n")

	cmd := doctor.NewCommand()
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for addon missing from go.mod")
	}
}

func TestDoctor_RequiredPropertyEnvVarMissing(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, "[project]\nname = \"demo\"\n")
	writeFile(t, "application.properties", "jwt.secret=${JWT_SECRET}\n")
	writeFile(t, ".env", "# empty\n")

	cmd := doctor.NewCommand()
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing required env var")
	}
	if !strings.Contains(err.Error(), "doctor found issues") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDoctor_RequiredPropertyEnvVarFromOSEnv(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, "[project]\nname = \"demo\"\n")
	writeFile(t, "application.properties", "jwt.secret=${JWT_SECRET}\n")
	writeFile(t, ".env", "# empty\n")
	t.Setenv("JWT_SECRET", "supersecret")

	cmd := doctor.NewCommand()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error when var is in OS env, got: %v", err)
	}
}

func TestDoctor_LegacyFallbackToKeelTomlEnv(t *testing.T) {
	setupDir(t)

	writeFile(t, keeltoml.DefaultPath, `
[[env]]
key      = "JWT_SECRET"
required = true
`)
	writeFile(t, ".env", "JWT_SECRET=legacy-secret\n")

	cmd := doctor.NewCommand()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected legacy keel.toml fallback to work, got: %v", err)
	}
}

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
	t.Cleanup(func() { _ = os.Chdir(orig) })
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
