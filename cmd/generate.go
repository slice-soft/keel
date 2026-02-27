package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/slice-soft/keel/internal/generator"
	"github.com/spf13/cobra"
)

var (
	withCrud     bool
	noService    bool
	noRepository bool
	withTests    bool
	yesFlag      bool
)

var generateCmd = &cobra.Command{
	Use:     "generate <type> [name]",
	Aliases: []string{"g"},
	Short:   "Generate Keel files",
	Long: `Generate files based on the specified type.

Available types:
  module      Full module (controller + service + repository)
  controller  Standalone controller (no service dependency)
  service     Standalone service (no repository dependency)
  repository  Standalone repository
  middleware  Generic middleware
  guard       Access guard (implements core.Guard)
  dto         DTOs (create, update, response)
  crud        Full module with CRUD and DTOs
  scheduler   Job scheduler registrar
  job         Standalone job with its handler
  checker     Health checker (implements core.HealthChecker)
  event       Publisher + Subscriber for a domain
  hook        Shutdown hook (compatible with app.OnShutdown)

Examples:
  keel generate module users
  keel g module users --with-crud
  keel g module users --no-service
  keel g module users --no-repository
  keel g controller products
  keel g service orders
  keel g guard admin
  keel g crud orders
  keel g scheduler reports
  keel g job cleanup
  keel g checker redis
  keel g event orders
  keel g hook database`,
}

func init() {
	// Global flag: skip all interactive prompts and use defaults
	generateCmd.PersistentFlags().BoolVarP(&yesFlag, "yes", "y", false, "Skip all interactive prompts and use defaults")

	// Module structure flags
	genModuleCmd.Flags().BoolVar(&withCrud, "with-crud", false, "Generate full CRUD with DTOs")
	genModuleCmd.Flags().BoolVar(&noService, "no-service", false, "Controller only (omit service and repository)")
	genModuleCmd.Flags().BoolVar(&noRepository, "no-repository", false, "Omit repository (service + controller only)")
	genModuleCmd.MarkFlagsMutuallyExclusive("with-crud", "no-service")
	genModuleCmd.MarkFlagsMutuallyExclusive("with-crud", "no-repository")
	genModuleCmd.MarkFlagsMutuallyExclusive("no-service", "no-repository")
	genModuleCmd.Flags().BoolVar(&withTests, "with-tests", false, "Generate test skeleton files")

	// Test flag on other generators
	genCrudCmd.Flags().BoolVar(&withTests, "with-tests", false, "Generate test skeleton files")
	genControllerCmd.Flags().BoolVar(&withTests, "with-tests", false, "Generate test skeleton files")
	genServiceCmd.Flags().BoolVar(&withTests, "with-tests", false, "Generate test skeleton files")
	genRepositoryCmd.Flags().BoolVar(&withTests, "with-tests", false, "Generate test skeleton files")

	generateCmd.AddCommand(genModuleCmd)
	generateCmd.AddCommand(genControllerCmd)
	generateCmd.AddCommand(genServiceCmd)
	generateCmd.AddCommand(genRepositoryCmd)
	generateCmd.AddCommand(genMiddlewareCmd)
	generateCmd.AddCommand(genGuardCmd)
	generateCmd.AddCommand(genDtoCmd)
	generateCmd.AddCommand(genCrudCmd)
	generateCmd.AddCommand(genSchedulerCmd)
	generateCmd.AddCommand(genJobCmd)
	generateCmd.AddCommand(genCheckerCmd)
	generateCmd.AddCommand(genEventCmd)
	generateCmd.AddCommand(genHookCmd)
}

// — generate module —

var genModuleCmd = &cobra.Command{
	Use:   "module [name]",
	Short: "Generate a full module",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}

		// Prompt for name if not provided
		if name == "" && !yesFlag {
			if err := promptName("Module name?", "users", &name); err != nil {
				return err
			}
		}
		if name == "" {
			return fmt.Errorf("module name is required")
		}

		// Determine structure
		structMode := "full"
		structFromFlag := withCrud || noService || noRepository
		if structFromFlag {
			switch {
			case withCrud:
				structMode = "with-crud"
			case noService:
				structMode = "no-service"
			case noRepository:
				structMode = "no-repository"
			}
		} else if !yesFlag {
			if err := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("Module structure?").
						Options(
							huh.NewOption("Controller + Service + Repository", "full"),
							huh.NewOption("Controller + Service", "no-repository"),
							huh.NewOption("Controller only", "no-service"),
							huh.NewOption("Full CRUD with DTOs", "with-crud"),
						).
						Value(&structMode),
				),
			).WithTheme(keelTheme).Run(); err != nil {
				return err
			}
		}

		// Prompt for tests (unless --with-tests or --yes)
		genTests := withTests
		if !cmd.Flags().Changed("with-tests") && !yesFlag {
			if err := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Generate test skeletons?").
						Value(&genTests),
				),
			).WithTheme(keelTheme).Run(); err != nil {
				return err
			}
		}

		data := generator.NewData(name)
		data.ModuleName = generator.ReadModuleName()
		base := filepath.Join("internal", "modules", data.PackageName)

		var files []struct{ tmpl, dest string }

		switch structMode {
		case "no-service":
			files = []struct{ tmpl, dest string }{
				{"templates/module/module_ctrl_only.go.tmpl", filepath.Join(base, data.PackageName+".module.go")},
				{"templates/controller/controller.go.tmpl", filepath.Join(base, data.PackageName+".controller.go")},
			}
			if genTests {
				files = append(files, struct{ tmpl, dest string }{
					"templates/module/controller_test.go.tmpl",
					filepath.Join(base, data.PackageName+".controller_test.go"),
				})
			}

		case "no-repository":
			files = []struct{ tmpl, dest string }{
				{"templates/module/module_no_repo.go.tmpl", filepath.Join(base, data.PackageName+".module.go")},
				{"templates/module/controller.go.tmpl", filepath.Join(base, data.PackageName+".controller.go")},
				{"templates/service/service.go.tmpl", filepath.Join(base, data.PackageName+".service.go")},
			}
			if genTests {
				files = append(files,
					struct{ tmpl, dest string }{"templates/module/service_test.go.tmpl", filepath.Join(base, data.PackageName+".service_test.go")},
					struct{ tmpl, dest string }{"templates/module/controller_test.go.tmpl", filepath.Join(base, data.PackageName+".controller_test.go")},
				)
			}

		case "with-crud":
			files = []struct{ tmpl, dest string }{
				{"templates/module/module.go.tmpl", filepath.Join(base, data.PackageName+".module.go")},
				{"templates/module/controller_crud.go.tmpl", filepath.Join(base, data.PackageName+".controller.go")},
				{"templates/module/service.go.tmpl", filepath.Join(base, data.PackageName+".service.go")},
				{"templates/module/repository.go.tmpl", filepath.Join(base, data.PackageName+".repository.go")},
				{"templates/dto/dto.go.tmpl", filepath.Join(base, "dto", data.SnakeName+".dto.go")},
			}
			if genTests {
				files = append(files,
					struct{ tmpl, dest string }{"templates/module/service_test.go.tmpl", filepath.Join(base, data.PackageName+".service_test.go")},
					struct{ tmpl, dest string }{"templates/module/controller_test.go.tmpl", filepath.Join(base, data.PackageName+".controller_test.go")},
					struct{ tmpl, dest string }{"templates/module/repository_test.go.tmpl", filepath.Join(base, data.PackageName+".repository_test.go")},
				)
			}

		default: // full
			files = []struct{ tmpl, dest string }{
				{"templates/module/module.go.tmpl", filepath.Join(base, data.PackageName+".module.go")},
				{"templates/module/controller.go.tmpl", filepath.Join(base, data.PackageName+".controller.go")},
				{"templates/module/service.go.tmpl", filepath.Join(base, data.PackageName+".service.go")},
				{"templates/module/repository.go.tmpl", filepath.Join(base, data.PackageName+".repository.go")},
			}
			if genTests {
				files = append(files,
					struct{ tmpl, dest string }{"templates/module/service_test.go.tmpl", filepath.Join(base, data.PackageName+".service_test.go")},
					struct{ tmpl, dest string }{"templates/module/controller_test.go.tmpl", filepath.Join(base, data.PackageName+".controller_test.go")},
					struct{ tmpl, dest string }{"templates/module/repository_test.go.tmpl", filepath.Join(base, data.PackageName+".repository_test.go")},
				)
			}
		}

		fmt.Printf("\n⚓  Generating module: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate controller —

var genControllerCmd = &cobra.Command{
	Use:   "controller [name]",
	Short: "Generate a standalone controller",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := resolveName(args, "Controller name?", "users")
		if err != nil {
			return err
		}

		genTests := withTests
		if !cmd.Flags().Changed("with-tests") && !yesFlag {
			if err := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Generate test skeletons?").
						Value(&genTests),
				),
			).WithTheme(keelTheme).Run(); err != nil {
				return err
			}
		}

		data := generator.NewData(name)
		base := filepath.Join("internal", "modules", data.PackageName)

		files := []struct{ tmpl, dest string }{
			{"templates/controller/controller.go.tmpl", filepath.Join(base, data.PackageName+".controller.go")},
		}
		if genTests {
			files = append(files, struct{ tmpl, dest string }{
				"templates/module/controller_test.go.tmpl",
				filepath.Join(base, data.PackageName+".controller_test.go"),
			})
		}

		fmt.Printf("\n⚓  Generating controller: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate service —

var genServiceCmd = &cobra.Command{
	Use:   "service [name]",
	Short: "Generate a standalone service",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := resolveName(args, "Service name?", "users")
		if err != nil {
			return err
		}

		genTests := withTests
		if !cmd.Flags().Changed("with-tests") && !yesFlag {
			if err := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Generate test skeletons?").
						Value(&genTests),
				),
			).WithTheme(keelTheme).Run(); err != nil {
				return err
			}
		}

		data := generator.NewData(name)
		base := filepath.Join("internal", "modules", data.PackageName)

		files := []struct{ tmpl, dest string }{
			{"templates/service/service.go.tmpl", filepath.Join(base, data.PackageName+".service.go")},
		}
		if genTests {
			files = append(files, struct{ tmpl, dest string }{
				"templates/module/service_test.go.tmpl",
				filepath.Join(base, data.PackageName+".service_test.go"),
			})
		}

		fmt.Printf("\n⚓  Generating service: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate repository —

var genRepositoryCmd = &cobra.Command{
	Use:   "repository [name]",
	Short: "Generate a repository",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := resolveName(args, "Repository name?", "users")
		if err != nil {
			return err
		}

		genTests := withTests
		if !cmd.Flags().Changed("with-tests") && !yesFlag {
			if err := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Generate test skeletons?").
						Value(&genTests),
				),
			).WithTheme(keelTheme).Run(); err != nil {
				return err
			}
		}

		data := generator.NewData(name)
		base := filepath.Join("internal", "modules", data.PackageName)

		files := []struct{ tmpl, dest string }{
			{"templates/module/repository.go.tmpl", filepath.Join(base, data.PackageName+".repository.go")},
		}
		if genTests {
			files = append(files, struct{ tmpl, dest string }{
				"templates/module/repository_test.go.tmpl",
				filepath.Join(base, data.PackageName+".repository_test.go"),
			})
		}

		fmt.Printf("\n⚓  Generating repository: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate middleware —

var genMiddlewareCmd = &cobra.Command{
	Use:   "middleware [name]",
	Short: "Generate a middleware",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := resolveName(args, "Middleware name?", "auth")
		if err != nil {
			return err
		}
		data := generator.NewData(name)
		dest := filepath.Join("internal", "middleware", data.SnakeName+".middleware.go")

		files := []struct{ tmpl, dest string }{
			{"templates/middleware/middleware.go.tmpl", dest},
		}

		fmt.Printf("\n⚓  Generating middleware: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate guard —

var genGuardCmd = &cobra.Command{
	Use:   "guard [name]",
	Short: "Generate an access guard",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := resolveName(args, "Guard name?", "admin")
		if err != nil {
			return err
		}
		data := generator.NewData(name)
		dest := filepath.Join("internal", "guards", data.SnakeName+".guard.go")

		files := []struct{ tmpl, dest string }{
			{"templates/guard/guard.go.tmpl", dest},
		}

		fmt.Printf("\n⚓  Generating guard: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate dto —

var genDtoCmd = &cobra.Command{
	Use:   "dto [name]",
	Short: "Generate standalone DTOs",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := resolveName(args, "DTO name?", "users")
		if err != nil {
			return err
		}
		data := generator.NewData(name)
		dest := filepath.Join("internal", "modules", data.PackageName, "dto", data.SnakeName+".dto.go")

		files := []struct{ tmpl, dest string }{
			{"templates/dto/dto.go.tmpl", dest},
		}

		fmt.Printf("\n⚓  Generating DTOs: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate crud —

var genCrudCmd = &cobra.Command{
	Use:   "crud [name]",
	Short: "Generate a full module with CRUD and DTOs",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := resolveName(args, "Module name?", "users")
		if err != nil {
			return err
		}

		genTests := withTests
		if !cmd.Flags().Changed("with-tests") && !yesFlag {
			if err := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Generate test skeletons?").
						Value(&genTests),
				),
			).WithTheme(keelTheme).Run(); err != nil {
				return err
			}
		}

		data := generator.NewData(name)
		data.ModuleName = generator.ReadModuleName()
		base := filepath.Join("internal", "modules", data.PackageName)

		files := []struct{ tmpl, dest string }{
			{"templates/module/module.go.tmpl", filepath.Join(base, data.PackageName+".module.go")},
			{"templates/module/controller_crud.go.tmpl", filepath.Join(base, data.PackageName+".controller.go")},
			{"templates/module/service.go.tmpl", filepath.Join(base, data.PackageName+".service.go")},
			{"templates/module/repository.go.tmpl", filepath.Join(base, data.PackageName+".repository.go")},
			{"templates/dto/dto.go.tmpl", filepath.Join(base, "dto", data.SnakeName+".dto.go")},
		}
		if genTests {
			files = append(files,
				struct{ tmpl, dest string }{"templates/module/service_test.go.tmpl", filepath.Join(base, data.PackageName+".service_test.go")},
				struct{ tmpl, dest string }{"templates/module/controller_test.go.tmpl", filepath.Join(base, data.PackageName+".controller_test.go")},
				struct{ tmpl, dest string }{"templates/module/repository_test.go.tmpl", filepath.Join(base, data.PackageName+".repository_test.go")},
			)
		}

		fmt.Printf("\n⚓  Generating full CRUD: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate scheduler —

var genSchedulerCmd = &cobra.Command{
	Use:   "scheduler [name]",
	Short: "Generate a job scheduler registrar",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := resolveName(args, "Scheduler name?", "reports")
		if err != nil {
			return err
		}
		data := generator.NewData(name)
		dest := filepath.Join("internal", "scheduler", data.SnakeName+".scheduler.go")

		files := []struct{ tmpl, dest string }{
			{"templates/scheduler/scheduler.go.tmpl", dest},
		}

		fmt.Printf("\n⚓  Generating scheduler: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate job —

var genJobCmd = &cobra.Command{
	Use:   "job [name]",
	Short: "Generate a standalone job with its handler",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := resolveName(args, "Job name?", "cleanup")
		if err != nil {
			return err
		}
		data := generator.NewData(name)
		dest := filepath.Join("internal", "scheduler", data.SnakeName+".job.go")

		files := []struct{ tmpl, dest string }{
			{"templates/scheduler/job.go.tmpl", dest},
		}

		fmt.Printf("\n⚓  Generating job: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate checker —

var genCheckerCmd = &cobra.Command{
	Use:   "checker [name]",
	Short: "Generate a health checker",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := resolveName(args, "Checker name?", "redis")
		if err != nil {
			return err
		}
		data := generator.NewData(name)
		dest := filepath.Join("internal", "checkers", data.SnakeName+".checker.go")

		files := []struct{ tmpl, dest string }{
			{"templates/checker/checker.go.tmpl", dest},
		}

		fmt.Printf("\n⚓  Generating checker: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate event —

var genEventCmd = &cobra.Command{
	Use:   "event [name]",
	Short: "Generate publisher + subscriber for a domain",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := resolveName(args, "Event name?", "orders")
		if err != nil {
			return err
		}
		data := generator.NewData(name)
		base := filepath.Join("internal", "events")

		files := []struct{ tmpl, dest string }{
			{"templates/event/publisher.go.tmpl", filepath.Join(base, data.SnakeName+".publisher.go")},
			{"templates/event/subscriber.go.tmpl", filepath.Join(base, data.SnakeName+".subscriber.go")},
		}

		fmt.Printf("\n⚓  Generating event: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate hook —

var genHookCmd = &cobra.Command{
	Use:   "hook [name]",
	Short: "Generate a shutdown hook",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := resolveName(args, "Hook name?", "database")
		if err != nil {
			return err
		}
		data := generator.NewData(name)
		dest := filepath.Join("internal", "hooks", data.SnakeName+".hook.go")

		files := []struct{ tmpl, dest string }{
			{"templates/hook/hook.go.tmpl", dest},
		}

		fmt.Printf("\n⚓  Generating hook: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — helper —

func renderFiles(files []struct{ tmpl, dest string }, data generator.Data) error {
	for _, f := range files {
		if generator.FileExists(f.dest) {
			fmt.Printf("  ⚠  already exists: %s (skipped)\n", f.dest)
			continue
		}
		if err := generator.RenderToFile(f.tmpl, f.dest, data); err != nil {
			return err
		}
		fmt.Printf("  ✓  %s\n", f.dest)
	}
	fmt.Println("\n  ✅ Done")
	return nil
}
