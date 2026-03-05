package completion

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNewCommandIncludesShellGenerators(t *testing.T) {
	root := &cobra.Command{Use: "keel"}
	cmd := NewCommand(root)

	required := []string{"bash", "fish", "powershell", "zsh", "install"}
	for _, name := range required {
		if _, _, err := cmd.Find([]string{name}); err != nil {
			t.Fatalf("expected subcommand %q to exist: %v", name, err)
		}
	}
}
