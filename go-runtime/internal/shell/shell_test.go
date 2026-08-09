package shell

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// shell_test.go — Sprint 8 T13 (innovation #3)
// ═══════════════════════════════════════════════════════════════════════════

func TestT13NewObserver_NotNil(t *testing.T) {
	o := NewObserver()
	if o == nil {
		t.Fatal("NewObserver returned nil")
	}
	if len(o.hooks) != 0 {
		t.Errorf("expected 0 hooks, got %d", len(o.hooks))
	}
}

func TestT13Register_Hook(t *testing.T) {
	o := NewObserver()
	called := 0
	o.Register(func(e Event) {
		called++
	})
	if len(o.hooks) != 1 {
		t.Errorf("expected 1 hook, got %d", len(o.hooks))
	}
}

func TestT13Emit_Trigger(t *testing.T) {
	o := NewObserver()
	var mu sync.Mutex
	called := 0
	o.Register(func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		called++
	})
	o.Emit(Event{Kind: EventCommandStart, Command: "test"})
	o.Wait()
	mu.Lock()
	defer mu.Unlock()
	if called != 1 {
		t.Errorf("hook should fire once, got %d", called)
	}
}

func TestT13Emit_Multiple(t *testing.T) {
	o := NewObserver()
	var called atomic.Int32
	o.Register(func(e Event) {
		called.Add(1)
	})
	for i := 0; i < 5; i++ {
		o.Emit(Event{Kind: EventCommandEnd})
	}
	o.Wait()
	if called.Load() != 5 {
		t.Errorf("expected 5 fires, got %d", called.Load())
	}
}

func TestT13Start_Once(t *testing.T) {
	o := NewObserver()
	ctx := context.Background()
	if err := o.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := o.Start(ctx); err == nil {
		t.Error("second Start should error (already running)")
	}
	o.Stop()
}

func TestT13Stop_Idempotent(t *testing.T) {
	o := NewObserver()
	o.Stop()
	// Second stop should not panic
	o.Stop()
}

func TestT13Stop_BeforeStart(t *testing.T) {
	o := NewObserver()
	// Stop before Start — should not panic
	o.Stop()
}

func TestT13Suggest_ForcePush(t *testing.T) {
	s := Suggest("git push --force origin main")
	if len(s) == 0 {
		t.Error("force push should generate suggestion")
	}
}

func TestT13Suggest_RmRf(t *testing.T) {
	s := Suggest("rm -rf /")
	if len(s) == 0 {
		t.Error("rm -rf / should generate suggestion")
	}
}

func TestT13Suggest_RmRfHome(t *testing.T) {
	s := Suggest("sudo rm -rf ~")
	if len(s) == 0 {
		t.Error("rm -rf ~ should generate suggestion")
	}
}

func TestT13Suggest_Sudo(t *testing.T) {
	s := Suggest("sudo apt update")
	if len(s) == 0 {
		t.Error("sudo should generate suggestion")
	}
}

func TestT13Suggest_PipInstall(t *testing.T) {
	s := Suggest("pip install requests")
	if len(s) == 0 {
		t.Error("pip install should generate suggestion")
	}
}

func TestT13Suggest_NpmInstall(t *testing.T) {
	s := Suggest("npm install lodash")
	if len(s) == 0 {
		t.Error("npm install should generate suggestion")
	}
}

func TestT13Suggest_NetworkEgress(t *testing.T) {
	s := Suggest("curl http://example.com")
	if len(s) == 0 {
		t.Error("external curl should generate suggestion")
	}
}

func TestT13Suggest_LocalhostSafe(t *testing.T) {
	s := Suggest("curl http://localhost:8080/health")
	if len(s) != 0 {
		t.Errorf("localhost curl should be safe, got %d suggestions", len(s))
	}
}

func TestT13Suggest_SafeCommand(t *testing.T) {
	s := Suggest("git status")
	if len(s) != 0 {
		t.Errorf("git status should be safe, got %d suggestions", len(s))
	}
}

func TestT13IsRisky(t *testing.T) {
	if !IsRisky("sudo rm -rf /") {
		t.Error("sudo rm should be risky")
	}
	if !IsRisky("git push --force") {
		t.Error("force push should be risky")
	}
	if IsRisky("git status") {
		t.Error("git status should not be risky")
	}
	if IsRisky("ls") {
		t.Error("ls should not be risky")
	}
}

func TestT13Run_Success(t *testing.T) {
	o := NewObserver()
	var fired []Event
	var mu sync.Mutex
	o.Register(func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		fired = append(fired, e)
	})
	ctx := context.Background()
	code, err := o.Run(ctx, "true")
	if err != nil {
		t.Errorf("true should succeed, got: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code for true should be 0, got %d", code)
	}
	o.Wait()
	mu.Lock()
	defer mu.Unlock()
	if len(fired) < 2 {
		t.Errorf("expected 2 events (start+end), got %d", len(fired))
	}
}

func TestT13Run_Failure(t *testing.T) {
	o := NewObserver()
	var fired []Event
	var mu sync.Mutex
	o.Register(func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		fired = append(fired, e)
	})
	_, err := o.Run(context.Background(), "false")
	if err == nil {
		t.Error("false should fail")
	}
	o.Wait()
	mu.Lock()
	defer mu.Unlock()
	hasFail := false
	for _, e := range fired {
		if e.Kind == EventCommandFail {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("expected EventCommandFail")
	}
}

func TestT13MarshalEvent(t *testing.T) {
	e := Event{
		Kind:     EventCommandEnd,
		Command:  "test",
		ExitCode: 0,
	}
	data, err := MarshalEvent(e)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got Event
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Command != "test" {
		t.Errorf("roundtrip mismatch: got %q", got.Command)
	}
}

func TestT13UnmarshalEvent(t *testing.T) {
	data := []byte(`{"kind":"command.start","command":"x","args":["a","b"]}`)
	e, err := UnmarshalEvent(data)
	if err != nil {
		t.Fatal(err)
	}
	if e.Kind != EventCommandStart {
		t.Errorf("expected CommandStart, got %q", e.Kind)
	}
	if len(e.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(e.Args))
	}
}

func TestT13EventKind_StringValues(t *testing.T) {
	kinds := []EventKind{
		EventCommandStart,
		EventCommandEnd,
		EventCommandFail,
	}
	for _, k := range kinds {
		if string(k) == "" {
			t.Errorf("EventKind empty")
		}
	}
}

func TestT13Event_StructFields(t *testing.T) {
	e := Event{
		Kind:     EventCommandEnd,
		Command:  "ls",
		Args:     []string{"-la"},
		ExitCode: 0,
	}
	if e.Kind != EventCommandEnd || e.Command != "ls" {
		t.Error("Event fields should match")
	}
}
