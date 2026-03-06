package addon

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	registryURL      = "https://raw.githubusercontent.com/slice-soft/ss-keel-addons/main/registry.json"
	registryCacheTTL = time.Hour
)

// RegistryEntry is a single addon entry in the official registry.
type RegistryEntry struct {
	Alias       string `json:"alias"`
	Repo        string `json:"repo"`
	Description string `json:"description"`
	Official    bool   `json:"official"`
}

// Registry is the full addon registry fetched from ss-keel-addons.
type Registry struct {
	Version string          `json:"version"`
	Addons  []RegistryEntry `json:"addons"`
}

// ResolveRepo maps an alias (e.g. "gorm") to a Go module path.
// Returns ("", false) when the alias is not in the registry.
func (r *Registry) ResolveRepo(alias string) (string, bool) {
	for _, a := range r.Addons {
		if a.Alias == alias {
			return a.Repo, true
		}
	}
	return "", false
}

// FetchRegistry returns the addon registry, using a local cache when fresh.
// Pass forceRefresh=true to bypass the cache.
func FetchRegistry(forceRefresh bool) (*Registry, error) {
	cachePath := registryCachePath()

	if !forceRefresh {
		if reg, ok := loadCachedRegistry(cachePath); ok {
			return reg, nil
		}
	}

	reg, err := fetchRegistryFromNetwork()
	if err != nil {
		// Fall back to stale cache rather than failing completely.
		if reg, ok := loadCachedRegistry(cachePath); ok {
			return reg, nil
		}
		return nil, fmt.Errorf("could not fetch addon registry: %w", err)
	}

	saveRegistryCache(cachePath, reg)
	return reg, nil
}

func fetchRegistryFromNetwork() (*Registry, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(registryURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var reg Registry
	if err := json.Unmarshal(body, &reg); err != nil {
		return nil, fmt.Errorf("invalid registry format: %w", err)
	}
	return &reg, nil
}

type cachedRegistry struct {
	FetchedAt time.Time `json:"fetched_at"`
	Registry  Registry  `json:"registry"`
}

func loadCachedRegistry(path string) (*Registry, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var cached cachedRegistry
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false
	}
	if time.Since(cached.FetchedAt) > registryCacheTTL {
		return nil, false
	}
	return &cached.Registry, true
}

func saveRegistryCache(path string, reg *Registry) {
	os.MkdirAll(filepath.Dir(path), 0755)
	cached := cachedRegistry{FetchedAt: time.Now(), Registry: *reg}
	data, err := json.Marshal(cached)
	if err != nil {
		return
	}
	os.WriteFile(path, data, 0644)
}

func registryCachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".keel", "registry.json")
}
