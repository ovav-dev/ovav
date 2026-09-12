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
	if isTranslationLabel(path, line) {
		return secretFalsePos
	}
	fixture := isKnownFixture(line) || isLocalTestDefault(path, line)
	if fixture && isTestOrLocalContext(path, line) {
		if isDocumentationLine(line) {
			return secretFalsePos
		}
		return secretFixture
	}
	if isDynamicSecretExpression(line) {
		return secretFalsePos
	}
	if isDocumentationLine(line) && isKnownFixture(line) {
		return secretFalsePos
	}
	// Unknown static credentials and high-confidence token patterns remain A,
	// including in tests, comments, fixtures, and .env files.
	_ = label
	return secretReal
}

func isDynamicSecretExpression(line string) bool {
	lower := strings.ToLower(line)
	if strings.Contains(lower, ":'") && strings.Contains(lower, "password") {
		return true // PostgreSQL psql variable, e.g. :'runtime_password'
	}
	for _, marker := range []string{"${", "$env:", "os.environ", "os.getenv", "process.env", "read_env_value", "\\getenv"} {
		if strings.Contains(lower, marker) {
			// A default operator means a literal fallback is still present. It is
			// classified as B only when the fallback is an explicit fixture.
			if strings.Contains(lower, ":-") || strings.Contains(lower, ":=") {
				return !hasStaticDefault(lower)
			}
			return false
		}
	}
	if bareShellVariablePattern.MatchString(lower) {
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

func hasStaticDefault(line string) bool {
	for _, operator := range []string{":-", ":="} {
		if index := strings.Index(line, operator); index >= 0 {
			rest := strings.TrimSpace(line[index+len(operator):])
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
