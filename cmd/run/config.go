package run

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func ensureKeelConfigExists(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("keel.toml not found in current directory")
		}
		return fmt.Errorf("failed to access %s: %w", path, err)
	}
	return nil
}

func loadScriptsFromConfig(path string) (map[string]string, error) {
	cfg := viper.New()
	cfg.SetConfigFile(path)

	if err := cfg.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read keel.toml: %w", err)
	}

	scripts := cfg.GetStringMapString("scripts")
	if len(scripts) == 0 {
		return nil, fmt.Errorf("no scripts defined in keel.toml")
	}

	return scripts, nil
}

func findScriptCommand(scripts map[string]string, scriptName string) (string, error) {
	scriptCmd, ok := scripts[scriptName]
	if !ok || scriptCmd == "" {
		return "", fmt.Errorf("script '%s' does not exist in keel.toml", scriptName)
	}
	return scriptCmd, nil
}
