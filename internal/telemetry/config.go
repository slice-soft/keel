package telemetry

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type keelConfig struct {
	Telemetry bool   `json:"telemetry"`
	InstallID string `json:"install_id"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".keel", "config.json"), nil
}

// readConfig reads ~/.keel/config.json.
// Returns (config, true, nil) on success, (zero, false, nil) if file doesn't exist.
func readConfig() (keelConfig, bool, error) {
	path, err := configPath()
	if err != nil {
		return keelConfig{}, false, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return keelConfig{}, false, nil
	}
	if err != nil {
		return keelConfig{}, true, err
	}
	var cfg keelConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return keelConfig{}, true, err
	}
	return cfg, true, nil
}

func writeConfig(cfg keelConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// initConfig creates ~/.keel/config.json on first run with telemetry enabled.
// Prints a one-time notice to stderr.
func initConfig() (keelConfig, error) {
	cfg := keelConfig{Telemetry: true, InstallID: newUUID()}
	if err := writeConfig(cfg); err != nil {
		return cfg, err
	}
	fmt.Fprintln(os.Stderr, "\nKeel collects anonymous usage data (command name, version, OS/arch).")
	fmt.Fprintln(os.Stderr, "No personal data, file paths, or project names are ever sent.")
	fmt.Fprintln(os.Stderr, "  Opt out: keel telemetry disable  or  KEEL_TELEMETRY=off")
	fmt.Fprintln(os.Stderr)
	return cfg, nil
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant RFC 4122
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
