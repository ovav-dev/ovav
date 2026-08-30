package hostsync

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func validateOpenCodeBootstrap(content []byte) error {
	var config struct {
		Schema string `json:"$schema"`
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return fmt.Errorf("decode schema-only OpenCode JSON: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return err
	}
	if config.Schema != "https://opencode.ai/config.json" {
		return errors.New("OpenCode bootstrap must contain only the official $schema URL")
	}
	return nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var trailing interface{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("OpenCode bootstrap contains trailing JSON content")
	}
	return nil
}

func validateWSLResourcePolicy(content []byte) error {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	if !strings.Contains(text, "natural full WSL stop") ||
		!strings.Contains(text, "MUST NOT automatically run wsl --shutdown or wsl --terminate") {
		return errors.New("WSL policy must document natural-stop activation and prohibit automatic shutdown/terminate")
	}
	want := map[string]map[string]string{
		"wsl2": {
			"memory": "4GB", "processors": "4", "swap": "4GB", "networkingMode": "mirrored",
			"dnsTunneling": "true", "autoProxy": "true", "firewall": "true",
		},
		"experimental": {"autoMemoryReclaim": "dropCache"},
	}
	got := make(map[string]map[string]string)
	section := ""
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			if _, ok := want[section]; !ok {
				return fmt.Errorf("unsupported WSL section %q", section)
			}
			if _, exists := got[section]; exists {
				return fmt.Errorf("duplicate WSL section %q", section)
			}
			got[section] = make(map[string]string)
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || section == "" {
			return fmt.Errorf("invalid WSL policy line %q", line)
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if _, ok := want[section][key]; !ok {
			return fmt.Errorf("unsupported WSL key %q in section %q", key, section)
		}
		if _, exists := got[section][key]; exists {
			return fmt.Errorf("duplicate WSL key %q", key)
		}
		got[section][key] = value
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan WSL policy: %w", err)
	}
	for sectionName, keys := range want {
		if len(got[sectionName]) != len(keys) {
			return fmt.Errorf("WSL section %q is incomplete", sectionName)
		}
		for key, value := range keys {
			if got[sectionName][key] != value {
				return fmt.Errorf("WSL %s.%s must equal %q", sectionName, key, value)
			}
		}
	}
	return nil
}

func validateWarpWSLTab(content []byte) error {
	top := make(map[string]string)
	pane := make(map[string]string)
	paneCount := 0
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == "[[panes]]" {
			paneCount++
			continue
		}
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("invalid Warp Tab Config line %q", line)
		}
		key, raw = strings.TrimSpace(key), strings.TrimSpace(raw)
		target := top
		if paneCount > 0 {
			target = pane
		}
		if _, exists := target[key]; exists {
			return fmt.Errorf("duplicate Warp key %q", key)
		}
		target[key] = raw
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan Warp Tab Config: %w", err)
	}
	if paneCount != 1 {
		return errors.New("Warp Tab Config must contain exactly one pane")
	}
	if err := validateWarpTop(top); err != nil {
		return err
	}
	return validateWarpPane(pane)
}

func validateWarpTop(values map[string]string) error {
	for key := range values {
		if key != "name" && key != "title" && key != "color" {
			return fmt.Errorf("unsupported Warp top-level key %q", key)
		}
	}
	name, err := quoted(values["name"])
	if err != nil || name == "" {
		return errors.New("Warp Tab Config requires a non-empty name")
	}
	if raw, ok := values["title"]; ok {
		if title, err := quoted(raw); err != nil || title == "" {
			return errors.New("Warp title must be a non-empty string")
		}
	}
	if raw, ok := values["color"]; ok {
		color, err := quoted(raw)
		if err != nil {
			return errors.New("Warp color must be a quoted official value")
		}
		valid := map[string]bool{"black": true, "red": true, "green": true, "yellow": true, "blue": true, "magenta": true, "cyan": true, "white": true}
		if !valid[color] {
			return fmt.Errorf("unsupported Warp color %q", color)
		}
	}
	return nil
}

func validateWarpPane(values map[string]string) error {
	for key := range values {
		if key != "id" && key != "type" && key != "directory" && key != "is_focused" {
			return fmt.Errorf("unsupported Warp pane key %q", key)
		}
	}
	id, idErr := quoted(values["id"])
	paneType, typeErr := quoted(values["type"])
	directory, directoryErr := quoted(values["directory"])
	if idErr != nil || id == "" || typeErr != nil || paneType != "terminal" || directoryErr != nil || directory != "~" {
		return errors.New("Warp pane must have an id, type=terminal, and directory=~")
	}
	if focused, ok := values["is_focused"]; ok && focused != "true" && focused != "false" {
		return errors.New("Warp is_focused must be boolean")
	}
	return nil
}

func quoted(raw string) (string, error) {
	if raw == "" {
		return "", errors.New("missing quoted value")
	}
	value, err := strconv.Unquote(raw)
	if err != nil {
		return "", err
	}
	return value, nil
}
