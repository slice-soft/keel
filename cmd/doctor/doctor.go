package doctor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/slice-soft/keel/internal/appproperties"
	"github.com/slice-soft/keel/internal/keeltoml"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose the health of the current Keel project",
		Long: `Check keel.toml consistency, addon installation, and required environment variables.

  ✓  ok
  ✗  error — must fix
  ⚠  warning — should fix`,
		Args: cobra.NoArgs,
		RunE: runDoctor,
	}
}

// injectable for tests
var (
	loadKeelTomlFn              = func(path string) (*keeltoml.KeelToml, error) { return keeltoml.Load(path) }
	loadApplicationPropertiesFn = func(path string) (*appproperties.Document, error) { return appproperties.Load(path) }
	readGoModFn                 = func() ([]byte, error) { return os.ReadFile("go.mod") }
	readDotEnvFn                = func() ([]byte, error) { return os.ReadFile(".env") }
	lookupOSEnvFn               = os.LookupEnv
	runGoModTidyDiffFn          = func() (string, error) { return runGoCommand("mod", "tidy", "-diff") }
	runGoBuildFn                = func() (string, error) { return runGoCommand("build", "./...") }
)

func runDoctor(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println("  Keel Doctor — project health check")
	fmt.Println()

	hasErrors := false
	hasWarnings := false

	// 1. keel.toml — valid and parseable.
	kt, ok := checkKeelToml(&hasErrors)
	appDoc, hasAppProperties := checkApplicationProperties(&hasErrors)

	// 2. Addons declared in keel.toml are installed in go.mod.
	if ok {
		goModData, _ := readGoModFn()
		checkAddonsInGoMod(kt, string(goModData), &hasErrors)
	}

	// 3. Required env vars are set.
	envData, _ := readDotEnvFn()
	switch {
	case hasAppProperties:
		checkRequiredPropertyEnvVars(appDoc, string(envData), &hasErrors)
		checkPlaceholderPropertyEnvVars(appDoc, string(envData), &hasWarnings)
	case ok:
		checkRequiredEnvVars(kt, string(envData), &hasErrors)
		checkPlaceholderLegacyEnvVars(kt, string(envData), &hasWarnings)
	}

	// 4. OAuth-specific: warn when installed but zero providers are configured.
	if ok {
		checkOAuthConfiguration(kt, string(envData), &hasWarnings)
	}

	checkCompileReadiness(&hasErrors)

	printSummary(hasErrors, hasWarnings)
	return summaryErr(hasErrors)
}

// checkKeelToml validates that keel.toml exists and is parseable.
// Returns the parsed doc (nil on failure) and whether the check passed.
func checkKeelToml(hasErrors *bool) (*keeltoml.KeelToml, bool) {
	if _, err := os.Stat(keeltoml.DefaultPath); os.IsNotExist(err) {
		checkWarn("keel.toml not found — run: keel init")
		// Not a hard error; addons/env checks are skipped.
		return nil, false
	}

	kt, err := loadKeelTomlFn(keeltoml.DefaultPath)
	if err != nil {
		checkErr(fmt.Sprintf("keel.toml is not valid TOML: %v", err))
		*hasErrors = true
		return nil, false
	}

	checkOk("keel.toml is valid")
	return kt, true
}

// checkApplicationProperties validates that application.properties exists and is parseable.
func checkApplicationProperties(hasErrors *bool) (*appproperties.Document, bool) {
	if _, err := os.Stat(appproperties.DefaultPath); os.IsNotExist(err) {
		checkWarn("application.properties not found — generate it for the new runtime config contract")
		return nil, false
	}

	doc, err := loadApplicationPropertiesFn(appproperties.DefaultPath)
	if err != nil {
		checkErr(fmt.Sprintf("application.properties is not valid: %v", err))
		*hasErrors = true
		return nil, false
	}

	checkOk("application.properties is valid")
	return doc, true
}

// checkAddonsInGoMod verifies each [[addons]] entry is present in go.mod.
func checkAddonsInGoMod(kt *keeltoml.KeelToml, goMod string, hasErrors *bool) {
	if len(kt.Addons) == 0 {
		checkWarn("no addons declared in keel.toml")
		return
	}
	for _, addon := range kt.Addons {
		needle := addonNeedle(addon)
		if strings.Contains(goMod, needle) {
			checkOk(fmt.Sprintf("addon %q found in go.mod", addon.ID))
		} else {
			checkErr(fmt.Sprintf("addon %q not found in go.mod — run: keel add %s", addon.ID, addon.ID))
			*hasErrors = true
		}
	}
}

// addonNeedle returns the string to search for in go.mod for a given addon.
// Prefers the full repo path when available, falls back to a heuristic.
func addonNeedle(addon keeltoml.AddonEntry) string {
	if addon.Repo != "" {
		return addon.Repo
	}
	// Heuristic: most official Keel addons are named ss-keel-<id>.
	return "ss-keel-" + addon.ID
}

// checkRequiredEnvVars verifies that each required [[env]] entry has a value.
func checkRequiredEnvVars(kt *keeltoml.KeelToml, dotEnv string, hasErrors *bool) {
	if len(kt.Env) == 0 {
		return
	}
	for _, ev := range kt.Env {
		if !ev.Required {
			continue
		}
		if _, ok := keeltoml.LookupEnvValue(dotEnv, ev.Key); ok {
			checkOk(fmt.Sprintf("required var %s is set", ev.Key))
			continue
		}
		if _, ok := lookupOSEnvFn(ev.Key); ok {
			checkOk(fmt.Sprintf("required var %s is set (via OS env)", ev.Key))
			continue
		}
		checkErr(fmt.Sprintf("required var %s is not set — add it to .env", ev.Key))
		*hasErrors = true
	}
}

func checkRequiredPropertyEnvVars(doc *appproperties.Document, dotEnv string, hasErrors *bool) {
	if len(doc.EnvVars) == 0 {
		return
	}

	for _, envVar := range doc.EnvVars {
		if envVar.HasDefault {
			continue
		}
		if _, ok := keeltoml.LookupEnvValue(dotEnv, envVar.Key); ok {
			checkOk(fmt.Sprintf("required var %s is set", envVar.Key))
			continue
		}
		if _, ok := lookupOSEnvFn(envVar.Key); ok {
			checkOk(fmt.Sprintf("required var %s is set (via OS env)", envVar.Key))
			continue
		}
		checkErr(fmt.Sprintf("required var %s is not set — add it to .env", envVar.Key))
		*hasErrors = true
	}
}

func checkPlaceholderPropertyEnvVars(doc *appproperties.Document, dotEnv string, hasWarnings *bool) {
	for _, envVar := range doc.EnvVars {
		if !looksSensitiveEnvKey(envVar.Key) {
			continue
		}
		value, source, ok := resolveEnvValue(dotEnv, envVar.Key)
		if !ok && envVar.HasDefault {
			value = envVar.Default
			source = "application.properties default"
			ok = true
		}
		if !ok || !isPlaceholderSecretValue(value) {
			continue
		}
		checkWarn(fmt.Sprintf("sensitive var %s uses an insecure placeholder (%s) — replace it before production", envVar.Key, source))
		*hasWarnings = true
	}
}

func checkPlaceholderLegacyEnvVars(kt *keeltoml.KeelToml, dotEnv string, hasWarnings *bool) {
	if len(kt.Env) == 0 {
		return
	}

	for _, envVar := range kt.Env {
		if !envVar.Secret && !looksSensitiveEnvKey(envVar.Key) {
			continue
		}
		value, source, ok := resolveEnvValue(dotEnv, envVar.Key)
		if !ok && envVar.Default != "" {
			value = envVar.Default
			source = "keel.toml default"
			ok = true
		}
		if !ok || !isPlaceholderSecretValue(value) {
			continue
		}
		checkWarn(fmt.Sprintf("sensitive var %s uses an insecure placeholder (%s) — replace it before production", envVar.Key, source))
		*hasWarnings = true
	}
}

// checkOAuthConfiguration warns when the oauth addon is installed but no
// provider has a complete credential pair (client ID + client secret).
// A project in this state mounts zero OAuth provider routes even though
// the addon reports as installed.
func checkOAuthConfiguration(kt *keeltoml.KeelToml, dotEnv string, hasWarnings *bool) {
	if !addonInstalled(kt, "oauth") {
		return
	}

	providers := []string{"GOOGLE", "GITHUB", "GITLAB"}
	for _, p := range providers {
		idKey := "OAUTH_" + p + "_CLIENT_ID"
		secretKey := "OAUTH_" + p + "_CLIENT_SECRET"

		idVal, idInEnv := keeltoml.LookupEnvValue(dotEnv, idKey)
		secretVal, secretInEnv := keeltoml.LookupEnvValue(dotEnv, secretKey)

		if !idInEnv {
			idVal, idInEnv = lookupOSEnvFn(idKey)
		}
		if !secretInEnv {
			secretVal, secretInEnv = lookupOSEnvFn(secretKey)
		}

		if idInEnv && secretInEnv && strings.TrimSpace(idVal) != "" && strings.TrimSpace(secretVal) != "" {
			return
		}
	}

	checkWarn("oauth addon is installed but no provider credentials are configured — " +
		"set at least one OAUTH_*_CLIENT_ID + OAUTH_*_CLIENT_SECRET pair in .env")
	*hasWarnings = true
}

// addonInstalled reports whether an addon with the given id is declared in keel.toml.
func addonInstalled(kt *keeltoml.KeelToml, id string) bool {
	for _, a := range kt.Addons {
		if a.ID == id {
			return true
		}
	}
	return false
}

func checkCompileReadiness(hasErrors *bool) {
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		checkWarn("go.mod not found — skipping module readiness checks")
		return
	}

	if _, err := os.Stat("cmd/main.go"); os.IsNotExist(err) {
		checkWarn("cmd/main.go not found — skipping compile-readiness checks")
		return
	}

	tidyOutput, err := runGoModTidyDiffFn()
	if err != nil {
		checkErr("go.mod/go.sum are not tidy — run: go mod tidy")
		printCommandOutput(tidyOutput)
		checkWarn("skipping go build ./... until module metadata is clean")
		*hasErrors = true
		return
	}
	checkOk("go.mod/go.sum are tidy")

	buildOutput, err := runGoBuildFn()
	if err != nil {
		checkErr("go build ./... failed")
		printCommandOutput(buildOutput)
		*hasErrors = true
		return
	}
	checkOk("go build ./... passed")
}

func printSummary(hasErrors, hasWarnings bool) {
	fmt.Println()
	fmt.Println(summaryMessage(hasErrors, hasWarnings))
	fmt.Println("  ℹ  checks are static (keel.toml, go.mod, env vars, go build) — runtime connectivity to databases, Redis, or external services is not verified")
	fmt.Println()
}

func summaryErr(hasErrors bool) error {
	if hasErrors {
		return fmt.Errorf("doctor found issues")
	}
	return nil
}

func checkOk(msg string) {
	fmt.Printf("  ✓  %s\n", msg)
}

func checkErr(msg string) {
	fmt.Printf("  ✗  %s\n", msg)
}

func checkWarn(msg string) {
	fmt.Printf("  ⚠  %s\n", msg)
}

func summaryMessage(hasErrors, hasWarnings bool) string {
	switch {
	case hasErrors:
		return "  ✗  doctor found issues — fix them before running the application"
	case hasWarnings:
		return "  ⚠  project looks healthy, but review warnings before production"
	default:
		return "  ✓  project looks healthy"
	}
}

func resolveEnvValue(dotEnv, key string) (value, source string, ok bool) {
	if value, ok := keeltoml.LookupEnvValue(dotEnv, key); ok {
		return value, ".env", true
	}
	if value, ok := lookupOSEnvFn(key); ok {
		return value, "OS env", true
	}
	return "", "", false
}

func looksSensitiveEnvKey(key string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(key))
	switch {
	case strings.Contains(normalized, "SECRET"):
		return true
	case strings.Contains(normalized, "PASSWORD"):
		return true
	case strings.Contains(normalized, "PRIVATE_KEY"):
		return true
	case strings.Contains(normalized, "SIGNING_KEY"):
		return true
	default:
		return false
	}
}

func isPlaceholderSecretValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}

	normalized := strings.ToLower(trimmed)
	switch {
	case strings.Contains(normalized, "change-me"):
		return true
	case strings.Contains(normalized, "changeme"):
		return true
	case strings.Contains(normalized, "replace-me"):
		return true
	case strings.Contains(normalized, "replace_this"):
		return true
	case strings.Contains(normalized, "your-secret"):
		return true
	case strings.Contains(normalized, "example"):
		return true
	case strings.Contains(normalized, "sample"):
		return true
	case strings.Contains(normalized, "dev-secret"):
		return true
	case strings.Contains(normalized, "test-secret"):
		return true
	case normalized == "secret":
		return true
	default:
		return false
	}
}

func runGoCommand(args ...string) (string, error) {
	cmd := exec.Command("go", args...)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Run()
	return strings.TrimSpace(output.String()), err
}

func printCommandOutput(raw string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}

	lines := strings.Split(raw, "\n")
	if len(lines) > 8 {
		lines = append(lines[:8], "...")
	}

	for _, line := range lines {
		fmt.Printf("     %s\n", line)
	}
}
