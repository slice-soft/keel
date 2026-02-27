package cmd
package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommand(t *testing.T) {
	// Test that root command is properly initialized
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}

	if rootCmd.Use != "keel" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "keel")
	}
}

func TestVersionCommand(t *testing.T) {
	// Find version subcommand
	versionCmd := findCommand(rootCmd, "version")
	if versionCmd == nil {
		t.Fatal("version command not found")
	}

	// Test version command output
	buf := new(bytes.Buffer)
	versionCmd.SetOut(buf)
	versionCmd.SetErr(buf)

	err := versionCmd.RunE(versionCmd, []string{})
	if err != nil {
		t.Errorf("version command failed: %v", err)
	}
}

func TestSubcommandsExist(t *testing.T) {



















}	return nil	}		}			return cmd		if cmd.Name() == name {	for _, cmd := range parent.Commands() {func findCommand(parent *cobra.Command, name string) *cobra.Command {// Helper function to find a command by name}	}		}			t.Errorf("expected command %q not found", cmdName)		if cmd == nil {		cmd := findCommand(rootCmd, cmdName)	for _, cmdName := range expectedCommands {	expectedCommands := []string{"generate", "new", "run", "upgrade", "version"}