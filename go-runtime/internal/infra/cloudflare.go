package infra

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// CFDNSRecord represents a Cloudflare DNS record.
type CFDNSRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	Proxied bool   `json:"proxied"`
	TTL     int    `json:"ttl"`
}

// CFTunnelConfig represents a Cloudflare tunnel configuration.
type CFTunnelConfig struct {
	TunnelID   string   `json:"tunnel_id"`
	TunnelName string   `json:"tunnel_name"`
	Hostnames  []string `json:"hostnames"`
}

// cfAPI is the base URL for Cloudflare API v4.
// Var (not const) to allow test override via httptest.NewServer.
var cfAPI = "https://api.cloudflare.com/client/v4"

// cfCredentials holds Cloudflare authentication material.
type cfCredentials struct {
	apiToken string // Bearer token (API Token)
	apiKey   string // X-Auth-Key (Global API Key)
	email    string // X-Auth-Email (required for Global API Key)
}

// loadCFCredentials reads Cloudflare credentials from vault.
// Supports both API Token (Bearer) and Global API Key (X-Auth-Key + X-Auth-Email).
func loadCFCredentials(vaultDir string) (*cfCredentials, error) {
	c := &cfCredentials{}

	if data, err := os.ReadFile(filepath.Join(vaultDir, "CF_API_TOKEN")); err == nil {
		c.apiToken = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile(filepath.Join(vaultDir, "CF_API_KEY")); err == nil {
		c.apiKey = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile(filepath.Join(vaultDir, "CF_EMAIL")); err == nil {
		c.email = strings.TrimSpace(string(data))
	}

	if c.apiToken == "" && c.apiKey == "" {
		return nil, fmt.Errorf("no Cloudflare credentials in vault — set CF_API_TOKEN or CF_API_KEY + CF_EMAIL")
	}
	if c.apiKey != "" && c.email == "" {
		return nil, fmt.Errorf("CF_API_KEY requires CF_EMAIL in vault")
	}

	return c, nil
}

// cfCall makes an authenticated request to the Cloudflare API.
// Prefers API Token (Bearer) if available, falls back to Global API Key.
func cfCall(method, path string, body io.Reader, vaultDir string) ([]byte, error) {
	creds, err := loadCFCredentials(vaultDir)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, cfAPI+path, body)
	if err != nil {
		return nil, err
	}

	if creds.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+creds.apiToken)
	} else {
		req.Header.Set("X-Auth-Email", creds.email)
		req.Header.Set("X-Auth-Key", creds.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Cloudflare API: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return data, fmt.Errorf("Cloudflare API %s %s: HTTP %d — %s", method, path, resp.StatusCode, string(data))
	}

	return data, nil
}

// cfAPIVerify tests that the API credentials work.
func cfAPIVerify(token string) error {
	// Try Bearer first
	req, _ := http.NewRequest("GET", cfAPI+"/user/tokens/verify", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Cloudflare verify: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil
	}

	// Bearer failed — might be a Global API Key, try zone list as verification
	req2, _ := http.NewRequest("GET", cfAPI+"/zones?per_page=1", nil)
	req2.Header.Set("X-Auth-Email", os.Getenv("CF_EMAIL"))
	req2.Header.Set("X-Auth-Key", token)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		return fmt.Errorf("Cloudflare verify: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("Cloudflare verify failed: Bearer=%d, GlobalKey=%d", resp.StatusCode, resp2.StatusCode)
}

// DNSResult holds the outcome of a DNS operation.
type DNSResult struct {
	ZoneID string
	Record CFDNSRecord
	Action string // "deleted", "found", "not_found", "error"
	Error  string
}

// DeleteDNSRecord removes a DNS record by name from a zone.
func DeleteDNSRecord(zoneName, recordName, vaultDir string) (DNSResult, error) {
	// Step 1: Get zone ID
	zoneID, err := getZoneID(zoneName, vaultDir)
	if err != nil {
		return DNSResult{Action: "error", Error: err.Error()}, err
	}

	// Step 2: List DNS records matching the name
	records, err := listDNSRecords(zoneID, recordName, vaultDir)
	if err != nil {
		return DNSResult{ZoneID: zoneID, Action: "error", Error: err.Error()}, err
	}

	if len(records) == 0 {
		return DNSResult{ZoneID: zoneID, Action: "not_found",
			Error: fmt.Sprintf("no DNS record found for %q", recordName)}, nil
	}

	// Step 3: Delete each matching record
	var lastResult DNSResult
	for _, rec := range records {
		path := fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, rec.ID)
		data, err := cfCall("DELETE", path, nil, vaultDir)
		if err != nil {
			return DNSResult{ZoneID: zoneID, Record: rec, Action: "error", Error: err.Error()}, err
		}

		var result struct {
			Success bool `json:"success"`
		}
		json.Unmarshal(data, &result)

		if result.Success {
			lastResult = DNSResult{ZoneID: zoneID, Record: rec, Action: "deleted"}
		} else {
			lastResult = DNSResult{ZoneID: zoneID, Record: rec, Action: "error",
				Error: fmt.Sprintf("API returned success=false: %s", string(data))}
		}
	}

	return lastResult, nil
}

// ListDNSRecords returns all DNS records for a zone.
func ListDNSRecords(zoneName, vaultDir string) ([]CFDNSRecord, error) {
	zoneID, err := getZoneID(zoneName, vaultDir)
	if err != nil {
		return nil, err
	}
	return listDNSRecords(zoneID, "", vaultDir)
}

// CheckDNSRecord verifies if a DNS record exists and returns its details.
func CheckDNSRecord(zoneName, recordName, vaultDir string) (DNSResult, error) {
	zoneID, err := getZoneID(zoneName, vaultDir)
	if err != nil {
		return DNSResult{Action: "error", Error: err.Error()}, err
	}

	records, err := listDNSRecords(zoneID, recordName, vaultDir)
	if err != nil {
		return DNSResult{ZoneID: zoneID, Action: "error", Error: err.Error()}, err
	}

	if len(records) == 0 {
		return DNSResult{ZoneID: zoneID, Action: "not_found",
			Error: fmt.Sprintf("%q → NXDOMAIN (good — already removed)", recordName)}, nil
	}

	return DNSResult{ZoneID: zoneID, Record: records[0], Action: "found",
		Error: fmt.Sprintf("%q → %s (proxied=%v)", records[0].Name, records[0].Content, records[0].Proxied)}, nil
}

// getZoneID resolves a zone name to its Cloudflare zone ID.
func getZoneID(zoneName, vaultDir string) (string, error) {
	data, err := cfCall("GET", fmt.Sprintf("/zones?name=%s", zoneName), nil, vaultDir)
	if err != nil {
		return "", err
	}

	var result struct {
		Result []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parse zone response: %w", err)
	}

	if len(result.Result) == 0 {
		return "", fmt.Errorf("zone %q not found", zoneName)
	}

	return result.Result[0].ID, nil
}

// listDNSRecords returns DNS records for a zone, optionally filtered by name.
func listDNSRecords(zoneID, nameFilter, vaultDir string) ([]CFDNSRecord, error) {
	path := fmt.Sprintf("/zones/%s/dns_records?per_page=100", zoneID)
	if nameFilter != "" {
		path += "&name=" + nameFilter
	}

	data, err := cfCall("GET", path, nil, vaultDir)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result []CFDNSRecord `json:"result"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse DNS records: %w", err)
	}

	return result.Result, nil
}

// TunnelInfo represents a Cloudflare tunnel with its hostnames.
type TunnelInfo struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Hostnames []string `json:"hostnames"`
}

// ListTunnels returns all tunnels for the account.
func ListTunnels(vaultDir string) ([]TunnelInfo, error) {
	accountID, err := getAccountID(vaultDir)
	if err != nil {
		return nil, err
	}

	data, err := cfCall("GET", fmt.Sprintf("/accounts/%s/cfd_tunnel", accountID), nil, vaultDir)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse tunnels: %w", err)
	}

	var tunnels []TunnelInfo
	for _, t := range result.Result {
		info := TunnelInfo{ID: t.ID, Name: t.Name}
		// Get tunnel config to extract hostnames
		cfg, _ := cfCall("GET", fmt.Sprintf("/accounts/%s/cfd_tunnel/%s/configurations", accountID, t.ID), nil, vaultDir)
		if cfg != nil {
			var cfgResult struct {
				Result struct {
					Config struct {
						Ingress []struct {
							Hostname string `json:"hostname"`
							Service  string `json:"service"`
						} `json:"ingress"`
					} `json:"config"`
				} `json:"result"`
			}
			if json.Unmarshal(cfg, &cfgResult) == nil {
				for _, ing := range cfgResult.Result.Config.Ingress {
					if ing.Hostname != "" {
						info.Hostnames = append(info.Hostnames, ing.Hostname)
					}
				}
			}
		}
		tunnels = append(tunnels, info)
	}

	return tunnels, nil
}

// VerifyTunnelHostname checks if a hostname is in the tunnel ingress and returns findings.
func VerifyTunnelHostname(hostname, vaultDir string) (string, error) {
	tunnels, err := ListTunnels(vaultDir)
	if err != nil {
		return "", fmt.Errorf("list tunnels: %w", err)
	}

	for _, t := range tunnels {
		for _, h := range t.Hostnames {
			if h == hostname {
				return fmt.Sprintf("FOUND in tunnel %q (%s)", t.Name, t.ID), nil
			}
		}
	}

	return fmt.Sprintf("NOT in any tunnel (good — %q is clean)", hostname), nil
}

// getAccountID returns the Cloudflare account ID from vault or API.
func getAccountID(vaultDir string) (string, error) {
	// Check vault first
	data, err := os.ReadFile(filepath.Join(vaultDir, "CF_ACCOUNT_ID"))
	if err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	// Try to get from API
	apiData, err := cfCall("GET", "/accounts?per_page=1", nil, vaultDir)
	if err != nil {
		return "", err
	}

	var result struct {
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(apiData, &result); err != nil {
		return "", fmt.Errorf("parse accounts: %w", err)
	}
	if len(result.Result) == 0 {
		return "", fmt.Errorf("no Cloudflare accounts found")
	}

	return result.Result[0].ID, nil
}
