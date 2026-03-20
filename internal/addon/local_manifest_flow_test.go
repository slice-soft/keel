package addon

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLocalJWTAndOAuthManifestFlowCompiles(t *testing.T) {
	workspaceRoot := testWorkspaceRoot(t)
	root := t.TempDir()
	withWorkingDir(t, root)
	resetExecCommand(t)

	seedLocalAddonProject(t, root, workspaceRoot)

	execCommand = func(name string, args ...string) *exec.Cmd {
		if name != "go" {
			t.Fatalf("unexpected command name: %s %#v", name, args)
		}
		return exec.Command("true")
	}

	jwtManifest := readManifestFile(t, filepath.Join(workspaceRoot, "ss-keel-jwt", "keel-addon.json"))
	if err := Install(jwtManifest); err != nil {
		t.Fatalf("jwt Install returned error: %v", err)
	}

	jwtMain := mustReadMain(t, root)
	if !strings.Contains(jwtMain, "jwtProvider := setupJWT(app, appLogger)") {
		t.Fatalf("expected standalone jwt wiring to capture provider, got:\n%s", jwtMain)
	}
	if !strings.Contains(jwtMain, "_ = jwtProvider") {
		t.Fatalf("expected standalone jwt wiring to suppress the unused provider, got:\n%s", jwtMain)
	}
	if strings.Contains(jwtMain, "_ = setupJWT(app, appLogger)") {
		t.Fatalf("expected placeholder jwt setup to be removed, got:\n%s", jwtMain)
	}

	runLocalTidyAndBuild(t, root, "jwt-only")

	oauthManifest := readManifestFile(t, filepath.Join(workspaceRoot, "ss-keel-oauth", "keel-addon.json"))
	if err := Install(oauthManifest); err != nil {
		t.Fatalf("oauth Install returned error: %v", err)
	}

	oauthMain := mustReadMain(t, root)
	if !strings.Contains(oauthMain, "jwtProvider := setupJWT(app, appLogger)") {
		t.Fatalf("expected oauth wiring to keep the jwt provider, got:\n%s", oauthMain)
	}
	if !strings.Contains(oauthMain, "setupOAuth(app, jwtProvider, appLogger)") {
		t.Fatalf("expected oauth wiring to use the jwt provider, got:\n%s", oauthMain)
	}
	if strings.Contains(oauthMain, "_ = jwtProvider") {
		t.Fatalf("expected oauth wiring to replace the standalone jwt placeholder, got:\n%s", oauthMain)
	}

	runLocalTidyAndBuild(t, root, "jwt-oauth")
}

func testWorkspaceRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

func readManifestFile(t *testing.T, path string) *Manifest {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read manifest %s: %v", path, err)
	}

	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("failed to decode manifest %s: %v", path, err)
	}
	return &manifest
}

func seedLocalAddonProject(t *testing.T, root, workspaceRoot string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0755); err != nil {
		t.Fatalf("failed to create cmd directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal", "modules", "starter"), 0755); err != nil {
		t.Fatalf("failed to create starter module directory: %v", err)
	}

	goMod := `module example.com/demo

go 1.25.0

replace github.com/slice-soft/ss-keel-core => ` + filepath.Join(workspaceRoot, "ss-keel-core") + `
replace github.com/slice-soft/ss-keel-jwt => ` + filepath.Join(workspaceRoot, "ss-keel-jwt") + `
replace github.com/slice-soft/ss-keel-oauth => ` + filepath.Join(workspaceRoot, "ss-keel-oauth") + `
`
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	mainGo := `package main

import (
	"github.com/slice-soft/ss-keel-core/config"
	"github.com/slice-soft/ss-keel-core/core"
	"github.com/slice-soft/ss-keel-core/logger"
)

func main() {
	appLogger := logger.NewLogger(config.GetEnvOrDefault("APP_ENV", "development") == "production")
	app := core.New(core.KConfig{
		Port:        config.GetEnvIntOrDefault("PORT", 7331),
		ServiceName: config.GetEnvOrDefault("SERVICE_NAME", "demo"),
		Env:         config.GetEnvOrDefault("APP_ENV", "development"),
	})

	// Register your modules here:

	if err := app.Listen(); err != nil {
		appLogger.Error("failed to start app: %v", err)
	}
}
`
	if err := os.WriteFile(filepath.Join(root, "cmd", "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatalf("failed to write cmd/main.go: %v", err)
	}

	starterModule := `package starter

type Module struct{}
`
	if err := os.WriteFile(filepath.Join(root, "internal", "modules", "starter", "module.go"), []byte(starterModule), 0644); err != nil {
		t.Fatalf("failed to write starter module: %v", err)
	}
}

func mustReadMain(t *testing.T, root string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(root, "cmd", "main.go"))
	if err != nil {
		t.Fatalf("failed to read cmd/main.go: %v", err)
	}
	return string(content)
}

func runLocalTidyAndBuild(t *testing.T, root, cacheSuffix string) {
	t.Helper()

	env := append(os.Environ(),
		"GOCACHE="+filepath.Join(os.TempDir(), "keel-addon-flow-"+cacheSuffix),
		"GOWORK=off",
	)

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = root
	tidyCmd.Env = env
	tidyOutput, err := tidyCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, string(tidyOutput))
	}

	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = root
	buildCmd.Env = env
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build ./... failed: %v\n%s", err, string(buildOutput))
	}
}
