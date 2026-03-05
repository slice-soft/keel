package initcmd

import (
	"fmt"
	"os"
	"os/exec"
)

const airInstallCommand = "go install github.com/air-verse/air@latest"

var lookPath = exec.LookPath
var runAirInstall = func() error {
	installCmd := exec.Command("go", "install", "github.com/air-verse/air@latest")
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	return installCmd.Run()
}

func ensureAirReady(useAir bool) error {
	if !useAir {
		return nil
	}

	if airInstalled() {
		fmt.Println("  ✓ Air is already installed")
		return nil
	}

	fmt.Println("  ⚠  Air is not installed on your PATH.")
	fmt.Printf("  Installing Air with: %s\n", airInstallCommand)

	if err := installAirBinary(); err != nil {
		return fmt.Errorf("failed to install Air: %w", err)
	}

	if airInstalled() {
		fmt.Println("  ✓ Air installed")
		return nil
	}

	fmt.Println("  ✓ Air installed (restart your shell if 'air' is not available yet)")
	return nil
}

func airInstalled() bool {
	_, err := lookPath("air")
	return err == nil
}

func installAirBinary() error {
	return runAirInstall()
}
