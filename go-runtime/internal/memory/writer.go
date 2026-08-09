package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// SessionWriter buffers memory cards during a session and auto-flushes
// on significant events (commit, governance action, session end).
//
// This is the AUTONOMOUS layer — no manual `ovav memory store` required.
// Cards are proposed automatically by OVAV SYSTEMS based on events and
// decisions made during the session.
//
// Flow:
//
//	Propose() → buffer (in-memory)
//	Flush()    → Governor.Store() each card + clear buffer
//
// Auto-flush triggers:
//   - Git post-commit hook fires → Flush()
//   - Governance decision made   → Flush()
//   - Session end (checkpoint)   → Flush()
//   - Error/failure             → Store error card + Flush()
type SessionWriter struct {
	mu      sync.Mutex
	buffer  []Card
	am      *AgentMemory
	root    string
	flushes int   // how many times Flush() has been called
	lastErr error // last error encountered
}

// sessionBufferPath is where unsent cards are staged between sessions.
const sessionBufferPath = ".ovav/runtime/memory_session_buffer.yaml"

// NewSessionWriter creates a SessionWriter backed by an AgentMemory store.
func NewSessionWriter(am *AgentMemory, root string) *SessionWriter {
	sw := &SessionWriter{
		am:   am,
		root: root,
	}
	// Load any leftover staged cards from previous session
	sw.loadStaged()
	return sw
}

// Propose adds a card to the write buffer.
// The card is NOT persisted until Flush() is called.
// If the buffer exceeds 50 cards, auto-flushes to prevent memory bloat.
func (sw *SessionWriter) Propose(card Card) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	// Auto-assign metadata
	if card.ID == "" {
		card.ID = generateCardID(card.ProposedBy, card.Summary)
	}
	if card.Status == "" {
		card.Status = StatusActive
	}
	if card.LastConfirmed == "" {
		card.LastConfirmed = time.Now().Format("2006-01-02")
	}

	sw.buffer = append(sw.buffer, card)

	// Auto-flush at 50 cards (prevent unbounded growth)
	if len(sw.buffer) >= 50 {
		sw.flushLocked()
	}

	return nil
}

// Flush persists all buffered cards and clears the buffer.
// Returns the number of cards flushed and any error encountered.
func (sw *SessionWriter) Flush() (int, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	n, err := sw.flushLocked()
	return n, err
}

func (sw *SessionWriter) flushLocked() (int, error) {
	if len(sw.buffer) == 0 {
		return 0, nil
	}

	flushed := 0
	var lastErr error
	for _, card := range sw.buffer {
		_, err := sw.am.Store(card, StoreOptions{
			AgentID: card.ProposedBy,
			Commit:  card.Commit,
			Force:   card.ProposedBy == "ovav-system", // system cards bypass classifier
		})
		if err != nil {
			lastErr = fmt.Errorf("flush card %s: %w", card.ID, err)
			continue
		}
		flushed++
	}

	sw.buffer = sw.buffer[:0]
	sw.flushes++
	if lastErr != nil {
		sw.lastErr = lastErr
	}
	return flushed, lastErr
}

// Stage persists the buffer to disk without flushing to memory.
// Used for crash recovery: if the session crashes, staged cards are
// reloaded on next SessionWriter creation.
func (sw *SessionWriter) Stage() error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if len(sw.buffer) == 0 {
		// Clear any stale stage file
		stagePath := filepath.Join(sw.root, sessionBufferPath)
		os.Remove(stagePath)
		return nil
	}

	type stagedFile struct {
		Cards      []Card `yaml:"cards"`
		StagedAt   string `yaml:"staged_at"`
		FlushCount int    `yaml:"flush_count"`
	}

	sf := stagedFile{
		Cards:      sw.buffer,
		StagedAt:   time.Now().Format(time.RFC3339),
		FlushCount: sw.flushes,
	}

	data, err := yaml.Marshal(&sf)
	if err != nil {
		return fmt.Errorf("marshal stage file: %w", err)
	}

	stagePath := filepath.Join(sw.root, sessionBufferPath)
	tmpPath := stagePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write stage file: %w", err)
	}
	if err := os.Rename(tmpPath, stagePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename stage file: %w", err)
	}

	return nil
}

func (sw *SessionWriter) loadStaged() {
	stagePath := filepath.Join(sw.root, sessionBufferPath)
	data, err := os.ReadFile(stagePath)
	if err != nil {
		return // No staged file — that's fine
	}

	var sf struct {
		Cards      []Card `yaml:"cards"`
		StagedAt   string `yaml:"staged_at"`
		FlushCount int    `yaml:"flush_count"`
	}
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return // Corrupt stage file — discard
	}

	sw.mu.Lock()
	sw.buffer = append(sw.buffer, sf.Cards...)
	sw.flushes = sf.FlushCount
	sw.mu.Unlock()
}

// BufferLen returns the number of cards currently in the buffer.
func (sw *SessionWriter) BufferLen() int {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return len(sw.buffer)
}

// FlushCount returns how many times Flush() has been called.
func (sw *SessionWriter) FlushCount() int {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.flushes
}

// LastErr returns the last error encountered during flush.
func (sw *SessionWriter) LastErr() error {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.lastErr
}

// ── Autonomous event proposes ───────────────────────────────────────────────

// ProposeCommit creates a memory card for a git commit event.
// This is called automatically by the post-commit hook.
func (sw *SessionWriter) ProposeCommit(commitHash, message, author, branch string) error {
	card := Card{
		Topic:   "git commit: " + branch,
		Summary: message,
		OperationalRule: fmt.Sprintf(
			"Committed to %s by %s — hash: %s. Every commit represents a completed unit of work.",
			branch, author, commitHash[:7]),
		Status:     StatusActive,
		Priority:   "NORMAL",
		ProposedBy: "ovav-system",
		Tags:       []string{"git", "commit", "autonomous"},
		Commit:     commitHash,
	}
	return sw.Propose(card)
}

// ProposeDecision creates a memory card for a governance decision.
// Called by governance hooks when a significant decision is made.
func (sw *SessionWriter) ProposeDecision(decision, rationale, decidedBy string) error {
	card := Card{
		Topic:   "governance: " + decision,
		Summary: decision,
		OperationalRule: fmt.Sprintf(
			"Decision rationale: %s. Decided by: %s. This decision governs future OVAV behavior.",
			rationale, decidedBy),
		Status:     StatusActive,
		Priority:   "HIGH",
		ProposedBy: "ovav-system",
		Tags:       []string{"governance", "decision", "autonomous"},
	}
	return sw.Propose(card)
}

// ProposeError creates a memory card for an error encountered.
// This lets OVAV remember failures and avoid repeating them.
func (sw *SessionWriter) ProposeError(errorMsg, context string) error {
	card := Card{
		Topic:   "error: " + context,
		Summary: fmt.Sprintf("Error in %s: %s", context, errorMsg),
		OperationalRule: fmt.Sprintf(
			"Do not repeat the action that caused this error. Investigate root cause before retrying."),
		Status:     StatusActive,
		Priority:   "HIGH",
		ProposedBy: "ovav-system",
		Tags:       []string{"error", "autonomous", "error_recovery"},
	}
	return sw.Propose(card)
}

// ProposeValidatorResult stores validator outcomes as memory cards.
// Auto-stored after each validate run.
func (sw *SessionWriter) ProposeValidatorResult(validatorName string, passed bool, issues []string) error {
	summary := fmt.Sprintf("Validator %s: %s", validatorName, boolToWord(passed))
	if len(issues) > 0 {
		summary += fmt.Sprintf(" — issues: %v", issues)
	}
	card := Card{
		Topic:   "validator: " + validatorName,
		Summary: summary,
		OperationalRule: fmt.Sprintf(
			"Validator %s outcome stored. Passed=%v. Issues=%v. Used for trend analysis.",
			validatorName, passed, issues),
		Status:     StatusActive,
		Priority:   normalPriority(passed),
		ProposedBy: "ovav-system",
		Tags:       []string{"validator", "autonomous", "monitoring"},
	}
	return sw.Propose(card)
}

// ProposeSessionSummary creates a card summarizing what happened in a session.
// Called at session end (via checkpoint writer → ovav memory flush).
func (sw *SessionWriter) ProposeSessionSummary(sessionID, summary string, tasksDone []string) error {
	tags := []string{"session", "autonomous", "summary"}
	rule := "Session summary — "
	if len(tasksDone) > 0 {
		rule += fmt.Sprintf("Tasks completed: %s. ", joinStr(tasksDone, ", "))
	}
	rule += "Store as historical record of OVAV SYSTEMS activity."

	card := Card{
		Topic:           "session: " + sessionID,
		Summary:         summary,
		OperationalRule: rule,
		Status:          StatusActive,
		Priority:        "NORMAL",
		ProposedBy:      "ovav-system",
		Tags:            tags,
		SessionID:       sessionID,
	}
	return sw.Propose(card)
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func boolToWord(b bool) string {
	if b {
		return "PASS"
	}
	return "FAIL"
}

func normalPriority(ok bool) string {
	if ok {
		return "LOW"
	}
	return "HIGH"
}

func joinStr(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for i := 1; i < len(ss); i++ {
		result += sep + ss[i]
	}
	return result
}
