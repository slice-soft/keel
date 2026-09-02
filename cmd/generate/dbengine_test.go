package generate

import (
	"os"
	"path/filepath"
	"testing"

	generator "github.com/slice-soft/keel/internal/generator/generate"
)

func TestResolveDBEngine(t *testing.T) {
	tests := []struct {
		name       string
		envFile    string
		properties string
		want       string
	}{
		{
			name: "defaults to sqlite when the project says nothing",
			want: generator.DefaultDBEngine,
		},
		{
			name:       "reads the default out of a ${VAR:default} placeholder",
			properties: "app.name=demo\ndatabase.engine=${DATABASE_ENGINE:postgres}\n",
			want:       "postgres",
		},
		{
			name:       "reads a literal property value",
			properties: "database.engine=mysql\n",
			want:       "mysql",
		},
		{
			name:       ".env wins over the properties default",
			envFile:    "DATABASE_ENGINE=sqlserver\n",
			properties: "database.engine=${DATABASE_ENGINE:postgres}\n",
			want:       "sqlserver",
		},
		{
			name:       "comments and blank lines are skipped",
			properties: "# database.engine=mysql\n\ndatabase.engine=postgres\n",
			want:       "postgres",
		},
		{
			name:       "placeholder without a default falls through to sqlite",
			properties: "database.engine=${DATABASE_ENGINE}\n",
			want:       generator.DefaultDBEngine,
		},
		{
			name:       "an unrelated property is not mistaken for the engine",
			properties: "database.url=${DATABASE_URL:./app.db}\n",
			want:       generator.DefaultDBEngine,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chdirToTemp(t)
			if tt.envFile != "" {
				writeFile(t, ".env", tt.envFile)
			}
			if tt.properties != "" {
				writeFile(t, "application.properties", tt.properties)
			}

			if got := resolveDBEngine(); got != tt.want {
				t.Errorf("resolveDBEngine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSchemaPathFor(t *testing.T) {
	want := filepath.Join("db", "schema", "order_items.sql")
	if got := SchemaPathFor("order_items"); got != want {
		t.Errorf("SchemaPathFor() = %q, want %q", got, want)
	}
}

func chdirToTemp(t *testing.T) {
	t.Helper()

	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
