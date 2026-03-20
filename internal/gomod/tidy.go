package gomod

import (
	"fmt"
	"io"
	"os/exec"
)

// CommandRunner creates exec.Cmd values for Go module maintenance commands.
type CommandRunner func(name string, args ...string) *exec.Cmd

// RunTidy executes "go mod tidy" in dir and streams command output to the provided writers.
func RunTidy(commandRunner CommandRunner, dir string, stdout, stderr io.Writer) error {
	if commandRunner == nil {
		commandRunner = exec.Command
	}
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	fmt.Fprintln(stdout, "  → go mod tidy")

	cmd := commandRunner("go", "mod", "tidy")
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}
	return nil
}
