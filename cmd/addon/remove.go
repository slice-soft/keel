package addon

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/slice-soft/keel/internal/addon"
	"github.com/slice-soft/keel/internal/keeltoml"
	"github.com/spf13/cobra"
)

var (
	removeAutoApprove bool

	removeFetchManifestFn = addon.FetchManifest
	removeUninstallFn     = addon.Uninstall
)

func newRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <alias>",
		Short: "Remove a Keel addon from the current project",
		Long: `Remove a Keel addon and undo its wiring from cmd/main.go, env files, and keel.toml.

  keel addon remove gorm
  keel addon remove jwt`,
		Args: cobra.ExactArgs(1),
		RunE: runRemove,
	}
	cmd.Flags().BoolVarP(&removeAutoApprove, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func runRemove(_ *cobra.Command, args []string) error {
	if err := validateKeelProject(); err != nil {
		return err
	}

	id := strings.TrimSpace(args[0])

	kt, err := keeltoml.Load(keeltoml.DefaultPath)
	if err != nil {
		return fmt.Errorf("could not read keel.toml: %w", err)
	}

	entry, ok := findAddon(kt, id)
	if !ok {
		return fmt.Errorf("addon %q not found in keel.toml", id)
	}

	if !removeAutoApprove {
		fmt.Printf("\n  Remove addon %q (%s)?\n", id, entry.Version)
		fmt.Printf("  This will undo wiring in cmd/main.go, env files, and keel.toml. [y/N] ")

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			fmt.Println("  Aborted.")
			return nil
		}
	}

	fmt.Printf("\n  Removing %s...\n\n", id)

	manifest, err := removeFetchManifestFn(entry.Repo)
	if err != nil {
		return fmt.Errorf("could not fetch addon manifest for %s: %w\n\n  Tip: remove the [[addons]] entry from keel.toml manually and run go mod tidy", entry.Repo, err)
	}

	if err := removeUninstallFn(manifest); err != nil {
		return err
	}

	fmt.Printf("\n  ✓ %s removed successfully\n\n", id)
	return nil
}

func findAddon(kt *keeltoml.KeelToml, id string) (keeltoml.AddonEntry, bool) {
	for _, a := range kt.Addons {
		if a.ID == id {
			return a, true
		}
	}
	return keeltoml.AddonEntry{}, false
}

func validateKeelProject() error {
	required := []string{"go.mod", "cmd/main.go", "internal"}
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			return errors.New("keel addon must be executed inside a Keel project")
		}
	}
	return nil
}
