// OVAV Permissions Verifier
package permissions

import (
	"encoding/json"
	"os"
	"strings"
)

// VerifyPermissionAuthority checks permission_authority.json integrity.
// Returns true if schema_version contains "v2" or "v3".
func VerifyPermissionAuthority(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return false
	}

	schemaVersion, ok := m["schema_version"].(string)
	if !ok {
		return false
	}

	return strings.Contains(schemaVersion, "v2") || strings.Contains(schemaVersion, "v3")
}
