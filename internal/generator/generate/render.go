package generate

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"text/template"

	templatefs "github.com/slice-soft/keel/internal/generator"
)

// RenderToFile renders a template and writes it to destPath.
func RenderToFile(tmplPath, destPath string, data Data) error {
	content, err := templatefs.ReadTemplate(tmplPath)
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

	var rendered []byte
	buffer := &bytes.Buffer{}
	if err := tmpl.Execute(buffer, data); err != nil {
		return err
	}
	rendered = buffer.Bytes()

	if filepath.Ext(destPath) == ".go" {
		formatted, err := format.Source(rendered)
		if err == nil {
			rendered = formatted
		}
	}

	if err := os.WriteFile(destPath, rendered, 0644); err != nil {
		return fmt.Errorf("error writing file %s: %w", destPath, err)
	}
	return nil
}

// FileExists returns true when the file already exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
