package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/slice-soft/ss-keel-cli/internal/generator"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:     "generate <type> <name>",
	Aliases: []string{"g"},
	Short:   "Genera archivos de Keel",
	Long: `Genera archivos según el tipo especificado.

Tipos disponibles:
  module      Módulo completo (controller + service + repository)
  controller  Solo el controller
  service     Solo el service
  repository  Solo el repository
  middleware  Middleware genérico
  guard       Guard de acceso/autorización
  dto         DTOs (create, update, response)
  crud        Módulo completo con CRUD y DTOs

Ejemplos:
  keel generate module users
  keel g module users
  keel g controller products
  keel g guard admin
  keel g crud orders`,
}

func init() {
	generateCmd.AddCommand(genModuleCmd)
	generateCmd.AddCommand(genControllerCmd)
	generateCmd.AddCommand(genServiceCmd)
	generateCmd.AddCommand(genRepositoryCmd)
	generateCmd.AddCommand(genMiddlewareCmd)
	generateCmd.AddCommand(genGuardCmd)
	generateCmd.AddCommand(genDtoCmd)
	generateCmd.AddCommand(genCrudCmd)
}

// — generate module —

var genModuleCmd = &cobra.Command{
	Use:   "module <name>",
	Short: "Genera un módulo completo",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
		base := filepath.Join("internal", "modules", data.PackageName)

		files := []struct{ tmpl, dest string }{
			{"templates/module/module.go.tmpl", filepath.Join(base, data.PackageName+".module.go")},
			{"templates/module/controller.go.tmpl", filepath.Join(base, data.PackageName+".controller.go")},
			{"templates/module/service.go.tmpl", filepath.Join(base, data.PackageName+".service.go")},
			{"templates/module/repository.go.tmpl", filepath.Join(base, data.PackageName+".repository.go")},
		}

		fmt.Printf("\n⚓  Generando módulo: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate controller —

var genControllerCmd = &cobra.Command{
	Use:   "controller <name>",
	Short: "Genera un controller",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
		base := filepath.Join("internal", "modules", data.PackageName)

		files := []struct{ tmpl, dest string }{
			{"templates/module/controller.go.tmpl", filepath.Join(base, data.PackageName+".controller.go")},
		}

		fmt.Printf("\n⚓  Generando controller: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate service —

var genServiceCmd = &cobra.Command{
	Use:   "service <name>",
	Short: "Genera un service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
		base := filepath.Join("internal", "modules", data.PackageName)

		files := []struct{ tmpl, dest string }{
			{"templates/module/service.go.tmpl", filepath.Join(base, data.PackageName+".service.go")},
		}

		fmt.Printf("\n⚓  Generando service: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate repository —

var genRepositoryCmd = &cobra.Command{
	Use:   "repository <name>",
	Short: "Genera un repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
		base := filepath.Join("internal", "modules", data.PackageName)

		files := []struct{ tmpl, dest string }{
			{"templates/module/repository.go.tmpl", filepath.Join(base, data.PackageName+".repository.go")},
		}

		fmt.Printf("\n⚓  Generando repository: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate middleware —

var genMiddlewareCmd = &cobra.Command{
	Use:   "middleware <name>",
	Short: "Genera un middleware",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
		dest := filepath.Join("internal", "middleware", data.SnakeName+".middleware.go")

		files := []struct{ tmpl, dest string }{
			{"templates/middleware/middleware.go.tmpl", dest},
		}

		fmt.Printf("\n⚓  Generando middleware: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate guard —

var genGuardCmd = &cobra.Command{
	Use:   "guard <name>",
	Short: "Genera un guard de acceso",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
		dest := filepath.Join("internal", "guards", data.SnakeName+".guard.go")

		files := []struct{ tmpl, dest string }{
			{"templates/guard/guard.go.tmpl", dest},
		}

		fmt.Printf("\n⚓  Generando guard: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate dto —

var genDtoCmd = &cobra.Command{
	Use:   "dto <name>",
	Short: "Genera DTOs standalone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
		dest := filepath.Join("internal", "modules", data.PackageName, "dto", data.SnakeName+".dto.go")

		files := []struct{ tmpl, dest string }{
			{"templates/dto/dto.go.tmpl", dest},
		}

		fmt.Printf("\n⚓  Generando dto: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — generate crud —

var genCrudCmd = &cobra.Command{
	Use:   "crud <name>",
	Short: "Genera módulo completo con CRUD y DTOs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := generator.NewData(args[0])
		base := filepath.Join("internal", "modules", data.PackageName)

		files := []struct{ tmpl, dest string }{
			{"templates/module/module.go.tmpl", filepath.Join(base, data.PackageName+".module.go")},
			{"templates/module/controller.go.tmpl", filepath.Join(base, data.PackageName+".controller.go")},
			{"templates/module/service.go.tmpl", filepath.Join(base, data.PackageName+".service.go")},
			{"templates/module/repository.go.tmpl", filepath.Join(base, data.PackageName+".repository.go")},
			{"templates/dto/dto.go.tmpl", filepath.Join(base, "dto", data.SnakeName+".dto.go")},
		}

		fmt.Printf("\n⚓  Generando CRUD completo: %s\n\n", data.PascalName)
		return renderFiles(files, data)
	},
}

// — helper —

func renderFiles(files []struct{ tmpl, dest string }, data generator.Data) error {
	for _, f := range files {
		if generator.FileExists(f.dest) {
			fmt.Printf("  ⚠  ya existe: %s (omitido)\n", f.dest)
			continue
		}
		if err := generator.RenderToFile(f.tmpl, f.dest, data); err != nil {
			return err
		}
		fmt.Printf("  ✓  %s\n", f.dest)
	}
	fmt.Println("\n  ✅ Listo\n")
	return nil
}
