package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/slice-soft/keel/internal/addon"
)

func TestResolvePersistenceBackend(t *testing.T) {
	tests := []struct {
		name        string
		opts        Options
		wantBackend repositoryBackend
		wantFlag    bool
		wantErr     bool
	}{
		{
			name:        "no persistence flags",
			opts:        Options{},
			wantBackend: repositoryBackendStub,
		},
		{
			name:        "gorm flag",
			opts:        Options{UseGormPersistence: true},
			wantBackend: repositoryBackendGorm,
			wantFlag:    true,
		},
		{
			name:        "mongo flag",
			opts:        Options{UseMongoPersistence: true},
			wantBackend: repositoryBackendMongo,
			wantFlag:    true,
		},
		{
			name:    "conflicting flags",
			opts:    Options{UseMongoPersistence: true, UseGormPersistence: true},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBackend, gotFlag, err := resolvePersistenceBackend(tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotBackend != tt.wantBackend || gotFlag != tt.wantFlag {
				t.Fatalf("resolvePersistenceBackend(%#v) = (%s, %t), want (%s, %t)", tt.opts, gotBackend, gotFlag, tt.wantBackend, tt.wantFlag)
			}
		})
	}
}

func TestEnsurePersistenceAddonInstalled(t *testing.T) {
	t.Run("skips install when addon already exists", func(t *testing.T) {
		root := t.TempDir()
		oldWD, _ := os.Getwd()
		defer func() { _ = os.Chdir(oldWD) }()

		mustWrite(t, filepath.Join(root, "go.mod"), "module example.com/app\nrequire github.com/slice-soft/ss-keel-gorm v0.0.0\n")
		if err := os.Chdir(root); err != nil {
			t.Fatalf("chdir failed: %v", err)
		}

		previousInstallOfficialAddon := installOfficialAddonFn
		t.Cleanup(func() {
			installOfficialAddonFn = previousInstallOfficialAddon
		})

		installOfficialAddonFn = func(alias string, forceRefresh bool) (*addon.Manifest, error) {
			t.Fatalf("installOfficialAddonFn should not be called")
			return nil, nil
		}

		if err := ensurePersistenceAddonInstalled(repositoryBackendGorm); err != nil {
			t.Fatalf("ensurePersistenceAddonInstalled returned error: %v", err)
		}
	})

	t.Run("installs missing official addon", func(t *testing.T) {
		root := t.TempDir()
		oldWD, _ := os.Getwd()
		defer func() { _ = os.Chdir(oldWD) }()

		mustWrite(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
		if err := os.Chdir(root); err != nil {
			t.Fatalf("chdir failed: %v", err)
		}

		previousInstallOfficialAddon := installOfficialAddonFn
		t.Cleanup(func() {
			installOfficialAddonFn = previousInstallOfficialAddon
		})

		called := false
		installOfficialAddonFn = func(alias string, forceRefresh bool) (*addon.Manifest, error) {
			called = true
			if alias != "mongo" {
				t.Fatalf("unexpected alias: %s", alias)
			}
			if forceRefresh {
				t.Fatal("did not expect force refresh")
			}
			return &addon.Manifest{Name: "ss-keel-mongo"}, nil
		}

		if err := ensurePersistenceAddonInstalled(repositoryBackendMongo); err != nil {
			t.Fatalf("ensurePersistenceAddonInstalled returned error: %v", err)
		}
		if !called {
			t.Fatal("expected installOfficialAddonFn to be called")
		}
	})
}
