package addon

import (
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/slice-soft/keel/internal/gomod"
	"github.com/slice-soft/keel/internal/keeltoml"
)

var execCommand = exec.Command
var envPlaceholderPattern = regexp.MustCompile(`\{\{\s*([A-Z0-9_]+)\s*\}\}`)

// Install executes all steps defined in the manifest inside the current Keel project.
func Install(m *Manifest) error {
	snap := newInstallSnapshot(m)

	var notes []Step
	for _, step := range m.Steps {
		if step.Type == "note" {
			notes = append(notes, step)
			continue
		}
		if err := runStep(step, m.Name); err != nil {
			fmt.Printf("  ⚠  reverting changes\n")
			snap.restore()
			return fmt.Errorf("step %q failed: %w", step.Type, err)
		}
	}

	if err := runGoModTidy(); err != nil {
		fmt.Printf("  ⚠  %v\n", err)
	}

	// Update keel.toml — only reached when all steps succeeded.
	if err := mergeIntoKeelToml(m); err != nil {
		fmt.Printf("  ⚠  could not update keel.toml: %v\n", err)
	}

	for _, step := range notes {
		if err := stepNote(step); err != nil {
			return fmt.Errorf("step %q failed: %w", step.Type, err)
		}
	}

	return nil
}

// installSnapshot captures the content of every file an addon install can
// modify. Restoring it brings those files back to their exact pre-install state,
// including any uncommitted changes the user had before running keel add.
type installSnapshot struct {
	// nil value means the file did not exist before install; restore deletes it.
	files map[string][]byte
}

func newInstallSnapshot(m *Manifest) *installSnapshot {
	targets := []string{
		"go.mod", "go.sum",
		".env", ".env.example",
		"application.properties",
		"cmd/main.go",
	}
	for _, s := range m.Steps {
		if s.Type == "create_provider_file" && s.Filename != "" {
			targets = append(targets, s.Filename)
		}
	}

	snap := &installSnapshot{files: make(map[string][]byte, len(targets))}
	for _, path := range targets {
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			snap.files[path] = nil
		} else if err == nil {
			b := make([]byte, len(data))
			copy(b, data)
			snap.files[path] = b
		}
		// unreadable for other reasons: skip — cannot restore what we cannot read
	}
	return snap
}

// restore reverts every snapshotted file to its pre-install state.
// Files that did not exist before install are removed.
func (s *installSnapshot) restore() {
	for path, content := range s.files {
		if content == nil {
			_ = os.Remove(path)
		} else {
			_ = os.WriteFile(path, content, 0644)
		}
	}
}

func runStep(s Step, addonName string) error {
	switch s.Type {
	case "go_get":
		return stepGoGet(s)
	case "env":
		return stepEnv(s)
	case "property":
		return stepProperty(s)
	case "main_import":
		return stepMainImport(s)
	case "main_code":
		return stepMainCode(s)
	case "create_provider_file":
		return stepCreateProviderFile(s)
	case "note":
		return stepNote(s)
	default:
		return fmt.Errorf("unknown step type %q in %s", s.Type, addonName)
	}
}

// stepGoGet runs: go get <package> and prints a summary of go.mod changes.
func stepGoGet(s Step) error {
	pkg := strings.TrimSpace(s.Package)
	if pkg == "" {
		return fmt.Errorf("go_get step is missing 'package'")
	}

	before, _ := os.ReadFile("go.mod")

	target := resolveGoGetTarget(pkg)
	fmt.Printf("  → go get %s\n", target)
	cmd := execCommand("go", "get", target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	printGoModChurnSummary(before)
	return nil
}

// printGoModChurnSummary compares the go.mod snapshot taken before go get
// against the current go.mod and prints a short summary when the go directive
// or indirect dependency count changed.
func printGoModChurnSummary(before []byte) {
	after, err := os.ReadFile("go.mod")
	if err != nil || string(before) == string(after) {
		return
	}

	oldDirective := extractGoDirective(string(before))
	newDirective := extractGoDirective(string(after))
	if oldDirective != "" && newDirective != "" && oldDirective != newDirective {
		fmt.Printf("  ℹ  go directive updated: %s → %s\n", oldDirective, newDirective)
	}

	added := countNewIndirectDeps(string(before), string(after))
	if added > 0 {
		fmt.Printf("  ℹ  %d indirect dependenc%s added to go.mod\n", added, pluralSuffix(added, "y", "ies"))
	}
}

func extractGoDirective(goMod string) string {
	for line := range strings.SplitSeq(goMod, "\n") {
		trimmed := strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(trimmed, "go "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

func countNewIndirectDeps(before, after string) int {
	beforeLines := indirectDepSet(before)
	count := 0
	for line := range strings.SplitSeq(after, "\n") {
		if strings.Contains(line, "// indirect") && !beforeLines[strings.TrimSpace(line)] {
			count++
		}
	}
	return count
}

func indirectDepSet(goMod string) map[string]bool {
	set := make(map[string]bool)
	for line := range strings.SplitSeq(goMod, "\n") {
		if strings.Contains(line, "// indirect") {
			set[strings.TrimSpace(line)] = true
		}
	}
	return set
}

func pluralSuffix(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func resolveGoGetTarget(pkg string) string {
	if strings.Contains(pkg, "@") {
		return pkg
	}
	return pkg + "@latest"
}

func runGoModTidy() error {
	return gomod.RunTidy(execCommand, ".", os.Stdout, os.Stderr)
}

// stepEnv adds KEY=example to .env and .env.example if the key is not already present.
func stepEnv(s Step) error {
	if s.Key == "" {
		return fmt.Errorf("env step is missing 'key'")
	}

	envContent, _ := os.ReadFile(".env")
	envExampleContent, _ := os.ReadFile(".env.example")
	lookupContent := string(envContent)
	if lookupContent == "" {
		lookupContent = string(envExampleContent)
	}

	envValue := expandEnvExample(s.Example, lookupContent)
	addedEnv, err := appendEnvKey(".env", s.Key, envValue)
	if err != nil {
		return err
	}
	if addedEnv {
		fmt.Printf("  → added %s to .env\n", s.Key)
	}

	exampleLookupContent := string(envExampleContent)
	if exampleLookupContent == "" {
		exampleLookupContent = string(envContent)
	} else if len(envContent) > 0 {
		exampleLookupContent += "\n" + string(envContent)
	}

	exampleValue := expandEnvExample(s.Example, exampleLookupContent)
	addedExample, err := appendEnvKey(".env.example", s.Key, exampleValue)
	if err != nil {
		return err
	}
	if addedExample {
		fmt.Printf("  → added %s to .env.example\n", s.Key)
	}

	return nil
}

// stepProperty adds key=value to application.properties if the key is not
// already present.
func stepProperty(s Step) error {
	if strings.TrimSpace(s.Key) == "" {
		return fmt.Errorf("property step is missing 'key'")
	}

	added, err := appendPropertyKey("application.properties", s.Key, s.Example, s.Description)
	if err != nil {
		return err
	}
	if added {
		fmt.Printf("  → added %s to application.properties\n", s.Key)
	}

	return nil
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

func appendEnvKey(filename, key, value string) (bool, error) {
	existing, _ := os.ReadFile(filename)
	if _, ok := lookupEnvValue(string(existing), key); ok {
		return false, nil
	}

	line := key + "=" + value + "\n"
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, err
	}
	defer f.Close()

	if _, err := f.WriteString(line); err != nil {
		return false, err
	}
	return true, nil
}

func appendPropertyKey(filename, key, value, description string) (bool, error) {
	existing, _ := os.ReadFile(filename)
	if _, ok := lookupPropertyValue(string(existing), key); ok {
		return false, nil
	}

	var builder strings.Builder
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		builder.WriteByte('\n')
	}
	builder.WriteByte('\n')
	if strings.TrimSpace(description) != "" {
		builder.WriteString("# ")
		builder.WriteString(strings.TrimSpace(description))
		builder.WriteByte('\n')
	}
	builder.WriteString(key)
	builder.WriteByte('=')
	builder.WriteString(value)
	builder.WriteByte('\n')

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, err
	}
	defer f.Close()

	if _, err := f.WriteString(builder.String()); err != nil {
		return false, err
	}
	return true, nil
}

func lookupPropertyValue(content, key string) (string, bool) {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "!") {
			continue
		}

		separator := strings.IndexAny(trimmed, "=:")
		if separator == -1 {
			continue
		}

		lineKey := strings.TrimSpace(trimmed[:separator])
		if lineKey != key {
			continue
		}

		return strings.TrimSpace(trimmed[separator+1:]), true
	}
	return "", false
}

// mergeIntoKeelToml writes the addon's metadata into keel.toml and validates
// the result. If the file becomes invalid TOML after the write, it is restored
// to its pre-merge state and an error is returned.
// A missing keel.toml is created automatically; errors are non-fatal (caller prints a warning).
func mergeIntoKeelToml(m *Manifest) error {
	before, _ := os.ReadFile(keeltoml.DefaultPath)

	changed, err := keeltoml.MergeAddon(
		keeltoml.DefaultPath,
		addonIDForManifest(m),
		installedAddonVersion(m),
		m.Repo,
		m.Capabilities,
		m.Resources,
		envEntriesFromSteps(m),
	)
	if err != nil {
		return err
	}

	if _, validateErr := keeltoml.Load(keeltoml.DefaultPath); validateErr != nil {
		if before == nil {
			_ = os.Remove(keeltoml.DefaultPath)
		} else {
			_ = os.WriteFile(keeltoml.DefaultPath, before, 0644)
		}
		return fmt.Errorf("keel.toml became invalid after merge — reverted: %w", validateErr)
	}

	if changed {
		fmt.Printf("  → updated keel.toml\n")
	}
	return nil
}

// envEntriesFromSteps extracts env step metadata from the manifest and maps it
// to keeltoml.EnvEntry so capabilities, secrets, and descriptions are persisted.
func envEntriesFromSteps(m *Manifest) []keeltoml.EnvEntry {
	source := addonIDForManifest(m)
	var entries []keeltoml.EnvEntry
	for _, s := range m.Steps {
		if s.Type != "env" || s.Key == "" {
			continue
		}
		entries = append(entries, keeltoml.EnvEntry{
			Key:         s.Key,
			Source:      source,
			Required:    s.Required,
			Secret:      s.Secret,
			Default:     s.Example,
			Description: s.Description,
		})
	}
	return entries
}

func addonIDForManifest(m *Manifest) string {
	if strings.HasPrefix(m.Repo, "github.com/slice-soft/ss-keel-") {
		return strings.TrimPrefix(m.Repo, "github.com/slice-soft/ss-keel-")
	}
	if strings.HasPrefix(m.Name, "ss-keel-") {
		return strings.TrimPrefix(m.Name, "ss-keel-")
	}
	if m.Name != "" {
		return m.Name
	}
	return filepath.Base(m.Repo)
}

func installedAddonVersion(m *Manifest) string {
	if strings.TrimSpace(m.Repo) == "" {
		return strings.TrimSpace(m.Version)
	}

	goMod, err := os.ReadFile("go.mod")
	if err != nil {
		return strings.TrimSpace(m.Version)
	}

	lines := strings.Split(string(goMod), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch {
		case len(fields) >= 3 && fields[0] == "require" && fields[1] == m.Repo:
			return fields[2]
		case fields[0] == m.Repo:
			return fields[1]
		}
	}

	return strings.TrimSpace(m.Version)
}

func stepNote(s Step) error {
	message := strings.TrimSpace(s.Message)
	if message == "" {
		message = strings.TrimSpace(s.Description)
	}
	if message == "" {
		return fmt.Errorf("note step is missing 'message'")
	}

	lines := strings.Split(message, "\n")
	for i, line := range lines {
		line = strings.TrimRight(line, " ")
		switch {
		case i == 0:
			fmt.Printf("  ℹ  %s\n", line)
		case line == "":
			fmt.Println()
		default:
			fmt.Printf("     %s\n", line)
		}
	}
	return nil
}

// stepCreateProviderFile creates a dedicated Go file (e.g. cmd/setup_provider.go) that
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
