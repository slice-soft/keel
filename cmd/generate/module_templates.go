package generate

import (
	"path/filepath"

	generator "github.com/slice-soft/keel/internal/generator/generate"
)

func buildModuleTemplateFiles(moduleName string, repositoryChoice repositoryBackend, transactional bool) []genFile {
	data := generator.NewData(moduleName)

	baseDir := moduleDir(moduleName)
	profileDir := moduleTemplateProfileDir(repositoryChoice)
	files := []genFile{
		{
			template: filepath.Join("templates", "modules", profileDir, "dto.go.tmpl"),
			dest:     filepath.Join(baseDir, data.SnakeName+"_dto.go"),
			data:     data,
		},
		{
			template: filepath.Join("templates", "modules", profileDir, "entity.go.tmpl"),
			dest:     filepath.Join(baseDir, data.SnakeName+"_entity.go"),
			data:     data,
		},
		{
			template: filepath.Join("templates", "modules", profileDir, "service.go.tmpl"),
			dest:     filepath.Join(baseDir, data.SnakeName+"_service.go"),
			data:     data,
		},
		{
			template: filepath.Join("templates", "modules", profileDir, "service_test.go.tmpl"),
			dest:     filepath.Join(baseDir, data.SnakeName+"_service_test.go"),
			data:     data,
		},
	}

	if !transactional {
		files = append(files,
			genFile{
				template: filepath.Join("templates", "modules", profileDir, "controller.go.tmpl"),
				dest:     filepath.Join(baseDir, data.SnakeName+"_controller.go"),
				data:     data,
			},
			genFile{
				template: filepath.Join("templates", "modules", profileDir, "controller_test.go.tmpl"),
				dest:     filepath.Join(baseDir, data.SnakeName+"_controller_test.go"),
				data:     data,
			},
		)
	}

	if repositoryChoice != repositoryBackendStub {
		files = append(files,
			genFile{
				template: filepath.Join("templates", "modules", profileDir, "repository.go.tmpl"),
				dest:     filepath.Join(baseDir, data.SnakeName+"_repository.go"),
				data:     data,
			},
			genFile{
				template: filepath.Join("templates", "modules", profileDir, "repository_test.go.tmpl"),
				dest:     filepath.Join(baseDir, data.SnakeName+"_repository_test.go"),
				data:     data,
			},
		)
	}

	return files
}

func moduleTemplateProfileDir(repositoryChoice repositoryBackend) string {
	switch repositoryChoice {
	case repositoryBackendGorm:
		return "gorm"
	case repositoryBackendMongo:
		return "mongo"
	default:
		return "base"
	}
}
