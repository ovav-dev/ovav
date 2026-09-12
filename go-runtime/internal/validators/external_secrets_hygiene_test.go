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
		{"real static credential", "config.ts", `password = "synthetic-secret-material"`, "Password in plaintext", secretReal},
		{"local test default", "e2e/login.spec.ts", `password: "TuPasswordF12"`, "Password in plaintext", secretFixture},
		{"local database default", "scripts/bootstrap-local.sh", `DATABASE_URL="postgresql://user:postgres@127.0.0.1:5432/db"`, "Database connection string with credentials", secretFixture},
		{"runtime psql variable", "db/init.sh", `ALTER ROLE app PASSWORD :'runtime_password';`, "Password in plaintext", secretFalsePos},
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
