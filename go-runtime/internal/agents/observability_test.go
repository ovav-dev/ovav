package agents

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestNewTraceEventDeterministic(t *testing.T) {
	input := validTraceEventInput()

	first, err := NewTraceEvent(input)
	if err != nil {
		t.Fatalf("create first trace event: %v", err)
	}
	second, err := NewTraceEvent(input)
	if err != nil {
		t.Fatalf("create second trace event: %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same input produced different events:\nfirst:  %#v\nsecond: %#v", first, second)
	}
	if first.TraceID == "" || first.Timestamp != input.Timestamp {
		t.Fatalf("deterministic fields not preserved: %#v", first)
	}
}

func TestTraceEventValidate(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*TraceEvent)
	}{
		{name: "missing trace ID", mutate: func(event *TraceEvent) { event.TraceID = "" }},
		{name: "missing service area", mutate: func(event *TraceEvent) { event.ServiceArea = "" }},
		{name: "missing lead", mutate: func(event *TraceEvent) { event.Lead = "" }},
		{name: "missing mode", mutate: func(event *TraceEvent) { event.Mode = "" }},
		{name: "missing decision packet field", mutate: func(event *TraceEvent) { event.DecisionPacket.RiskLevel = "" }},
		{name: "missing delivery contract", mutate: func(event *TraceEvent) { event.DeliveryContract = "" }},
		{name: "missing eval results", mutate: func(event *TraceEvent) {
			event.EvalsPassed = nil
			event.EvalsFailed = nil
		}},
		{name: "inconsistent packet service area", mutate: func(event *TraceEvent) { event.DecisionPacket.ServiceArea = ServiceAreaResearch }},
		{name: "negative token estimate", mutate: func(event *TraceEvent) { event.TokenUsageEstimate = -1 }},
		{name: "secret metadata", mutate: func(event *TraceEvent) { event.SourceRequests = []string{"password=hunter2"} }},
		{name: "raw chat metadata", mutate: func(event *TraceEvent) { event.ContextDecisions = []string{"raw_chat:user-message"} }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := mustTraceEvent(t, validTraceEventInput())
			test.mutate(&event)
			if err := event.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestTraceMetadataRejectsSecretShapesAndEntropy(t *testing.T) {
	values := []string{
		"token=ghp_0123456789abcdefghijklmnop",
		"cookie: session=0123456789abcdef",
		"Authorization: Basic Zm9vOmJhcg==",
		"-----BEGIN PRIVATE KEY-----",
		"AKIAIOSFODNN7EXAMPLE",
		"mF_9.B5f-4.1JqM2pR8xY7zK6vW3nT0sL",
	}
	for _, value := range values {
		t.Run(value[:min(len(value), 20)], func(t *testing.T) {
			if err := validateMetadata("test", value, true); err == nil {
				t.Fatalf("validateMetadata(%q) accepted secret-like value", value)
			}
		})
	}
}

func TestTraceEventJSONMatchesPolicyFields(t *testing.T) {
	event := mustTraceEvent(t, validTraceEventInput())
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal trace event: %v", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("decode trace event: %v", err)
	}
	required := []string{
		"trace_id", "timestamp", "service_area", "lead", "mode", "decision_packet",
		"source_requests", "context_decisions", "tool_calls", "handoffs", "model_used",
		"token_usage_estimate", "cost_estimate", "evals_passed", "evals_failed", "delivery_contract",
	}
	for _, field := range required {
		if _, ok := fields[field]; !ok {
			t.Errorf("serialized trace missing policy field %q: %s", field, data)
		}
	}
	if len(fields) != len(required) {
		t.Errorf("serialized trace has fields outside policy: %s", data)
	}
	for _, forbidden := range []string{"secret", "raw_chat", "raw_prompt", "conversation", "message"} {
		if _, ok := fields[forbidden]; ok {
			t.Errorf("serialized trace exposes forbidden field %q", forbidden)
		}
	}
}

func TestMemoryTraceSinkConcurrentAppend(t *testing.T) {
	sink := NewMemoryTraceSink()
	const count = 100

	var wait sync.WaitGroup
	for i := range count {
		wait.Add(1)
		go func() {
			defer wait.Done()
			input := validTraceEventInput()
			input.TraceSeed = fmt.Sprintf("route-%d", i)
			event := mustTraceEvent(t, input)
			if err := sink.Append(context.Background(), event); err != nil {
				t.Errorf("append memory trace: %v", err)
			}
		}()
	}
	wait.Wait()

	if got := len(sink.Events()); got != count {
		t.Fatalf("got %d events, want %d", got, count)
	}
}

func TestFileTraceSinkConcurrentAppend(t *testing.T) {
	root := t.TempDir()
	relativePath := filepath.Join(".ovav", "traces", "events.jsonl")
	sink, err := NewFileTraceSink(root, relativePath)
	if err != nil {
		t.Fatalf("create file trace sink: %v", err)
	}

	const count = 100
	var wait sync.WaitGroup
	for i := range count {
		wait.Add(1)
		go func() {
			defer wait.Done()
			input := validTraceEventInput()
			input.TraceSeed = fmt.Sprintf("file-route-%d", i)
			event := mustTraceEvent(t, input)
			if err := sink.Append(context.Background(), event); err != nil {
				t.Errorf("append file trace: %v", err)
			}
		}()
	}
	wait.Wait()

	path := filepath.Join(root, relativePath)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat trace file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("trace file mode = %o, want 600", got)
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat trace directory: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Errorf("trace directory mode = %o, want 700", got)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open trace file: %v", err)
	}
	defer file.Close()

	lines := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event TraceEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("decode trace line %d: %v", lines, err)
		}
		lines++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan trace file: %v", err)
	}
	if lines != count {
		t.Fatalf("got %d complete trace lines, want %d", lines, count)
	}
}

func TestNewFileTraceSinkRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatalf("create test symlink: %v", err)
	}

	tests := []struct {
		name string
		path string
	}{
		{name: "empty", path: ""},
		{name: "root", path: "."},
		{name: "absolute", path: filepath.Join(root, "events.jsonl")},
		{name: "parent escape", path: filepath.Join("..", "events.jsonl")},
		{name: "symlink escape", path: filepath.Join("linked", "events.jsonl")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewFileTraceSink(root, test.path); err == nil {
				t.Fatal("expected unsafe path error")
			}
		})
	}
}

func TestTraceSinksRejectInvalidEvents(t *testing.T) {
	event := mustTraceEvent(t, validTraceEventInput())
	event.Lead = ""
	sinks := []struct {
		name string
		new  func(*testing.T) TraceSink
	}{
		{name: "memory", new: func(*testing.T) TraceSink { return NewMemoryTraceSink() }},
		{name: "file", new: func(t *testing.T) TraceSink {
			sink, err := NewFileTraceSink(t.TempDir(), filepath.Join("traces", "events.jsonl"))
			if err != nil {
				t.Fatalf("create file trace sink: %v", err)
			}
			return sink
		}},
	}

	for _, test := range sinks {
		t.Run(test.name, func(t *testing.T) {
			if err := test.new(t).Append(context.Background(), event); err == nil {
				t.Fatal("expected invalid event error")
			}
		})
	}
}

func TestRouteRequestWithTrace(t *testing.T) {
	sink := NewMemoryTraceSink()
	input := validTraceEventInput()
	input.ServiceArea = ""
	input.DecisionPacket.ServiceArea = ""

	area, err := RouteRequestWithTrace(context.Background(), "platform runtime change", input, sink)
	if err != nil {
		t.Fatalf("route request with trace: %v", err)
	}
	if area != ServiceAreaPlatform {
		t.Fatalf("area = %q, want %q", area, ServiceAreaPlatform)
	}
	events := sink.Events()
	if len(events) != 1 || events[0].ServiceArea != area || events[0].DecisionPacket.ServiceArea != area {
		t.Fatalf("routing trace not recorded consistently: %#v", events)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = RouteRequestWithTrace(canceled, "platform runtime change", input, sink)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected wrapped context cancellation, got %v", err)
	}
}

func validTraceEventInput() TraceEventInput {
	return TraceEventInput{
		TraceSeed:   "route-request-42",
		Timestamp:   time.Date(2026, time.August, 12, 10, 30, 0, 123, time.UTC),
		ServiceArea: ServiceAreaPlatform,
		Lead:        "lead-thavren",
		Mode:        TraceMode("focused_squad"),
		DecisionPacket: DecisionPacket{
			ServiceArea:      ServiceAreaPlatform,
			Lead:             "lead-thavren",
			TaskSize:         "medium",
			RiskLevel:        "medium",
			DelegationMode:   "focused_squad",
			ContextBudget:    "T3_focused",
			ModelTier:        "reasoning",
			ValidationMode:   "race",
			DeliveryContract: "implementation_delivery",
		},
		SourceRequests:     []string{"request-42"},
		ContextDecisions:   []string{"source-local"},
		ToolCalls:          []string{"go-test"},
		Handoffs:           []string{},
		ModelUsed:          "openai/gpt-5.6-luna",
		TokenUsageEstimate: 1200,
		CostEstimate:       0.02,
		EvalsPassed:        []string{"tool_capability_boundary", "delivery_contract_match"},
		EvalsFailed:        []string{},
		DeliveryContract:   "implementation_delivery",
	}
}

func mustTraceEvent(t *testing.T, input TraceEventInput) TraceEvent {
	t.Helper()
	event, err := NewTraceEvent(input)
	if err != nil {
		t.Fatalf("create trace event: %v", err)
	}
	return event
}
