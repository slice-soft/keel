package addon

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeCachedRegistry(t *testing.T, path string, cache cachedRegistry) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}
	data, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("failed to marshal cache: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write cache: %v", err)
	}
}

func TestResolveRepo(t *testing.T) {
	reg := &Registry{
		Addons: []RegistryEntry{
			{Alias: "gorm", Repo: "github.com/slice-soft/ss-keel-gorm"},
		},
	}

	repo, ok := reg.ResolveRepo("gorm")
	if !ok || repo != "github.com/slice-soft/ss-keel-gorm" {
		t.Fatalf("unexpected resolve result: repo=%q ok=%t", repo, ok)
	}

	repo, ok = reg.ResolveRepo("unknown")
	if ok || repo != "" {
		t.Fatalf("expected missing alias to return empty,false got repo=%q ok=%t", repo, ok)
	}
}

func TestFetchRegistryFromNetwork(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != registryURL {
				t.Fatalf("unexpected registry URL: %s", req.URL.String())
			}
			body := `{"version":"1","addons":[{"alias":"gorm","repo":"github.com/slice-soft/ss-keel-gorm","official":true}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioNopCloser(body),
				Header:     make(http.Header),
			}, nil
		})

		reg, err := fetchRegistryFromNetwork()
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if reg.Version != "1" || len(reg.Addons) != 1 || reg.Addons[0].Alias != "gorm" {
			t.Fatalf("unexpected registry: %#v", reg)
		}
	})

	t.Run("request error", func(t *testing.T) {
		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})

		_, err := fetchRegistryFromNetwork()
		if err == nil {
			t.Fatalf("expected request error, got nil")
		}
	})

	t.Run("status error", func(t *testing.T) {
		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       ioNopCloser("bad gateway"),
				Header:     make(http.Header),
			}, nil
		})

		_, err := fetchRegistryFromNetwork()
		if err == nil || !strings.Contains(err.Error(), "HTTP 502") {
			t.Fatalf("expected status error, got %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioNopCloser("{invalid"),
				Header:     make(http.Header),
			}, nil
		})

		_, err := fetchRegistryFromNetwork()
		if err == nil || !strings.Contains(err.Error(), "invalid registry format") {
			t.Fatalf("expected json format error, got %v", err)
		}
	})
}

func TestLoadCachedRegistry(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "registry.json")

	if reg, ok := loadCachedRegistry(cachePath); ok || reg != nil {
		t.Fatalf("expected missing cache to return nil,false")
	}

	if err := os.WriteFile(cachePath, []byte("{invalid"), 0644); err != nil {
		t.Fatalf("failed to write invalid cache: %v", err)
	}
	if reg, ok := loadCachedRegistry(cachePath); ok || reg != nil {
		t.Fatalf("expected invalid cache to return nil,false")
	}

	writeCachedRegistry(t, cachePath, cachedRegistry{
		FetchedAt: time.Now().Add(-2 * registryCacheTTL),
		Registry:  Registry{Version: "expired"},
	})
	if reg, ok := loadCachedRegistry(cachePath); ok || reg != nil {
		t.Fatalf("expected expired cache to return nil,false")
	}

	writeCachedRegistry(t, cachePath, cachedRegistry{
		FetchedAt: time.Now(),
		Registry: Registry{
			Version: "fresh",
			Addons:  []RegistryEntry{{Alias: "gorm", Repo: "github.com/slice-soft/ss-keel-gorm"}},
		},
	})
	reg, ok := loadCachedRegistry(cachePath)
	if !ok || reg == nil {
		t.Fatalf("expected fresh cache to load")
	}
	if reg.Version != "fresh" || len(reg.Addons) != 1 || reg.Addons[0].Alias != "gorm" {
		t.Fatalf("unexpected cached registry: %#v", reg)
	}
}

func TestSaveRegistryCacheAndPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := registryCachePath()
	wantPath := filepath.Join(home, ".keel", "registry.json")
	if path != wantPath {
		t.Fatalf("expected cache path %q, got %q", wantPath, path)
	}

	saveRegistryCache(path, &Registry{
		Version: "saved",
		Addons:  []RegistryEntry{{Alias: "x", Repo: "github.com/acme/x"}},
	})

	reg, ok := loadCachedRegistry(path)
	if !ok || reg == nil {
		t.Fatalf("expected saved cache to be readable")
	}
	if reg.Version != "saved" {
		t.Fatalf("unexpected saved registry: %#v", reg)
	}
}

func TestFetchRegistry(t *testing.T) {
	t.Run("uses fresh cache without network", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		cachePath := filepath.Join(home, ".keel", "registry.json")
		writeCachedRegistry(t, cachePath, cachedRegistry{
			FetchedAt: time.Now(),
			Registry: Registry{
				Version: "cached",
				Addons:  []RegistryEntry{{Alias: "gorm", Repo: "github.com/slice-soft/ss-keel-gorm"}},
			},
		})

		called := false
		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			called = true
			return nil, errors.New("should not be called")
		})

		reg, err := FetchRegistry(false)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if called {
			t.Fatalf("did not expect network call when cache is fresh")
		}
		if reg.Version != "cached" {
			t.Fatalf("expected cached version, got %q", reg.Version)
		}
	})

	t.Run("force refresh updates cache from network", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		cachePath := filepath.Join(home, ".keel", "registry.json")
		writeCachedRegistry(t, cachePath, cachedRegistry{
			FetchedAt: time.Now(),
			Registry:  Registry{Version: "old"},
		})

		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			body := `{"version":"new","addons":[{"alias":"sql","repo":"github.com/acme/sql","official":true}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioNopCloser(body),
				Header:     make(http.Header),
			}, nil
		})

		reg, err := FetchRegistry(true)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if reg.Version != "new" || len(reg.Addons) != 1 || reg.Addons[0].Alias != "sql" {
			t.Fatalf("unexpected refreshed registry: %#v", reg)
		}

		saved, ok := loadCachedRegistry(cachePath)
		if !ok || saved == nil || saved.Version != "new" {
			t.Fatalf("expected refreshed registry to be cached")
		}
	})

	t.Run("falls back to cache when network fails", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		cachePath := filepath.Join(home, ".keel", "registry.json")
		writeCachedRegistry(t, cachePath, cachedRegistry{
			FetchedAt: time.Now(),
			Registry:  Registry{Version: "cached"},
		})

		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})

		reg, err := FetchRegistry(true)
		if err != nil {
			t.Fatalf("expected fallback to cache, got error %v", err)
		}
		if reg.Version != "cached" {
			t.Fatalf("expected cached fallback, got %#v", reg)
		}
	})

	t.Run("returns error when network fails and cache missing", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})

		_, err := FetchRegistry(true)
		if err == nil || !strings.Contains(err.Error(), "could not fetch addon registry") {
			t.Fatalf("expected wrapped fetch error, got %v", err)
		}
	})
}

func ioNopCloser(body string) *readCloser {
	return &readCloser{reader: strings.NewReader(body)}
}

type readCloser struct {
	reader *strings.Reader
}

func (r *readCloser) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *readCloser) Close() error {
	return nil
}
