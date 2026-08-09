// Package tracker implements OVAV's token usage tracking.
package tracker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ovav/ovav/internal/connect/providers"
	"gopkg.in/yaml.v3"
)

// TrackedProvider represents a provider being tracked.
type TrackedProvider struct {
	ID        string    `yaml:"id"`
	Type      string    `yaml:"type"`
	APIKey    string    `yaml:"api_key,omitempty"`
	Enabled   bool      `yaml:"enabled"`
	AddedAt   time.Time `yaml:"added_at"`
	LastCheck time.Time `yaml:"last_check"`
}

// UsageRecord represents a single usage record.
type UsageRecord struct {
	ID          string    `yaml:"id"`
	ProviderID  string    `yaml:"provider_id"`
	Model       string    `yaml:"model"`
	InputTokens int       `yaml:"input_tokens"`
	OutputTokens int      `yaml:"output_tokens"`
	TotalTokens int       `yaml:"total_tokens"`
	CostUSD     float64   `yaml:"cost_usd"`
	Timestamp   time.Time `yaml:"timestamp"`
}

// UsageSummary represents aggregated usage for a period.
type UsageSummary struct {
	ProviderID   string    `yaml:"provider_id"`
	Period       string    `yaml:"period"`
	StartDate    time.Time `yaml:"start_date"`
	EndDate      time.Time `yaml:"end_date"`
	TotalCalls   int       `yaml:"total_calls"`
	TotalInput   int       `yaml:"total_input_tokens"`
	TotalOutput  int       `yaml:"total_output_tokens"`
	TotalTokens  int       `yaml:"total_tokens"`
	TotalCostUSD float64   `yaml:"total_cost_usd"`
}

// Tracker manages token usage tracking across providers.
type Tracker struct {
	dataDir   string
	providers map[string]*TrackedProvider
}

// New creates a new usage tracker.
func New(dataDir string) *Tracker {
	if dataDir == "" {
		dataDir = ".ovav/connect"
	}
	return &Tracker{
		dataDir:   dataDir,
		providers: make(map[string]*TrackedProvider),
	}
}

// LoadProviders loads providers from disk.
func (t *Tracker) LoadProviders() error {
	providersFile := filepath.Join(t.dataDir, "providers.yaml")
	data, err := os.ReadFile(providersFile)
	if err != nil {
		return nil // No providers yet
	}

	var providers []*TrackedProvider
	if err := yaml.Unmarshal(data, &providers); err != nil {
		return fmt.Errorf("parse providers: %w", err)
	}

	for _, p := range providers {
		t.providers[p.ID] = p
	}

	return nil
}

// SaveProviders saves providers to disk.
func (t *Tracker) SaveProviders() error {
	if err := os.MkdirAll(t.dataDir, 0755); err != nil {
		return fmt.Errorf("mkdir data dir: %w", err)
	}

	var providers []*TrackedProvider
	for _, p := range t.providers {
		providers = append(providers, p)
	}

	data, err := yaml.Marshal(providers)
	if err != nil {
		return fmt.Errorf("marshal providers: %w", err)
	}

	providersFile := filepath.Join(t.dataDir, "providers.yaml")
	return os.WriteFile(providersFile, data, 0644)
}

// AddProvider adds a new provider connection.
func (t *Tracker) AddProvider(provider *TrackedProvider) error {
	if provider.ID == "" {
		provider.ID = provider.Type + "-" + time.Now().Format("20060102")
	}
	provider.AddedAt = time.Now()
	t.providers[provider.ID] = provider
	return t.SaveProviders()
}

// ListProviders returns all configured providers.
func (t *Tracker) ListProviders() []*TrackedProvider {
	var result []*TrackedProvider
	for _, p := range t.providers {
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].AddedAt.Before(result[j].AddedAt)
	})
	return result
}

// RemoveProvider removes a provider by ID.
func (t *Tracker) RemoveProvider(id string) error {
	delete(t.providers, id)
	return t.SaveProviders()
}

// RecordUsage records a usage event.
func (t *Tracker) RecordUsage(record *UsageRecord) error {
	if err := os.MkdirAll(t.dataDir, 0755); err != nil {
		return fmt.Errorf("mkdir data dir: %w", err)
	}

	dateStr := record.Timestamp.Format("2006-01-02")
	usageDir := filepath.Join(t.dataDir, "usage", record.ProviderID, dateStr)
	if err := os.MkdirAll(usageDir, 0755); err != nil {
		return fmt.Errorf("mkdir usage dir: %w", err)
	}

	filename := filepath.Join(usageDir, fmt.Sprintf("%s.yaml", record.ID))
	data, err := yaml.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// GetUsageHistory returns usage records for a provider.
func (t *Tracker) GetUsageHistory(providerID string, since time.Time) ([]UsageRecord, error) {
	var records []UsageRecord

	usageDir := filepath.Join(t.dataDir, "usage", providerID)
	if _, err := os.Stat(usageDir); os.IsNotExist(err) {
		return records, nil
	}

	entries, err := os.ReadDir(usageDir)
	if err != nil {
		return nil, fmt.Errorf("read usage dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dateStr := entry.Name()
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil || date.Before(since) {
			continue
		}

		dayDir := filepath.Join(usageDir, dateStr)
		files, _ := os.ReadDir(dayDir)
		for _, f := range files {
			data, err := os.ReadFile(filepath.Join(dayDir, f.Name()))
			if err != nil {
				continue
			}
			var record UsageRecord
			if err := yaml.Unmarshal(data, &record); err != nil {
				continue
			}
			records = append(records, record)
		}
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp.After(records[j].Timestamp)
	})

	return records, nil
}

// GetTodayUsage returns today's usage summary.
func (t *Tracker) GetTodayUsage(providerID string) (*UsageSummary, error) {
	today := time.Now().Truncate(24 * time.Hour)
	records, err := t.GetUsageHistory(providerID, today)
	if err != nil {
		return nil, err
	}

	summary := &UsageSummary{
		ProviderID: providerID,
		Period:     "daily",
		StartDate:  today,
		EndDate:    today.Add(24 * time.Hour),
	}

	for _, r := range records {
		summary.TotalCalls++
		summary.TotalInput += r.InputTokens
		summary.TotalOutput += r.OutputTokens
		summary.TotalTokens += r.TotalTokens
		summary.TotalCostUSD += r.CostUSD
	}

	return summary, nil
}

// GetMonthUsage returns this month's usage summary.
func (t *Tracker) GetMonthUsage(providerID string) (*UsageSummary, error) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	records, err := t.GetUsageHistory(providerID, monthStart)
	if err != nil {
		return nil, err
	}

	summary := &UsageSummary{
		ProviderID: providerID,
		Period:     "monthly",
		StartDate:  monthStart,
		EndDate:    now,
	}

	for _, r := range records {
		summary.TotalCalls++
		summary.TotalInput += r.InputTokens
		summary.TotalOutput += r.OutputTokens
		summary.TotalTokens += r.TotalTokens
		summary.TotalCostUSD += r.CostUSD
	}

	return summary, nil
}

// GetProvider returns a provider by ID.
func (t *Tracker) GetProvider(id string) *TrackedProvider {
	return t.providers[id]
}

// CreateProviderClient creates an API client for a provider.
func CreateProviderClient(provider *TrackedProvider) (ProviderClient, error) {
	switch provider.Type {
	case "openai":
		return providers.NewOpenAI(provider.APIKey), nil
	case "anthropic":
		return providers.NewAnthropic(provider.APIKey), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", provider.Type)
	}
}

// ProviderClient is the interface for provider API clients.
type ProviderClient interface {
	Name() string
	Type() string
	FetchUsage() (*providers.ProviderUsage, error)
	FetchBalance() (*providers.BalanceInfo, error)
}

// Record represents a parsed API response for recording.
type Record struct {
	Model        string
	InputTokens  int
	OutputTokens int
}

// CalculateCost estimates cost based on model and tokens.
func CalculateCost(model string, inputTokens, outputTokens int) float64 {
	// Simple cost estimation based on model
	costPer1M := map[string]float64{
		"gpt-4o":         5.00,
		"gpt-4o-mini":     0.15,
		"gpt-4-turbo":    10.00,
		"gpt-3.5-turbo":  0.50,
		"claude-opus-3":   15.00,
		"claude-sonnet-3": 3.00,
		"claude-haiku-3":  0.25,
	}

	costPerM := costPer1M["default"]
	if c, ok := costPer1M[model]; ok {
		costPerM = c
	}

	totalTokens := inputTokens + outputTokens
	return (float64(totalTokens) / 1_000_000.0) * costPerM
}

// MarshalJSON implements json.Marshaler for UsageRecord.
func (r UsageRecord) MarshalJSON() ([]byte, error) {
	type Alias UsageRecord
	return json.Marshal(&struct {
		Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     Alias(r),
		Timestamp: r.Timestamp.Format(time.RFC3339),
	})
}
