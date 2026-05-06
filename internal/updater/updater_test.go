package updater

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func stubHTTPTransport(t *testing.T, fn roundTripFunc) {
	t.Helper()
	previous := http.DefaultTransport
	http.DefaultTransport = fn
	t.Cleanup(func() {
		http.DefaultTransport = previous
	})
}

func readUpdateMessage(t *testing.T, ch chan string) string {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for update message")
		return ""
	}
}

func resetUpgradeDeps(t *testing.T) {
	t.Helper()
	previousFetchLatestRelease := fetchLatestReleaseFn
	previousDownloadBinary := downloadBinaryFn
	previousReplaceBinary := replaceBinaryFn
	previousExecutablePath := executablePathFn
	previousEvalSymlinks := evalSymlinksFn
	previousRemoveFile := removeFileFn
	previousUserHomeDir := userHomeDirFn
	previousGetenv := getenvFn
	previousReadBuildInfo := readBuildInfoFn
	previousRunUpdateCommand := runUpdateCommandFn
	t.Cleanup(func() {
		fetchLatestReleaseFn = previousFetchLatestRelease
		downloadBinaryFn = previousDownloadBinary
		replaceBinaryFn = previousReplaceBinary
		executablePathFn = previousExecutablePath
		evalSymlinksFn = previousEvalSymlinks
		removeFileFn = previousRemoveFile
		userHomeDirFn = previousUserHomeDir
		getenvFn = previousGetenv
		readBuildInfoFn = previousReadBuildInfo
		runUpdateCommandFn = previousRunUpdateCommand
	})
}

func TestFetchLatestRelease(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			if req.URL.Host != "api.github.com" {
				t.Fatalf("unexpected host: %s", req.URL.Host)
			}
			if req.URL.Path != "/repos/slice-soft/keel-cli/releases/latest" {
				t.Fatalf("unexpected path: %s", req.URL.Path)
			}
			body := `{"tag_name":"v1.2.3","assets":[{"name":"keel","browser_download_url":"https://example.com/keel"}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		})

		release, err := fetchLatestRelease()
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if release.TagName != "v1.2.3" {
			t.Fatalf("unexpected tag: %q", release.TagName)
		}
		if len(release.Assets) != 1 || release.Assets[0].Name != "keel" {
			t.Fatalf("unexpected assets: %#v", release.Assets)
		}
	})

	t.Run("status error", func(t *testing.T) {
		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("boom")),
				Header:     make(http.Header),
			}, nil
		})

		_, err := fetchLatestRelease()
		if err == nil || !strings.Contains(err.Error(), "GitHub API responded 500") {
			t.Fatalf("expected status error, got %v", err)
		}
	})

	t.Run("decode error", func(t *testing.T) {
		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("{invalid-json")),
				Header:     make(http.Header),
			}, nil
		})

		_, err := fetchLatestRelease()
		if err == nil {
			t.Fatalf("expected decode error, got nil")
		}
	})
}

func TestFetchLatestVersion(t *testing.T) {
	stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"tag_name":"v9.9.9","assets":[]}`)),
			Header:     make(http.Header),
		}, nil
	})

	version, err := fetchLatestVersion()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if version != "v9.9.9" {
		t.Fatalf("expected v9.9.9, got %q", version)
	}
}

func TestDownloadBinary(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("binary-content")),
				Header:     make(http.Header),
			}, nil
		})

		path, err := downloadBinary("https://example.com/keel")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		t.Cleanup(func() { _ = os.Remove(path) })

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read downloaded binary: %v", err)
		}
		if string(content) != "binary-content" {
			t.Fatalf("unexpected downloaded content: %q", string(content))
		}
	})

	t.Run("request error", func(t *testing.T) {
		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})

		_, err := downloadBinary("https://example.com/keel")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestReplaceBinary(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir := t.TempDir()
		targetPath := filepath.Join(dir, "keel")
		newBinary := filepath.Join(dir, "keel-new")

		if err := os.WriteFile(targetPath, []byte("old-binary"), 0755); err != nil {
			t.Fatalf("failed to write old binary: %v", err)
		}
		if err := os.WriteFile(newBinary, []byte("new-binary"), 0644); err != nil {
			t.Fatalf("failed to write new binary: %v", err)
		}

		if err := replaceBinary(newBinary, targetPath); err != nil {
			t.Fatalf("replaceBinary returned error: %v", err)
		}

		content, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("failed to read replaced binary: %v", err)
		}
		if string(content) != "new-binary" {
			t.Fatalf("unexpected target content: %q", string(content))
		}
		if _, err := os.Stat(targetPath + ".bak"); !os.IsNotExist(err) {
			t.Fatalf("expected backup file to be removed")
		}
	})

	t.Run("missing target returns error", func(t *testing.T) {
		dir := t.TempDir()
		targetPath := filepath.Join(dir, "missing")
		newBinary := filepath.Join(dir, "keel-new")
		if err := os.WriteFile(newBinary, []byte("new-binary"), 0644); err != nil {
			t.Fatalf("failed to write new binary: %v", err)
		}

		err := replaceBinary(newBinary, targetPath)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestPathHelpersAndCheckState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if got, want := keelDir(), filepath.Join(home, ".keel"); got != want {
		t.Fatalf("expected keelDir %q, got %q", want, got)
	}
	if got, want := lastCheckFile(), filepath.Join(home, ".keel", "last_check"); got != want {
		t.Fatalf("expected lastCheckFile %q, got %q", want, got)
	}

	if !shouldCheck() {
		t.Fatalf("expected shouldCheck to be true when no cache exists")
	}

	if err := os.MkdirAll(keelDir(), 0755); err != nil {
		t.Fatalf("failed to create keelDir: %v", err)
	}

	if err := os.WriteFile(lastCheckFile(), []byte("not-a-time"), 0644); err != nil {
		t.Fatalf("failed to write invalid check state: %v", err)
	}
	if !shouldCheck() {
		t.Fatalf("expected shouldCheck to be true for invalid timestamp")
	}

	oldData, _ := time.Now().Add(-(checkInterval + time.Hour)).MarshalText()
	if err := os.WriteFile(lastCheckFile(), oldData, 0644); err != nil {
		t.Fatalf("failed to write old check state: %v", err)
	}
	if !shouldCheck() {
		t.Fatalf("expected shouldCheck to be true for old timestamp")
	}

	recentData, _ := time.Now().MarshalText()
	if err := os.WriteFile(lastCheckFile(), recentData, 0644); err != nil {
		t.Fatalf("failed to write recent check state: %v", err)
	}
	if shouldCheck() {
		t.Fatalf("expected shouldCheck to be false for recent timestamp")
	}

	saveLastCheck()
	if _, err := os.Stat(lastCheckFile()); err != nil {
		t.Fatalf("expected last_check to exist after saveLastCheck: %v", err)
	}
}

func TestBuildAssetName(t *testing.T) {
	got := buildAssetName()
	wantPrefix := "keel_" + runtime.GOOS + "_" + runtime.GOARCH
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("expected prefix %q, got %q", wantPrefix, got)
	}
	if runtime.GOOS == "windows" && !strings.HasSuffix(got, ".exe") {
		t.Fatalf("expected .exe suffix on windows")
	}
	if runtime.GOOS != "windows" && strings.HasSuffix(got, ".exe") {
		t.Fatalf("did not expect .exe suffix on non-windows")
	}
}

func TestIsNewer(t *testing.T) {
	if !isNewer("v2.0.0", "v1.0.0") {
		t.Fatalf("expected v2.0.0 to be newer than v1.0.0")
	}
	if isNewer("v1.0.0", "v1.0.0") {
		t.Fatalf("expected equal versions to not be newer")
	}
	if isNewer("1.0.0", "v1.0.0") {
		t.Fatalf("expected prefixed equal versions to not be newer")
	}
}

func TestCheckAndNotify(t *testing.T) {
	t.Run("skips check when recent timestamp exists", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		if err := os.MkdirAll(filepath.Join(home, ".keel"), 0755); err != nil {
			t.Fatalf("failed creating keel dir: %v", err)
		}
		now, _ := time.Now().MarshalText()
		if err := os.WriteFile(filepath.Join(home, ".keel", "last_check"), now, 0644); err != nil {
			t.Fatalf("failed writing last_check: %v", err)
		}

		msg := readUpdateMessage(t, CheckAndNotify("v1.0.0"))
		if msg != "" {
			t.Fatalf("expected empty message, got %q", msg)
		}
	})

	t.Run("notifies when a new version exists", func(t *testing.T) {
		resetUpgradeDeps(t)
		home := t.TempDir()
		t.Setenv("HOME", home)
		executablePathFn = func() (string, error) {
			return "/opt/homebrew/bin/keel", nil
		}
		evalSymlinksFn = func(path string) (string, error) {
			return "/opt/homebrew/Cellar/keel/1.0.0/bin/keel", nil
		}
		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"tag_name":"v9.9.9","assets":[]}`)),
				Header:     make(http.Header),
			}, nil
		})

		msg := readUpdateMessage(t, CheckAndNotify("v1.0.0"))
		if !strings.Contains(msg, "New version available: v9.9.9") {
			t.Fatalf("expected update message, got %q", msg)
		}
		if !strings.Contains(msg, "Update with: brew upgrade slice-soft/tap/keel") {
			t.Fatalf("expected Homebrew update command, got %q", msg)
		}
		if _, err := os.Stat(filepath.Join(home, ".keel", "last_check")); err != nil {
			t.Fatalf("expected saveLastCheck to write last_check: %v", err)
		}
	})

	t.Run("returns empty when already on latest", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		stubHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"tag_name":"v1.0.0","assets":[]}`)),
				Header:     make(http.Header),
			}, nil
		})

		msg := readUpdateMessage(t, CheckAndNotify("v1.0.0"))
		if msg != "" {
			t.Fatalf("expected empty message, got %q", msg)
		}
	})
}

func TestUpgrade(t *testing.T) {
	t.Run("runs external command for Homebrew installs", func(t *testing.T) {
		resetUpgradeDeps(t)
		executablePathFn = func() (string, error) {
			return "/opt/homebrew/bin/keel", nil
		}
		evalSymlinksFn = func(path string) (string, error) {
			return "/opt/homebrew/Cellar/keel/1.0.0/bin/keel", nil
		}
		var ranCommand string
		runUpdateCommandFn = func(command string) error {
			ranCommand = command
			return nil
		}

		if err := Upgrade("v1.0.0"); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if ranCommand != HomebrewUpdateCommand {
			t.Fatalf("expected Homebrew update command, got %q", ranCommand)
		}
	})

	t.Run("runs external command for go install", func(t *testing.T) {
		resetUpgradeDeps(t)
		binDir := t.TempDir()
		getenvFn = func(key string) string {
			if key == "GOBIN" {
				return binDir
			}
			return ""
		}
		executablePathFn = func() (string, error) {
			return filepath.Join(binDir, keelExecutableName()), nil
		}
		evalSymlinksFn = func(path string) (string, error) {
			return path, nil
		}
		var ranCommand string
		runUpdateCommandFn = func(command string) error {
			ranCommand = command
			return nil
		}

		if err := Upgrade("v1.0.0"); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if ranCommand != GoInstallUpdateCommand {
			t.Fatalf("expected go install update command, got %q", ranCommand)
		}
	})

	t.Run("returns error when external command fails", func(t *testing.T) {
		resetUpgradeDeps(t)
		binDir := t.TempDir()
		getenvFn = func(key string) string {
			if key == "GOBIN" {
				return binDir
			}
			return ""
		}
		executablePathFn = func() (string, error) {
			return filepath.Join(binDir, keelExecutableName()), nil
		}
		evalSymlinksFn = func(path string) (string, error) {
			return path, nil
		}
		runUpdateCommandFn = func(command string) error {
			return errors.New("command failed")
		}

		err := Upgrade("v1.0.0")
		if err == nil || !strings.Contains(err.Error(), "error running") {
			t.Fatalf("expected command error, got %v", err)
		}
	})

	t.Run("prints manual update message for unknown installs", func(t *testing.T) {
		resetUpgradeDeps(t)
		fetchLatestReleaseFn = func() (*Release, error) {
			t.Fatal("unknown installs must not query GitHub releases")
			return nil, nil
		}
		runUpdateCommandFn = func(command string) error {
			t.Fatal("unknown installs must not run update commands")
			return nil
		}
		executablePathFn = func() (string, error) {
			return filepath.Join(t.TempDir(), keelExecutableName()), nil
		}
		evalSymlinksFn = func(path string) (string, error) {
			return path, nil
		}

		if err := Upgrade("v1.0.0"); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})
}
