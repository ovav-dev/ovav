package validators

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/alerts"
)

// SecretsHygiene scans the codebase for plaintext secrets.
// Checks for hardcoded API keys, tokens, passwords, and other credentials
// that should be in environment variables or the OVAV vault.
type SecretsHygiene struct{}

func NewSecretsHygiene() *SecretsHygiene { return &SecretsHygiene{} }

func (s *SecretsHygiene) ID() string   { return "secrets_hygiene" }
func (s *SecretsHygiene) Name() string { return "Secrets Hygiene" }
func (s *SecretsHygiene) Description() string {
	return "Scans codebase for plaintext secrets, tokens, and credentials"
}
func (s *SecretsHygiene) Weight() int { return 20 }

// secretPattern represents a regex pattern for detecting secrets.
type secretPattern struct {
	re    *regexp.Regexp
	label string
}

var secretPatterns = []secretPattern{
	{regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*["'][A-Za-z0-9_\-]{20,}["']`), "API key in plaintext"},
	{regexp.MustCompile(`(?i)(auth[_-]?token|bearer[_-]?token)\s*[:=]\s*["'][A-Za-z0-9_\-\.]{20,}["']`), "Auth token in plaintext"},
	{regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*["'][^"']{4,}["']`), "Password in plaintext"},
	{regexp.MustCompile(`(?i)(secret|private[_-]?key)\s*[:=]\s*["'][A-Za-z0-9_\-+/=]{20,}["']`), "Secret/private key in plaintext"},
	{regexp.MustCompile(`(?i)ghp_[A-Za-z0-9_]{36,}`), "GitHub personal access token"},
	{regexp.MustCompile(`(?i)gho_[A-Za-z0-9_]{36,}`), "GitHub OAuth token"},
	{regexp.MustCompile(`(?i)AKIA[0-9A-Z]{16}`), "AWS access key ID"},
	{regexp.MustCompile(`(?i)sk[-_]live[-_][0-9a-zA-Z]{24,}`), "Stripe live secret key"},
	{regexp.MustCompile(`(?i)rk[-_]live[-_][0-9a-zA-Z]{24,}`), "Stripe restricted key"},
	{regexp.MustCompile(`(?i)xox[bpras]-[0-9A-Za-z\-]{10,}`), "Slack bot token"},
	{regexp.MustCompile(`(?i)-----BEGIN\s+(RSA|EC|DSA|OPENSSH)\s+PRIVATE\s+KEY-----`), "Private key block"},
	{regexp.MustCompile(`(?i)(SECRET|TOKEN|PASSWORD|API_KEY)\s*=\s*["'][A-Za-z0-9_\-]{8,}["']`), "Env-style secret in config"},
	// ── N1 Refuerzo — 15 nuevos patrones (2026-06-19) ──
	{regexp.MustCompile(`(?i)eyJ[A-Za-z0-9\-_=]+\.[A-Za-z0-9\-_=]+\.?[A-Za-z0-9\-_.+/=]*`), "JWT token in plaintext"},
	{regexp.MustCompile(`(?i)sk-[A-Za-z0-9]{32,}`), "OpenAI/LLM API key"},
	{regexp.MustCompile(`(?i)(cf|cloudflare)[_-]?(api[_-]?(token|key)|token|key)\s*[:=]\s*["'][A-Za-z0-9_\-]{20,}["']`), "Cloudflare API credential"},
	{regexp.MustCompile(`(?i)(mongodb(\+srv)?|postgres(ql)?|mysql|redis)://[^@\s]+@[^/\s]+`), "Database connection string with credentials"},
	{regexp.MustCompile(`(?i)(DATABASE_URL|MONGO_URI|REDIS_URL|PG_URL)\s*=\s*["'][^"']{10,}["']`), "Database URL in config"},
	{regexp.MustCompile(`(?i)sk-ant-[A-Za-z0-9\-_]{32,}`), "Anthropic API key"},
	{regexp.MustCompile(`(?i)AIza[0-9A-Za-z\-_]{35}`), "Google API key"},
	{regexp.MustCompile(`(?i)glpat-[A-Za-z0-9\-_]{20,}`), "GitLab personal access token"},
	{regexp.MustCompile(`(?i)(NPM_TOKEN|GITHUB_TOKEN|DOCKER_PASSWORD)\s*=\s*["'][A-Za-z0-9_\-]{8,}["']`), "CI/CD token in config"},
	{regexp.MustCompile(`(?i)(private_key|privateKey)\s*\n?\s*[:=]\s*\n?\s*["']-----BEGIN`), "Private key in object notation"},
	{regexp.MustCompile(`(?i)(SECRET|TOKEN|KEY|PASSWORD|CREDENTIAL|AUTH)_(NAME|VALUE|DATA|STRING)?\s*=\s*["'][A-Za-z0-9_\-\.+/=]{8,}["']`), "Environment variable with secret-like value"},
	{regexp.MustCompile(`(?i)hf_[A-Za-z0-9]{32,}`), "HuggingFace API token"},
	{regexp.MustCompile(`(?i)(sendgrid|mailgun|twilio|slack)[_-]?(api[_-]?key|token)\s*[:=]\s*["'][A-Za-z0-9_\-]{20,}["']`), "Service API key"},
	{regexp.MustCompile(`(?i)(access[_-]?key[_-]?id|secret[_-]?access[_-]?key)\s*[:=]\s*["'][A-Za-z0-9/+=]{16,}["']`), "AWS-style credential pair"},
	{regexp.MustCompile(`(?i)firebase[_-]?(api[_-]?key|token|secret)\s*[:=]\s*["'][A-Za-z0-9_\-]{20,}["']`), "Firebase credential"},
}

// skipDirs are directories never scanned for secrets.
var skipDirs = map[string]bool{
	".git":              true,
	"node_modules":      true,
	"__pycache__":       true,
	".wrangler":         true,
	"dist":              true,
	"vendor":            true,
	".venv":             true, // Python virtual environment (third-party packages)
	"venv":              true, // alternative virtual env name
	".ovav/vault":       true, // encrypted vault — expected to have encrypted data
	"go-runtime/bin":    true, // compiled binaries
	"integrity_backups": true, // backup snapshots may contain test key material
	".github":           true, // CI workflow files use ${{ secrets.X }} legitimately
	"data":              true, // runtime data dirs (DNI caches, backups) — always gitignored
	".mimocode":         true, // MiMo Code runtime workspace
	".opencode":         true, // OpenCode runtime workspace
}

// skipFiles are specific files that are expected to contain secret-like patterns.
var skipFiles = map[string]bool{
	".gitleaks.toml":             true,
	"rego_engine.py":             true,
	"permission_authority.json":  true,
	"secrets_hygiene.go":         true, // this file contains patterns
	"secrets_hygiene_test.go":    true, // test fixtures
	"validators_test.go":         true, // test fixtures with mock secrets
	"check_ovav_ssh_profile.py":  true, // test SSH key fixtures
	"ovav_public_export_gate.py": true, // contains export test key fixture
	"minimax_direct_env.sh":      true, // placeholder API key template
	"provider_setup.sh":          true, // placeholder API key template
	"setup_minimax_direct.sh":    true, // placeholder API key template
}

// scanExts are file extensions scanned for secrets.
var scanExts = map[string]bool{
	".py":   true,
	".go":   true,
	".js":   true,
	".ts":   true,
	".tsx":  true,
	".jsx":  true,
	".yaml": true,
	".yml":  true,
	".json": true,
	".toml": true,
	".sh":   true,
	".bash": true,
	".env":  true,
	".cfg":  true,
	".ini":  true,
	".tf":   true,
}

func (s *SecretsHygiene) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string
	scannedFiles := 0
	scannedLines := 0

	// Consumer-mode detection: if .ovav/config.yaml has ovav_mode: consumer,
	// add extra skip patterns for known false positives (i18n, mock data)
	consumerMode := false
	configPath := filepath.Join(root, ".ovav", "config.yaml")
	if data, err := os.ReadFile(configPath); err == nil {
		if strings.Contains(string(data), "ovav_mode: consumer") || strings.Contains(string(data), "ovav_mode: \"consumer\"") {
			consumerMode = true
		}
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable paths
		}

		// Skip directories
		if d.IsDir() {
			name := d.Name()
			if skipDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}

		// Compute relative path for filtering
		rel, _ := filepath.Rel(root, path)

		// Skip specific files
		if skipFiles[filepath.Base(path)] {
			return nil
		}

		// Skip files without scannable extensions
		ext := strings.ToLower(filepath.Ext(path))
		if !scanExts[ext] {
			return nil
		}

		// Skip test fixtures and mock data
		if strings.Contains(rel, "testdata") || strings.Contains(rel, "fixtures") || strings.Contains(rel, "__pycache__") {
			return nil
		}
		// Skip test files (*_test.go, *_test.py, *.test.ts, etc.) and mock files
		if strings.HasSuffix(rel, "_test.go") || strings.HasSuffix(rel, "_test.py") ||
			strings.Contains(rel, "_test.") || strings.HasSuffix(rel, ".test.ts") ||
			strings.HasSuffix(rel, ".test.tsx") {
			return nil
		}

		// Consumer-mode: skip known false-positive directories
		if consumerMode {
			if strings.Contains(rel, "mock") || strings.Contains(rel, "demo") ||
				strings.Contains(rel, "fixtures") || strings.Contains(rel, "__tests__") ||
				strings.Contains(rel, "i18n") || strings.Contains(rel, "locales") {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Skip comment lines (less likely to have real secrets)
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
				continue
			}
			// Skip HTML/XML comments
			if strings.HasPrefix(trimmed, "<!--") {
				continue
			}

			for _, pat := range secretPatterns {
				if loc := pat.re.FindStringIndex(line); loc != nil {
					matched := line[loc[0]:loc[1]]
					// Redact the secret value for safe reporting
					redacted := pat.re.ReplaceAllString(matched, "[REDACTED]")
					issues = append(issues, fmt.Sprintf("%s:%d: %s — %s", rel, lineNum, pat.label, redacted))
					break // one issue per line is enough
				}
			}
		}
		scannedFiles++
		scannedLines += lineNum
		return nil
	})

	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		return Result{
			ID: s.ID(), Name: s.Name(), Status: "error", Weight: s.Weight(),
			Message:  fmt.Sprintf("ERROR walking file tree: %v", err),
			Duration: time.Since(start),
		}
	}

	if len(issues) == 0 {
		return Result{
			ID: s.ID(), Name: s.Name(), Status: "pass", Weight: s.Weight(),
			Message:  fmt.Sprintf("PASS secrets hygiene — %d file(s) / %d line(s) scanned, no secrets found", scannedFiles, scannedLines),
			Duration: time.Since(start),
		}
	}

	// ── Create persistent alerts for detected secrets ──
	alertMgr := alerts.NewManager(root)
	for _, issue := range issues {
		parts := strings.SplitN(issue, ": ", 3)
		fileName := ""
		lineNum := 0
		if len(parts) >= 1 {
			fileLine := strings.SplitN(parts[0], ":", 2)
			fileName = fileLine[0]
			if len(fileLine) > 1 {
				fmt.Sscanf(fileLine[1], "%d", &lineNum)
			}
		}
		title := "Plaintext secret detected"
		if len(parts) >= 2 {
			title = parts[1]
		}
		// Non-blocking: alert persistence failure must not crash the validator
		alertMgr.Create(alerts.CatSecrets, alerts.SevCritical, title, issue, fileName, lineNum)
	}

	return Result{
		ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
		Message: fmt.Sprintf("FAIL secrets hygiene — %d potential secret(s) detected in %d file(s)", len(issues), scannedFiles),
		Issues:  issues, Duration: time.Since(start),
	}
}

var _ Validator = (*SecretsHygiene)(nil)
