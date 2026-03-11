package generate

import (
	"os"
	"strings"
)

// IsAddonInstalled reports whether the given Go module path is present
// as a direct dependency in the go.mod of the current directory.
//
//	generator.IsAddonInstalled("github.com/slice-soft/ss-keel-gorm")
func IsAddonInstalled(modulePath string) bool {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), modulePath)
}
