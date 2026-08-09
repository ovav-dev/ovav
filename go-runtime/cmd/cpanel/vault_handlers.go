// OVAV cPanel — Vault HTTP handlers.
//
// Endpoints for per-user secret management.
// All require Bearer JWT with user_id claim.
// Tier-based slot limits applied at add time.

package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// ── Vault HTTP middleware ─────────────────────────────────────────────────────

// requireUser extracts and validates user_id from JWT.
func requireUser(r *http.Request) (string, *User, error) {
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" || token == authHeader {
		return "", nil, &apiError{"no token", http.StatusUnauthorized}
	}

	claims, err := verifyJWT(token)
	if err != nil {
		return "", nil, &apiError{err.Error(), http.StatusUnauthorized}
	}
	if claims.UserID == "" {
		return "", nil, &apiError{"not a user account", http.StatusUnauthorized}
	}

	store := getUserStore()
	user := store.GetByID(claims.UserID)
	if user == nil {
		return "", nil, &apiError{"user not found", http.StatusNotFound}
	}

	return claims.UserID, user, nil
}

type apiError struct {
	msg  string
	code int
}

func (e *apiError) Error() string { return e.msg }

// ── Handlers ─────────────────────────────────────────────────────────────────

// handleVaultSecretsList returns all secrets (metadata only, no values).
// GET /api/v1/vault/secrets
func handleVaultSecretsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, err := requireUser(r)
	if err != nil {
		sendError(w, err.Error(), err.(*apiError).code)
		return
	}

	secrets, err := ListSecrets(userID)
	if err != nil {
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	count := CountSecrets(userID)
	limit := SlotLimit(userID)

	sendOK(w, map[string]interface{}{
		"secrets": secrets,
		"count":   count,
		"limit":   limit,
		"tier":    getUserTier(userID),
	})
}

// handleVaultSecretGet returns a single secret with its decrypted value.
// GET /api/v1/vault/secrets/:name
func handleVaultSecretGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, err := requireUser(r)
	if err != nil {
		sendError(w, err.Error(), err.(*apiError).code)
		return
	}

	name := extractURLParam(r, "name")
	if name == "" {
		sendError(w, "secret name required", http.StatusBadRequest)
		return
	}

	secret, err := GetSecret(userID, name)
	if err != nil {
		sendError(w, err.Error(), http.StatusNotFound)
		return
	}

	sendOK(w, map[string]interface{}{
		"id":          secret.ID,
		"name":        secret.Name,
		"type":        secret.Type,
		"value":       secret.Value, // decrypted if key available
		"metadata":    secret.Metadata,
		"source":      secret.Source,
		"source_path": secret.SourcePath,
		"created_at":  secret.CreatedAt,
		"updated_at":  secret.UpdatedAt,
	})
}

// handleVaultSecretAdd adds a new secret.
// POST /api/v1/vault/secrets
func handleVaultSecretAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, err := requireUser(r)
	if err != nil {
		sendError(w, err.Error(), err.(*apiError).code)
		return
	}

	var body struct {
		Name       string            `json:"name"`
		Value      string            `json:"value"`
		Type       string            `json:"type"`
		Source     string            `json:"source"`
		SourcePath string            `json:"source_path"`
		Metadata   map[string]string `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if body.Name == "" || body.Value == "" {
		sendError(w, "name and value are required", http.StatusBadRequest)
		return
	}

	// Infer type if not provided
	secretType := body.Type
	if secretType == "" {
		secretType = inferSecretType(body.Name)
	}
	if body.Source == "" {
		body.Source = "manual"
	}

	secret, err := AddSecret(userID, body.Name, body.Value, secretType, body.Source, body.SourcePath, body.Metadata)
	if err != nil {
		sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	sendOK(w, map[string]interface{}{
		"id":      secret.ID,
		"name":    secret.Name,
		"type":    secret.Type,
		"source":  secret.Source,
		"message": "secret added successfully",
	})
}

// handleVaultSecretDelete deletes a secret.
// DELETE /api/v1/vault/secrets/:name
func handleVaultSecretDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, err := requireUser(r)
	if err != nil {
		sendError(w, err.Error(), err.(*apiError).code)
		return
	}

	name := extractURLParam(r, "name")
	if name == "" {
		sendError(w, "secret name required", http.StatusBadRequest)
		return
	}

	if err := RemoveSecret(userID, name); err != nil {
		sendError(w, err.Error(), http.StatusNotFound)
		return
	}

	sendOK(w, map[string]interface{}{
		"name":    name,
		"message": "secret deleted successfully",
	})
}

// handleVaultUserLogin handles user vault login — sets the vault encryption key.
// POST /api/v1/vault/login { email, password }
func handleVaultUserLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	store := getUserStore()
	user, err := store.Authenticate(body.Email, body.Password)
	if err != nil {
		sendError(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	// Derive vault key and set it in the vault store for this user
	vaultKey := GetUserVaultKey(body.Email, body.Password)
	SetVaultKey(user.ID, vaultKey)

	// Issue JWT
	if err := initJWT(); err != nil {
		sendError(w, "JWT init failed", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	claims := jwtClaims{
		Sub:    user.ID,
		Role:   "user",
		UserID: user.ID,
		Email:  user.Email,
		Iat:    now.Unix(),
		Exp:    now.Add(24 * time.Hour).Unix(),
	}

	token, err := signJWT(claims)
	if err != nil {
		sendError(w, "JWT signing failed", http.StatusInternalServerError)
		return
	}

	jwtSessLock.Lock()
	jwtSessions[token] = sessionInfo{
		Token:     token,
		Role:      "user",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	jwtSessLock.Unlock()

	sendOK(w, map[string]interface{}{
		"token":      token,
		"user_id":    user.ID,
		"email":      user.Email,
		"tier":       user.Tier,
		"slots":      TierSlots[user.Tier],
		"expires_at": claims.Exp,
		"token_type": "Bearer",
	})
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func getUserTier(userID string) string {
	store := getUserStore()
	user := store.GetByID(userID)
	if user == nil {
		return "free"
	}
	return user.Tier
}

func inferSecretType(name string) string {
	upper := strings.ToUpper(name)
	if strings.Contains(upper, "GITHUB") || strings.Contains(upper, "GH_") {
		return "api_token"
	}
	if strings.Contains(upper, "AWS") || strings.Contains(upper, "AZURE") || strings.Contains(upper, "GCP") || strings.Contains(upper, "CLOUD") {
		return "cloud_key"
	}
	if strings.Contains(upper, "DB_") || strings.Contains(upper, "DATABASE") || strings.Contains(upper, "POSTGRES") || strings.Contains(upper, "MYSQL") {
		return "db_credential"
	}
	if strings.Contains(upper, "OAUTH") || strings.Contains(upper, "BEARER") || strings.Contains(upper, "ACCESS_TOKEN") {
		return "oauth_creds"
	}
	if strings.Contains(upper, "HMAC") || strings.Contains(upper, "ENCRYPTION_KEY") || strings.Contains(upper, "SECRET_KEY") {
		return "encryption_key"
	}
	return "api_token"
}

// extractURLParam extracts a path parameter from the request.
// This is a simple implementation — in production you'd use gorilla/mux or chi.
func extractURLParam(r *http.Request, name string) string {
	// Match :name pattern in the route
	// Simple approach: trim the prefix based on known routes
	path := r.URL.Path

	// For /api/v1/vault/secrets/:name
	if strings.HasPrefix(path, "/api/v1/vault/secrets/") {
		return strings.TrimPrefix(path, "/api/v1/vault/secrets/")
	}
	return ""
}
