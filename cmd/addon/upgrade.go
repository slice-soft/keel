package addon

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/slice-soft/keel/internal/addon"
	"github.com/slice-soft/keel/internal/gomod"
	"github.com/slice-soft/keel/internal/keeltoml"
	"github.com/spf13/cobra"
)

var (
	upgradeForceRefresh bool

	upgradeGoGetFn            = runGoGet
	upgradeInstalledVersionFn = addon.InstalledVersion
)

func newUpgradeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade [alias]",
		Short: "Upgrade addons to their latest version",
		Long: `Upgrade one or all Keel addons installed in the current project.

  keel addon upgrade          # upgrade all installed addons
  keel addon upgrade gorm     # upgrade only gorm`,
		Args: cobra.MaximumNArgs(1),
		RunE: runUpgrade,
	}
	cmd.Flags().BoolVar(&upgradeForceRefresh, "refresh", false, "Force refresh of the addon registry cache")
	return cmd
}

func runUpgrade(_ *cobra.Command, args []string) error {
	if err := validateKeelProject(); err != nil {
		return err
	}

	kt, err := keeltoml.Load(keeltoml.DefaultPath)
	if err != nil {
		return fmt.Errorf("could not read keel.toml: %w", err)
	}

	targets := kt.Addons
	if len(args) == 1 {
		entry, ok := findAddon(kt, args[0])
		if !ok {
			return fmt.Errorf("addon %q not found in keel.toml", args[0])
		}
		targets = []keeltoml.AddonEntry{entry}
	}

	if len(targets) == 0 {
		fmt.Println("  No addons installed.")
		return nil
	}

	fmt.Printf("\n  Upgrading %d addon(s)...\n\n", len(targets))

	upgraded := 0
	for _, entry := range targets {
		changed, err := upgradeAddon(entry)
		if err != nil {
			fmt.Printf("  ⚠  %s: %v\n", entry.ID, err)
			continue
		}
		if changed {
			upgraded++
		}
	}

	if upgraded > 0 {
		if err := gomod.RunTidy(nil, ".", os.Stdout, os.Stderr); err != nil {
			fmt.Printf("  ⚠  %v\n", err)
		}
	}

	fmt.Printf("\n  ✓ upgrade complete (%d updated)\n\n", upgraded)
	return nil
}

func upgradeAddon(entry keeltoml.AddonEntry) (bool, error) {
	if entry.Repo == "" {
		return false, fmt.Errorf("no repo in keel.toml for addon %q", entry.ID)
	}

	prevVersion := upgradeInstalledVersionFn(entry.Repo)

	if err := upgradeGoGetFn(entry.Repo); err != nil {
		return false, err
	}

	newVersion := upgradeInstalledVersionFn(entry.Repo)
	if newVersion == "" {
		newVersion = prevVersion
	}

	if newVersion == prevVersion {
		fmt.Printf("  ✓ %s already at %s\n", entry.ID, prevVersion)
		return false, nil
	}

	if _, err := keeltoml.UpdateAddonVersion(keeltoml.DefaultPath, entry.ID, newVersion); err != nil {
		fmt.Printf("  ⚠  could not update keel.toml for %s: %v\n", entry.ID, err)
	}

	fmt.Printf("  ✓ %s %s → %s\n", entry.ID, prevVersion, newVersion)
	return true, nil
}

func runGoGet(repo string) error {
	target := repo + "@latest"
	fmt.Printf("  → go get %s\n", target)
	cmd := exec.Command("go", "get", target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go get %s: %w", target, err)
	}
	return nil
}
