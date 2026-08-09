// OVAV cPanel v5.1 — Identity verification endpoint.
//
// POST /api/v1/auth/cli-verify — Verify a CLI identity against the registry.
// Accepts {key_hash, challenge} and returns identity info if matched.
// Enables future web login integration without building the full UI now.

package main

import (
	"encoding/json"
	"net/http"

	"github.com/ovav/ovav/internal/identity"
)

// ── CLI verify endpoint ──────────────────────────────────────────────────────

type cliVerifyRequest struct {
	KeyHash   string `json:"key_hash"`
	Challenge string `json:"challenge,omitempty"`
}

type cliVerifyResponse struct {
	Valid    bool             `json:"valid"`
	Identity *identityPayload `json:"identity,omitempty"`
	Error    string           `json:"error,omitempty"`
}

type identityPayload struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	Level       int      `json:"level"`
	Permissions []string `json:"permissions"`
	Status      string   `json:"status"`
}

func handleCLIVerify(w http.ResponseWriter, r *http.Request) {
	var req cliVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, cliVerifyResponse{
			Valid: false,
			Error: "invalid request body",
		}, http.StatusBadRequest)
		return
	}

	if req.KeyHash == "" {
		sendJSON(w, cliVerifyResponse{
			Valid: false,
			Error: "key_hash is required",
		}, http.StatusBadRequest)
		return
	}

	// Load identity registry from repo root
	reg, err := identity.LoadRegistry(RepoRoot)
	if err != nil {
		sendJSON(w, cliVerifyResponse{
			Valid: false,
			Error: "identity registry unavailable",
		}, http.StatusInternalServerError)
		return
	}

	// Find identity by key_hash
	id, err := identity.FindIdentity(reg, req.KeyHash)
	if err != nil {
		sendJSON(w, cliVerifyResponse{
			Valid: false,
			Error: "identity not recognized",
		}, http.StatusUnauthorized)
		return
	}

	// Identity found and active
	sendOK(w, cliVerifyResponse{
		Valid: true,
		Identity: &identityPayload{
			ID:          id.ID,
			Name:        id.Name,
			Role:        id.Role,
			Level:       id.Level,
			Permissions: id.Permissions,
			Status:      id.Status,
		},
	})
}
