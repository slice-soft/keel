package generate

import "testing"

func TestNewData(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantPackage string
		wantPascal  string
		wantCamel   string
		wantKebab   string
		wantSnake   string
	}{
		{
			name:        "simple",
			input:       "users",
			wantPackage: "users",
			wantPascal:  "Users",
			wantCamel:   "users",
			wantKebab:   "users",
			wantSnake:   "users",
		},
		{
			name:        "kebab case",
			input:       "user-profile",
			wantPackage: "userprofile",
			wantPascal:  "UserProfile",
			wantCamel:   "userProfile",
			wantKebab:   "user-profile",
			wantSnake:   "user_profile",
		},
		{
			name:        "mixed separators",
			input:       "user_profile service",
			wantPackage: "userprofileservice",
			wantPascal:  "UserProfileService",
			wantCamel:   "userProfileService",
			wantKebab:   "user-profile-service",
			wantSnake:   "user_profile_service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewData(tt.input)
			if got.PackageName != tt.wantPackage {
				t.Fatalf("PackageName: expected %q, got %q", tt.wantPackage, got.PackageName)
			}
			if got.PascalName != tt.wantPascal {
				t.Fatalf("PascalName: expected %q, got %q", tt.wantPascal, got.PascalName)
			}
			if got.CamelName != tt.wantCamel {
				t.Fatalf("CamelName: expected %q, got %q", tt.wantCamel, got.CamelName)
			}
			if got.KebabName != tt.wantKebab {
				t.Fatalf("KebabName: expected %q, got %q", tt.wantKebab, got.KebabName)
			}
			if got.SnakeName != tt.wantSnake {
				t.Fatalf("SnakeName: expected %q, got %q", tt.wantSnake, got.SnakeName)
			}
		})
	}
}

func TestNewProjectDataSetsFlags(t *testing.T) {
	data := NewProjectData("my-backend", "github.com/slice-soft/my-backend", true, false, true, true, false)

	if data.AppName != "my-backend" {
		t.Fatalf("expected AppName to be my-backend, got %q", data.AppName)
	}
	if data.ModuleName != "github.com/slice-soft/my-backend" {
		t.Fatalf("unexpected ModuleName: %q", data.ModuleName)
	}
	if !data.UseAir {
		t.Fatalf("expected UseAir to be true")
	}
	if data.UseAirConfig {
		t.Fatalf("expected UseAirConfig to be false")
	}
	if !data.UseEnv {
		t.Fatalf("expected UseEnv to be true")
	}
	if !data.UseStarterModule {
		t.Fatalf("expected UseStarterModule to be true")
	}
	if data.TemplateMode != "new" {
		t.Fatalf("expected TemplateMode to be new, got %q", data.TemplateMode)
	}
}

func TestNewInitData(t *testing.T) {
	data := NewInitData("my-backend", true, true)

	if data.AppName != "my-backend" {
		t.Fatalf("expected AppName to be my-backend, got %q", data.AppName)
	}
	if data.TemplateMode != "init" {
		t.Fatalf("expected TemplateMode to be init, got %q", data.TemplateMode)
	}
	if !data.UseAir {
		t.Fatalf("expected UseAir to be true")
	}
	if !data.UseAirConfig {
		t.Fatalf("expected UseAirConfig to be true")
	}
	if data.UseEnv {
		t.Fatalf("expected UseEnv to be false")
	}
}

func TestNewInitDataWithoutAirConfig(t *testing.T) {
	data := NewInitData("my-backend", true, false)
	if data.UseAirConfig {
		t.Fatalf("expected UseAirConfig to be false")
	}
}
