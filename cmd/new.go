package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/slice-soft/ss-keel-cli/internal/generator"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <app-name>",
	Short: "Create a new Keel project",
	Args:  cobra.ExactArgs(1),
	RunE:  runNew,
}

var moduleFlag string

func init() {
	newCmd.Flags().StringVar(&moduleFlag, "module", "", "Go module name (e.g. github.com/user/my-app)")
}

func runNew(cmd *cobra.Command, args []string) error {
	appName := args[0]

	moduleName := moduleFlag
	if moduleName == "" {
		moduleName = appName
	}

	data := generator.NewProjectData(appName, moduleName)

	if _, err := os.Stat(appName); err == nil {
		return fmt.Errorf("directory '%s' already exists", appName)
	}

	fmt.Printf("\n⚓  Creating Keel project: %s\n\n", appName)

	files := []struct {
		tmpl string
		dest string
	}{
		{"templates/project/main.go.tmpl", filepath.Join(appName, "cmd", "main.go")},
		{"templates/project/go.mod.tmpl", filepath.Join(appName, "go.mod")},
		{"templates/project/keel.toml.tmpl", filepath.Join(appName, "keel.toml")},
		{"templates/project/air.toml.tmpl", filepath.Join(appName, ".air.toml")},
		{"templates/project/env.tmpl", filepath.Join(appName, ".env")},
	}

	for _, f := range files {
		if err := generator.RenderToFile(f.tmpl, f.dest, data); err != nil {
			return fmt.Errorf("error generating %s: %w", f.dest, err)
		}
		fmt.Printf("  ✓ %s\n", f.dest)
	}

	dirs := []string{
		filepath.Join(appName, "internal", "modules"),
		filepath.Join(appName, "internal", "middleware"),
		filepath.Join(appName, "internal", "guards"),
		filepath.Join(appName, "internal", "scheduler"),
		filepath.Join(appName, "internal", "checkers"),
		filepath.Join(appName, "internal", "events"),
		filepath.Join(appName, "internal", "hooks"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
		fmt.Printf("  ✓ %s/\n", d)
	}

	fmt.Printf(`
  ✅ Project '%s' created successfully

  Next steps:
    cd %s
    keel run dev

`, appName, appName)

	return nil
}
