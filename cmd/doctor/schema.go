package doctor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	modulesDir      = "internal/modules"
	schemaDir       = "db/schema"
	gormModulePath  = "github.com/slice-soft/ss-keel-gorm"
	tableNameMethod = "TableName"
)

// gormModule is a module under internal/modules whose repository is backed by
// ss-keel-gorm, together with the table it expects to exist.
type gormModule struct {
	Package    string
	Table      string
	SchemaPath string
	// TablePinned is true when the table name came from an entity's TableName()
	// method. When false it was derived from the package name, so a mismatch is
	// possible and the message says so.
	TablePinned bool
}

// checkGormSchemas verifies that every GORM-backed module has the base DDL that
// provisions its table. Keel runs no migrations, so a missing schema file is not
// cosmetic: the table will not exist and every endpoint of that module answers
// 500 at the first request.
func checkGormSchemas(hasErrors *bool) {
	modules, err := discoverGormModules(modulesDir)
	if err != nil || len(modules) == 0 {
		return
	}

	for _, module := range modules {
		if fileExists(module.SchemaPath) {
			checkOk(fmt.Sprintf("module %q has its base schema (%s)", module.Package, module.SchemaPath))
			continue
		}

		*hasErrors = true
		checkErr(fmt.Sprintf("module %q is GORM-backed but %s is missing — table %q will not exist and every %s endpoint will fail with 500",
			module.Package, module.SchemaPath, module.Table, module.Package))
		if !module.TablePinned {
			fmt.Printf("       Its entity has no TableName() method, so GORM derives %q from the Go type.\n", module.Table)
			fmt.Println("       Add a TableName() method to pin a stable table name, then write its DDL.")
		}
		fmt.Printf("       Regenerate it with: keel generate module %s --gorm\n", module.Package)
	}
}

// discoverGormModules scans the modules directory and returns one entry per
// module that imports ss-keel-gorm.
func discoverGormModules(root string) ([]gormModule, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var modules []gormModule
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		usesGorm, table, entityType := inspectModuleDir(dir)
		if !usesGorm {
			continue
		}

		pinned := table != ""
		if !pinned {
			// No TableName() to trust, so report the table GORM itself would
			// derive from the entity type — that is the one the app queries.
			table = gormDefaultTableName(entityType)
		}
		modules = append(modules, gormModule{
			Package:     entry.Name(),
			Table:       table,
			SchemaPath:  filepath.Join(schemaDir, table+".sql"),
			TablePinned: pinned,
		})
	}
	return modules, nil
}

// inspectModuleDir reports whether a module imports ss-keel-gorm, the table an
// entity pins through its TableName() method (empty when none does), and the
// name of the type that embeds database.EntityBase.
func inspectModuleDir(dir string) (usesGorm bool, table string, entityType string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, "", ""
	}

	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			continue
		}
		if importsPath(file, gormModulePath) {
			usesGorm = true
		}
		if table == "" {
			table = tableNameFromFile(file)
		}
		if entityType == "" {
			entityType = entityTypeFromFile(file)
		}
	}
	return usesGorm, table, entityType
}

// entityTypeFromFile returns the name of the struct that embeds
// database.EntityBase, which is what GORM derives a table name from.
func entityTypeFromFile(file *ast.File) string {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok || structType.Fields == nil {
				continue
			}
			for _, field := range structType.Fields.List {
				if len(field.Names) > 0 {
					continue // named field, not an embed
				}
				if selector, ok := field.Type.(*ast.SelectorExpr); ok && selector.Sel.Name == "EntityBase" {
					return typeSpec.Name.Name
				}
			}
		}
	}
	return ""
}

func importsPath(file *ast.File, prefix string) bool {
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// tableNameFromFile returns the string a `func (T) TableName() string` method
// returns, or "" when the file declares no such method with a literal return.
func tableNameFromFile(file *ast.File) string {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name.Name != tableNameMethod || fn.Body == nil {
			continue
		}
		for _, stmt := range fn.Body.List {
			ret, ok := stmt.(*ast.ReturnStmt)
			if !ok || len(ret.Results) != 1 {
				continue
			}
			lit, ok := ret.Results[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			if value, err := strconv.Unquote(lit.Value); err == nil && value != "" {
				return value
			}
		}
	}
	return ""
}

// gormDefaultTableName reproduces the table GORM derives from a Go type when no
// TableName() method pins one: the snake_case of the type with its last word
// pluralised, so TasksEntity becomes tasks_entities.
func gormDefaultTableName(typeName string) string {
	snake := toSnakeCase(typeName)
	if snake == "" {
		return ""
	}
	parts := strings.Split(snake, "_")
	parts[len(parts)-1] = pluralize(parts[len(parts)-1])
	return strings.Join(parts, "_")
}

// toSnakeCase converts a Go identifier to snake_case (OrderItemEntity ->
// order_item_entity).
func toSnakeCase(value string) string {
	var out strings.Builder
	for i, r := range value {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				out.WriteByte('_')
			}
			out.WriteRune(r - 'A' + 'a')
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

func pluralize(word string) string {
	switch {
	case word == "":
		return word
	case strings.HasSuffix(word, "y") && len(word) > 1 && !isVowel(word[len(word)-2]):
		return word[:len(word)-1] + "ies"
	case strings.HasSuffix(word, "s"), strings.HasSuffix(word, "x"),
		strings.HasSuffix(word, "z"), strings.HasSuffix(word, "ch"),
		strings.HasSuffix(word, "sh"):
		return word + "es"
	default:
		return word + "s"
	}
}

func isVowel(ch byte) bool {
	switch ch {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	default:
		return false
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
