package generator

import "embed"

//go:embed templates
var templatesFS embed.FS

// ReadTemplate returns an embedded template by path.
func ReadTemplate(path string) ([]byte, error) {
	return templatesFS.ReadFile(path)
}
