package gomod

import (
	"os"
	"regexp"
)

var goDirectivePatchPattern = regexp.MustCompile(`(?m)^go (\d+\.\d+)\.\d+$`)

// NormalizeDirective rewrites patch-level go directives like "go 1.25.7"
// to their stable major.minor form ("go 1.25").
func NormalizeDirective(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	normalized := goDirectivePatchPattern.ReplaceAll(content, []byte("go $1"))
	if string(normalized) == string(content) {
		return nil
	}

	return os.WriteFile(path, normalized, 0644)
}
