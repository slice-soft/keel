package generate

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	generator "github.com/slice-soft/keel/internal/generator/generate"
)

type moduleDatabaseNeeds struct {
	gorm  bool
	mongo bool
}

func ensureModuleRegisteredInMain(moduleName string) error {
	modulePath := generator.ReadModuleName()
	if modulePath == "" {
		return fmt.Errorf(invalidProjectMessage)
	}
	needsDatabase, err := moduleNeedsDatabase(moduleName)
	if err != nil {
		return err
	}

	data := generator.NewData(moduleName)
	importPath := fmt.Sprintf("\"%s/internal/modules/%s\"", modulePath, data.PackageName)
	useLine := fmt.Sprintf("\tapp.Use(%s.NewModule(appLogger))", data.PackageName)
	useLineWithGormDB := fmt.Sprintf("\tapp.Use(%s.NewModule(appLogger, db))", data.PackageName)
	useLineWithMongoDB := fmt.Sprintf("\tapp.Use(%s.NewModule(appLogger, mongoClient))", data.PackageName)
	useLineWithBothDBs := fmt.Sprintf("\tapp.Use(%s.NewModule(appLogger, db, mongoClient))", data.PackageName)

	targetUseLine := useLine
	switch {
	case needsDatabase.gorm && needsDatabase.mongo:
		targetUseLine = useLineWithBothDBs
	case needsDatabase.gorm:
		targetUseLine = useLineWithGormDB
	case needsDatabase.mongo:
		targetUseLine = useLineWithMongoDB
	}

	return updateMainGo(func(content string) string {
		content = ensureAppLoggerBootstrap(content)
		if needsDatabase.gorm {
			content = ensureGormDatabaseBootstrap(content)
		}
		if needsDatabase.mongo {
			content = ensureMongoDatabaseBootstrap(content)
		}
		if !strings.Contains(content, importPath) {
			content = addImport(content, importPath)
		}

		useLineCandidates := []string{useLine, useLineWithGormDB, useLineWithMongoDB, useLineWithBothDBs}
		for _, candidate := range useLineCandidates {
			if candidate == targetUseLine {
				continue
			}
			content = strings.ReplaceAll(content, candidate, targetUseLine)
		}

		if !strings.Contains(content, targetUseLine) {
			content = addMainLine(content, targetUseLine)
		}
		return content
	})
}

func ensureStandaloneServiceRegisteredInMain(componentName string) error {
	modulePath := generator.ReadModuleName()
	if modulePath == "" {
		return fmt.Errorf(invalidProjectMessage)
	}

	data := generator.NewData(componentName)
	importPath := fmt.Sprintf("\"%s/internal/services\"", modulePath)
	useLine := fmt.Sprintf("\t_ = services.New%sService(appLogger)", data.PascalName)

	return updateMainGo(func(content string) string {
		content = ensureAppLoggerBootstrap(content)
		if !strings.Contains(content, importPath) {
			content = addImport(content, importPath)
		}
		if !strings.Contains(content, useLine) {
			content = addMainLine(content, useLine)
		}
		return content
	})
}

func ensureStandaloneControllerRegisteredInMain(componentName string) error {
	modulePath := generator.ReadModuleName()
	if modulePath == "" {
		return fmt.Errorf(invalidProjectMessage)
	}

	data := generator.NewData(componentName)
	importPath := fmt.Sprintf("\"%s/internal/controllers\"", modulePath)
	registerLine := fmt.Sprintf("\tapp.RegisterController(controllers.New%sController(nil, appLogger))", data.PascalName)

	return updateMainGo(func(content string) string {
		content = ensureAppLoggerBootstrap(content)
		if !strings.Contains(content, importPath) {
			content = addImport(content, importPath)
		}
		if !strings.Contains(content, registerLine) {
			content = addMainLine(content, registerLine)
		}
		return content
	})
}

func ensureInlineStandaloneControllerRegisteredInMain(componentName string) error {
	data := generator.NewData(componentName)
	routeLine := fmt.Sprintf("\t\t\thttpx.GET(%q, func(c *httpx.Ctx) error {\n\t\t\t\treturn c.OK(map[string]string{\"component\": %q})\n\t\t\t}).\n\t\t\t\tTag(%q).\n\t\t\t\tDescribe(%q),", "/"+data.KebabName, data.KebabName, data.KebabName, "Handle "+data.KebabName+" endpoint")
	registerLine := "\tapp.RegisterController(contracts.ControllerFunc[httpx.Route](func() []httpx.Route {\n\t\treturn []httpx.Route{\n" + routeLine + "\n\t\t}\n\t}))"

	return updateMainGo(func(content string) string {
		if !strings.Contains(content, "\"github.com/slice-soft/ss-keel-core/contracts\"") {
			content = addImport(content, "\"github.com/slice-soft/ss-keel-core/contracts\"")
		}
		if !strings.Contains(content, "\"github.com/slice-soft/ss-keel-core/core/httpx\"") {
			content = addImport(content, "\"github.com/slice-soft/ss-keel-core/core/httpx\"")
		}
		if !strings.Contains(content, "/"+data.KebabName) {
			content = addMainLine(content, registerLine)
		}
		return content
	})
}

func ensureSchedulerRegisteredInMain(componentName string) error {
	modulePath := generator.ReadModuleName()
	if modulePath == "" {
		return fmt.Errorf(invalidProjectMessage)
	}

	data := generator.NewData(componentName)
	importPath := fmt.Sprintf("\"%s/internal/scheduler\"", modulePath)
	setupLine := "\tgeneratedScheduler := scheduler.NewInMemoryScheduler()\n\tapp.RegisterScheduler(generatedScheduler)"
	registerLine := fmt.Sprintf("\tif err := scheduler.Register%sJobs(generatedScheduler, appLogger); err != nil {\n\t\tappLogger.Error(\"failed to register %s jobs: %%v\", err)\n\t}", data.PascalName, data.KebabName)

	return updateMainGo(func(content string) string {
		content = ensureAppLoggerBootstrap(content)
		if !strings.Contains(content, importPath) {
			content = addImport(content, importPath)
		}
		if !strings.Contains(content, "generatedScheduler := scheduler.NewInMemoryScheduler()") {
			content = addMainLine(content, setupLine)
		}
		if !strings.Contains(content, fmt.Sprintf("scheduler.Register%sJobs(generatedScheduler)", data.PascalName)) {
			content = addMainLine(content, registerLine)
		}
		return content
	})
}

func ensureCheckerRegisteredInMain(componentName string) error {
	modulePath := generator.ReadModuleName()
	if modulePath == "" {
		return fmt.Errorf(invalidProjectMessage)
	}

	data := generator.NewData(componentName)
	importPath := fmt.Sprintf("\"%s/internal/checkers\"", modulePath)
	registerLine := fmt.Sprintf("\tapp.RegisterHealthChecker(checkers.New%sChecker(appLogger))", data.PascalName)

	return updateMainGo(func(content string) string {
		content = ensureAppLoggerBootstrap(content)
		if !strings.Contains(content, importPath) {
			content = addImport(content, importPath)
		}
		if !strings.Contains(content, registerLine) {
			content = addMainLine(content, registerLine)
		}
		return content
	})
}

func ensureHookRegisteredInMain(componentName string) error {
	modulePath := generator.ReadModuleName()
	if modulePath == "" {
		return fmt.Errorf(invalidProjectMessage)
	}

	data := generator.NewData(componentName)
	importPath := fmt.Sprintf("\"%s/internal/hooks\"", modulePath)
	registerLine := fmt.Sprintf("\tapp.OnShutdown(hooks.New%sHook(appLogger).OnShutdown)", data.PascalName)

	return updateMainGo(func(content string) string {
		content = ensureAppLoggerBootstrap(content)
		if !strings.Contains(content, importPath) {
			content = addImport(content, importPath)
		}
		if !strings.Contains(content, registerLine) {
			content = addMainLine(content, registerLine)
		}
		return content
	})
}

func updateMainGo(transform func(string) string) error {
	const mainPath = "cmd/main.go"
	body, err := os.ReadFile(mainPath)
	if err != nil {
		return err
	}

	original := string(body)
	updated := transform(original)
	formatted, err := format.Source([]byte(updated))
	if err == nil {
		updated = string(formatted)
	}
	if updated == original {
		return nil
	}

	if err := os.WriteFile(mainPath, []byte(updated), 0644); err != nil {
		return err
	}
	logUpdated(mainPath)
	return nil
}

func addImport(content, importPath string) string {
	start := strings.Index(content, "import (")
	if start == -1 {
		return content
	}
	end := strings.Index(content[start:], ")")
	if end == -1 {
		return content
	}
	end += start

	block := content[start:end]
	if strings.Contains(block, importPath) {
		return content
	}

	insert := "\n\t" + importPath
	return content[:end] + insert + content[end:]
}

func addMainLine(content, line string) string {
	return addMainLineWithAnchor(content, line, "before_listen")
}

func addMainLineWithAnchor(content, line, anchor string) string {
	switch anchor {
	case "", "before_listen":
		if updated, ok := addMainLineBeforeMarkersIfFound(content, line, []string{
			"\tlog.Fatal(app.Listen())",
			"\tif err := app.Listen(); err != nil {",
			"log.Fatal(app.Listen())",
			"if err := app.Listen(); err != nil {",
		}); ok {
			return updated
		}
	case "before_modules":
		if updated, ok := addMainLineBeforeMarkersIfFound(content, line, []string{
			"\t// Register your modules here:",
			"// Register your modules here:",
			"\tapp.Use(",
			"app.Use(",
		}); ok {
			return updated
		}
		return addMainLine(content, line)
	default:
		return addMainLine(content, line)
	}
	return content
}

func addMainLineBeforeMarkersIfFound(content, line string, markers []string) (string, bool) {
	for _, marker := range markers {
		idx := strings.Index(content, marker)
		if idx != -1 {
			return content[:idx] + line + "\n\n" + content[idx:], true
		}
	}
	return content, false
}

func ensureAppLoggerBootstrap(content string) string {
	configImport := "\"github.com/slice-soft/ss-keel-core/config\""
	if !strings.Contains(content, configImport) {
		content = addImport(content, configImport)
	}

	loggerImport := "\"github.com/slice-soft/ss-keel-core/logger\""
	if !strings.Contains(content, loggerImport) {
		content = addImport(content, loggerImport)
	}

	loggerInit := "\tappLogger := logger.NewLogger(config.GetEnvOrDefault(\"APP_ENV\", \"development\") == \"production\")"
	if !strings.Contains(content, "appLogger := logger.NewLogger(") {
		content = addMainLineWithAnchor(content, loggerInit, "before_modules")
	}

	return content
}

func ensureGormDatabaseBootstrap(content string) string {
	// If the keel add gorm addon already set up setupGorm()/setupDatabase(), reuse that variable.
	if strings.Contains(content, "setupGorm(") || strings.Contains(content, "setupDatabase(") {
		return content
	}

	databaseImport := "\"github.com/slice-soft/ss-keel-gorm/database\""
	if !strings.Contains(content, databaseImport) {
		content = addImport(content, databaseImport)
	}

	if strings.Contains(content, "database.New(") {
		return content
	}

	setupLine := "\tdatabaseURL := config.GetEnvOrDefault(\"DATABASE_URL\", \"postgres://user:pass@localhost:5432/db?sslmode=disable\")\n\tdb, err := database.New(database.Config{\n\t\tEngine: database.EnginePostgres,\n\t\tDSN:    databaseURL,\n\t\tLogger: appLogger,\n\t})\n\tif err != nil {\n\t\tappLogger.Error(\"failed to start app: %v\", err)\n\t}\n\tdefer db.Close()\n\tapp.RegisterHealthChecker(database.NewHealthChecker(db))"
	return addMainLineWithAnchor(content, setupLine, "before_modules")
}

func ensureMongoDatabaseBootstrap(content string) string {
	// If the keel add mongo addon already set up setupMongo(), reuse that variable.
	if strings.Contains(content, "setupMongo(") {
		return content
	}

	mongoImport := "\"github.com/slice-soft/ss-keel-mongo/mongo\""
	if !strings.Contains(content, mongoImport) {
		content = addImport(content, mongoImport)
	}

	if strings.Contains(content, "mongo.New(") {
		return content
	}

	setupLine := "\tmongoURI := config.GetEnvOrDefault(\"MONGO_URI\", \"mongodb://localhost:27017\")\n\tmongoDatabase := config.GetEnvOrDefault(\"MONGO_DATABASE\", \"app\")\n\n\tmongoClient, err := mongo.New(mongo.Config{\n\t\tURI:      mongoURI,\n\t\tDatabase: mongoDatabase,\n\t\tLogger:   appLogger,\n\t})\n\tif err != nil {\n\t\tappLogger.Error(\"failed to start app: %v\", err)\n\t}\n\tdefer mongoClient.Close()\n\tapp.RegisterHealthChecker(mongo.NewHealthChecker(mongoClient))"
	return addMainLineWithAnchor(content, setupLine, "before_modules")
}

func moduleNeedsDatabase(moduleName string) (moduleDatabaseNeeds, error) {
	needs := moduleDatabaseNeeds{}

	repositories, err := listComponents(moduleDir(moduleName), "_repository.go")
	if err != nil {
		if os.IsNotExist(err) {
			return needs, nil
		}
		return needs, err
	}

	for _, repositoryName := range repositories {
		repositoryPath := filepath.Join(moduleDir(moduleName), repositoryName+"_repository.go")
		content, readErr := os.ReadFile(repositoryPath)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				continue
			}
			return needs, readErr
		}
		switch repositoryBackendFromContent(string(content)) {
		case repositoryBackendGorm:
			needs.gorm = true
		case repositoryBackendMongo:
			needs.mongo = true
		}

		if needs.gorm && needs.mongo {
			break
		}
	}

	return needs, nil
}
