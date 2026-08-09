package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// MemoryPolicy validates the memory_policy.yaml registry structure.
type MemoryPolicy struct{}

func NewMemoryPolicy() *MemoryPolicy { return &MemoryPolicy{} }

func (m *MemoryPolicy) ID() string   { return "validate_memory_policy" }
func (m *MemoryPolicy) Name() string { return "Memory Policy Validator" }
func (m *MemoryPolicy) Description() string {
	return "Validates memory_policy.yaml privacy tags and write pipeline"
}
func (m *MemoryPolicy) Weight() int { return 6 }

func (m *MemoryPolicy) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	policyPath := filepath.Join(root, ".ovav", "registry", "memory_policy.yaml")
	data, err := os.ReadFile(policyPath)
	if err != nil {
		return Result{
			ID: m.ID(), Name: m.Name(), Status: "skip", Weight: m.Weight(),
			Message:  fmt.Sprintf("SKIP — memory_policy.yaml not found: %v", err),
			Duration: time.Since(start),
		}
	}

	var doc struct {
		MemoryPolicy struct {
			PrivacyTags    map[string]interface{} `yaml:"privacy_tags"`
			WritePipeline  []string               `yaml:"write_pipeline"`
			RecallPipeline []string               `yaml:"recall_pipeline"`
		} `yaml:"memory_policy"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		issues = append(issues, fmt.Sprintf("YAML parse error: %v", err))
		return Result{
			ID: m.ID(), Name: m.Name(), Status: "fail", Weight: m.Weight(),
			Message:  "FAIL — memory_policy.yaml parse error",
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	mp := doc.MemoryPolicy

	// Check privacy_tags
	expectedTags := map[string]bool{
		"public_project": true, "internal_project": true, "sensitive_local": true,
		"secret": true, "identity_or_personal": true,
	}
	if len(mp.PrivacyTags) != len(expectedTags) {
		issues = append(issues, fmt.Sprintf("privacy_tags: expected %d tags, got %d", len(expectedTags), len(mp.PrivacyTags)))
	} else {
		for tag := range expectedTags {
			if _, ok := mp.PrivacyTags[tag]; !ok {
				issues = append(issues, fmt.Sprintf("privacy_tags: missing '%s'", tag))
			}
		}
	}

	// Check write_pipeline
	expectedWrite := []string{"go_memory_privacy_classifier", "go_memory_redactor", "go_memory_write_gateway"}
	if !stringSlicesEqual(mp.WritePipeline, expectedWrite) {
		issues = append(issues, "write_pipeline must enforce privacy classifier, redaction, and gateway")
	}

	// Check recall_pipeline
	expectedRecall := []string{"go_memory_recall_filter"}
	if !stringSlicesEqual(mp.RecallPipeline, expectedRecall) {
		issues = append(issues, "recall_pipeline must require go_memory_recall_filter")
	}

	if len(issues) > 0 {
		return Result{
			ID: m.ID(), Name: m.Name(), Status: "fail", Weight: m.Weight(),
			Message:  fmt.Sprintf("FAIL — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: m.ID(), Name: m.Name(), Status: "pass", Weight: m.Weight(),
		Message:  "PASS — memory_policy.yaml valid",
		Duration: time.Since(start),
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

var _ Validator = (*MemoryPolicy)(nil)
