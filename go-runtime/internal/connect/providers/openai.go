// Package providers implements provider-specific API integrations.
package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// UsageResponse represents OpenAI usage response.
type UsageResponse struct {
	TotalTokens int `json:"total_tokens"`
}

// BalanceResponse represents OpenAI balance response.
type BalanceResponse struct {
	TotalGranted   float64 `json:"total_granted"`
	TotalUsed      float64 `json:"total_used"`
	TotalAvailable float64 `json:"total_available"`
}

// OpenAIProvider implements OpenAI API integration.
type OpenAIProvider struct {
	APIKey string
}

// NewOpenAI creates a new OpenAI provider.
func NewOpenAI(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{APIKey: apiKey}
}

// Name returns the provider name.
func (p *OpenAIProvider) Name() string {
	return "OpenAI"
}

// Type returns the provider type.
func (p *OpenAIProvider) Type() string {
	return "openai"
}

// FetchUsage fetches current usage from OpenAI API.
func (p *OpenAIProvider) FetchUsage() (*ProviderUsage, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Fetch organization usage
	req, err := http.NewRequest("GET", "https://api.openai.com/v1/usage", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch usage: %w", err)
	}
	defer resp.Body.Close()

	usage := &ProviderUsage{
		Timestamp: time.Now(),
	}

	if resp.StatusCode == http.StatusOK {
		var usageResp UsageResponse
		if err := json.NewDecoder(resp.Body).Decode(&usageResp); err == nil {
			usage.TotalTokens = usageResp.TotalTokens
		}
	}

	return usage, nil
}

// FetchBalance fetches current account balance.
func (p *OpenAIProvider) FetchBalance() (*BalanceInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", "https://api.openai.com/v1/dashboard/billing CreditGrants", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch balance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &BalanceInfo{
			Provider: p.Name(),
			Available: 0,
			Error: fmt.Sprintf("HTTP %d", resp.StatusCode),
		}, nil
	}

	var balanceResp BalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&balanceResp); err != nil {
		return nil, fmt.Errorf("parse balance: %w", err)
	}

	return &BalanceInfo{
		Provider:  p.Name(),
		Granted:  balanceResp.TotalGranted,
		Used:     balanceResp.TotalUsed,
		Available: balanceResp.TotalAvailable,
	}, nil
}

// ProviderUsage holds provider usage data.
type ProviderUsage struct {
	Timestamp   time.Time
	TotalTokens int
}

// BalanceInfo holds balance information.
type BalanceInfo struct {
	Provider   string
	Granted    float64
	Used       float64
	Available  float64
	PlanName   string
	Error      string
}
