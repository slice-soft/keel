package generate

import "testing"

func TestDialectFor(t *testing.T) {
	tests := []struct {
		name              string
		engine            string
		wantEngine        string
		wantIDType        string
		wantTimestampType string
		wantIfNotExists   bool
		wantKnown         bool
	}{
		{
			name:              "sqlite",
			engine:            "sqlite",
			wantEngine:        "sqlite",
			wantIDType:        "TEXT",
			wantTimestampType: "INTEGER",
			wantIfNotExists:   true,
			wantKnown:         true,
		},
		{
			name:              "postgres",
			engine:            "postgres",
			wantEngine:        "postgres",
			wantIDType:        "VARCHAR(36)",
			wantTimestampType: "BIGINT",
			wantIfNotExists:   true,
			wantKnown:         true,
		},
		{
			name:              "mysql",
			engine:            "mysql",
			wantEngine:        "mysql",
			wantIDType:        "VARCHAR(36)",
			wantTimestampType: "BIGINT",
			wantIfNotExists:   true,
			wantKnown:         true,
		},
		{
			name:              "sqlserver has no CREATE TABLE IF NOT EXISTS",
			engine:            "sqlserver",
			wantEngine:        "sqlserver",
			wantIDType:        "NVARCHAR(36)",
			wantTimestampType: "BIGINT",
			wantIfNotExists:   false,
			wantKnown:         true,
		},
		{
			name:              "case and spacing are normalised",
			engine:            "  PostgreS  ",
			wantEngine:        "postgres",
			wantIDType:        "VARCHAR(36)",
			wantTimestampType: "BIGINT",
			wantIfNotExists:   true,
			wantKnown:         true,
		},
		{
			name:              "empty falls back to the default engine",
			engine:            "",
			wantEngine:        DefaultDBEngine,
			wantIDType:        "TEXT",
			wantTimestampType: "INTEGER",
			wantIfNotExists:   true,
			wantKnown:         true,
		},
		{
			name:              "unknown engine keeps its name and is flagged",
			engine:            "cockroach",
			wantEngine:        "cockroach",
			wantIDType:        "VARCHAR(36)",
			wantTimestampType: "BIGINT",
			wantIfNotExists:   true,
			wantKnown:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DialectFor(tt.engine)
			if got.Engine != tt.wantEngine {
				t.Errorf("Engine = %q, want %q", got.Engine, tt.wantEngine)
			}
			if got.IDType != tt.wantIDType {
				t.Errorf("IDType = %q, want %q", got.IDType, tt.wantIDType)
			}
			if got.TimestampType != tt.wantTimestampType {
				t.Errorf("TimestampType = %q, want %q", got.TimestampType, tt.wantTimestampType)
			}
			if got.SupportsIfNotExists != tt.wantIfNotExists {
				t.Errorf("SupportsIfNotExists = %v, want %v", got.SupportsIfNotExists, tt.wantIfNotExists)
			}
			if got.Known != tt.wantKnown {
				t.Errorf("Known = %v, want %v", got.Known, tt.wantKnown)
			}
		})
	}
}

// A known engine must never be reported through the ANSI fallback, otherwise the
// generated DDL would silently carry types for the wrong database.
func TestDialectFor_KnownEnginesAreNeverFallback(t *testing.T) {
	for engine := range sqlDialects {
		if !DialectFor(engine).Known {
			t.Errorf("engine %q resolved to the unknown-engine fallback", engine)
		}
	}
}

func TestNewData_TableNameIsPluralSnake(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"tasks", "tasks"},
		{"user", "users"},
		{"order-item", "order_items"},
		{"company", "companies"},
		{"person", "people"},
		{"series", "series"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			data := NewData(tt.input)
			if data.TableName != tt.want {
				t.Errorf("TableName = %q, want %q", data.TableName, tt.want)
			}
			if data.PluralSnakeName != tt.want {
				t.Errorf("PluralSnakeName = %q, want %q", data.PluralSnakeName, tt.want)
			}
		})
	}
}
