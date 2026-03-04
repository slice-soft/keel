package generator

import (
	"fmt"
	"os/exec"
	"strings"
)

func getLatestModuleVersion(module string) (string, error) {
	cmd := exec.Command("go", "list", "-m", module+"@latest")

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error obteniendo última versión de %s: %w", module, err)
	}

	version := strings.TrimSpace(string(output))
	fmt.Printf("  ✓ Última versión: %s\n", version)

	return version, nil
}
