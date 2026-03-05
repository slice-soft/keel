package generate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandConfiguration(t *testing.T) {
	cmd := NewCommand()
	if cmd.Use != "generate [type] [name]" {
		t.Fatalf("unexpected command use: %q", cmd.Use)
	}
	if cmd.RunE == nil {
		t.Fatalf("expected RunE to be configured")
	}

	requiredFlags := []string{"transactional", "with-repository", "in-main"}
	for _, name := range requiredFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("expected flag %q to exist", name)
		}
	}
}

func TestRunGenerateDelegatesToExecute(t *testing.T) {
	previousExecute := executeFn
	previousTransactional := transactionalModule
	previousWithRepository := withRepository
	previousInMain := inMain
	t.Cleanup(func() {
		executeFn = previousExecute
		transactionalModule = previousTransactional
		withRepository = previousWithRepository
		inMain = previousInMain
	})

	transactionalModule = true
	withRepository = true
	inMain = true

	called := false
	executeFn = func(genType, rawName string, opts Options) error {
		called = true
		if genType != "module" || rawName != "users" {
			t.Fatalf("unexpected args: %s %s", genType, rawName)
		}
		if !opts.TransactionalModule || !opts.WithRepository || !opts.ControllerInMain {
			t.Fatalf("unexpected opts: %#v", opts)
		}
		return nil
	}

	if err := runGenerate(nil, []string{"module", "users"}); err != nil {
		t.Fatalf("runGenerate returned error: %v", err)
	}
	if !called {
		t.Fatalf("expected execute to be called")
	}

	executeFn = func(genType, rawName string, opts Options) error {
		return errors.New("execute failed")
	}
	if err := runGenerate(nil, []string{"module", "users"}); err == nil {
		t.Fatalf("expected delegated execute error")
	}
}

func TestBuildEventFiles(t *testing.T) {
	files := buildEventFiles("user-created", filepath.Join("internal", "events"))
	if len(files) != 3 {
		t.Fatalf("expected 3 event files, got %d", len(files))
	}

	destinations := []string{files[0].dest, files[1].dest, files[2].dest}
	joined := strings.Join(destinations, ",")
	if !strings.Contains(joined, "user_created_publisher.go") ||
		!strings.Contains(joined, "user_created_subscriber.go") ||
		!strings.Contains(joined, "user_created_event_test.go") {
		t.Fatalf("unexpected event file destinations: %v", destinations)
	}
}

func TestGenerateUnsupportedBranches(t *testing.T) {
	if err := generateStandalone(typeModule, "users", Options{}); err == nil {
		t.Fatalf("expected generateStandalone to reject unsupported standalone module generation")
	}

	if err := generateInModule(typeScheduler, "users", "nightly"); err == nil {
		t.Fatalf("expected generateInModule to reject scheduler in module format")
	}
}

func TestEnsureModuleExistsDetectsPackageMismatch(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	seedProject(t, root)
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	moduleDir := filepath.Join(root, "internal", "modules", "users")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "bad.go"), []byte("package wrong\n"), 0644); err != nil {
		t.Fatalf("failed writing mismatched package file: %v", err)
	}

	err := ensureModuleExists("users")
	if err == nil || !strings.Contains(err.Error(), "module package mismatch") {
		t.Fatalf("expected package mismatch error, got %v", err)
	}
}
