package generator

import "testing"

func TestGetLatestModuleVersionInvalidModule(t *testing.T) {
	tests := []struct {
		name      string
		module    string
		wantError bool
	}{
		{name: "invalid module path", module: "%%%", wantError: true},
		{name: "valid module path", module: "github.com/slice-soft/ss-keel-core", wantError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getLatestModuleVersion(tt.module)
			if tt.wantError && err == nil {
				t.Fatalf("expected error for module %q", tt.module)
			}
			if !tt.wantError && err != nil {
				t.Fatalf("unexpected error for module %q: %v", tt.module, err)
			}
		})
	}
}
