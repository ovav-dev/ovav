package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// AnthropicProvider implements Anthropic API integration.
type AnthropicProvider struct {
	APIKey string
}

// NewAnthropic creates a new Anthropic provider.
func NewAnthropic(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{APIKey: apiKey}
}

// Name returns the provider name.
func (p *AnthropicProvider) Name() string {
	return "Anthropic"
}

// Type returns the provider type.
func (p *AnthropicProvider) Type() string {
	return "anthropic"
}

// FetchUsage fetches current usage from Anthropic API.
func (p *AnthropicProvider) FetchUsage() (*ProviderUsage, error) {
	// Anthropic doesn't have a public usage API, so we return nil
	// The actual token tracking would come from the conversation logs
	return &ProviderUsage{
		Timestamp:   time.Now(),
		TotalTokens: 0,
	}, nil
}

// FetchBalance fetches current account balance.
func (p *AnthropicProvider) FetchBalance() (*BalanceInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", "https://api.anthropic.com/v1/users/self", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch balance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &BalanceInfo{
			Provider:  p.Name(),
			Available: 0,
			Error:     fmt.Sprintf("HTTP %d", resp.StatusCode),
		}, nil
	}

	var userResp struct {
		Subscription struct {
			Plan struct {
				Name string `json:"name"`
			} `json:"plan"`
		} `json:"subscription"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &BalanceInfo{
		Provider: p.Name(),
		PlanName: userResp.Subscription.Plan.Name,
	}, nil
}
