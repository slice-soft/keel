package appproperties

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const DefaultPath = "application.properties"

// EnvVar describes one environment placeholder referenced from
// application.properties.
type EnvVar struct {
	Key        string
	Default    string
	HasDefault bool
}

// Document contains the environment placeholders discovered in a properties
// file, preserving declaration order.
type Document struct {
	EnvVars []EnvVar
}

// Load reads and parses application.properties.
func Load(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Document{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return Parse(data)
}

// Parse extracts environment placeholders from raw application.properties
// content.
func Parse(data []byte) (*Document, error) {
	doc := &Document{}
	seen := map[string]struct{}{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}

		separator := strings.IndexAny(line, "=:")
		if separator == -1 {
			return nil, fmt.Errorf("line %d: expected key=value", lineNumber)
		}

		value := strings.TrimSpace(line[separator+1:])
		for _, envVar := range extractEnvVars(value) {
			if _, ok := seen[envVar.Key]; ok {
				continue
			}
			seen[envVar.Key] = struct{}{}
			doc.EnvVars = append(doc.EnvVars, envVar)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return doc, nil
}

func extractEnvVars(value string) []EnvVar {
	var envVars []EnvVar

	for {
		start := strings.Index(value, "${")
		if start == -1 {
			return envVars
		}

		value = value[start+2:]
		end := strings.Index(value, "}")
		if end == -1 {
			return envVars
		}

		token := value[:end]
		value = value[end+1:]

		hasDefault := strings.Contains(token, ":")
		parts := strings.SplitN(token, ":", 2)
		key := strings.TrimSpace(parts[0])
		if key == "" || !looksLikeEnvKey(key) {
			continue
		}

		envVar := EnvVar{Key: key, HasDefault: hasDefault}
		if hasDefault {
			envVar.Default = parts[1]
		}
		envVars = append(envVars, envVar)
	}
}

func looksLikeEnvKey(key string) bool {
	for _, r := range key {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return false
		}
	}
	return true
}
