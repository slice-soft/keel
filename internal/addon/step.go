package addon

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const addonManifestFile = "keel-addon.json"

// Manifest is the keel-addon.json structure from an addon repo.
type Manifest struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Repo         string   `json:"repo"`
	DependsOn    []string `json:"depends_on,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Resources    []string `json:"resources,omitempty"`
	Steps        []Step   `json:"steps"`
}

// Step is a single installation action defined in keel-addon.json.
type Step struct {
	// Type is one of: go_get | env | property | main_import | main_code |
	// create_provider_file | note
	Type string `json:"type"`

	// go_get
	Package string `json:"package,omitempty"`

	// env | property
	Key         string `json:"key,omitempty"`
	Example     string `json:"example,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
	Message     string `json:"message,omitempty"`

	// main_import
	Path string `json:"path,omitempty"`

	// main_code
	Anchor string `json:"anchor,omitempty"` // "before_listen" | "before_modules"
	Guard  string `json:"guard,omitempty"`  // skip if already present
	// Replace, when set, replaces the first line in cmd/main.go containing this
	// substring with Code before falling back to insertion.
	Replace string `json:"replace,omitempty"`
	Code    string `json:"code,omitempty"`

	// create_provider_file creates a dedicated setup file with an initializer
	// function, keeping cmd/main.go clean. Guard checks the file before creating.
	Filename string `json:"filename,omitempty"` // e.g. "cmd/setup_provider.go"
	Content  string `json:"content,omitempty"`  // full Go source for the file
}

// FetchManifest downloads keel-addon.json from a Go module path.
// Supports github.com module paths and local addon directories.
func FetchManifest(repo string) (*Manifest, error) {
	if manifest, ok, err := loadManifestFromLocalPath(repo); ok {
		return manifest, err
	}

	rawURL, err := rawManifestURL(repo)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("could not fetch keel-addon.json from %s: %w", repo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%s does not have a keel-addon.json — it may not be a Keel addon", repo)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("keel-addon.json fetch returned HTTP %d for %s", resp.StatusCode, repo)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var m Manifest
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("invalid keel-addon.json in %s: %w", repo, err)
	}
	return &m, nil
}

func loadManifestFromLocalPath(repo string) (*Manifest, bool, error) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return nil, false, nil
	}

	if !filepath.IsAbs(repo) && !strings.HasPrefix(repo, ".") {
		return nil, false, nil
	}

	info, err := os.Stat(repo)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, true, fmt.Errorf("could not access local addon path %s: %w", repo, err)
	}
	if !info.IsDir() {
		return nil, true, fmt.Errorf("local addon path %s is not a directory", repo)
	}

	path := filepath.Join(repo, addonManifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, true, fmt.Errorf("%s does not have a %s", repo, addonManifestFile)
		}
		return nil, true, fmt.Errorf("could not read %s: %w", path, err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, true, fmt.Errorf("invalid keel-addon.json in %s: %w", repo, err)
	}
	return &m, true, nil
}

// rawManifestURL builds the raw content URL for keel-addon.json given a Go module path.
// Only github.com is supported at this time.
func rawManifestURL(repo string) (string, error) {
	// repo = "github.com/slice-soft/ss-keel-gorm"
	if !strings.HasPrefix(repo, "github.com/") {
		return "", fmt.Errorf("only github.com repos are supported (got %q)", repo)
	}
	// strip "github.com/" prefix → "slice-soft/ss-keel-gorm"
	path := strings.TrimPrefix(repo, "github.com/")
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/main/%s", path, addonManifestFile), nil
}

// LoadLocalManifest reads keel-addon.json from the current directory.
// Used for testing an addon locally before publishing.
func LoadLocalManifest() (*Manifest, error) {
	data, err := os.ReadFile(addonManifestFile)
	if err != nil {
		return nil, fmt.Errorf("keel-addon.json not found in current directory")
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid keel-addon.json: %w", err)
	}
	return &m, nil
}
