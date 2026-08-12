// provider_cli.go — OVAV Provider Switching CLI
//
// Provides runtime provider selection so users can choose between
// hyper credits, MiniMax Direct subscription, or any other provider
// without env var conflicts.
//
// Runtime config: ~/.ovav/provider.json
// Priority: runtime config > OVAV_* env vars > auto-detect

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ActiveProvider represents the current active provider selection
type ActiveProvider struct {
	Provider   string    `json:"provider"`
	Model      string    `json:"model"`
	APIKeyRef string    `json:"api_key_ref"` // Reference name, not the actual key
	APIKey    string    `json:"-"`          // Never persisted
	Endpoint  string    `json:"endpoint"`
	SetAt     time.Time `json:"set_at"`
	SetBy     string    `json:"set_by"` // "user" or "auto"
}

// ProviderConfig maps user-friendly names to provider settings
var ProviderConfigs = map[string]struct {
	Provider  string
	Model    string
	APIKeyEnv string
	Endpoint string
	Help     string
}{
	"hyper": {
		Provider:  "hyper",
		Model:     "deepseek-v4-flash",
		APIKeyEnv: "HYPER_API_KEY",
		Endpoint:  "https://hyper.charm.land",
		Help:      "Use hyper.charm.land credits (95 remaining)",
	},
	"minimax_direct": {
		Provider:  "minimax",
		Model:     "minimax/MiniMax-M2.7",
		APIKeyEnv: "ANTHROPIC_API_KEY",
		Endpoint:  "https://api.minimax.io/anthropic/v1",
		Help:      "Use your MiniMax monthly subscription (sk-cp-* key)",
	},
	"minimax_hyper": {
		Provider:  "aihubmix",
		Model:     "aihubmix/coding-minimax-m3",
		APIKeyEnv: "MINIMAX_API_KEY",
		Endpoint:  "https://api.minimaxi.chat/v1",
		Help:      "Use MiniMax via aihubmix (hyper proxy)",
	},
	"openai": {
		Provider:  "openai",
		Model:     "gpt-4o",
		APIKeyEnv: "OPENAI_API_KEY",
		Endpoint:  "https://api.openai.com/v1",
		Help:      "Use OpenAI Direct API",
	},
	"anthropic": {
		Provider:  "anthropic",
		Model:     "claude-sonnet-3",
		APIKeyEnv: "ANTHROPIC_API_KEY",
		Endpoint:  "https://api.anthropic.com/v1",
		Help:      "Use Anthropic Direct API (sk-ant-* key)",
	},
	"openrouter": {
		Provider:  "openrouter",
		Model:     "deepseek-v4-pro",
		APIKeyEnv: "OPENROUTER_API_KEY",
		Endpoint:  "https://openrouter.ai/api/v1",
		Help:      "Use OpenRouter multi-provider aggregation",
	},
}

// providerConfigPath returns the path to the runtime provider config file
func providerConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "~"
	}
	return filepath.Join(home, ".ovav", "provider.json")
}

// loadActiveProvider loads the current active provider from runtime config
func loadActiveProvider() (*ActiveProvider, error) {
	path := providerConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p ActiveProvider
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// saveActiveProvider saves the active provider to runtime config
func saveActiveProvider(p *ActiveProvider) error {
	path := providerConfigPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// Resolve API key from environment at save time
	// This ensures we store the actual key, not a reference
	cfg := ProviderConfigs[p.Provider]
	apiKey := os.Getenv(cfg.APIKeyEnv)
	if apiKey != "" {
		p.APIKey = apiKey // Store actual key (runtime reads this directly)
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// applyProviderEnv applies the provider settings to environment variables
// This makes CRUSH and other tools use the selected provider
func applyProviderEnv(p *ActiveProvider) error {
	cfg, ok := ProviderConfigs[p.Provider]
	if !ok {
		return fmt.Errorf("unknown provider: %s", p.Provider)
	}

	// Get the actual API key from environment
	apiKey := os.Getenv(cfg.APIKeyEnv)
	if apiKey == "" {
		return fmt.Errorf("API key not found: %s (set %s)", cfg.APIKeyEnv, cfg.APIKeyEnv)
	}

	// Set OVAV override env vars to force the provider selection
	// This ensures CRUSH and OVAV use the same provider
	os.Setenv("OVAV_PROVIDER", cfg.Provider)
	os.Setenv("OVAV_MODEL", p.Model)
	os.Setenv("OVAV_API_KEY", apiKey)
	os.Setenv("OVAV_BASE_URL", cfg.Endpoint)

	// Also set provider-specific env vars for CRUSH
	switch p.Provider {
	case "hyper":
		os.Setenv("HYPER_API_KEY", apiKey)
	case "minimax":
		// MiniMax Direct uses ANTHROPIC_API_KEY for the sk-cp- key
		os.Setenv("ANTHROPIC_API_KEY", apiKey)
		os.Setenv("ANTHROPIC_API_ENDPOINT", cfg.Endpoint)
	case "aihubmix":
		os.Setenv("MINIMAX_API_KEY", apiKey)
	case "openai":
		os.Setenv("OPENAI_API_KEY", apiKey)
	case "anthropic":
		os.Setenv("ANTHROPIC_API_KEY", apiKey)
	case "openrouter":
		os.Setenv("OPENROUTER_API_KEY", apiKey)
	}

	return nil
}

// cmdProvider handles the 'ovav provider' subcommand
func cmdProvider(args []string) int {
	if len(args) == 0 {
		return providerList(args)
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list", "ls", "show":
		return providerList(rest)
	case "use", "switch", "set":
		return providerUse(rest)
	case "status", "current", "active":
		return providerStatus(rest)
	case "unset", "clear", "off":
		return providerUnset(rest)
	case "--help", "-h", "help":
		providerHelp()
		return 0
	default:
		// Try to use as provider name
		if _, ok := ProviderConfigs[sub]; ok {
			return providerUse([]string{sub})
		}
		fmt.Fprintf(os.Stderr, "Unknown provider command: %s\n", sub)
		providerHelp()
		return 2
	}
}

// providerList shows all available providers
func providerList(args []string) int {
	jsonOut := false
	for _, a := range args {
		if a == "--json" {
			jsonOut = true
		}
	}

	// Load current active provider
	current, _ := loadActiveProvider()
	currentName := ""
	if current != nil {
		currentName = current.Provider
	}

	var names []string
	for name := range ProviderConfigs {
		names = append(names, name)
	}
	sort.Strings(names)

	if jsonOut {
		type providerInfo struct {
			Name     string `json:"name"`
			Provider string `json:"provider"`
			Model    string `json:"model"`
			Endpoint string `json:"endpoint"`
			Help     string `json:"help"`
			Active   bool   `json:"active"`
			KeySet   bool   `json:"key_set"`
		}
		var providers []providerInfo
		for _, name := range names {
			cfg := ProviderConfigs[name]
			info := providerInfo{
				Name:     name,
				Provider: cfg.Provider,
				Model:    cfg.Model,
				Endpoint: cfg.Endpoint,
				Help:     cfg.Help,
				Active:   name == currentName,
				KeySet:   os.Getenv(cfg.APIKeyEnv) != "",
			}
			providers = append(providers, info)
		}
		data, _ := json.MarshalIndent(struct {
			Status    string        `json:"status"`
			Current   string        `json:"current"`
			Providers []providerInfo `json:"providers"`
		}{"ok", currentName, providers}, "", "  ")
		fmt.Println(string(data))
		return 0
	}

	fmt.Println("🔗 OVAV Provider Selection")
	fmt.Println()
	fmt.Println("Available providers:")
	fmt.Println()

	for _, name := range names {
		cfg := ProviderConfigs[name]
		active := ""
		if name == currentName {
			active = " ✅ ACTIVE"
		}
		keySet := ""
		if os.Getenv(cfg.APIKeyEnv) != "" {
			keySet = " 🔑"
		}
		fmt.Printf("  %-18s %s%s%s\n", name, cfg.Help, keySet, active)
	}

	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  ovav provider list              Show all providers")
	fmt.Println("  ovav provider use <name>       Switch to provider")
	fmt.Println("  ovav provider status           Show current provider")
	fmt.Println("  ovav provider unset            Clear selection")
	fmt.Println()
	fmt.Printf("Current: %s\n", currentName)
	return 0
}

// providerUse switches to a different provider
func providerUse(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: ovav provider use <name>")
		fmt.Fprintln(os.Stderr, "Run 'ovav provider list' to see available providers.")
		return 2
	}

	name := args[0]
	cfg, ok := ProviderConfigs[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown provider: %s\n", name)
		fmt.Fprintln(os.Stderr, "Run 'ovav provider list' to see available providers.")
		return 1
	}

	// Check if API key is available
	apiKey := os.Getenv(cfg.APIKeyEnv)
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "❌ API key not found: %s\n", cfg.APIKeyEnv)
		fmt.Fprintf(os.Stderr, "   Set it with: export %s=<your-key>\n", cfg.APIKeyEnv)
		return 1
	}

	// Create and save the active provider
	p := &ActiveProvider{
		Provider:  name,
		Model:     cfg.Model,
		APIKeyRef: cfg.APIKeyEnv,
		Endpoint:  cfg.Endpoint,
		SetAt:     time.Now(),
		SetBy:     "user",
	}

	if err := saveActiveProvider(p); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to save: %v\n", err)
		return 1
	}

	// Apply to environment
	if err := applyProviderEnv(p); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: saved config but env apply failed: %v\n", err)
	}

	fmt.Printf("✅ Provider switched to: %s\n", name)
	fmt.Printf("   Model:     %s\n", cfg.Model)
	fmt.Printf("   Endpoint:  %s\n", cfg.Endpoint)
	fmt.Println()
	fmt.Println("⚠️  Restart your CRUSH session for changes to take full effect.")
	fmt.Println("   Or source your shell config to apply env vars immediately.")

	return 0
}

// providerStatus shows the current active provider
func providerStatus(args []string) int {
	jsonOut := false
	for _, a := range args {
		if a == "--json" {
			jsonOut = true
		}
	}

	p, err := loadActiveProvider()
	if err != nil {
		if os.IsNotExist(err) {
			if jsonOut {
				fmt.Println(`{"status":"no_selection","message":"No provider selected"}`)
			} else {
				fmt.Println("🔗 No provider selected.")
				fmt.Println("   Run 'ovav provider list' to see available providers.")
			}
			return 0
		}
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		return 1
	}

	cfg, ok := ProviderConfigs[p.Provider]
	if !ok {
		fmt.Printf("⚠️  Unknown provider: %s\n", p.Provider)
		return 1
	}

	if jsonOut {
		data, _ := json.MarshalIndent(struct {
			Status   string    `json:"status"`
			Provider string    `json:"provider"`
			Name     string    `json:"name"`
			Model    string    `json:"model"`
			Endpoint string    `json:"endpoint"`
			KeyRef   string    `json:"api_key_ref"`
			SetAt    time.Time `json:"set_at"`
			SetBy    string    `json:"set_by"`
		}{
			"active",
			p.Provider,
			p.Provider,
			cfg.Model,
			cfg.Endpoint,
			p.APIKeyRef,
			p.SetAt,
			p.SetBy,
		}, "", "  ")
		fmt.Println(string(data))
		return 0
	}

	fmt.Println("🔗 Current Provider")
	fmt.Println()
	fmt.Printf("  Provider:  %s\n", p.Provider)
	fmt.Printf("  Model:     %s\n", cfg.Model)
	fmt.Printf("  Endpoint:  %s\n", cfg.Endpoint)
	fmt.Printf("  API Key:   %s (env var)\n", p.APIKeyRef)
	fmt.Printf("  Set At:    %s\n", p.SetAt.Format("2006-01-02 15:04"))
	fmt.Printf("  Set By:    %s\n", p.SetBy)

	// Check if key is actually set
	apiKey := os.Getenv(cfg.APIKeyEnv)
	if apiKey == "" {
		fmt.Println()
		fmt.Println("⚠️  WARNING: API key not set in environment!")
		fmt.Printf("   Set with: export %s=<your-key>\n", cfg.APIKeyEnv)
	} else {
		keyPreview := apiKey
		if len(keyPreview) > 12 {
			keyPreview = "..." + keyPreview[len(keyPreview)-12:]
		}
		fmt.Printf("  Key:       %s ✅\n", keyPreview)
	}

	return 0
}

// providerUnset clears the active provider selection
func providerUnset(args []string) int {
	path := providerConfigPath()
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No provider selected — nothing to clear.")
			return 0
		}
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		return 1
	}

	// Clear OVAV override env vars
	os.Unsetenv("OVAV_PROVIDER")
	os.Unsetenv("OVAV_MODEL")
	os.Unsetenv("OVAV_API_KEY")
	os.Unsetenv("OVAV_BASE_URL")

	fmt.Println("✅ Provider selection cleared.")
	fmt.Println("   OVAV will auto-detect provider from available API keys.")
	fmt.Println("   Run 'ovav provider use <name>' to set a specific provider.")

	return 0
}

// providerHelp prints help for the provider command
func providerHelp() {
	fmt.Print(`OVAV Provider Switching — Select your AI provider at runtime

Usage:
  ovav provider list              List all available providers
  ovav provider use <name>        Switch to a provider
  ovav provider status           Show current provider
  ovav provider unset            Clear selection

Available Providers:
  hyper           hyper.charm.land credits (95 remaining)
  minimax_direct  Your MiniMax monthly subscription
  minimax_hyper   MiniMax via aihubmix (hyper proxy)
  openai          OpenAI Direct API
  anthropic       Anthropic Direct API
  openrouter      OpenRouter multi-provider

Examples:
  ovav provider list
  ovav provider use minimax_direct
  ovav provider use hyper
  ovav provider status
  ovav provider unset

Notes:
  - Provider selection is persisted in ~/.ovav/provider.json
  - Set your API keys as environment variables:
    export MINIMAX_API_KEY="..."
    export ANTHROPIC_API_KEY="..."
    export HYPER_API_KEY="..."
    export OPENAI_API_KEY="..."
`)
}
