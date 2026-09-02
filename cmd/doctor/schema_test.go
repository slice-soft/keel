package doctor

import (
	"path/filepath"
	"testing"
)

const gormEntityWithTableName = `package tasks

import "github.com/slice-soft/ss-keel-gorm/database"

type TasksEntity struct {
	database.EntityBase
	Name string ` + "`json:\"name\"`" + `
}

func (TasksEntity) TableName() string {
	return "tasks"
}
`

const gormEntityWithoutTableName = `package orderitem

import "github.com/slice-soft/ss-keel-gorm/database"

type OrderItemEntity struct {
	database.EntityBase
	Name string ` + "`json:\"name\"`" + `
}
`

const mongoEntity = `package posts

import "github.com/slice-soft/ss-keel-mongo/mongo"

type PostsEntity struct {
	mongo.EntityBase
	Name string ` + "`json:\"name\"`" + `
}
`

func TestDiscoverGormModules_UsesPinnedTableName(t *testing.T) {
	doctorSetupDir(t)
	writeDoctorFile(t, filepath.Join(modulesDir, "tasks", "tasks_entity.go"), gormEntityWithTableName)

	modules, err := discoverGormModules(modulesDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(modules))
	}

	module := modules[0]
	if module.Table != "tasks" {
		t.Errorf("Table = %q, want %q", module.Table, "tasks")
	}
	if !module.TablePinned {
		t.Error("expected TablePinned to be true when TableName() is present")
	}
	if want := filepath.Join(schemaDir, "tasks.sql"); module.SchemaPath != want {
		t.Errorf("SchemaPath = %q, want %q", module.SchemaPath, want)
	}
}

// Without TableName() the reported table must be the one GORM itself derives,
// otherwise the diagnosis would name a table the application never queries.
func TestDiscoverGormModules_FallsBackToGormDerivedTable(t *testing.T) {
	doctorSetupDir(t)
	writeDoctorFile(t, filepath.Join(modulesDir, "orderitem", "order_item_entity.go"), gormEntityWithoutTableName)

	modules, err := discoverGormModules(modulesDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(modules))
	}
	if modules[0].Table != "order_item_entities" {
		t.Errorf("Table = %q, want %q", modules[0].Table, "order_item_entities")
	}
	if modules[0].TablePinned {
		t.Error("expected TablePinned to be false when TableName() is absent")
	}
}

func TestDiscoverGormModules_IgnoresNonGormModules(t *testing.T) {
	doctorSetupDir(t)
	writeDoctorFile(t, filepath.Join(modulesDir, "posts", "posts_entity.go"), mongoEntity)
	writeDoctorFile(t, filepath.Join(modulesDir, "starter", "module.go"), "package starter\n")

	modules, err := discoverGormModules(modulesDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(modules) != 0 {
		t.Fatalf("expected no GORM modules, got %d", len(modules))
	}
}

func TestCheckGormSchemas_ErrorsWhenSchemaIsMissing(t *testing.T) {
	doctorSetupDir(t)
	writeDoctorFile(t, filepath.Join(modulesDir, "tasks", "tasks_entity.go"), gormEntityWithTableName)

	hasErrors := false
	checkGormSchemas(&hasErrors)
	if !hasErrors {
		t.Fatal("expected a missing schema to be a hard error")
	}
}

func TestCheckGormSchemas_PassesWhenSchemaExists(t *testing.T) {
	doctorSetupDir(t)
	writeDoctorFile(t, filepath.Join(modulesDir, "tasks", "tasks_entity.go"), gormEntityWithTableName)
	writeDoctorFile(t, filepath.Join(schemaDir, "tasks.sql"), "CREATE TABLE tasks (id TEXT);\n")

	hasErrors := false
	checkGormSchemas(&hasErrors)
	if hasErrors {
		t.Fatal("expected no error when the schema file exists")
	}
}

// A project with no modules directory at all must not be flagged.
func TestCheckGormSchemas_NoModulesDirIsNotAnError(t *testing.T) {
	doctorSetupDir(t)

	hasErrors := false
	checkGormSchemas(&hasErrors)
	if hasErrors {
		t.Fatal("expected no error when internal/modules does not exist")
	}
}

func TestGormDefaultTableName(t *testing.T) {
	tests := []struct {
		typeName string
		want     string
	}{
		{"TasksEntity", "tasks_entities"},
		{"OrderItemEntity", "order_item_entities"},
		{"UserEntity", "user_entities"},
		{"Company", "companies"},
		{"Box", "boxes"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			if got := gormDefaultTableName(tt.typeName); got != tt.want {
				t.Errorf("gormDefaultTableName(%q) = %q, want %q", tt.typeName, got, tt.want)
			}
		})
	}
}
