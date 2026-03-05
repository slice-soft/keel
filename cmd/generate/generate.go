package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/slice-soft/keel/internal/generator"
)

const (
	typeModule     = "module"
	typeService    = "service"
	typeController = "controller"
	typeRepository = "repository"
	typeMiddleware = "middleware"
	typeGuard      = "guard"
	typeScheduler  = "scheduler"
	typeEvent      = "event"
	typeChecker    = "checker"
	typeHook       = "hook"
)

type Options struct {
	TransactionalModule bool
	WithRepository      bool
	ControllerInMain    bool
}

var supportedTypes = map[string]struct{}{
	typeModule: {}, typeService: {}, typeController: {}, typeRepository: {},
	typeMiddleware: {}, typeGuard: {}, typeScheduler: {}, typeEvent: {}, typeChecker: {}, typeHook: {},
}

var typeAliases = map[string]string{
	"m":          typeModule,
	"mod":        typeModule,
	"s":          typeService,
	"svc":        typeService,
	"c":          typeController,
	"ctrl":       typeController,
	"r":          typeRepository,
	"repo":       typeRepository,
	"mw":         typeMiddleware,
	"middleware": typeMiddleware,
	"guard":      typeGuard,
	"gd":         typeGuard,
	"sch":        typeScheduler,
	"ev":         typeEvent,
	"chk":        typeChecker,
	"hk":         typeHook,
}

type genFile struct {
	template string
	dest     string
	data     generator.Data
}

func logCreated(path string) {
	fmt.Printf("  + created %s\n", path)
}

func logUpdated(path string) {
	fmt.Printf("  ~ updated %s\n", path)
}

func execute(genType, rawName string, opts Options) error {
	if err := validateKeelProject(); err != nil {
		return err
	}

	resolvedType, err := resolveType(genType)
	if err != nil {
		return err
	}

	parsed, err := parseName(rawName)
	if err != nil {
		return err
	}

	if resolvedType == typeModule {
		if !parsed.standalone {
			return fmt.Errorf("module name must not contain '/'")
		}
		if err := generateModule(parsed.componentName, opts); err != nil {
			return err
		}
		return ensureModuleRegisteredInMain(parsed.componentName)
	}

	if opts.TransactionalModule || opts.WithRepository {
		return fmt.Errorf("--transactional and --with-repository are only valid for module generation")
	}
	if opts.ControllerInMain && !(resolvedType == typeController && parsed.standalone) {
		return fmt.Errorf("--in-main is only valid for standalone controller generation")
	}

	if parsed.standalone {
		return generateStandalone(resolvedType, parsed.componentName, opts)
	}

	if err := ensureModuleExists(parsed.moduleName); err != nil {
		return err
	}
	if err := generateInModule(resolvedType, parsed.moduleName, parsed.componentName); err != nil {
		return err
	}
	return regenerateModuleRegistry(parsed.moduleName)
}

func resolveType(raw string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(raw))
	if v, ok := typeAliases[key]; ok {
		key = v
	}
	if _, ok := supportedTypes[key]; !ok {
		return "", fmt.Errorf("unsupported generator type: %s", raw)
	}
	return key, nil
}

func generateModule(name string, opts Options) error {
	if err := ensureModuleExists(name); err != nil {
		return err
	}

	files := buildSimpleFiles(typeService, name, moduleDir(name), "service.go.tmpl", "service_test.go.tmpl", name)
	if !opts.TransactionalModule {
		files = append(files, buildSimpleFiles(typeController, name, moduleDir(name), "controller.go.tmpl", "controller_test.go.tmpl", name)...)
	}
	if opts.WithRepository {
		files = append(files, buildSimpleFiles(typeRepository, name, moduleDir(name), "repository.go.tmpl", "repository_test.go.tmpl", name)...)
	}

	for _, file := range files {
		if generator.FileExists(file.dest) {
			continue
		}
		if err := generator.RenderToFile(file.template, file.dest, file.data); err != nil {
			return err
		}
		logCreated(file.dest)
	}

	return regenerateModuleRegistry(name)
}

func generateStandalone(genType, componentName string, opts Options) error {
	switch genType {
	case typeService:
		if err := createFiles(buildStandaloneServiceFiles(componentName)); err != nil {
			return err
		}
		return ensureStandaloneServiceRegisteredInMain(componentName)
	case typeController:
		if opts.ControllerInMain {
			return ensureInlineStandaloneControllerRegisteredInMain(componentName)
		}
		if err := createFiles(buildStandaloneControllerFiles(componentName)); err != nil {
			return err
		}
		return ensureStandaloneControllerRegisteredInMain(componentName)
	case typeMiddleware:
		return createFiles(buildSimpleFiles(genType, componentName, filepath.Join("internal", "middleware"), "middleware.go.tmpl", "middleware_test.go.tmpl", ""))
	case typeGuard:
		return createFiles(buildSimpleFiles(genType, componentName, filepath.Join("internal", "guards"), "guard.go.tmpl", "guard_test.go.tmpl", ""))
	case typeScheduler:
		if err := createFiles(buildSchedulerFiles(componentName, filepath.Join("internal", "scheduler"))); err != nil {
			return err
		}
		return ensureSchedulerRegisteredInMain(componentName)
	case typeEvent:
		return createFiles(buildEventFiles(componentName, filepath.Join("internal", "events")))
	case typeChecker:
		if err := createFiles(buildSimpleFiles(genType, componentName, filepath.Join("internal", "checkers"), "checker.go.tmpl", "checker_test.go.tmpl", "")); err != nil {
			return err
		}
		return ensureCheckerRegisteredInMain(componentName)
	case typeHook:
		if err := createFiles(buildSimpleFiles(genType, componentName, filepath.Join("internal", "hooks"), "hook.go.tmpl", "hook_test.go.tmpl", "")); err != nil {
			return err
		}
		return ensureHookRegisteredInMain(componentName)
	default:
		return fmt.Errorf("%s requires module/name format", genType)
	}
}

func generateInModule(genType, moduleName, componentName string) error {
	baseDir := moduleDir(moduleName)
	switch genType {
	case typeService:
		return createFiles(buildSimpleFiles(genType, componentName, baseDir, "service.go.tmpl", "service_test.go.tmpl", moduleName))
	case typeController:
		return createFiles(buildSimpleFiles(genType, componentName, baseDir, "controller.go.tmpl", "controller_test.go.tmpl", moduleName))
	case typeRepository:
		return createFiles(buildSimpleFiles(genType, componentName, baseDir, "repository.go.tmpl", "repository_test.go.tmpl", moduleName))
	default:
		return fmt.Errorf("%s does not support module/name format", genType)
	}
}

func ensureModuleExists(name string) error {
	if err := createFiles(buildModuleFiles(name)); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return err
		}
	}
	if err := validateExistingModulePackage(name); err != nil {
		return err
	}
	if err := regenerateModuleRegistry(name); err != nil {
		return err
	}
	return ensureModuleRegisteredInMain(name)
}

func validateExistingModulePackage(moduleName string) error {
	dir := moduleDir(moduleName)
	expected := generator.NewData(moduleName).PackageName

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		pkg, readErr := readPackageName(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			return readErr
		}
		if pkg != "" && pkg != expected {
			return fmt.Errorf("module package mismatch: expected '%s', found '%s' in %s", expected, pkg, entry.Name())
		}
	}

	return nil
}

func readPackageName(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "package ")), nil
		}
	}
	return "", nil
}

func createFiles(files []genFile) error {
	for _, file := range files {
		if generator.FileExists(file.dest) {
			return fmt.Errorf("file already exists: %s", file.dest)
		}
		if err := generator.RenderToFile(file.template, file.dest, file.data); err != nil {
			return err
		}
		logCreated(file.dest)
	}
	return nil
}

func buildModuleFiles(moduleName string) []genFile {
	moduleData := generator.NewData(moduleName)
	moduleData.Services = []generator.ComponentRegistration{}
	moduleData.Controllers = []generator.ComponentRegistration{}
	moduleData.Repositories = []generator.ComponentRegistration{}

	return []genFile{
		{
			template: "templates/generate/module/module.go.tmpl",
			dest:     moduleRegistryPath(moduleName),
			data:     moduleData,
		},
		{
			template: "templates/generate/module/module_test.go.tmpl",
			dest:     filepath.Join(moduleDir(moduleName), moduleData.SnakeName+"_module_test.go"),
			data:     moduleData,
		},
	}
}

func buildSimpleFiles(genType, componentName, baseDir, implTemplate, testTemplate, packageOverride string) []genFile {
	data := generator.NewData(componentName)
	if packageOverride != "" {
		data.PackageName = generator.NewData(packageOverride).PackageName
	}

	return []genFile{
		{
			template: filepath.Join("templates", "generate", genType, implTemplate),
			dest:     filepath.Join(baseDir, data.SnakeName+"_"+genType+".go"),
			data:     data,
		},
		{
			template: filepath.Join("templates", "generate", genType, testTemplate),
			dest:     filepath.Join(baseDir, data.SnakeName+"_"+genType+"_test.go"),
			data:     data,
		},
	}
}

func buildStandaloneServiceFiles(componentName string) []genFile {
	return buildSimpleFiles(typeService, componentName, filepath.Join("internal", "services"), "service.go.tmpl", "service_test.go.tmpl", "services")
}

func buildStandaloneControllerFiles(componentName string) []genFile {
	return buildSimpleFiles(typeController, componentName, filepath.Join("internal", "controllers"), "controller.go.tmpl", "controller_test.go.tmpl", "controllers")
}

func buildSchedulerFiles(componentName, baseDir string) []genFile {
	data := generator.NewData(componentName)
	files := []genFile{
		{template: "templates/generate/scheduler/job.go.tmpl", dest: filepath.Join(baseDir, data.SnakeName+"_job.go"), data: data},
		{template: "templates/generate/scheduler/scheduler.go.tmpl", dest: filepath.Join(baseDir, data.SnakeName+"_scheduler.go"), data: data},
		{template: "templates/generate/scheduler/scheduler_test.go.tmpl", dest: filepath.Join(baseDir, data.SnakeName+"_scheduler_test.go"), data: data},
	}

	runtimePath := filepath.Join(baseDir, "in_memory_scheduler.go")
	if !generator.FileExists(runtimePath) {
		files = append(files, genFile{
			template: "templates/generate/scheduler/in_memory_scheduler.go.tmpl",
			dest:     runtimePath,
			data:     data,
		})
	}

	runtimeTestPath := filepath.Join(baseDir, "in_memory_scheduler_test.go")
	if !generator.FileExists(runtimeTestPath) {
		files = append(files, genFile{
			template: "templates/generate/scheduler/in_memory_scheduler_test.go.tmpl",
			dest:     runtimeTestPath,
			data:     data,
		})
	}

	return files
}

func buildEventFiles(componentName, baseDir string) []genFile {
	data := generator.NewData(componentName)
	return []genFile{
		{template: "templates/generate/event/publisher.go.tmpl", dest: filepath.Join(baseDir, data.SnakeName+"_publisher.go"), data: data},
		{template: "templates/generate/event/subscriber.go.tmpl", dest: filepath.Join(baseDir, data.SnakeName+"_subscriber.go"), data: data},
		{template: "templates/generate/event/event_test.go.tmpl", dest: filepath.Join(baseDir, data.SnakeName+"_event_test.go"), data: data},
	}
}

func moduleDir(moduleName string) string {
	return filepath.Join("internal", "modules", generator.NewData(moduleName).PackageName)
}

func moduleRegistryPath(moduleName string) string {
	data := generator.NewData(moduleName)
	modulePath := filepath.Join(moduleDir(moduleName), data.SnakeName+"_module.go")
	legacyPath := filepath.Join(moduleDir(moduleName), "module.go")
	if generator.FileExists(legacyPath) {
		return legacyPath
	}
	return modulePath
}

func regenerateModuleRegistry(moduleName string) error {
	services, err := listComponents(moduleDir(moduleName), "_service.go")
	if err != nil {
		return err
	}
	controllers, err := listComponents(moduleDir(moduleName), "_controller.go")
	if err != nil {
		return err
	}
	repositories, err := listComponents(moduleDir(moduleName), "_repository.go")
	if err != nil {
		return err
	}

	moduleData := generator.NewData(moduleName)
	moduleData.Services = toRegistrations(services)
	moduleData.Controllers = toRegistrations(controllers)
	moduleData.Repositories = toRegistrations(repositories)

	dest := moduleRegistryPath(moduleName)
	alreadyExisted := generator.FileExists(dest)
	if err := generator.RenderToFile("templates/generate/module/module.go.tmpl", dest, moduleData); err != nil {
		return err
	}
	if alreadyExisted {
		logUpdated(dest)
	} else {
		logCreated(dest)
	}
	return nil
}

func listComponents(dir, suffix string) ([]string, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*"+suffix))
	if err != nil {
		return nil, err
	}

	items := make([]string, 0, len(entries))
	for _, file := range entries {
		base := filepath.Base(file)
		if strings.HasSuffix(base, "_test.go") {
			continue
		}
		name := strings.TrimSuffix(base, suffix)
		if name == "" {
			continue
		}
		items = append(items, name)
	}
	sort.Strings(items)
	return items, nil
}

func toRegistrations(names []string) []generator.ComponentRegistration {
	items := make([]generator.ComponentRegistration, 0, len(names))
	for _, name := range names {
		d := generator.NewData(name)
		items = append(items, generator.ComponentRegistration{
			Name:       name,
			PascalName: d.PascalName,
			VarName:    d.CamelName,
		})
	}
	return items
}
