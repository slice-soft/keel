package add

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/slice-soft/keel/internal/addon"
	"github.com/spf13/cobra"
)

var forceRefresh bool

const invalidProjectMessage = "keel add must be executed inside a Keel project"

var (
	fetchRegistryFn = addon.FetchRegistry
	fetchManifestFn = addon.FetchManifest
	installAddonFn  = addon.Install
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [alias|repo]",
		Short: "Install a Keel addon into the current project",
		Long: `Install a Keel addon and wire it automatically into cmd/main.go.

  keel add gorm                              # official alias
  keel add github.com/username/my-addon      # any repo with keel-addon.json
  keel add ../path/to/my-addon               # local addon repo for unpublished changes`,
		Args: cobra.ExactArgs(1),
		RunE: runAdd,
	}
	cmd.Flags().BoolVar(&forceRefresh, "refresh", false, "Force refresh of the addon registry cache")
	return cmd
}

func runAdd(cmd *cobra.Command, args []string) error {
	if err := validateKeelProject(); err != nil {
		return err
	}

	target := strings.TrimSpace(args[0])

	reg, err := fetchRegistryFn(forceRefresh)
	if err != nil {
		// Non-fatal: we can still install by direct repo path.
		fmt.Fprintf(os.Stderr, "  ⚠  Could not fetch addon registry: %v\n", err)
	}

	repo, isOfficial := resolveRepo(target, reg)

	if !isOfficial {
		fmt.Printf("\n  ⚠  %q is not in the official Keel addon registry.\n", repo)
		fmt.Printf("     Verify community addons at: https://github.com/slice-soft/ss-keel-addons\n")
		fmt.Printf("     Install anyway? [y/N] ")

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			fmt.Println("  Aborted.")
			return nil
		}
	}

	fmt.Printf("\n  Installing %s...\n\n", repo)

	manifest, err := fetchManifestFn(repo)
	if err != nil {
		return err
	}

	// Resolve and offer to install declared dependencies before the addon itself.
	if err := handleDependencies(manifest.DependsOn, reg); err != nil {
		return err
	}

	if err := installAddonFn(manifest); err != nil {
		return err
	}

	fmt.Printf("\n  ✓ %s installed successfully\n\n", manifest.Name)
	return nil
}

// handleDependencies checks whether the addons listed in depends_on are already
// installed (present in go.mod) and, if not, offers to install them first.
func handleDependencies(deps []string, reg *addon.Registry) error {
	if len(deps) == 0 {
		return nil
	}

	goMod, err := os.ReadFile("go.mod")
	if err != nil {
		return nil // can't read go.mod — skip silently
	}

	reader := bufio.NewReader(os.Stdin)

	for _, dep := range deps {
		depRepo, _ := resolveRepo(dep, reg)

		// Dependency already installed — nothing to do.
		if strings.Contains(string(goMod), depRepo) {
			continue
		}

		fmt.Printf("\n  ℹ  This addon depends on %q which is not installed yet.\n", dep)
		fmt.Printf("     Install %q now? [Y/n] ", dep)

		answer, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) == "n" {
			fmt.Printf("  ⚠  Skipped dependency %q — run 'keel add %s' before using this addon.\n", dep, dep)
			continue
		}

		fmt.Printf("\n  Installing dependency %s...\n\n", depRepo)

		depManifest, err := fetchManifestFn(depRepo)
		if err != nil {
			return fmt.Errorf("failed to fetch dependency %q: %w", dep, err)
		}
		if err := installAddonFn(depManifest); err != nil {
			return fmt.Errorf("failed to install dependency %q: %w", dep, err)
		}

		fmt.Printf("\n  ✓ dependency %s installed\n", dep)
	}
	return nil
}

func validateKeelProject() error {
	required := []string{"go.mod", "cmd/main.go", "internal"}
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			return errors.New(invalidProjectMessage)
		}
	}
	return nil
}

// resolveRepo maps an alias or full repo path to a module path + whether it's official.
func resolveRepo(target string, reg *addon.Registry) (repo string, official bool) {
	// Full module path (e.g. github.com/user/repo) — skip registry lookup.
	if strings.Contains(target, "/") {
		return target, false
	}

	// Alias lookup.
	if reg != nil {
		if resolved, ok := reg.ResolveRepo(target); ok {
			return resolved, true
		}
	}

	// Unknown alias — treat as-is and warn.
	return target, false
}
