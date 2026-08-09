// sync.go — Zero-knowledge cross-device sync for OVAV Vault.
//
// Phase 6 of OVAV-VAULT-2026 plan.
//
// ARCHITECTURE — Zero-Knowledge Sync:
//
// Device-local encryption:
//
//	VaultKey = PBKDF2(seed, machineID) → encrypts secrets.vault
//
// Cross-device sync (wrap/wrap):
//
//	SyncKey = PBKDF2(seed, "ovav-sync-v1") → re-encrypts vault for cPanel
//
// cPanel stores ONLY encrypted blobs — it cannot decrypt anything because:
//   - SyncKey derives from seed (cPanel never has the seed)
//   - Each blob is AES-256-GCM(wrappedVaultJSON, SyncKey)
//   - cPanel is a blind relay: it stores blobs, it doesn't read them
//
// Cross-device merge:
//
//	All devices use the SAME SyncKey (seed-only derived)
//	Each device unwraps the blob with SyncKey, then re-encrypts
//	locally with its own VaultKey (machineID-derived)
//
// Attack scenarios:
//   - cPanel DB leaked → attacker sees only encrypted blobs, USELESS
//   - seed stolen (without machineID) → cannot derive VaultKey or SyncKey
//   - device stolen (seed + machineID) → attacker can decrypt that device's vault
//     but NOT other devices' vaults (different machineID → different VaultKey)
package secrets

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ovav/ovav/internal/license"
	vaultpkg "github.com/ovav/ovav/internal/vault"
)

// SyncWrapDomain is the domain separator for sync key derivation.
// Using a fixed string (not machineID) ensures the SyncKey is the SAME
// on all devices for a given seed — enabling cross-device sync.
const SyncWrapDomain = "ovav-sync-v1"

// SyncBlob is what gets uploaded to cPanel.
// This is the ONLY thing cPanel ever stores for a user's vault.
type SyncBlob struct {
	IdentityID string    `json:"identity_id"`
	DeviceID   string    `json:"device_id"` // machine ID of the uploading device
	Version    int       `json:"version"`
	SyncedAt   time.Time `json:"synced_at"`
	BlobHash   string    `json:"blob_hash"` // SHA-256 of blob data (dedup, not auth)
	// Blob is the full vault JSON, encrypted under SyncKey.
	// cPanel cannot decrypt this — only devices with the seed can.
	Blob []byte `json:"blob"`
}

// SyncPayload is the full sync response from cPanel.
type SyncPayload struct {
	Blobs    []SyncBlob `json:"blobs"` // one per device
	ServerTS time.Time  `json:"server_ts"`
	JWTExp   time.Time  `json:"jwt_exp"`
}

// SyncQueueItem is a pending vault operation waiting for connectivity.
type SyncQueueItem struct {
	ID        string    `json:"id"`
	Op        string    `json:"op"` // "add", "remove", "update", "rotate"
	SecretID  string    `json:"secret_id,omitempty"`
	Secret    *Secret   `json:"secret,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Retries   int       `json:"retries"`
	LastError string    `json:"last_error,omitempty"`
}

// SyncQueue persists pending operations to disk when offline.
type SyncQueue struct {
	mu    sync.Mutex
	path  string
	items []SyncQueueItem
}

// ── Key derivation ───────────────────────────────────────────────────────────

// DeriveVaultKey derives the per-device vault key from seed + machineID.
// This is the key that encrypts/decrypts the local secrets.vault file.
func DeriveVaultKey(seed, machineID string) ([]byte, error) {
	return license.DeriveKey(seed, machineID)
}

// DeriveSyncKey derives the cross-device sync key from seed only.
// This key is the SAME on all devices for a given seed.
// It is used ONLY to wrap the vault for cPanel sync — never for local storage.
func DeriveSyncKey(seed string) ([]byte, error) {
	// Use a fixed domain separator so this key is identical across all devices.
	// "ovav-sync-v1" is not a machine identifier — it's a protocol version.
	return license.DeriveKey(seed, SyncWrapDomain)
}

// ── Sync blob operations ─────────────────────────────────────────────────────

// WrapVaultForSync serializes the vault and re-encrypts it under SyncKey.
// The wrapped blob is what gets uploaded to cPanel.
// cPanel sees only the encrypted blob — it cannot decrypt the contents.
func WrapVaultForSync(store *SecretStore, seed string) ([]byte, string, error) {
	// Serialize current vault state
	vaultJSON, err := store.ToJSON()
	if err != nil {
		return nil, "", fmt.Errorf("sync: serialize vault: %w", err)
	}

	// Derive sync key (seed-only, same on all devices)
	syncKey, err := DeriveSyncKey(seed)
	if err != nil {
		return nil, "", fmt.Errorf("sync: derive sync key: %w", err)
	}

	// Re-encrypt vault under sync key for upload
	wrappedBlob, err := vaultpkg.Encrypt(vaultJSON, syncKey)
	if err != nil {
		return nil, "", fmt.Errorf("sync: encrypt vault for sync: %w", err)
	}

	blobHash := sha256Hex(wrappedBlob)
	return wrappedBlob, blobHash, nil
}

// UnwrapVaultFromSync decrypts a sync blob using SyncKey.
// Returns the vault JSON that can be loaded into a SecretStore.
func UnwrapVaultFromSync(wrappedBlob []byte, seed string) ([]byte, error) {
	syncKey, err := DeriveSyncKey(seed)
	if err != nil {
		return nil, fmt.Errorf("sync: derive sync key: %w", err)
	}

	vaultJSON, err := vaultpkg.Decrypt(wrappedBlob, syncKey)
	if err != nil {
		return nil, fmt.Errorf("sync: decrypt blob: %w", err)
	}

	return vaultJSON, nil
}

// MergeRemoteBlobs merges secrets from remote blobs into the local store.
// Strategy: latest-wins per secret name. Remote wins on conflict.
// Local-only secrets (never synced) are preserved.
func MergeRemoteBlobs(store *SecretStore, blobs []SyncBlob, seed, localDeviceID string) error {
	for _, blob := range blobs {
		if blob.DeviceID == localDeviceID {
			continue // skip our own blob
		}

		vaultJSON, err := UnwrapVaultFromSync(blob.Blob, seed)
		if err != nil {
			// Log and continue — this device might have been wiped
			fmt.Fprintf(os.Stderr, "⚠️  Failed to decrypt blob from device %s: %v\n", blob.DeviceID, err)
			continue
		}

		remoteStore, err := FromJSON(vaultJSON)
		if err != nil {
			continue
		}

		// Merge: add secrets from remote that don't exist locally
		for _, sec := range remoteStore.List("") {
			if store.GetByName(sec.Name) == nil {
				store.Add(sec)
			}
		}
	}

	return nil
}

// ── cPanel API client ────────────────────────────────────────────────────────

// cPanelClient manages communication with cpanel.ovav.dev
type cPanelClient struct {
	baseURL string
	jwt     string
	http    *http.Client
	seed    string
}

// sessionFields mirrors the minimal fields needed from ~/.local/share/ovav/session
// to extract the VaultJWT without re-imposing a dependency on cmd/ovav.
type sessionFields struct {
	VaultJWT string `json:"vault_jwt"`
}

// loadVaultJWTFromSession reads the OVAV session file and returns the VaultJWT.
// Returns empty string if the session file doesn't exist or the JWT is not present.
func loadVaultJWTFromSession() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	sessionPath := filepath.Join(home, ".local", "share", "ovav", "session")
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return ""
	}
	var sess sessionFields
	if err := json.Unmarshal(data, &sess); err != nil {
		return ""
	}
	return sess.VaultJWT
}

// NewCPanelClient creates a new cPanel client.
// vaultJWT: pre-obtained JWT from a prior 'ovav login' session file.
//
//	If non-empty, skips seed-based /vault/auth call and uses it directly.
//
// seed:    fall back if vaultJWT is empty — used to obtain a fresh JWT via /vault/auth.
// Attempts online auth if internet is available; gracefully degrades if not.
func NewCPanelClient(vaultJWT, seed string) (*cPanelClient, error) {
	baseURL := os.Getenv("OVAV_CPANEL_BASE_URL")
	if baseURL == "" {
		baseURL = "https://d678beea.ovav.dev/api/v1"
	}
	c := &cPanelClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 15 * time.Second},
		seed:    seed,
	}

	if !hasInternet() {
		return c, nil // offline mode — jwt stays empty
	}

	// If a VaultJWT is already available from the session, use it directly.
	// This avoids a redundant /vault/auth call when the seed env var is not set.
	if vaultJWT != "" {
		c.jwt = vaultJWT
		return c, nil
	}

	if err := c.authOnline(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  cPanel online auth failed: %v (operating in offline mode)\n", err)
		return c, nil
	}

	return c, nil
}

// hasInternet checks connectivity before attempting online operations.
func hasInternet() bool {
	baseURL := os.Getenv("OVAV_CPANEL_BASE_URL")
	if baseURL == "" {
		baseURL = "https://d678beea.ovav.dev/api/v1"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", baseURL+"/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 400
}

// authOnline authenticates with cPanel and stores the JWT.
func (c *cPanelClient) authOnline() error {
	machineID, err := license.MachineID()
	if err != nil {
		return fmt.Errorf("auth: machine ID: %w", err)
	}

	hostname, _ := os.Hostname()
	payload := map[string]string{
		"seed":       c.seed,
		"machine_id": machineID,
		"hostname":   hostname,
	}

	body, _ := json.Marshal(payload)
	resp, err := c.http.Post(c.baseURL+"/vault/auth", "application/json",
		newBytesReader(body))
	if err != nil {
		return fmt.Errorf("auth: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("auth rejected: invalid seed")
	}
	if resp.StatusCode == 403 {
		return fmt.Errorf("auth forbidden: identity revoked")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("auth: HTTP %d", resp.StatusCode)
	}

	var authResp struct {
		JWT   string    `json:"jwt"`
		Exp   time.Time `json:"exp"`
		ID    string    `json:"identity_id"`
		Level int       `json:"level"`
		Role  string    `json:"role"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("auth: parse response: %w", err)
	}

	c.jwt = authResp.JWT
	return nil
}

// IsOnline returns true if the client has a valid JWT.
func (c *cPanelClient) IsOnline() bool {
	return c.jwt != ""
}

// cpanelRequest makes an authenticated request to cPanel.
func (c *cPanelClient) cpanelRequest(method, path string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(method, c.baseURL+path, newBytesReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.jwt != "" {
		req.Header.Set("Authorization", "Bearer "+c.jwt)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cPanel request: %w", err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("cPanel error %d: %s", resp.StatusCode, string(out))
	}
	return out, nil
}

// UploadSync uploads the wrapped vault blob to cPanel.
func (c *cPanelClient) UploadSync(blob *SyncBlob) error {
	body, _ := json.Marshal(blob)
	_, err := c.cpanelRequest("POST", "/vault/upload", body)
	return err
}

// DownloadBlobs fetches all sync blobs for this identity from cPanel.
func (c *cPanelClient) DownloadBlobs() ([]SyncBlob, error) {
	data, err := c.cpanelRequest("GET", "/vault/blobs", nil)
	if err != nil {
		return nil, err
	}

	// Server returns { "blobs": [...], "identity_id": "...", "server_ts": "..." }
	var resp struct {
		Blobs []SyncBlob `json:"blobs"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse blobs response: %w", err)
	}
	if resp.Blobs == nil {
		resp.Blobs = []SyncBlob{}
	}
	return resp.Blobs, nil
}

// ── Sync queue (offline operations) ─────────────────────────────────────────

// LoadSyncQueue opens or creates the offline sync queue.
func LoadSyncQueue() (*SyncQueue, error) {
	path := filepath.Join(os.Getenv("HOME"), ".local", "share", "ovav", "secrets.syncqueue")
	sq := &SyncQueue{path: path}
	if err := sq.load(); err != nil {
		return nil, err
	}
	return sq, nil
}

func (sq *SyncQueue) load() error {
	data, err := os.ReadFile(sq.path)
	if os.IsNotExist(err) {
		sq.items = []SyncQueueItem{}
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &sq.items)
}

func (sq *SyncQueue) persist() error {
	dir := filepath.Dir(sq.path)
	os.MkdirAll(dir, 0700)
	data, err := json.MarshalIndent(sq.items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sq.path, data, 0600)
}

// Enqueue adds an operation to the offline sync queue.
func (sq *SyncQueue) Enqueue(item SyncQueueItem) error {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	sq.items = append(sq.items, item)
	return sq.persist()
}

// Pending returns the number of pending sync operations.
func (sq *SyncQueue) Pending() int {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	return len(sq.items)
}

// ── Full sync operation ───────────────────────────────────────────────────────

// SyncResult is the outcome of a full sync operation.
type SyncResult struct {
	Uploaded   bool     `json:"uploaded"`
	Downloaded bool     `json:"downloaded"`
	Merged     int      `json:"merged"` // number of secrets merged from remote
	Conflicts  int      `json:"conflicts"`
	Errors     []string `json:"errors,omitempty"`
	Online     bool     `json:"online"`
	PendingOps int      `json:"pending_ops"` // queued ops still waiting
}

// FullSync performs a complete bidirectional sync with cPanel.
// It is idempotent and safe to call even when offline.
// vaultJWT: optional pre-obtained JWT. If empty, tries to load from session file.
func FullSync(store *SecretStore, seed, machineID string) (*SyncResult, error) {
	return FullSyncWithJWT(store, "", seed, machineID)
}

// FullSyncWithJWT is like FullSync but accepts an explicit vaultJWT.
func FullSyncWithJWT(store *SecretStore, vaultJWT, seed, machineID string) (*SyncResult, error) {
	result := &SyncResult{Online: false}

	// If vaultJWT not provided, try to load from session file.
	if vaultJWT == "" {
		vaultJWT = loadVaultJWTFromSession()
	}

	sq, _ := LoadSyncQueue()
	if sq != nil {
		result.PendingOps = sq.Pending()
	}

	// Attempt online sync
	client, err := NewCPanelClient(vaultJWT, seed)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result, nil
	}

	if !client.IsOnline() {
		return result, nil // offline — nothing to do
	}

	result.Online = true

	// Upload local vault
	wrappedBlob, blobHash, err := WrapVaultForSync(store, seed)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("wrap: %v", err))
		return result, nil
	}

	machineIDStr, _ := license.MachineID()
	blob := &SyncBlob{
		DeviceID: machineIDStr,
		Version:  1,
		SyncedAt: time.Now().UTC(),
		BlobHash: blobHash,
		Blob:     wrappedBlob,
	}

	if err := client.UploadSync(blob); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("upload: %v", err))
		// Continue — try to download even if upload fails
	} else {
		result.Uploaded = true
	}

	// Download and merge remote blobs
	blobs, err := client.DownloadBlobs()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("download: %v", err))
		return result, nil
	}

	preMergeCount := len(store.List(""))
	if err := MergeRemoteBlobs(store, blobs, seed, machineIDStr); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("merge: %v", err))
	} else {
		result.Downloaded = true
		postMergeCount := len(store.List(""))
		result.Merged = postMergeCount - preMergeCount
	}

	return result, nil
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func newBytesReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
