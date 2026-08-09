// Package validators provides OVAV governance validators.
package validators

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DomainTracker tracks all domains ever seen by the context firewall,
// computes confidence scores, and auto-suggests domain approvals.
type DomainTracker struct {
	mu          sync.RWMutex
	path        string
	Approved    map[string]bool   `json:"approved"`           // code-allowlist domains
	Community   map[string]bool   `json:"community_approved"` // user-approved domains
	Seen        map[string]*Entry `json:"seen"`               // all domains ever seen
	Suggestions []string          `json:"suggestions"`        // new domains needing review
}

// Entry represents a tracked domain with confidence metadata.
type Entry struct {
	Domain     string   `json:"domain"`
	FirstSeen  string   `json:"first_seen"`
	LastSeen   string   `json:"last_seen"`
	Sources    []string `json:"sources"`    // unique source files
	Count      int      `json:"count"`      // total appearances
	Confidence float64  `json:"confidence"` // 0.0–1.0
	Status     string   `json:"status"`     // "code" | "community" | "new" | "rejected"
}

// CheckResult is returned by Tracker.Check() — describes domain disposition.
type CheckResult struct {
	Domain     string
	Approved   bool    // true if domain can pass without issue
	Status     string  // "code" | "community" | "new" | "rejected"
	Confidence float64 // 0.0–1.0
	Suggestion bool    // true if this domain should be suggested for approval
	Source     string  // file where domain was found (for tracking)
}

// NewDomainTracker creates or loads the domain tracker.
// registryPath should point to .ovav/security/domain_registry.json.
func NewDomainTracker(registryPath string) *DomainTracker {
	dt := &DomainTracker{
		path:      registryPath,
		Approved:  map[string]bool{},
		Community: map[string]bool{},
		Seen:      map[string]*Entry{},
	}
	// Seed with current approvedDomains as "code" status
	for d := range approvedDomains {
		dt.Approved[d] = true
	}
	dt.Load()
	return dt
}

// Load reads the tracker registry from disk (if exists).
func (dt *DomainTracker) Load() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	data, err := os.ReadFile(dt.path)
	if err != nil {
		return // no registry yet — start fresh
	}
	var reg struct {
		Community   map[string]bool   `json:"community_approved"`
		Seen        map[string]*Entry `json:"seen"`
		Suggestions []string          `json:"suggestions"`
	}
	if err := json.Unmarshal(data, &reg); err != nil {
		return
	}
	dt.Community = reg.Community
	if dt.Community == nil {
		dt.Community = map[string]bool{}
	}
	dt.Seen = reg.Seen
	if dt.Seen == nil {
		dt.Seen = map[string]*Entry{}
	}
	dt.Suggestions = reg.Suggestions
	if dt.Suggestions == nil {
		dt.Suggestions = []string{}
	}
	// Merge community into approved map for fast lookup
	for d := range dt.Community {
		dt.Approved[d] = true
	}
}

// Save persists the tracker registry to disk.
func (dt *DomainTracker) Save() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	// Ensure directory exists
	dir := filepath.Dir(dt.path)
	os.MkdirAll(dir, 0755)
	data, err := json.MarshalIndent(struct {
		Community   map[string]bool   `json:"community_approved"`
		Seen        map[string]*Entry `json:"seen"`
		Suggestions []string          `json:"suggestions"`
	}{
		Community:   dt.Community,
		Seen:        dt.Seen,
		Suggestions: dt.Suggestions,
	}, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(dt.path, data, 0644)
}

// isApprovedParent checks if a domain or any of its parent domains is approved.
// e.g. docs.github.com → github.com is approved → returns true.
func (dt *DomainTracker) isApprovedParent(domain string) bool {
	if dt.Approved[domain] {
		return true
	}
	parts := strings.Split(domain, ".")
	for i := 1; i < len(parts); i++ {
		parent := strings.Join(parts[i:], ".")
		if dt.Approved[parent] {
			return true
		}
	}
	return false
}

// Check determines the disposition of a domain.
// It returns a CheckResult with approval status, confidence, and whether
// this domain should be auto-suggested for the community allowlist.
func (dt *DomainTracker) Check(domain, sourceFile string) CheckResult {
	domain = strings.ToLower(domain)
	now := time.Now().UTC().Format(time.RFC3339)

	// 1. Code allowlist — always approved (with parent-domain matching)
	if dt.isApprovedParent(domain) {
		dt.trackDomain(domain, sourceFile, now, "code")
		return CheckResult{Domain: domain, Approved: true, Status: "code", Confidence: 1.0}
	}

	// 2. Community allowlist — approved with high confidence
	if dt.Community[domain] {
		dt.trackDomain(domain, sourceFile, now, "community")
		return CheckResult{Domain: domain, Approved: true, Status: "community", Confidence: 1.0}
	}

	// 3. New domain — compute confidence
	entry, exists := dt.Seen[domain]
	if !exists {
		entry = &Entry{
			Domain:    domain,
			FirstSeen: now,
			Sources:   []string{},
			Status:    "new",
		}
		dt.Seen[domain] = entry
	}

	entry.LastSeen = now
	entry.Count++
	if !sliceContains(entry.Sources, sourceFile) {
		entry.Sources = append(entry.Sources, sourceFile)
	}

	// Confidence: 0.0–0.4 = low (suspicious, fail)
	//              0.4–0.7 = medium (suggest but don't auto-approve)
	//              0.7–1.0 = high (auto-suggest approval)
	entry.Confidence = dt.computeConfidence(entry)

	// If high confidence AND first time seeing it from a legitimate source
	// file, auto-add to community and save
	highConfidence := entry.Confidence >= 0.8
	safeSource := isKnownSafeSource(sourceFile)

	if highConfidence && safeSource && !dt.Community[domain] {
		dt.Community[domain] = true
		dt.Approved[domain] = true
		entry.Status = "community"
		dt.Save()
		return CheckResult{
			Domain:     domain,
			Approved:   true,
			Status:     "community",
			Confidence: entry.Confidence,
			Suggestion: false, // already auto-approved, no need to suggest
			Source:     sourceFile,
		}
	}

	// If medium confidence, mark as suggestion
	if entry.Confidence >= 0.4 && !sliceContains(dt.Suggestions, domain) {
		dt.Suggestions = append(dt.Suggestions, domain)
		dt.Save()
		return CheckResult{
			Domain:     domain,
			Approved:   false,
			Status:     "new",
			Confidence: entry.Confidence,
			Suggestion: true,
			Source:     sourceFile,
		}
	}

	return CheckResult{
		Domain:     domain,
		Approved:   false,
		Status:     "new",
		Confidence: entry.Confidence,
		Suggestion: entry.Confidence >= 0.4,
		Source:     sourceFile,
	}
}

// trackDomain records a domain hit without re-computing confidence.
func (dt *DomainTracker) trackDomain(domain, sourceFile, now, status string) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	entry, exists := dt.Seen[domain]
	if !exists {
		entry = &Entry{Domain: domain, FirstSeen: now, Sources: []string{}, Status: status}
		dt.Seen[domain] = entry
	}
	entry.LastSeen = now
	entry.Count++
	if !sliceContains(entry.Sources, sourceFile) {
		entry.Sources = append(entry.Sources, sourceFile)
	}
	entry.Status = status
}

// computeConfidence calculates a 0.0–1.0 confidence score.
// Heuristics:
//   - Count: more appearances = more confidence (up to 5)
//   - Multiple source files = more confidence
//   - Appears in known-safe source files = more confidence
func (dt *DomainTracker) computeConfidence(entry *Entry) float64 {
	if entry == nil {
		return 0.0
	}
	var score float64
	// Count factor: 1–5 appearances → 0.0–0.5
	countScore := float64(minInt(entry.Count, 5)) / 5.0 * 0.5
	score += countScore
	// Source diversity factor: 1 file = 0.1, 2+ files = 0.2
	sourceScore := 0.1
	if len(entry.Sources) >= 2 {
		sourceScore = 0.2
	}
	if len(entry.Sources) >= 4 {
		sourceScore = 0.3
	}
	score += sourceScore
	// Known-safe source factor: +0.1 for each safe source file
	safeCount := 0
	for _, src := range entry.Sources {
		if isKnownSafeSource(src) {
			safeCount++
		}
	}
	score += float64(minInt(safeCount, 3)) * 0.05
	return minFloat(score, 1.0)
}

// isKnownSafeSource returns true if the file path is a known safe context.
func isKnownSafeSource(path string) bool {
	safePrefixes := []string{
		".ovav/service_areas/",
		".ovav/plan/",
		".ovav/handoffs/",
		".ovav/governance/",
		".ovav/security/",
		"go-runtime/internal/runtimes/opencode/agents/",
		"go-runtime/internal/runtimes/claude-code/agents/",
		".ovav/artifacts/",
		".ovav/registry/",
	}
	for _, prefix := range safePrefixes {
		if strings.Contains(path, prefix) {
			return true
		}
	}
	return false
}

// GetSuggestions returns the current list of suggested domains.
func (dt *DomainTracker) GetSuggestions() []string {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	suggestions := make([]string, len(dt.Suggestions))
	copy(suggestions, dt.Suggestions)
	return suggestions
}

// ClearSuggestions removes all pending suggestions (call after review).
func (dt *DomainTracker) ClearSuggestions() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.Suggestions = []string{}
}

// Approve adds a domain to the community allowlist.
func (dt *DomainTracker) Approve(domain string) {
	domain = strings.ToLower(domain)
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.Community[domain] = true
	dt.Approved[domain] = true
	if entry, exists := dt.Seen[domain]; exists {
		entry.Status = "community"
	}
	// Remove from suggestions
	newSugs := make([]string, 0, len(dt.Suggestions))
	for _, s := range dt.Suggestions {
		if s != domain {
			newSugs = append(newSugs, s)
		}
	}
	dt.Suggestions = newSugs
	dt.Save()
}

// Reject marks a domain as permanently rejected.
func (dt *DomainTracker) Reject(domain string) {
	domain = strings.ToLower(domain)
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if entry, exists := dt.Seen[domain]; exists {
		entry.Status = "rejected"
	}
	// Remove from suggestions
	newSugs := make([]string, 0, len(dt.Suggestions))
	for _, s := range dt.Suggestions {
		if s != domain {
			newSugs = append(newSugs, s)
		}
	}
	dt.Suggestions = newSugs
}

// Stats returns a summary of the tracker state.
func (dt *DomainTracker) Stats() (approved, community, seen, suggestions int) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return len(dt.Approved), len(dt.Community), len(dt.Seen), len(dt.Suggestions)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func sliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
