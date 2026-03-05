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
		if err := generator.RenderToFile(f.tmpl, f.dest, data); err != nil {
			return fmt.Errorf("error generating %s: %w", f.dest, err)
		}
		fmt.Printf("  ✓ %s\n", f.dest)
	}
	return nil
}

func buildProjectFiles(appName string, includeAirConfig, useEnv, includeStarterModule bool) []projectFile {
	files := []projectFile{
		{tmpl: "templates/project/main.go.tmpl", dest: filepath.Join(appName, "cmd", "main.go")},
		{tmpl: "templates/project/go.mod.tmpl", dest: filepath.Join(appName, "go.mod")},
		{tmpl: "templates/project/keel.toml.tmpl", dest: filepath.Join(appName, "keel.toml")},
		{tmpl: "templates/project/readme.tmpl", dest: filepath.Join(appName, "README.md")},
		{tmpl: "templates/project/gitignore.tmpl", dest: filepath.Join(appName, ".gitignore")},
	}

	if useEnv {
		files = append(files, projectFile{
			tmpl: "templates/project/env.tmpl",
			dest: filepath.Join(appName, ".env"),
		})
	}

	if includeAirConfig {
		files = append(files, projectFile{
			tmpl: "templates/project/air.toml.tmpl",
			dest: filepath.Join(appName, ".air.toml"),
		})
	}

	if includeStarterModule {
		modulesPath := "internal/modules"
		filesModule := []projectFile{
			{tmpl: "templates/module/starter_module.go.tmpl", dest: filepath.Join(appName, modulesPath, "starter", "module.go")},
			{tmpl: "templates/module/starter_service.go.tmpl", dest: filepath.Join(appName, modulesPath, "starter", "service.go")},
			{tmpl: "templates/module/starter_controller.go.tmpl", dest: filepath.Join(appName, modulesPath, "starter", "controller.go")},
			{tmpl: "templates/module/starter_dto.go.tmpl", dest: filepath.Join(appName, modulesPath, "starter", "dto.go")},
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
