package upgrade

import (
	"testing"
)

func TestNewCommandRunsUpgradeWithCurrentVersion(t *testing.T) {
	previousUpgradeFn := upgradeFn
	t.Cleanup(func() {
		upgradeFn = previousUpgradeFn
	})

	called := false
	upgradeFn = func(currentVersion string) error {
		called = true
		if currentVersion != "v1.2.3" {
			t.Fatalf("expected current version v1.2.3, got %q", currentVersion)
		}
		return nil
	}

	cmd := NewCommand(func() string { return "v1.2.3" })
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !called {
		t.Fatalf("expected upgrade function to be called")
	}
}
