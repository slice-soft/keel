package generate

import (
	"bufio"
	"os"
	"strings"

	"github.com/slice-soft/keel/internal/appproperties"
	generator "github.com/slice-soft/keel/internal/generator/generate"
)

const databaseEngineProperty = "database.engine"
const databaseEngineEnvKey = "DATABASE_ENGINE"

// resolveDBEngine reports which database engine the project is configured for,
// so the generated DDL uses that engine's column types. It follows the same
// precedence the running app does: .env wins over the default declared in
// application.properties, and sqlite is assumed when neither says anything.
func resolveDBEngine() string {
	if engine := engineFromEnvFile(".env"); engine != "" {
		return engine
	}
	if engine := engineFromProperties(appproperties.DefaultPath); engine != "" {
		return engine
	}
	return generator.DefaultDBEngine
}

func engineFromEnvFile(path string) string {
	return scanKeyValue(path, func(key string) bool { return key == databaseEngineEnvKey })
}

// engineFromProperties reads database.engine, resolving the ${VAR:default}
// placeholder form to its default when the property uses one.
func engineFromProperties(path string) string {
	raw := scanKeyValue(path, func(key string) bool { return key == databaseEngineProperty })
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "${") || !strings.HasSuffix(raw, "}") {
		return raw
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(raw, "${"), "}")
	if _, def, found := strings.Cut(inner, ":"); found {
		return strings.TrimSpace(def)
	}
	return ""
}

// scanKeyValue returns the value of the first KEY=VALUE line whose key matches,
// skipping blanks and comments. Both .env and application.properties use this
// shape, so one scanner serves both.
func scanKeyValue(path string, matches func(key string) bool) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || !matches(strings.TrimSpace(key)) {
			continue
		}
		return strings.TrimSpace(value)
	}
	return ""
}
