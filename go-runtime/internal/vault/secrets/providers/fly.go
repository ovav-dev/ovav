// providers/fly.go — Fly.io Secrets discovery via flyctl.
//
// flyctl allows reading secret values directly (unlike GitHub).
// This provider discovers AND reads values (if running on a machine
// where flyctl is authenticated with access to the app).
package providers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ovav/ovav/internal/vault/secrets/types"
)

// FlyDiscoveryResult holds a discovered Fly.io secret.
type FlyDiscoveryResult struct {
	Name    string
	Value   []byte // raw value — treat as sensitive
	Type    types.SecretType
	AppName string
	Source  string // "fly_secrets"
}

// DiscoverFlySecrets lists all secrets for a Fly.io app via `fly secrets list`.
// Values ARE readable via flyctl.
func DiscoverFlySecrets(appName string) ([]FlyDiscoveryResult, error) {
	cmd := exec.Command("fly", "secrets", "list", "-a", appName, "--json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("fly secrets list: %w", err)
	}

	var raw []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parsing fly secrets: %w", err)
	}

	results := make([]FlyDiscoveryResult, 0, len(raw))
	for _, s := range raw {
		results = append(results, FlyDiscoveryResult{
			Name:    s.Key,
			Value:   []byte(s.Value),
			Type:    types.InferType(s.Key),
			AppName: appName,
			Source:  "fly_secrets",
		})
	}

	return results, nil
}

// DiscoverAllFlyApps discovers secrets from all Fly.io apps
// accessible to the authenticated flyctl context.
func DiscoverAllFlyApps() (map[string][]FlyDiscoveryResult, error) {
	cmd := exec.Command("fly", "apps", "list", "--json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("fly apps list: %w", err)
	}

	var apps []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(out, &apps); err != nil {
		return nil, fmt.Errorf("parsing fly apps: %w", err)
	}

	all := make(map[string][]FlyDiscoveryResult)
	for _, app := range apps {
		secrets, err := DiscoverFlySecrets(app.Name)
		if err != nil {
			continue
		}
		if len(secrets) > 0 {
			all[app.Name] = secrets
		}
	}

	return all, nil
}

// DiscoverFlySecretsFromOutput parses `fly secrets list` raw output
// (when --json is not available or for compatibility).
func DiscoverFlySecretsFromOutput(output string) ([]FlyDiscoveryResult, error) {
	var results []FlyDiscoveryResult
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Format: NAME=VALUE
		idx := strings.Index(line, "=")
		if idx == -1 {
			continue
		}
		name := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		results = append(results, FlyDiscoveryResult{
			Name:   name,
			Value:  []byte(value),
			Type:   types.InferType(name),
			Source: "fly_secrets",
		})
	}
	return results, scanner.Err()
}
