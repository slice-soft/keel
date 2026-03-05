package generator

import "testing"

func TestGetLatestModuleVersionInvalidModule(t *testing.T) {
	_, err := getLatestModuleVersion("%%%")
	if err == nil {
		t.Fatalf("expected error for invalid module path")
	}
}
