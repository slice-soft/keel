package generate

import "strings"

var (
	kebabReplacer    = strings.NewReplacer("_", "-", " ", "-")
	snakeReplacer    = strings.NewReplacer("-", "_", " ", "_")
	packageReplacer  = strings.NewReplacer("-", "", "_", "", " ", "")
	irregularPlurals = map[string]string{
		"child":  "children",
		"man":    "men",
		"mouse":  "mice",
		"person": "people",
		"woman":  "women",
	}
	uncountableNouns = map[string]struct{}{
		"equipment":   {},
		"information": {},
		"money":       {},
		"rice":        {},
		"series":      {},
		"species":     {},
	}
)

func normalizeLower(value string, replacer *strings.Replacer) string {
	return replacer.Replace(strings.ToLower(value))
}

func toKebab(value string) string {
	return normalizeLower(value, kebabReplacer)
}

func toPluralKebab(value string) string {
	kebab := toKebab(value)
	if kebab == "" {
		return ""
	}

	parts := strings.Split(kebab, "-")
	parts[len(parts)-1] = pluralizeWord(parts[len(parts)-1])
	return strings.Join(parts, "-")
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

func pluralizeWord(word string) string {
	if word == "" {
		return ""
	}
	if _, ok := uncountableNouns[word]; ok {
		return word
	}
	if plural, ok := irregularPlurals[word]; ok {
		return plural
	}
	for _, plural := range irregularPlurals {
		if word == plural {
			return word
		}
	}

	switch {
	case hasAnySuffix(word, "ies", "ches", "shes", "xes", "zes", "ses", "oes"):
		return word
	case strings.HasSuffix(word, "s") &&
		!hasAnySuffix(word, "ss", "us", "is", "as"):
		return word
	case strings.HasSuffix(word, "y") && len(word) > 1 && !isVowel(word[len(word)-2]):
		return word[:len(word)-1] + "ies"
	case hasAnySuffix(word, "ch", "sh", "x", "z", "s", "o"):
		return word + "es"
	default:
		return word + "s"
	}
}

func hasAnySuffix(value string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(value, suffix) {
			return true
		}
	}
	return false
}

func isVowel(ch byte) bool {
	switch ch {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	default:
		return false
	}
}
