package gitflow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// PrePushResult holds the results of a pre-push intelligence scan.
// Replaces: tools/validators/pre_push_intelligence.py (Go-native v1.0)
type PrePushResult struct {
	Files          PrePushFiles   `json:"files"`
	NoiseFindings  []NoiseFinding `json:"noise,omitempty"`
	SecretFindings []SecretFind   `json:"secrets,omitempty"`
	EmptyFindings  []EmptyFinding `json:"empty,omitempty"`
	BlockingIssues int            `json:"blocking_issues"`
	TotalIssues    int            `json:"total_issues"`
	Clean          bool           `json:"clean"`
}

type PrePushFiles struct {
	Staged    int `json:"staged"`
	Unstaged  int `json:"unstaged"`
	Untracked int `json:"untracked"`
	Total     int `json:"total"`
}

type NoiseFinding struct {
	File    string `json:"file"`
	Pattern string `json:"pattern"`
	Detail  string `json:"detail"`
}

type SecretFind struct {
	Line       string `json:"line"`
	SecretType string `json:"secret_type"`
	Detail     string `json:"detail"`
}

type EmptyFinding struct {
	File   string `json:"file"`
	Issue  string `json:"issue"`
	Detail string `json:"detail"`
}

// noisePatterns — files that should never be committed.
var noisePatterns = []*regexp.Regexp{
	regexp.MustCompile(`__pycache__/`),
	regexp.MustCompile(`\.pyc$`),
	regexp.MustCompile(`\.pytest_cache/`),
	regexp.MustCompile(`node_modules/`),
	regexp.MustCompile(`\.log$`),
	regexp.MustCompile(`\.tmp$`),
	regexp.MustCompile(`\.backup$`),
	regexp.MustCompile(`\.bak$`),
	regexp.MustCompile(`\.zip$`),
	regexp.MustCompile(`\.pdf$`),
	regexp.MustCompile(`\.local/`),
	regexp.MustCompile(`\.config/`),
	regexp.MustCompile(`\.DS_Store$`),
	regexp.MustCompile(`Thumbs\.db$`),
	regexp.MustCompile(`\.ovav/runtime/logs/`),
	regexp.MustCompile(`\.ovav/runtime/sessions/`),
	regexp.MustCompile(`\.ovav/integrity/`),
	regexp.MustCompile(`/tmp/`),
}

// secretPatterns — regex patterns for common secret types.
var secretPatterns = []struct {
	re    *regexp.Regexp
	label string
}{
	{regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`), "OpenAI API key"},
	{regexp.MustCompile(`sk-ant-[a-zA-Z0-9_-]{20,}`), "Anthropic API key"},
	{regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`), "Google API key"},
	{regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`), "GitHub personal token"},
	{regexp.MustCompile(`gho_[a-zA-Z0-9]{36}`), "GitHub OAuth token"},
	{regexp.MustCompile(`github_pat_[a-zA-Z0-9_]{40,}`), "GitHub fine-grained token"},
	{regexp.MustCompile(`AKIA[0-9A-Z]{16}`), "AWS Access Key"},
	{regexp.MustCompile(`password\s*[:=]\s*["'][^"']+["']`), "Hardcoded password"},
	{regexp.MustCompile(`secret\s*[:=]\s*["'][^"']+["']`), "Hardcoded secret"},
	{regexp.MustCompile(`token\s*[:=]\s*["'][^"']{10,}["']`), "Hardcoded token"},
	{regexp.MustCompile(`BEGIN (RSA|DSA|EC|OPENSSH) PRIVATE KEY`), "Private key"},
}

// PrePushScan runs a pre-push intelligence scan on the repository at repoRoot.
// It checks staged, unstaged, and untracked files for noise, secrets, and empty files.
func PrePushScan(repoRoot string) (*PrePushResult, error) {
	r := &PrePushResult{Clean: true}

	// Gather file lists
	staged := gitFileList(repoRoot, "diff", "--cached", "--name-only")
	unstaged := gitFileList(repoRoot, "diff", "--name-only")
	untracked := gitFileList(repoRoot, "ls-files", "--others", "--exclude-standard")

	r.Files = PrePushFiles{
		Staged:    len(staged),
		Unstaged:  len(unstaged),
		Untracked: len(untracked),
		Total:     len(staged) + len(unstaged) + len(untracked),
	}

	allFiles := append(append(staged, unstaged...), untracked...)

	// 1. Noise detection
	for _, f := range allFiles {
		for _, pat := range noisePatterns {
			if pat.MatchString(f) {
				r.NoiseFindings = append(r.NoiseFindings, NoiseFinding{
					File:    f,
					Pattern: pat.String(),
					Detail:  "Noise file — should not be committed",
				})
				r.Clean = false
				break
			}
		}
	}
	r.TotalIssues += len(r.NoiseFindings)

	// 2. Secret detection (scan diff for added lines)
	diffText := gitOutput(repoRoot, "diff", "HEAD")
	for _, line := range strings.Split(diffText, "\n") {
		if !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
			continue
		}
		content := line[1:]
		for _, sp := range secretPatterns {
			if sp.re.MatchString(content) {
				display := content
				if len(display) > 100 {
					display = display[:100] + "..."
				}
				r.SecretFindings = append(r.SecretFindings, SecretFind{
					Line:       display,
					SecretType: sp.label,
					Detail:     fmt.Sprintf("Potential %s found in diff", sp.label),
				})
				r.Clean = false
				r.BlockingIssues++
				break
			}
		}
	}
	r.TotalIssues += len(r.SecretFindings)

	// 3. Empty file detection
	for _, f := range allFiles {
		fullPath := filepath.Join(repoRoot, f)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		if info.Size() == 0 {
			r.EmptyFindings = append(r.EmptyFindings, EmptyFinding{
				File:   f,
				Issue:  "empty_file",
				Detail: "File is empty (0 bytes)",
			})
		}
	}
	r.TotalIssues += len(r.EmptyFindings)

	return r, nil
}

// gitFileList runs a git command and returns the output as a list of non-empty lines.
func gitFileList(repoRoot string, args ...string) []string {
	out := gitOutput(repoRoot, args...)
	if out == "" {
		return nil
	}
	var files []string
	for _, f := range strings.Split(out, "\n") {
		f = strings.TrimSpace(f)
		if f != "" {
			files = append(files, f)
		}
	}
	return files
}

// gitOutput runs a git command and returns stdout as string.
func gitOutput(repoRoot string, args ...string) string {
	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
