package generate

import (
	"fmt"
	"go/format"
	"os"
	"strings"

	"github.com/slice-soft/keel/internal/generator"
)

func ensureModuleRegisteredInMain(moduleName string) error {
	modulePath := generator.ReadModuleName()
	if modulePath == "" {
		return fmt.Errorf(invalidProjectMessage)
	}

	data := generator.NewData(moduleName)
	importPath := fmt.Sprintf("\"%s/internal/modules/%s\"", modulePath, data.PackageName)
	useLine := fmt.Sprintf("\tapp.Use(%s.NewModule(appLogger))", data.PackageName)

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
	registerLine := fmt.Sprintf("\tapp.RegisterController(controllers.New%sController(appLogger))", data.PascalName)

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
	routeLine := fmt.Sprintf("\t\t\tcore.GET(%q, func(c *core.Ctx) error {\n\t\t\t\treturn c.OK(map[string]string{\"component\": %q})\n\t\t\t}).\n\t\t\t\tTag(%q).\n\t\t\t\tDescribe(%q),", "/"+data.KebabName, data.KebabName, data.KebabName, "Handle "+data.KebabName+" endpoint")
	registerLine := "\tapp.RegisterController(core.ControllerFunc(func() []core.Route {\n\t\treturn []core.Route{\n" + routeLine + "\n\t\t}\n\t}))"

	return updateMainGo(func(content string) string {
		if !strings.Contains(content, "\"github.com/slice-soft/ss-keel-core/core\"") {
			content = addImport(content, "\"github.com/slice-soft/ss-keel-core/core\"")
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
	markers := []string{
		"\tlog.Fatal(app.Listen())",
		"\tif err := app.Listen(); err != nil {",
		"log.Fatal(app.Listen())",
		"if err := app.Listen(); err != nil {",
	}

	for _, marker := range markers {
		idx := strings.Index(content, marker)
		if idx != -1 {
			return content[:idx] + line + "\n\n" + content[idx:]
		}
	}

	return content
}

func ensureAppLoggerBootstrap(content string) string {
	loggerImport := "\"github.com/slice-soft/ss-keel-core/logger\""
	if !strings.Contains(content, loggerImport) {
		content = addImport(content, loggerImport)
	}

	loggerInit := "\tappLogger := logger.NewLogger(config.GetEnvOrDefault(\"APP_ENV\", \"development\") == \"production\")"
	if !strings.Contains(content, "appLogger := logger.NewLogger(") {
		content = addMainLine(content, loggerInit)
	}

	return content
}
