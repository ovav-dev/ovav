package permissions

import (
	"errors"
	"os"
	"testing"
)

func TestSimulateCases(t *testing.T) {
	tests := []struct {
		name    string
		input   SimulationCase
		allowed bool
		wantErr bool
	}{
		{
			name:    "allow",
			input:   SimulationCase{Name: "git read", Governor: GovernorBash, Input: "git status", Operator: "andres", ExpectAllowed: true},
			allowed: true,
		},
		{
			name:    "deny",
			input:   SimulationCase{Name: "sudo", Governor: GovernorBash, Input: "sudo id", Operator: "andres", ExpectAllowed: false},
			allowed: false,
		},
		{
			name:    "unknown defaults deny",
			input:   SimulationCase{Name: "unknown state", Governor: GovernorNewState, Input: "not_a_state", ExpectAllowed: false},
			allowed: false,
		},
		{
			name:    "failed expectation",
			input:   SimulationCase{Name: "wrong policy expectation", Governor: GovernorSystemPath, Input: "/etc/hosts", Operation: "read", ExpectAllowed: false},
			allowed: true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := SimulateCases([]SimulationCase{tt.input})
			if (err != nil) != tt.wantErr {
				t.Fatalf("SimulateCases() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(results) != 1 {
				t.Fatalf("got %d results, want 1", len(results))
			}
			if results[0].Allowed != tt.allowed {
				t.Errorf("Allowed = %v, want %v", results[0].Allowed, tt.allowed)
			}
			if results[0].Passed == tt.wantErr {
				t.Errorf("Passed = %v, want %v", results[0].Passed, !tt.wantErr)
			}
		})
	}
}

func TestSimulateCasesDetectsPolicyContradiction(t *testing.T) {
	cases := []SimulationCase{
		{Name: "allow expectation", Governor: GovernorPlugin, Input: "local", Operator: "thavren", PluginType: "native", ExpectAllowed: true},
		{Name: "deny expectation", Governor: GovernorPlugin, Input: "local", Operator: "thavren", PluginType: "native", ExpectAllowed: false},
	}

	results, err := SimulateCases(cases)
	if !errors.Is(err, ErrPolicyContradiction) {
		t.Fatalf("error = %v, want ErrPolicyContradiction", err)
	}
	if len(results) != len(cases) {
		t.Fatalf("got %d results, want %d", len(results), len(cases))
	}
	for _, result := range results {
		if !result.Contradiction || result.Passed {
			t.Errorf("contradiction result = %+v", result)
		}
	}
}

func TestSimulateExercisesCanonicalGovernors(t *testing.T) {
	if err := Simulate(); err != nil {
		t.Fatalf("Simulate() error = %v", err)
	}

	seen := make(map[Governor]bool)
	for _, test := range DefaultSimulationCases() {
		seen[test.Governor] = true
	}
	for _, governor := range []Governor{GovernorBash, GovernorSystemPath, GovernorPlugin, GovernorNewState} {
		if !seen[governor] {
			t.Errorf("default simulation does not exercise %s", governor)
		}
	}
}

func TestVerifyPermissionAuthorityVersions(t *testing.T) {
	tests := []struct {
		name   string
		policy string
		legacy bool
		v3     bool
	}{
		{name: "legacy v2", policy: `{"schema_version":"ovav.permission_authority.v2"}`, legacy: true},
		{name: "valid v3", policy: `{"schema_version":"ovav.permission_authority.v3","authority":"OVAV","governor":"Thavren","materialized_targets":["opencode.json"]}`, legacy: true, v3: true},
		{name: "invalid v3", policy: `{"schema_version":"ovav.permission_authority.v3","authority":"OVAV"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := t.TempDir() + "/permission_authority.json"
			if err := os.WriteFile(path, []byte(tt.policy), 0o600); err != nil {
				t.Fatalf("write policy: %v", err)
			}
			if got := VerifyPermissionAuthority(path); got != tt.legacy {
				t.Errorf("VerifyPermissionAuthority() = %v, want %v", got, tt.legacy)
			}
			if got := VerifyPermissionAuthorityV3(path); got != tt.v3 {
				t.Errorf("VerifyPermissionAuthorityV3() = %v, want %v", got, tt.v3)
			}
		})
	}
}
