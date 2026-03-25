package keeltoml_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slice-soft/keel/internal/keeltoml"
)

// ---- Parse ----------------------------------------------------------------

func TestParse_EmptyFile(t *testing.T) {
	kt, err := keeltoml.Parse([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(kt.Addons) != 0 || len(kt.Env) != 0 {
		t.Fatalf("expected empty slices, got addons=%d env=%d", len(kt.Addons), len(kt.Env))
	}
}

func TestParse_InvalidTOML(t *testing.T) {
	_, err := keeltoml.Parse([]byte("[[this is not valid toml"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParse_AddonsAndEnv(t *testing.T) {
	raw := `
[[addons]]
id           = "gorm"
version      = "1.2.0"
repo         = "github.com/slice-soft/ss-keel-gorm"
capabilities = ["database"]
resources    = ["postgres"]

[[env]]
key      = "DB_DSN"
source   = "gorm"
required = true
secret   = true
description = "PostgreSQL DSN"

[[env]]
key      = "PORT"
source   = "core"
required = false
default  = "8080"
`
	kt, err := keeltoml.Parse([]byte(raw))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(kt.Addons) != 1 {
		t.Fatalf("expected 1 addon, got %d", len(kt.Addons))
	}
	a := kt.Addons[0]
	if a.ID != "gorm" {
		t.Errorf("addon ID: want gorm, got %s", a.ID)
	}
	if a.Version != "1.2.0" {
		t.Errorf("addon Version: want 1.2.0, got %s", a.Version)
	}
	if len(a.Capabilities) != 1 || a.Capabilities[0] != "database" {
		t.Errorf("addon Capabilities: want [database], got %v", a.Capabilities)
	}

	if len(kt.Env) != 2 {
		t.Fatalf("expected 2 env entries, got %d", len(kt.Env))
	}
	if kt.Env[0].Key != "DB_DSN" || !kt.Env[0].Required || !kt.Env[0].Secret {
		t.Errorf("env[0] mismatch: %+v", kt.Env[0])
	}
	if kt.Env[1].Key != "PORT" || kt.Env[1].Required || kt.Env[1].Default != "8080" {
		t.Errorf("env[1] mismatch: %+v", kt.Env[1])
	}
}

func TestParse_UnknownSectionsIgnored(t *testing.T) {
	raw := `
[app]
name    = "myapp"
version = "1.0.0"

[scripts]
dev   = "air"
build = "go build ./..."

[features]
air = true

[[addons]]
id = "redis"
`
	kt, err := keeltoml.Parse([]byte(raw))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(kt.Addons) != 1 || kt.Addons[0].ID != "redis" {
		t.Errorf("expected addons=[redis], got %v", kt.Addons)
	}
}

// ---- Load -----------------------------------------------------------------

func TestLoad_FileNotExist(t *testing.T) {
	kt, err := keeltoml.Load("/nonexistent/keel.toml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if kt == nil {
		t.Fatal("expected non-nil KeelToml")
	}
}

func TestLoad_RealFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keel.toml")
	content := `
[[addons]]
id = "jwt"

[[env]]
key      = "JWT_SECRET"
required = true
secret   = true
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	kt, err := keeltoml.Load(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if len(kt.Addons) != 1 || kt.Addons[0].ID != "jwt" {
		t.Errorf("unexpected addons: %v", kt.Addons)
	}
	if len(kt.Env) != 1 || kt.Env[0].Key != "JWT_SECRET" {
		t.Errorf("unexpected env: %v", kt.Env)
	}
}

// ---- MergeAddon -----------------------------------------------------------

func TestMergeAddon_NewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keel.toml")

	envVars := []keeltoml.EnvEntry{
		{Key: "DB_DSN", Source: "gorm", Required: true, Secret: true, Description: "PostgreSQL DSN"},
	}

	changed, err := keeltoml.MergeAddon(path, "gorm", "1.2.0", "github.com/slice-soft/ss-keel-gorm",
		[]string{"database"}, []string{"postgres"}, envVars)
	if err != nil {
		t.Fatalf("merge error: %v", err)
	}
	if !changed {
		t.Error("expected changed=true for new file")
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	assertContains(t, content, `id           = "gorm"`)
	assertContains(t, content, `version      = "1.2.0"`)
	assertContains(t, content, `repo         = "github.com/slice-soft/ss-keel-gorm"`)
	assertContains(t, content, `capabilities = ["database"]`)
	assertContains(t, content, `resources    = ["postgres"]`)
	assertContains(t, content, `key      = "DB_DSN"`)
	assertContains(t, content, `required = true`)
	assertContains(t, content, `secret   = true`)
}

func TestMergeAddon_AppendsToExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keel.toml")

	existing := `[app]
name = "myapp"
version = "1.0.0"
`
	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	changed, err := keeltoml.MergeAddon(path, "redis", "2.0.0", "github.com/slice-soft/ss-keel-redis",
		[]string{"cache"}, []string{"redis"}, nil)
	if err != nil {
		t.Fatalf("merge error: %v", err)
	}
	if !changed {
		t.Error("expected changed=true")
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	// Original content preserved
	assertContains(t, content, `name = "myapp"`)
	// New content appended
	assertContains(t, content, `id           = "redis"`)
}

func TestMergeAddon_NoDuplicateAddon(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keel.toml")

	// First install
	_, err := keeltoml.MergeAddon(path, "gorm", "1.0.0", "", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Second install — same addon
	changed, err := keeltoml.MergeAddon(path, "gorm", "1.0.0", "", nil, nil, nil)
	if err != nil {
		t.Fatalf("second merge error: %v", err)
	}
	if changed {
		t.Error("expected changed=false on duplicate addon")
	}

	data, _ := os.ReadFile(path)
	count := strings.Count(string(data), `id           = "gorm"`)
	if count != 1 {
		t.Errorf("expected 1 addon entry, found %d", count)
	}
}

func TestMergeAddon_NoDuplicateEnvKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keel.toml")

	env1 := []keeltoml.EnvEntry{{Key: "DB_HOST", Source: "gorm", Required: true}}

	// First merge
	if _, err := keeltoml.MergeAddon(path, "gorm", "1.0.0", "", nil, nil, env1); err != nil {
		t.Fatal(err)
	}

	// Second merge with same env key (different addon, same key)
	env2 := []keeltoml.EnvEntry{{Key: "DB_HOST", Source: "gorm", Required: true}}
	changed, err := keeltoml.MergeAddon(path, "gorm", "1.0.0", "", nil, nil, env2)
	if err != nil {
		t.Fatal(err)
	}
	// Addon already exists and env key already exists → nothing changes
	if changed {
		t.Error("expected changed=false when addon and all env keys already present")
	}

	data, _ := os.ReadFile(path)
	count := strings.Count(string(data), `key      = "DB_HOST"`)
	if count != 1 {
		t.Errorf("expected 1 DB_HOST entry, found %d", count)
	}
}

func TestMergeAddon_MultipleAddons(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keel.toml")

	envGorm := []keeltoml.EnvEntry{{Key: "DB_DSN", Source: "gorm", Required: true, Secret: true}}
	envRedis := []keeltoml.EnvEntry{{Key: "REDIS_URL", Source: "redis", Required: true}}

	if _, err := keeltoml.MergeAddon(path, "gorm", "1.0.0", "github.com/slice-soft/ss-keel-gorm",
		[]string{"database"}, []string{"postgres"}, envGorm); err != nil {
		t.Fatal(err)
	}
	if _, err := keeltoml.MergeAddon(path, "redis", "2.0.0", "github.com/slice-soft/ss-keel-redis",
		[]string{"cache"}, []string{"redis"}, envRedis); err != nil {
		t.Fatal(err)
	}

	kt, err := keeltoml.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(kt.Addons) != 2 {
		t.Errorf("expected 2 addons, got %d", len(kt.Addons))
	}
	if len(kt.Env) != 2 {
		t.Errorf("expected 2 env entries, got %d", len(kt.Env))
	}
}

func TestMergeAddon_NoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keel.toml")

	// File without trailing newline
	if err := os.WriteFile(path, []byte("[app]\nname = \"app\""), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := keeltoml.MergeAddon(path, "jwt", "", "", nil, nil, nil); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	// Should be valid TOML
	if _, err := keeltoml.Parse(data); err != nil {
		t.Errorf("result is not valid TOML: %v\n%s", err, data)
	}
}

// ---- LookupEnvValue -------------------------------------------------------

func TestLookupEnvValue(t *testing.T) {
	content := `
APP_ENV=production
DB_HOST=localhost
# COMMENTED=value
EMPTY=
`
	tests := []struct {
		key      string
		wantVal  string
		wantFound bool
	}{
		{"APP_ENV", "production", true},
		{"DB_HOST", "localhost", true},
		{"COMMENTED", "", false},
		{"EMPTY", "", true},
		{"MISSING", "", false},
	}
	for _, tt := range tests {
		val, found := keeltoml.LookupEnvValue(content, tt.key)
		if found != tt.wantFound || val != tt.wantVal {
			t.Errorf("LookupEnvValue(%q): got (%q, %v), want (%q, %v)",
				tt.key, val, found, tt.wantVal, tt.wantFound)
		}
	}
}

// ---- helpers --------------------------------------------------------------

func assertContains(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Errorf("expected content to contain %q\n\ncontent:\n%s", sub, s)
	}
}
