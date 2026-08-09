// OVAV cPanel — TOTP 2FA (GOV-010)
//
// Time-based One-Time Password using HMAC-SHA1.
// Stdlib only: crypto/hmac, crypto/sha1, encoding/base32.
//
// Flow:
//   POST /api/v1/auth/totp/setup   — Generate TOTP secret + QR data URL
//   POST /api/v1/auth/totp/verify  — Validate TOTP code, issue JWT
//
// Secret stored encrypted in .ovav/runtime/totp_secret.json
// Allowlist: only admin emails can set up TOTP.

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ── TOTP Algorithm (RFC 6238) ──────────────────────────────────────

const totpDigits = 6
const totpPeriod = 30

func generateTOTPSecret() (string, error) {
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes), nil
}

func computeTOTP(secret string, t time.Time) uint32 {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return 0
	}

	counter := uint64(t.Unix() / totpPeriod)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	code := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return code % 1000000
}

func validateTOTP(secret string, code string) bool {
	if len(code) != totpDigits {
		return false
	}

	now := time.Now().UTC()
	// Allow 1 step before/after for clock drift
	for _, step := range []int{-1, 0, 1} {
		t := now.Add(time.Duration(step) * totpPeriod * time.Second)
		expected := fmt.Sprintf("%06d", computeTOTP(secret, t))
		if expected == code {
			return true
		}
	}
	return false
}

// ── Secret Storage (AES-256-GCM encrypted) ──────────────────────────

type totpEntry struct {
	Email     string `json:"email"`
	Secret    string `json:"secret"`
	CreatedAt int64  `json:"created_at"`
	Verified  bool   `json:"verified"`
}

func totpStoragePath() string {
	return filepath.Join(RepoRoot, ".ovav", "runtime", "totp_secret.json")
}

func loadTOTPEntry() (*totpEntry, error) {
	path := totpStoragePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Decrypt using vault-derived key (HMAC of RepoRoot)
	key := deriveStorageKey()
	decrypted, err := aesDecrypt(data, key)
	if err != nil {
		return nil, err
	}

	var entry totpEntry
	if err := json.Unmarshal(decrypted, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func saveTOTPEntry(entry *totpEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	key := deriveStorageKey()
	encrypted, err := aesEncrypt(data, key)
	if err != nil {
		return err
	}

	path := totpStoragePath()
	os.MkdirAll(filepath.Dir(path), 0700)
	return os.WriteFile(path, encrypted, 0600)
}

func deriveStorageKey() []byte {
	h := sha256.Sum256([]byte(RepoRoot + ":ovav-totp-vault"))
	return h[:]
}

func aesEncrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func aesDecrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// ── OTP URL for QR code ────────────────────────────────────────────

func totpURL(email, secret string) string {
	issuer := "OVAV-cPanel"
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		issuer, email, secret, issuer, totpDigits, totpPeriod)
}

// ── Handlers ────────────────────────────────────────────────────────

// handleTOTPSetup generates a TOTP secret and returns QR code data.
// Only callable after successful Google OAuth (admin email verified).
// POST /api/v1/auth/totp/setup  { "email": "..." }
func handleTOTPSetup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	// Allowlist check
	if !isAdminEmail(req.Email) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "not authorized"})
		return
	}

	// Check if TOTP already configured
	if existing, _ := loadTOTPEntry(); existing != nil && existing.Verified {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "already_configured",
			"message": "TOTP already set up — use your existing authenticator app",
		})
		return
	}

	secret, err := generateTOTPSecret()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "secret generation failed"})
		return
	}

	entry := &totpEntry{
		Email:     req.Email,
		Secret:    secret,
		CreatedAt: time.Now().Unix(),
		Verified:  false,
	}
	if err := saveTOTPEntry(entry); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "save failed"})
		return
	}

	url := totpURL(req.Email, secret)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "setup_required",
		"secret":  secret,
		"otpauth": url,
		"message": "Scan this QR code with Google Authenticator",
	})
}

// handleTOTPVerify validates a TOTP code and issues JWT on success.
// POST /api/v1/auth/totp/verify  { "email": "...", "code": "123456" }
func handleTOTPVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if !isAdminEmail(req.Email) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "not authorized"})
		return
	}

	entry, err := loadTOTPEntry()
	if err != nil || entry == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "TOTP not configured — run setup first"})
		return
	}

	if !validateTOTP(entry.Secret, req.Code) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid code"})
		return
	}

	// Mark as verified on first successful validation
	if !entry.Verified {
		entry.Verified = true
		saveTOTPEntry(entry)
	}

	// Issue JWT
	if err := initJWT(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "JWT init failed"})
		return
	}

	claims := jwtClaims{
		Sub:  req.Email,
		Role: "admin",
		Iat:  time.Now().Unix(),
		Exp:  time.Now().Add(24 * time.Hour).Unix(),
	}
	token, err := signJWT(claims)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token generation failed"})
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "ovav_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "verified",
		"token":   token,
		"role":    "admin",
		"message": "TOTP verified — redirecting to dashboard",
	})
}

// isAdminEmail checks against the ADMIN_EMAILS allowlist.
func isAdminEmail(email string) bool {
	adminEmails := os.Getenv("ADMIN_EMAILS")
	if adminEmails == "" {
		return false
	}
	for _, admin := range strings.Split(adminEmails, ",") {
		if strings.TrimSpace(admin) == email {
			return true
		}
	}
	return false
}
