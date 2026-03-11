package generate

import "testing"

func TestNameConverters(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantKebab   string
		wantSnake   string
		wantPackage string
		wantPascal  string
		wantCamel   string
	}{
		{
			name:        "dash",
			input:       "my-service",
			wantKebab:   "my-service",
			wantSnake:   "my_service",
			wantPackage: "myservice",
			wantPascal:  "MyService",
			wantCamel:   "myService",
		},
		{
			name:        "underscore",
			input:       "my_service",
			wantKebab:   "my-service",
			wantSnake:   "my_service",
			wantPackage: "myservice",
			wantPascal:  "MyService",
			wantCamel:   "myService",
		},
		{
			name:        "space and upper",
			input:       "My Service",
			wantKebab:   "my-service",
			wantSnake:   "my_service",
			wantPackage: "myservice",
			wantPascal:  "MyService",
			wantCamel:   "myService",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toKebab(tt.input); got != tt.wantKebab {
				t.Fatalf("toKebab: expected %q, got %q", tt.wantKebab, got)
			}
			if got := toSnake(tt.input); got != tt.wantSnake {
				t.Fatalf("toSnake: expected %q, got %q", tt.wantSnake, got)
			}
			if got := toPackage(tt.input); got != tt.wantPackage {
				t.Fatalf("toPackage: expected %q, got %q", tt.wantPackage, got)
			}
			if got := toPascal(tt.input); got != tt.wantPascal {
				t.Fatalf("toPascal: expected %q, got %q", tt.wantPascal, got)
			}
			if got := toCamel(tt.wantPascal); got != tt.wantCamel {
				t.Fatalf("toCamel: expected %q, got %q", tt.wantCamel, got)
			}
		})
	}
}

func TestToCamelEmpty(t *testing.T) {
	if got := toCamel(""); got != "" {
		t.Fatalf("expected empty camel case for empty input, got %q", got)
	}
}
