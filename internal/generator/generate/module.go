package generate

import (
	"os"
	"strings"
)

// ReadModuleName reads the go.mod in the current directory and returns the module name.
func ReadModuleName() string {
	data, _ := os.ReadFile("go.mod")
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}
