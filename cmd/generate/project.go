package generate

import (
	"errors"
	"os"
)

const invalidProjectMessage = "keel generate must be executed inside a Keel project"

func validateKeelProject() error {
	required := []string{"go.mod", "cmd/main.go", "internal"}
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			return errors.New(invalidProjectMessage)
		}
	}
	return nil
}
