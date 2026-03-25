package new

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveProjectNameRequiresArgWithYesFlag(t *testing.T) {
	previous := yesFlag
	yesFlag = true
	t.Cleanup(func() { yesFlag = previous })

	_, err := resolveProjectName(nil)
	if err == nil {
		t.Fatalf("expected error when --yes is enabled and project name is missing")
	}
}

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid", input: "my-backend", wantErr: false},
		{name: "empty", input: "", wantErr: true},
		{name: "contains space", input: "my backend", wantErr: true},
		{name: "contains slash", input: "github.com/slice-soft/app", wantErr: true},
		{name: "contains backslash", input: "my\\app", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProjectName(tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestValidateModulePath(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		allowLocal bool
		wantErr    bool
	}{
		{name: "remote valid", input: "github.com/slice-soft/my-backend", allowLocal: false, wantErr: false},
		{name: "remote missing namespace", input: "my-backend", allowLocal: false, wantErr: true},
		{name: "local valid", input: "my-backend", allowLocal: true, wantErr: false},
		{name: "contains space", input: "github.com/slice-soft/my backend", allowLocal: false, wantErr: true},
		{name: "trailing slash", input: "github.com/slice-soft/my-backend/", allowLocal: false, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateModulePath(tt.input, tt.allowLocal)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestValidateCustomDomain(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid", input: "code.example.com", wantErr: false},
		{name: "empty", input: "", wantErr: true},
		{name: "with protocol", input: "https://code.example.com", wantErr: true},
		{name: "contains space", input: "code example.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCustomDomain(tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestBuildProjectFiles(t *testing.T) {
	appName := "my-backend"

	filesWithStarter := buildProjectFiles(appName, true, true, true)
	filesWithoutAir := buildProjectFiles(appName, false, true, true)
	filesWithoutEnv := buildProjectFiles(appName, true, false, true)
	filesWithoutStarter := buildProjectFiles(appName, true, true, false)

	required := map[string]bool{
		filepath.Join(appName, "cmd", "main.go"):         false,
		filepath.Join(appName, "go.mod"):                 false,
		filepath.Join(appName, "keel.toml"):              false,
		filepath.Join(appName, "application.properties"): false,
		filepath.Join(appName, "README.md"):              false,
		filepath.Join(appName, ".gitignore"):             false,
	}
	for _, f := range filesWithStarter {
		if _, ok := required[f.Destination]; ok {
			required[f.Destination] = true
		}
	}
	for path, found := range required {
		if !found {
			t.Fatalf("expected required generated file %s", path)
		}
	}

	hasAirWithConfig := false
	for _, f := range filesWithStarter {
		if f.Destination == filepath.Join(appName, ".air.toml") {
			hasAirWithConfig = true
			break
		}
	}
	if !hasAirWithConfig {
		t.Fatalf("expected .air.toml when includeAirConfig=true")
	}

	hasAirWithoutConfig := false
	for _, f := range filesWithoutAir {
		if f.Destination == filepath.Join(appName, ".air.toml") {
			hasAirWithoutConfig = true
			break
		}
	}
	if hasAirWithoutConfig {
		t.Fatalf("did not expect .air.toml when includeAirConfig=false")
	}

	hasEnvWithoutSupport := false
	for _, f := range filesWithoutEnv {
		if f.Destination == filepath.Join(appName, ".env") {
			hasEnvWithoutSupport = true
			break
		}
	}
	if hasEnvWithoutSupport {
		t.Fatalf("did not expect .env when useEnv=false")
	}

	hasStarterFiles := false
	for _, f := range filesWithStarter {
		if strings.Contains(f.Destination, filepath.Join(appName, "internal", "modules", "starter")) {
			hasStarterFiles = true
			break
		}
	}
	if !hasStarterFiles {
		t.Fatalf("expected starter module files when includeStarterModule=true")
	}

	hasStarterWithoutFlag := false
	for _, f := range filesWithoutStarter {
		if strings.Contains(f.Destination, filepath.Join(appName, "internal", "modules", "starter")) {
			hasStarterWithoutFlag = true
			break
		}
	}
	if hasStarterWithoutFlag {
		t.Fatalf("did not expect starter module files when includeStarterModule=false")
	}
}

func TestProjectNameFromModule(t *testing.T) {
	tests := []struct {
		name      string
		module    string
		want      string
		expectErr bool
	}{
		{name: "github module", module: "github.com/slice-soft/my-backend", want: "my-backend"},
		{name: "local module", module: "my-backend", want: "my-backend"},
		{name: "invalid tail", module: "github.com/slice-soft/my backend", expectErr: true},
		{name: "trailing slash", module: "github.com/slice-soft/", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := projectNameFromModule(tt.module)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil (value=%q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestCreateProjectDirectories(t *testing.T) {
	root := t.TempDir()
	appName := filepath.Join(root, "my-backend")

	if err := createProjectDirectories(appName, true, false); err != nil {
		t.Fatalf("createProjectDirectories returned error: %v", err)
	}

	expectedDirs := []string{
		filepath.Join(appName, "internal", "middleware"),
		filepath.Join(appName, "internal", "guards"),
		filepath.Join(appName, "internal", "scheduler"),
		filepath.Join(appName, "internal", "checkers"),
		filepath.Join(appName, "internal", "events"),
		filepath.Join(appName, "internal", "hooks"),
	}

	for _, d := range expectedDirs {
		info, err := os.Stat(d)
		if err != nil {
			t.Fatalf("expected directory %s to exist: %v", d, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory", d)
		}
	}
}

func TestDefaultModulePath(t *testing.T) {
	got := defaultModulePath("my-backend")
	want := "github.com/my-github-user/my-backend"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestValidateNonEmpty(t *testing.T) {
	validate := validateNonEmpty("Field")
	if err := validate("value"); err != nil {
		t.Fatalf("expected nil error for non-empty value, got %v", err)
	}
	if err := validate("   "); err == nil {
		t.Fatalf("expected error for empty value")
	}
}

func TestCollectProjectSetupWithDefaults(t *testing.T) {
	t.Run("returns defaults when project directory does not exist", func(t *testing.T) {
		oldPath := os.Getenv("PATH")
		binDir := t.TempDir()
		airName := "air"
		if runtime.GOOS == "windows" {
			airName = "air.exe"
		}
		airPath := filepath.Join(binDir, airName)
		if err := os.WriteFile(airPath, []byte(""), 0755); err != nil {
			t.Fatalf("failed to create fake air binary: %v", err)
		}
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+oldPath)

		cwd := t.TempDir()
		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get wd: %v", err)
		}
		defer func() { _ = os.Chdir(oldWD) }()
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}

		setup, err := collectProjectSetupWithDefaults("my-backend")
		if err != nil {
			t.Fatalf("collectProjectSetupWithDefaults returned error: %v", err)
		}

		if setup.appName != "my-backend" {
			t.Fatalf("unexpected appName: %q", setup.appName)
		}
		if setup.moduleName != "github.com/my-github-user/my-backend" {
			t.Fatalf("unexpected moduleName: %q", setup.moduleName)
		}
		if !setup.useAir || !setup.includeAirConfig || !setup.useEnv || !setup.initGit || !setup.installDeps {
			t.Fatalf("expected default boolean flags to be enabled: %#v", setup)
		}
	})

	t.Run("fails when project directory already exists", func(t *testing.T) {
		cwd := t.TempDir()
		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get wd: %v", err)
		}
		defer func() { _ = os.Chdir(oldWD) }()
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}
		if err := os.Mkdir("existing-app", 0755); err != nil {
			t.Fatalf("failed to create existing directory: %v", err)
		}

		_, err = collectProjectSetupWithDefaults("existing-app")
		if err == nil {
			t.Fatalf("expected error when directory exists")
		}
	})
}
