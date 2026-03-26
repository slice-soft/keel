package doctor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

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
