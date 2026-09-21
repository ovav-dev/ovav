package agents

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	// ErrInvalidTrace indicates that a trace does not satisfy the observability policy.
	ErrInvalidTrace = errors.New("invalid observability trace")
	// ErrUnsafeTraceSink indicates that a file sink is not confined to its source root.
	ErrUnsafeTraceSink = errors.New("unsafe observability trace sink")
)

// TraceID is the stable identifier of one observability event.
type TraceID string

// TraceMode identifies the governed routing or delegation mode for an event.
type TraceMode string

// DecisionPacket captures the governed decision metadata required by delegation policy.
type DecisionPacket struct {
	ServiceArea      ServiceArea `json:"service_area"`
	Lead             string      `json:"lead"`
	TaskSize         string      `json:"task_size"`
	RiskLevel        string      `json:"risk_level"`
	DelegationMode   string      `json:"delegation_mode"`
	ContextBudget    string      `json:"context_budget"`
	ModelTier        string      `json:"model_tier"`
	ValidationMode   string      `json:"validation_mode"`
	DeliveryContract string      `json:"delivery_contract"`
}

// TraceEvent contains the required fields from observability_policy.yaml.
// It intentionally has no request text, prompt, response, chat, or secret fields.
type TraceEvent struct {
	TraceID            TraceID        `json:"trace_id"`
	Timestamp          time.Time      `json:"timestamp"`
	ServiceArea        ServiceArea    `json:"service_area"`
	Lead               string         `json:"lead"`
	Mode               TraceMode      `json:"mode"`
	DecisionPacket     DecisionPacket `json:"decision_packet"`
	SourceRequests     []string       `json:"source_requests"`
	ContextDecisions   []string       `json:"context_decisions"`
	ToolCalls          []string       `json:"tool_calls"`
	Handoffs           []string       `json:"handoffs"`
	ModelUsed          string         `json:"model_used"`
	TokenUsageEstimate int64          `json:"token_usage_estimate"`
	CostEstimate       float64        `json:"cost_estimate"`
	EvalsPassed        []string       `json:"evals_passed"`
	EvalsFailed        []string       `json:"evals_failed"`
	DeliveryContract   string         `json:"delivery_contract"`
}

type traceEventJSON TraceEvent

// TraceEventInput provides deterministic construction data. TraceSeed must be
// a stable, non-secret operation identifier rather than request or chat text.
type TraceEventInput struct {
	TraceSeed          string
	Timestamp          time.Time
	ServiceArea        ServiceArea
	Lead               string
	Mode               TraceMode
	DecisionPacket     DecisionPacket
	SourceRequests     []string
	ContextDecisions   []string
	ToolCalls          []string
	Handoffs           []string
	ModelUsed          string
	TokenUsageEstimate int64
	CostEstimate       float64
	EvalsPassed        []string
	EvalsFailed        []string
	DeliveryContract   string
}

// NewTraceEvent deterministically creates and validates a trace event. The
// same complete input produces the same TraceID and event.
func NewTraceEvent(input TraceEventInput) (TraceEvent, error) {
	if err := validateMetadata("trace seed", input.TraceSeed, true); err != nil {
		return TraceEvent{}, fmt.Errorf("create trace event: %w", err)
	}
	if input.Timestamp.IsZero() {
		return TraceEvent{}, fmt.Errorf("create trace event: %w: timestamp is required", ErrInvalidTrace)
	}

	event := TraceEvent{
		Timestamp:          input.Timestamp.UTC(),
		ServiceArea:        input.ServiceArea,
		Lead:               input.Lead,
		Mode:               input.Mode,
		DecisionPacket:     input.DecisionPacket,
		SourceRequests:     cloneStrings(input.SourceRequests),
		ContextDecisions:   cloneStrings(input.ContextDecisions),
		ToolCalls:          cloneStrings(input.ToolCalls),
		Handoffs:           cloneStrings(input.Handoffs),
		ModelUsed:          input.ModelUsed,
		TokenUsageEstimate: input.TokenUsageEstimate,
		CostEstimate:       input.CostEstimate,
		EvalsPassed:        cloneStrings(input.EvalsPassed),
		EvalsFailed:        cloneStrings(input.EvalsFailed),
		DeliveryContract:   input.DeliveryContract,
	}

	material := struct {
		TraceSeed string         `json:"trace_seed"`
		Event     traceEventJSON `json:"event"`
	}{TraceSeed: input.TraceSeed, Event: traceEventJSON(event)}
	data, err := json.Marshal(material)
	if err != nil {
		return TraceEvent{}, fmt.Errorf("create trace event identifier: %w", err)
	}
	sum := sha256.Sum256(data)
	event.TraceID = TraceID("trace-" + hex.EncodeToString(sum[:]))

	if err := event.Validate(); err != nil {
		return TraceEvent{}, fmt.Errorf("create trace event: %w", err)
	}
	return event, nil
}

// Validate enforces required non-trivial fields and metadata-only trace data.
func (event TraceEvent) Validate() error {
	if !validTraceID(event.TraceID) {
		return fmt.Errorf("%w: trace ID is required and must use the generated format", ErrInvalidTrace)
	}
	if event.Timestamp.IsZero() {
		return fmt.Errorf("%w: timestamp is required", ErrInvalidTrace)
	}
	if event.TokenUsageEstimate < 0 {
		return fmt.Errorf("%w: token usage estimate cannot be negative", ErrInvalidTrace)
	}
	if event.CostEstimate < 0 || math.IsNaN(event.CostEstimate) || math.IsInf(event.CostEstimate, 0) {
		return fmt.Errorf("%w: cost estimate must be finite and non-negative", ErrInvalidTrace)
	}

	if strings.TrimSpace(string(event.ServiceArea)) == "" {
		return fmt.Errorf("%w: service area is required for non-trivial events", ErrInvalidTrace)
	}
	if strings.TrimSpace(event.Lead) == "" {
		return fmt.Errorf("%w: lead is required for non-trivial events", ErrInvalidTrace)
	}
	if strings.TrimSpace(string(event.Mode)) == "" {
		return fmt.Errorf("%w: mode is required for non-trivial events", ErrInvalidTrace)
	}
	if err := event.DecisionPacket.validate(); err != nil {
		return fmt.Errorf("%w: decision packet: %v", ErrInvalidTrace, err)
	}
	if strings.TrimSpace(event.DeliveryContract) == "" {
		return fmt.Errorf("%w: delivery contract is required for non-trivial events", ErrInvalidTrace)
	}
	if len(event.EvalsPassed)+len(event.EvalsFailed) == 0 {
		return fmt.Errorf("%w: eval results are required for non-trivial events", ErrInvalidTrace)
	}
	if event.DecisionPacket.ServiceArea != event.ServiceArea {
		return fmt.Errorf("%w: decision packet service area does not match event", ErrInvalidTrace)
	}
	if event.DecisionPacket.Lead != event.Lead {
		return fmt.Errorf("%w: decision packet lead does not match event", ErrInvalidTrace)
	}
	if event.DecisionPacket.DeliveryContract != event.DeliveryContract {
		return fmt.Errorf("%w: decision packet delivery contract does not match event", ErrInvalidTrace)
	}

	metadata := []struct {
		name     string
		value    string
		required bool
	}{
		{name: "service area", value: string(event.ServiceArea), required: true},
		{name: "lead", value: event.Lead, required: true},
		{name: "mode", value: string(event.Mode), required: true},
		{name: "model used", value: event.ModelUsed},
		{name: "delivery contract", value: event.DeliveryContract, required: true},
	}
	for _, field := range metadata {
		if err := validateMetadata(field.name, field.value, field.required); err != nil {
			return err
		}
	}
	for name, values := range map[string][]string{
		"source request":   event.SourceRequests,
		"context decision": event.ContextDecisions,
		"tool call":        event.ToolCalls,
		"handoff":          event.Handoffs,
		"passed eval":      event.EvalsPassed,
		"failed eval":      event.EvalsFailed,
	} {
		for _, value := range values {
			if err := validateMetadata(name, value, true); err != nil {
				return err
			}
		}
	}
	if err := event.DecisionPacket.validateMetadata(); err != nil {
		return err
	}
	return nil
}

// MarshalJSON validates an event before serializing all policy-required fields.
func (event TraceEvent) MarshalJSON() ([]byte, error) {
	if err := event.Validate(); err != nil {
		return nil, fmt.Errorf("marshal trace event: %w", err)
	}
	data, err := json.Marshal(traceEventJSON(event))
	if err != nil {
		return nil, fmt.Errorf("marshal trace event: %w", err)
	}
	return data, nil
}

func (packet DecisionPacket) validate() error {
	required := []struct {
		name  string
		value string
	}{
		{name: "service area", value: string(packet.ServiceArea)},
		{name: "lead", value: packet.Lead},
		{name: "task size", value: packet.TaskSize},
		{name: "risk level", value: packet.RiskLevel},
		{name: "delegation mode", value: packet.DelegationMode},
		{name: "context budget", value: packet.ContextBudget},
		{name: "model tier", value: packet.ModelTier},
		{name: "validation mode", value: packet.ValidationMode},
		{name: "delivery contract", value: packet.DeliveryContract},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is required", field.name)
		}
	}
	return nil
}

func (packet DecisionPacket) validateMetadata() error {
	fields := []struct {
		name  string
		value string
	}{
		{name: "decision packet service area", value: string(packet.ServiceArea)},
		{name: "decision packet lead", value: packet.Lead},
		{name: "decision packet task size", value: packet.TaskSize},
		{name: "decision packet risk level", value: packet.RiskLevel},
		{name: "decision packet delegation mode", value: packet.DelegationMode},
		{name: "decision packet context budget", value: packet.ContextBudget},
		{name: "decision packet model tier", value: packet.ModelTier},
		{name: "decision packet validation mode", value: packet.ValidationMode},
		{name: "decision packet delivery contract", value: packet.DeliveryContract},
	}
	for _, field := range fields {
		if err := validateMetadata(field.name, field.value, false); err != nil {
			return err
		}
	}
	return nil
}

func validTraceID(id TraceID) bool {
	const prefix = "trace-"
	value := string(id)
	if len(value) != len(prefix)+sha256.Size*2 || !strings.HasPrefix(value, prefix) {
		return false
	}
	_, err := hex.DecodeString(value[len(prefix):])
	return err == nil
}

func validateMetadata(name, value string, required bool) error {
	trimmed := strings.TrimSpace(value)
	if required && trimmed == "" {
		return fmt.Errorf("%w: %s is required", ErrInvalidTrace, name)
	}
	if trimmed == "" {
		return nil
	}
	if len(trimmed) > 256 || strings.ContainsAny(trimmed, "\r\n") {
		return fmt.Errorf("%w: %s must be compact single-line metadata", ErrInvalidTrace, name)
	}
	lower := strings.ToLower(trimmed)
	for _, forbidden := range []string{
		"password", "passwd", "secret", "token", "api_key", "apikey", "authorization",
		"bearer ", "cookie", "set-cookie", "private_key", "private key", "access_token", "refresh_token", "raw_chat",
		"raw chat", "raw_prompt", "raw prompt", "conversation transcript",
	} {
		if strings.Contains(lower, forbidden) {
			return fmt.Errorf("%w: %s contains forbidden secret or raw-chat metadata", ErrInvalidTrace, name)
		}
	}
	for _, pattern := range secretMetadataPatterns {
		if pattern.MatchString(trimmed) {
			return fmt.Errorf("%w: %s contains a secret-shaped metadata value", ErrInvalidTrace, name)
		}
	}
	if highEntropyMetadata(trimmed) {
		return fmt.Errorf("%w: %s contains a high-entropy metadata value", ErrInvalidTrace, name)
	}
	return nil
}

var secretMetadataPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)-----BEGIN(?: [A-Z]+)? PRIVATE KEY-----`),
	regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{20,}|glpat-[A-Za-z0-9_-]{20,})\b`),
	regexp.MustCompile(`(?i)\b(?:password|passwd|token|secret|authorization|cookie)\s*[:=]`),
}

func highEntropyMetadata(value string) bool {
	if len(value) < 32 || strings.Contains(value, " ") {
		return false
	}
	classes := 0
	for _, pattern := range []string{`[a-z]`, `[A-Z]`, `[0-9]`, `[-_./+=]`} {
		if regexp.MustCompile(pattern).MatchString(value) {
			classes++
		}
	}
	return classes >= 3
}

// TraceSink is an append-only destination for validated trace events.
type TraceSink interface {
	Append(context.Context, TraceEvent) error
}

// MemoryTraceSink is a synchronized append-only sink intended for tests and ephemeral use.
type MemoryTraceSink struct {
	mu     sync.RWMutex
	events []TraceEvent
}

// NewMemoryTraceSink creates an empty in-memory sink.
func NewMemoryTraceSink() *MemoryTraceSink {
	return &MemoryTraceSink{events: make([]TraceEvent, 0)}
}

// Append validates and stores a defensive copy of event.
func (sink *MemoryTraceSink) Append(ctx context.Context, event TraceEvent) error {
	if sink == nil {
		return fmt.Errorf("append memory trace: %w: nil sink", ErrInvalidTrace)
	}
	if ctx == nil {
		return fmt.Errorf("append memory trace: %w: nil context", ErrInvalidTrace)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("append memory trace: %w", err)
	}
	if err := event.Validate(); err != nil {
		return fmt.Errorf("append memory trace: %w", err)
	}

	sink.mu.Lock()
	sink.events = append(sink.events, cloneTraceEvent(event))
	sink.mu.Unlock()
	return nil
}

// Events returns a defensive snapshot in append order.
func (sink *MemoryTraceSink) Events() []TraceEvent {
	if sink == nil {
		return nil
	}
	sink.mu.RLock()
	defer sink.mu.RUnlock()
	events := make([]TraceEvent, len(sink.events))
	for index, event := range sink.events {
		events[index] = cloneTraceEvent(event)
	}
	return events
}

// FileTraceSink appends JSON Lines under one source root.
type FileTraceSink struct {
	mu           sync.Mutex
	sourceRoot   string
	relativePath string
}

// NewFileTraceSink creates a source-local sink. relativePath must identify a
// file beneath sourceRoot without symlink or parent traversal.
func NewFileTraceSink(sourceRoot, relativePath string) (*FileTraceSink, error) {
	if strings.TrimSpace(sourceRoot) == "" {
		return nil, fmt.Errorf("create file trace sink: %w: source root is required", ErrUnsafeTraceSink)
	}
	absRoot, err := filepath.Abs(sourceRoot)
	if err != nil {
		return nil, fmt.Errorf("create file trace sink: resolve source root: %w", err)
	}
	rootInfo, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("create file trace sink: inspect source root: %w", err)
	}
	if !rootInfo.IsDir() {
		return nil, fmt.Errorf("create file trace sink: %w: source root is not a directory", ErrUnsafeTraceSink)
	}

	cleanPath := filepath.Clean(relativePath)
	if strings.TrimSpace(relativePath) == "" || cleanPath == "." || filepath.IsAbs(relativePath) || cleanPath != relativePath {
		return nil, fmt.Errorf("create file trace sink: %w: path must be a clean relative file path", ErrUnsafeTraceSink)
	}
	for _, part := range strings.Split(cleanPath, string(filepath.Separator)) {
		if part == "" || part == ".." {
			return nil, fmt.Errorf("create file trace sink: %w: path escapes source root", ErrUnsafeTraceSink)
		}
	}
	if err := rejectSymlinkComponents(absRoot, cleanPath); err != nil {
		return nil, fmt.Errorf("create file trace sink: %w", err)
	}

	return &FileTraceSink{sourceRoot: absRoot, relativePath: cleanPath}, nil
}

// Append validates event and atomically appends one JSON line.
func (sink *FileTraceSink) Append(ctx context.Context, event TraceEvent) error {
	if sink == nil {
		return fmt.Errorf("append file trace: %w: nil sink", ErrUnsafeTraceSink)
	}
	if ctx == nil {
		return fmt.Errorf("append file trace: %w: nil context", ErrInvalidTrace)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("append file trace: %w", err)
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("append file trace: %w", err)
	}
	data = append(data, '\n')

	sink.mu.Lock()
	defer sink.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("append file trace: %w", err)
	}
	if err := rejectSymlinkComponents(sink.sourceRoot, sink.relativePath); err != nil {
		return fmt.Errorf("append file trace: %w", err)
	}

	root, err := os.OpenRoot(sink.sourceRoot)
	if err != nil {
		return fmt.Errorf("append file trace: open source root: %w", err)
	}
	defer root.Close()

	parent := filepath.Dir(sink.relativePath)
	if parent != "." {
		if err := root.MkdirAll(parent, 0o700); err != nil {
			return fmt.Errorf("append file trace: create trace directory: %w", err)
		}
		parentDir, err := root.Open(parent)
		if err != nil {
			return fmt.Errorf("append file trace: open trace directory: %w", err)
		}
		if err := parentDir.Chmod(0o700); err != nil {
			parentDir.Close()
			return fmt.Errorf("append file trace: secure trace directory: %w", err)
		}
		if err := parentDir.Close(); err != nil {
			return fmt.Errorf("append file trace: close trace directory: %w", err)
		}
	}

	file, err := root.OpenFile(sink.relativePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("append file trace: open trace file: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return fmt.Errorf("append file trace: secure trace file: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("append file trace: inspect trace file: %w", err)
	}
	if !info.Mode().IsRegular() {
		file.Close()
		return fmt.Errorf("append file trace: %w: trace path is not a regular file", ErrUnsafeTraceSink)
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("append file trace: write event: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("append file trace: sync event: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("append file trace: close trace file: %w", err)
	}
	return nil
}

func rejectSymlinkComponents(root, relativePath string) error {
	current := root
	parts := strings.Split(relativePath, string(filepath.Separator))
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return fmt.Errorf("inspect trace path: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: trace path contains a symlink", ErrUnsafeTraceSink)
		}
		if index < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("%w: trace path parent is not a directory", ErrUnsafeTraceSink)
		}
	}
	return nil
}

func cloneTraceEvent(event TraceEvent) TraceEvent {
	event.SourceRequests = cloneStrings(event.SourceRequests)
	event.ContextDecisions = cloneStrings(event.ContextDecisions)
	event.ToolCalls = cloneStrings(event.ToolCalls)
	event.Handoffs = cloneStrings(event.Handoffs)
	event.EvalsPassed = cloneStrings(event.EvalsPassed)
	event.EvalsFailed = cloneStrings(event.EvalsFailed)
	return event
}

func cloneStrings(values []string) []string {
	result := make([]string, len(values))
	copy(result, values)
	return result
}

// RouteRequestWithTrace explicitly traces a non-trivial routing decision.
// RouteRequest remains untraced; callers must opt into this API with a sink.
func RouteRequestWithTrace(ctx context.Context, request string, input TraceEventInput, sink TraceSink) (ServiceArea, error) {
	if ctx == nil {
		return "", fmt.Errorf("route request with trace: %w: nil context", ErrInvalidTrace)
	}
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("route request with trace: %w", err)
	}
	if sink == nil {
		return "", fmt.Errorf("route request with trace: %w: trace sink is required", ErrInvalidTrace)
	}
	area, err := RouteRequest(request)
	if err != nil {
		return "", fmt.Errorf("route request with trace: %w", err)
	}
	input.ServiceArea = area
	input.DecisionPacket.ServiceArea = area
	event, err := NewTraceEvent(input)
	if err != nil {
		return "", fmt.Errorf("route request with trace: %w", err)
	}
	if err := sink.Append(ctx, event); err != nil {
		return "", fmt.Errorf("route request with trace: append decision: %w", err)
	}
	return area, nil
}

var _ TraceSink = (*MemoryTraceSink)(nil)
var _ TraceSink = (*FileTraceSink)(nil)
