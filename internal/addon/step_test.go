package addon

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previous)
	})
}

func TestRawManifestURL(t *testing.T) {
	url, err := rawManifestURL("github.com/slice-soft/ss-keel-gorm")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	want := "https://raw.githubusercontent.com/slice-soft/ss-keel-gorm/main/keel-addon.json"
	if url != want {
		t.Fatalf("expected %q, got %q", want, url)
	}

	_, err = rawManifestURL("gitlab.com/acme/addon")
	if err == nil || !strings.Contains(err.Error(), "only github.com repos are supported") {
		t.Fatalf("expected unsupported host error, got %v", err)
	}
}

func TestFetchManifest(t *testing.T) {
	t.Run("unsupported repo", func(t *testing.T) {
		_, err := FetchManifest("bitbucket.org/acme/addon")
		if err == nil || !strings.Contains(err.Error(), "only github.com repos are supported") {
			t.Fatalf("expected unsupported repo error, got %v", err)
		}
	})

	t.Run("local path success", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, addonManifestFile)
		body := `{"name":"local-addon","version":"1.0.0","repo":"github.com/acme/local-addon","steps":[{"type":"env","key":"TOKEN","example":"abc"}]}`
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatalf("failed to write manifest: %v", err)
		}

		manifest, err := FetchManifest(dir)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if manifest.Name != "local-addon" || manifest.Repo != "github.com/acme/local-addon" {
			t.Fatalf("unexpected manifest: %#v", manifest)
		}
	})

	t.Run("local path missing manifest", func(t *testing.T) {
		dir := t.TempDir()

		_, err := FetchManifest(dir)
		if err == nil || !strings.Contains(err.Error(), "does not have a keel-addon.json") {
			t.Fatalf("expected missing local manifest error, got %v", err)
		}
	})

	t.Run("request error", func(t *testing.T) {
		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})

		_, err := FetchManifest("github.com/acme/addon")
		if err == nil || !strings.Contains(err.Error(), "could not fetch keel-addon.json") {
			t.Fatalf("expected request error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       ioNopCloser("missing"),
				Header:     make(http.Header),
			}, nil
		})

		_, err := FetchManifest("github.com/acme/addon")
		if err == nil || !strings.Contains(err.Error(), "does not have a keel-addon.json") {
			t.Fatalf("expected not found error, got %v", err)
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

		_, err := FetchManifest("github.com/acme/addon")
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

		_, err := FetchManifest("github.com/acme/addon")
		if err == nil || !strings.Contains(err.Error(), "invalid keel-addon.json") {
			t.Fatalf("expected json error, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			want := "https://raw.githubusercontent.com/acme/addon/main/keel-addon.json"
			if req.URL.String() != want {
				t.Fatalf("unexpected URL: %s", req.URL.String())
			}
			body := `{"name":"gorm","version":"1.0.0","repo":"github.com/acme/addon","steps":[{"type":"env","key":"DB_HOST","example":"localhost"}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioNopCloser(body),
				Header:     make(http.Header),
			}, nil
		})

		manifest, err := FetchManifest("github.com/acme/addon")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if manifest.Name != "gorm" || manifest.Repo != "github.com/acme/addon" || len(manifest.Steps) != 1 {
			t.Fatalf("unexpected manifest: %#v", manifest)
		}
	})
}

func TestLoadLocalManifest(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		dir := t.TempDir()
		withWorkingDir(t, dir)

		_, err := LoadLocalManifest()
		if err == nil || !strings.Contains(err.Error(), "not found in current directory") {
			t.Fatalf("expected missing file error, got %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		dir := t.TempDir()
		withWorkingDir(t, dir)

		path := filepath.Join(dir, addonManifestFile)
		if err := os.WriteFile(path, []byte("{invalid"), 0644); err != nil {
			t.Fatalf("failed to write manifest: %v", err)
		}

		_, err := LoadLocalManifest()
		if err == nil || !strings.Contains(err.Error(), "invalid keel-addon.json") {
			t.Fatalf("expected invalid json error, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		dir := t.TempDir()
		withWorkingDir(t, dir)

		path := filepath.Join(dir, addonManifestFile)
		body := `{"name":"x","version":"1.0.0","steps":[{"type":"main_import","path":"github.com/acme/x"}]}`
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatalf("failed to write manifest: %v", err)
		}

		manifest, err := LoadLocalManifest()
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if manifest.Name != "x" || len(manifest.Steps) != 1 || manifest.Steps[0].Type != "main_import" {
			t.Fatalf("unexpected manifest: %#v", manifest)
		}
	})
}
