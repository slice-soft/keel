package generate

import (
	"path/filepath"

	generator "github.com/slice-soft/keel/internal/generator/generate"
)

func buildModuleTemplateFiles(moduleName string, repositoryChoice repositoryBackend, transactional bool) []genFile {
	data := generator.NewData(moduleName)
	data.SQL = generator.DialectFor(resolveDBEngine())

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

	if repositoryChoice == repositoryBackendGorm {
		files = append(files, gormSchemaFile(data))
	}

	return files
}

// SchemaDir holds the base DDL of every GORM-backed module. Keel does not run
// migrations, so these files are what provisions the database.
const SchemaDir = "db/schema"

// SchemaPathFor returns the DDL path for a table name.
func SchemaPathFor(tableName string) string {
	return filepath.Join(SchemaDir, tableName+".sql")
}

// gormSchemaFile describes the base DDL generated alongside a GORM repository.
func gormSchemaFile(data generator.Data) genFile {
	return genFile{
		template: filepath.Join("templates", "modules", "gorm", "schema.sql.tmpl"),
		dest:     SchemaPathFor(data.TableName),
		data:     data,
	}
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
