package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Skills validates the skills.yaml, skill_rule_packs.yaml, and skill_scores.yaml registries.
// Replaces: validate_skills.py
type Skills struct{}

func NewSkills() *Skills { return &Skills{} }

func (s *Skills) ID() string   { return "validate_skills" }
func (s *Skills) Name() string { return "Skills Validator" }
func (s *Skills) Description() string {
	return "Validates skills registry YAML structure, rule packs, scores, and cross-references"
}
func (s *Skills) Weight() int { return 6 }

func (s *Skills) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	registryDir := filepath.Join(root, ".ovav", "registry")

	// Load skills.yaml
	skillsPath := filepath.Join(registryDir, "skills.yaml")
	skillsData, err := os.ReadFile(skillsPath)
	if err != nil {
		return Result{
			ID: s.ID(), Name: s.Name(), Status: "skip", Weight: s.Weight(),
			Message:  fmt.Sprintf("SKIP — skills.yaml not found: %v", err),
			Duration: time.Since(start),
		}
	}
	var skillsDoc struct {
		Skills map[string]map[string]interface{} `yaml:"skills"`
	}
	if err := yaml.Unmarshal(skillsData, &skillsDoc); err != nil {
		return Result{
			ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message:  "FAIL — skills.yaml YAML parse error",
			Issues:   []string{fmt.Sprintf("YAML error: %v", err)},
			Duration: time.Since(start),
		}
	}

	// Load skill_rule_packs.yaml
	rulePacksPath := filepath.Join(registryDir, "skill_rule_packs.yaml")
	rulePacksData, err := os.ReadFile(rulePacksPath)
	if err != nil {
		return Result{
			ID: s.ID(), Name: s.Name(), Status: "skip", Weight: s.Weight(),
			Message:  "SKIP — skill_rule_packs.yaml not found",
			Duration: time.Since(start),
		}
	}
	var rulePacksDoc struct {
		SkillRulePacks map[string]interface{} `yaml:"skill_rule_packs"`
	}
	if err := yaml.Unmarshal(rulePacksData, &rulePacksDoc); err != nil {
		issues = append(issues, "skill_rule_packs.yaml must contain a skill_rule_packs mapping")
	}

	// Load skill_scores.yaml
	scoresPath := filepath.Join(registryDir, "skill_scores.yaml")
	scoresData, err := os.ReadFile(scoresPath)
	var blockedThreshold float64 = 0
	if err == nil {
		var scoresDoc struct {
			SkillScores struct {
				Thresholds map[string]float64 `yaml:"thresholds"`
			} `yaml:"skill_scores"`
		}
		if yaml.Unmarshal(scoresData, &scoresDoc) == nil {
			blockedThreshold = scoresDoc.SkillScores.Thresholds["blocked"]
		}
	}

	rulePacks := rulePacksDoc.SkillRulePacks

	for skillID, skill := range skillsDoc.Skills {
		// Required fields
		requiredFields := []string{"owner_profile", "owner_lane", "provenance", "risk_level", "permission_level"}
		for _, field := range requiredFields {
			if v, ok := skill[field].(string); !ok || strings.TrimSpace(v) == "" {
				issues = append(issues, fmt.Sprintf("%s: %s is required", skillID, field))
			}
		}

		// Rule pack reference
		rulePackRef, _ := skill["rule_pack_ref"].(string)
		if rulePackRef == "" {
			rulePackRef = skillID
		}
		if _, ok := rulePacks[rulePackRef]; !ok {
			issues = append(issues, fmt.Sprintf("%s: must reference a rule pack (looking for '%s')", skillID, rulePackRef))
		}

		// Score — accept int or float64 (yaml.v3 uses int for whole numbers)
		var scoreValue float64
		switch v := skill["score"].(type) {
		case float64:
			scoreValue = v
		case int:
			scoreValue = float64(v)
		default:
			issues = append(issues, fmt.Sprintf("%s: score is required (got %T)", skillID, skill["score"]))
			continue
		}
		if scoreValue < blockedThreshold {
			issues = append(issues, fmt.Sprintf("%s: score %.1f is below blocked threshold %.1f", skillID, scoreValue, blockedThreshold))
		}

		// Eval status
		evalStatus, _ := skill["eval_status"].(string)
		status, _ := skill["status"].(string)
		if evalStatus == "" {
			if status != "experimental" {
				issues = append(issues, fmt.Sprintf("%s: without eval_status must be experimental", skillID))
			}
		} else if evalStatus != "passed" && evalStatus != "pending" && evalStatus != "failed" {
			issues = append(issues, fmt.Sprintf("%s: eval_status must be passed, pending, or failed", skillID))
		}

		// Memory write
		if mw, ok := skill["memory_write"].(bool); ok && mw {
			issues = append(issues, fmt.Sprintf("%s: memory_write requires the memory gateway", skillID))
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message:  fmt.Sprintf("FAIL — %d issue(s) in %d skills", len(issues), len(skillsDoc.Skills)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: s.ID(), Name: s.Name(), Status: "pass", Weight: s.Weight(),
		Message:  fmt.Sprintf("PASS — %d skills validated", len(skillsDoc.Skills)),
		Duration: time.Since(start),
	}
}

var _ Validator = (*Skills)(nil)
