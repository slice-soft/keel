package new

import (
	"fmt"
	"strings"
)

func validateProjectName(value string) error {
	name := strings.TrimSpace(value)
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if strings.ContainsAny(name, " \t\n\r") {
		return fmt.Errorf("project name cannot contain spaces")
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("project name must not contain '/' or '\\'")
	}
	return nil
}

func validateModulePath(value string, allowLocal bool) error {
	module := strings.TrimSpace(value)
	if module == "" {
		return fmt.Errorf("module path cannot be empty")
	}
	if strings.ContainsAny(module, "\\ \t\n\r") {
		return fmt.Errorf("module path cannot contain spaces or '\\'")
	}
	if strings.HasPrefix(module, "/") || strings.HasSuffix(module, "/") {
		return fmt.Errorf("module path cannot start or end with '/'")
	}
	if !allowLocal && !strings.Contains(module, "/") {
		return fmt.Errorf("module path must include a domain or namespace (e.g. github.com/user/app)")
	}
	return nil
}

func validateNonEmpty(fieldName string) func(string) error {
	return func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s cannot be empty", fieldName)
		}
		return nil
	}
}

func validateCustomDomain(value string) error {
	domain := strings.TrimSpace(value)
	if domain == "" {
		return fmt.Errorf("custom domain cannot be empty")
	}
	if strings.Contains(domain, "://") {
		return fmt.Errorf("custom domain must not include protocol")
	}
	if strings.ContainsAny(domain, "\\ \t\n\r") {
		return fmt.Errorf("custom domain cannot contain spaces or '\\'")
	}
	return nil
}
