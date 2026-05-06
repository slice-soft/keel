package updater

import (
	"path/filepath"
	"runtime/debug"
	"testing"
)

func resetInstallationDeps(t *testing.T) {
	t.Helper()
	previousExecutablePath := executablePathFn
	previousEvalSymlinks := evalSymlinksFn
	previousUserHomeDir := userHomeDirFn
	previousGetenv := getenvFn
	previousReadBuildInfo := readBuildInfoFn
	t.Cleanup(func() {
		executablePathFn = previousExecutablePath
		evalSymlinksFn = previousEvalSymlinks
		userHomeDirFn = previousUserHomeDir
		getenvFn = previousGetenv
		readBuildInfoFn = previousReadBuildInfo
	})

	readBuildInfoFn = func() (*debug.BuildInfo, bool) {
		return nil, false
	}
	getenvFn = func(string) string {
		return ""
	}
}

func TestDetectInstallation(t *testing.T) {
	t.Run("detects Homebrew through resolved Cellar path", func(t *testing.T) {
		resetInstallationDeps(t)
		executablePathFn = func() (string, error) {
			return "/opt/homebrew/bin/keel", nil
		}
		evalSymlinksFn = func(path string) (string, error) {
			return "/opt/homebrew/Cellar/keel/1.2.3/bin/keel", nil
		}

		got := DetectInstallation()
		if got.Source != SourceHomebrew {
			t.Fatalf("expected Homebrew source, got %s", got.Source)
		}
		if got.UpdateCommand != HomebrewUpdateCommand {
			t.Fatalf("expected Homebrew update command, got %q", got.UpdateCommand)
		}
		if got.SupportsSelfUpgrade() {
			t.Fatalf("expected Homebrew install to disable self-upgrade")
		}
	})

	t.Run("detects go install through GOBIN", func(t *testing.T) {
		resetInstallationDeps(t)
		binDir := t.TempDir()
		executable := filepath.Join(binDir, keelExecutableName())
		getenvFn = func(key string) string {
			if key == "GOBIN" {
				return binDir
			}
			return ""
		}
		executablePathFn = func() (string, error) {
			return executable, nil
		}
		evalSymlinksFn = func(path string) (string, error) {
			return path, nil
		}

		got := DetectInstallation()
		if got.Source != SourceGoInstall {
			t.Fatalf("expected go install source, got %s", got.Source)
		}
		if got.UpdateCommand != GoInstallUpdateCommand {
			t.Fatalf("expected go install update command, got %q", got.UpdateCommand)
		}
		if got.SupportsSelfUpgrade() {
			t.Fatalf("expected go install to disable self-upgrade")
		}
	})

	t.Run("detects go install from embedded module version", func(t *testing.T) {
		resetInstallationDeps(t)
		executablePathFn = func() (string, error) {
			return filepath.Join(t.TempDir(), keelExecutableName()), nil
		}
		evalSymlinksFn = func(path string) (string, error) {
			return path, nil
		}
		readBuildInfoFn = func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{
				Main: debug.Module{
					Path:    ModulePath,
					Version: "v1.2.3",
				},
			}, true
		}

		got := DetectInstallation()
		if got.Source != SourceGoInstall {
			t.Fatalf("expected go install source, got %s", got.Source)
		}
	})

	t.Run("falls back to unknown with manual update instruction", func(t *testing.T) {
		resetInstallationDeps(t)
		executablePathFn = func() (string, error) {
			return filepath.Join(t.TempDir(), keelExecutableName()), nil
		}
		evalSymlinksFn = func(path string) (string, error) {
			return path, nil
		}

		got := DetectInstallation()
		if got.Source != SourceUnknown {
			t.Fatalf("expected unknown source, got %s", got.Source)
		}
		if got.UpdateCommand != "" {
			t.Fatalf("expected empty update command, got %q", got.UpdateCommand)
		}
		if got.SupportsSelfUpgrade() {
			t.Fatalf("expected unknown install to disable self-upgrade")
		}
		if got.UpdateNotice() != ManualUpdateInstruction {
			t.Fatalf("expected manual update notice, got %q", got.UpdateNotice())
		}
		if got.VersionUpdateLine() != "update: "+ManualUpdateInstruction {
			t.Fatalf("expected manual version update line, got %q", got.VersionUpdateLine())
		}
	})
}
