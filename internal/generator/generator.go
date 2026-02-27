package generator

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates
var templates embed.FS

// Data contiene las variables disponibles en todos los templates.
type Data struct {
	AppName     string // mi-app
	ModuleName  string // github.com/user/mi-app
	PackageName string // users
	PascalName  string // Users
	CamelName   string // users
	KebabName   string // users
	SnakeName   string // users
}

// NewData construye el Data a partir del nombre en cualquier formato.
func NewData(name string) Data {
	kebab := toKebab(name)
	pascal := toPascal(name)
	return Data{
		PackageName: toPackage(name),
		PascalName:  pascal,
		CamelName:   toCamel(pascal),
		KebabName:   kebab,
		SnakeName:   toSnake(name),
	}
}

// NewProjectData construye el Data para un proyecto nuevo.
func NewProjectData(appName, moduleName string) Data {
	d := NewData(appName)
	d.AppName = appName
	d.ModuleName = moduleName
	return d
}

// RenderToFile renderiza un template y lo escribe en destPath.
func RenderToFile(tmplPath, destPath string, data Data) error {
	content, err := templates.ReadFile(tmplPath)
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

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("error creando archivo %s: %w", destPath, err)
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

// FileExists retorna true si el archivo ya existe.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// — Conversiones de nombre —

func toKebab(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

func toSnake(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

func toPackage(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func toPascal(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	var result strings.Builder
	for _, p := range parts {
		if len(p) > 0 {
			result.WriteString(strings.ToUpper(p[:1]) + p[1:])
		}
	}
	return result.String()
}

func toCamel(pascal string) string {
	if pascal == "" {
		return ""
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}
