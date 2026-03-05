package new

import (
	"os"
	"path/filepath"
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
		filepath.Join(appName, "cmd", "main.go"): false,
		filepath.Join(appName, "go.mod"):         false,
		filepath.Join(appName, "keel.toml"):      false,
		filepath.Join(appName, "README.md"):      false,
		filepath.Join(appName, ".gitignore"):     false,
	}
	for _, f := range filesWithStarter {
		if _, ok := required[f.dest]; ok {
			required[f.dest] = true
		}
	}
	for path, found := range required {
		if !found {
			t.Fatalf("expected required generated file %s", path)
		}
	}

	hasAirWithConfig := false
	for _, f := range filesWithStarter {
		if f.dest == filepath.Join(appName, ".air.toml") {
			hasAirWithConfig = true
			break
		}
	}
	if !hasAirWithConfig {
		t.Fatalf("expected .air.toml when includeAirConfig=true")
	}

	hasAirWithoutConfig := false
	for _, f := range filesWithoutAir {
		if f.dest == filepath.Join(appName, ".air.toml") {
			hasAirWithoutConfig = true
			break
		}
	}
	if hasAirWithoutConfig {
		t.Fatalf("did not expect .air.toml when includeAirConfig=false")
	}

	hasEnvWithoutSupport := false
	for _, f := range filesWithoutEnv {
		if f.dest == filepath.Join(appName, ".env") {
			hasEnvWithoutSupport = true
			break
		}
	}
	if hasEnvWithoutSupport {
		t.Fatalf("did not expect .env when useEnv=false")
	}

	hasStarterFiles := false
	for _, f := range filesWithStarter {
		if strings.Contains(f.dest, filepath.Join(appName, "internal", "modules", "starter")) {
			hasStarterFiles = true
			break
		}
	}
	if !hasStarterFiles {
		t.Fatalf("expected starter module files when includeStarterModule=true")
	}

	hasStarterWithoutFlag := false
	for _, f := range filesWithoutStarter {
		if strings.Contains(f.dest, filepath.Join(appName, "internal", "modules", "starter")) {
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

func TestInferGitHubOwner(t *testing.T) {
	tests := []struct {
		name       string
		githubUser string
		userEmail  string
		userName   string
		want       string
	}{
		{
			name:       "prefers github user",
			githubUser: "Slice-Soft",
			userEmail:  "dev@example.com",
			userName:   "Dev Team",
			want:       "slice-soft",
		},
		{
			name:      "falls back to email local part",
			userEmail: "my.user+team@example.com",
			userName:  "Dev Team",
			want:      "my-user-team",
		},
		{
			name:     "falls back to user name",
			userName: "Slice Soft Backend",
			want:     "slice-soft-backend",
		},
		{
			name: "uses default when all values are invalid",
			want: "my-github-user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferGitHubOwner(tt.githubUser, tt.userEmail, tt.userName)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestSanitizeGitHubOwner(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "trim lower", input: "  Slice-Soft  ", want: "slice-soft"},
		{name: "replace invalid chars", input: "my.user+team", want: "my-user-team"},
		{name: "collapse hyphens", input: "my---team__", want: "my-team"},
		{name: "empty", input: "   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeGitHubOwner(tt.input)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
