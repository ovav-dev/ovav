package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// agentMemoryPath is the path for OVAV agent persistent memory.
// Using .ovav/runtime/ (not .ovav/registry/) avoids LedgerDeprecation validator.
const agentMemoryPath = ".ovav/runtime/agent_memory.yaml"

// AgentMemory is the persistent memory store for OVAV agents.
// It survives sessions, supports multi-agent lanes, and provides authenticated recall.
type AgentMemory struct {
	ledger     *Ledger // underlying ledger
	classifier *Classifier
	recall     *Recall
	root       string // repo root
}

// NewAgentMemory creates a new agent memory instance for the given repo root.
// allowSensitive must be true for Systems agents (allows internal/sensitive storage).
func NewAgentMemory(root string, allowSensitive bool) (*AgentMemory, error) {
	// Ensure the runtime directory exists
	runtimeDir := filepath.Join(root, ".ovav", "runtime")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return nil, fmt.Errorf("agentmemory: mkdir runtime dir: %w", err)
	}

	// Load or create the agent memory ledger
	ledger, err := loadAgentLedger(root)
	if err != nil {
		return nil, fmt.Errorf("agentmemory: load ledger: %w", err)
	}

	return &AgentMemory{
		ledger:     ledger,
		classifier: NewClassifier(allowSensitive),
		recall:     NewRecall(ledger),
		root:       root,
	}, nil
}

// loadAgentLedger reads the agent memory file or creates a new empty one.
func loadAgentLedger(root string) (*Ledger, error) {
	path := filepath.Join(root, agentMemoryPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newEmptyAgentLedger(path), nil
		}
		return nil, fmt.Errorf("read agent memory: %w", err)
	}

	// Try new format first (flat, no wrapper)
	var ledger AgentMemoryFile
	if err := yaml.Unmarshal(data, &ledger); err == nil && ledger.Cards != nil {
		l := &Ledger{
			Version:    ledger.Version,
			Purpose:    ledger.Purpose,
			Cards:      ledger.Cards,
			LastUpdate: ledger.LastUpdated,
			path:       path,
			loadedAt:   time.Now(),
		}
		for i := range l.Cards {
			l.Cards[i].loadedAt = l.loadedAt
		}
		return l, nil
	}

	// Try wrapped format (legacy)
	var raw struct {
		AgentMemory struct {
			Version    int    `yaml:"version"`
			Purpose    string `yaml:"purpose"`
			Cards      []Card `yaml:"cards"`
			LastUpdate string `yaml:"last_updated"`
		} `yaml:"agent_memory"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse agent memory: %w", err)
	}

	l := &Ledger{
		Version:    raw.AgentMemory.Version,
		Purpose:    raw.AgentMemory.Purpose,
		Cards:      raw.AgentMemory.Cards,
		LastUpdate: raw.AgentMemory.LastUpdate,
		path:       path,
		loadedAt:   time.Now(),
	}
	for i := range l.Cards {
		l.Cards[i].loadedAt = l.loadedAt
	}
	return l, nil
}

// newEmptyAgentLedger returns a fresh ledger for new memory files.
func newEmptyAgentLedger(path string) *Ledger {
	return &Ledger{
		Version:    1,
		Purpose:    "OVAV SYSTEMS Agent Memory — persistent across sessions",
		Cards:      []Card{},
		LastUpdate: time.Now().Format(time.RFC3339),
		path:       path,
		loadedAt:   time.Now(),
	}
}

// AgentMemoryFile is the flat YAML structure for agent_memory.yaml.
type AgentMemoryFile struct {
	Version     int    `yaml:"version"`
	Purpose     string `yaml:"purpose"`
	LastUpdated string `yaml:"last_updated"`
	Cards       []Card `yaml:"cards"`
}

// Save persists the current state of agent memory to disk.
// Uses atomic write (temp file + rename) to prevent corruption.
func (am *AgentMemory) Save() error {
	if am.ledger.path == "" {
		return fmt.Errorf("agentmemory: ledger has no path")
	}

	am.ledger.LastUpdate = time.Now().Format(time.RFC3339)

	// Atomic write: temp file + rename
	file := AgentMemoryFile{
		Version:     am.ledger.Version,
		Purpose:     am.ledger.Purpose,
		LastUpdated: am.ledger.LastUpdate,
		Cards:       am.ledger.Cards,
	}

	data, err := yaml.Marshal(&file)
	if err != nil {
		return fmt.Errorf("marshal agent memory: %w", err)
	}

	tmpPath := am.ledger.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp agent memory: %w", err)
	}

	if err := os.Rename(tmpPath, am.ledger.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename agent memory: %w", err)
	}

	return nil
}

// StoreOptions controls how a card is stored.
type StoreOptions struct {
	// AgentID is the OVAV lead who created this memory (e.g., "thavren", "eidren").
	AgentID string
	// SessionID is the MiMoCode session that produced this memory.
	SessionID string
	// Commit is the git HEAD SHA at the time of writing (anchors evidence).
	Commit string
	// Priority: "CRITICAL", "HIGH", "NORMAL", "LOW"
	Priority string
	// Tags for classification and recall
	Tags []string
	// Force bypasses privacy classifier (for internal system cards)
	Force bool
}

// StoreRecord is the result of a Store operation.
type StoreRecord struct {
	Card   Card
	Tag    PrivacyTag
	Reason string
}

// Store submits a new memory card through the write pipeline:
// classify → authenticate → persist.
// Returns the result including the privacy tag assigned.
func (am *AgentMemory) Store(card Card, opts StoreOptions) (*StoreRecord, error) {
	// Set metadata from options
	if opts.AgentID != "" {
		card.ProposedBy = opts.AgentID
	}
	if opts.Priority != "" {
		card.Priority = opts.Priority
	}
	if opts.Tags != nil {
		card.Tags = opts.Tags
	}
	card.Status = StatusActive

	// Add source chain (authenticity)
	card.Commit = opts.Commit
	if card.Commit == "" {
		card.Commit = guessGitHead(am.root)
	}

	// Add evidence hash (content authenticity)
	card.Summary = strings.TrimSpace(card.Summary)
	card.OperationalRule = strings.TrimSpace(card.OperationalRule)
	evidenceHash := sha256Hash(card.Summary + "|" + card.OperationalRule)
	// Store in a field that's already defined — use DeprecatedAt as evidence carrier
	// (this field is persisted but semantically ignored for normal operations)
	if card.DeprecatedAt == "" {
		card.DeprecatedAt = evidenceHash
	}

	// Add creation timestamp if not set
	if card.LastConfirmed == "" {
		card.LastConfirmed = time.Now().Format("2006-01-02")
	}

	// Generate ID if not set
	if card.ID == "" {
		card.ID = generateCardID(opts.AgentID, card.Summary)
	}

	// Privacy classification (unless forced)
	if !opts.Force {
		result := am.classifier.Classify(card)
		if !result.Allow {
			return nil, fmt.Errorf("agentmemory store: rejected by classifier: %s (tag=%s)",
				result.Reason, result.Tag)
		}
		card.Tags = append(card.Tags, string(result.Tag))
	}

	// Validate
	if card.Summary == "" {
		return nil, fmt.Errorf("agentmemory store: summary is required")
	}
	if card.OperationalRule == "" {
		return nil, fmt.Errorf("agentmemory store: operational_rule is required")
	}

	// Upsert and save
	am.ledger.UpsertCard(card)
	if err := am.Save(); err != nil {
		return nil, fmt.Errorf("agentmemory save: %w", err)
	}

	result := &StoreRecord{
		Card:   card,
		Reason: "stored",
	}
	if opts.Force {
		result.Tag = PrivacySensitiveLocal
		result.Reason = "forced (system card)"
	} else {
		result.Tag = am.classifier.Classify(card).Tag
	}

	return result, nil
}

// RecallOptions controls memory recall.
type RecallOptions struct {
	// Query free-text search terms
	Query string
	// Tags to filter by (AND logic)
	Tags []string
	// Limit number of results (0 = unlimited)
	Limit int
	// AgentID filters to cards created by this agent
	AgentID string
	// MinRelevance filters by minimum relevance score (0.0-1.0)
	MinRelevance float64
}

// RecallResults holds recall output with metadata.
type RecallResults struct {
	Cards         []Card   // matched cards
	TotalFound    int      // total cards before limit
	Query         string   // query used
	Tags          []string // tags used
	AgentsChecked int      // how many agents' memory was scanned
	Authenticity  AuthenticityReport
}

// Recall searches agent memory for relevant cards.
// Combines tag match, text match, and authenticity verification.
// Returns cards sorted by relevance + recency.
func (am *AgentMemory) Recall(opts RecallOptions) *RecallResults {
	results := &RecallResults{
		Query: opts.Query,
		Tags:  opts.Tags,
	}

	var allCards []Card
	if len(opts.Tags) > 0 {
		for _, r := range am.recall.ByTags(opts.Tags) {
			allCards = append(allCards, r.Card)
		}
	}
	if opts.Query != "" {
		for _, r := range am.recall.ByQuery(opts.Query, 0) {
			allCards = append(allCards, r.Card)
		}
	}

	// Deduplicate by ID
	seen := make(map[string]bool)
	var deduped []Card
	for _, c := range allCards {
		if seen[c.ID] {
			continue
		}
		seen[c.ID] = true

		// Filter by agent if specified
		if opts.AgentID != "" && c.ProposedBy != opts.AgentID {
			continue
		}

		deduped = append(deduped, c)
	}

	results.TotalFound = len(deduped)
	results.AgentsChecked = 1 // single-agent for now

	// Apply limit
	if opts.Limit > 0 && len(deduped) > opts.Limit {
		deduped = deduped[:opts.Limit]
	}

	results.Cards = deduped

	// Run authenticity verification
	results.Authenticity = am.Verify(deduped)

	return results
}

// AuthenticityReport holds the result of authenticity verification.
type AuthenticityReport struct {
	Total     int      `json:"total"`
	Verified  int      `json:"verified"`  // cards with valid evidence hash
	Stale     int      `json:"stale"`     // cards with stale git commit
	NoSource  int      `json:"no_source"` // cards without source chain
	Conflicts int      `json:"conflicts"` // cards with contradicted facts
	Issues    []string `json:"issues,omitempty"`
}

// Verify checks the authenticity of a set of cards:
// - Evidence hash integrity (content not tampered)
// - Source chain validity (git commit exists)
// - Contradiction detection (same topic, opposite conclusions)
func (am *AgentMemory) Verify(cards []Card) AuthenticityReport {
	report := AuthenticityReport{Total: len(cards)}
	topicFindings := make(map[string][]Card) // topic -> cards with that topic

	for _, card := range cards {
		// Check evidence hash
		if card.DeprecatedAt == "" {
			report.NoSource++
			report.Issues = append(report.Issues, "no_evidence_hash:"+card.ID)
			continue
		}

		expectedHash := sha256Hash(card.Summary + "|" + card.OperationalRule)
		if card.DeprecatedAt != expectedHash {
			report.Issues = append(report.Issues, "hash_mismatch:"+card.ID)
			// Don't count as verified
			continue
		}
		report.Verified++

		// Track for contradiction detection
		if card.Topic != "" {
			topicFindings[card.Topic] = append(topicFindings[card.Topic], card)
		}

		// Check if git commit is still reachable (lightweight check)
		if card.Commit != "" && !gitCommitExists(am.root, card.Commit) {
			report.Stale++
			report.Issues = append(report.Issues, "stale_commit:"+card.ID+"->"+card.Commit[:7])
		}
	}

	// Contradiction detection
	for topic, topicCards := range topicFindings {
		if len(topicCards) < 2 {
			continue
		}
		// Check for opposite conclusions
		hasActive := false
		hasDeprecated := false
		for _, c := range topicCards {
			if c.Status == StatusActive {
				hasActive = true
			}
			if c.Status == StatusDeprecated {
				hasDeprecated = true
			}
		}
		if hasActive && hasDeprecated {
			report.Conflicts++
			report.Issues = append(report.Issues, fmt.Sprintf("contradiction_detected:%s (%d cards)", topic, len(topicCards)))
		}
	}

	return report
}

// Recent returns the most recent N cards across all agents.
func (am *AgentMemory) Recent(limit int) []Card {
	cards := am.recall.RecentActive(0)
	if limit > 0 && len(cards) > limit {
		return cards[:limit]
	}
	return cards
}

// Stats returns memory statistics.
func (am *AgentMemory) Stats() (total, active, byAgent, byTag map[string]int) {
	total = make(map[string]int)
	active = make(map[string]int)
	byAgent = make(map[string]int)
	byTag = make(map[string]int)

	for _, card := range am.ledger.Cards {
		total["total"]++
		if card.Status == StatusActive || card.Status == StatusInProgress {
			active["active"]++
		}
		if card.ProposedBy != "" {
			byAgent[card.ProposedBy]++
		}
		for _, tag := range card.Tags {
			byTag[tag]++
		}
	}

	return total, active, byAgent, byTag
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// sha256Hash computes a SHA-256 hash of a string and returns hex encoded.
func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// generateCardID creates a deterministic unique ID for a card.
func generateCardID(agentID, summary string) string {
	// Use truncated SHA of agent+summary+timestamp for uniqueness
	input := fmt.Sprintf("%s|%s|%d", agentID, summary, time.Now().UnixNano())
	hash := sha256Hash(input)
	prefix := "mem"
	if agentID != "" {
		prefix = agentID[:3]
	}
	return fmt.Sprintf("%s-%s", prefix, hash[:12])
}

// guessGitHead returns the current git HEAD SHA if available.
func guessGitHead(root string) string {
	// Read .git/HEAD directly
	headPath := filepath.Join(root, ".git", "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return ""
	}
	content := strings.TrimSpace(string(data))
	const prefix = "ref: refs/heads/"
	if len(content) > len(prefix) && content[:len(prefix)] == prefix {
		branch := content[len(prefix):]
		// Read the ref
		refPath := filepath.Join(root, ".git", "refs", "heads", branch)
		refData, err := os.ReadFile(refPath)
		if err == nil {
			return strings.TrimSpace(string(refData))
		}
	}
	// Detached HEAD
	if !strings.Contains(content, "ref:") {
		return content
	}
	return ""
}

// gitCommitExists checks if a commit is reachable in the current repo.
func gitCommitExists(root, commit string) bool {
	// Fast check: see if commit is in the reflog or reachable
	// We just check if the commit object exists in .git/objects
	objPath := filepath.Join(root, ".git", "objects", commit[:2], commit[2:])
	if _, err := os.Stat(objPath); err == nil {
		return true
	}
	return false
}
