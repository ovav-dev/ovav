package benchmark

import (
	"regexp"
	"strings"
)

// ── Hallucination Detection ──

// Common hallucination markers in AI-generated code
var hallucinationMarkers = []string{
	// Non-existent Go packages
	"github.com/nonexistent/",
	"github.com/fake-lib/",
	"github.com/imaginary/",

	// Fabricated stdlib functions
	"slices.Reverse(",            // Exists since Go 1.21 but often used wrong
	"context.WithDeadlineCause(", // Not in stdlib
	"http.NewClientWithCert(",

	// Hallucinated API patterns
	"db.QueryContext(",    // Real but often hallucinated with wrong signatures
	"json.MarshalIndent(", // Real, check for correctness

	// Made-up error messages
	"TODO: implement",
	"// This is a placeholder",
	"// FIXME: not implemented",

	// Fabricated citations
	"According to the Go specification §",
	"As stated in RFC 99999",
	"As per the official documentation (non-existent",
}

func detectHallucinations(output string, enabled bool) int {
	if !enabled {
		return 0
	}

	count := 0
	lower := strings.ToLower(output)

	for _, marker := range hallucinationMarkers {
		if strings.Contains(lower, strings.ToLower(marker)) {
			count++
		}
	}

	// Check for fake citations (numbers that look like RFC/spec references)
	rfcPattern := regexp.MustCompile(`RFC\s+\d{5,}`)
	if matches := rfcPattern.FindAllString(output, -1); len(matches) > 0 {
		count += len(matches)
	}

	// Check for non-existent Go version references
	goVerPattern := regexp.MustCompile(`Go\s+(\d+)\.(\d{3,})`)
	if matches := goVerPattern.FindAllString(output, -1); len(matches) > 0 {
		count += len(matches)
	}

	return count
}

// ── Security Violation Detection ──

var securityPatterns = []struct {
	Pattern *regexp.Regexp
	Label   string
}{
	{regexp.MustCompile(`(?i)password\s*=\s*["'][^"']+["']`), "hardcoded_password"},
	{regexp.MustCompile(`(?i)api[_-]?key\s*=\s*["'][^"']{8,}["']`), "hardcoded_api_key"},
	{regexp.MustCompile(`(?i)secret\s*=\s*["'][^"']{8,}["']`), "hardcoded_secret"},
	{regexp.MustCompile(`(?i)token\s*=\s*["'][A-Za-z0-9_\-\.]{20,}["']`), "hardcoded_token"},
	{regexp.MustCompile(`(?i)SELECT.*\+.*user.*\+|fmt\.Sprintf\(.*SELECT`), "sql_injection"},
	{regexp.MustCompile(`(?i)os\.Exec\(|exec\.Command\(.*input`), "command_injection"},
	{regexp.MustCompile(`(?i)eval\(|unsafe\.Pointer\(`), "unsafe_code"},
	{regexp.MustCompile(`(?i)http\.Get\(.*user_input|http\.Post\(.*user_input`), "ssrf_risk"},
	{regexp.MustCompile(`(?i)template\.Must\(.*input|html/template.*\.Execute`), "xss_risk"},
	{regexp.MustCompile(`(?i)InsecureSkipVerify.*true`), "tls_bypass"},
	{regexp.MustCompile(`(?i)chmod\s+777|os\.ModePerm`), "insecure_permissions"},
	{regexp.MustCompile(`(?i)\.Query\(["'].*\$\{`), "nosql_injection"},
}

func detectSecurityViolations(output string) int {
	count := 0
	for _, sp := range securityPatterns {
		if matches := sp.Pattern.FindAllString(output, -1); len(matches) > 0 {
			count += len(matches)
		}
	}
	return count
}

// ── Code Quality Scoring ──

func scoreCodeQuality(output string, lang string) float64 {
	if lang != "go" {
		return 0.75 // Neutral for non-Go tasks
	}

	score := 1.0

	// Check for common Go anti-patterns
	antiPatterns := []struct {
		Pattern *regexp.Regexp
		Penalty float64
	}{
		// Error handling
		{regexp.MustCompile(`if err != nil \{\s*\n\s*panic\(`), 0.15},
		{regexp.MustCompile(`_ = .*\n`), 0.05},      // Blank identifier for error
		{regexp.MustCompile(`\.Error\(\)\)`), 0.02}, // Using Error() instead of %v

		// Naming
		{regexp.MustCompile(`func [A-Z]`), 0.05},      // Exported function without comment
		{regexp.MustCompile(`var [a-z]{1,2} `), 0.02}, // Single-letter variables (outside loops)

		// Concurrency
		{regexp.MustCompile(`go func\(\) \{`), 0.03},                       // Goroutine without context/recovery
		{regexp.MustCompile(`\.Lock\(\)\n[^}]*\n[^}]*\.Unlock\(\)`), 0.10}, // Lock without defer

		// Performance
		{regexp.MustCompile(`\+=\s*"`), 0.02},              // String concatenation in loop
		{regexp.MustCompile(`fmt\.Sprintf.*%s.*%s`), 0.02}, // Could use strings.Builder
	}

	for _, ap := range antiPatterns {
		if matches := ap.Pattern.FindAllString(output, -1); len(matches) > 0 {
			score -= ap.Penalty * float64(len(matches))
		}
	}

	// Bonus for good practices
	bonusPatterns := []struct {
		Pattern *regexp.Regexp
		Bonus   float64
	}{
		{regexp.MustCompile(`defer .*\.Close\(\)`), 0.03},
		{regexp.MustCompile(`context\.Context\)`), 0.05},
		{regexp.MustCompile(`errors\.(Is|As)\(`), 0.05},
		{regexp.MustCompile(`t\.(Run|Helper|Cleanup|Parallel)\(`), 0.05},            // Good test practices
		{regexp.MustCompile(`// \w+ (validates|returns|creates|implements)`), 0.03}, // Good comments
	}

	for _, bp := range bonusPatterns {
		if matches := bp.Pattern.FindAllString(output, -1); len(matches) > 0 {
			score += bp.Bonus * float64(len(matches))
		}
	}

	if score < 0 {
		score = 0
	}
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// estimateTestPassRate estimates whether generated tests would pass
func estimateTestPassRate(output string) float64 {
	// Heuristic: if code includes test functions with proper assertions, higher pass rate
	hasTests := regexp.MustCompile(`func Test\w+\(t \*testing\.T\)`).MatchString(output)
	hasAssertions := regexp.MustCompile(`t\.(Errorf|Fatalf|Logf)\(`).MatchString(output)
	hasTableTests := regexp.MustCompile(`\{\s*name:\s*"`).MatchString(output)
	hasSubtests := regexp.MustCompile(`t\.Run\(`).MatchString(output)

	if !hasTests {
		return 0.0
	}

	score := 0.5 // Base: tests exist
	if hasAssertions {
		score += 0.15
	}
	if hasTableTests {
		score += 0.2
	}
	if hasSubtests {
		score += 0.15
	}

	if score > 1.0 {
		score = 1.0
	}
	return score
}

// estimateLintScore estimates golangci-lint compliance
func estimateLintScore(output string) float64 {
	score := 1.0

	lintViolations := []*regexp.Regexp{
		regexp.MustCompile(`package .+ // want`),                                // Uncommented exported
		regexp.MustCompile(`var \w+ = \w+\(\)\s*//\s*$`),                        // Missing error check hint
		regexp.MustCompile(`if err != nil \{\s*\n\s*return nil, err\s*\n\s*\}`), // Could wrap error
	}

	for _, lv := range lintViolations {
		if matches := lv.FindAllString(output, -1); len(matches) > 0 {
			score -= 0.05 * float64(len(matches))
		}
	}

	if score < 0 {
		score = 0
	}
	return score
}
