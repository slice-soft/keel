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

// RenderToFile renderiza un template y lo escribe en destPath.
func RenderToFile(tmplPath, destPath string, data Data) error {
	content, err := templatesFS.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("template no encontrado: %s", tmplPath)
	}

	tmpl, err := template.New("").Parse(string(content))
	if err != nil {
		return fmt.Errorf("error parseando template: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("error creando archivo %s: %w", destPath, err)
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

// FileExists retorna true si el archivo ya existe.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
