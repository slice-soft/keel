package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseName(t *testing.T) {
	t.Run("module component", func(t *testing.T) {
		got, err := parseName("users/validate")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.moduleName != "users" || got.componentName != "validate" || got.standalone {
			t.Fatalf("unexpected parse result: %#v", got)
		}
	})

	t.Run("standalone", func(t *testing.T) {
		got, err := parseName("validate-email")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.standalone || got.componentName != "validate-email" {
			t.Fatalf("unexpected parse result: %#v", got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		if _, err := parseName("users//bad"); err == nil {
			t.Fatal("expected parse error")
		}
	})
}

func TestResolveTypeAlias(t *testing.T) {
	tests := map[string]string{
		"service": typeService,
		"s":       typeService,
		"svc":     typeService,
		"c":       typeController,
		"r":       typeRepository,
		"m":       typeModule,
	}

	for in, want := range tests {
		got, err := resolveType(in)
		if err != nil {
			t.Fatalf("resolveType(%q) returned error: %v", in, err)
		}
		if got != want {
			t.Fatalf("resolveType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateKeelProject(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	if err := validateKeelProject(); err == nil {
		t.Fatal("expected invalid project error")
	}

	mustWrite(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	mustMkdir(t, filepath.Join(root, "cmd"))
	mustWrite(t, filepath.Join(root, "cmd", "main.go"), "package main\nfunc main(){}\n")
	mustMkdir(t, filepath.Join(root, "internal"))

	if err := validateKeelProject(); err != nil {
		t.Fatalf("expected valid project, got %v", err)
	}
}

func TestGenerateModuleDefaultsAndAlias(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	if err := execute("m", "users", Options{}); err != nil {
		t.Fatalf("generate module failed: %v", err)
	}

	assertFile(t, filepath.Join(root, "internal", "modules", "users", "users_module.go"))
	assertFile(t, filepath.Join(root, "internal", "modules", "users", "users_service.go"))
	assertFile(t, filepath.Join(root, "internal", "modules", "users", "users_controller.go"))
	moduleContent := mustRead(t, filepath.Join(root, "internal", "modules", "users", "users_module.go"))
	if !strings.Contains(moduleContent, "usersService := NewUsersService(m.log)") {
		t.Fatalf("expected service registration in module, got:\\n%s", moduleContent)
	}
	if !strings.Contains(moduleContent, "usersController := NewUsersController(usersService, m.log)") {
		t.Fatalf("expected controller to receive service dependency, got:\\n%s", moduleContent)
	}

	mainContent := mustRead(t, filepath.Join(root, "cmd", "main.go"))
	if !strings.Contains(mainContent, "app.Use(users.NewModule(appLogger))") {
		t.Fatalf("expected users module registration in cmd/main.go, got:\n%s", mainContent)
	}
	if !strings.Contains(mainContent, "appLogger := logger.NewLogger(") {
		t.Fatalf("expected logger bootstrap in cmd/main.go, got:\n%s", mainContent)
	}
}

func TestGenerateTransactionalModuleWithRepository(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	opts := Options{TransactionalModule: true, WithRepository: true}
	if err := execute("module", "payments", opts); err != nil {
		t.Fatalf("generate module failed: %v", err)
	}

	assertFile(t, filepath.Join(root, "internal", "modules", "payments", "payments_service.go"))
	assertFile(t, filepath.Join(root, "internal", "modules", "payments", "payments_repository.go"))
	if _, err := os.Stat(filepath.Join(root, "internal", "modules", "payments", "payments_controller.go")); err == nil {
		t.Fatal("did not expect controller for transactional module")
	}
}

func TestGenerateModuleWithGormRepositoryWiresDatabaseInMain(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	seedProject(t, root)
	enableGormAddonInGoMod(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	opts := Options{WithRepository: true}
	if err := execute("module", "payments", opts); err != nil {
		t.Fatalf("generate module failed: %v", err)
	}

	moduleContent := mustRead(t, filepath.Join(root, "internal", "modules", "payments", "payments_module.go"))
	if !strings.Contains(moduleContent, "paymentsRepository := NewPaymentsRepository(m.log, m.db)") {
		t.Fatalf("expected module to wire repository with logger and db, got:\n%s", moduleContent)
	}
	if !strings.Contains(moduleContent, "paymentsService := NewPaymentsServiceWithRepository(paymentsRepository, m.log)") {
		t.Fatalf("expected module to wire service with repository, got:\n%s", moduleContent)
	}
	if !strings.Contains(moduleContent, "paymentsController := NewPaymentsController(paymentsService, m.log)") {
		t.Fatalf("expected module to wire controller with service, got:\n%s", moduleContent)
	}

	mainContent := mustRead(t, filepath.Join(root, "cmd", "main.go"))
	if !strings.Contains(mainContent, "\"github.com/slice-soft/ss-keel-gorm/database\"") {
		t.Fatalf("expected database import in main.go, got:\n%s", mainContent)
	}
	if !strings.Contains(mainContent, "db, err := database.New(database.Config{") {
		t.Fatalf("expected database bootstrap in main.go, got:\n%s", mainContent)
	}
	if !strings.Contains(mainContent, "app.Use(payments.NewModule(appLogger, db))") {
		t.Fatalf("expected module registration with db in main.go, got:\n%s", mainContent)
	}
}

func TestGenerateRepositoryInModuleRewiresMainWithDatabase(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	seedProject(t, root)
	enableGormAddonInGoMod(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	if err := execute("module", "billing", Options{}); err != nil {
		t.Fatalf("generate module failed: %v", err)
	}
	if err := execute("repository", "billing/billing", Options{}); err != nil {
		t.Fatalf("generate repository failed: %v", err)
	}

	mainContent := mustRead(t, filepath.Join(root, "cmd", "main.go"))
	if !strings.Contains(mainContent, "app.Use(billing.NewModule(appLogger, db))") {
		t.Fatalf("expected module registration to be rewired with db, got:\n%s", mainContent)
	}
}

func TestGenerateModuleScopedServiceKeepsModulePackage(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	if err := execute("module", "users", Options{}); err != nil {
		t.Fatalf("generate module failed: %v", err)
	}
	if err := execute("s", "users/validate", Options{}); err != nil {
		t.Fatalf("generate module service failed: %v", err)
	}

	content := mustRead(t, filepath.Join(root, "internal", "modules", "users", "validate_service.go"))
	if !strings.Contains(content, "package users") {
		t.Fatalf("expected service package to remain 'users', got:\n%s", content)
	}
}

func TestGenerateStandaloneService(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	if err := execute("s", "validate-email", Options{}); err != nil {
		t.Fatalf("generate standalone service failed: %v", err)
	}

	servicePath := filepath.Join(root, "internal", "services", "validate_email_service.go")
	assertFile(t, servicePath)
	serviceContent := mustRead(t, servicePath)
	if !strings.Contains(serviceContent, "package services") {
		t.Fatalf("expected standalone service package to be services, got:\\n%s", serviceContent)
	}

	mainContent := mustRead(t, filepath.Join(root, "cmd", "main.go"))
	if !strings.Contains(mainContent, "services.NewValidateEmailService") {
		t.Fatalf("expected standalone service registration in cmd/main.go, got:\n%s", mainContent)
	}
}

func TestGenerateStandaloneControllerFileAndRegisterMain(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	if err := execute("c", "ops-status", Options{}); err != nil {
		t.Fatalf("generate standalone controller failed: %v", err)
	}

	controllerPath := filepath.Join(root, "internal", "controllers", "ops_status_controller.go")
	assertFile(t, controllerPath)
	controllerContent := mustRead(t, controllerPath)
	if !strings.Contains(controllerContent, "package controllers") {
		t.Fatalf("expected standalone controller package to be controllers, got:\n%s", controllerContent)
	}

	mainContent := mustRead(t, filepath.Join(root, "cmd", "main.go"))
	if !strings.Contains(mainContent, "controllers.NewOpsStatusController") {
		t.Fatalf("expected standalone controller registration in cmd/main.go, got:\n%s", mainContent)
	}
}

func TestGenerateStandaloneControllerInlineInMain(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	if err := execute("controller", "ops-ping", Options{ControllerInMain: true}); err != nil {
		t.Fatalf("generate inline standalone controller failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "internal", "controllers", "ops_ping_controller.go")); err == nil {
		t.Fatal("did not expect controller file for --in-main mode")
	}

	mainContent := mustRead(t, filepath.Join(root, "cmd", "main.go"))
	if !strings.Contains(mainContent, `core.GET("/ops-ping"`) {
		t.Fatalf("expected inline route in cmd/main.go, got:\n%s", mainContent)
	}
}

func TestInMainFlagValidation(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	err := execute("service", "validate-email", Options{ControllerInMain: true})
	if err == nil {
		t.Fatal("expected validation error for --in-main with non-controller")
	}
}

func TestGenerateStandaloneSchedulerCheckerHookRegistersMain(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	if err := execute("sch", "nightly-jobs", Options{}); err != nil {
		t.Fatalf("generate scheduler failed: %v", err)
	}
	if err := execute("chk", "cache", Options{}); err != nil {
		t.Fatalf("generate checker failed: %v", err)
	}
	if err := execute("hk", "shutdown", Options{}); err != nil {
		t.Fatalf("generate hook failed: %v", err)
	}

	assertFile(t, filepath.Join(root, "internal", "scheduler", "in_memory_scheduler.go"))
	assertFile(t, filepath.Join(root, "internal", "scheduler", "nightly_jobs_scheduler.go"))
	assertFile(t, filepath.Join(root, "internal", "checkers", "cache_checker.go"))
	assertFile(t, filepath.Join(root, "internal", "hooks", "shutdown_hook.go"))

	mainContent := mustRead(t, filepath.Join(root, "cmd", "main.go"))
	if !strings.Contains(mainContent, "generatedScheduler := scheduler.NewInMemoryScheduler()") {
		t.Fatalf("expected scheduler setup in main.go, got:\n%s", mainContent)
	}
	if !strings.Contains(mainContent, "app.RegisterScheduler(generatedScheduler)") {
		t.Fatalf("expected scheduler registration in main.go, got:\n%s", mainContent)
	}
	if !strings.Contains(mainContent, "scheduler.RegisterNightlyJobsJobs(generatedScheduler, appLogger)") {
		t.Fatalf("expected scheduler job registration in main.go, got:\n%s", mainContent)
	}
	if !strings.Contains(mainContent, "app.RegisterHealthChecker(checkers.NewCacheChecker(appLogger))") {
		t.Fatalf("expected checker registration in main.go, got:\n%s", mainContent)
	}
	if !strings.Contains(mainContent, "app.OnShutdown(hooks.NewShutdownHook(appLogger).OnShutdown)") {
		t.Fatalf("expected hook registration in main.go, got:\n%s", mainContent)
	}
}

func seedProject(t *testing.T, root string) {
	t.Helper()
	mustWrite(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	mustMkdir(t, filepath.Join(root, "cmd"))
	mustWrite(t, filepath.Join(root, "cmd", "main.go"), `package main

import (
	"log"

	"github.com/slice-soft/ss-keel-core/config"
	"github.com/slice-soft/ss-keel-core/core"
)

func main() {
	app := core.New(core.KConfig{
		Port:        config.GetEnvIntOrDefault("PORT", 7331),
		ServiceName: config.GetEnvOrDefault("SERVICE_NAME", "app"),
		Env:         config.GetEnvOrDefault("APP_ENV", "development"),
	})

	log.Fatal(app.Listen())
}
`)
	mustMkdir(t, filepath.Join(root, "internal"))
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("mkdir failed (%s): %v", path, err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	mustMkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write failed (%s): %v", path, err)
	}
}

func assertFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s: %v", path, err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed (%s): %v", path, err)
	}
	return string(b)
}

func enableGormAddonInGoMod(t *testing.T, root string) {
	t.Helper()
	mustWrite(t, filepath.Join(root, "go.mod"), `module example.com/app

require github.com/slice-soft/ss-keel-gorm v0.0.0
`)
}
