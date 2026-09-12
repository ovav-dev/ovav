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
)

// Validate classifies every match, but only A findings are gate blocking.
// B and C never write alert state into the external repository.
func (s *ExternalSecretsHygiene) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	findings, scannedFiles, err := scanExternalSecretFindings(ctx, root)
	if err != nil {
		return Result{ID: s.ID(), Name: s.Name(), Status: "error", Weight: s.Weight(),
			Message: fmt.Sprintf("ERROR walking file tree: %v", err), Duration: time.Since(start)}
	}

	realCount, fixtureCount, falsePositiveCount := 0, 0, 0
	var issues []string
	for _, finding := range findings {
		switch finding.classification {
		case secretReal:
			realCount++
			issues = append(issues, formatExternalSecretFinding(finding))
		case secretFixture:
			fixtureCount++
		case secretFalsePos:
			falsePositiveCount++
		}
	}
	if realCount == 0 {
		return Result{ID: s.ID(), Name: s.Name(), Status: "pass", Weight: s.Weight(),
			Message: fmt.Sprintf("PASS external secrets hygiene — 0 real secret(s); %d fixture(s), %d contextual false positive(s); %d file(s) scanned", fixtureCount, falsePositiveCount, scannedFiles), Duration: time.Since(start)}
	}
	return Result{ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
		Message: fmt.Sprintf("FAIL external secrets hygiene — %d real secret(s) detected in %d file(s)", realCount, scannedFiles), Issues: issues, Duration: time.Since(start)}
}

func scanExternalSecretFindings(ctx context.Context, root string) ([]secretFinding, int, error) {
	var findings []secretFinding
	scannedFiles := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if externalSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		if !isExternalScannablePath(rel) {
			return nil
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
			for _, pat := range secretPatterns {
				if loc := pat.re.FindStringIndex(line); loc != nil {
					matched := line[loc[0]:loc[1]]
					findings = append(findings, secretFinding{
						path: rel, line: lineNum, pattern: pat, matched: matched,
						classification: classifyExternalFinding(rel, line, pat.label),
					})
					break
				}
			}
		}
		scannedFiles++
		return nil
	})
	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		return findings, scannedFiles, err
	}
	return findings, scannedFiles, nil
}

// Only generated/dependency trees are skipped for consumers. In particular,
// .github, data, backups, .env files, and credential-named files remain in the
// scan so real material cannot hide behind a broad repository allowlist.
var externalSkipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"__pycache__":  true,
	".wrangler":    true,
	"dist":         true,
	"vendor":       true,
	".venv":        true,
	"venv":         true,
	".mimocode":    true,
	".opencode":    true,
}

func isExternalScannablePath(path string) bool {
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".env") || strings.Contains(strings.ToLower(base), "credential") || strings.Contains(strings.ToLower(base), "secret") {
		return true
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pem", ".key", ".p12", ".pfx":
		return true
	default:
		return scanExts[strings.ToLower(filepath.Ext(path))]
	}
}

func formatExternalSecretFinding(finding secretFinding) string {
	redacted := finding.pattern.re.ReplaceAllString(finding.matched, "[REDACTED]")
	return fmt.Sprintf("%s:%d: %s [%s] — %s", finding.path, finding.line, finding.pattern.label, finding.classification, redacted)
}

func classifyExternalFinding(path, line, label string) secretClassification {
	// High-confidence credentials are never downgraded by surrounding context.
	// A test file or a comment can still contain a copied live token.
	if isHighConfidenceExternalSecret(label) || isExternalEnvFile(path) {
		return secretReal
	}
	if isTranslationLabel(path, line) {
		return secretFalsePos
	}
	if isSemanticErrorCode(line) {
		return secretFalsePos
	}
	if isDynamicSecretExpression(line) {
		return secretFalsePos
	}
	if isDevelopmentDSN(path, line) {
		return secretFixture
	}
	if isTestFixtureSecret(path, line) {
		return secretFixture
	}
	if isDocumentationFixture(line) {
		return secretFalsePos
	}
	// Unknown static credentials remain A, including in tests, comments,
	// fixture-looking directories, and environment files.
	return secretReal
}

func isHighConfidenceExternalSecret(label string) bool {
	lower := strings.ToLower(label)
	for _, marker := range []string{
		"api key", "auth token", "github", "aws", "stripe", "slack", "jwt",
		"openai", "cloudflare", "anthropic", "google", "gitlab", "ci/cd",
		"private key", "service api", "firebase",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func isExternalEnvFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return base == ".env"
}

var semanticErrorCodePattern = regexp.MustCompile(`(?i)\b[a-z_][a-z0-9_]*(?:err|error|status|reason|message)[a-z0-9_]*\s*(?:=|:)\s*["'][a-z][a-z0-9]*(?:_[a-z0-9]+)+["']`)
var weakPasswordCodePattern = regexp.MustCompile(`(?i)\b[a-z_][a-z0-9_]*code[a-z0-9_]*\s*(?:=|:)\s*["']weak_password["']`)

func isSemanticErrorCode(line string) bool {
	return semanticErrorCodePattern.MatchString(line) || weakPasswordCodePattern.MatchString(line)
}

func isDynamicSecretExpression(line string) bool {
	lower := strings.ToLower(line)
	if strings.Contains(lower, ":'") && strings.Contains(lower, "password") {
		return true // PostgreSQL psql variable, e.g. :'runtime_password'
	}
	for _, marker := range []string{"${", "$env:", "os.environ", "os.getenv", "process.env", "read_env_value", "\\getenv"} {
		if strings.Contains(lower, marker) {
			// A literal fallback is not dynamic by itself. It can become B only
			// through the narrow test/local fixture checks below.
			return !hasStaticDefault(lower)
		}
	}
	if commandSubstitutionPattern.MatchString(lower) || bareShellVariablePattern.MatchString(lower) {
		return true
	}
	if strings.Contains(lower, "quote(") || strings.Contains(lower, "f\"") || strings.Contains(lower, "f'") {
		if strings.Contains(lower, "quote(") && (strings.Contains(lower, "postgresql://") || strings.Contains(lower, "database_url")) {
			return true
		}
		return (strings.Contains(lower, "password") || strings.Contains(lower, "postgresql://") || strings.Contains(lower, "database_url")) && strings.Contains(lower, "{")
	}
	return false
}

var bareShellVariablePattern = regexp.MustCompile(`\$[a-z_][a-z0-9_]*`)
var commandSubstitutionPattern = regexp.MustCompile(`\$\([^)]*\)|` + "`" + `[^"]+` + "`")

func hasStaticDefault(line string) bool {
	for _, operator := range []string{":-", ":="} {
		if index := strings.Index(line, operator); index >= 0 {
			rest := strings.TrimSpace(line[index+len(operator):])
			rest = strings.TrimLeft(rest, "\"' })")
			return rest != "" && !strings.HasPrefix(rest, "$") && !strings.HasPrefix(rest, "{")
		}
	}
	return false
}

func isTranslationLabel(path, line string) bool {
	normalized := filepath.ToSlash(strings.ToLower(path))
	if !strings.Contains(normalized, "/i18n/") && !strings.Contains(normalized, "/locales/") {
		return false
	}
	return translationLabelPattern.MatchString(line)
}

var translationLabelPattern = regexp.MustCompile(`(?i)\b[a-z][a-z0-9]*(password|passwd|pwd)(placeholder|label|error|match|confirm)?\s*:`)

func isKnownFixture(line string) bool {
	lower := strings.ToLower(line)
	for _, marker := range []string{"tupasswordf12", "fixture-secret", "testpassword", "password123", "admin123", "changeme", "changeit", "dev-secret", "placeholder", "postgres"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

var externalPasswordAssignmentPattern = regexp.MustCompile(`(?i)\b(?:[a-z_][a-z0-9_]*)?password\b\s*[:=]\s*["']([^"']+)["']|\b(?:passwd|pwd)\b\s*[:=]\s*["']([^"']+)["']`)

func isTestFixtureSecret(path, line string) bool {
	if !isFixtureContext(path) || isDevelopmentDSN(path, line) {
		return false
	}
	if match := externalPasswordAssignmentPattern.FindStringSubmatch(line); len(match) > 1 {
		for _, value := range match[1:] {
			if value != "" {
				return isSyntheticFixtureValue(value)
			}
		}
	}
	return isKnownFixture(line)
}

func isSyntheticFixtureValue(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"test", "fixture", "synthetic", "dummy", "fake", "example", "sample", "password", "passwd", "secure", "changeme", "changeit", "dev", "local"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func isFixtureContext(path string) bool {
	if isTestPath(path) {
		return true
	}
	normalized := filepath.ToSlash(strings.ToLower(path))
	if filepath.Base(normalized) == ".env.example" {
		return true
	}
	for _, marker := range []string{"/local/", "-local", "local-", "/dev/", "-dev", "dev-", "cleanup", "bootstrap", "status"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func isTestPath(path string) bool {
	normalized := filepath.ToSlash(strings.ToLower(path))
	base := filepath.Base(normalized)
	if strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, "_test.py") ||
		strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, ".test.tsx") ||
		strings.HasSuffix(base, ".test.js") || strings.HasSuffix(base, ".test.jsx") ||
		strings.HasSuffix(base, ".spec.ts") || strings.HasSuffix(base, ".spec.tsx") ||
		strings.HasSuffix(base, ".spec.js") || strings.HasSuffix(base, ".spec.jsx") {
		return true
	}
	for _, segment := range strings.Split(normalized, "/") {
		if segment == "test" || segment == "tests" || segment == "testdata" || segment == "fixtures" || segment == "e2e" || segment == "__tests__" {
			return true
		}
	}
	return false
}

func isDevelopmentDSN(path, line string) bool {
	lower := strings.ToLower(line)
	if !strings.Contains(lower, "://") || (!strings.Contains(lower, "postgres") && !strings.Contains(lower, "mysql") && !strings.Contains(lower, "redis") && !strings.Contains(lower, "mongodb")) {
		return false
	}
	local := strings.Contains(lower, "localhost") || strings.Contains(lower, "127.0.0.1") || strings.Contains(lower, "::1")
	if !local || !isFixtureContext(path) {
		return false
	}
	return isKnownFixture(line) || strings.Contains(lower, "test") || strings.Contains(lower, "dev")
}

func isDocumentationFixture(line string) bool {
	if !isDocumentationLine(line) {
		return false
	}
	lower := strings.ToLower(line)
	for _, marker := range []string{"fixture", "example", "placeholder", "synthetic", "dummy", "sample"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func isLocalTestDefault(path, line string) bool {
	lower := strings.ToLower(line)
	if !strings.Contains(lower, ":-") && !strings.Contains(lower, ":=") {
		return false
	}
	if !hasStaticDefault(lower) || !strings.Contains(lower, "password") {
		return false
	}
	normalized := filepath.ToSlash(strings.ToLower(path))
	for _, marker := range []string{"/e2e/", ".spec.", ".test.", "test-", "-test", "dev-", "-dev", "playwright"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func isTestOrLocalContext(path, line string) bool {
	normalized := filepath.ToSlash(strings.ToLower(path))
	for _, marker := range []string{"/e2e/", ".spec.", ".test.", "test-", "-test", "dev-", "-dev", "local", "cleanup", "status"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	lower := strings.ToLower(line)
	return strings.Contains(lower, "localhost") || strings.Contains(lower, "127.0.0.1") || strings.Contains(lower, "e2e_") || strings.Contains(lower, "docker compose")
}

func isDocumentationLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "<!--") {
		return true
	}
	return strings.HasPrefix(trimmed, "echo ") || strings.HasPrefix(trimmed, "printf ")
}
