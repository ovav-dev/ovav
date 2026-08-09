package data

import (
	"runtime"

	"github.com/ovav/ovav/internal/cli"
	"github.com/ovav/ovav/internal/doctor"
)

// SystemInfo holds all health/status data for the Cockpit.
type SystemInfo struct {
	// Git
	Branch    string
	SHA       string
	Dirty     string // "clean" or "dirty"
	Remote    string
	HasRemote bool

	// OVAV
	OVAVRoot string

	// Go
	GoVersion string

	// Caps
	PlanVersion string
	CapsDone    int
	CapsPending int
	Strategy    string

	// Doctor
	DoctorPass  int
	DoctorFail  int
	DoctorWarn  int
	DoctorTotal int
}

// GatherSystemInfo collects all system health data.
func GatherSystemInfo(root string, caps *CapsData, quick bool) SystemInfo {
	info := SystemInfo{
		OVAVRoot:  root,
		GoVersion: runtime.Version(),
	}

	// Git info via cli package
	info.Branch, info.SHA, info.Dirty = cli.GitInfo()
	info.HasRemote = cli.HasGitRemote()
	if info.HasRemote {
		info.Remote = cli.GitRemoteURL()
	}

	// Caps summary
	if caps != nil {
		info.PlanVersion = caps.PlanVersion
		info.Strategy = caps.Strategy
		info.CapsDone = countDoneCaps(caps)
		info.CapsPending = len(caps.Pending)
	}

	// Doctor checks
	results := doctor.Run(quick)
	info.DoctorTotal = len(results)
	for _, r := range results {
		switch r.Status {
		case "pass":
			info.DoctorPass++
		case "fail":
			info.DoctorFail++
		case "warn":
			info.DoctorWarn++
		}
	}

	return info
}

func countDoneCaps(caps *CapsData) int {
	n := 0
	for _, c := range caps.Caps {
		if c.Status == "done" {
			n++
		}
	}
	return n
}
