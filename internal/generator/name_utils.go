package generator

import "strings"

var (
	kebabReplacer   = strings.NewReplacer("_", "-", " ", "-")
	snakeReplacer   = strings.NewReplacer("-", "_", " ", "_")
	packageReplacer = strings.NewReplacer("-", "", "_", "", " ", "")
)

func normalizeLower(value string, replacer *strings.Replacer) string {
	return replacer.Replace(strings.ToLower(value))
}

func toKebab(value string) string {
	return normalizeLower(value, kebabReplacer)
}

func toSnake(value string) string {
	return normalizeLower(value, snakeReplacer)
}

func toPackage(value string) string {
	return normalizeLower(value, packageReplacer)
}

func toPascal(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})

	var result strings.Builder
	for _, p := range parts {
		if p != "" {
			result.WriteString(strings.ToUpper(p[:1]) + p[1:])
		}
	}

	return result.String()
}

func toCamel(pascal string) string {
	if pascal == "" {
		return ""
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}
