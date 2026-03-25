package doctor

import (
	"fmt"
	"os"
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
)

func runDoctor(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println("  Keel Doctor — project health check")
	fmt.Println()

	hasErrors := false

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
	case ok:
		checkRequiredEnvVars(kt, string(envData), &hasErrors)
	}

	printSummary(hasErrors)
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

func printSummary(hasErrors bool) {
	fmt.Println()
	if hasErrors {
		fmt.Println("  ✗  doctor found issues — fix them before running the application")
	} else {
		fmt.Println("  ✓  project looks healthy")
	}
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
