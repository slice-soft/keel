package keeltoml

import (
	"fmt"
	"os"
	"strings"
)

// RemoveAddon removes the [[addons]] entry with the given id and all [[env]]
// entries whose source matches id from the keel.toml at path.
// Returns true when the file was modified.
func RemoveAddon(path, id string) (bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("reading %s: %w", path, err)
	}

	result, changed := filterAddonBlocks(string(data), id)
	if !changed {
		return false, nil
	}

	if err := os.WriteFile(path, []byte(result), 0644); err != nil {
		return false, fmt.Errorf("writing %s: %w", path, err)
	}
	return true, nil
}

// UpdateAddonVersion sets the version field in the [[addons]] block matching id.
// Returns true when the file was modified.
func UpdateAddonVersion(path, id, version string) (bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("reading %s: %w", path, err)
	}

	result, changed := rewriteAddonVersion(string(data), id, version)
	if !changed {
		return false, nil
	}

	if err := os.WriteFile(path, []byte(result), 0644); err != nil {
		return false, fmt.Errorf("writing %s: %w", path, err)
	}
	return true, nil
}

// filterAddonBlocks removes [[addons]] blocks matching id and [[env]] blocks
// whose source matches id from raw TOML content.
func filterAddonBlocks(content, id string) (string, bool) {
	blocks := splitTomlBlocks(content)
	changed := false
	var kept []string

	for _, b := range blocks {
		if blockIsAddonWithID(b, id) || blockIsEnvWithSource(b, id) {
			changed = true
			continue
		}
		kept = append(kept, b)
	}

	if !changed {
		return content, false
	}

	result := strings.Join(kept, "")
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	return result, true
}

// rewriteAddonVersion updates the version field in the [[addons]] block matching id.
func rewriteAddonVersion(content, id, version string) (string, bool) {
	blocks := splitTomlBlocks(content)
	changed := false

	for i, b := range blocks {
		if !blockIsAddonWithID(b, id) {
			continue
		}
		updated := setVersionInBlock(b, version)
		if updated != b {
			blocks[i] = updated
			changed = true
		}
	}

	if !changed {
		return content, false
	}
	return strings.Join(blocks, ""), true
}

// splitTomlBlocks splits raw TOML content into blocks where each block starts
// at a [section] or [[array]] header. Content before the first header is the
// first block (the preamble). Uses byte positions to preserve all whitespace.
func splitTomlBlocks(content string) []string {
	lines := strings.Split(content, "\n")

	lineStart := make([]int, len(lines))
	pos := 0
	for i, line := range lines {
		lineStart[i] = pos
		pos += len(line) + 1
	}

	var blocks []string
	blockStart := 0

	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "[") {
			end := min(lineStart[i], len(content))
			blocks = append(blocks, content[blockStart:end])
			blockStart = lineStart[i]
		}
	}

	if blockStart <= len(content) {
		blocks = append(blocks, content[blockStart:])
	}

	return blocks
}

func blockIsAddonWithID(block, target string) bool {
	return blockHasHeader(block, "[[addons]]") && blockFieldEquals(block, "id", target)
}

func blockIsEnvWithSource(block, target string) bool {
	return blockHasHeader(block, "[[env]]") && blockFieldEquals(block, "source", target)
}

func blockHasHeader(block, header string) bool {
	for line := range strings.SplitSeq(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		return trimmed == header
	}
	return false
}

func blockFieldEquals(block, field, value string) bool {
	for line := range strings.SplitSeq(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, field) {
			continue
		}
		return extractTomlStringValue(trimmed) == value
	}
	return false
}

func setVersionInBlock(block, version string) string {
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "version") {
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		lines[i] = fmt.Sprintf("%sversion      = %q", indent, version)
	}
	return strings.Join(lines, "\n")
}

// extractTomlStringValue extracts the value from a TOML assignment like key = "value".
func extractTomlStringValue(line string) string {
	_, after, ok := strings.Cut(line, "=")
	if !ok {
		return ""
	}
	val := strings.TrimSpace(after)
	if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') {
		return val[1 : len(val)-1]
	}
	return val
}
