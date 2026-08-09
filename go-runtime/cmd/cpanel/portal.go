// OVAV Portal v1.0 — User-facing endpoints (usage, API keys, portal-specific).
//
// All routes require user JWT authentication (Bearer token from user-login/register).
// These are separate from admin cPanel routes.

package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ── Portal store (API keys + usage per user) ─────────────────────────────────

type APIKey struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	KeyHash   [32]byte  `json:"-"`   // never exposed, stored as hash
	KeyPrefix string    `json:"key"` // first 8 chars only
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used,omitempty"`
}

type PortalData struct {
	mu         sync.RWMutex
	APIKeys    []APIKey   `json:"api_keys"`
	Usage      UsageStats `json:"usage"`
	APIKeyPath string     `json:"-"`
}

type UsageStats struct {
	SecretsCount int    `json:"secrets_count"`
	APIHits      int    `json:"api_hits"` // incremented by portal middleware
	SyncCount    int    `json:"sync_count"`
	Tier         string `json:"tier"`
	Plan         string `json:"plan"`
	SlotsTotal   int    `json:"slots_total"`
}

var portalStore *PortalData
var portalStoreOnce sync.Once

func getPortalStore(userID string) *PortalData {
	portalStoreOnce.Do(func() {
		portalStore = &PortalData{
			APIKeys:    []APIKey{},
			Usage:      UsageStats{Tier: "free", Plan: "free", SlotsTotal: 5},
			APIKeyPath: filepath.Join(RepoRoot, ".ovav", "vault", "portal.json"),
		}
		portalStore.load()
	})
	return portalStore
}

// randomID generates a random alphanumeric string of n chars.
func randomID(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}

func (p *PortalData) load() {
	data, err := os.ReadFile(p.APIKeyPath)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		return
	}
	json.Unmarshal(data, &p.APIKeys)
}

func (p *PortalData) save() error {
	data, err := json.Marshal(p.APIKeys)
	if err != nil {
		return err
	}
	return os.WriteFile(p.APIKeyPath, data, 0600)
}

// ── Portal auth middleware — validates user JWT, extracts userID ─────────────

func portalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" || token == auth {
			sendError(w, "no token", http.StatusUnauthorized)
			return
		}
		claims, err := verifyJWT(token)
		if err != nil {
			sendError(w, "invalid: "+err.Error(), http.StatusUnauthorized)
			return
		}
		if claims.UserID == "" {
			sendError(w, "not a user session", http.StatusUnauthorized)
			return
		}
		// Increment API hits
		p := getPortalStore(claims.UserID)
		p.mu.Lock()
		p.Usage.APIHits++
		p.mu.Unlock()
		r = setPortalClaims(r, claims)
		next(w, r)
	}
}

type portalClaimsKey string

const portalClaimsCtxKey portalClaimsKey = "portalClaims"

func setPortalClaims(r *http.Request, claims *jwtClaims) *http.Request {
	ctx := context.WithValue(r.Context(), portalClaimsCtxKey, claims)
	return r.WithContext(ctx)
}

func getPortalClaims(r *http.Request) *jwtClaims {
	claims, _ := r.Context().Value(portalClaimsCtxKey).(*jwtClaims)
	return claims
}

// ── GET /api/v1/portal/me ────────────────────────────────────────────────────

func handlePortalMe(w http.ResponseWriter, r *http.Request) {
	claims := getPortalClaims(r)
	if claims == nil {
		sendError(w, "not authenticated", http.StatusUnauthorized)
		return
	}
	store := getUserStore()
	user := store.GetByID(claims.UserID)
	if user == nil {
		sendError(w, "user not found", http.StatusNotFound)
		return
	}
	p := getPortalStore(claims.UserID)
	p.mu.RLock()
	secretsCount := p.Usage.SecretsCount
	apiHits := p.Usage.APIHits
	syncCount := p.Usage.SyncCount
	p.mu.RUnlock()
	sendOK(w, map[string]interface{}{
		"user_id":       user.ID,
		"email":         user.Email,
		"name":          user.Name,
		"tier":          user.Tier,
		"plan":          user.Plan,
		"slots":         TierSlots[user.Tier],
		"secrets_count": secretsCount,
		"api_hits":      apiHits,
		"sync_count":    syncCount,
	})
}

// ── GET /api/v1/portal/usage ────────────────────────────────────────────────

func handlePortalUsage(w http.ResponseWriter, r *http.Request) {
	claims := getPortalClaims(r)
	if claims == nil {
		sendError(w, "not authenticated", http.StatusUnauthorized)
		return
	}
	store := getUserStore()
	user := store.GetByID(claims.UserID)
	tier := "free"
	plan := "free"
	slots := 5
	if user != nil {
		tier = user.Tier
		plan = user.Plan
		slots = TierSlots[tier]
	}
	p := getPortalStore(claims.UserID)
	p.mu.RLock()
	stats := UsageStats{
		SecretsCount: p.Usage.SecretsCount,
		APIHits:      p.Usage.APIHits,
		SyncCount:    p.Usage.SyncCount,
		Tier:         tier,
		Plan:         plan,
		SlotsTotal:   slots,
	}
	p.mu.RUnlock()
	sendOK(w, stats)
}

// ── GET /api/v1/portal/api-keys ─────────────────────────────────────────────

func handleAPIKeysList(w http.ResponseWriter, r *http.Request) {
	claims := getPortalClaims(r)
	if claims == nil {
		sendError(w, "not authenticated", http.StatusUnauthorized)
		return
	}
	p := getPortalStore(claims.UserID)
	p.mu.RLock()
	keys := make([]APIKey, len(p.APIKeys))
	copy(keys, p.APIKeys)
	p.mu.RUnlock()
	sendOK(w, map[string]interface{}{"api_keys": keys})
}

// ── POST /api/v1/portal/api-keys ────────────────────────────────────────────

func handleAPIKeyCreate(w http.ResponseWriter, r *http.Request) {
	claims := getPortalClaims(r)
	if claims == nil {
		sendError(w, "not authenticated", http.StatusUnauthorized)
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		sendError(w, "name required", http.StatusBadRequest)
		return
	}
	// Generate a random API key
	keyBytes := make([]byte, 24)
	rand.Read(keyBytes)
	key := "ovav_" + base64.URLEncoding.EncodeToString(keyBytes)
	prefix := key[:16]

	h := sha256Hash([]byte(key))
	p := getPortalStore(claims.UserID)
	p.mu.Lock()
	ak := APIKey{
		ID:        fmt.Sprintf("ak_%s", randomID(8)),
		UserID:    claims.UserID,
		Name:      body.Name,
		KeyHash:   h,
		KeyPrefix: prefix,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}
	p.APIKeys = append(p.APIKeys, ak)
	p.save()
	p.mu.Unlock()
	sendOK(w, map[string]interface{}{
		"id":      ak.ID,
		"name":    ak.Name,
		"key":     key, // full key shown ONCE
		"created": ak.CreatedAt.Format(time.RFC3339),
	})
}

// ── DELETE /api/v1/portal/api-keys/:id ──────────────────────────────────────

func handleAPIKeyDelete(w http.ResponseWriter, r *http.Request) {
	claims := getPortalClaims(r)
	if claims == nil {
		sendError(w, "not authenticated", http.StatusUnauthorized)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/portal/api-keys/")
	p := getPortalStore(claims.UserID)
	p.mu.Lock()
	defer p.mu.Unlock()
	found := false
	for i, k := range p.APIKeys {
		if k.ID == id && k.UserID == claims.UserID {
			p.APIKeys = append(p.APIKeys[:i], p.APIKeys[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		sendError(w, "key not found", http.StatusNotFound)
		return
	}
	p.save()
	sendOK(w, map[string]interface{}{"deleted": true})
}
