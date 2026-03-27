package generate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetGenerateDeps(t *testing.T) {
	t.Helper()

	previousEnsurePersistenceAddonInstalled := ensurePersistenceAddonInstalledFn
	previousRunGoModTidy := runGoModTidyFn
	t.Cleanup(func() {
		ensurePersistenceAddonInstalledFn = previousEnsurePersistenceAddonInstalled
		runGoModTidyFn = previousRunGoModTidy
	})
}

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

func TestDatabaseBootstrapIsPlacedBeforeModules(t *testing.T) {
	mainWithModules := `package main

import (
	"log"

	"github.com/slice-soft/ss-keel-core/config"
	"github.com/slice-soft/ss-keel-core/core"
)

func main() {
	app := core.New(core.KConfig{
		Port: config.GetInt("PORT"),
	})

	// Register your modules here:
	app.Use(starter.NewModule(appLogger))

	log.Fatal(app.Listen())
}
`

	tests := []struct {
		name      string
		bootstrap func(string) string
		needle    string
	}{
		{
			name:      "gorm",
			bootstrap: ensureGormDatabaseBootstrap,
			needle:    "dbConfig := config.MustLoadConfig[database.Config]()",
		},
		{
			name:      "mongo",
			bootstrap: ensureMongoDatabaseBootstrap,
			needle:    "mongoConfig := config.MustLoadConfig[mongo.Config]()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated := tt.bootstrap(mainWithModules)
			if !strings.Contains(updated, tt.needle) {
				t.Fatalf("expected bootstrap code %q, got:\n%s", tt.needle, updated)
			}
			if strings.Index(updated, tt.needle) > strings.Index(updated, "// Register your modules here:") {
				t.Fatalf("expected bootstrap code before modules, got:\n%s", updated)
			}
		})
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
	assertFile(t, filepath.Join(root, "internal", "modules", "users", "users_dto.go"))
	assertFile(t, filepath.Join(root, "internal", "modules", "users", "users_entity.go"))
	assertFile(t, filepath.Join(root, "internal", "modules", "users", "users_service.go"))
	assertFile(t, filepath.Join(root, "internal", "modules", "users", "users_controller.go"))
	if _, err := os.Stat(filepath.Join(root, "internal", "modules", "users", "users_repository.go")); err == nil {
		t.Fatal("did not expect repository by default")
	}
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

func TestGenerateBaseModuleUsesPluralRouteForSingularName(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	if err := execute("module", "user", Options{}); err != nil {
		t.Fatalf("generate module failed: %v", err)
	}

	controllerContent := mustRead(t, filepath.Join(root, "internal", "modules", "user", "user_controller.go"))
	if !strings.Contains(controllerContent, `httpx.GET("/users", c.Hello)`) {
		t.Fatalf("expected singular module route to be pluralized, got:\n%s", controllerContent)
	}
}

func TestGenerateModuleRunsGoModTidy(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	resetGenerateDeps(t)
	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	tidyCalled := 0
	runGoModTidyFn = func() error {
		tidyCalled++
		return nil
	}

	if err := execute("module", "users", Options{}); err != nil {
		t.Fatalf("generate module failed: %v", err)
	}
	if tidyCalled != 1 {
		t.Fatalf("expected go mod tidy to run once, got %d", tidyCalled)
	}
}

func TestGenerateModuleReturnsGoModTidyError(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	resetGenerateDeps(t)
	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	runGoModTidyFn = func() error {
		return errors.New("go mod tidy failed: boom")
	}

	err := execute("module", "users", Options{})
	if err == nil || !strings.Contains(err.Error(), "go mod tidy failed: boom") {
		t.Fatalf("expected go mod tidy error to be returned, got %v", err)
	}
}

func TestGenerateModuleTestKeptInSyncWithModuleSignature(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	resetGenerateDeps(t)
	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	ensurePersistenceAddonInstalledFn = func(_ repositoryBackend) error { return nil }

	if err := execute("module", "orders", Options{UseGormPersistence: true}); err != nil {
		t.Fatalf("generate module failed: %v", err)
	}

	moduleTestContent := mustRead(t, filepath.Join(root, "internal", "modules", "orders", "orders_module_test.go"))
	if !strings.Contains(moduleTestContent, "NewModule(logger.NewLogger(false), nil)") {
		t.Fatalf("expected module_test.go to pass nil db arg after gorm module generation, got:\n%s", moduleTestContent)
	}
}

func TestGenerateMongoModuleTestKeptInSyncWithModuleSignature(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	resetGenerateDeps(t)
	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	ensurePersistenceAddonInstalledFn = func(_ repositoryBackend) error { return nil }

	if err := execute("module", "orders", Options{UseMongoPersistence: true}); err != nil {
		t.Fatalf("generate module failed: %v", err)
	}

	moduleTestContent := mustRead(t, filepath.Join(root, "internal", "modules", "orders", "orders_module_test.go"))
	if !strings.Contains(moduleTestContent, "NewModule(logger.NewLogger(false), nil)") {
		t.Fatalf("expected module_test.go to pass nil mongoClient arg after mongo module generation, got:\n%s", moduleTestContent)
	}
}

func TestGenerateTransactionalModuleWithGormPersistence(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	resetGenerateDeps(t)
	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	ensurePersistenceAddonInstalledFn = func(backend repositoryBackend) error {
		if backend != repositoryBackendGorm {
			t.Fatalf("unexpected backend: %s", backend)
		}
		return nil
	}

	opts := Options{TransactionalModule: true, UseGormPersistence: true}
	if err := execute("module", "payments", opts); err != nil {
		t.Fatalf("generate module failed: %v", err)
	}

	assertFile(t, filepath.Join(root, "internal", "modules", "payments", "payments_service.go"))
	assertFile(t, filepath.Join(root, "internal", "modules", "payments", "payments_repository.go"))
	if _, err := os.Stat(filepath.Join(root, "internal", "modules", "payments", "payments_controller.go")); err == nil {
		t.Fatal("did not expect controller for transactional module")
	}
}

func TestGenerateModuleWithGormFlagWiresDatabaseInMain(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	resetGenerateDeps(t)
	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	called := false
	ensurePersistenceAddonInstalledFn = func(backend repositoryBackend) error {
		called = true
		if backend != repositoryBackendGorm {
			t.Fatalf("unexpected backend: %s", backend)
		}
		return nil
	}

	opts := Options{UseGormPersistence: true}
	if err := execute("module", "payments", opts); err != nil {
		t.Fatalf("generate module failed: %v", err)
	}
	if !called {
		t.Fatal("expected gorm addon install path to run")
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

	entityContent := mustRead(t, filepath.Join(root, "internal", "modules", "payments", "payments_entity.go"))
	if !strings.Contains(entityContent, "database.EntityBase") {
		t.Fatalf("expected entity to embed database.EntityBase, got:\n%s", entityContent)
	}
	if !strings.Contains(entityContent, "github.com/slice-soft/ss-keel-gorm/database") {
		t.Fatalf("expected gorm database import in entity, got:\n%s", entityContent)
	}

	serviceContent := mustRead(t, filepath.Join(root, "internal", "modules", "payments", "payments_service.go"))
	if strings.Contains(serviceContent, "uuid.NewString()") {
		t.Fatalf("did not expect service to generate UUIDs, got:\n%s", serviceContent)
	}
	if !strings.Contains(serviceContent, "entity := &PaymentsEntity{\n\t\tName: strings.TrimSpace(req.Name),\n\t}") {
		t.Fatalf("expected service create flow to initialize only business fields, got:\n%s", serviceContent)
	}

	repositoryContent := mustRead(t, filepath.Join(root, "internal", "modules", "payments", "payments_repository.go"))
	if strings.Contains(repositoryContent, "\n\tPatch(ctx context.Context, id string,") {
		t.Fatalf("did not expect repository contract to redeclare Patch, got:\n%s", repositoryContent)
	}

	mainContent := mustRead(t, filepath.Join(root, "cmd", "main.go"))
	if !strings.Contains(mainContent, "\"github.com/slice-soft/ss-keel-gorm/database\"") {
		t.Fatalf("expected database import in main.go, got:\n%s", mainContent)
	}
	if !strings.Contains(mainContent, "dbConfig := config.MustLoadConfig[database.Config]()") {
		t.Fatalf("expected database bootstrap in main.go, got:\n%s", mainContent)
	}
	if !strings.Contains(mainContent, "app.Use(payments.NewModule(appLogger, db))") {
		t.Fatalf("expected module registration with db in main.go, got:\n%s", mainContent)
	}
}

func TestGenerateModuleWithMongoFlagWiresDatabaseInMain(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	resetGenerateDeps(t)
	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	called := false
	ensurePersistenceAddonInstalledFn = func(backend repositoryBackend) error {
		called = true
		if backend != repositoryBackendMongo {
			t.Fatalf("unexpected backend: %s", backend)
		}
		return nil
	}

	opts := Options{UseMongoPersistence: true}
	if err := execute("module", "payments", opts); err != nil {
		t.Fatalf("generate module failed: %v", err)
	}
	if !called {
		t.Fatal("expected mongo addon install path to run")
	}

	moduleContent := mustRead(t, filepath.Join(root, "internal", "modules", "payments", "payments_module.go"))
	if !strings.Contains(moduleContent, "paymentsRepository := NewPaymentsRepository(m.log, m.mongoClient)") {
		t.Fatalf("expected module to wire repository with logger and mongo client, got:\n%s", moduleContent)
	}
	if !strings.Contains(moduleContent, "\"github.com/slice-soft/ss-keel-mongo/mongo\"") {
		t.Fatalf("expected mongo import in module, got:\n%s", moduleContent)
	}

	entityContent := mustRead(t, filepath.Join(root, "internal", "modules", "payments", "payments_entity.go"))
	if strings.Contains(entityContent, "primitive.ObjectID") {
		t.Fatalf("did not expect ObjectID in the domain entity, got:\n%s", entityContent)
	}
	if !strings.Contains(entityContent, "mongo.EntityBase") {
		t.Fatalf("expected entity to embed mongo.EntityBase, got:\n%s", entityContent)
	}
	if !strings.Contains(entityContent, "github.com/slice-soft/ss-keel-mongo/mongo") {
		t.Fatalf("expected mongo import in entity, got:\n%s", entityContent)
	}
	if strings.Contains(entityContent, `bson:"name"`) {
		t.Fatalf("did not expect mongo persistence tags in the domain entity, got:\n%s", entityContent)
	}

	repositoryContent := mustRead(t, filepath.Join(root, "internal", "modules", "payments", "payments_repository.go"))
	if !strings.Contains(repositoryContent, "type PaymentsMongoDocument struct") {
		t.Fatalf("expected mongo repository to keep its persistence document in the repository layer, got:\n%s", repositoryContent)
	}
	if strings.Contains(repositoryContent, "\n\tPatch(ctx context.Context, id string,") {
		t.Fatalf("did not expect repository contract to redeclare Patch, got:\n%s", repositoryContent)
	}

	mainContent := mustRead(t, filepath.Join(root, "cmd", "main.go"))
	if !strings.Contains(mainContent, "\"github.com/slice-soft/ss-keel-mongo/mongo\"") {
		t.Fatalf("expected mongo import in main.go, got:\n%s", mainContent)
	}
	if !strings.Contains(mainContent, "mongoConfig := config.MustLoadConfig[mongo.Config]()") {
		t.Fatalf("expected mongo bootstrap in main.go, got:\n%s", mainContent)
	}
	if !strings.Contains(mainContent, "app.Use(payments.NewModule(appLogger, mongoClient))") {
		t.Fatalf("expected module registration with mongo client in main.go, got:\n%s", mainContent)
	}
}

func TestGeneratePersistenceModulesUsePluralRestRoutes(t *testing.T) {
	tests := []struct {
		name  string
		opts  Options
		setup func(*testing.T)
	}{
		{
			name: "gorm",
			opts: Options{UseGormPersistence: true},
			setup: func(t *testing.T) {
				ensurePersistenceAddonInstalledFn = func(backend repositoryBackend) error {
					if backend != repositoryBackendGorm {
						t.Fatalf("unexpected backend: %s", backend)
					}
					return nil
				}
			},
		},
		{
			name: "mongo",
			opts: Options{UseMongoPersistence: true},
			setup: func(t *testing.T) {
				ensurePersistenceAddonInstalledFn = func(backend repositoryBackend) error {
					if backend != repositoryBackendMongo {
						t.Fatalf("unexpected backend: %s", backend)
					}
					return nil
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			oldWD, _ := os.Getwd()
			defer func() { _ = os.Chdir(oldWD) }()

			resetGenerateDeps(t)
			seedProject(t, root)

			if err := os.Chdir(root); err != nil {
				t.Fatalf("chdir failed: %v", err)
			}

			tt.setup(t)

			if err := execute("module", "user", tt.opts); err != nil {
				t.Fatalf("generate module failed: %v", err)
			}

			controllerContent := mustRead(t, filepath.Join(root, "internal", "modules", "user", "user_controller.go"))
			for _, want := range []string{
				`httpx.GET("/users", c.List)`,
				`httpx.GET("/users/:id", c.GetByID)`,
				`httpx.POST("/users", c.Create)`,
				`httpx.PUT("/users/:id", c.Update)`,
				`httpx.PATCH("/users/:id", c.Patch)`,
				`httpx.DELETE("/users/:id", c.Delete)`,
				`WithResponse(httpx.WithResponse[struct{}](204))`,
			} {
				if !strings.Contains(controllerContent, want) {
					t.Fatalf("expected controller to contain %q, got:\n%s", want, controllerContent)
				}
			}
		})
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

	moduleContent := mustRead(t, filepath.Join(root, "internal", "modules", "billing", "billing_module.go"))
	if !strings.Contains(moduleContent, "billingService := NewBillingServiceWithRepository(billingRepository, m.log)") {
		t.Fatalf("expected module to rewire the base service with repository support, got:\n%s", moduleContent)
	}

	repositoryContent := mustRead(t, filepath.Join(root, "internal", "modules", "billing", "billing_repository.go"))
	if strings.Contains(repositoryContent, "type BillingEntity struct") {
		t.Fatalf("did not expect repository generation to redefine the existing entity, got:\n%s", repositoryContent)
	}

	mainContent := mustRead(t, filepath.Join(root, "cmd", "main.go"))
	if !strings.Contains(mainContent, "app.Use(billing.NewModule(appLogger, db))") {
		t.Fatalf("expected module registration to be rewired with db, got:\n%s", mainContent)
	}
}

func TestGenerateRepositoryCreatesSeparateEntityFile(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	if err := execute("module", "billing", Options{}); err != nil {
		t.Fatalf("generate module failed: %v", err)
	}
	if err := execute("repository", "billing/invoice", Options{}); err != nil {
		t.Fatalf("generate repository failed: %v", err)
	}

	entityContent := mustRead(t, filepath.Join(root, "internal", "modules", "billing", "invoice_entity.go"))
	if !strings.Contains(entityContent, "type InvoiceEntity struct") {
		t.Fatalf("expected separate invoice entity file, got:\n%s", entityContent)
	}

	repositoryContent := mustRead(t, filepath.Join(root, "internal", "modules", "billing", "invoice_repository.go"))
	if strings.Contains(repositoryContent, "type InvoiceEntity struct") {
		t.Fatalf("did not expect repository file to define the entity, got:\n%s", repositoryContent)
	}
	if !strings.Contains(repositoryContent, "contracts.Repository[InvoiceEntity, string, httpx.PageQuery, httpx.Page[InvoiceEntity]]") {
		t.Fatalf("expected repository to use the core contracts with the separate entity, got:\n%s", repositoryContent)
	}
}

func TestGenerateGormRepositoryCreatesEntityWithDatabaseBase(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	resetGenerateDeps(t)
	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	ensurePersistenceAddonInstalledFn = func(backend repositoryBackend) error {
		if backend != repositoryBackendGorm {
			t.Fatalf("unexpected backend: %s", backend)
		}
		return nil
	}

	if err := execute("module", "billing", Options{}); err != nil {
		t.Fatalf("generate module failed: %v", err)
	}
	if err := execute("repository", "billing/customer", Options{UseGormPersistence: true}); err != nil {
		t.Fatalf("generate repository failed: %v", err)
	}

	entityContent := mustRead(t, filepath.Join(root, "internal", "modules", "billing", "customer_entity.go"))
	if !strings.Contains(entityContent, "database.EntityBase") {
		t.Fatalf("expected gorm repository entity to embed database.EntityBase, got:\n%s", entityContent)
	}
	if !strings.Contains(entityContent, "github.com/slice-soft/ss-keel-gorm/database") {
		t.Fatalf("expected gorm repository entity to import the database package, got:\n%s", entityContent)
	}
}

func TestGenerateMongoRepositoryCreatesSeparateEntityFile(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	resetGenerateDeps(t)
	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	ensurePersistenceAddonInstalledFn = func(backend repositoryBackend) error {
		if backend != repositoryBackendMongo {
			t.Fatalf("unexpected backend: %s", backend)
		}
		return nil
	}

	if err := execute("module", "billing", Options{}); err != nil {
		t.Fatalf("generate module failed: %v", err)
	}
	if err := execute("repository", "billing/customer", Options{UseMongoPersistence: true}); err != nil {
		t.Fatalf("generate repository failed: %v", err)
	}

	entityContent := mustRead(t, filepath.Join(root, "internal", "modules", "billing", "customer_entity.go"))
	if strings.Contains(entityContent, "primitive.ObjectID") || strings.Contains(entityContent, `bson:"name"`) {
		t.Fatalf("did not expect persistence document details in the generated domain entity, got:\n%s", entityContent)
	}
	if !strings.Contains(entityContent, "mongo.EntityBase") {
		t.Fatalf("expected mongo repository entity to embed mongo.EntityBase, got:\n%s", entityContent)
	}
	if !strings.Contains(entityContent, "github.com/slice-soft/ss-keel-mongo/mongo") {
		t.Fatalf("expected mongo repository entity to import the mongo package, got:\n%s", entityContent)
	}

	repositoryContent := mustRead(t, filepath.Join(root, "internal", "modules", "billing", "customer_repository.go"))
	if strings.Contains(repositoryContent, "type CustomerEntity struct") {
		t.Fatalf("did not expect repository file to define the entity, got:\n%s", repositoryContent)
	}
	if !strings.Contains(repositoryContent, "type CustomerMongoDocument struct") {
		t.Fatalf("expected repository to define a persistence document instead, got:\n%s", repositoryContent)
	}
}

func TestGenerateRepositoryWithBothAddonsPromptsBackendSelection(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	seedProject(t, root)
	enableBothAddonsInGoMod(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	previousPrompt := promptRepositoryBackendFn
	t.Cleanup(func() {
		promptRepositoryBackendFn = previousPrompt
	})

	promptCalled := false
	promptRepositoryBackendFn = func() (repositoryBackend, error) {
		promptCalled = true
		return repositoryBackendMongo, nil
	}

	if err := execute("module", "billing", Options{}); err != nil {
		t.Fatalf("generate module failed: %v", err)
	}
	if err := execute("repository", "billing/billing", Options{}); err != nil {
		t.Fatalf("generate repository failed: %v", err)
	}
	if !promptCalled {
		t.Fatal("expected repository backend prompt to be used when both addons are installed")
	}

	repositoryContent := mustRead(t, filepath.Join(root, "internal", "modules", "billing", "billing_repository.go"))
	if !strings.Contains(repositoryContent, "\"github.com/slice-soft/ss-keel-mongo/mongo\"") {
		t.Fatalf("expected mongo repository template, got:\n%s", repositoryContent)
	}

	mainContent := mustRead(t, filepath.Join(root, "cmd", "main.go"))
	if !strings.Contains(mainContent, "app.Use(billing.NewModule(appLogger, mongoClient))") {
		t.Fatalf("expected module registration to use mongo client, got:\n%s", mainContent)
	}
}

func TestPersistenceFlagsValidationForNonRepositoryGeneration(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	err := execute("service", "users", Options{UseMongoPersistence: true})
	if err == nil || !strings.Contains(err.Error(), "--mongo and --gorm are only valid for module or repository generation") {
		t.Fatalf("expected persistence flag validation error, got: %v", err)
	}
}

func TestPersistenceFlagsMustNotConflict(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()

	seedProject(t, root)

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	err := execute("module", "users", Options{UseMongoPersistence: true, UseGormPersistence: true})
	if err == nil || !strings.Contains(err.Error(), "--mongo and --gorm cannot be used together") {
		t.Fatalf("expected conflicting persistence flags error, got: %v", err)
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
	if !strings.Contains(mainContent, "app.RegisterController(contracts.ControllerFunc[httpx.Route]") {
		t.Fatalf("expected inline controller registration in cmd/main.go, got:\n%s", mainContent)
	}
	if !strings.Contains(mainContent, `httpx.GET("/ops-ping"`) {
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
	previousRunGoModTidy := runGoModTidyFn
	runGoModTidyFn = func() error { return nil }
	t.Cleanup(func() {
		runGoModTidyFn = previousRunGoModTidy
	})

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
		Port:        config.GetInt("PORT"),
		ServiceName: config.GetString("SERVICE_NAME"),
		Env:         config.GetString("APP_ENV"),
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

func enableBothAddonsInGoMod(t *testing.T, root string) {
	t.Helper()
	mustWrite(t, filepath.Join(root, "go.mod"), `module example.com/app

require (
	github.com/slice-soft/ss-keel-gorm v0.0.0
	github.com/slice-soft/ss-keel-mongo v0.0.0
)
	`)
}
