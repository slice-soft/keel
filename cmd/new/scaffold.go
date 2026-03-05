package new

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/slice-soft/keel/internal/generator"
)

func scaffoldProject(setup projectSetup) error {
	useStarterModule := !setup.withoutStarterModule
	useFolderStructure := setup.withFolderStructure

	data := generator.NewProjectData(
		setup.appName,
		setup.moduleName,
		setup.useAir,
		setup.includeAirConfig,
		setup.useEnv,
		useStarterModule,
		useFolderStructure,
	)

	fmt.Printf("\n  Creating project files...\n\n")
	if err := renderProjectFiles(setup.appName, setup.includeAirConfig, setup.useEnv, useStarterModule, data); err != nil {
		return err
	}

	return createProjectDirectories(setup.appName, setup.withFolderStructure, setup.withoutStarterModule)
}

func renderProjectFiles(appName string, includeAirConfig, useEnv, includeStarterModule bool, data generator.Data) error {
	files := buildProjectFiles(appName, includeAirConfig, useEnv, includeStarterModule)
	for _, f := range files {
		if err := generator.RenderToFile(f.TemplatePath, f.Destination, data); err != nil {
			return fmt.Errorf("error generating %s: %w", f.Destination, err)
		}
		fmt.Printf("  ✓ %s\n", f.Destination)
	}
	return nil
}

func buildProjectFiles(appName string, includeAirConfig, useEnv, includeStarterModule bool) []ProjectFile {
	files := []ProjectFile{
		{TemplatePath: "templates/project/main.go.tmpl", Destination: filepath.Join(appName, "cmd", "main.go")},
		{TemplatePath: "templates/project/go.mod.tmpl", Destination: filepath.Join(appName, "go.mod")},
		{TemplatePath: "templates/project/keel.toml.tmpl", Destination: filepath.Join(appName, "keel.toml")},
		{TemplatePath: "templates/project/readme.tmpl", Destination: filepath.Join(appName, "README.md")},
		{TemplatePath: "templates/project/gitignore.tmpl", Destination: filepath.Join(appName, ".gitignore")},
	}

	if useEnv {
		files = append(files, ProjectFile{
			TemplatePath: "templates/project/env.tmpl",
			Destination:  filepath.Join(appName, ".env"),
		})
	}

	if includeAirConfig {
		files = append(files, ProjectFile{
			TemplatePath: "templates/project/air.toml.tmpl",
			Destination:  filepath.Join(appName, ".air.toml"),
		})
	}

	if includeStarterModule {
		modulesPath := "internal/modules"
		filesModule := []ProjectFile{
			{TemplatePath: "templates/module/starter_module.go.tmpl", Destination: filepath.Join(appName, modulesPath, "starter", "module.go")},
			{TemplatePath: "templates/module/starter_service.go.tmpl", Destination: filepath.Join(appName, modulesPath, "starter", "service.go")},
			{TemplatePath: "templates/module/starter_controller.go.tmpl", Destination: filepath.Join(appName, modulesPath, "starter", "controller.go")},
			{TemplatePath: "templates/module/starter_dto.go.tmpl", Destination: filepath.Join(appName, modulesPath, "starter", "dto.go")},
		}
		files = append(files, filesModule...)
	}

	return files
}

func createProjectDirectories(appName string, withFolderStructure, withoutStarterModule bool) error {
	if !withFolderStructure {
		return nil
	}

	dirs := []string{
		filepath.Join(appName, "internal", "middleware"),
		filepath.Join(appName, "internal", "guards"),
		filepath.Join(appName, "internal", "scheduler"),
		filepath.Join(appName, "internal", "checkers"),
		filepath.Join(appName, "internal", "events"),
		filepath.Join(appName, "internal", "hooks"),
	}

	if withoutStarterModule {
		dirs = append(dirs, filepath.Join(appName, "internal", "modules"))
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
		fmt.Printf("  ✓ %s/\n", d)
	}

	return nil
}
