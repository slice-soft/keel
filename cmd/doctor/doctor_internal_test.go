package doctor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/slice-soft/keel/internal/appproperties"
	"github.com/slice-soft/keel/internal/keeltoml"
)

func TestDoctor_ProjectReadinessChecksPassWhenTidyAndBuildPass(t *testing.T) {
	doctorSetupDir(t)
	resetDoctorDeps(t)

	writeDoctorFile(t, keeltoml.DefaultPath, "[project]\nname = \"demo\"\n")
	writeDoctorFile(t, "application.properties", "app.name=demo\n")
	writeDoctorFile(t, "go.mod", "module myapp\n\ngo 1.25.0\n")
	writeDoctorFile(t, "cmd/main.go", "package main\n\nfunc main() {}\n")

	tidyCalled := false
	buildCalled := false
	runGoModTidyDiffFn = func() (string, error) {
		tidyCalled = true
		return "", nil
	}
	runGoBuildFn = func() (string, error) {
		buildCalled = true
		return "", nil
	}

	cmd := NewCommand()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !tidyCalled {
		t.Fatal("expected go mod tidy -diff check to run")
	}
	if !buildCalled {
		t.Fatal("expected go build ./... check to run")
	}
}

func TestDoctor_FailsWhenModuleMetadataIsDirty(t *testing.T) {
	doctorSetupDir(t)
	resetDoctorDeps(t)

	writeDoctorFile(t, keeltoml.DefaultPath, "[project]\nname = \"demo\"\n")
	writeDoctorFile(t, "application.properties", "app.name=demo\n")
	writeDoctorFile(t, "go.mod", "module myapp\n\ngo 1.25.0\n")
	writeDoctorFile(t, "cmd/main.go", "package main\n\nfunc main() {}\n")

	buildCalled := false
	runGoModTidyDiffFn = func() (string, error) {
		return "diff current/go.mod tidy/go.mod\n-go 1.25\n+go 1.25.0", errors.New("module needs tidy")
	}
	runGoBuildFn = func() (string, error) {
		buildCalled = true
		return "", nil
	}

	cmd := NewCommand()
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when module metadata is dirty")
	}
	if buildCalled {
		t.Fatal("expected build check to be skipped when tidy diff fails")
	}
}

func TestDoctor_FailsWhenBuildFails(t *testing.T) {
	doctorSetupDir(t)
	resetDoctorDeps(t)

	writeDoctorFile(t, keeltoml.DefaultPath, "[project]\nname = \"demo\"\n")
	writeDoctorFile(t, "application.properties", "app.name=demo\n")
	writeDoctorFile(t, "go.mod", "module myapp\n\ngo 1.25.0\n")
	writeDoctorFile(t, "cmd/main.go", "package main\n\nfunc main() {}\n")

	runGoModTidyDiffFn = func() (string, error) {
		return "", nil
	}
	runGoBuildFn = func() (string, error) {
		return "cmd/main.go:3:2: undefined: app", errors.New("build failed")
	}

	cmd := NewCommand()
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when go build fails")
	}
}

func TestCheckPlaceholderPropertyEnvVars_WarnsOnInsecureDefault(t *testing.T) {
	resetDoctorDeps(t)

	doc := &appproperties.Document{
		EnvVars: []appproperties.EnvVar{
			{Key: "JWT_SECRET", Default: "change-me-in-production", HasDefault: true},
		},
	}

	hasWarnings := false
	checkPlaceholderPropertyEnvVars(doc, "", &hasWarnings)
	if !hasWarnings {
		t.Fatal("expected placeholder property default to trigger warning")
	}
}

func TestCheckPlaceholderPropertyEnvVars_DoesNotWarnOnStrongValue(t *testing.T) {
	resetDoctorDeps(t)

	doc := &appproperties.Document{
		EnvVars: []appproperties.EnvVar{
			{Key: "JWT_SECRET", Default: "change-me-in-production", HasDefault: true},
		},
	}

	hasWarnings := false
	checkPlaceholderPropertyEnvVars(doc, "JWT_SECRET=super-secret-value-123\n", &hasWarnings)
	if hasWarnings {
		t.Fatal("expected strong secret to avoid warning")
	}
}

func TestCheckPlaceholderLegacyEnvVars_WarnsOnLegacyDefault(t *testing.T) {
	resetDoctorDeps(t)

	kt := &keeltoml.KeelToml{
		Env: []keeltoml.EnvEntry{
			{Key: "JWT_SECRET", Secret: true, Default: "change-me-in-production"},
		},
	}

	hasWarnings := false
	checkPlaceholderLegacyEnvVars(kt, "", &hasWarnings)
	if !hasWarnings {
		t.Fatal("expected legacy placeholder default to trigger warning")
	}
}

func TestCheckOAuthConfiguration_WarnsWhenNoProviderConfigured(t *testing.T) {
	resetDoctorDeps(t)

	kt := &keeltoml.KeelToml{
		Addons: []keeltoml.AddonEntry{{ID: "oauth"}},
	}
	dotEnv := "OAUTH_GOOGLE_CLIENT_ID=\nOAUTH_GOOGLE_CLIENT_SECRET=\n"

	hasWarnings := false
	checkOAuthConfiguration(kt, dotEnv, &hasWarnings)
	if !hasWarnings {
		t.Fatal("expected warning when oauth installed but no provider credentials set")
	}
}

func TestCheckOAuthConfiguration_NoWarnWhenProviderConfigured(t *testing.T) {
	resetDoctorDeps(t)

	kt := &keeltoml.KeelToml{
		Addons: []keeltoml.AddonEntry{{ID: "oauth"}},
	}
	dotEnv := "OAUTH_GOOGLE_CLIENT_ID=real-id\nOAUTH_GOOGLE_CLIENT_SECRET=real-secret\n"

	hasWarnings := false
	checkOAuthConfiguration(kt, dotEnv, &hasWarnings)
	if hasWarnings {
		t.Fatal("expected no warning when at least one provider is fully configured")
	}
}

func TestCheckOAuthConfiguration_NoWarnWhenOAuthNotInstalled(t *testing.T) {
	resetDoctorDeps(t)

	kt := &keeltoml.KeelToml{
		Addons: []keeltoml.AddonEntry{{ID: "jwt"}},
	}

	hasWarnings := false
	checkOAuthConfiguration(kt, "", &hasWarnings)
	if hasWarnings {
		t.Fatal("expected no warning when oauth addon is not installed")
	}
}

func TestCheckOAuthConfiguration_NoWarnWhenProviderInOSEnv(t *testing.T) {
	resetDoctorDeps(t)

	lookupOSEnvFn = func(key string) (string, bool) {
		switch key {
		case "OAUTH_GITHUB_CLIENT_ID":
			return "gh-id", true
		case "OAUTH_GITHUB_CLIENT_SECRET":
			return "gh-secret", true
		}
		return "", false
	}

	kt := &keeltoml.KeelToml{
		Addons: []keeltoml.AddonEntry{{ID: "oauth"}},
	}

	hasWarnings := false
	checkOAuthConfiguration(kt, "", &hasWarnings)
	if hasWarnings {
		t.Fatal("expected no warning when provider credentials are set via OS env")
	}
}

func TestSummaryMessage(t *testing.T) {
	if got := summaryMessage(false, false); got != "  ✓  project looks healthy" {
		t.Fatalf("unexpected healthy summary: %q", got)
	}
	if got := summaryMessage(false, true); got != "  ⚠  project looks healthy, but review warnings before production" {
		t.Fatalf("unexpected warning summary: %q", got)
	}
	if got := summaryMessage(true, true); got != "  ✗  doctor found issues — fix them before running the application" {
		t.Fatalf("unexpected error summary: %q", got)
	}
}

func resetDoctorDeps(t *testing.T) {
	t.Helper()

	prevLoadKeelToml := loadKeelTomlFn
	prevLoadApplicationProperties := loadApplicationPropertiesFn
	prevReadGoMod := readGoModFn
	prevReadDotEnv := readDotEnvFn
	prevLookupOSEnv := lookupOSEnvFn
	prevRunGoModTidyDiff := runGoModTidyDiffFn
	prevRunGoBuild := runGoBuildFn

	t.Cleanup(func() {
		loadKeelTomlFn = prevLoadKeelToml
		loadApplicationPropertiesFn = prevLoadApplicationProperties
		readGoModFn = prevReadGoMod
		readDotEnvFn = prevReadDotEnv
		lookupOSEnvFn = prevLookupOSEnv
		runGoModTidyDiffFn = prevRunGoModTidyDiff
		runGoBuildFn = prevRunGoBuild
	})
}

func doctorSetupDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previous)
	})

	return dir
}

func writeDoctorFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
