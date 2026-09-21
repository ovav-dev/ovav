package validators

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"time"
)

func isYOLOPolicy(root string, policy map[string]interface{}) bool {
	if !hasCanonicalYOLOMarker(policy) {
		return false
	}

	// Git HEAD is OVAV's canonical temporal authority. A worktree-only marker
	// cannot activate relaxed YOLO invariants.
	tracked, err := exec.Command("git", "-C", root, "show", "HEAD:.ovav/policy/permission_authority.json").Output()
	if err != nil {
		return false
	}
	current, err := readRegularFileNoFollow(filepath.Join(root, ".ovav", "policy", "permission_authority.json"))
	if err != nil {
		return false
	}
	return bytes.Equal(current, tracked)
}

func hasCanonicalYOLOMarker(policy map[string]interface{}) bool {
	if strVal(policy, "schema_version") != "ovav.permission_authority.v3" {
		return false
	}
	if strVal(policy, "authority") != "OVAV SYSTEMS is canonical. CEO Alexander Salvador governs. OVAV AGENTS operates with total freedom when installed." {
		return false
	}
	marker, ok := policy["_ovav_yolo"].(map[string]interface{})
	if !ok {
		return false
	}
	applied, err := time.Parse("2006-01-02", strVal(marker, "applied"))
	if marker["enabled"] != true || err != nil || applied.After(time.Now().UTC().AddDate(0, 0, 1)) {
		return false
	}
	for _, field := range []string{
		"bash_default", "edit_default", "write_default", "read_default",
		"external_directory_default", "doom_loop_default",
	} {
		if strVal(marker, field) != "allow" {
			return false
		}
	}
	return true
}
