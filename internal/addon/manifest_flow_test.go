package addon

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestJWTAndOAuthManifestFlowCompiles(t *testing.T) {
	root := t.TempDir()
	withWorkingDir(t, root)
	resetExecCommand(t)

	flow := seedAddonFlowFixtures(t, root)
	seedAddonProject(t, root, flow)

	execCommand = func(name string, args ...string) *exec.Cmd {
		if name != "go" {
			t.Fatalf("unexpected command name: %s %#v", name, args)
		}
		return exec.Command("true")
	}

	if err := Install(flow.jwtManifest); err != nil {
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

	runTidyAndBuild(t, root, "jwt-only")

	if err := Install(flow.oauthManifest); err != nil {
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

	runTidyAndBuild(t, root, "jwt-oauth")
}

type addonFlowFixtures struct {
	coreDir       string
	jwtDir        string
	oauthDir      string
	jwtManifest   *Manifest
	oauthManifest *Manifest
}

func seedAddonFlowFixtures(t *testing.T, root string) addonFlowFixtures {
	t.Helper()

	depsRoot := filepath.Join(root, "testdeps")
	coreDir := filepath.Join(depsRoot, "ss-keel-core")
	jwtDir := filepath.Join(depsRoot, "ss-keel-jwt")
	oauthDir := filepath.Join(depsRoot, "ss-keel-oauth")

	seedStubKeelCore(t, coreDir)
	seedStubJWTAddon(t, jwtDir)
	seedStubOAuthAddon(t, oauthDir)

	return addonFlowFixtures{
		coreDir:       coreDir,
		jwtDir:        jwtDir,
		oauthDir:      oauthDir,
		jwtManifest:   jwtManifestFixture(),
		oauthManifest: oauthManifestFixture(),
	}
}

func seedAddonProject(t *testing.T, root string, flow addonFlowFixtures) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0755); err != nil {
		t.Fatalf("failed to create cmd directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal", "modules", "starter"), 0755); err != nil {
		t.Fatalf("failed to create starter module directory: %v", err)
	}

	goMod := `module example.com/demo

go 1.25.0

replace github.com/slice-soft/ss-keel-core => ` + flow.coreDir + `
replace github.com/slice-soft/ss-keel-jwt => ` + flow.jwtDir + `
replace github.com/slice-soft/ss-keel-oauth => ` + flow.oauthDir + `
`
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	mainGo := `package main

import (
	"github.com/slice-soft/ss-keel-core/core"
	"github.com/slice-soft/ss-keel-core/logger"
)

func main() {
	appLogger := logger.NewLogger(false)
	app := core.New(core.KConfig{
		Port:        7331,
		ServiceName: "demo",
		Env:         "development",
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

func jwtManifestFixture() *Manifest {
	return &Manifest{
		Name:    "ss-keel-jwt",
		Version: "0.1.0",
		Repo:    "github.com/slice-soft/ss-keel-jwt",
		Steps: []Step{
			{Type: "go_get", Package: "github.com/slice-soft/ss-keel-jwt"},
			{Type: "env", Key: "JWT_SECRET", Example: "change-me-in-production"},
			{
				Type:     "create_provider_file",
				Filename: "cmd/setup_jwt.go",
				Guard:    "func setupJWT(",
				Content: `package main

import (
	"github.com/slice-soft/ss-keel-core/core"
	"github.com/slice-soft/ss-keel-core/logger"
	"github.com/slice-soft/ss-keel-jwt/jwt"
)

func setupJWT(app *core.App, log *logger.Logger) *jwt.JWT {
	_ = app

	jwtProvider, err := jwt.New(jwt.Config{Logger: log})
	if err != nil {
		log.Error("failed to initialize JWT: %v", err)
	}
	return jwtProvider
}
`,
			},
			{
				Type:   "main_code",
				Anchor: "before_modules",
				Guard:  "setupJWT(",
				Code: `jwtProvider := setupJWT(app, appLogger)
// TODO: use jwtProvider.Middleware() to protect routes
_ = jwtProvider`,
			},
		},
	}
}

func oauthManifestFixture() *Manifest {
	return &Manifest{
		Name:      "ss-keel-oauth",
		Version:   "0.1.0",
		Repo:      "github.com/slice-soft/ss-keel-oauth",
		DependsOn: []string{"jwt"},
		Steps: []Step{
			{Type: "go_get", Package: "github.com/slice-soft/ss-keel-oauth"},
			{Type: "env", Key: "OAUTH_REDIRECT_BASE_URL", Example: "http://localhost:7331"},
			{
				Type:     "create_provider_file",
				Filename: "cmd/setup_oauth.go",
				Guard:    "func setupOAuth(",
				Content: `package main

import (
	"github.com/slice-soft/ss-keel-core/core"
	"github.com/slice-soft/ss-keel-core/logger"
	"github.com/slice-soft/ss-keel-jwt/jwt"
	"github.com/slice-soft/ss-keel-oauth/oauth"
)

func setupOAuth(app *core.App, jwtProvider *jwt.JWT, log *logger.Logger) {
	oauthManager := oauth.New(oauth.Config{
		Signer: jwtProvider,
		Logger: log,
	})
	app.RegisterController(oauth.NewController(oauthManager, "/auth"))
}
`,
			},
			{
				Type:    "main_code",
				Anchor:  "before_modules",
				Guard:   "jwtProvider := setupJWT(",
				Replace: "setupJWT(app, appLogger)",
				Code:    "jwtProvider := setupJWT(app, appLogger)",
			},
			{
				Type:    "main_code",
				Anchor:  "before_modules",
				Guard:   "setupOAuth(",
				Replace: "_ = jwtProvider",
				Code:    "setupOAuth(app, jwtProvider, appLogger)",
			},
		},
	}
}

func seedStubKeelCore(t *testing.T, dir string) {
	t.Helper()

	writeStubFile(t, filepath.Join(dir, "go.mod"), `module github.com/slice-soft/ss-keel-core

go 1.25.0
`)
	writeStubFile(t, filepath.Join(dir, "core", "core.go"), `package core

type KConfig struct {
	Port        int
	ServiceName string
	Env         string
}

type App struct{}

func New(KConfig) *App { return &App{} }

func (a *App) Listen() error { return nil }

func (a *App) RegisterController(any) {}
`)
	writeStubFile(t, filepath.Join(dir, "logger", "logger.go"), `package logger

type Logger struct{}

func NewLogger(bool) *Logger { return &Logger{} }

func (l *Logger) Error(string, ...any) {}
`)
}

func seedStubJWTAddon(t *testing.T, dir string) {
	t.Helper()

	writeStubFile(t, filepath.Join(dir, "go.mod"), `module github.com/slice-soft/ss-keel-jwt

go 1.25.0
`)
	writeStubFile(t, filepath.Join(dir, "jwt", "jwt.go"), `package jwt

type Config struct {
	Logger any
}

type JWT struct{}

func New(Config) (*JWT, error) { return &JWT{}, nil }
`)
}

func seedStubOAuthAddon(t *testing.T, dir string) {
	t.Helper()

	writeStubFile(t, filepath.Join(dir, "go.mod"), `module github.com/slice-soft/ss-keel-oauth

go 1.25.0
`)
	writeStubFile(t, filepath.Join(dir, "oauth", "oauth.go"), `package oauth

type Config struct {
	Signer any
	Logger any
}

type Manager struct{}

type Controller struct{}

func New(Config) *Manager { return &Manager{} }

func NewController(*Manager, string) *Controller { return &Controller{} }
`)
}

func writeStubFile(t *testing.T, path, body string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
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

func runTidyAndBuild(t *testing.T, root, cacheSuffix string) {
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
