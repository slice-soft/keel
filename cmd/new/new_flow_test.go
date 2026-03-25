package new

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	generator "github.com/slice-soft/keel/internal/generator/generate"
)

func TestNewCommandConfiguration(t *testing.T) {
	cmd := NewCommand()

	if cmd.Use != "new [project-name]" {
		t.Fatalf("unexpected command use: %q", cmd.Use)
	}
	if cmd.RunE == nil {
		t.Fatalf("expected RunE to be configured")
	}

	requiredFlags := []string{"without-starter-module", "with-folder-structure", "yes"}
	for _, flagName := range requiredFlags {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Fatalf("expected flag %q to be registered", flagName)
		}
	}
}

func TestRunNew(t *testing.T) {
	previousCollect := collectProjectSetupFn
	previousScaffold := scaffoldProjectFn
	previousRunPostSetup := runPostSetupFn
	t.Cleanup(func() {
		collectProjectSetupFn = previousCollect
		scaffoldProjectFn = previousScaffold
		runPostSetupFn = previousRunPostSetup
	})

	t.Run("returns setup error", func(t *testing.T) {
		collectProjectSetupFn = func(args []string) (projectSetup, error) {
			return projectSetup{}, errors.New("setup failed")
		}
		scaffoldProjectFn = scaffoldProject
		runPostSetupFn = runPostSetup

		err := runNew(nil, []string{"app"})
		if err == nil || err.Error() != "setup failed" {
			t.Fatalf("expected setup error, got %v", err)
		}
	})

	t.Run("returns scaffold error", func(t *testing.T) {
		collectProjectSetupFn = func(args []string) (projectSetup, error) {
			return projectSetup{appName: "app"}, nil
		}
		scaffoldProjectFn = func(setup projectSetup) error {
			return errors.New("scaffold failed")
		}
		runPostSetupFn = runPostSetup

		err := runNew(nil, []string{"app"})
		if err == nil || err.Error() != "scaffold failed" {
			t.Fatalf("expected scaffold error, got %v", err)
		}
	})

	t.Run("runs post setup on success", func(t *testing.T) {
		postSetupCalls := 0
		collectProjectSetupFn = func(args []string) (projectSetup, error) {
			return projectSetup{appName: "app"}, nil
		}
		scaffoldProjectFn = func(setup projectSetup) error {
			return nil
		}
		runPostSetupFn = func(setup projectSetup) {
			postSetupCalls++
		}

		if err := runNew(nil, []string{"app"}); err != nil {
			t.Fatalf("runNew returned error: %v", err)
		}
		if postSetupCalls != 1 {
			t.Fatalf("expected runPostSetup to run once, got %d", postSetupCalls)
		}
	})
}

func TestCollectProjectSetup(t *testing.T) {
	previousResolve := resolveProjectNameFn
	previousPromptModulePath := promptModulePathFn
	previousPromptAirSetup := promptAirSetupFn
	previousPromptYesNo := promptYesNoFn
	previousYesFlag := yesFlag
	previousWithoutStarter := withoutStarterModule
	previousWithFolder := withFolderStructure
	t.Cleanup(func() {
		resolveProjectNameFn = previousResolve
		promptModulePathFn = previousPromptModulePath
		promptAirSetupFn = previousPromptAirSetup
		promptYesNoFn = previousPromptYesNo
		yesFlag = previousYesFlag
		withoutStarterModule = previousWithoutStarter
		withFolderStructure = previousWithFolder
	})

	root := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	resolveProjectNameFn = func(args []string) (string, error) { return "demo", nil }
	promptModulePathFn = func(appName string) (string, error) { return "github.com/acme/demo", nil }
	promptAirSetupFn = func() (bool, bool, error) { return true, false, nil }
	promptYesNoFn = func(title string, defaultValue bool) (bool, error) {
		switch title {
		case "Include .env support?":
			return true, nil
		case "Initialize a new git repository?":
			return false, nil
		case "Install dependencies?":
			return true, nil
		default:
			return false, errors.New("unexpected prompt title")
		}
	}

	yesFlag = false
	withoutStarterModule = true
	withFolderStructure = true

	setup, err := collectProjectSetup(nil)
	if err != nil {
		t.Fatalf("collectProjectSetup returned error: %v", err)
	}

	if setup.appName != "demo" || setup.moduleName != "github.com/acme/demo" {
		t.Fatalf("unexpected setup values: %#v", setup)
	}
	if !setup.useAir || setup.includeAirConfig || !setup.useEnv || setup.initGit || !setup.installDeps {
		t.Fatalf("unexpected setup booleans: %#v", setup)
	}
	if !setup.withoutStarterModule || !setup.withFolderStructure {
		t.Fatalf("expected cli flags to propagate into setup: %#v", setup)
	}

	if err := os.Mkdir("demo", 0755); err != nil {
		t.Fatalf("failed to seed existing dir: %v", err)
	}
	_, err = collectProjectSetup(nil)
	if err == nil || err.Error() != "directory 'demo' already exists" {
		t.Fatalf("expected existing directory error, got %v", err)
	}

	t.Run("returns prompt module path error", func(t *testing.T) {
		if err := os.RemoveAll("demo"); err != nil {
			t.Fatalf("failed removing demo dir: %v", err)
		}
		promptModulePathFn = func(appName string) (string, error) {
			return "", errors.New("module prompt failed")
		}

		_, err := collectProjectSetup(nil)
		if err == nil || err.Error() != "module prompt failed" {
			t.Fatalf("expected prompt module path error, got %v", err)
		}
	})

	t.Run("returns prompt air setup error", func(t *testing.T) {
		promptModulePathFn = func(appName string) (string, error) { return "github.com/acme/demo", nil }
		promptAirSetupFn = func() (bool, bool, error) {
			return false, false, errors.New("air setup failed")
		}

		_, err := collectProjectSetup(nil)
		if err == nil || err.Error() != "air setup failed" {
			t.Fatalf("expected prompt air setup error, got %v", err)
		}
	})

	t.Run("returns yes no prompt errors", func(t *testing.T) {
		promptAirSetupFn = func() (bool, bool, error) { return true, false, nil }
		promptYesNoFn = func(title string, defaultValue bool) (bool, error) {
			return false, errors.New("yes no failed")
		}

		_, err := collectProjectSetup(nil)
		if err == nil || err.Error() != "yes no failed" {
			t.Fatalf("expected prompt yes/no error, got %v", err)
		}
	})
}

func TestCollectProjectSetupWithDefaultsAirBranches(t *testing.T) {
	previousAirInstalledFn := airInstalledFn
	previousInstallAirBinaryFn := installAirBinaryFn
	t.Cleanup(func() {
		airInstalledFn = previousAirInstalledFn
		installAirBinaryFn = previousInstallAirBinaryFn
	})

	root := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Run("air already installed", func(t *testing.T) {
		airInstalledFn = func() bool { return true }
		installCalled := false
		installAirBinaryFn = func() error {
			installCalled = true
			return nil
		}

		setup, err := collectProjectSetupWithDefaults("api")
		if err != nil {
			t.Fatalf("collectProjectSetupWithDefaults returned error: %v", err)
		}
		if setup.moduleName != "github.com/my-github-user/api" {
			t.Fatalf("unexpected module path: %q", setup.moduleName)
		}
		if !setup.skipInitialCommit {
			t.Fatalf("expected automatic mode to defer the initial commit")
		}
		if installCalled {
			t.Fatalf("did not expect install to run when air is already installed")
		}
	})

	t.Run("air missing and install fails", func(t *testing.T) {
		airInstalledFn = func() bool { return false }
		installCalled := false
		installAirBinaryFn = func() error {
			installCalled = true
			return errors.New("install failed")
		}

		setup, err := collectProjectSetupWithDefaults("worker")
		if err != nil {
			t.Fatalf("collectProjectSetupWithDefaults returned error: %v", err)
		}
		if setup.appName != "worker" {
			t.Fatalf("unexpected appName: %q", setup.appName)
		}
		if !setup.skipInitialCommit {
			t.Fatalf("expected automatic mode to defer the initial commit")
		}
		if !installCalled {
			t.Fatalf("expected install to run when air is missing")
		}
	})
}

func TestScaffoldProjectAndHelpers(t *testing.T) {
	t.Run("scaffold project writes files", func(t *testing.T) {
		root := t.TempDir()
		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get wd: %v", err)
		}
		defer func() { _ = os.Chdir(oldWD) }()
		if err := os.Chdir(root); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}

		setup := projectSetup{
			appName:              "demo",
			moduleName:           "github.com/acme/demo",
			useAir:               true,
			includeAirConfig:     true,
			useEnv:               true,
			withoutStarterModule: false,
			withFolderStructure:  true,
		}

		if err := scaffoldProject(setup); err != nil {
			t.Fatalf("scaffoldProject returned error: %v", err)
		}

		requiredPaths := []string{
			filepath.Join(root, "demo", "cmd", "main.go"),
			filepath.Join(root, "demo", "go.mod"),
			filepath.Join(root, "demo", "application.properties"),
			filepath.Join(root, "demo", "README.md"),
			filepath.Join(root, "demo", "internal", "modules", "starter", "module.go"),
			filepath.Join(root, "demo", "internal", "middleware"),
		}
		for _, path := range requiredPaths {
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("expected generated path %s: %v", path, err)
			}
		}
	})

	t.Run("render project files returns error for invalid destination", func(t *testing.T) {
		data := generator.NewProjectData("demo", "github.com/acme/demo", false, false, false, false, false)
		err := renderProjectFiles("bad\x00path", false, false, false, data)
		if err == nil {
			t.Fatalf("expected renderProjectFiles error for invalid path")
		}
	})

	t.Run("creates modules directory when starter module is disabled", func(t *testing.T) {
		root := t.TempDir()
		appName := filepath.Join(root, "demo")
		if err := createProjectDirectories(appName, true, true); err != nil {
			t.Fatalf("createProjectDirectories returned error: %v", err)
		}
		if _, err := os.Stat(filepath.Join(appName, "internal", "modules")); err != nil {
			t.Fatalf("expected internal/modules directory: %v", err)
		}
	})

	t.Run("returns nil when folder structure is disabled", func(t *testing.T) {
		root := t.TempDir()
		appName := filepath.Join(root, "demo")
		if err := createProjectDirectories(appName, false, false); err != nil {
			t.Fatalf("expected nil error when folder structure is disabled, got %v", err)
		}
	})
}

func TestRunPostSetupAndCreateInitialCommit(t *testing.T) {
	t.Run("runPostSetup executes init, tidy and initial commit paths", func(t *testing.T) {
		previousCommandRunner := commandRunner
		previousCreateInitialCommit := createInitialCommitFn
		t.Cleanup(func() {
			commandRunner = previousCommandRunner
			createInitialCommitFn = previousCreateInitialCommit
		})

		commandRunner = func(name string, args ...string) *exec.Cmd {
			return exec.Command("go", "version")
		}

		createInitialCommitCalls := 0
		createInitialCommitFn = func(projectDir string) error {
			createInitialCommitCalls++
			return nil
		}

		appDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(appDir, "go.mod"), []byte("module example.com/demo\n\ngo 1.25.7\n"), 0644); err != nil {
			t.Fatalf("failed to seed go.mod: %v", err)
		}

		runPostSetup(projectSetup{
			appName:     appDir,
			initGit:     true,
			installDeps: true,
		})

		if createInitialCommitCalls != 1 {
			t.Fatalf("expected createInitialCommit to be called once, got %d", createInitialCommitCalls)
		}
		goModContent, err := os.ReadFile(filepath.Join(appDir, "go.mod"))
		if err != nil {
			t.Fatalf("failed to read go.mod: %v", err)
		}
		if !strings.Contains(string(goModContent), "go 1.25\n") {
			t.Fatalf("expected normalized go directive, got %q", string(goModContent))
		}
	})

	t.Run("runPostSetup skips initial commit when requested", func(t *testing.T) {
		previousCommandRunner := commandRunner
		previousCreateInitialCommit := createInitialCommitFn
		t.Cleanup(func() {
			commandRunner = previousCommandRunner
			createInitialCommitFn = previousCreateInitialCommit
		})

		commandRunner = func(name string, args ...string) *exec.Cmd {
			return exec.Command("go", "version")
		}

		createInitialCommitCalls := 0
		createInitialCommitFn = func(projectDir string) error {
			createInitialCommitCalls++
			return nil
		}

		runPostSetup(projectSetup{
			appName:           t.TempDir(),
			initGit:           true,
			installDeps:       true,
			skipInitialCommit: true,
		})

		if createInitialCommitCalls != 0 {
			t.Fatalf("expected createInitialCommit to be skipped, got %d calls", createInitialCommitCalls)
		}
	})

	t.Run("createInitialCommit succeeds in git repository", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git not available in PATH")
		}

		dir := t.TempDir()
		if err := exec.Command("git", "init", dir).Run(); err != nil {
			t.Fatalf("git init failed: %v", err)
		}
		if err := exec.Command("git", "-C", dir, "config", "user.email", "test@example.com").Run(); err != nil {
			t.Fatalf("git config user.email failed: %v", err)
		}
		if err := exec.Command("git", "-C", dir, "config", "user.name", "Test User").Run(); err != nil {
			t.Fatalf("git config user.name failed: %v", err)
		}

		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# demo\n"), 0644); err != nil {
			t.Fatalf("failed to write project file: %v", err)
		}

		if err := createInitialCommit(dir); err != nil {
			t.Fatalf("createInitialCommit returned error: %v", err)
		}
	})

	t.Run("createInitialCommit returns error for missing directory", func(t *testing.T) {
		err := createInitialCommit(filepath.Join(t.TempDir(), "missing"))
		if err == nil {
			t.Fatalf("expected error for missing project directory")
		}
	})
}

func TestInstallAirBinary(t *testing.T) {
	previousCommandRunner := commandRunner
	t.Cleanup(func() {
		commandRunner = previousCommandRunner
	})

	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("go", "version")
	}
	if err := installAirBinary(); err != nil {
		t.Fatalf("expected successful installAirBinary with stubbed command, got %v", err)
	}

	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("go", "tool", "definitely-not-a-real-tool")
	}
	if err := installAirBinary(); err == nil {
		t.Fatalf("expected installAirBinary to return command error")
	}
}
