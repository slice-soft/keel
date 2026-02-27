package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/slice-soft/keel/internal/generator"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new [app-name]",
	Short: "Create a new Keel project",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runNew,
}

var moduleFlag string

func init() {
	newCmd.Flags().StringVar(&moduleFlag, "module", "", "Go module name (e.g. github.com/user/my-app)")
}

func runNew(cmd *cobra.Command, args []string) error {
	appName := ""
	if len(args) > 0 {
		appName = args[0]
	}

	fmt.Println()
	fmt.Println("  ⚓  Welcome to Keel!")
	fmt.Println()

	// Step 1: ask for project name if not provided
	if appName == "" {
		nameForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Where should we create your project?").
					Placeholder("my-app").
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("project name cannot be empty")
						}
						return nil
					}).
					Value(&appName),
			),
		).WithTheme(huh.ThemeCharm())

		if err := nameForm.Run(); err != nil {
			return err
		}
	}

	if _, err := os.Stat(appName); err == nil {
		return fmt.Errorf("directory '%s' already exists", appName)
	}

	// Step 2: ask for module name, git and deps
	moduleName := moduleFlag
	if moduleName == "" {
		moduleName = appName
	}

	initGit := true
	installDeps := true

	configFields := []huh.Field{
		huh.NewInput().
			Title("Go module name?").
			Placeholder("github.com/user/" + appName).
			Value(&moduleName),
		huh.NewConfirm().
			Title("Initialize a new git repository?").
			Value(&initGit),
		huh.NewConfirm().
			Title("Install dependencies?").
			Description("Runs go mod tidy").
			Value(&installDeps),
	}

	// Skip module question if --module flag was explicitly provided
	if moduleFlag != "" {
		configFields = configFields[1:]
	}

	configForm := huh.NewForm(
		huh.NewGroup(configFields...),
	).WithTheme(huh.ThemeCharm())

	if err := configForm.Run(); err != nil {
		return err
	}

	// Generate files
	data := generator.NewProjectData(appName, moduleName)

	fmt.Printf("\n  Creating project files...\n\n")

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

	// Post-step: git init
	if initGit {
		fmt.Println()
		gitCmd := exec.Command("git", "init", appName)
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr
		if err := gitCmd.Run(); err != nil {
			fmt.Printf("  ⚠  git init failed: %v\n", err)
		} else {
			fmt.Println("  ✓ Git repository initialized")
		}
	}

	// Post-step: go mod tidy
	if installDeps {
		fmt.Println()
		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = appName
		tidyCmd.Stdout = os.Stdout
		tidyCmd.Stderr = os.Stderr
		if err := tidyCmd.Run(); err != nil {
			fmt.Printf("  ⚠  go mod tidy failed: %v\n", err)
		} else {
			fmt.Println("  ✓ Dependencies installed")
		}
	}

	fmt.Printf(`
  ✅ Project '%s' ready!

  Next steps:
    cd %s
    keel run dev

`, appName, appName)

	return nil
}
