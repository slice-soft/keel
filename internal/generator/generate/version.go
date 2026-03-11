package generate

import (
	"fmt"
	"os/exec"
	"strings"
)

func getLatestModuleVersion(module string) (string, error) {
	cmd := exec.Command("go", "list", "-m", module+"@latest")

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error getting latest version for %s: %w", module, err)
	}

	version := strings.TrimSpace(string(output))
	fmt.Printf("  ✓ Latest version: %s\n", version)

	return version, nil
}
