package addon

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func resetExecCommand(t *testing.T) {
	t.Helper()
	previous := execCommand
	t.Cleanup(func() {
		execCommand = previous
	})
}

func writeMainFile(t *testing.T, root, body string) string {
	t.Helper()
	path := filepath.Join(root, "cmd", "main.go")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create cmd directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("failed to write cmd/main.go: %v", err)
	}
	return path
}

const sampleMain = `package main

import (
	"log"
)

func main() {
	log.Fatal(app.Listen())
}
`

const sampleMainWithModules = `package main

import (
	"log"
)

func main() {
	// Register your modules here:
	app.Use(starter.NewModule(appLogger))

	log.Fatal(app.Listen())
}
`

func TestInstall(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)
		resetExecCommand(t)
		writeMainFile(t, root, sampleMain)
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n\ngo 1.25.7\n"), 0644); err != nil {
			t.Fatalf("failed to seed go.mod: %v", err)
		}

		tidyCalled := false
		execCommand = func(name string, args ...string) *exec.Cmd {
			if name != "go" || len(args) != 2 || args[0] != "mod" || args[1] != "tidy" {
				t.Fatalf("unexpected command: %s %#v", name, args)
			}
			tidyCalled = true
			return exec.Command("true")
		}

		manifest := &Manifest{
			Name: "gorm",
			Steps: []Step{
				{Type: "property", Key: "database.url", Example: "${DATABASE_URL:./app.db}"},
				{Type: "env", Key: "DB_HOST", Example: "localhost"},
				{Type: "main_import", Path: "github.com/acme/addon"},
				{Type: "main_code", Guard: "app.Use(addon.Middleware())", Code: "app.Use(addon.Middleware())"},
			},
		}

		if err := Install(manifest); err != nil {
			t.Fatalf("Install returned error: %v", err)
		}

		envContent, err := os.ReadFile(filepath.Join(root, ".env"))
		if err != nil {
			t.Fatalf("failed to read .env: %v", err)
		}
		if !strings.Contains(string(envContent), "DB_HOST=localhost") {
			t.Fatalf("expected env var to be added, got %q", string(envContent))
		}
		envExampleContent, err := os.ReadFile(filepath.Join(root, ".env.example"))
		if err != nil {
			t.Fatalf("failed to read .env.example: %v", err)
		}
		if !strings.Contains(string(envExampleContent), "DB_HOST=localhost") {
			t.Fatalf("expected env example var to be added, got %q", string(envExampleContent))
		}
		propertiesContent, err := os.ReadFile(filepath.Join(root, "application.properties"))
		if err != nil {
			t.Fatalf("failed to read application.properties: %v", err)
		}
		if !strings.Contains(string(propertiesContent), "database.url=${DATABASE_URL:./app.db}") {
			t.Fatalf("expected property to be added, got %q", string(propertiesContent))
		}

		mainContent, err := os.ReadFile(filepath.Join(root, "cmd", "main.go"))
		if err != nil {
			t.Fatalf("failed to read cmd/main.go: %v", err)
		}
		text := string(mainContent)
		if !strings.Contains(text, `"github.com/acme/addon"`) {
			t.Fatalf("expected import to be added, got %q", text)
		}
		if !strings.Contains(text, "app.Use(addon.Middleware())") {
			t.Fatalf("expected code to be wired, got %q", text)
		}
		goModContent, err := os.ReadFile(filepath.Join(root, "go.mod"))
		if err != nil {
			t.Fatalf("failed to read go.mod: %v", err)
		}
		if !strings.Contains(string(goModContent), "go 1.25\n") {
			t.Fatalf("expected normalized go directive, got %q", string(goModContent))
		}
		if !tidyCalled {
			t.Fatalf("expected go mod tidy to be executed")
		}
	})

	t.Run("wraps failing step", func(t *testing.T) {
		err := Install(&Manifest{
			Name:  "broken",
			Steps: []Step{{Type: "unknown"}},
		})
		if err == nil || !strings.Contains(err.Error(), `step "unknown" failed`) {
			t.Fatalf("expected wrapped step error, got %v", err)
		}
	})

	t.Run("continues on tidy error", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)
		resetExecCommand(t)

		execCommand = func(name string, args ...string) *exec.Cmd {
			if name != "go" || len(args) != 2 || args[0] != "mod" || args[1] != "tidy" {
				t.Fatalf("unexpected command: %s %#v", name, args)
			}
			return exec.Command("false")
		}

		if err := Install(&Manifest{Name: "gorm"}); err != nil {
			t.Fatalf("expected install to continue on tidy failure, got %v", err)
		}
	})
}

func TestRunStep(t *testing.T) {
	t.Run("go_get dispatch", func(t *testing.T) {
		resetExecCommand(t)

		execCommand = func(name string, args ...string) *exec.Cmd {
			if name != "go" {
				t.Fatalf("unexpected command name: %s", name)
			}
			if len(args) != 2 || args[0] != "get" || args[1] != "github.com/acme/addon@latest" {
				t.Fatalf("unexpected args: %#v", args)
			}
			return exec.Command("true")
		}

		if err := runStep(Step{Type: "go_get", Package: "github.com/acme/addon"}, "addon"); err != nil {
			t.Fatalf("runStep(go_get) returned error: %v", err)
		}
	})

	t.Run("env dispatch", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)

		if err := runStep(Step{Type: "env", Key: "TOKEN", Example: "abc"}, "addon"); err != nil {
			t.Fatalf("runStep(env) returned error: %v", err)
		}
	})

	t.Run("property dispatch", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)

		if err := runStep(Step{Type: "property", Key: "redis.url", Example: "${REDIS_URL:redis://localhost:6379}"}, "addon"); err != nil {
			t.Fatalf("runStep(property) returned error: %v", err)
		}
	})

	t.Run("main_import dispatch", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)
		writeMainFile(t, root, sampleMain)

		if err := runStep(Step{Type: "main_import", Path: "github.com/acme/addon"}, "addon"); err != nil {
			t.Fatalf("runStep(main_import) returned error: %v", err)
		}
	})

	t.Run("main_code dispatch", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)
		writeMainFile(t, root, sampleMain)

		if err := runStep(Step{Type: "main_code", Code: "app.Use(x)", Guard: "app.Use(x)"}, "addon"); err != nil {
			t.Fatalf("runStep(main_code) returned error: %v", err)
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		err := runStep(Step{Type: "unknown"}, "addon")
		if err == nil || !strings.Contains(err.Error(), "unknown step type") {
			t.Fatalf("expected unknown type error, got %v", err)
		}
	})
}

func TestStepGoGet(t *testing.T) {
	t.Run("missing package", func(t *testing.T) {
		err := stepGoGet(Step{})
		if err == nil || !strings.Contains(err.Error(), "missing 'package'") {
			t.Fatalf("expected missing package error, got %v", err)
		}
	})

	t.Run("command error", func(t *testing.T) {
		resetExecCommand(t)
		execCommand = func(name string, args ...string) *exec.Cmd {
			return exec.Command("false")
		}

		err := stepGoGet(Step{Package: "github.com/acme/fail"})
		if err == nil {
			t.Fatalf("expected command error, got nil")
		}
	})

	t.Run("keeps explicit version", func(t *testing.T) {
		resetExecCommand(t)

		execCommand = func(name string, args ...string) *exec.Cmd {
			if name != "go" {
				t.Fatalf("unexpected command name: %s", name)
			}
			if len(args) != 2 || args[0] != "get" || args[1] != "github.com/acme/addon@v1.0.1" {
				t.Fatalf("unexpected args: %#v", args)
			}
			return exec.Command("true")
		}

		if err := stepGoGet(Step{Package: "github.com/acme/addon@v1.0.1"}); err != nil {
			t.Fatalf("stepGoGet returned error: %v", err)
		}
	})
}

func TestResolveGoGetTarget(t *testing.T) {
	tests := []struct {
		name string
		pkg  string
		want string
	}{
		{
			name: "adds latest when version is missing",
			pkg:  "github.com/acme/addon",
			want: "github.com/acme/addon@latest",
		},
		{
			name: "keeps explicit semver",
			pkg:  "github.com/acme/addon@v1.0.1",
			want: "github.com/acme/addon@v1.0.1",
		},
		{
			name: "keeps latest when already explicit",
			pkg:  "github.com/acme/addon@latest",
			want: "github.com/acme/addon@latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveGoGetTarget(tt.pkg)
			if got != tt.want {
				t.Fatalf("resolveGoGetTarget(%q) = %q, want %q", tt.pkg, got, tt.want)
			}
		})
	}
}

func TestStepEnv(t *testing.T) {
	t.Run("missing key", func(t *testing.T) {
		err := stepEnv(Step{})
		if err == nil || !strings.Contains(err.Error(), "missing 'key'") {
			t.Fatalf("expected missing key error, got %v", err)
		}
	})

	t.Run("adds value and is idempotent", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)

		if err := stepEnv(Step{Key: "API_KEY", Example: "secret"}); err != nil {
			t.Fatalf("first stepEnv returned error: %v", err)
		}
		if err := stepEnv(Step{Key: "API_KEY", Example: "other"}); err != nil {
			t.Fatalf("second stepEnv returned error: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(root, ".env"))
		if err != nil {
			t.Fatalf("failed to read .env: %v", err)
		}
		text := string(content)
		if strings.Count(text, "API_KEY=") != 1 {
			t.Fatalf("expected API_KEY once, got %q", text)
		}
		if !strings.Contains(text, "API_KEY=secret") {
			t.Fatalf("expected initial API_KEY value, got %q", text)
		}

		exampleContent, err := os.ReadFile(filepath.Join(root, ".env.example"))
		if err != nil {
			t.Fatalf("failed to read .env.example: %v", err)
		}
		exampleText := string(exampleContent)
		if strings.Count(exampleText, "API_KEY=") != 1 {
			t.Fatalf("expected API_KEY once in example env, got %q", exampleText)
		}
		if !strings.Contains(exampleText, "API_KEY=secret") {
			t.Fatalf("expected initial API_KEY value in .env.example, got %q", exampleText)
		}
	})

	t.Run("expands placeholders from existing env values", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)

		if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SERVICE_NAME=demo-app\n"), 0644); err != nil {
			t.Fatalf("failed to seed .env: %v", err)
		}

		if err := stepEnv(Step{Key: "JWT_ISSUER", Example: "{{SERVICE_NAME}}"}); err != nil {
			t.Fatalf("stepEnv returned error: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(root, ".env"))
		if err != nil {
			t.Fatalf("failed to read .env: %v", err)
		}
		if !strings.Contains(string(content), "JWT_ISSUER=demo-app") {
			t.Fatalf("expected expanded placeholder, got %q", string(content))
		}

		exampleContent, err := os.ReadFile(filepath.Join(root, ".env.example"))
		if err != nil {
			t.Fatalf("failed to read .env.example: %v", err)
		}
		if !strings.Contains(string(exampleContent), "JWT_ISSUER=demo-app") {
			t.Fatalf("expected expanded placeholder in .env.example, got %q", string(exampleContent))
		}
	})
}

func TestStepProperty(t *testing.T) {
	t.Run("missing key", func(t *testing.T) {
		err := stepProperty(Step{})
		if err == nil || !strings.Contains(err.Error(), "missing 'key'") {
			t.Fatalf("expected missing key error, got %v", err)
		}
	})

	t.Run("adds value and is idempotent", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)

		if err := stepProperty(Step{Key: "database.url", Example: "${DATABASE_URL:./app.db}"}); err != nil {
			t.Fatalf("first stepProperty returned error: %v", err)
		}
		if err := stepProperty(Step{Key: "database.url", Example: "${DATABASE_URL:./other.db}"}); err != nil {
			t.Fatalf("second stepProperty returned error: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(root, "application.properties"))
		if err != nil {
			t.Fatalf("failed to read application.properties: %v", err)
		}
		text := string(content)
		if strings.Count(text, "database.url=") != 1 {
			t.Fatalf("expected database.url once, got %q", text)
		}
		if !strings.Contains(text, "database.url=${DATABASE_URL:./app.db}") {
			t.Fatalf("expected initial database.url value, got %q", text)
		}
	})
}

func TestStepNote(t *testing.T) {
	t.Run("missing message", func(t *testing.T) {
		err := stepNote(Step{})
		if err == nil || !strings.Contains(err.Error(), "missing 'message'") {
			t.Fatalf("expected missing message error, got %v", err)
		}
	})

	t.Run("accepts description fallback", func(t *testing.T) {
		if err := stepNote(Step{Description: "next step"}); err != nil {
			t.Fatalf("stepNote returned error: %v", err)
		}
	})
}

func TestStepMainImport(t *testing.T) {
	t.Run("missing path", func(t *testing.T) {
		err := stepMainImport(Step{})
		if err == nil || !strings.Contains(err.Error(), "missing 'path'") {
			t.Fatalf("expected missing path error, got %v", err)
		}
	})

	t.Run("missing cmd main", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)

		err := stepMainImport(Step{Path: "github.com/acme/addon"})
		if err == nil || !strings.Contains(err.Error(), "run keel add inside a Keel project") {
			t.Fatalf("expected missing main.go error, got %v", err)
		}
	})

	t.Run("adds import once", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)
		writeMainFile(t, root, sampleMain)

		step := Step{Path: "github.com/acme/addon"}
		if err := stepMainImport(step); err != nil {
			t.Fatalf("first stepMainImport returned error: %v", err)
		}
		if err := stepMainImport(step); err != nil {
			t.Fatalf("second stepMainImport returned error: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(root, "cmd", "main.go"))
		if err != nil {
			t.Fatalf("failed to read main.go: %v", err)
		}
		if strings.Count(string(content), `"github.com/acme/addon"`) != 1 {
			t.Fatalf("expected import once, got %q", string(content))
		}
	})
}

func TestStepMainCode(t *testing.T) {
	t.Run("missing code", func(t *testing.T) {
		err := stepMainCode(Step{})
		if err == nil || !strings.Contains(err.Error(), "missing 'code'") {
			t.Fatalf("expected missing code error, got %v", err)
		}
	})

	t.Run("adds code before listen and guards duplicates", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)
		writeMainFile(t, root, sampleMain)

		step := Step{
			Code:  "app.Use(gorm.Middleware())",
			Guard: "app.Use(gorm.Middleware())",
		}

		if err := stepMainCode(step); err != nil {
			t.Fatalf("first stepMainCode returned error: %v", err)
		}
		if err := stepMainCode(step); err != nil {
			t.Fatalf("second stepMainCode returned error: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(root, "cmd", "main.go"))
		if err != nil {
			t.Fatalf("failed to read main.go: %v", err)
		}
		text := string(content)
		if strings.Count(text, "app.Use(gorm.Middleware())") != 1 {
			t.Fatalf("expected guard code once, got %q", text)
		}
		if strings.Index(text, "app.Use(gorm.Middleware())") > strings.Index(text, "log.Fatal(app.Listen())") {
			t.Fatalf("expected inserted code before app.Listen call, got %q", text)
		}
	})

	t.Run("supports before_modules anchor", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)
		writeMainFile(t, root, sampleMainWithModules)

		step := Step{
			Anchor: "before_modules",
			Code:   "db := setupDatabase(app, appLogger)\ndefer db.Close()",
			Guard:  "setupDatabase(",
		}

		if err := stepMainCode(step); err != nil {
			t.Fatalf("stepMainCode returned error: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(root, "cmd", "main.go"))
		if err != nil {
			t.Fatalf("failed to read main.go: %v", err)
		}
		text := string(content)
		if strings.Index(text, "db := setupDatabase(app, appLogger)") > strings.Index(text, "// Register your modules here:") {
			t.Fatalf("expected setup code before module section, got %q", text)
		}
	})

	t.Run("replaces matching line before inserting", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)
		writeMainFile(t, root, sampleMainWithModules)

		if err := updateMainGo(func(content string) string {
			return strings.Replace(content, "// Register your modules here:\n", "_ = setupProvider(app)\n\n\t// Register your modules here:\n", 1)
		}); err != nil {
			t.Fatalf("failed to seed provider setup line: %v", err)
		}

		step := Step{
			Anchor:  "before_modules",
			Guard:   "provider := setupProvider(",
			Replace: "setupProvider(app)",
			Code:    "provider := setupProvider(app)",
		}

		if err := stepMainCode(step); err != nil {
			t.Fatalf("stepMainCode returned error: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(root, "cmd", "main.go"))
		if err != nil {
			t.Fatalf("failed to read main.go: %v", err)
		}
		text := string(content)
		if strings.Contains(text, "_ = setupProvider(app)") {
			t.Fatalf("expected placeholder provider setup to be replaced, got %q", text)
		}
		if strings.Count(text, "provider := setupProvider(app)") != 1 {
			t.Fatalf("expected rewritten provider line once, got %q", text)
		}
	})

	t.Run("replaces placeholder discard when wiring dependent code", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)
		writeMainFile(t, root, sampleMainWithModules)

		if err := updateMainGo(func(content string) string {
			return strings.Replace(content, "// Register your modules here:\n", "provider := setupProvider(app)\n\t_ = provider\n\n\t// Register your modules here:\n", 1)
		}); err != nil {
			t.Fatalf("failed to seed provider wiring: %v", err)
		}

		step := Step{
			Anchor:  "before_modules",
			Guard:   "wireFeature(",
			Replace: "_ = provider",
			Code:    "wireFeature(app, provider)",
		}

		if err := stepMainCode(step); err != nil {
			t.Fatalf("stepMainCode returned error: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(root, "cmd", "main.go"))
		if err != nil {
			t.Fatalf("failed to read main.go: %v", err)
		}
		text := string(content)
		if strings.Contains(text, "_ = provider") {
			t.Fatalf("expected placeholder provider discard to be replaced, got %q", text)
		}
		if strings.Count(text, "wireFeature(app, provider)") != 1 {
			t.Fatalf("expected dependent wiring once, got %q", text)
		}
		if strings.Count(text, "provider := setupProvider(app)") != 1 {
			t.Fatalf("expected provider wiring to remain once, got %q", text)
		}
	})
}

func TestUpdateMainGo(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)

		err := updateMainGo(func(content string) string { return content + "\n" })
		if err == nil || !strings.Contains(err.Error(), "cmd/main.go not found") {
			t.Fatalf("expected missing file error, got %v", err)
		}
	})

	t.Run("no changes", func(t *testing.T) {
		root := t.TempDir()
		withWorkingDir(t, root)
		mainPath := writeMainFile(t, root, sampleMain)

		before, err := os.ReadFile(mainPath)
		if err != nil {
			t.Fatalf("failed to read main.go before update: %v", err)
		}

		if err := updateMainGo(func(content string) string { return content }); err != nil {
			t.Fatalf("expected nil error on no-op transform, got %v", err)
		}

		after, err := os.ReadFile(mainPath)
		if err != nil {
			t.Fatalf("failed to read main.go after update: %v", err)
		}
		if string(after) != string(before) {
			t.Fatalf("expected main.go to remain unchanged")
		}
	})
}

func TestAddImport(t *testing.T) {
	t.Run("adds import in block", func(t *testing.T) {
		original := "package main\n\nimport (\n\t\"log\"\n)\n"
		updated := addImport(original, `"github.com/acme/addon"`)
		if !strings.Contains(updated, `"github.com/acme/addon"`) {
			t.Fatalf("expected import to be added, got %q", updated)
		}
	})

	t.Run("returns original when import block missing", func(t *testing.T) {
		original := "package main\n"
		updated := addImport(original, `"github.com/acme/addon"`)
		if updated != original {
			t.Fatalf("expected unchanged content, got %q", updated)
		}
	})
}

func TestAddMainLine(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "tabbed log fatal marker",
			content: "func main() {\n\tlog.Fatal(app.Listen())\n}\n",
		},
		{
			name:    "if err marker",
			content: "func main() {\n\tif err := app.Listen(); err != nil {\n\t\tpanic(err)\n\t}\n}\n",
		},
		{
			name:    "non-tabbed marker",
			content: "func main() {\nlog.Fatal(app.Listen())\n}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated := addMainLine(tt.content, "\tapp.Use(middleware)")
			if !strings.Contains(updated, "app.Use(middleware)") {
				t.Fatalf("expected line insertion, got %q", updated)
			}
		})
	}

	t.Run("returns original when marker missing", func(t *testing.T) {
		original := "func main() {\n\tprintln(\"hello\")\n}\n"
		updated := addMainLine(original, "\tapp.Use(middleware)")
		if updated != original {
			t.Fatalf("expected unchanged content when marker is missing")
		}
	})
}
