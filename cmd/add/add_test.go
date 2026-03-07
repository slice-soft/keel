package add

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slice-soft/keel/internal/addon"
)

func resetAddDeps(t *testing.T) {
	t.Helper()

	prevFetchRegistry := fetchRegistryFn
	prevFetchManifest := fetchManifestFn
	prevInstallAddon := installAddonFn
	prevForceRefresh := forceRefresh

	t.Cleanup(func() {
		fetchRegistryFn = prevFetchRegistry
		fetchManifestFn = prevFetchManifest
		installAddonFn = prevInstallAddon
		forceRefresh = prevForceRefresh
	})
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previous)
	})
}

func setupKeelProject(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	mainPath := filepath.Join(root, "cmd", "main.go")
	if err := os.MkdirAll(filepath.Dir(mainPath), 0755); err != nil {
		t.Fatalf("failed to create cmd directory: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main(){}\n"), 0644); err != nil {
		t.Fatalf("failed to write cmd/main.go: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal"), 0755); err != nil {
		t.Fatalf("failed to create internal directory: %v", err)
	}
}

func setStdin(t *testing.T, input string) {
	t.Helper()

	previous := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("failed to write stdin input: %v", err)
	}
	_ = w.Close()

	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = previous
		_ = r.Close()
	})
}

func TestNewCommand(t *testing.T) {
	cmd := NewCommand()
	if cmd.Use != "add [alias|repo]" {
		t.Fatalf("unexpected command use: %q", cmd.Use)
	}
	if cmd.RunE == nil {
		t.Fatalf("expected RunE to be configured")
	}
	if cmd.Flags().Lookup("refresh") == nil {
		t.Fatalf("expected --refresh flag")
	}
}

func TestResolveRepo(t *testing.T) {
	reg := &addon.Registry{
		Addons: []addon.RegistryEntry{
			{Alias: "gorm", Repo: "github.com/slice-soft/ss-keel-gorm"},
		},
	}

	tests := []struct {
		name         string
		target       string
		registry     *addon.Registry
		wantRepo     string
		wantOfficial bool
	}{
		{
			name:         "full repo path skips registry",
			target:       "github.com/acme/addon",
			registry:     reg,
			wantRepo:     "github.com/acme/addon",
			wantOfficial: false,
		},
		{
			name:         "official alias from registry",
			target:       "gorm",
			registry:     reg,
			wantRepo:     "github.com/slice-soft/ss-keel-gorm",
			wantOfficial: true,
		},
		{
			name:         "unknown alias without registry",
			target:       "custom",
			registry:     nil,
			wantRepo:     "custom",
			wantOfficial: false,
		},
		{
			name:         "unknown alias with registry",
			target:       "custom",
			registry:     reg,
			wantRepo:     "custom",
			wantOfficial: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRepo, gotOfficial := resolveRepo(tt.target, tt.registry)
			if gotRepo != tt.wantRepo || gotOfficial != tt.wantOfficial {
				t.Fatalf("resolveRepo(%q) = (%q, %t), want (%q, %t)", tt.target, gotRepo, gotOfficial, tt.wantRepo, tt.wantOfficial)
			}
		})
	}
}

func TestRunAddOfficialAlias(t *testing.T) {
	resetAddDeps(t)
	setupKeelProject(t)

	forceRefresh = true

	calledRegistry := false
	calledManifest := false
	calledInstall := false

	fetchRegistryFn = func(refresh bool) (*addon.Registry, error) {
		calledRegistry = true
		if !refresh {
			t.Fatalf("expected force refresh to be forwarded")
		}
		return &addon.Registry{
			Addons: []addon.RegistryEntry{
				{Alias: "gorm", Repo: "github.com/slice-soft/ss-keel-gorm"},
			},
		}, nil
	}
	fetchManifestFn = func(repo string) (*addon.Manifest, error) {
		calledManifest = true
		if repo != "github.com/slice-soft/ss-keel-gorm" {
			t.Fatalf("unexpected repo passed to fetchManifest: %q", repo)
		}
		return &addon.Manifest{Name: "gorm"}, nil
	}
	installAddonFn = func(m *addon.Manifest) error {
		calledInstall = true
		if m.Name != "gorm" {
			t.Fatalf("unexpected manifest passed to installer: %+v", m)
		}
		return nil
	}

	if err := runAdd(nil, []string{"gorm"}); err != nil {
		t.Fatalf("runAdd returned error: %v", err)
	}
	if !calledRegistry || !calledManifest || !calledInstall {
		t.Fatalf("expected all dependencies to be called, got registry=%t manifest=%t install=%t", calledRegistry, calledManifest, calledInstall)
	}
}

func TestRunAddCommunityAddonAbort(t *testing.T) {
	resetAddDeps(t)
	setupKeelProject(t)
	setStdin(t, "n\n")

	calledManifest := false
	calledInstall := false

	fetchRegistryFn = func(refresh bool) (*addon.Registry, error) {
		return nil, errors.New("registry down")
	}
	fetchManifestFn = func(repo string) (*addon.Manifest, error) {
		calledManifest = true
		return &addon.Manifest{Name: "custom"}, nil
	}
	installAddonFn = func(m *addon.Manifest) error {
		calledInstall = true
		return nil
	}

	if err := runAdd(nil, []string{"custom-addon"}); err != nil {
		t.Fatalf("runAdd returned error: %v", err)
	}
	if calledManifest || calledInstall {
		t.Fatalf("expected install flow to abort before fetching manifest")
	}
}

func TestRunAddCommunityAddonErrors(t *testing.T) {
	t.Run("manifest fetch error", func(t *testing.T) {
		resetAddDeps(t)
		setupKeelProject(t)
		setStdin(t, "y\n")

		wantErr := errors.New("manifest not found")

		fetchRegistryFn = func(refresh bool) (*addon.Registry, error) {
			return nil, nil
		}
		fetchManifestFn = func(repo string) (*addon.Manifest, error) {
			return nil, wantErr
		}
		installAddonFn = func(m *addon.Manifest) error {
			t.Fatalf("install should not be called on manifest error")
			return nil
		}

		err := runAdd(nil, []string{"github.com/acme/addon"})
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected manifest error, got %v", err)
		}
	})

	t.Run("install error", func(t *testing.T) {
		resetAddDeps(t)
		setupKeelProject(t)
		setStdin(t, "y\n")

		wantErr := errors.New("install failed")

		fetchRegistryFn = func(refresh bool) (*addon.Registry, error) {
			return nil, nil
		}
		fetchManifestFn = func(repo string) (*addon.Manifest, error) {
			return &addon.Manifest{Name: "custom"}, nil
		}
		installAddonFn = func(m *addon.Manifest) error {
			return wantErr
		}

		err := runAdd(nil, []string{"github.com/acme/addon"})
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected install error, got %v", err)
		}
	})
}

func TestRunAddInvalidProject(t *testing.T) {
	resetAddDeps(t)

	root := t.TempDir()
	withWorkingDir(t, root)

	fetchRegistryFn = func(refresh bool) (*addon.Registry, error) {
		t.Fatalf("registry should not be called for invalid project")
		return nil, nil
	}
	fetchManifestFn = func(repo string) (*addon.Manifest, error) {
		t.Fatalf("manifest should not be called for invalid project")
		return nil, nil
	}
	installAddonFn = func(m *addon.Manifest) error {
		t.Fatalf("installer should not be called for invalid project")
		return nil
	}

	err := runAdd(nil, []string{"gorm"})
	if err == nil || !strings.Contains(err.Error(), invalidProjectMessage) {
		t.Fatalf("expected invalid project error, got %v", err)
	}
}
