package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSyncFromApplicationProperties(t *testing.T) {
	withTempProjectDir(t, func() {
		writeFile(t, "application.properties", `
app.name=${SERVICE_NAME:demo}
jwt.secret=${JWT_SECRET:change-me}
jwt.issuer=${JWT_ISSUER:}
`)

		if err := runSync(nil, nil); err != nil {
			t.Fatalf("runSync returned error: %v", err)
		}

		content := readFile(t, ".env.example")
		for _, expected := range []string{
			"SERVICE_NAME=demo",
			"JWT_SECRET=change-me",
			"JWT_ISSUER=",
		} {
			if !strings.Contains(content, expected) {
				t.Fatalf("expected .env.example to contain %q, got:\n%s", expected, content)
			}
		}
	})
}

func TestRunGenerateFromApplicationProperties(t *testing.T) {
	withTempProjectDir(t, func() {
		writeFile(t, "application.properties", `
app.name=${SERVICE_NAME:demo}
jwt.secret=${JWT_SECRET:change-me}
jwt.issuer=${JWT_ISSUER}
`)

		if err := runGenerate(nil, nil); err != nil {
			t.Fatalf("runGenerate returned error: %v", err)
		}

		content := readFile(t, ".env")
		if !strings.Contains(content, "# SERVICE_NAME=demo") {
			t.Fatalf("expected optional SERVICE_NAME entry, got:\n%s", content)
		}
		if !strings.Contains(content, "# JWT_SECRET=change-me") {
			t.Fatalf("expected optional JWT_SECRET entry, got:\n%s", content)
		}
		if !strings.Contains(content, "JWT_ISSUER=") {
			t.Fatalf("expected required JWT_ISSUER entry, got:\n%s", content)
		}
	})
}

func TestRunCheckFromApplicationProperties(t *testing.T) {
	withTempProjectDir(t, func() {
		writeFile(t, "application.properties", `
app.name=${SERVICE_NAME:demo}
jwt.issuer=${JWT_ISSUER}
`)
		writeFile(t, ".env", "JWT_ISSUER=keel-demo\n")

		if err := runCheck(nil, nil); err != nil {
			t.Fatalf("runCheck returned error: %v", err)
		}
	})
}

func TestRunCheckMissingRequiredFromApplicationProperties(t *testing.T) {
	withTempProjectDir(t, func() {
		writeFile(t, "application.properties", "jwt.issuer=${JWT_ISSUER}\n")

		err := runCheck(nil, nil)
		if err == nil {
			t.Fatal("expected missing required env error")
		}
	})
}

func TestLegacyFallbackToKeelToml(t *testing.T) {
	withTempProjectDir(t, func() {
		writeFile(t, "keel.toml", `
[[env]]
key = "DATABASE_URL"
required = true
default = "./app.db"
`)

		if err := runSync(nil, nil); err != nil {
			t.Fatalf("runSync returned error: %v", err)
		}

		content := readFile(t, ".env.example")
		if !strings.Contains(content, "DATABASE_URL=./app.db") {
			t.Fatalf("expected legacy fallback in .env.example, got:\n%s", content)
		}
	})
}

func TestRunSyncReturnsErrorForInvalidApplicationProperties(t *testing.T) {
	withTempProjectDir(t, func() {
		writeFile(t, "application.properties", "invalid-line")

		err := runSync(nil, nil)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})
}

func TestRunSyncNoEnvVarsDeclared(t *testing.T) {
	withTempProjectDir(t, func() {
		writeFile(t, "application.properties", "app.name=demo\n")
		if err := runSync(nil, nil); err != nil {
			t.Fatalf("runSync returned error: %v", err)
		}
		if _, err := os.Stat(".env.example"); !os.IsNotExist(err) {
			t.Fatal("expected .env.example to not be created when no placeholders exist")
		}
	})
}

func TestRunSyncIdempotent(t *testing.T) {
	withTempProjectDir(t, func() {
		writeFile(t, "application.properties", "x=${DB_URL:postgres://localhost}\n")
		if err := runSync(nil, nil); err != nil {
			t.Fatalf("first runSync failed: %v", err)
		}
		first := readFile(t, ".env.example")
		if err := runSync(nil, nil); err != nil {
			t.Fatalf("second runSync failed: %v", err)
		}
		second := readFile(t, ".env.example")
		if first != second {
			t.Fatalf("runSync is not idempotent:\nfirst:\n%s\nsecond:\n%s", first, second)
		}
		if strings.Count(second, "DB_URL") != 1 {
			t.Fatalf("expected DB_URL to appear exactly once, got:\n%s", second)
		}
	})
}

func TestRunSyncWithSecretAndDescription(t *testing.T) {
	withTempProjectDir(t, func() {
		writeFile(t, "keel.toml", `
[[env]]
key = "API_SECRET"
required = true
secret = true
description = "External API secret key"
`)
		if err := runSync(nil, nil); err != nil {
			t.Fatalf("runSync returned error: %v", err)
		}
		content := readFile(t, ".env.example")
		if !strings.Contains(content, "# External API secret key") {
			t.Fatalf("expected description comment, got:\n%s", content)
		}
		if !strings.Contains(content, "API_SECRET=your-secret-here") {
			t.Fatalf("expected secret placeholder, got:\n%s", content)
		}
	})
}

func TestRunGenerateNoEnvVarsDeclared(t *testing.T) {
	withTempProjectDir(t, func() {
		writeFile(t, "application.properties", "app.name=demo\n")
		if err := runGenerate(nil, nil); err != nil {
			t.Fatalf("runGenerate returned error: %v", err)
		}
		if _, err := os.Stat(".env"); !os.IsNotExist(err) {
			t.Fatal("expected .env to not be created when no placeholders exist")
		}
	})
}

func TestRunGenerateIdempotent(t *testing.T) {
	withTempProjectDir(t, func() {
		writeFile(t, "application.properties", "x=${PORT:8080}\n")
		if err := runGenerate(nil, nil); err != nil {
			t.Fatalf("first runGenerate failed: %v", err)
		}
		first := readFile(t, ".env")
		if err := runGenerate(nil, nil); err != nil {
			t.Fatalf("second runGenerate failed: %v", err)
		}
		second := readFile(t, ".env")
		if first != second {
			t.Fatalf("runGenerate is not idempotent:\nfirst:\n%s\nsecond:\n%s", first, second)
		}
		if strings.Count(second, "PORT") != 1 {
			t.Fatalf("expected PORT to appear exactly once, got:\n%s", second)
		}
	})
}

func TestRunCheckViaOSEnv(t *testing.T) {
	withTempProjectDir(t, func() {
		writeFile(t, "application.properties", "x=${KEEL_TEST_VAR_OS}\n")
		t.Setenv("KEEL_TEST_VAR_OS", "present")

		if err := runCheck(nil, nil); err != nil {
			t.Fatalf("runCheck returned error when var is set in OS env: %v", err)
		}
	})
}

func TestRunCheckIntentionallyEmpty(t *testing.T) {
	withTempProjectDir(t, func() {
		writeFile(t, "application.properties", "x=${OPT_VAR:default}\n")
		writeFile(t, ".env", "OPT_VAR=\n")

		if err := runCheck(nil, nil); err != nil {
			t.Fatalf("runCheck should accept intentionally empty optional var: %v", err)
		}
	})
}

func TestRunCheckOptionalMissing(t *testing.T) {
	withTempProjectDir(t, func() {
		writeFile(t, "application.properties", "x=${OPT_VAR:default}\n")

		if err := runCheck(nil, nil); err != nil {
			t.Fatalf("runCheck should not fail for missing optional var: %v", err)
		}
	})
}

func TestExampleValue(t *testing.T) {
	cases := []struct {
		name string
		ev   declaredEnvVar
		want string
	}{
		{"secret without default", declaredEnvVar{Secret: true}, "your-secret-here"},
		{"secret with default", declaredEnvVar{Secret: true, HasDefault: true, Default: "val"}, "val"},
		{"has default", declaredEnvVar{HasDefault: true, Default: "abc"}, "abc"},
		{"required no default", declaredEnvVar{Required: true}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := exampleValue(tc.ev)
			if got != tc.want {
				t.Errorf("exampleValue(%+v) = %q, want %q", tc.ev, got, tc.want)
			}
		})
	}
}

func TestBuildComment(t *testing.T) {
	if got := buildComment(declaredEnvVar{}); got != "" {
		t.Errorf("expected empty comment for empty description, got %q", got)
	}
	if got := buildComment(declaredEnvVar{Description: "my desc"}); got != "# my desc" {
		t.Errorf("unexpected comment: %q", got)
	}
}

func withTempProjectDir(t *testing.T, fn func()) {
	t.Helper()

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	prevLoadApplicationPropertiesFn := loadApplicationPropertiesFn
	prevLoadKeelTomlFn := loadKeelTomlFn
	prevReadFileFn := readFileFn
	prevLookupOSEnvFn := lookupOSEnvFn
	prevStatFileFn := statFileFn
	t.Cleanup(func() {
		loadApplicationPropertiesFn = prevLoadApplicationPropertiesFn
		loadKeelTomlFn = prevLoadKeelTomlFn
		readFileFn = prevReadFileFn
		lookupOSEnvFn = prevLookupOSEnvFn
		statFileFn = prevStatFileFn
	})

	fn()
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimLeft(content, "\n")), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(content)
}
