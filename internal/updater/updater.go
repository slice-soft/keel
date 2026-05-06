package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	repoOwner     = "slice-soft"
	repoName      = "keel-cli"
	checkInterval = 24 * time.Hour
)

var fetchLatestReleaseFn = fetchLatestRelease
var downloadBinaryFn = downloadBinary
var replaceBinaryFn = replaceBinary
var executablePathFn = os.Executable
var evalSymlinksFn = filepath.EvalSymlinks
var removeFileFn = os.Remove
var runUpdateCommandFn = runUpdateCommand

// Release represents the GitHub API release response.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckAndNotify checks for a new version in the background.
// Returns a channel that emits an update notice (or empty string if no update).
// Non-blocking — the caller reads the channel at the end of the command.
func CheckAndNotify(currentVersion string) chan string {
	ch := make(chan string, 1)

	go func() {
		defer close(ch)

		if !shouldCheck() {
			ch <- ""
			return
		}

		latest, err := fetchLatestVersion()
		if err != nil || latest == "" {
			ch <- ""
			return
		}

		saveLastCheck()

		if isNewer(latest, currentVersion) {
			install := DetectInstallation()
			ch <- fmt.Sprintf(
				"\n  💡 New version available: %s (you have %s)\n     %s\n",
				latest, currentVersion, install.UpdateNotice(),
			)
		} else {
			ch <- ""
		}
	}()

	return ch
}

// Upgrade runs the supported update command for the detected installation.
func Upgrade(currentVersion string) error {
	install := DetectInstallation()
	if install.UpdateCommand != "" {
		fmt.Printf("\n⚓  Keel CLI is managed by %s.\n", install.Source)
		fmt.Printf("  Running: %s\n\n", install.UpdateCommand)
		if err := runUpdateCommandFn(install.UpdateCommand); err != nil {
			return fmt.Errorf("error running %q: %w", install.UpdateCommand, err)
		}
		fmt.Print("\n  ✅ Keel update command completed\n\n")
		return nil
	}

	if install.Source == SourceUnknown {
		fmt.Println("\n⚓  Keel CLI installation source could not be detected.")
		fmt.Printf("  %s\n\n", ManualUpdateInstruction)
		return nil
	}

	fmt.Println("\n⚓  Checking latest version...")

	release, err := fetchLatestReleaseFn()
	if err != nil {
		return fmt.Errorf("error querying GitHub: %w", err)
	}

	if !isNewer(release.TagName, currentVersion) {
		fmt.Printf("  ✅ You already have the latest version (%s)\n\n", currentVersion)
		return nil
	}

	fmt.Printf("  New version: %s (you have %s)\n", release.TagName, currentVersion)

	assetName := buildAssetName()
	downloadURL := ""
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("  Downloading %s...\n", assetName)

	tmpFile, err := downloadBinaryFn(downloadURL)
	if err != nil {
		return fmt.Errorf("error downloading binary: %w", err)
	}
	defer removeFileFn(tmpFile)

	execPath, err := executablePathFn()
	if err != nil {
		return fmt.Errorf("error resolving executable path: %w", err)
	}
	execPath, _ = evalSymlinksFn(execPath)

	fmt.Println("  Installing...")

	if err := replaceBinaryFn(tmpFile, execPath); err != nil {
		return fmt.Errorf("error installing: %w", err)
	}

	fmt.Printf("\n  ✅ keel updated to %s\n\n", release.TagName)
	return nil
}

func runUpdateCommand(command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty update command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func fetchLatestVersion() (string, error) {
	r, err := fetchLatestRelease()
	if err != nil {
		return "", err
	}
	return r.TagName, nil
}

func fetchLatestRelease() (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API responded %d", resp.StatusCode)
	}
	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func downloadBinary(url string) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	tmp, err := os.CreateTemp("", "keel-update-*")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return "", err
	}
	return tmp.Name(), nil
}

// replaceBinary atomically replaces the current binary with a safety backup.
func replaceBinary(newBinary, targetPath string) error {
	if err := os.Chmod(newBinary, 0755); err != nil {
		return err
	}
	backupPath := targetPath + ".bak"
	if err := os.Rename(targetPath, backupPath); err != nil {
		return err
	}
	if err := os.Rename(newBinary, targetPath); err != nil {
		os.Rename(backupPath, targetPath) // restore if install fails
		return err
	}
	os.Remove(backupPath)
	return nil
}

// buildAssetName builds the release asset name for the current OS/arch.
func buildAssetName() string {
	name := fmt.Sprintf("keel_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func keelDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".keel")
}

func lastCheckFile() string {
	return filepath.Join(keelDir(), "last_check")
}

func shouldCheck() bool {
	data, err := os.ReadFile(lastCheckFile())
	if err != nil {
		return true
	}
	var last time.Time
	if err := last.UnmarshalText(data); err != nil {
		return true
	}
	return time.Since(last) > checkInterval
}

func saveLastCheck() {
	os.MkdirAll(keelDir(), 0755)
	data, _ := time.Now().MarshalText()
	os.WriteFile(lastCheckFile(), data, 0644)
}

// isNewer compares semver versions using simple string comparison.
func isNewer(latest, current string) bool {
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")
	return latest != current && latest > current
}
