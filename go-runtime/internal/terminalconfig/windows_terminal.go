// Package terminalconfig builds safe, merge-ready terminal configuration projections.
package terminalconfig

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var windowsTerminalGUID = regexp.MustCompile(`^\{[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\}$`)

// Plan describes a projection without touching the installed terminal settings.
type Plan struct {
	Destination string `json:"destination"`
	Backup      string `json:"backup"`
	Merged      []byte `json:"-"`
}

// PlanWindowsTerminal merges an OVAV fragment and creates a deterministic backup plan.
func PlanWindowsTerminal(current, fragment []byte, destination string, now time.Time) (Plan, error) {
	var base map[string]interface{}
	if err := json.Unmarshal(current, &base); err != nil {
		return Plan{}, fmt.Errorf("parse installed settings: %w", err)
	}
	var overlay map[string]interface{}
	if err := json.Unmarshal(fragment, &overlay); err != nil {
		return Plan{}, fmt.Errorf("parse OVAV fragment: %w", err)
	}
	if err := validateWindowsTerminal124Subset(base, false); err != nil {
		return Plan{}, fmt.Errorf("validate installed settings: %w", err)
	}
	if err := validateWindowsTerminal124Subset(overlay, true); err != nil {
		return Plan{}, fmt.Errorf("validate OVAV fragment: %w", err)
	}

	merged := mergeObject(base, overlay)
	if err := validateWindowsTerminal124Subset(merged, false); err != nil {
		return Plan{}, fmt.Errorf("validate merged settings: %w", err)
	}
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return Plan{}, fmt.Errorf("encode merged settings: %w", err)
	}
	data = append(data, '\n')
	if !json.Valid(data) {
		return Plan{}, fmt.Errorf("merged settings are not valid JSON")
	}

	stamp := now.UTC().Format("20060102T150405Z")
	return Plan{
		Destination: destination,
		Backup:      destination + ".ovav-backup-" + stamp,
		Merged:      data,
	}, nil
}

// validateWindowsTerminal124Subset checks the structural surface OVAV projects.
// It is intentionally not represented as Microsoft's complete vendor schema.
func validateWindowsTerminal124Subset(settings map[string]interface{}, requireLocalPairs bool) error {
	profiles, err := optionalObject(settings, "profiles")
	if err != nil {
		return err
	}
	if profiles == nil {
		return fmt.Errorf("profiles is required and must be an object")
	}
	if profiles != nil {
		if _, err := optionalObject(profiles, "defaults"); err != nil {
			return fmt.Errorf("profiles.%w", err)
		}
		list, err := optionalArray(profiles, "list")
		if err != nil {
			return fmt.Errorf("profiles.%w", err)
		}
		for i, raw := range list {
			profile, ok := raw.(map[string]interface{})
			if !ok {
				return fmt.Errorf("profiles.list[%d] must be an object", i)
			}
			if requireLocalPairs {
				if err := requiredString(profile, "name"); err != nil {
					return fmt.Errorf("profiles.list[%d].%w", i, err)
				}
				if _, found := profile["guid"]; !found {
					return fmt.Errorf("profiles.list[%d].guid is required", i)
				}
				if _, found := profile["commandline"]; !found {
					return fmt.Errorf("profiles.list[%d].commandline is required", i)
				}
			}
			if guid, found := profile["guid"]; found {
				value, ok := guid.(string)
				if !ok || !windowsTerminalGUID.MatchString(value) {
					return fmt.Errorf("profiles.list[%d].guid must be a valid GUID in braces", i)
				}
			}
			if command, found := profile["commandline"]; found {
				if value, ok := command.(string); !ok || strings.TrimSpace(value) == "" {
					return fmt.Errorf("profiles.list[%d].commandline must be a string", i)
				}
			}
		}
	}
	if value, found := settings["defaultProfile"]; found {
		guid, ok := value.(string)
		if !ok || !windowsTerminalGUID.MatchString(guid) {
			return fmt.Errorf("defaultProfile must be a valid GUID in braces")
		}
	}

	schemes, err := namedObjects(settings, "schemes")
	if err != nil {
		return err
	}
	themes, err := namedObjects(settings, "themes")
	if err != nil {
		return err
	}
	if err := validatePairedReference(settings["theme"], "theme", themes, requireLocalPairs); err != nil {
		return err
	}
	if profiles != nil {
		defaults, _ := profiles["defaults"].(map[string]interface{})
		if err := validatePairedReference(defaults["colorScheme"], "profiles.defaults.colorScheme", schemes, requireLocalPairs); err != nil {
			return err
		}
	}
	for _, key := range []string{"actions", "keybindings"} {
		actions, err := optionalArray(settings, key)
		if err != nil {
			return err
		}
		for i, raw := range actions {
			action, ok := raw.(map[string]interface{})
			if !ok {
				return fmt.Errorf("%s[%d] must be an object", key, i)
			}
			command, found := action["command"]
			if !found && key == "keybindings" {
				if err := requiredString(action, "id"); err != nil {
					return fmt.Errorf("keybindings[%d] requires command or id", i)
				}
				continue
			}
			if !found {
				return fmt.Errorf("actions[%d].command is required", i)
			}
			switch value := command.(type) {
			case string:
				if strings.TrimSpace(value) == "" {
					return fmt.Errorf("%s[%d].command must not be empty", key, i)
				}
			case map[string]interface{}:
				if err := requiredString(value, "action"); err != nil {
					return fmt.Errorf("%s[%d].command.%w", key, i, err)
				}
			default:
				return fmt.Errorf("%s[%d].command must be a string or object", key, i)
			}
		}
	}
	return nil
}

func optionalObject(parent map[string]interface{}, key string) (map[string]interface{}, error) {
	raw, found := parent[key]
	if !found {
		return nil, nil
	}
	value, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%s must be an object", key)
	}
	return value, nil
}

func optionalArray(parent map[string]interface{}, key string) ([]interface{}, error) {
	raw, found := parent[key]
	if !found {
		return nil, nil
	}
	value, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("%s must be an array", key)
	}
	return value, nil
}

func namedObjects(settings map[string]interface{}, key string) (map[string]struct{}, error) {
	items, err := optionalArray(settings, key)
	if err != nil {
		return nil, err
	}
	names := make(map[string]struct{}, len(items))
	for i, raw := range items {
		item, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("%s[%d] must be an object", key, i)
		}
		if err := requiredString(item, "name"); err != nil {
			return nil, fmt.Errorf("%s[%d].%w", key, i, err)
		}
		name := item["name"].(string)
		if _, duplicate := names[name]; duplicate {
			return nil, fmt.Errorf("%s contains duplicate name %q", key, name)
		}
		names[name] = struct{}{}
	}
	return names, nil
}

func requiredString(parent map[string]interface{}, key string) error {
	value, ok := parent[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s must be a non-empty string", key)
	}
	return nil
}

func validatePairedReference(raw interface{}, field string, names map[string]struct{}, requireLocalPairs bool) error {
	if raw == nil {
		return nil
	}
	if value, ok := raw.(string); ok {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must not be empty", field)
		}
		return nil
	}
	pair, ok := raw.(map[string]interface{})
	if !ok {
		return fmt.Errorf("%s must be a string or light/dark object", field)
	}
	for _, mode := range []string{"light", "dark"} {
		value, ok := pair[mode].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must pair light and dark names", field)
		}
		if requireLocalPairs {
			if _, found := names[value]; !found {
				kind := "scheme"
				if field == "theme" {
					kind = "theme"
				}
				return fmt.Errorf("%s.%s references unknown %s %q", field, mode, kind, value)
			}
		}
	}
	return nil
}

func mergeObject(base, overlay map[string]interface{}) map[string]interface{} {
	result := cloneObject(base)
	for key, incoming := range overlay {
		switch key {
		case "schemes", "themes":
			result[key] = mergeNamedArray(asArray(result[key]), asArray(incoming))
		case "actions", "keybindings":
			result[key] = mergeActionArray(asArray(result[key]), asArray(incoming))
		case "profiles":
			result[key] = mergeProfiles(asObject(result[key]), asObject(incoming))
		default:
			current, currentOK := result[key].(map[string]interface{})
			next, nextOK := incoming.(map[string]interface{})
			if currentOK && nextOK {
				result[key] = mergeObject(current, next)
			} else {
				result[key] = incoming
			}
		}
	}
	return result
}

func mergeActionArray(base, overlay []interface{}) []interface{} {
	return mergeArray(base, overlay, func(item map[string]interface{}) string {
		for _, key := range []string{"keys", "id", "name"} {
			if value, ok := item[key].(string); ok && value != "" {
				return key + ":" + value
			}
		}
		return ""
	})
}

func mergeProfiles(base, overlay map[string]interface{}) map[string]interface{} {
	result := mergeObject(base, overlay)
	result["list"] = mergeProfileArray(asArray(base["list"]), asArray(overlay["list"]))
	return result
}

func mergeNamedArray(base, overlay []interface{}) []interface{} {
	return mergeArray(base, overlay, func(item map[string]interface{}) string {
		value, _ := item["name"].(string)
		return value
	})
}

func mergeProfileArray(base, overlay []interface{}) []interface{} {
	return mergeArray(base, overlay, func(item map[string]interface{}) string {
		if value, ok := item["guid"].(string); ok {
			return value
		}
		value, _ := item["name"].(string)
		return value
	})
}

func mergeArray(base, overlay []interface{}, identity func(map[string]interface{}) string) []interface{} {
	result := append([]interface{}{}, base...)
	index := make(map[string]int)
	for i, raw := range result {
		if item, ok := raw.(map[string]interface{}); ok {
			if key := identity(item); key != "" {
				index[key] = i
			}
		}
	}
	for _, raw := range overlay {
		item, ok := raw.(map[string]interface{})
		if !ok {
			result = append(result, raw)
			continue
		}
		key := identity(item)
		if i, found := index[key]; found && key != "" {
			if existing, ok := result[i].(map[string]interface{}); ok {
				result[i] = mergeObject(existing, item)
			} else {
				result[i] = item
			}
			continue
		}
		index[key] = len(result)
		result = append(result, item)
	}
	return result
}

func cloneObject(value map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func asObject(value interface{}) map[string]interface{} {
	result, _ := value.(map[string]interface{})
	if result == nil {
		return map[string]interface{}{}
	}
	return result
}

func asArray(value interface{}) []interface{} {
	result, _ := value.([]interface{})
	return result
}

// DefaultWindowsTerminalDestination returns the installed-settings path for planning only.
func DefaultWindowsTerminalDestination(localAppData string) string {
	return filepath.Join(localAppData, "Packages", "Microsoft.WindowsTerminal_8wekyb3d8bbwe", "LocalState", "settings.json")
}
