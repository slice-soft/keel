package addon

import (
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var execCommand = exec.Command
var envPlaceholderPattern = regexp.MustCompile(`\{\{\s*([A-Z0-9_]+)\s*\}\}`)

// Install executes all steps defined in the manifest inside the current Keel project.
func Install(m *Manifest) error {
	for _, step := range m.Steps {
		if err := runStep(step, m.Name); err != nil {
			return fmt.Errorf("step %q failed: %w", step.Type, err)
		}
	}

	if err := runGoModTidy(); err != nil {
		fmt.Printf("  ⚠  %v\n", err)
	}

	return nil
}

func runStep(s Step, addonName string) error {
	switch s.Type {
	case "go_get":
		return stepGoGet(s)
	case "env":
		return stepEnv(s)
	case "main_import":
		return stepMainImport(s)
	case "main_code":
		return stepMainCode(s)
	case "create_provider_file":
		return stepCreateProviderFile(s)
	default:
		return fmt.Errorf("unknown step type %q in %s", s.Type, addonName)
	}
}

// stepGoGet runs: go get <package>
func stepGoGet(s Step) error {
	pkg := strings.TrimSpace(s.Package)
	if pkg == "" {
		return fmt.Errorf("go_get step is missing 'package'")
	}

	target := resolveGoGetTarget(pkg)
	fmt.Printf("  → go get %s\n", target)
	cmd := execCommand("go", "get", target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func resolveGoGetTarget(pkg string) string {
	if strings.Contains(pkg, "@") {
		return pkg
	}
	return pkg + "@latest"
}

func runGoModTidy() error {
	fmt.Printf("  → go mod tidy\n")
	cmd := execCommand("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}
	return nil
}

// stepEnv adds KEY=example to .env if the key is not already present.
func stepEnv(s Step) error {
	if s.Key == "" {
		return fmt.Errorf("env step is missing 'key'")
	}
	const envFile = ".env"

	existing, _ := os.ReadFile(envFile)
	if strings.Contains(string(existing), s.Key+"=") {
		return nil // already set
	}

	line := s.Key + "=" + expandEnvExample(s.Example, string(existing)) + "\n"
	f, err := os.OpenFile(envFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(line)
	if err == nil {
		fmt.Printf("  → added %s to .env\n", s.Key)
	}
	return err
}

// stepMainImport adds an import path to cmd/main.go.
func stepMainImport(s Step) error {
	if s.Path == "" {
		return fmt.Errorf("main_import step is missing 'path'")
	}
	return updateMainGo(func(content string) string {
		importPath := fmt.Sprintf("%q", s.Path)
		if strings.Contains(content, importPath) {
			return content
		}
		fmt.Printf("  → added import %s to cmd/main.go\n", s.Path)
		return addImport(content, importPath)
	})
}

// stepMainCode inserts or rewrites a code block in cmd/main.go.
// The guard field prevents duplicate insertion.
func stepMainCode(s Step) error {
	if s.Code == "" {
		return fmt.Errorf("main_code step is missing 'code'")
	}
	return updateMainGo(func(content string) string {
		if s.Guard != "" && strings.Contains(content, s.Guard) {
			return content // already wired
		}
		if s.Replace != "" {
			updated, replaced := replaceMainLine(content, s.Replace, "\t"+strings.ReplaceAll(s.Code, "\n", "\n\t"))
			if replaced {
				fmt.Printf("  → updated %s wiring in cmd/main.go\n", s.Replace)
				return updated
			}
		}
		fmt.Printf("  → wired %s into cmd/main.go\n", s.Guard)
		return addMainLineWithAnchor(content, "\t"+strings.ReplaceAll(s.Code, "\n", "\n\t"), s.Anchor)
	})
}

func updateMainGo(transform func(string) string) error {
	const mainPath = "cmd/main.go"
	body, err := os.ReadFile(mainPath)
	if err != nil {
		return fmt.Errorf("cmd/main.go not found — run keel add inside a Keel project")
	}

	original := string(body)
	updated := transform(original)
	if updated == original {
		return nil
	}

	formatted, err := format.Source([]byte(updated))
	if err == nil {
		updated = string(formatted)
	}
	return os.WriteFile(mainPath, []byte(updated), 0644)
}

func addImport(content, importPath string) string {
	start := strings.Index(content, "import (")
	if start == -1 {
		return content
	}
	end := strings.Index(content[start:], ")")
	if end == -1 {
		return content
	}
	end += start
	return content[:end] + "\n\t" + importPath + content[end:]
}

func addMainLine(content, line string) string {
	return addMainLineWithAnchor(content, line, "before_listen")
}

func addMainLineWithAnchor(content, line, anchor string) string {
	switch anchor {
	case "", "before_listen":
		return addMainLineBeforeMarkers(content, line, []string{
			"\tlog.Fatal(app.Listen())",
			"\tif err := app.Listen(); err != nil {",
			"log.Fatal(app.Listen())",
			"if err := app.Listen(); err != nil {",
		})
	case "before_modules":
		if updated, ok := addMainLineBeforeMarkersIfFound(content, line, []string{
			"\t// Register your modules here:",
			"// Register your modules here:",
			"\tapp.Use(",
			"app.Use(",
		}); ok {
			return updated
		}
		return addMainLine(content, line)
	default:
		return addMainLine(content, line)
	}
}

func addMainLineBeforeMarkers(content, line string, markers []string) string {
	if updated, ok := addMainLineBeforeMarkersIfFound(content, line, markers); ok {
		return updated
	}
	return content
}

func addMainLineBeforeMarkersIfFound(content, line string, markers []string) (string, bool) {
	for _, marker := range markers {
		idx := strings.Index(content, marker)
		if idx != -1 {
			return content[:idx] + line + "\n\n" + content[idx:], true
		}
	}
	return content, false
}

func replaceMainLine(content, match, replacement string) (string, bool) {
	start := 0
	for start < len(content) {
		lineStart := start
		lineEnd := strings.IndexByte(content[start:], '\n')
		if lineEnd == -1 {
			lineEnd = len(content)
		} else {
			lineEnd += start
		}

		line := content[lineStart:lineEnd]
		if strings.Contains(line, match) {
			return content[:lineStart] + replacement + content[lineEnd:], true
		}

		if lineEnd == len(content) {
			break
		}
		start = lineEnd + 1
	}
	return content, false
}

func addMainLineBeforeListen(content, line string) string {
	markers := []string{
		"\tlog.Fatal(app.Listen())",
		"\tif err := app.Listen(); err != nil {",
		"log.Fatal(app.Listen())",
		"if err := app.Listen(); err != nil {",
	}
	return addMainLineBeforeMarkers(content, line, markers)
}

func expandEnvExample(example, envFileContent string) string {
	return envPlaceholderPattern.ReplaceAllStringFunc(example, func(match string) string {
		submatches := envPlaceholderPattern.FindStringSubmatch(match)
		if len(submatches) != 2 {
			return ""
		}

		key := submatches[1]
		if value, ok := lookupEnvValue(envFileContent, key); ok {
			return value
		}
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
		return ""
	})
}

func lookupEnvValue(envFileContent, key string) (string, bool) {
	for _, line := range strings.Split(envFileContent, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(trimmed, key+"=") {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(trimmed, key+"=")), true
	}
	return "", false
}

// stepCreateProviderFile creates a dedicated Go file (e.g. cmd/setup_jwt.go) that
// holds an addon initializer function, keeping cmd/main.go slim.
// If the file already exists and contains the guard string, the step is skipped.
func stepCreateProviderFile(s Step) error {
	if s.Filename == "" {
		return fmt.Errorf("create_provider_file step is missing 'filename'")
	}
	if s.Content == "" {
		return fmt.Errorf("create_provider_file step is missing 'content'")
	}

	// Skip if already created (idempotent).
	if s.Guard != "" {
		if existing, err := os.ReadFile(s.Filename); err == nil {
			if strings.Contains(string(existing), s.Guard) {
				return nil
			}
		}
	}

	if err := os.MkdirAll(filepath.Dir(s.Filename), 0755); err != nil {
		return fmt.Errorf("could not create directory for %s: %w", s.Filename, err)
	}

	// Format the Go source so gofmt is always satisfied.
	src := []byte(s.Content)
	if formatted, err := format.Source(src); err == nil {
		src = formatted
	}

	if err := os.WriteFile(s.Filename, src, 0644); err != nil {
		return fmt.Errorf("could not write %s: %w", s.Filename, err)
	}

	fmt.Printf("  → created %s\n", s.Filename)
	return nil
}
