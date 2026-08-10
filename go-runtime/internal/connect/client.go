// Package connect implements OVAV-CONNECT — direct API model connection.
// OVAV-CONNECT bypasses CLI's actor tool limitations by connecting
// directly to AI provider APIs (OpenAI-compatible).
//
// Architecture:
// - OVAV owns the prompt/routing/agent logic
// - CLI only handles terminal input/output
// - No dependency on actor tool whitelist restrictions
package connect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/runtime"
)

// Provider types
type Provider string

const (
	ProviderOpenAI     Provider = "openai"
	ProviderAnthropic  Provider = "anthropic"
	ProviderOpenRouter Provider = "openrouter"
	ProviderAzure      Provider = "azure"
	ProviderMiniMax    Provider = "minimax"
)

// Config holds OVAV-CONNECT configuration
type Config struct {
	Provider   Provider `json:"provider"`
	APIKey     string   `json:"api_key"`
	BaseURL    string   `json:"base_url"`
	Model      string   `json:"model"`
	MaxTokens  int      `json:"max_tokens"`
	TimeoutSec int      `json:"timeout_sec"`
}

// LoadConfig loads OVAV-CONNECT config from environment
func LoadConfig() *Config {
	cfg := &Config{
		Provider:   ProviderOpenAI,
		Model:      getEnv("OVAV_MODEL", "gpt-4o"),
		MaxTokens:  4096,
		TimeoutSec: 120,
	}

	// Detect provider from environment
	if key := os.Getenv("MINIMAX_API_KEY"); key != "" {
		cfg.Provider = ProviderMiniMax
		cfg.APIKey = key
		cfg.BaseURL = getEnv("MINIMAX_BASE_URL", "https://api.minimaxi.chat/v1")
		cfg.Model = getEnv("MINIMAX_MODEL", "MiniMax-Text-01")
	} else if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		cfg.Provider = ProviderOpenAI
		cfg.APIKey = key
		cfg.BaseURL = getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1")
	} else if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		cfg.Provider = ProviderAnthropic
		cfg.APIKey = key
		cfg.BaseURL = getEnv("ANTHROPIC_BASE_URL", "https://api.anthropic.com/v1")
	} else if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		cfg.Provider = ProviderOpenRouter
		cfg.APIKey = key
		cfg.BaseURL = getEnv("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1")
	}

	// Allow override via OVAV_PROVIDER, OVAV_API_KEY, OVAV_BASE_URL
	if p := os.Getenv("OVAV_PROVIDER"); p != "" {
		cfg.Provider = Provider(p)
	}
	if k := os.Getenv("OVAV_API_KEY"); k != "" {
		cfg.APIKey = k
	}
	if u := os.Getenv("OVAV_BASE_URL"); u != "" {
		cfg.BaseURL = u
	}
	if m := os.Getenv("OVAV_MODEL"); m != "" {
		cfg.Model = m
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Request represents an API chat request
type Request struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	System      string    `json:"system,omitempty"`
}

// Response represents an API chat response
type Response struct {
	Content string `json:"content"`
	Model   string `json:"model"`
	Usage   Usage  `json:"usage"`
}

// AnthropicResponse represents Anthropic API response
type AnthropicResponse struct {
	Content string `json:"content"`
	Model   string `json:"model"`
	Usage   Usage  `json:"usage"`
}

// Usage represents token usage
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Client is the OVAV-CONNECT API client
type Client struct {
	Config *Config
	client *http.Client
}

// NewClient creates a new OVAV-CONNECT client
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = LoadConfig()
	}
	if cfg.TimeoutSec == 0 {
		cfg.TimeoutSec = 120
	}
	return &Client{
		Config: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSec) * time.Second,
		},
	}
}

// Invoke sends a task to the configured API and returns the response
func (c *Client) Invoke(agentID string, task string) (*Response, error) {
	// Build system prompt from agent profile
	systemPrompt := c.BuildSystemPrompt(agentID)

	// Build messages
	var messages []Message
	if systemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}
	messages = append(messages, Message{
		Role:    "user",
		Content: task,
	})

	// Make request based on provider
	switch c.Config.Provider {
	case ProviderAnthropic:
		return c.invokeAnthropic(systemPrompt, task)
	default:
		return c.invokeOpenAI(messages)
	}
}

// BuildSystemPrompt builds the system prompt from agent profile
func (c *Client) BuildSystemPrompt(agentID string) string {
	// Get delegation payload which includes agent profile
	payload, err := runtime.BuildDelegationPayload(agentID, "system-prompt-generation")
	if err != nil || payload.Profile.SystemPrompt == "" {
		return c.defaultSystemPrompt(agentID)
	}

	// Extract just the agent content from the YAML frontmatter
	content := payload.Profile.SystemPrompt

	// Add OVAV runtime instructions
	ovavInstructions := `
## OVAV Runtime Context
- You are operating as part of OVAV AGENTS system
- Current workspace: ` + payload.Workspace.WorktreeRoot + `
- Git branch: ` + payload.Workspace.Branch + ` (` + payload.Workspace.Head + `)
- Workspace status: ` + payload.Workspace.Status + `
- Current time: ` + time.Now().UTC().Format(time.RFC3339) + `Z

## Response Requirements
- Respond in Spanish unless user requests otherwise
- Keep responses compact (50-150 words)
- Use icons and tables for visual hierarchy
- Result first, explanation after
`

	return content + "\n" + ovavInstructions
}

// defaultSystemPrompt returns a fallback system prompt
func (c *Client) defaultSystemPrompt(agentID string) string {
	area := runtime.DetectServiceArea(agentID)
	lead := runtime.LeadForArea(area)

	return fmt.Sprintf(`You are %s, operating within the OVAV AGENTS system.

## Your Role
- Service Area: %s
- Lead: %s
- You are an expert in your domain with full access to OVAV governance tools

## Operating Context
- Always respond in Spanish
- Use compact format (50-150 words)
- Result first, explanation after
- Use icons and tables for visual hierarchy

## OVAV AGENTS
OVAV AGENTS is an advanced multi-agent system with 10 service areas,
each led by world-class experts. The system uses governanance gates,
harnesses, and automated delegation to route work efficiently.

## Critical Rules
- NEW IDEA without PLAN → ask "¿Querés que armemos el plan?"
- MULTI-AREA projects → coordinate with relevant leads
- Output language: Spanish only
`, agentID, area, lead)
}

// invokeOpenAI makes an OpenAI-compatible API request
func (c *Client) invokeOpenAI(messages []Message) (*Response, error) {
	reqBody := Request{
		Model:       c.Config.Model,
		Messages:    messages,
		MaxTokens:   c.Config.MaxTokens,
		Temperature: 0.7,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.Config.BaseURL
	if !strings.HasSuffix(url, "/v1/chat/completions") {
		url = strings.TrimSuffix(url, "/") + "/v1/chat/completions"
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Config.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
		Model string `json:"model"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices")
	}

	return &Response{
		Content: apiResp.Choices[0].Message.Content,
		Model:   apiResp.Model,
		Usage: Usage{
			InputTokens:  apiResp.Usage.PromptTokens,
			OutputTokens: apiResp.Usage.CompletionTokens,
		},
	}, nil
}

// invokeAnthropic makes an Anthropic API request
func (c *Client) invokeAnthropic(system, task string) (*Response, error) {
	reqBody := map[string]interface{}{
		"model": c.Config.Model,
		"messages": []map[string]string{
			{"role": "user", "content": task},
		},
		"max_tokens":  c.Config.MaxTokens,
		"temperature": 0.7,
	}
	if system != "" {
		reqBody["system"] = system
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.Config.BaseURL
	if !strings.HasSuffix(url, "/v1/messages") {
		url = strings.TrimSuffix(url, "/") + "/v1/messages"
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.Config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var apiResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &Response{
		Content: apiResp.Content,
		Model:   apiResp.Model,
	}, nil
}

// IsConfigured returns true if OVAV-CONNECT is properly configured
func (c *Client) IsConfigured() bool {
	return c.Config.APIKey != "" && c.Config.BaseURL != ""
}
