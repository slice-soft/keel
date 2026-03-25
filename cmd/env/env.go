// Package env provides the `keel env` command group for managing
// environment variables declared in keel.toml.
package env

import (
	"fmt"
	"os"
	"strings"

	"github.com/slice-soft/keel/internal/keeltoml"
	"github.com/spf13/cobra"
)

// NewCommand builds the `keel env` parent command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage environment variables for the current Keel project",
		Long: `Commands to sync, generate, and check environment variables
declared in keel.toml.

  keel env sync      — generate / update .env.example from keel.toml
  keel env generate  — generate .env from keel.toml (only missing keys)
  keel env check     — validate .env against keel.toml required vars`,
	}
	cmd.AddCommand(newSyncCommand())
	cmd.AddCommand(newGenerateCommand())
	cmd.AddCommand(newCheckCommand())
	return cmd
}

// injectable for tests
var (
	loadKeelTomlFn = func(path string) (*keeltoml.KeelToml, error) { return keeltoml.Load(path) }
	readFileFn     = os.ReadFile
	lookupOSEnvFn  = os.LookupEnv
)

// ---- keel env sync --------------------------------------------------------

func newSyncCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Generate or update .env.example from keel.toml",
		Long: `Reads all [[env]] entries from keel.toml and updates .env.example.

  - secret = true  → KEY=your-secret-here
  - secret = false → KEY= (empty) or KEY=<default>
  - Existing manual entries not in keel.toml are preserved
  - Real values are never written`,
		Args: cobra.NoArgs,
		RunE: runSync,
	}
}

func runSync(_ *cobra.Command, _ []string) error {
	kt, err := loadKeelTomlFn(keeltoml.DefaultPath)
	if err != nil {
		return fmt.Errorf("reading keel.toml: %w", err)
	}
	if len(kt.Env) == 0 {
		fmt.Println("  ℹ  no [[env]] entries in keel.toml — nothing to sync")
		return nil
	}

	const examplePath = ".env.example"
	existing, _ := readFileFn(examplePath)

	var added int
	var appendBuf strings.Builder

	for _, ev := range kt.Env {
		// Skip if key already in .env.example (manual entry preserved).
		if _, ok := keeltoml.LookupEnvValue(string(existing), ev.Key); ok {
			continue
		}

		value := exampleValue(ev)
		comment := buildComment(ev)

		if comment != "" {
			fmt.Fprintf(&appendBuf, "\n%s\n", comment)
		}
		fmt.Fprintf(&appendBuf, "%s=%s\n", ev.Key, value)
		added++
	}

	if added == 0 {
		fmt.Printf("  ✓  .env.example is up to date\n")
		return nil
	}

	toAppend := appendBuf.String()
	prefix := ""
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		prefix = "\n"
	}

	f, err := os.OpenFile(examplePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", examplePath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(prefix + toAppend); err != nil {
		return fmt.Errorf("writing %s: %w", examplePath, err)
	}

	fmt.Printf("  ✓  added %d key(s) to .env.example\n", added)
	return nil
}

// exampleValue returns a safe placeholder value for .env.example.
func exampleValue(ev keeltoml.EnvEntry) string {
	if ev.Secret {
		return "your-secret-here"
	}
	if ev.Default != "" {
		return ev.Default
	}
	return ""
}

// buildComment returns a # comment line describing the env var, or "".
func buildComment(ev keeltoml.EnvEntry) string {
	var parts []string
	if ev.Description != "" {
		parts = append(parts, ev.Description)
	}
	if ev.Source != "" {
		parts = append(parts, "source: "+ev.Source)
	}
	if ev.Required {
		parts = append(parts, "required")
	} else {
		parts = append(parts, "optional")
	}
	if ev.Secret {
		parts = append(parts, "secret")
	}
	if len(parts) == 0 {
		return ""
	}
	return "# " + strings.Join(parts, " | ")
}

// ---- keel env generate ----------------------------------------------------

func newGenerateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generate .env from keel.toml (only adds missing keys)",
		Long: `Reads all [[env]] entries from keel.toml and updates .env.

  - Required vars without a default → KEY= (empty, ready to fill)
  - Optional vars with a default    → # KEY=default (commented)
  - If .env already exists, only missing keys are added (no overwrite)`,
		Args: cobra.NoArgs,
		RunE: runGenerate,
	}
}

func runGenerate(_ *cobra.Command, _ []string) error {
	kt, err := loadKeelTomlFn(keeltoml.DefaultPath)
	if err != nil {
		return fmt.Errorf("reading keel.toml: %w", err)
	}
	if len(kt.Env) == 0 {
		fmt.Println("  ℹ  no [[env]] entries in keel.toml — nothing to generate")
		return nil
	}

	const dotEnvPath = ".env"
	existing, _ := readFileFn(dotEnvPath)

	var added int
	var appendBuf strings.Builder

	for _, ev := range kt.Env {
		if _, ok := keeltoml.LookupEnvValue(string(existing), ev.Key); ok {
			continue // already present — skip
		}

		if ev.Required {
			fmt.Fprintf(&appendBuf, "%s=\n", ev.Key)
		} else {
			defaultVal := ev.Default
			fmt.Fprintf(&appendBuf, "# %s=%s\n", ev.Key, defaultVal)
		}
		added++
	}

	if added == 0 {
		fmt.Printf("  ✓  .env already has all declared keys\n")
		return nil
	}

	toAppend := appendBuf.String()
	prefix := ""
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		prefix = "\n"
	}

	f, err := os.OpenFile(dotEnvPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", dotEnvPath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(prefix + toAppend); err != nil {
		return fmt.Errorf("writing %s: %w", dotEnvPath, err)
	}

	fmt.Printf("  ✓  added %d key(s) to .env\n", added)
	return nil
}

// ---- keel env check -------------------------------------------------------

func newCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Validate .env against required vars in keel.toml",
		Long: `Compares environment variables declared in keel.toml against .env and the OS environment.

  ✓  var is set
  ✗  required var is missing — exits with non-zero
  ⚠  optional var is missing`,
		Args: cobra.NoArgs,
		RunE: runCheck,
	}
}

func runCheck(_ *cobra.Command, _ []string) error {
	kt, err := loadKeelTomlFn(keeltoml.DefaultPath)
	if err != nil {
		return fmt.Errorf("reading keel.toml: %w", err)
	}
	if len(kt.Env) == 0 {
		fmt.Println("  ℹ  no [[env]] entries in keel.toml — nothing to check")
		return nil
	}

	dotEnv, _ := readFileFn(".env")
	dotEnvStr := string(dotEnv)

	fmt.Println()
	hasErrors := false

	for _, ev := range kt.Env {
		val, inFile := keeltoml.LookupEnvValue(dotEnvStr, ev.Key)
		_, inOS := lookupOSEnvFn(ev.Key)

		switch {
		case inFile && val != "":
			fmt.Printf("  ✓  %s is set\n", ev.Key)
		case inOS:
			fmt.Printf("  ✓  %s is set (via OS env)\n", ev.Key)
		case inFile && val == "" && !ev.Required:
			// Present but empty and optional — treat as ok (user intentionally left empty)
			fmt.Printf("  ✓  %s is set (empty)\n", ev.Key)
		case ev.Required:
			fmt.Printf("  ✗  %s is required but not set\n", ev.Key)
			hasErrors = true
		default:
			suffix := ""
			if ev.Default != "" {
				suffix = fmt.Sprintf(" (default: %q)", ev.Default)
			}
			fmt.Printf("  ⚠  %s is optional and not set%s\n", ev.Key, suffix)
		}
	}

	fmt.Println()
	if hasErrors {
		return fmt.Errorf("missing required environment variables")
	}
	return nil
}
