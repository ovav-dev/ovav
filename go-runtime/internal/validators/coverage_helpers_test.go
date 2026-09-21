package validators

import (
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// setsEqual returns true if two sets have identical keys (values ignored for map[string]bool).
func setsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

// mapsEqual returns true if two maps have identical keys and values.
func mapsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

// sortedKeys returns a sorted slice of keys from the given map.
func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// parseAgentFrontmatter reads a markdown file and extracts YAML frontmatter.
// Returns a map of the parsed frontmatter fields, or an error if the file
// doesn't exist, has no frontmatter delimiter, or has malformed YAML.
func parseAgentFrontmatter(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)

	// Must start with opening ---
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return nil, &frontmatterError{"no frontmatter delimiter"}
	}

	// Find closing ---
	endIdx := strings.Index(content[3:], "---\n")
	if endIdx < 0 {
		endIdx = strings.Index(content[3:], "---\r\n")
		if endIdx < 0 {
			return nil, &frontmatterError{"unclosed frontmatter"}
		}
		endIdx += 3 // account for the slice offset
	}
	endIdx += 3 // include the closing ---

	yamlContent := content[3:endIdx]
	var fm map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return nil, &frontmatterError{"invalid YAML: " + err.Error()}
	}
	if fm == nil {
		return nil, &frontmatterError{"empty frontmatter"}
	}
	return fm, nil
}

type frontmatterError struct{ msg string }

func (e *frontmatterError) Error() string { return e.msg }
