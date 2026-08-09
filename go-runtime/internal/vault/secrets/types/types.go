// Package types defines the core OVAV Vault secret taxonomy.
// No dependencies on the vault or secrets packages — used by providers.
package types

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// SecretType represents the 7-type OVAV secret taxonomy.
type SecretType string

const (
	TypeAPIToken      SecretType = "api_token"
	TypeOAuthCreds    SecretType = "oauth_creds"
	TypeDBCredential  SecretType = "db_credential"
	TypeCloudKey      SecretType = "cloud_key"
	TypeEncryptionKey SecretType = "encryption_key"
	TypeUserSecret    SecretType = "user_secret"
	TypeTunnelToken   SecretType = "tunnel_token"
)

// AllTypes returns all valid SecretType values.
func AllTypes() []SecretType {
	return []SecretType{
		TypeAPIToken, TypeOAuthCreds, TypeDBCredential,
		TypeCloudKey, TypeEncryptionKey, TypeUserSecret, TypeTunnelToken,
	}
}

// IsValid returns true if t is a known secret type.
func (t SecretType) IsValid() bool {
	for _, v := range AllTypes() {
		if t == v {
			return true
		}
	}
	return false
}

// Secret is the core vault secret record.
type Secret struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Type      SecretType `json:"type"`
	Provider  string     `json:"provider"`
	Source    string     `json:"source"`
	Value     []byte     `json:"-"` // never serialized
	Hash      string     `json:"hash"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
	Rotatable bool       `json:"rotatable"`
	Tags      []string   `json:"tags,omitempty"`
	Metadata  Metadata   `json:"metadata,omitempty"`
}

// Metadata is provider-specific key-value data.
type Metadata map[string]string

// ComputeHash returns a SHA256 hash of the secret value for verification.
func ComputeHash(value []byte) string {
	h := sha256.Sum256(value)
	return hex.EncodeToString(h[:])
}

// InferType infers SecretType from a secret name string.
// Ordering matters: more specific patterns must come before broad ones.
func InferType(name string) SecretType {
	upper := strings.ToUpper(name)

	// TUNNEL must be before CLOUD (CLOUDFLARE_TUNNEL_TOKEN would match CLOUD first)
	if strings.Contains(upper, "TUNNEL_TOKEN") {
		return TypeTunnelToken
	}

	// OAuth (check before general API/TOKEN — OAUTH has specific structure)
	if strings.Contains(upper, "OAUTH") ||
		strings.Contains(upper, "CLIENT_SECRET") || strings.Contains(upper, "CLIENT_ID") {
		return TypeOAuthCreds
	}

	// DB credentials
	dbPatterns := []string{"DATABASE_URL", "DB_PASSWORD", "POSTGRES_PASSWORD",
		"MYSQL_PASSWORD", "REDIS_PASSWORD", "MONGODB_URI"}
	for _, p := range dbPatterns {
		if strings.Contains(upper, p) {
			return TypeDBCredential
		}
	}

	// Cloud keys (AWS, GCP, AZURE — but not CF which is TypeAPIToken)
	cloudPatterns := []string{"AWS_", "GCP_", "AZURE_", "GOOGLE_APPLICATION_CREDENTIALS"}
	for _, p := range cloudPatterns {
		if strings.Contains(upper, p) {
			return TypeCloudKey
		}
	}

	// Encryption keys
	encPatterns := []string{"HMAC", "JWT_SECRET", "ENCRYPTION_KEY",
		"SIGNING_KEY", "AUTH_TOKEN"}
	for _, p := range encPatterns {
		if strings.Contains(upper, p) {
			return TypeEncryptionKey
		}
	}

	// Cloudflare is an api_token (not cloud_key)
	if strings.Contains(upper, "CLOUDFLARE") || strings.Contains(upper, "CF_") {
		return TypeAPIToken
	}

	// Fly.io is an api_token
	if strings.Contains(upper, "FLY_") || strings.Contains(upper, "FLYIO") {
		return TypeAPIToken
	}

	// User secrets (Firebase, DNI — specific platforms) — before generic API/TOKEN
	if strings.Contains(upper, "FIREBASE") || strings.Contains(upper, "DNI") {
		return TypeUserSecret
	}

	// Generic API/TOKEN patterns (fallback)
	if strings.Contains(upper, "API_KEY") || strings.Contains(upper, "API_TOKEN") ||
		strings.Contains(upper, "_TOKEN") || strings.Contains(upper, "_SECRET") {
		return TypeAPIToken
	}

	return TypeAPIToken
}
