package new

import (
	"fmt"
	"os"
	"os/exec"
)

func runPostSetup(setup projectSetup) {
	gitInitialized := false

	if setup.initGit {
		fmt.Println()
		gitCmd := exec.Command("git", "init", setup.appName)
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr
		if err := gitCmd.Run(); err != nil {
			fmt.Printf("  ⚠  git init failed: %v\n", err)
		} else {
			gitInitialized = true
			fmt.Println("  ✓ Git repository initialized")
		}
	}

	if setup.installDeps {
		fmt.Println()
		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = setup.appName
		tidyCmd.Stdout = os.Stdout
		tidyCmd.Stderr = os.Stderr
		if err := tidyCmd.Run(); err != nil {
			fmt.Printf("  ⚠  go mod tidy failed: %v\n", err)
		} else {
			fmt.Println("  ✓ Dependencies installed")
		}
	}

	if gitInitialized {
		if err := createInitialCommit(setup.appName); err != nil {
			fmt.Printf("  ⚠  initial commit failed: %v\n", err)
		} else {
			fmt.Println("  ✓ Initial commit created")
		}
	}
}

func airInstalled() bool {
	_, err := exec.LookPath("air")
	return err == nil
}

func installAirBinary() error {
	installCmd := exec.Command("go", "install", "github.com/air-verse/air@latest")
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	return installCmd.Run()
}

func createInitialCommit(projectDir string) error {
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = projectDir
	addCmd.Stdout = os.Stdout
	addCmd.Stderr = os.Stderr
	if err := addCmd.Run(); err != nil {
		return err
	}

	commitCmd := exec.Command("git", "commit", "-m", "feat: initial commit keel framework")
	commitCmd.Dir = projectDir
	commitCmd.Stdout = os.Stdout
	commitCmd.Stderr = os.Stderr
	return commitCmd.Run()
}
