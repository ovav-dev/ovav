package data

import (
	"testing"
)

func TestCountDoneCaps(t *testing.T) {
	t.Run("mixed caps", func(t *testing.T) {
		caps := &CapsData{
			Caps: map[string]Cap{
				"C1": {Status: "done"},
				"C2": {Status: "done"},
				"C3": {Status: "pending"},
				"C4": {Status: "in_progress"},
			},
		}
		if n := countDoneCaps(caps); n != 2 {
			t.Errorf("expected 2, got %d", n)
		}
	})

	t.Run("all done", func(t *testing.T) {
		caps := &CapsData{
			Caps: map[string]Cap{
				"C1": {Status: "done"},
				"C2": {Status: "done"},
			},
		}
		if n := countDoneCaps(caps); n != 2 {
			t.Errorf("expected 2, got %d", n)
		}
	})

	t.Run("none done", func(t *testing.T) {
		caps := &CapsData{
			Caps: map[string]Cap{
				"C1": {Status: "pending"},
				"C2": {Status: "in_progress"},
			},
		}
		if n := countDoneCaps(caps); n != 0 {
			t.Errorf("expected 0, got %d", n)
		}
	})

	t.Run("empty caps", func(t *testing.T) {
		caps := &CapsData{Caps: map[string]Cap{}}
		if n := countDoneCaps(caps); n != 0 {
			t.Errorf("expected 0, got %d", n)
		}
	})
}

func TestGatherSystemInfo_WithNilCaps(t *testing.T) {
	// GatherSystemInfo should not panic with nil caps
	info := GatherSystemInfo("/tmp/nonexistent", nil, true)
	if info.OVAVRoot != "/tmp/nonexistent" {
		t.Errorf("expected OVAVRoot /tmp/nonexistent, got %q", info.OVAVRoot)
	}
	if info.GoVersion == "" {
		t.Error("expected non-empty GoVersion")
	}
	if info.PlanVersion != "" {
		t.Errorf("expected empty PlanVersion with nil caps, got %q", info.PlanVersion)
	}
	if info.CapsDone != 0 {
		t.Errorf("expected 0 CapsDone with nil caps, got %d", info.CapsDone)
	}
}

func TestGatherSystemInfo_WithCaps(t *testing.T) {
	caps := &CapsData{
		PlanVersion: "v1.9",
		Strategy:    "Go+TS",
		Caps: map[string]Cap{
			"C1": {Status: "done"},
			"C2": {Status: "done"},
			"C3": {Status: "pending"},
		},
		Pending: []PendingCap{
			{Name: "P1"},
			{Name: "P2"},
		},
	}
	info := GatherSystemInfo("/tmp/test", caps, true)
	if info.PlanVersion != "v1.9" {
		t.Errorf("expected v1.9, got %q", info.PlanVersion)
	}
	if info.Strategy != "Go+TS" {
		t.Errorf("expected Go+TS, got %q", info.Strategy)
	}
	if info.CapsDone != 2 {
		t.Errorf("expected 2 CapsDone, got %d", info.CapsDone)
	}
	if info.CapsPending != 2 {
		t.Errorf("expected 2 CapsPending, got %d", info.CapsPending)
	}
}

func TestGatherSystemInfo_GoVersion(t *testing.T) {
	info := GatherSystemInfo(".", nil, true)
	if info.GoVersion == "" {
		t.Error("expected GoVersion to be set from runtime.Version()")
	}
}

func TestSystemInfoStruct(t *testing.T) {
	// Verify all fields are accessible
	info := SystemInfo{
		Branch:      "main",
		SHA:         "abc123",
		Dirty:       "clean",
		Remote:      "https://github.com/test",
		HasRemote:   true,
		OVAVRoot:    "/test",
		GoVersion:   "go1.22",
		PlanVersion: "v1.9",
		CapsDone:    5,
		CapsPending: 3,
		Strategy:    "Go+TS",
		DoctorPass:  8,
		DoctorFail:  1,
		DoctorWarn:  2,
		DoctorTotal: 11,
	}
	if info.Branch != "main" {
		t.Error("SystemInfo Branch field not accessible")
	}
	if info.DoctorTotal != 11 {
		t.Error("SystemInfo DoctorTotal field not accessible")
	}
}
