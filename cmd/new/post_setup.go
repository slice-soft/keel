package new

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/slice-soft/keel/internal/gomod"
)

var commandRunner = exec.Command
var createInitialCommitFn = createInitialCommit
var lookPathFn = exec.LookPath

func runPostSetup(setup projectSetup) {
	gitInitialized := false

	if setup.initGit {
		fmt.Println()
		gitCmd := commandRunner("git", "init", setup.appName)
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
		if err := gomod.RunTidy(commandRunner, setup.appName, os.Stdout, os.Stderr); err != nil {
			fmt.Printf("  ⚠  go mod tidy failed: %v\n", err)
		} else {
			fmt.Println("  ✓ Dependencies installed")
		}
	}

	if gitInitialized {
		if setup.skipInitialCommit {
			fmt.Println("  ⚠  Initial commit skipped: update go.mod to replace the placeholder module path first.")
			return
		}
		if err := createInitialCommitFn(setup.appName); err != nil {
			fmt.Printf("  ⚠  initial commit failed: %v\n", err)
		} else {
			fmt.Println("  ✓ Initial commit created")
		}
	}
}

func airInstalled() bool {
	_, err := lookPathFn("air")
	return err == nil
}

func installAirBinary() error {
	installCmd := commandRunner("go", "install", "github.com/air-verse/air@latest")
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	return installCmd.Run()
}

func createInitialCommit(projectDir string) error {
	addCmd := commandRunner("git", "add", ".")
	addCmd.Dir = projectDir
	addCmd.Stdout = os.Stdout
	addCmd.Stderr = os.Stderr
	if err := addCmd.Run(); err != nil {
		return err
	}

	commitCmd := commandRunner("git", "commit", "-m", "feat: initial commit keel framework")
	commitCmd.Dir = projectDir
	commitCmd.Stdout = os.Stdout
	commitCmd.Stderr = os.Stderr
	return commitCmd.Run()
}
