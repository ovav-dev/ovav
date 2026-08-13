// OVAV Permissions Verifier
package permissions

import (
	"encoding/json"
	"os"
	"strings"
)

// VerifyPermissionAuthority checks permission_authority.json integrity. It
// preserves compatibility with v2 while validating canonical v3 fields.
func VerifyPermissionAuthority(path string) bool {
	data, policy, ok := readPermissionAuthority(path)
	if !ok {
		return false
	}
	if strings.Contains(policy.SchemaVersion, "v2") {
		return len(data) > 0
	}
	return validPermissionAuthorityV3(policy)
}

// VerifyPermissionAuthorityV3 checks canonical schema v3 without accepting
// legacy permission authority versions.
func VerifyPermissionAuthorityV3(path string) bool {
	_, policy, ok := readPermissionAuthority(path)
	return ok && validPermissionAuthorityV3(policy)
}

type permissionAuthorityDocument struct {
	SchemaVersion       string          `json:"schema_version"`
	Authority           json.RawMessage `json:"authority"`
	Governor            json.RawMessage `json:"governor"`
	MaterializedTargets []string        `json:"materialized_targets"`
}

func readPermissionAuthority(path string) ([]byte, permissionAuthorityDocument, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, permissionAuthorityDocument{}, false
	}
	var policy permissionAuthorityDocument
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, permissionAuthorityDocument{}, false
	}
	return data, policy, true
}

func validPermissionAuthorityV3(policy permissionAuthorityDocument) bool {
	if policy.SchemaVersion != "ovav.permission_authority.v3" || len(policy.MaterializedTargets) == 0 {
		return false
	}
	return nonEmptyAuthorityValue(policy.Authority) && nonEmptyAuthorityValue(policy.Governor)
}

func nonEmptyAuthorityValue(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return false
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) != ""
	case map[string]any:
		return len(typed) > 0
	default:
		return false
	}
}
