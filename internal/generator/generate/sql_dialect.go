package generate

import "strings"

// DefaultDBEngine is the engine assumed when application.properties does not
// declare one. It matches the default written by `keel add gorm`.
const DefaultDBEngine = "sqlite"

// SQLDialect holds the column types used to render the base DDL of a module.
// Keel entities always share the same three columns from database.EntityBase
// (a UUID string primary key and two millisecond timestamps), so a dialect only
// needs to say how those map onto each engine.
type SQLDialect struct {
	Engine              string
	IDType              string
	TimestampType       string
	TextType            string
	SupportsIfNotExists bool
	// Known is false when the engine is not one the CLI ships types for —
	// typically an engine registered through database.RegisterDialector.
	// The DDL is still rendered, with ANSI types and a warning comment.
	Known bool
}

var sqlDialects = map[string]SQLDialect{
	"sqlite": {
		IDType: "TEXT", TimestampType: "INTEGER", TextType: "TEXT",
		SupportsIfNotExists: true,
	},
	"postgres": {
		IDType: "VARCHAR(36)", TimestampType: "BIGINT", TextType: "VARCHAR(255)",
		SupportsIfNotExists: true,
	},
	"mysql": {
		IDType: "VARCHAR(36)", TimestampType: "BIGINT", TextType: "VARCHAR(255)",
		SupportsIfNotExists: true,
	},
	"mariadb": {
		IDType: "VARCHAR(36)", TimestampType: "BIGINT", TextType: "VARCHAR(255)",
		SupportsIfNotExists: true,
	},
	"sqlserver": {
		IDType: "NVARCHAR(36)", TimestampType: "BIGINT", TextType: "NVARCHAR(255)",
		SupportsIfNotExists: false,
	},
}

// DialectFor returns the SQL dialect for an engine name as written in
// application.properties. Unknown engines fall back to ANSI types flagged with
// Known=false so the generated file can warn about it instead of silently
// emitting types for the wrong database.
func DialectFor(engine string) SQLDialect {
	key := strings.ToLower(strings.TrimSpace(engine))
	if key == "" {
		key = DefaultDBEngine
	}
	if dialect, ok := sqlDialects[key]; ok {
		dialect.Engine = key
		dialect.Known = true
		return dialect
	}
	return SQLDialect{
		Engine:              key,
		IDType:              "VARCHAR(36)",
		TimestampType:       "BIGINT",
		TextType:            "VARCHAR(255)",
		SupportsIfNotExists: true,
		Known:               false,
	}
}
