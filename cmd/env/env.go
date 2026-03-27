// Package env provides the `keel env` command group for managing
// environment variables referenced by application.properties.
package env

import (
	"fmt"
	"os"
	"strings"

	"github.com/slice-soft/keel/internal/appproperties"
	"github.com/slice-soft/keel/internal/keeltoml"
	"github.com/spf13/cobra"
)

type declaredEnvVar struct {
	Key         string
	Default     string
	HasDefault  bool
	Description string
	Required    bool
	Secret      bool
}

// NewCommand builds the `keel env` parent command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage environment variables for the current Keel project",
		Long: `Commands to sync, generate, and check environment variables
referenced by application.properties.

  keel env sync      — generate / update .env.example from application.properties
  keel env generate  — generate .env from application.properties (only missing keys)
  keel env check     — validate required variables used by application.properties`,
	}
	cmd.AddCommand(newSyncCommand())
	cmd.AddCommand(newGenerateCommand())
	cmd.AddCommand(newCheckCommand())
	return cmd
}

// injectable for tests
var (
	loadApplicationPropertiesFn = func(path string) (*appproperties.Document, error) { return appproperties.Load(path) }
	loadKeelTomlFn              = func(path string) (*keeltoml.KeelToml, error) { return keeltoml.Load(path) }
	readFileFn                  = os.ReadFile
	lookupOSEnvFn               = os.LookupEnv
	statFileFn                  = os.Stat
)

// ---- keel env sync --------------------------------------------------------

func newSyncCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Generate or update .env.example from application.properties",
		Long: `Reads environment placeholders from application.properties and updates
.env.example.

  - ${KEY}           → KEY=
  - ${KEY:default}   → KEY=default
  - Existing manual entries not in application.properties are preserved`,
		Args: cobra.NoArgs,
		RunE: runSync,
	}
}

func runSync(_ *cobra.Command, _ []string) error {
	declaredEnvVars, source, err := loadDeclaredEnvVars()
	if err != nil {
		return err
	}
	if len(declaredEnvVars) == 0 {
		fmt.Printf("  ℹ  no environment variables declared in %s — nothing to sync\n", source)
		return nil
	}

	const examplePath = ".env.example"
	existing, _ := readFileFn(examplePath)

	var added int
	var appendBuf strings.Builder

	for _, envVar := range declaredEnvVars {
		if _, ok := keeltoml.LookupEnvValue(string(existing), envVar.Key); ok {
			continue
		}

		if comment := buildComment(envVar); comment != "" {
			fmt.Fprintf(&appendBuf, "\n%s\n", comment)
		}
		fmt.Fprintf(&appendBuf, "%s=%s\n", envVar.Key, exampleValue(envVar))
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

// ---- keel env generate ----------------------------------------------------

func newGenerateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generate .env from application.properties (only adds missing keys)",
		Long: `Reads environment placeholders from application.properties and updates .env.

  - ${KEY}           → KEY=
  - ${KEY:default}   → # KEY=default
  - If .env already exists, only missing keys are added (no overwrite)`,
		Args: cobra.NoArgs,
		RunE: runGenerate,
	}
}

func runGenerate(_ *cobra.Command, _ []string) error {
	declaredEnvVars, source, err := loadDeclaredEnvVars()
	if err != nil {
		return err
	}
	if len(declaredEnvVars) == 0 {
		fmt.Printf("  ℹ  no environment variables declared in %s — nothing to generate\n", source)
		return nil
	}

	const dotEnvPath = ".env"
	existing, _ := readFileFn(dotEnvPath)

	var added int
	var appendBuf strings.Builder

	for _, envVar := range declaredEnvVars {
		if envKeyInFile(string(existing), envVar.Key) {
			continue
		}

		if envVar.Required {
			fmt.Fprintf(&appendBuf, "%s=\n", envVar.Key)
		} else {
			fmt.Fprintf(&appendBuf, "# %s=%s\n", envVar.Key, envVar.Default)
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
		Short: "Validate .env against application.properties requirements",
		Long: `Compares variables referenced by application.properties against .env and the OS environment.

  ✓  var is set
  ✗  required var is missing — exits with non-zero
  ⚠  optional var is missing`,
		Args: cobra.NoArgs,
		RunE: runCheck,
	}
}

func runCheck(_ *cobra.Command, _ []string) error {
	declaredEnvVars, source, err := loadDeclaredEnvVars()
	if err != nil {
		return err
	}
	if len(declaredEnvVars) == 0 {
		fmt.Printf("  ℹ  no environment variables declared in %s — nothing to check\n", source)
		return nil
	}

	dotEnv, _ := readFileFn(".env")
	dotEnvStr := string(dotEnv)

	fmt.Println()
	hasErrors := false

	for _, envVar := range declaredEnvVars {
		val, inFile := keeltoml.LookupEnvValue(dotEnvStr, envVar.Key)
		_, inOS := lookupOSEnvFn(envVar.Key)

		switch {
		case inFile && val != "":
			fmt.Printf("  ✓  %s is set\n", envVar.Key)
		case inOS:
			fmt.Printf("  ✓  %s is set (via OS env)\n", envVar.Key)
		case inFile && val == "" && !envVar.Required:
			fmt.Printf("  ✓  %s is intentionally empty\n", envVar.Key)
		case envVar.Required:
			fmt.Printf("  ✗  %s is missing\n", envVar.Key)
			hasErrors = true
		default:
			fmt.Printf("  ⚠  %s is not set (optional)\n", envVar.Key)
		}
	}

	fmt.Println()
	if hasErrors {
		return fmt.Errorf("missing required environment variables")
	}
	return nil
}

func loadDeclaredEnvVars() ([]declaredEnvVar, string, error) {
	if _, err := statFileFn(appproperties.DefaultPath); err == nil {
		doc, err := loadApplicationPropertiesFn(appproperties.DefaultPath)
		if err != nil {
			return nil, appproperties.DefaultPath, fmt.Errorf("reading %s: %w", appproperties.DefaultPath, err)
		}

		envVars := make([]declaredEnvVar, 0, len(doc.EnvVars))
		for _, envVar := range doc.EnvVars {
			envVars = append(envVars, declaredEnvVar{
				Key:        envVar.Key,
				Default:    envVar.Default,
				HasDefault: envVar.HasDefault,
				Required:   !envVar.HasDefault,
			})
		}
		return envVars, appproperties.DefaultPath, nil
	}

	kt, err := loadKeelTomlFn(keeltoml.DefaultPath)
	if err != nil {
		return nil, keeltoml.DefaultPath, fmt.Errorf("reading %s: %w", keeltoml.DefaultPath, err)
	}

	envVars := make([]declaredEnvVar, 0, len(kt.Env))
	for _, envVar := range kt.Env {
		envVars = append(envVars, declaredEnvVar{
			Key:         envVar.Key,
			Default:     envVar.Default,
			HasDefault:  envVar.Default != "",
			Description: envVar.Description,
			Required:    envVar.Required,
			Secret:      envVar.Secret,
		})
	}

	return envVars, keeltoml.DefaultPath, nil
}

func exampleValue(envVar declaredEnvVar) string {
	if envVar.Secret && envVar.Default == "" {
		return "your-secret-here"
	}
	if envVar.HasDefault {
		return envVar.Default
	}
	return ""
}

func buildComment(envVar declaredEnvVar) string {
	if envVar.Description == "" {
		return ""
	}
	return "# " + envVar.Description
}

// envKeyInFile reports whether key is already present in the env file content,
// either as an active entry (KEY=value) or as a commented-out template (# KEY=value).
// This prevents runGenerate from appending duplicate commented entries on repeated runs.
func envKeyInFile(content, key string) bool {
	if _, ok := keeltoml.LookupEnvValue(content, key); ok {
		return true
	}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		inner := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
		if strings.HasPrefix(inner, key+"=") {
			return true
		}
	}
	return false
}
