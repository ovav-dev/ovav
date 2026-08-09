package secrets

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSecret(t *testing.T) {
	value := []byte("my-api-token-12345")
	sec := NewSecret("CF Production Token", TypeAPIToken, "cloudflare", "manual", value)

	if sec.ID == "" {
		t.Error("NewSecret: ID should be generated")
	}
	if sec.Name != "CF Production Token" {
		t.Errorf("NewSecret: Name = %q, want %q", sec.Name, "CF Production Token")
	}
	if sec.Type != TypeAPIToken {
		t.Errorf("NewSecret: Type = %v, want %v", sec.Type, TypeAPIToken)
	}
	if sec.Provider != "cloudflare" {
		t.Errorf("NewSecret: Provider = %q, want %q", sec.Provider, "cloudflare")
	}
	if sec.Source != "manual" {
		t.Errorf("NewSecret: Source = %q, want %q", sec.Source, "manual")
	}
	if sec.Hash == "" {
		t.Error("NewSecret: Hash should be computed")
	}
	if sec.CreatedAt.IsZero() {
		t.Error("NewSecret: CreatedAt should be set")
	}
	if sec.Rotatable {
		t.Error("NewSecret: Rotatable should default to false")
	}
}

func TestSecretStore_CRUD(t *testing.T) {
	store := NewSecretStore()
	sec := NewSecret("Test Secret", TypeAPIToken, "test", "manual", []byte("value"))

	// Add
	err := store.Add(sec)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Get
	retrieved := store.Get(sec.ID)
	if retrieved == nil {
		t.Fatal("Get: returned nil")
	}
	if retrieved.Name != sec.Name {
		t.Errorf("Get: Name = %q, want %q", retrieved.Name, sec.Name)
	}

	// Count
	if store.Count() != 1 {
		t.Errorf("Count: = %d, want 1", store.Count())
	}

	// List
	list := store.List("")
	if len(list) != 1 {
		t.Errorf("List: len = %d, want 1", len(list))
	}
	list = store.List(TypeOAuthCreds)
	if len(list) != 0 {
		t.Errorf("List(filter=oauth): len = %d, want 0", len(list))
	}

	// Remove
	err = store.Remove(sec.ID)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if store.Count() != 0 {
		t.Errorf("After Remove: Count = %d, want 0", store.Count())
	}

	// Remove non-existent
	err = store.Remove(sec.ID)
	if err == nil {
		t.Error("Remove non-existent: expected error, got nil")
	}
}

func TestSecretStore_DuplicateID(t *testing.T) {
	store := NewSecretStore()
	sec := NewSecret("Test", TypeAPIToken, "test", "manual", []byte("value"))

	store.Add(sec)
	err := store.Add(sec)
	if err == nil {
		t.Error("Add duplicate: expected error, got nil")
	}
}

func TestInferType(t *testing.T) {
	cases := []struct {
		name string
		want SecretType
	}{
		{"OAUTH_GOOGLE_CLIENT_SECRET", TypeOAuthCreds},
		{"CLIENT_ID", TypeOAuthCreds},
		{"MYSQL_PASSWORD", TypeDBCredential},
		{"DATABASE_URL", TypeDBCredential},
		{"AWS_ACCESS_KEY_ID", TypeCloudKey},
		{"GCP_API_KEY", TypeCloudKey},
		{"JWT_SECRET", TypeEncryptionKey},
		{"HMAC_KEY", TypeEncryptionKey},
		{"CLOUDFLARE_TUNNEL_TOKEN", TypeTunnelToken},
		{"FIREBASE_API_KEY", TypeUserSecret},
		{"DNI_API_KEY", TypeUserSecret},
		{"CF_API_TOKEN", TypeAPIToken},
		{"CLOUDFLARE_API_TOKEN", TypeAPIToken},
		{"FLY_API_TOKEN", TypeAPIToken},
		{"RESEND_API_KEY", TypeAPIToken},
	}

	for _, c := range cases {
		got := InferType(c.name)
		if got != c.want {
			t.Errorf("InferType(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestStorePersistence(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("val1")))
	store.Add(NewSecret("S2", TypeOAuthCreds, "google", "github", []byte("val2")))

	// Serialize
	data, err := store.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	// Deserialize
	store2, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON: %v", err)
	}

	if store2.Count() != 2 {
		t.Errorf("After round-trip: Count = %d, want 2", store2.Count())
	}

	// Verify both secrets are preserved (deterministic check by name)
	s1 := store2.GetByName("S1")
	if s1 == nil {
		t.Fatal("S1 not found after round-trip")
	}
	if s1.Type != TypeAPIToken {
		t.Errorf("After round-trip: S1.Type = %v, want %v", s1.Type, TypeAPIToken)
	}

	s2 := store2.GetByName("S2")
	if s2 == nil {
		t.Fatal("S2 not found after round-trip")
	}
	if s2.Type != TypeOAuthCreds {
		t.Errorf("After round-trip: S2.Type = %v, want %v", s2.Type, TypeOAuthCreds)
	}
}

func (s *SecretStore) GetAllIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.secrets))
	for id := range s.secrets {
		ids = append(ids, id)
	}
	return ids
}

func TestSecretStore_UpdateUsage(t *testing.T) {
	store := NewSecretStore()
	sec := NewSecret("Test", TypeAPIToken, "test", "manual", []byte("value"))
	store.Add(sec)

	if sec.LastUsed != nil {
		t.Error("LastUsed should be nil after creation")
	}

	store.UpdateUsage(sec.ID)

	if sec.LastUsed == nil {
		t.Fatal("LastUsed should be set after UpdateUsage")
	}
	if sec.LastUsed.IsZero() {
		t.Error("LastUsed should not be zero")
	}
}

func TestSecretStore_FilePersistence(t *testing.T) {
	// Use a temp file
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "secrets.vault")

	key := make([]byte, 32) // 32-byte AES-256 key
	for i := range key {
		key[i] = byte(i)
	}

	store := NewSecretStore()
	store.Add(NewSecret("File Test", TypeCloudKey, "aws", "manual", []byte("aws-key-value")))

	// Save
	err := store.SaveToPath(path, key)
	if err != nil {
		t.Fatalf("SaveToPath: %v", err)
	}

	// Verify file exists and is not plaintext JSON
	data, _ := os.ReadFile(path)
	if bytes.Contains(data, []byte("File Test")) {
		t.Error("Vault file should not contain plaintext secret name")
	}

	// Load
	store2, err := LoadFromPath(path, key)
	if err != nil {
		t.Fatalf("LoadFromPath: %v", err)
	}

	if store2.Count() != 1 {
		t.Errorf("After file round-trip: Count = %d, want 1", store2.Count())
	}

	loaded := store2.List("")[0]
	if loaded.Name != "File Test" {
		t.Errorf("After file round-trip: Name = %q, want %q", loaded.Name, "File Test")
	}
	if loaded.Type != TypeCloudKey {
		t.Errorf("After file round-trip: Type = %v, want %v", loaded.Type, TypeCloudKey)
	}
}

func TestSecretStore_FilePersistence_WrongKey(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "secrets.vault")

	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 1

	store := NewSecretStore()
	store.Add(NewSecret("Wrong Key Test", TypeAPIToken, "test", "manual", []byte("value")))

	err := store.SaveToPath(path, key1)
	if err != nil {
		t.Fatalf("SaveToPath: %v", err)
	}

	_, err = LoadFromPath(path, key2)
	if err == nil {
		t.Error("LoadWithWrongKey: expected error, got nil")
	}
}

func TestSecretStore_List_FilterType(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("v1")))
	store.Add(NewSecret("S2", TypeAPIToken, "fly", "manual", []byte("v2")))
	store.Add(NewSecret("S3", TypeOAuthCreds, "google", "manual", []byte("v3")))

	list := store.List(TypeAPIToken)
	if len(list) != 2 {
		t.Errorf("List(api_token): len = %d, want 2", len(list))
	}

	list = store.List(TypeOAuthCreds)
	if len(list) != 1 {
		t.Errorf("List(oauth_creds): len = %d, want 1", len(list))
	}

	list = store.List("")
	if len(list) != 3 {
		t.Errorf("List(all): len = %d, want 3", len(list))
	}
}

func TestMetadata_JSON(t *testing.T) {
	sec := NewSecret("Meta Test", TypeAPIToken, "test", "manual", []byte("val"))
	sec.Metadata["region"] = "us-east-1"
	sec.Metadata["env"] = "production"

	data, err := json.Marshal(sec)
	if err != nil {
		t.Fatalf("Marshal secret with metadata: %v", err)
	}

	var sec2 Secret
	if err := json.Unmarshal(data, &sec2); err != nil {
		t.Fatalf("Unmarshal secret with metadata: %v", err)
	}

	if sec2.Metadata["region"] != "us-east-1" {
		t.Errorf("Metadata region = %q, want %q", sec2.Metadata["region"], "us-east-1")
	}
	if sec2.Metadata["env"] != "production" {
		t.Errorf("Metadata env = %q, want %q", sec2.Metadata["env"], "production")
	}
}

func TestComputeHash(t *testing.T) {
	h1 := ComputeHash([]byte("test-value"))
	h2 := ComputeHash([]byte("test-value"))
	h3 := ComputeHash([]byte("different"))

	if h1 != h2 {
		t.Error("ComputeHash: same input should produce same hash")
	}
	if h1 == h3 {
		t.Error("ComputeHash: different inputs should produce different hashes")
	}
	if len(h1) != 64 { // SHA256 hex = 64 chars
		t.Errorf("ComputeHash: len = %d, want 64", len(h1))
	}
}

func TestStoreFormat_StoredAt(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("Time Test", TypeAPIToken, "test", "manual", []byte("value")))

	data, _ := store.ToJSON()
	var format StoreFormat
	json.Unmarshal(data, &format)

	if format.Version != 1 {
		t.Errorf("Version = %d, want 1", format.Version)
	}
	if format.StoredAt.IsZero() {
		t.Error("StoredAt should be set")
	}
	if len(format.Secrets) != 1 {
		t.Errorf("Secrets len = %d, want 1", len(format.Secrets))
	}
}

func TestNewSecret_ExpiresAt(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	sec := NewSecret("Expiring", TypeAPIToken, "test", "manual", []byte("val"))
	sec.ExpiresAt = &future

	if sec.ExpiresAt == nil {
		t.Fatal("ExpiresAt should be set")
	}
	if sec.ExpiresAt.After(future.Add(time.Second)) {
		t.Error("ExpiresAt should be within 1 second of expected")
	}
}

// ── InferType edge cases ─────────────────────────────────────────────────────

func TestInferType_Empty(t *testing.T) {
	typ := InferType("")
	// Empty string falls through to default: TypeAPIToken
	if typ != TypeAPIToken {
		t.Errorf("InferType empty = %q, want %q", typ, TypeAPIToken)
	}
}

func TestInferType_NoMatch(t *testing.T) {
	typ := InferType("RANDOM_GIBBERISH")
	// No pattern matches → default: TypeAPIToken
	if typ != TypeAPIToken {
		t.Errorf("InferType no-match = %q, want %q", typ, TypeAPIToken)
	}
}

func TestInferType_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name string
		want SecretType
	}{
		// Generic API/TOKEN patterns (fallback)
		{"TOKEN", TypeAPIToken},
		{"api_key", TypeAPIToken},
		{"Api_Key", TypeAPIToken},
		{"API_TOKEN", TypeAPIToken},
		{"_API_SECRET", TypeAPIToken},
		// Cloud keys require underscore prefix
		{"AWS_", TypeCloudKey},
		{"GCP_", TypeCloudKey},
		{"AZURE_", TypeCloudKey},
		{"GOOGLE_APPLICATION_CREDENTIALS", TypeCloudKey},
		// Cloudflare/Fly are api_token, not cloud_key
		{"CLOUDFLARE_API_TOKEN", TypeAPIToken},
		{"CF_API_KEY", TypeAPIToken},
		{"FLY_API_TOKEN", TypeAPIToken},
		// User secrets
		{"FIREBASE", TypeUserSecret},
		{"DNI", TypeUserSecret},
		// OAuth
		{"OAUTH", TypeOAuthCreds},
		{"CLIENT_SECRET", TypeOAuthCreds},
		{"CLIENT_ID", TypeOAuthCreds},
		// DB
		{"DATABASE_URL", TypeDBCredential},
		{"DB_PASSWORD", TypeDBCredential},
		{"POSTGRES_PASSWORD", TypeDBCredential},
		{"MYSQL_PASSWORD", TypeDBCredential},
		{"REDIS_PASSWORD", TypeDBCredential},
		// Encryption
		{"HMAC_SECRET", TypeEncryptionKey},
		{"JWT_SECRET", TypeEncryptionKey},
		{"ENCRYPTION_KEY", TypeEncryptionKey},
		{"SIGNING_KEY", TypeEncryptionKey},
		{"AUTH_TOKEN", TypeEncryptionKey},
		// Tunnel
		{"TUNNEL_TOKEN", TypeTunnelToken},
		// No match → default
		{"PASSWORD", TypeAPIToken},
		{"SECRET", TypeAPIToken},
	}

	for _, tc := range tests {
		got := InferType(tc.name)
		if got != tc.want {
			t.Errorf("InferType(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// ── ComputeHash edge cases ─────────────────────────────────────────────────────

func TestComputeHash_Empty(t *testing.T) {
	hash := ComputeHash([]byte{})
	if hash == "" {
		t.Error("ComputeHash empty: got empty string")
	}
}

func TestComputeHash_LongInput(t *testing.T) {
	// 1MB input
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	hash := ComputeHash(data)
	if hash == "" {
		t.Error("ComputeHash long: got empty string")
	}
	if len(hash) != 64 { // SHA256 hex = 64 chars
		t.Errorf("ComputeHash length = %d, want 64", len(hash))
	}
}

func TestComputeHash_Deterministic(t *testing.T) {
	data := []byte("hello world")
	h1 := ComputeHash(data)
	h2 := ComputeHash(data)
	if h1 != h2 {
		t.Error("ComputeHash: same input should produce same hash")
	}
}

// ── GetAllIDs ─────────────────────────────────────────────────────────────────

func TestGetAllIDs_Empty(t *testing.T) {
	store := NewSecretStore()
	ids := store.GetAllIDs()
	if len(ids) != 0 {
		t.Errorf("GetAllIDs empty store: got %d, want 0", len(ids))
	}
}

func TestGetAllIDs_Multiple(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("v1")))
	store.Add(NewSecret("S2", TypeCloudKey, "aws", "manual", []byte("v2")))
	store.Add(NewSecret("S3", TypeOAuthCreds, "github", "oauth", []byte("v3")))

	ids := store.GetAllIDs()
	if len(ids) != 3 {
		t.Errorf("GetAllIDs: got %d, want 3", len(ids))
	}

	// All should be unique
	seen := make(map[string]bool)
	for _, id := range ids {
		if seen[id] {
			t.Errorf("Duplicate ID: %q", id)
		}
		seen[id] = true
	}
}

func TestGetAllIDs_AfterRemove(t *testing.T) {
	store := NewSecretStore()
	s1 := NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("v1"))
	s2 := NewSecret("S2", TypeCloudKey, "aws", "manual", []byte("v2"))
	store.Add(s1)
	store.Add(s2)

	ids := store.GetAllIDs()
	if len(ids) != 2 {
		t.Errorf("Before remove: GetAllIDs = %d, want 2", len(ids))
	}

	store.Remove(s1.ID)
	idsAfter := store.GetAllIDs()
	if len(idsAfter) != 1 {
		t.Errorf("After remove: GetAllIDs = %d, want 1", len(idsAfter))
	}
}
