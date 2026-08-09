// OVAV cPanel — Vault Sync Server (Phase 6.2)
//
// Zero-knowledge vault sync relay. cPanel stores ONLY encrypted blobs.
// cPanel NEVER sees: seed, vault key, or any decrypted secret.
//
// Endpoints:
//   POST /api/v1/vault/auth       — Authenticate seed+machineID → JWT
//   GET  /api/v1/vault/blobs       — List all blobs for identity
//   POST /api/v1/vault/upload     — Upload encrypted vault blob
//   GET  /api/v1/vault/blob/:id   — Download a specific blob
//   DELETE /api/v1/vault/blob/:id — Delete a blob (device注销)
//
// Auth: JWT Bearer token (RS256, same as existing cPanel auth).
// Blob storage: ~/.ovav/vault/blobs/{identity_id}/{device_id}.blob
//
// This file is the SERVER counterpart to go-runtime/internal/vault/secrets/sync.go.

package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ── Seed registry (identity management) ───────────────────────────────────────

// vaultRegistry maps identityID → seed verification hash (SHA256(seed)).
// cPanel NEVER stores the raw seed. Only the hash, for auth verification.
type vaultRegistry struct {
	mu    sync.RWMutex
	seeds map[string]string // identityID → SHA256(seed)
}

var vaultReg = &vaultRegistry{seeds: make(map[string]string)}

// vaultRegistryPath returns the path to the vault registry file.
func vaultRegistryPath() string {
	return filepath.Join(RepoRoot, ".ovav", "vault", "registry.json")
}

// loadVaultRegistry loads the seed registry from disk.
func loadVaultRegistry() error {
	vaultReg.mu.Lock()
	defer vaultReg.mu.Unlock()

	path := vaultRegistryPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil // start with empty registry
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &vaultReg.seeds)
}

// saveVaultRegistry persists the seed registry to disk.
func (vr *vaultRegistry) save() error {
	vr.mu.RLock()
	defer vr.mu.RUnlock()

	path := vaultRegistryPath()
	os.MkdirAll(filepath.Dir(path), 0700)
	data, err := json.MarshalIndent(vr.seeds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// seedHash computes the verification hash for a seed.
// Uses SHA256(seed) — not PBKDF2, fast enough for server-side check.
func seedHash(seed string) string {
	h := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(h[:])
}

// getOrCreateIdentity returns existing identityID for this seed hash,
// or registers a new identity.
func (vr *vaultRegistry) getOrCreateIdentity(seedHash string) string {
	vr.mu.Lock()
	defer vr.mu.Unlock()

	for id, sh := range vr.seeds {
		if sh == seedHash {
			return id
		}
	}
	// New identity
	id := uuid.New().String()
	vr.seeds[id] = seedHash
	return id
}

// ── Blob storage ─────────────────────────────────────────────────────────────

// vaultBlobPath returns the path for a device's blob.
func vaultBlobPath(identityID, deviceID string) string {
	return filepath.Join(RepoRoot, ".ovav", "vault", "blobs", identityID, deviceID+".blob")
}

// vaultBlobsDir returns the directory containing all blobs for an identity.
func vaultBlobsDir(identityID string) string {
	return filepath.Join(RepoRoot, ".ovav", "vault", "blobs", identityID)
}

// blobInfo is the metadata stored alongside the encrypted blob.
type blobInfo struct {
	IdentityID string    `json:"identity_id"`
	DeviceID   string    `json:"device_id"`
	Version    int       `json:"version"`
	SyncedAt   time.Time `json:"synced_at"`
	BlobHash   string    `json:"blob_hash"` // SHA256 of the encrypted blob (dedup)
}

// saveBlob persists a blob and its metadata.
func saveBlob(identityID, deviceID string, blob []byte, blobHash string) error {
	dir := vaultBlobsDir(identityID)
	os.MkdirAll(dir, 0700)

	// Save blob data
	blobPath := vaultBlobPath(identityID, deviceID)
	if err := os.WriteFile(blobPath, blob, 0600); err != nil {
		return fmt.Errorf("write blob: %w", err)
	}

	// Save metadata
	metaPath := blobPath + ".meta"
	info := blobInfo{
		IdentityID: identityID,
		DeviceID:   deviceID,
		Version:    1,
		SyncedAt:   time.Now().UTC(),
		BlobHash:   blobHash,
	}
	metaData, _ := json.Marshal(info)
	return os.WriteFile(metaPath, metaData, 0600)
}

// listBlobs returns all blob metadata for an identity.
func listBlobs(identityID string) ([]blobInfo, error) {
	dir := vaultBlobsDir(identityID)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var blobs []blobInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".blob") {
			continue
		}
		metaPath := filepath.Join(dir, entry.Name()+".meta")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue // skip blobs without metadata
		}
		var info blobInfo
		if err := json.Unmarshal(data, &info); err != nil {
			continue
		}
		blobs = append(blobs, info)
	}
	return blobs, nil
}

// getBlob reads and returns a blob for a specific device.
func getBlob(identityID, deviceID string) ([]byte, error) {
	path := vaultBlobPath(identityID, deviceID)
	return os.ReadFile(path)
}

// deleteBlob removes a device's blob.
func deleteBlob(identityID, deviceID string) error {
	blobPath := vaultBlobPath(identityID, deviceID)
	os.Remove(blobPath)
	os.Remove(blobPath + ".meta")
	return nil
}

// ── JWT claims for vault ─────────────────────────────────────────────────────

type vaultJWTClaims struct {
	Sub       string `json:"sub"`  // identityID
	MachineID string `json:"mid"`  // machineID
	Role      string `json:"role"` // "vault-user"
	Iat       int64  `json:"iat"`
	Exp       int64  `json:"exp"`
}

// vaultAuthenticate is middleware that validates the JWT Bearer token
// and extracts the identityID and machineID.
func vaultAuthenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			sendError(w, "missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")

		claims, err := verifyVaultJWT(token)
		if err != nil {
			sendError(w, "invalid or expired token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Add claims to request context
		r = setVaultClaims(r, claims)
		next(w, r)
	}
}

type contextKey string

const vaultClaimsKey contextKey = "vaultClaims"

func setVaultClaims(r *http.Request, claims *vaultJWTClaims) *http.Request {
	ctx := context.WithValue(r.Context(), vaultClaimsKey, claims)
	return r.WithContext(ctx)
}

func getVaultClaims(r *http.Request) *vaultJWTClaims {
	claims, _ := r.Context().Value(vaultClaimsKey).(*vaultJWTClaims)
	return claims
}

// ── Vault JWT issuance ────────────────────────────────────────────────────────

var vaultJWTSessions = make(map[string]*vaultJWTClaims)
var vaultJWTLock sync.RWMutex

func issueVaultJWT(identityID, machineID string) (string, error) {
	now := time.Now()
	claims := &vaultJWTClaims{
		Sub:       identityID,
		MachineID: machineID,
		Role:      "vault-user",
		Iat:       now.Unix(),
		Exp:       now.Add(24 * time.Hour).Unix(),
	}

	if err := initJWT(); err != nil {
		return "", fmt.Errorf("init JWT: %w", err)
	}

	jwtKeyLock.RLock()
	privateKey := jwtPrivateKey
	jwtKeyLock.RUnlock()

	if privateKey == nil {
		return "", fmt.Errorf("JWT not initialized")
	}

	// Sign with RS256
	jwtHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))

	claimsJSON, _ := json.Marshal(claims)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	h := sha256.New()
	h.Write([]byte(jwtHeader + "." + claimsB64))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, h.Sum(nil))
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}
	sigB64 := base64.RawURLEncoding.EncodeToString(signature)

	token := jwtHeader + "." + claimsB64 + "." + sigB64

	// Store session
	vaultJWTLock.Lock()
	vaultJWTSessions[token] = claims
	vaultJWTLock.Unlock()

	return token, nil
}

func verifyVaultJWT(token string) (*vaultJWTClaims, error) {
	vaultJWTLock.RLock()
	claims, ok := vaultJWTSessions[token]
	vaultJWTLock.RUnlock()
	if !ok {
		return nil, fmt.Errorf("token not found")
	}

	now := time.Now().Unix()
	if claims.Exp < now {
		return nil, fmt.Errorf("token expired")
	}

	return claims, nil
}

// ── GET /health ───────────────────────────────────────────────────────────────

func handleVaultHealth(w http.ResponseWriter, r *http.Request) {
	issues := []string{}
	if err := loadVaultRegistry(); err != nil {
		issues = append(issues, "vault registry unavailable")
	}

	online := len(issues) == 0
	status := "ok"
	if len(issues) > 0 {
		status = "warning"
	}

	sendOK(w, map[string]interface{}{
		"service":    "ovav-vault-sync",
		"status":     status,
		"version":    Version,
		"online":     online,
		"blob_count": 0,
		"last_sync":  nil,
		"issues":     issues,
	})
}

// ── POST /api/v1/vault/auth ──────────────────────────────────────────────────
//
// Authenticate with seed + machineID + hostname.
// Creates or retrieves identity, returns JWT.
//
// Request:  { "seed": "...", "machine_id": "...", "hostname": "..." }
// Response: { "jwt": "...", "exp": "...", "identity_id": "...", "level": 1 }
func handleVaultAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limit auth attempts: 10/min per IP
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}
	if !checkVaultRateLimit(strings.TrimSpace(ip)) {
		sendError(w, "rate limit exceeded — try again in 1 minute", http.StatusTooManyRequests)
		return
	}

	var body struct {
		Seed      string `json:"seed"`
		MachineID string `json:"machine_id"`
		Hostname  string `json:"hostname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(body.Seed) < 16 {
		sendError(w, "seed too short", http.StatusBadRequest)
		return
	}
	if body.MachineID == "" {
		sendError(w, "machine_id required", http.StatusBadRequest)
		return
	}

	// Load registry and verify/create identity
	if err := loadVaultRegistry(); err != nil {
		sendError(w, "registry error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sh := seedHash(body.Seed)
	identityID := vaultReg.getOrCreateIdentity(sh)
	if err := vaultReg.save(); err != nil {
		sendError(w, "registry save failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Issue JWT
	jwt, err := issueVaultJWT(identityID, body.MachineID)
	if err != nil {
		sendError(w, "JWT issuance failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	exp := now.Add(24 * time.Hour)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jwt":         jwt,
		"exp":         exp.Format(time.RFC3339),
		"identity_id": identityID,
		"level":       1,
		"role":        "vault-user",
	})
}

// ── vault rate limiting ───────────────────────────────────────────────────────

var vaultAuthAttempts = make(map[string][]time.Time)
var vaultAuthMu sync.Mutex

func checkVaultRateLimit(ip string) bool {
	const maxAttempts = 10
	const window = time.Minute

	vaultAuthMu.Lock()
	defer vaultAuthMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	var recent []time.Time
	for _, t := range vaultAuthAttempts[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	vaultAuthAttempts[ip] = append(recent, now)
	return len(recent) <= maxAttempts
}

// ── GET /api/v1/vault/blobs ─────────────────────────────────────────────────
//
// Returns all blob metadata for the authenticated identity.
// Does NOT return the blob contents — only metadata.
func handleVaultBlobs(w http.ResponseWriter, r *http.Request) {
	claims := getVaultClaims(r)
	if claims == nil {
		sendError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	blobs, err := listBlobs(claims.Sub)
	if err != nil {
		sendError(w, "list blobs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sendOK(w, map[string]interface{}{
		"identity_id": claims.Sub,
		"blobs":       blobs,
		"server_ts":   time.Now().UTC(),
	})
}

// ── POST /api/v1/vault/upload ────────────────────────────────────────────────
//
// Upload an encrypted vault blob for this device.
// The blob is AES-256-GCM encrypted — cPanel cannot decrypt it.
func handleVaultUpload(w http.ResponseWriter, r *http.Request) {
	claims := getVaultClaims(r)
	if claims == nil {
		sendError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if r.ContentLength > 10*1024*1024 { // 10MB max
		sendError(w, "blob too large (max 10MB)", http.StatusRequestEntityTooLarge)
		return
	}

	blob, err := io.ReadAll(io.LimitReader(r.Body, 11*1024*1024))
	if err != nil {
		sendError(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Compute blob hash (SHA256) for deduplication
	blobHash := sha256Hex(string(blob))

	deviceID := claims.MachineID

	if err := saveBlob(claims.Sub, deviceID, blob, blobHash); err != nil {
		sendError(w, "save blob: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sendOK(w, map[string]interface{}{
		"identity_id": claims.Sub,
		"device_id":   deviceID,
		"blob_hash":   blobHash,
		"synced_at":   time.Now().UTC(),
	})
}

// ── GET /api/v1/vault/blob/:deviceID ─────────────────────────────────────────
//
// Download a specific device's blob by deviceID.
func handleVaultGetBlob(w http.ResponseWriter, r *http.Request) {
	claims := getVaultClaims(r)
	if claims == nil {
		sendError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	deviceID := strings.TrimPrefix(r.URL.Path, "/api/v1/vault/blob/")
	deviceID = strings.TrimSuffix(deviceID, "/")

	blob, err := getBlob(claims.Sub, deviceID)
	if os.IsNotExist(err) {
		sendError(w, "blob not found", http.StatusNotFound)
		return
	}
	if err != nil {
		sendError(w, "read blob: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="vault-%s.blob"`, deviceID))
	w.Write(blob)
}

// ── DELETE /api/v1/vault/blob/:deviceID ─────────────────────────────────────
//
// Delete a device's blob (device deregistration).
func handleVaultDeleteBlob(w http.ResponseWriter, r *http.Request) {
	claims := getVaultClaims(r)
	if claims == nil {
		sendError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	deviceID := strings.TrimPrefix(r.URL.Path, "/api/v1/vault/blob/")
	deviceID = strings.TrimSuffix(deviceID, "/")

	if err := deleteBlob(claims.Sub, deviceID); err != nil {
		sendError(w, "delete blob: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sendOK(w, map[string]string{
		"deleted": deviceID,
	})
}

// ── helpers ──────────────────────────────────────────────────────────────────
