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
