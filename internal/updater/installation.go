package updater

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
)

const (
	ModulePath              = "github.com/slice-soft/keel"
	HomebrewUpdateCommand   = "brew upgrade slice-soft/tap/keel"
	GoInstallUpdateCommand  = "go install github.com/slice-soft/keel@latest"
	ManualUpdateInstruction = "Please update Keel manually. To avoid this in the future, install Keel with Homebrew or go install."
	homebrewFormulaPathPart = "/Cellar/keel/"
)

type InstallationSource string

const (
	SourceHomebrew  InstallationSource = "Homebrew"
	SourceGoInstall InstallationSource = "go install"
	SourceUnknown   InstallationSource = "unknown"
)

type Installation struct {
	Source        InstallationSource
	Executable    string
	UpdateCommand string
}

func (i Installation) SupportsSelfUpgrade() bool {
	return false
}

func (i Installation) UpdateNotice() string {
	if i.UpdateCommand != "" {
		return "Update with: " + i.UpdateCommand
	}
	return ManualUpdateInstruction
}

func (i Installation) VersionUpdateLine() string {
	if i.UpdateCommand != "" {
		return "update command: " + i.UpdateCommand
	}
	return "update: " + ManualUpdateInstruction
}

var (
	userHomeDirFn   = os.UserHomeDir
	getenvFn        = os.Getenv
	readBuildInfoFn = debug.ReadBuildInfo
)

func DetectInstallation() Installation {
	executable, resolved := currentExecutablePath()
	if executable == "" && resolved == "" {
		return Installation{
			Source: SourceUnknown,
		}
	}

	paths := []string{executable}
	if resolved != "" && resolved != executable {
		paths = append(paths, resolved)
	}

	for _, path := range paths {
		if isHomebrewPath(path) {
			return Installation{
				Source:        SourceHomebrew,
				Executable:    path,
				UpdateCommand: HomebrewUpdateCommand,
			}
		}
	}

	for _, path := range paths {
		if isGoInstallPath(path) {
			return Installation{
				Source:        SourceGoInstall,
				Executable:    path,
				UpdateCommand: GoInstallUpdateCommand,
			}
		}
	}

	if buildInfoSuggestsGoInstall() {
		return Installation{
			Source:        SourceGoInstall,
			Executable:    bestExecutablePath(executable, resolved),
			UpdateCommand: GoInstallUpdateCommand,
		}
	}

	return Installation{
		Source:     SourceUnknown,
		Executable: bestExecutablePath(executable, resolved),
	}
}

func currentExecutablePath() (string, string) {
	executable, err := executablePathFn()
	if err != nil {
		return "", ""
	}

	resolved, err := evalSymlinksFn(executable)
	if err != nil {
		return executable, ""
	}
	return executable, resolved
}

func bestExecutablePath(executable, resolved string) string {
	if resolved != "" {
		return resolved
	}
	return executable
}

func isHomebrewPath(path string) bool {
	return strings.Contains(filepath.ToSlash(filepath.Clean(path)), homebrewFormulaPathPart)
}

func isGoInstallPath(path string) bool {
	if filepath.Base(path) != keelExecutableName() {
		return false
	}
	for _, binDir := range goInstallBinDirs() {
		if samePath(filepath.Dir(path), binDir) {
			return true
		}
	}
	return false
}

func goInstallBinDirs() []string {
	if gobin := strings.TrimSpace(getenvFn("GOBIN")); gobin != "" {
		return []string{gobin}
	}

	if gopath := strings.TrimSpace(getenvFn("GOPATH")); gopath != "" {
		entries := filepath.SplitList(gopath)
		dirs := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry != "" {
				dirs = append(dirs, filepath.Join(entry, "bin"))
			}
		}
		if len(dirs) > 0 {
			return dirs
		}
	}

	if home, err := userHomeDirFn(); err == nil && home != "" {
		return []string{filepath.Join(home, "go", "bin")}
	}
	return nil
}

func samePath(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}

func keelExecutableName() string {
	if runtime.GOOS == "windows" {
		return "keel.exe"
	}
	return "keel"
}

func buildInfoSuggestsGoInstall() bool {
	info, ok := readBuildInfoFn()
	if !ok {
		return false
	}

	version := strings.TrimSpace(info.Main.Version)
	return info.Main.Path == ModulePath && version != "" && version != "(devel)"
}
