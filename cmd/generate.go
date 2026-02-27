package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/slice-soft/ss-keel-cli/internal/generator"
	"github.com/spf13/cobra"
)

var (
	withCrud     bool
	noService    bool
	noRepository bool
)

var generateCmd = &cobra.Command{
	Use:     "generate <type> <name>",
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
	genModuleCmd.Flags().BoolVar(&withCrud, "with-crud", false, "Generate full CRUD with DTOs")
	genModuleCmd.Flags().BoolVar(&noService, "no-service", false, "Controller only (omit service and repository)")
	genModuleCmd.Flags().BoolVar(&noRepository, "no-repository", false, "Omit repository (service + controller only)")
	genModuleCmd.MarkFlagsMutuallyExclusive("with-crud", "no-service")
	genModuleCmd.MarkFlagsMutuallyExclusive("with-crud", "no-repository")
	genModuleCmd.MarkFlagsMutuallyExclusive("no-service", "no-repository")

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
	Use:   "module <name>",
	Short: "Generate a full module",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
		data.ModuleName = generator.ReadModuleName()
		base := filepath.Join("internal", "modules", data.PackageName)

		var files []struct{ tmpl, dest string }

		switch {
		case noService:
			// Controller only — no service, no repository
			files = []struct{ tmpl, dest string }{
				{"templates/module/module_ctrl_only.go.tmpl", filepath.Join(base, data.PackageName+".module.go")},
				{"templates/controller/controller.go.tmpl", filepath.Join(base, data.PackageName+".controller.go")},
			}

		case noRepository:
			// Controller + standalone service — no repository
			files = []struct{ tmpl, dest string }{
				{"templates/module/module_no_repo.go.tmpl", filepath.Join(base, data.PackageName+".module.go")},
				{"templates/module/controller.go.tmpl", filepath.Join(base, data.PackageName+".controller.go")},
				{"templates/service/service.go.tmpl", filepath.Join(base, data.PackageName+".service.go")},
			}

		case withCrud:
			// Full CRUD with DTOs
			files = []struct{ tmpl, dest string }{
				{"templates/module/module.go.tmpl", filepath.Join(base, data.PackageName+".module.go")},
				{"templates/module/controller_crud.go.tmpl", filepath.Join(base, data.PackageName+".controller.go")},
				{"templates/module/service.go.tmpl", filepath.Join(base, data.PackageName+".service.go")},
				{"templates/module/repository.go.tmpl", filepath.Join(base, data.PackageName+".repository.go")},
				{"templates/dto/dto.go.tmpl", filepath.Join(base, "dto", data.SnakeName+".dto.go")},
			}

		default:
			// Full module: controller + service + repository
			files = []struct{ tmpl, dest string }{
				{"templates/module/module.go.tmpl", filepath.Join(base, data.PackageName+".module.go")},
				{"templates/module/controller.go.tmpl", filepath.Join(base, data.PackageName+".controller.go")},
				{"templates/module/service.go.tmpl", filepath.Join(base, data.PackageName+".service.go")},
				{"templates/module/repository.go.tmpl", filepath.Join(base, data.PackageName+".repository.go")},
			}
		}

		fmt.Printf("\n⚓  Generating module: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate controller —

var genControllerCmd = &cobra.Command{
	Use:   "controller <name>",
	Short: "Generate a standalone controller",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
		base := filepath.Join("internal", "modules", data.PackageName)

		files := []struct{ tmpl, dest string }{
			{"templates/controller/controller.go.tmpl", filepath.Join(base, data.PackageName+".controller.go")},
		}

		fmt.Printf("\n⚓  Generating controller: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate service —

var genServiceCmd = &cobra.Command{
	Use:   "service <name>",
	Short: "Generate a standalone service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
		base := filepath.Join("internal", "modules", data.PackageName)

		files := []struct{ tmpl, dest string }{
			{"templates/service/service.go.tmpl", filepath.Join(base, data.PackageName+".service.go")},
		}

		fmt.Printf("\n⚓  Generating service: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate repository —

var genRepositoryCmd = &cobra.Command{
	Use:   "repository <name>",
	Short: "Generate a repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
		base := filepath.Join("internal", "modules", data.PackageName)

		files := []struct{ tmpl, dest string }{
			{"templates/module/repository.go.tmpl", filepath.Join(base, data.PackageName+".repository.go")},
		}

		fmt.Printf("\n⚓  Generating repository: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate middleware —

var genMiddlewareCmd = &cobra.Command{
	Use:   "middleware <name>",
	Short: "Generate a middleware",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
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
	Use:   "guard <name>",
	Short: "Generate an access guard",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
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
	Use:   "dto <name>",
	Short: "Generate standalone DTOs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
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
	Use:   "crud <name>",
	Short: "Generate a full module with CRUD and DTOs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
		data.ModuleName = generator.ReadModuleName()
		base := filepath.Join("internal", "modules", data.PackageName)

		files := []struct{ tmpl, dest string }{
			{"templates/module/module.go.tmpl", filepath.Join(base, data.PackageName+".module.go")},
			{"templates/module/controller_crud.go.tmpl", filepath.Join(base, data.PackageName+".controller.go")},
			{"templates/module/service.go.tmpl", filepath.Join(base, data.PackageName+".service.go")},
			{"templates/module/repository.go.tmpl", filepath.Join(base, data.PackageName+".repository.go")},
			{"templates/dto/dto.go.tmpl", filepath.Join(base, "dto", data.SnakeName+".dto.go")},
		}

		fmt.Printf("\n⚓  Generating full CRUD: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate scheduler —

var genSchedulerCmd = &cobra.Command{
	Use:   "scheduler <name>",
	Short: "Generate a job scheduler registrar",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
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
	Use:   "job <name>",
	Short: "Generate a standalone job with its handler",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
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
	Use:   "checker <name>",
	Short: "Generate a health checker",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
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
	Use:   "event <name>",
	Short: "Generate publisher + subscriber for a domain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
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
	Use:   "hook <name>",
	Short: "Generate a shutdown hook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
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
	fmt.Println("\n  ✅ Done\n")
	return nil
}
