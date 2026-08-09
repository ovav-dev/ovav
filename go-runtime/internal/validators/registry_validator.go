package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// RegistryValidator ensures required .ovav/registry/*.yaml files exist and
// parse correctly as valid YAML.
// Replaces: validate_registries.py
type RegistryValidator struct{}

func NewRegistryValidator() *RegistryValidator { return &RegistryValidator{} }

func (r *RegistryValidator) ID() string   { return "registry_validator" }
func (r *RegistryValidator) Name() string { return "Registry File Validator" }
func (r *RegistryValidator) Description() string {
	return "Validates required registry YAML files exist and parse correctly"
}
func (r *RegistryValidator) Weight() int { return 5 }

// Required registry files that must exist and parse as valid YAML.
var requiredRegistryFiles = []string{
	"auto_triggers.yaml",
	"capability_scores.yaml",
	"visible_surfaces.yaml",
}

func (r *RegistryValidator) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	registryDir := filepath.Join(root, ".ovav", "registry")

	for _, filename := range requiredRegistryFiles {
		fullPath := filepath.Join(registryDir, filename)
		relPath := fmt.Sprintf(".ovav/registry/%s", filename)

		data, err := os.ReadFile(fullPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("missing required registry file: %s", relPath))
			continue
		}

		// Validate YAML parses correctly
		var parsed interface{}
		if err := yaml.Unmarshal(data, &parsed); err != nil {
			issues = append(issues, fmt.Sprintf("failed to parse %s: %v", relPath, err))
			continue
		}

		// Ensure parsed content is non-nil (not an empty/near-empty file)
		if parsed == nil {
			issues = append(issues, fmt.Sprintf("%s parsed but yielded nil content (near-empty or invalid)", relPath))
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: r.ID(), Name: r.Name(), Status: "fail", Weight: r.Weight(),
			Message:  fmt.Sprintf("FAIL registry validation — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	return Result{
		ID: r.ID(), Name: r.Name(), Status: "pass", Weight: r.Weight(),
		Message:  "PASS registry validation — all required files exist and parse correctly",
		Duration: time.Since(start),
	}
}

var _ Validator = (*RegistryValidator)(nil)
