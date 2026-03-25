package initcmd

import (
	"fmt"
	"os"
	"path/filepath"

	newcmd "github.com/slice-soft/keel/cmd/new"
	generator "github.com/slice-soft/keel/internal/generator/generate"
)

var getwdFn = os.Getwd

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func validateKeelConfigDoesNotExist(path string) error {
	if fileExists(path) {
		return fmt.Errorf("keel.toml already exists in this directory")
	}
	return nil
}

func generateKeelConfig(destPath string, useAir, airConfigExists bool) error {
	appName, err := currentDirName()
	if err != nil {
		return err
	}

	data := generator.NewInitData(appName, useAir, airConfigExists)
	files := buildInitFiles(destPath, useAir, airConfigExists, fileExists("application.properties"))

	for _, f := range files {
		if err := generator.RenderToFile(f.TemplatePath, f.Destination, data); err != nil {
			return fmt.Errorf("failed generating %s: %w", f.Destination, err)
		}
		fmt.Printf("  ✓ %s created\n", f.Destination)
	}
	return nil
}

func buildInitFiles(keelConfigPath string, useAir, airConfigExists, applicationPropertiesExists bool) []newcmd.ProjectFile {
	files := []newcmd.ProjectFile{
		{
			TemplatePath: "templates/project/keel.toml.tmpl",
			Destination:  keelConfigPath,
		},
	}

	if !applicationPropertiesExists {
		files = append(files, newcmd.ProjectFile{
			TemplatePath: "templates/project/application.properties.tmpl",
			Destination:  "application.properties",
		})
	}

	if useAir && !airConfigExists {
		files = append(files, newcmd.ProjectFile{
			TemplatePath: "templates/project/air.toml.tmpl",
			Destination:  ".air.toml",
		})
	}

	return files
}

func currentDirName() (string, error) {
	wd, err := getwdFn()
	if err != nil {
		return "", fmt.Errorf("failed to resolve current directory: %w", err)
	}

	base := filepath.Base(wd)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "app", nil
	}

	return base, nil
}
