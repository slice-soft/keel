package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"time"
)

const endpoint = "https://telemetry.keel-go.dev/event"

type cliEvent struct {
	Event     string `json:"event"`
	Command   string `json:"command"`
	Version   string `json:"version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	InstallID string `json:"install_id"`
	GoVersion string `json:"go_version"`
}

// Send fires a cli_run ping in a background goroutine — never blocks, never returns errors.
// On first run it creates ~/.keel/config.json and prints a one-time notice.
func Send(command, version string) {
	if os.Getenv("KEEL_TELEMETRY") == "off" {
		return
	}

	cfg, exists, err := readConfig()
	if err != nil {
		return
	}
	if !exists {
		cfg, err = initConfig()
		if err != nil {
			return
		}
	}
	if !cfg.Telemetry {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		payload, _ := json.Marshal(cliEvent{
			Event:     "cli_run",
			Command:   command,
			Version:   version,
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
			InstallID: cfg.InstallID,
			GoVersion: runtime.Version(),
		})

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return
		}
		resp.Body.Close()
	}()
}

// IsEnabled reports whether telemetry is currently active.
// Returns true by default if ~/.keel/config.json doesn't exist yet.
func IsEnabled() bool {
	if os.Getenv("KEEL_TELEMETRY") == "off" {
		return false
	}
	cfg, exists, _ := readConfig()
	if !exists {
		return true
	}
	return cfg.Telemetry
}

// Enable turns on telemetry and saves the setting to ~/.keel/config.json.
func Enable() error {
	cfg, _, _ := readConfig()
	cfg.Telemetry = true
	if cfg.InstallID == "" {
		cfg.InstallID = newUUID()
	}
	return writeConfig(cfg)
}

// Disable turns off telemetry and saves the setting to ~/.keel/config.json.
func Disable() error {
	cfg, _, _ := readConfig()
	cfg.Telemetry = false
	return writeConfig(cfg)
}
