package new

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/slice-soft/keel/internal/generator"
	"github.com/spf13/cobra"
)

var withoutStarterModule bool
var withFolderStructure bool

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "new [project-name]",
		Aliases: []string{"n"},
		Short:   "Create a new Keel project",
		Args:    cobra.MaximumNArgs(1),
		RunE:    runNew,
	}

	cmd.Flags().BoolVar(
		&withoutStarterModule,
		"without-starter-module",
		false,
		"Skip creating the default 'starter' module (for advanced users)",
	)

	cmd.Flags().BoolVar(
		&withFolderStructure,
		"with-folder-structure",
		false,
		"Use a more opinionated folder structure with separate directories for middleware, guards, scheduler, checkers, events, and hooks (instead of a flat 'internal' directory)",
	)
	return cmd
}

type projectFile struct {
	tmpl string
	dest string
}

type projectSetup struct {
	appName              string
	moduleName           string
	useAir               bool
	includeAirConfig     bool
	useEnv               bool
	initGit              bool
	installDeps          bool
	withoutStarterModule bool
	withFolderStructure  bool
}

var keelTheme = huh.ThemeCharm()

func runNew(cmd *cobra.Command, args []string) error {
	printWelcome()

	setup, err := collectProjectSetup(args)
	if err != nil {
		return err
	}

	if err := scaffoldProject(setup); err != nil {
		return err
	}

	runPostSetup(setup)
	printProjectReady(setup.appName)
	return nil
}

func printWelcome() {
	fmt.Println()
	fmt.Println("Welcome to Keel!")
	fmt.Println()
}

func collectProjectSetup(args []string) (projectSetup, error) {
	initialAppName, err := resolveProjectName(args)
	if err != nil {
		return projectSetup{}, err
	}

	moduleName, err := promptModulePath(initialAppName)
	if err != nil {
		return projectSetup{}, err
	}

	appName, err := projectNameFromModule(moduleName)
	if err != nil {
		return projectSetup{}, err
	}

	if _, err := os.Stat(appName); err == nil {
		return projectSetup{}, fmt.Errorf("directory '%s' already exists", appName)
	}

	useAir, includeAirConfig, err := promptAirSetup()
	if err != nil {
		return projectSetup{}, err
	}

	useEnv, err := promptYesNo("Include .env support?", true)
	if err != nil {
		return projectSetup{}, err
	}

	initGit, err := promptYesNo("Initialize a new git repository?", true)
	if err != nil {
		return projectSetup{}, err
	}

	installDeps, err := promptYesNo("Install dependencies?", true)
	if err != nil {
		return projectSetup{}, err
	}

	return projectSetup{
		appName:              appName,
		moduleName:           moduleName,
		useAir:               useAir,
		includeAirConfig:     includeAirConfig,
		useEnv:               useEnv,
		initGit:              initGit,
		installDeps:          installDeps,
		withoutStarterModule: withoutStarterModule,
		withFolderStructure:  withFolderStructure,
	}, nil
}

func projectNameFromModule(moduleName string) (string, error) {
	parts := strings.Split(moduleName, "/")
	lastPart := parts[len(parts)-1]
	return resolveProjectName([]string{lastPart})
}

func scaffoldProject(setup projectSetup) error {
	useStarterModule := !setup.withoutStarterModule
	useFolderStructure := setup.withFolderStructure

	data := generator.NewProjectData(
		setup.appName,
		setup.moduleName,
		setup.useAir,
		setup.includeAirConfig,
		setup.useEnv,
		useStarterModule,
		useFolderStructure,
	)

	fmt.Printf("\n  Creating project files...\n\n")
	if err := renderProjectFiles(setup.appName, setup.includeAirConfig, setup.useEnv, useStarterModule, data); err != nil {
		return err
	}

	return createProjectDirectories(setup.appName, setup.withFolderStructure, setup.withoutStarterModule)
}

func renderProjectFiles(appName string, includeAirConfig, useEnv, includeStarterModule bool, data generator.Data) error {
	files := buildProjectFiles(appName, includeAirConfig, useEnv, includeStarterModule)
	for _, f := range files {
		if err := generator.RenderToFile(f.tmpl, f.dest, data); err != nil {
			return fmt.Errorf("error generating %s: %w", f.dest, err)
		}
		fmt.Printf("  ✓ %s\n", f.dest)
	}
	return nil
}

func createProjectDirectories(appName string, withFolderStructure, withoutStarterModule bool) error {
	if !withFolderStructure {
		return nil
	}

	dirs := []string{
		filepath.Join(appName, "internal", "middleware"),
		filepath.Join(appName, "internal", "guards"),
		filepath.Join(appName, "internal", "scheduler"),
		filepath.Join(appName, "internal", "checkers"),
		filepath.Join(appName, "internal", "events"),
		filepath.Join(appName, "internal", "hooks"),
	}

	if withoutStarterModule {
		dirs = append(dirs, filepath.Join(appName, "internal", "modules"))
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
		fmt.Printf("  ✓ %s/\n", d)
	}

	return nil
}

func runPostSetup(setup projectSetup) {
	if setup.initGit {
		fmt.Println()
		gitCmd := exec.Command("git", "init", setup.appName)
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr
		if err := gitCmd.Run(); err != nil {
			fmt.Printf("  ⚠  git init failed: %v\n", err)
		} else {
			fmt.Println("  ✓ Git repository initialized")
		}
	}

	if setup.installDeps {
		fmt.Println()
		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = setup.appName
		tidyCmd.Stdout = os.Stdout
		tidyCmd.Stderr = os.Stderr
		if err := tidyCmd.Run(); err != nil {
			fmt.Printf("  ⚠  go mod tidy failed: %v\n", err)
		} else {
			fmt.Println("  ✓ Dependencies installed")
		}
	}
}

func printProjectReady(appName string) {
	fmt.Printf(`
  ✅ Project '%s' ready!

  Next steps:
    cd %s
    keel run dev

`, appName, appName)
}

func resolveProjectName(args []string) (string, error) {
	if len(args) > 0 {
		if err := validateProjectName(args[0]); err != nil {
			return "", err
		}
		return strings.TrimSpace(args[0]), nil
	}

	appName := ""
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project name?").
				Placeholder("my-backend").
				Validate(validateProjectName).
				Value(&appName),
		),
	).WithTheme(keelTheme).Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(appName), nil
}

func validateProjectName(value string) error {
	name := strings.TrimSpace(value)
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if strings.ContainsAny(name, " \t\n\r") {
		return fmt.Errorf("project name cannot contain spaces")
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("project name must not contain '/' or '\\'")
	}
	return nil
}

func promptModulePath(appName string) (string, error) {
	host := "github"
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Where will this module be hosted?").
				Options(
					huh.NewOption("GitHub", "github"),
					huh.NewOption("GitLab", "gitlab"),
					huh.NewOption("Custom domain", "custom"),
					huh.NewOption("Local module", "local"),
				).
				Value(&host),
		),
	).WithTheme(keelTheme).Run(); err != nil {
		return "", err
	}

	var preview string
	var allowLocal bool

	switch host {
	case "github":
		owner := ""
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("GitHub username or organization?").
					Placeholder("slice-soft").
					Validate(validateNonEmpty("GitHub username or organization")).
					Value(&owner),
			),
		).WithTheme(keelTheme).Run(); err != nil {
			return "", err
		}
		preview = fmt.Sprintf("github.com/%s/%s", strings.TrimSpace(owner), appName)

	case "gitlab":
		owner := ""
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("GitLab username or group?").
					Placeholder("slice-soft").
					Validate(validateNonEmpty("GitLab username or group")).
					Value(&owner),
			),
		).WithTheme(keelTheme).Run(); err != nil {
			return "", err
		}
		preview = fmt.Sprintf("gitlab.com/%s/%s", strings.TrimSpace(owner), appName)

	case "custom":
		domain := ""
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Custom domain?").
					Placeholder("code.example.com").
					Validate(validateCustomDomain).
					Value(&domain),
			),
		).WithTheme(keelTheme).Run(); err != nil {
			return "", err
		}
		cleanDomain := strings.Trim(strings.TrimSpace(domain), "/")
		preview = fmt.Sprintf("%s/%s", cleanDomain, appName)

	case "local":
		allowLocal = true
		preview = appName

	default:
		return "", fmt.Errorf("unsupported host option: %s", host)
	}

	return confirmOrEditModulePath(preview, allowLocal)
}

func confirmOrEditModulePath(preview string, allowLocal bool) (string, error) {
	fmt.Println()
	fmt.Println("Module path preview")
	fmt.Println(preview)
	fmt.Println()

	usePreview := true
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Use this module path?").
				Affirmative("Yes").
				Negative("Edit").
				Value(&usePreview),
		),
	).WithTheme(keelTheme).Run(); err != nil {
		return "", err
	}

	if usePreview {
		return preview, nil
	}

	modulePath := preview
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Module path?").
				Placeholder(preview).
				Validate(func(s string) error {
					return validateModulePath(s, allowLocal)
				}).
				Value(&modulePath),
		),
	).WithTheme(keelTheme).Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(modulePath), nil
}

func validateModulePath(value string, allowLocal bool) error {
	module := strings.TrimSpace(value)
	if module == "" {
		return fmt.Errorf("module path cannot be empty")
	}
	if strings.ContainsAny(module, "\\ \t\n\r") {
		return fmt.Errorf("module path cannot contain spaces or '\\'")
	}
	if strings.HasPrefix(module, "/") || strings.HasSuffix(module, "/") {
		return fmt.Errorf("module path cannot start or end with '/'")
	}
	if !allowLocal && !strings.Contains(module, "/") {
		return fmt.Errorf("module path must include a domain or namespace (e.g. github.com/user/app)")
	}
	return nil
}

func validateNonEmpty(fieldName string) func(string) error {
	return func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s cannot be empty", fieldName)
		}
		return nil
	}
}

func validateCustomDomain(value string) error {
	domain := strings.TrimSpace(value)
	if domain == "" {
		return fmt.Errorf("custom domain cannot be empty")
	}
	if strings.Contains(domain, "://") {
		return fmt.Errorf("custom domain must not include protocol")
	}
	if strings.ContainsAny(domain, "\\ \t\n\r") {
		return fmt.Errorf("custom domain cannot contain spaces or '\\'")
	}
	return nil
}

func promptYesNo(title string, defaultValue bool) (bool, error) {
	choice := defaultValue
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Value(&choice).
				Affirmative("yes").
				Negative("No"),
		),
	).WithTheme(keelTheme).Run(); err != nil {
		return false, err
	}
	return choice, nil
}

func promptAirSetup() (bool, bool, error) {
	useAir, err := promptYesNo("Use Air for hot reload?", true)
	if err != nil {
		return false, false, err
	}
	if !useAir {
		return false, false, nil
	}

	includeAirConfig, err := promptYesNo("Include Air config file (.air.toml)?", true)
	if err != nil {
		return false, false, err
	}

	fmt.Println()
	if airInstalled() {
		fmt.Println("  ✓ Air is already installed")
		return true, includeAirConfig, nil
	}

	fmt.Println("  ⚠  Air is not installed on your PATH.")
	installAir, err := promptYesNo("Install Air now? (go install github.com/air-verse/air@latest)", true)
	if err != nil {
		return false, false, err
	}
	if !installAir {
		fmt.Println("  ⚠  Skipping Air installation")
		return true, includeAirConfig, nil
	}

	fmt.Println()
	fmt.Println("  Installing Air...")
	if err := installAirBinary(); err != nil {
		fmt.Printf("  ⚠  failed to install Air: %v\n", err)
		return true, includeAirConfig, nil
	}

	if airInstalled() {
		fmt.Println("  ✓ Air installed")
		return true, includeAirConfig, nil
	}

	fmt.Println("  ✓ Air installed (restart your shell if 'air' is not available yet)")
	return true, includeAirConfig, nil
}

func airInstalled() bool {
	_, err := exec.LookPath("air")
	return err == nil
}

func installAirBinary() error {
	installCmd := exec.Command("go", "install", "github.com/air-verse/air@latest")
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	return installCmd.Run()
}

func buildProjectFiles(appName string, includeAirConfig, useEnv, includeStarterModule bool) []projectFile {
	files := []projectFile{
		{tmpl: "templates/project/main.go.tmpl", dest: filepath.Join(appName, "cmd", "main.go")},
		{tmpl: "templates/project/go.mod.tmpl", dest: filepath.Join(appName, "go.mod")},
		{tmpl: "templates/project/keel.toml.tmpl", dest: filepath.Join(appName, "keel.toml")},
		{tmpl: "templates/project/readme.tmpl", dest: filepath.Join(appName, "README.md")},
		{tmpl: "templates/project/gitignore.tmpl", dest: filepath.Join(appName, ".gitignore")},
	}

	if useEnv {
		files = append(files, projectFile{
			tmpl: "templates/project/env.tmpl",
			dest: filepath.Join(appName, ".env"),
		})
	}

	if includeAirConfig {
		files = append(files, projectFile{
			tmpl: "templates/project/air.toml.tmpl",
			dest: filepath.Join(appName, ".air.toml"),
		})
	}

	if includeStarterModule {
		modulesPath := "internal/modules"
		filesModule := []projectFile{
			{tmpl: "templates/module/starter_module.go.tmpl", dest: filepath.Join(appName, modulesPath, "starter", "module.go")},
			{tmpl: "templates/module/starter_service.go.tmpl", dest: filepath.Join(appName, modulesPath, "starter", "service.go")},
			{tmpl: "templates/module/starter_controller.go.tmpl", dest: filepath.Join(appName, modulesPath, "starter", "controller.go")},
			{tmpl: "templates/module/starter_dto.go.tmpl", dest: filepath.Join(appName, modulesPath, "starter", "dto.go")},
		}
		files = append(files, filesModule...)
	}

	return files
}
