package generator

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates
var templatesFS embed.FS

// RenderToFile renders a template and writes it to destPath.
func RenderToFile(tmplPath, destPath string, data Data) error {
	content, err := templatesFS.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("template not found: %s", tmplPath)
	}

	tmpl, err := template.New("").Parse(string(content))
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", destPath, err)
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

// FileExists returns true when the file already exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
