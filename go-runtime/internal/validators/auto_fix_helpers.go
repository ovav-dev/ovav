package validators

import (
	"encoding/json"
	"os"
)

// jsonMarshalHelper and jsonUnmarshalHelper wrap encoding/json for use by
// auto_fix_registry.go without needing direct imports there.
func jsonMarshalHelper(v interface{}) ([]byte, error) {
	// Compact (no indentation) — JSONL requires one record per line
	return json.Marshal(v)
}

func jsonUnmarshalHelper(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func splitLinesString(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// fileExists checks if a file exists (helper used by Fix implementations).
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
