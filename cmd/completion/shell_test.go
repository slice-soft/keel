package completion

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDetectShellFromEnv(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "/bin/zsh", want: "zsh"},
		{in: "/usr/local/bin/bash", want: "bash"},
		{in: "/opt/homebrew/bin/fish", want: "fish"},
		{in: "", want: ""},
		{in: "/bin/sh", want: ""},
	}

	for _, tt := range tests {
		if got := detectShellFromEnv(tt.in); got != tt.want {
			t.Fatalf("detectShellFromEnv(%q): expected %q, got %q", tt.in, tt.want, got)
		}
	}
}

func TestMergeShellOptions(t *testing.T) {
	got := mergeShellOptions("zsh", []string{"bash", "zsh", "fish"})
	want := []string{"zsh", "bash", "fish"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestResolveConfigFile(t *testing.T) {
	home := t.TempDir()

	zshrc := filepath.Join(home, ".zshrc")
	if got, err := resolveConfigFile("zsh", home); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if got != zshrc {
		t.Fatalf("expected default zsh config %q, got %q", zshrc, got)
	}
}
