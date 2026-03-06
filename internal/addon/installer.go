package addon

import (
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"strings"
)

var execCommand = exec.Command

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

	line := s.Key + "=" + s.Example + "\n"
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

// stepMainCode inserts a code block before app.Listen() in cmd/main.go.
// The guard field prevents duplicate insertion.
func stepMainCode(s Step) error {
	if s.Code == "" {
		return fmt.Errorf("main_code step is missing 'code'")
	}
	return updateMainGo(func(content string) string {
		if s.Guard != "" && strings.Contains(content, s.Guard) {
			return content // already wired
		}
		fmt.Printf("  → wired %s into cmd/main.go\n", s.Guard)
		return addMainLine(content, "\t"+strings.ReplaceAll(s.Code, "\n", "\n\t"))
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
	markers := []string{
		"\tlog.Fatal(app.Listen())",
		"\tif err := app.Listen(); err != nil {",
		"log.Fatal(app.Listen())",
		"if err := app.Listen(); err != nil {",
	}
	for _, marker := range markers {
		idx := strings.Index(content, marker)
		if idx != -1 {
			return content[:idx] + line + "\n\n" + content[idx:]
		}
	}
	return content
}
