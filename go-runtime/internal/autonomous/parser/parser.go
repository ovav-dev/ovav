// Package parser implements OVAV's content parsing for research findings.
package parser

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/autonomous"
)

// Parser extracts structured findings from raw web content.
type Parser struct {
	targetID string
}

// New creates a new parser for a specific target.
func New(targetID string) *Parser {
	return &Parser{targetID: targetID}
}

// ParseHTML extracts findings from HTML content.
func (p *Parser) ParseHTML(content string) ([]autonomous.Finding, error) {
	var findings []autonomous.Finding

	// Extract title patterns based on target
	switch p.targetID {
	case "openai":
		findings = p.parseOpenAI(content)
	case "anthropic":
		findings = p.parseAnthropic(content)
	case "google-ai":
		findings = p.parseGoogleAI(content)
	case "openrouter":
		findings = p.parseOpenRouter(content)
	case "owasp":
		findings = p.parseOWASP(content)
	default:
		findings = p.parseGeneric(content)
	}

	return findings, nil
}

// parseOpenAI extracts OpenAI-specific findings.
func (p *Parser) parseOpenAI(content string) []autonomous.Finding {
	var findings []autonomous.Finding

	// Look for model announcements
	modelPattern := regexp.MustCompile(`(?i)(gpt-[45]|o[1-9]|model|deprecated|update)`)
	matches := modelPattern.FindAllString(content, -1)

	for i, match := range matches {
		findings = append(findings, autonomous.Finding{
			ID:          fmt.Sprintf("openai-%d", i),
			Title:       fmt.Sprintf("OpenAI mention: %s", match),
			Description: "Potential model update detected",
			Source:      "openai",
			URL:         "https://platform.openai.com/docs/changelog",
			Severity:    "info",
			Category:    "model",
			Discovered:  time.Now(),
		})
	}

	return findings
}

// parseAnthropic extracts Anthropic-specific findings.
func (p *Parser) parseAnthropic(content string) []autonomous.Finding {
	var findings []autonomous.Finding

	modelPattern := regexp.MustCompile(`(?i)(claude|opus|sonnet|haiku|update|release)`)
	matches := modelPattern.FindAllString(content, -1)

	for i, match := range matches {
		findings = append(findings, autonomous.Finding{
			ID:          fmt.Sprintf("anthropic-%d", i),
			Title:       fmt.Sprintf("Anthropic mention: %s", match),
			Description: "Potential Claude update detected",
			Source:      "anthropic",
			URL:         "https://docs.anthropic.com/en/release-notes/overview",
			Severity:    "info",
			Category:    "model",
			Discovered:  time.Now(),
		})
	}

	return findings
}

// parseGoogleAI extracts Google AI findings.
func (p *Parser) parseGoogleAI(content string) []autonomous.Finding {
	var findings []autonomous.Finding

	modelPattern := regexp.MustCompile(`(?i)(gemini|palm|update|model|release)`)
	matches := modelPattern.FindAllString(content, -1)

	for i, match := range matches {
		findings = append(findings, autonomous.Finding{
			ID:          fmt.Sprintf("google-ai-%d", i),
			Title:       fmt.Sprintf("Google AI mention: %s", match),
			Description: "Potential Gemini/Palm update detected",
			Source:      "google-ai",
			URL:         "https://ai.google/discover/palm-api",
			Severity:    "info",
			Category:    "model",
			Discovered:  time.Now(),
		})
	}

	return findings
}

// parseOpenRouter extracts OpenRouter findings.
func (p *Parser) parseOpenRouter(content string) []autonomous.Finding {
	var findings []autonomous.Finding

	modelPattern := regexp.MustCompile(`(?i)(model|pricing|new|available)`)
	matches := modelPattern.FindAllString(content, -1)

	for i, match := range matches {
		findings = append(findings, autonomous.Finding{
			ID:          fmt.Sprintf("openrouter-%d", i),
			Title:       fmt.Sprintf("OpenRouter mention: %s", match),
			Description: "Potential pricing/model update detected",
			Source:      "openrouter",
			URL:         "https://openrouter.ai/docs",
			Severity:    "info",
			Category:    "pricing",
			Discovered:  time.Now(),
		})
	}

	return findings
}

// parseOWASP extracts OWASP security findings.
func (p *Parser) parseOWASP(content string) []autonomous.Finding {
	var findings []autonomous.Finding

	// Look for vulnerability mentions
	vulnPattern := regexp.MustCompile(`(?i)(vulnerability|cve|security|advisory|injection|xss|csrf)`)
	matches := vulnPattern.FindAllString(content, -1)

	for i, match := range matches {
		findings = append(findings, autonomous.Finding{
			ID:          fmt.Sprintf("owasp-%d", i),
			Title:       fmt.Sprintf("Security mention: %s", match),
			Description: "Potential security advisory detected",
			Source:      "owasp",
			URL:         "https://owasp.org/",
			Severity:    "warning",
			Category:    "security",
			Discovered:  time.Now(),
		})
	}

	return findings
}

// parseGeneric extracts generic findings for unknown targets.
func (p *Parser) parseGeneric(content string) []autonomous.Finding {
	// Extract first 1000 chars as sample
	sample := content
	if len(sample) > 1000 {
		sample = sample[:1000]
	}

	// Remove HTML tags for text extraction
	clean := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(sample, " ")
	clean = strings.Join(strings.Fields(clean), " ")

	return []autonomous.Finding{
		{
			ID:          fmt.Sprintf("%s-sample", p.targetID),
			Title:       fmt.Sprintf("%s content sample", p.targetID),
			Description: clean,
			Source:      p.targetID,
			Severity:    "info",
			Category:    "general",
			Discovered:  time.Now(),
		},
	}
}

// ExtractChanges detects changes between old and new content.
func (p *Parser) ExtractChanges(oldContent, newContent string) []autonomous.Change {
	var changes []autonomous.Change

	oldWords := strings.Fields(strings.ToLower(oldContent))
	newWords := strings.Fields(strings.ToLower(newContent))

	// Simple diff: find words in new but not in old
	oldSet := make(map[string]bool)
	for _, w := range oldWords {
		oldSet[w] = true
	}

	var added []string
	for _, w := range newWords {
		if !oldSet[w] && len(w) > 3 {
			added = append(added, w)
		}
	}

	if len(added) > 0 {
		changes = append(changes, autonomous.Change{
			ID:          fmt.Sprintf("%s-change-%d", p.targetID, time.Now().Unix()),
			Type:        "added",
			Title:       fmt.Sprintf("New content detected on %s", p.targetID),
			Description: fmt.Sprintf("Found %d new terms", len(added)),
			Detected:    time.Now(),
		})
	}

	return changes
}
