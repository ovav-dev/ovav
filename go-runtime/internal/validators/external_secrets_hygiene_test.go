package validators

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyExternalFindingABC(t *testing.T) {
	tests := []struct {
		name, path, line, label string
		want                    secretClassification
	}{
		{"weak password error code", "backend/auth.go", `AuthErrWeakPassword = "weak_password"`, "Password in plaintext", secretFalsePos},
		{"runtime environment variable", "scripts/login.sh", `PASSWORD="$RUNTIME_PASSWORD"`, "Password in plaintext", secretFalsePos},
		{"required environment variable", "scripts/login.sh", `PASSWORD="${RUNTIME_PASSWORD:?required}"`, "Password in plaintext", secretFalsePos},
		{"generated shell password", "scripts/bootstrap-local.sh", `new_password="$(openssl rand -hex 12)"`, "Password in plaintext", secretFalsePos},
		{"local test default", "e2e/login.spec.ts", `password: "TuPasswordF12"`, "Password in plaintext", secretFixture},
		{"test secure fixture", "src/auth.test.ts", `password: "Secure1!"`, "Password in plaintext", secretFixture},
		{"test named fixture", "src/auth.test.ts", `password: "Test-password-123"`, "Password in plaintext", secretFixture},
		{"synthetic test fixture", "tests/auth.test.ts", `password: "SyntheticFixture-42"`, "Password in plaintext", secretFixture},
		{"local database default", "scripts/bootstrap-local.sh", `DATABASE_URL="postgresql://user:postgres@127.0.0.1:5432/db"`, "Database connection string with credentials", secretFixture},
		{"runtime psql variable", "db/init.sh", `ALTER ROLE app PASSWORD :'runtime_password';`, "Password in plaintext", secretFalsePos},
		{"documentation fixture", "README.md", `# password = "example-password"`, "Password in plaintext", secretFalsePos},
		{"unknown test password", "tests/auth.test.ts", `password: "correct-horse-battery-staple"`, "Password in plaintext", secretReal},
		{"unknown production password", "config.ts", `password = "synthetic-secret-material"`, "Password in plaintext", secretReal},
		{"non-local database credential", "config.ts", `DATABASE_URL="postgresql://app:synthetic-db-password@db.internal:5432/app"`, "Database connection string with credentials", secretReal},
		{"translation label", "src/i18n/en.ts", `loginPassword: "Password"`, "Password in plaintext", secretFalsePos},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyExternalFinding(tt.path, tt.line, tt.label); got != tt.want {
				t.Fatalf("classification = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExternalSecretsHygieneFixtureAndDynamicPass(t *testing.T) {
	root := t.TempDir()
	writeExternalSecretsTestFile(t, root, "e2e/login.spec.ts", `const user = { password: "TuPasswordF12" };`)
	writeExternalSecretsTestFile(t, root, "db/init.sh", `ALTER ROLE app PASSWORD :'runtime_password';`)

	result := NewExternalSecretsHygiene().Validate(context.Background(), root)
	if result.Status != "pass" {
		t.Fatalf("status = %s, want pass", result.Status)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("non-blocking findings became issues: %d", len(result.Issues))
	}
}

func TestExternalSecretsHygieneDoesNotIgnoreEnv(t *testing.T) {
	root := t.TempDir()
	writeExternalSecretsTestFile(t, root, ".env", `PASSWORD="synthetic-untracked-material"`)

	result := NewExternalSecretsHygiene().Validate(context.Background(), root)
	if result.Status != "fail" {
		t.Fatalf("status = %s, want fail for .env material", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("issue count = %d, want one redacted issue", len(result.Issues))
	}
}

func TestExternalSecretsHygieneFailClosedForTokensAndAPIKeys(t *testing.T) {
	tests := []struct {
		name, path, line, label string
	}{
		{"env file", ".env", `PASSWORD="synthetic-untracked-material"`, "Password in plaintext"},
		{"api key in test", "tests/auth.test.ts", `api_key = "synthetic-api-key-1234567890"`, "API key in plaintext"},
		{"token in fixture", "tests/fixtures/auth.ts", `ghp_0123456789abcdefghijklmnopqrstuvwxyzABCDEF`, "GitHub personal access token"},
		{"password outside test", "config.ts", `password = "not-clearly-a-fixture"`, "Password in plaintext"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyExternalFinding(tt.path, tt.line, tt.label); got != secretReal {
				t.Fatalf("classification = %q, want %q", got, secretReal)
			}
		})
	}
}

func TestExternalSecretsHygieneShellDefaultsAndDevelopmentTemplates(t *testing.T) {
	tests := []struct {
		name, path, line, label string
		want                    secretClassification
	}{
		{"nested required shell input", "scripts/audit-full.sh", `ADMIN_PASSWORD="${E2E_ADMIN_PASSWORD:-${OVAV_ADMIN_PASSWORD:-}}"`, "Password in plaintext", secretFalsePos},
		{"development fallback", "scripts/dev-scrapers.sh", `ADMIN_PASSWORD="${ADMIN_PASSWORD:-dev}"`, "Password in plaintext", secretFixture},
		{"development template DSN", ".env.example", `DATABASE_URL="postgresql://user:postgres@127.0.0.1:5432/db"`, "Database connection string with credentials", secretFixture},
		{"local runtime DSN remains dynamic", "scripts/local.sh", `DATABASE_URL="${AKRYNT_DATABASE_URL}"`, "Database URL in config", secretFalsePos},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyExternalFinding(tt.path, tt.line, tt.label); got != tt.want {
				t.Fatalf("classification = %q, want %q", got, tt.want)
			}
		})
	}
}

func writeExternalSecretsTestFile(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}
