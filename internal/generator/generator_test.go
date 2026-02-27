package generator
package generator

import (
	"testing"
)

func TestToKebab(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"PascalCase", "UserProfile", "user-profile"},
		{"camelCase", "userName", "user-name"},
		{"lowercase", "user", "user"},
		{"snake_case", "user_name", "user-name"},
		{"Already kebab", "user-name", "user-name"},
		{"Mixed", "UserName_Profile", "user-name-profile"},
	}






































































































































}	}		})			}				t.Errorf("SnakeName = %q, want %q", data.SnakeName, tt.expectedSnake)			if data.SnakeName != tt.expectedSnake {			}				t.Errorf("KebabName = %q, want %q", data.KebabName, tt.expectedKebab)			if data.KebabName != tt.expectedKebab {			}				t.Errorf("CamelName = %q, want %q", data.CamelName, tt.expectedCamel)			if data.CamelName != tt.expectedCamel {			}				t.Errorf("PascalName = %q, want %q", data.PascalName, tt.expectedPascal)			if data.PascalName != tt.expectedPascal {			data := NewData(tt.input)		t.Run(tt.name, func(t *testing.T) {	for _, tt := range tests {	}		},			expectedSnake:  "user",			expectedKebab:  "user",			expectedCamel:  "user",			expectedPascal: "User",			input:         "user",			name:          "single word",		{		},			expectedSnake:  "user_profile",			expectedKebab:  "user-profile",			expectedCamel:  "userProfile",			expectedPascal: "UserProfile",			input:         "user-profile",			name:          "kebab-case input",		{		},			expectedSnake:  "user_profile",			expectedKebab:  "user-profile",			expectedCamel:  "userProfile",			expectedPascal: "UserProfile",			input:         "UserProfile",			name:          "PascalCase input",		{	}{		expectedSnake  string		expectedKebab  string		expectedCamel  string		expectedPascal string		input         string		name          string	tests := []struct {func TestNewData(t *testing.T) {}	}		})			}				t.Errorf("toSnake(%q) = %q, want %q", tt.input, result, tt.expected)			if result != tt.expected {			result := toSnake(tt.input)		t.Run(tt.name, func(t *testing.T) {	for _, tt := range tests {	}		{"Already snake_case", "user_name", "user_name"},		{"lowercase", "user", "user"},		{"kebab-case", "user-name", "user_name"},		{"camelCase", "userName", "user_name"},		{"PascalCase", "UserProfile", "user_profile"},	}{		expected string		input    string		name     string	tests := []struct {func TestToSnake(t *testing.T) {}	}		})			}				t.Errorf("toCamel(%q) = %q, want %q", tt.input, result, tt.expected)			if result != tt.expected {			result := toCamel(tt.input)		t.Run(tt.name, func(t *testing.T) {	for _, tt := range tests {	}		{"Already camelCase", "userName", "userName"},		{"lowercase", "user", "user"},		{"PascalCase", "UserName", "userName"},		{"snake_case", "user_name", "userName"},		{"kebab-case", "user-profile", "userProfile"},	}{		expected string		input    string		name     string	tests := []struct {func TestToCamel(t *testing.T) {}	}		})			}				t.Errorf("toPascal(%q) = %q, want %q", tt.input, result, tt.expected)			if result != tt.expected {			result := toPascal(tt.input)		t.Run(tt.name, func(t *testing.T) {	for _, tt := range tests {	}		{"Already PascalCase", "UserProfile", "UserProfile"},		{"lowercase", "user", "User"},		{"camelCase", "userName", "UserName"},		{"snake_case", "user_name", "UserName"},		{"kebab-case", "user-profile", "UserProfile"},	}{		expected string		input    string		name     string	tests := []struct {func TestToPascal(t *testing.T) {}	}		})			}				t.Errorf("toKebab(%q) = %q, want %q", tt.input, result, tt.expected)			if result != tt.expected {			result := toKebab(tt.input)		t.Run(tt.name, func(t *testing.T) {	for _, tt := range tests {