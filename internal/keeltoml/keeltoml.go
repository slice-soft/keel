package keeltoml

import (
	"fmt"
	"os"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

const DefaultPath = "keel.toml"

// KeelMeta mirrors the [keel] section.
type KeelMeta struct {
	Version string `toml:"version,omitempty"`
}

// AddonEntry is one [[addons]] entry.
type AddonEntry struct {
	ID           string   `toml:"id"`
	Version      string   `toml:"version,omitempty"`
	Repo         string   `toml:"repo,omitempty"`
	Capabilities []string `toml:"capabilities,omitempty"`
	Resources    []string `toml:"resources,omitempty"`
}

// EnvEntry is one [[env]] entry.
type EnvEntry struct {
	Key         string `toml:"key"`
	Source      string `toml:"source,omitempty"`
	Required    bool   `toml:"required"`
	Secret      bool   `toml:"secret,omitempty"`
	Default     string `toml:"default,omitempty"`
	Description string `toml:"description,omitempty"`
}

// KeelToml holds only the sections this package cares about.
// Unknown sections (app, scripts, features) are silently ignored on decode.
type KeelToml struct {
	Keel   KeelMeta     `toml:"keel"`
	Addons []AddonEntry `toml:"addons"`
	Env    []EnvEntry   `toml:"env"`
}

// Load reads and parses keel.toml from path.
// If the file does not exist, returns an empty KeelToml with no error.
func Load(path string) (*KeelToml, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &KeelToml{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return parse(data, path)
}

// Parse decodes raw TOML bytes into a KeelToml.
// Exported for tests; prefer Load for file-based usage.
func Parse(data []byte) (*KeelToml, error) {
	return parse(data, "<input>")
}

func parse(data []byte, name string) (*KeelToml, error) {
	var kt KeelToml
	if err := toml.Unmarshal(data, &kt); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", name, err)
	}
	return &kt, nil
}

// MergeAddon appends new [[addons]] and [[env]] entries to path.
// Existing entries with the same ID / key are not duplicated.
// Returns true when the file was changed, false when all entries already exist.
// If path does not exist the file is created.
func MergeAddon(path, id, version, repo string, caps, resources []string, envVars []EnvEntry) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("reading %s: %w", path, err)
	}

	// Partial decode — only addons and env, ignore rest.
	var existing KeelToml
	if len(data) > 0 {
		if err := toml.Unmarshal(data, &existing); err != nil {
			return false, fmt.Errorf("parsing %s: %w", path, err)
		}
	}

	var buf strings.Builder

	// [[addons]] entry — only if addon not already registered.
	if !addonExists(existing.Addons, id) {
		fmt.Fprintf(&buf, "\n[[addons]]\n")
		fmt.Fprintf(&buf, "id           = %q\n", id)
		if version != "" {
			fmt.Fprintf(&buf, "version      = %q\n", version)
		}
		if repo != "" {
			fmt.Fprintf(&buf, "repo         = %q\n", repo)
		}
		if len(caps) > 0 {
			fmt.Fprintf(&buf, "capabilities = [%s]\n", quotedList(caps))
		}
		if len(resources) > 0 {
			fmt.Fprintf(&buf, "resources    = [%s]\n", quotedList(resources))
		}
	}

	// [[env]] entries — only keys not already declared.
	for _, ev := range envVars {
		if envKeyExists(existing.Env, ev.Key) {
			continue
		}
		fmt.Fprintf(&buf, "\n[[env]]\n")
		fmt.Fprintf(&buf, "key      = %q\n", ev.Key)
		if ev.Source != "" {
			fmt.Fprintf(&buf, "source   = %q\n", ev.Source)
		}
		fmt.Fprintf(&buf, "required = %v\n", ev.Required)
		if ev.Secret {
			fmt.Fprintf(&buf, "secret   = %v\n", ev.Secret)
		}
		if ev.Default != "" {
			fmt.Fprintf(&buf, "default  = %q\n", ev.Default)
		}
		if ev.Description != "" {
			fmt.Fprintf(&buf, "description = %q\n", ev.Description)
		}
	}

	toAppend := buf.String()
	if toAppend == "" {
		return false, nil
	}

	// Ensure the existing content ends with a newline before appending.
	prefix := ""
	if len(data) > 0 && data[len(data)-1] != '\n' {
		prefix = "\n"
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	if _, err := f.WriteString(prefix + toAppend); err != nil {
		return false, fmt.Errorf("writing %s: %w", path, err)
	}
	return true, nil
}

// LookupEnvValue finds the value of key in a .env file content string.
// Returns ("", false) when the key is not present or commented out.
func LookupEnvValue(envContent, key string) (string, bool) {
	for _, line := range strings.Split(envContent, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(trimmed, key+"=") {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(trimmed, key+"=")), true
	}
	return "", false
}

// addonExists reports whether an addon with the given id is in the slice.
func addonExists(addons []AddonEntry, id string) bool {
	for _, a := range addons {
		if a.ID == id {
			return true
		}
	}
	return false
}

// envKeyExists reports whether a key is already in the env slice.
func envKeyExists(envs []EnvEntry, key string) bool {
	for _, e := range envs {
		if e.Key == key {
			return true
		}
	}
	return false
}

// quotedList returns items as a TOML inline array of quoted strings.
// e.g. ["database", "cache"]
func quotedList(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(quoted, ", ")
}
