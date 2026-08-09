// OVAV cPanel — User account system.
//
// Passwords hashed with bcrypt (cost 12). Passwords NEVER stored in plaintext.
// Tier system: free (5 slots), premium (50 slots + revoke/rotate), enterprise (unlimited).

package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ── Constants ─────────────────────────────────────────────────────────────────

const (
	TierFree       = "free"
	TierPremium    = "premium"
	TierEnterprise = "enterprise"
)

// TierSlots defines secret count limits per subscription tier.
var TierSlots = map[string]int{
	TierFree:       5,
	TierPremium:    50,
	TierEnterprise: 999999,
}

// TierFeatures defines enabled features per tier.
var TierFeatures = map[string][]string{
	TierFree:       {"list", "get", "add", "remove"},
	TierPremium:    {"list", "get", "add", "remove", "revoke", "rotate", "sync"},
	TierEnterprise: {"list", "get", "add", "remove", "revoke", "rotate", "sync", "team", "audit"},
}

// ── User model ────────────────────────────────────────────────────────────────

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	Name         string    `json:"name"`
	Tier         string    `json:"tier"`
	Plan         string    `json:"plan"`
	Role         string    `json:"role"`           // "operator", "admin"
	TOTP         string    `json:"totp,omitempty"` // TOTP secret (base32)
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Active       bool      `json:"active"`
}

// ── User store ────────────────────────────────────────────────────────────────

type userStore struct {
	mu    sync.RWMutex
	users map[string]*User // key = lowercase email
	byID  map[string]*User // key = user ID
	path  string
}

var (
	usrStore     *userStore
	usrStoreOnce sync.Once
)

func getUserStore() *userStore {
	usrStoreOnce.Do(func() {
		store := &userStore{
			users: make(map[string]*User),
			byID:  make(map[string]*User),
			path:  usersDBPath(),
		}
		store.load() // ignore error — empty store on failure
		usrStore = store
	})
	return usrStore
}

func usersDBPath() string {
	return filepath.Join(RepoRoot, ".ovav", "vault", "users.json")
}

func (s *userStore) load() {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		return
	}
	var records []*User
	if err := json.Unmarshal(data, &records); err != nil {
		return
	}
	s.users = make(map[string]*User)
	s.byID = make(map[string]*User)
	for _, u := range records {
		key := strings.ToLower(u.Email)
		s.users[key] = u
		s.byID[u.ID] = u
	}
}

func (s *userStore) save() error {
	s.mu.RLock()
	records := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		records = append(records, u)
	}
	s.mu.RUnlock()

	os.MkdirAll(filepath.Dir(s.path), 0700)
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

// ── Password hashing ─────────────────────────────────────────────────────────

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// deriveVaultKey returns PBKDF2-HMAC-SHA256(password, salt, 100000, 32).
func deriveVaultKey(password, salt []byte) []byte {
	return pbkdf2(password, salt, 100000, 32)
}

func pbkdf2(password, salt []byte, iterations, keyLen int) []byte {
	hashLen := 32 // SHA256 output size
	result := make([]byte, keyLen)
	block := 1
	offset := 0

	for offset < keyLen {
		// U1 = HMAC-SHA256(password, salt || BE(block))
		be := []byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)}
		u := hmacSHA256(password, append(salt, be...))
		sum := make([]byte, len(u))
		copy(sum, u)

		// U2..Uc
		for j := 2; j <= iterations; j++ {
			uj := hmacSHA256(password, u)
			for k := 0; k < len(uj); k++ {
				sum[k] ^= uj[k]
			}
			copy(u, uj)
		}

		copy(result[offset:], sum)
		offset += hashLen
		block++
	}
	return result
}

func hmacSHA256(key, data []byte) []byte {
	// HMAC-SHA256: inneripad=0x36, outeripad=0x5c, 64-byte blocks
	ipad := make([]byte, 64)
	opad := make([]byte, 64)
	for i := 0; i < len(key) && i < 64; i++ {
		ipad[i] = key[i] ^ 0x36
		opad[i] = key[i] ^ 0x5c
	}
	for i := len(key); i < 64; i++ {
		ipad[i] = 0x36
		opad[i] = 0x5c
	}

	inner := sha256Hash(append(ipad, data...))
	innerB := inner[:] // [32]byte → []byte
	outer := sha256Hash(append(opad, innerB...))
	return outer[:] // [32]byte → []byte
}

func sha256Hash(data []byte) [32]byte {
	h := sha256.Sum256(data)
	return h
}

// generateUserID returns SHA256(email_lowercase)[:16] as base64url.
func generateUserID(email string) string {
	h := sha256Hash([]byte(strings.ToLower(email)))
	return base64.RawURLEncoding.EncodeToString(h[:16])
}

// encodeBase64URL returns base64url encoding of data (no padding).
func encodeBase64URL(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// generateRandomKey returns cryptographically random bytes as base64url string.
func generateRandomKey(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// ── User operations ──────────────────────────────────────────────────────────

func (s *userStore) Register(email, password, name string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !isValidEmail(email) {
		return nil, fmt.Errorf("invalid email")
	}
	if len(password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}
	if _, exists := s.users[email]; exists {
		return nil, fmt.Errorf("email already registered")
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("password hash failed: %w", err)
	}

	user := &User{
		ID:           generateUserID(email),
		Email:        email,
		PasswordHash: hash,
		Name:         strings.TrimSpace(name),
		Tier:         TierFree,
		Plan:         "free",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
		Active:       true,
	}

	s.users[email] = user
	s.byID[user.ID] = user

	if err := s.save(); err != nil {
		delete(s.users, email)
		delete(s.byID, user.ID)
		return nil, fmt.Errorf("save failed: %w", err)
	}
	return user, nil
}

func (s *userStore) Authenticate(email, password string) (*User, error) {
	s.mu.RLock()
	user, exists := s.users[strings.ToLower(strings.TrimSpace(email))]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("invalid email or password")
	}
	if !user.Active {
		return nil, fmt.Errorf("account suspended")
	}
	if !verifyPassword(password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid email or password")
	}
	return user, nil
}

func (s *userStore) GetByID(id string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byID[id]
}

func (s *userStore) GetByEmail(email string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.users[strings.ToLower(email)]
}

func (s *userStore) UpdateTier(userID, tier string) error {
	if _, ok := TierSlots[tier]; !ok {
		return fmt.Errorf("unknown tier: %s", tier)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, exists := s.byID[userID]
	if !exists {
		return fmt.Errorf("user not found")
	}
	user.Tier = tier
	user.UpdatedAt = time.Now().UTC()
	return s.save()
}

func (s *userStore) HasFeature(userID, feature string) bool {
	s.mu.RLock()
	user := s.byID[userID]
	s.mu.RUnlock()
	if user == nil {
		return false
	}
	for _, f := range TierFeatures[user.Tier] {
		if f == feature {
			return true
		}
	}
	return false
}

func (s *userStore) SlotLimit(userID string) int {
	s.mu.RLock()
	user := s.byID[userID]
	s.mu.RUnlock()
	if user == nil {
		return 0
	}
	return TierSlots[user.Tier]
}

// ── Utilities ────────────────────────────────────────────────────────────────

func isValidEmail(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	at := strings.Index(email, "@")
	if at < 1 || at > len(email)-3 {
		return false
	}
	dot := strings.LastIndex(email, ".")
	if dot < at+2 || dot > len(email)-2 {
		return false
	}
	return !strings.ContainsAny(email, " <>\"'")
}

// GetUserVaultKey returns the vault encryption key for a user.
// Key = PBKDF2(password, email_lower, 100000, 32)
func GetUserVaultKey(email, password string) []byte {
	return deriveVaultKey([]byte(password), []byte(strings.ToLower(email)))
}
