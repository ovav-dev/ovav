// Package autonomous implements OVAV's autonomous research system.
//
// This system runs scheduled research cycles to track:
// - AI provider updates (OpenAI, Anthropic, Google AI, OpenRouter)
// - Security vulnerabilities (OWASP)
// - Competitor feature comparisons
//
// All outputs are CLI-only (no web interface).
package autonomous

import "time"

// ResearchTarget represents a data source to research.
type ResearchTarget struct {
	ID          string       `yaml:"id"`
	Name        string       `yaml:"name"`
	URL         string       `yaml:"url"`
	Frequency   string       `yaml:"frequency"` // daily, weekly
	LastRun     time.Time    `yaml:"last_run"`
	NextRun     time.Time    `yaml:"next_run"`
	Enabled     bool         `yaml:"enabled"`
	Credentials *Credentials `yaml:"credentials,omitempty"`
}

// Credentials holds API keys or auth tokens for research targets.
type Credentials struct {
	APIKey string `yaml:"api_key,omitempty"`
	Token  string `yaml:"token,omitempty"`
}

// ResearchResult represents the output of a research cycle.
type ResearchResult struct {
	TargetID    string    `yaml:"target_id"`
	Timestamp   time.Time `yaml:"timestamp"`
	URLsScraped []string  `yaml:"urls_scraped"`
	Findings    []Finding `yaml:"findings"`
	Changes     []Change  `yaml:"changes"`
	Errors      []string  `yaml:"errors,omitempty"`
	DurationMs  int64     `yaml:"duration_ms"`
}

// Finding represents a discovered piece of intelligence.
type Finding struct {
	ID          string            `yaml:"id"`
	Title       string            `yaml:"title"`
	Description string            `yaml:"description"`
	Source      string            `yaml:"source"`
	URL         string            `yaml:"url"`
	Severity    string            `yaml:"severity"` // info, warning, critical
	Category    string            `yaml:"category"` // model, pricing, security, feature
	Metadata    map[string]string `yaml:"metadata,omitempty"`
	Discovered  time.Time         `yaml:"discovered"`
}

// Change represents a detected change from previous research.
type Change struct {
	ID          string    `yaml:"id"`
	Type        string    `yaml:"type"` // added, removed, modified
	Title       string    `yaml:"title"`
	Description string    `yaml:"description"`
	Before      string    `yaml:"before,omitempty"`
	After       string    `yaml:"after,omitempty"`
	Detected    time.Time `yaml:"detected"`
}

// ResearchStatus represents the current state of the research system.
type ResearchStatus struct {
	LastFullRun   time.Time        `yaml:"last_full_run"`
	NextScheduled time.Time        `yaml:"next_scheduled"`
	Targets       []ResearchTarget `yaml:"targets"`
	TotalFindings int              `yaml:"total_findings"`
	TotalChanges  int              `yaml:"total_changes"`
	Running       bool             `yaml:"running"`
}
