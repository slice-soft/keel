package generate

import (
	"fmt"
	"regexp"
	"strings"
)

var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-_]*(/[a-zA-Z0-9][a-zA-Z0-9-_]*)?$`)

type parsedName struct {
	moduleName    string
	componentName string
	standalone    bool
}

func parseName(raw string) (parsedName, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return parsedName{}, fmt.Errorf("name is required")
	}
	if !namePattern.MatchString(name) {
		return parsedName{}, fmt.Errorf("invalid name: %s", raw)
	}

	parts := strings.Split(name, "/")
	if len(parts) == 1 {
		return parsedName{componentName: parts[0], standalone: true}, nil
	}

	return parsedName{moduleName: parts[0], componentName: parts[1]}, nil
}
