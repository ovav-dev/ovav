package fde

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var knownNames = map[string][]string{
	"thavren": {"THAVREN_SELF_MODEL.yaml", "SELF_MODEL.yaml"},
	"eidren":  {"EIDREN_SELF_MODEL.yaml", "SELF_MODEL.yaml"},
	"valeria": {"SIGRUN_SELF_MODEL.yaml", "SELF_MODEL.yaml"},
	"dante":   {"TORVIN_SELF_MODEL.yaml", "SELF_MODEL.yaml"},
	"renata":  {"KELDA_SELF_MODEL.yaml", "SELF_MODEL.yaml"},
	"sofia":   {"SOFIA_SELF_MODEL.yaml", "SELF_MODEL.yaml"},
	"uriel":   {"URIEL_SELF_MODEL.yaml", "SELF_MODEL.yaml"},
	"kenji":   {"KENJI_SELF_MODEL.yaml", "SELF_MODEL.yaml"},
	"camila":  {"CAMILA_SELF_MODEL.yaml", "SELF_MODEL.yaml"},
}

// LoadBrainPack reads all brain YAML files for a lead.
func LoadBrainPack(repoRoot, areaID, leadID string) (*BrainPack, error) {
	areaDir := filepath.Join(repoRoot, ".ovav", "service_areas", areaID, leadID)
	if _, err := os.Stat(areaDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("fde: lead directory not found: %s", areaDir)
	}

	pack := &BrainPack{
		Lead:       leadID,
		Area:       areaID,
		LoadedFrom: areaDir,
	}

	pack.SelfModel = extract[SelfModel](areaDir, "self_model", leadID)
	pack.Criteria = extract[Criteria](areaDir, "criteria", "")
	pack.Evolution = extract[Evolution](areaDir, "evolution", "")
	pack.OpLevel = extract[OperatingLevel](areaDir, "foundational_law", "")
	pack.Rel = extract[Relationship](areaDir, "relationship", "")

	return pack, nil
}

// extract reads a YAML file, extracts the top-level key, and unmarshals into T.
func extract[T any](dir, key, leadID string) *T {
	candidates := candidatesForKey(key, leadID)
	path := findFile(dir, candidates)
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	// Parse as generic map, extract key, re-marshal, unmarshal into T
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil
	}
	inner, ok := raw[key]
	if !ok {
		return nil
	}
	subset, err := yaml.Marshal(inner)
	if err != nil {
		return nil
	}
	var result T
	if err := yaml.Unmarshal(subset, &result); err != nil {
		return nil
	}
	return &result
}

func candidatesForKey(key, leadID string) []string {
	switch key {
	case "self_model":
		c := []string{"SELF_MODEL.yaml"}
		if names, ok := knownNames[leadID]; ok {
			c = append(names, c...)
		}
		return c
	case "criteria":
		return []string{"CRITERIA.yaml"}
	case "evolution":
		return []string{"EVOLUTION.yaml"}
	case "foundational_law":
		return []string{"OPERATING_LEVEL.yaml"}
	case "relationship":
		return []string{"OVAV_RELATIONSHIP.yaml"}
	}
	return nil
}

func findFile(dir string, candidates []string) string {
	for _, name := range candidates {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
