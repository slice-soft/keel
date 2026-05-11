package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	keelAddonOrg          = "slice-soft"
	keelAddonModulePrefix = "github.com/slice-soft/ss-keel-"
	addonCheckInterval    = 24 * time.Hour
)

// AddonUpdate describes an installed addon with an available newer version.
type AddonUpdate struct {
	Name    string // short addon ID, e.g. "jwt"
	Current string // e.g. "v0.3.0"
	Latest  string // e.g. "v0.5.1"
}

var fetchAddonReleaseFn = fetchAddonLatestRelease

// IsNewer reports whether latest is a newer semver than current.
// Exported wrapper around the internal isNewer for use by other packages.
func IsNewer(latest, current string) bool {
	return isNewer(latest, current)
}

// ParseModuleVersion finds the version of a module path in go.mod content.
// Returns the version string (e.g. "v0.3.0") and whether it was found.
func ParseModuleVersion(goModContent, modulePath string) (string, bool) {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(modulePath) + `\s+(v\S+)`)
	if m := re.FindStringSubmatch(goModContent); len(m) > 1 {
		return m[1], true
	}
	return "", false
}

// FetchAddonLatestVersion queries GitHub for the latest release tag of an official ss-keel-* addon.
// addonID is the short name, e.g. "jwt" → repo "ss-keel-jwt".
func FetchAddonLatestVersion(addonID string) (string, error) {
	return fetchAddonReleaseFn(addonID)
}

func fetchAddonLatestRelease(addonID string) (string, error) {
	repo := "ss-keel-" + addonID
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", keelAddonOrg, repo)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API %d for %s", resp.StatusCode, repo)
	}
	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}

// parseInstalledKeelAddons extracts all github.com/slice-soft/ss-keel-* entries
// from go.mod content. Returns a map of addonID → version.
func parseInstalledKeelAddons(goModContent string) map[string]string {
	result := make(map[string]string)
	re := regexp.MustCompile(`(?m)^\s*(github\.com/slice-soft/ss-keel-\S+)\s+(v\S+)`)
	for _, m := range re.FindAllStringSubmatch(goModContent, -1) {
		module, version := m[1], m[2]
		name := strings.TrimPrefix(module, keelAddonModulePrefix)
		if name != "" && name != module {
			result[name] = version
		}
	}
	return result
}

// CheckAddonUpdatesAsync checks GitHub for the latest release of each installed
// ss-keel-* addon found in goModContent. Uses a 24h cache to avoid repeated API
// calls. Returns a channel that emits a (possibly nil) slice of AddonUpdate.
func CheckAddonUpdatesAsync(goModContent string) chan []AddonUpdate {
	ch := make(chan []AddonUpdate, 1)

	go func() {
		defer close(ch)

		if !shouldCheckAddons() {
			ch <- nil
			return
		}

		installed := parseInstalledKeelAddons(goModContent)
		if len(installed) == 0 {
			ch <- nil
			return
		}

		saveLastAddonCheck()

		var outdated []AddonUpdate
		for name, current := range installed {
			latest, err := fetchAddonReleaseFn(name)
			if err != nil || latest == "" {
				continue
			}
			if isNewer(latest, current) {
				outdated = append(outdated, AddonUpdate{Name: name, Current: current, Latest: latest})
			}
		}

		ch <- outdated
	}()

	return ch
}

func lastAddonCheckFile() string {
	return filepath.Join(keelDir(), "last_addon_check")
}

func shouldCheckAddons() bool {
	data, err := os.ReadFile(lastAddonCheckFile())
	if err != nil {
		return true
	}
	var last time.Time
	if err := last.UnmarshalText(data); err != nil {
		return true
	}
	return time.Since(last) > addonCheckInterval
}

func saveLastAddonCheck() {
	os.MkdirAll(keelDir(), 0755)
	data, _ := time.Now().MarshalText()
	os.WriteFile(lastAddonCheckFile(), data, 0644)
}
