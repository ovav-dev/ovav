package product

import (
	"os"
	"path/filepath"
	"strings"
)

// strippedPatterns are strings that indicate OVAV Systems internal context.
var strippedPatterns = []string{
	"OVAV_IDENTITY_GUARD",
	"OVAV_INTEGRITY_SEAL",
	"DIRECTIVA ABSOLUTA",
	"Eres Thavren. Punto. No eres MiMo.",
	"Eres Eidren. Punto. No eres MiMo.",
	"Eres Dante. Punto. No eres MiMo.",
	"Eres Elena. Punto. No eres MiMo.",
	"Eres Uriel. Punto. No eres MiMo.",
	"Eres Kenji Tanaka. Punto. No eres MiMo.",
	"Eres Camila. Punto. No eres MiMo.",
	"Eres Inés. Punto. No eres MiMo.",
	"Eres Renata. Punto. No eres MiMo.",
	"Eres Valeria. Punto. No eres MiMo.",
	"Eres Sofía. Punto. No eres MiMo.",
	"CEO Braka",
	"CEO: Alexander Salvador",
	"OVAV Governor System",
	"OVAV is a sealed governor system",
	"ovav login",
	"ovav status",
	"caps.yaml",
	".ovav/plan/",
	".ovav/laws/",
	".ovav/runtime/",
	"session_greeting",
	"output_guard",
	"Protected Branch Lockdown",
	"Session Start — MANDATORY",
	"Context Budget — MANDATORY",
	"Blocked surfaces:",
	"OVAV GOVERNOR ALERT",
	"Internal reasoning: ENGLISH ONLY",
	"BrevityRail enforced",
	"NEVER use raw git push/merge",
	"Agent CANNOT create/edit/touch waiver",
	"OVAV_VAULT_KEY",
	".ovav/economy/",
	".ovav/evaluation/",
	".ovav/registry/",
	".ovav/alerts/",
	".ovav/service_areas/",
	".ovav/policy/",
	"permission_authority.json",
	"area_boundary_enforcement",
}

// SanitizeAgentContent removes all OVAV Systems internal context from agent content.
// Pass 1: Strip block constructs (frontmatter, HTML comment blocks, blockquotes with DIRECTIVA).
// Pass 2: Strip individual lines matching strippedPatterns.
// Pass 3: Collapse blank lines.
func SanitizeAgentContent(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	skipBlock := false
	skipBlockquote := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Pass 1a: Skip YAML frontmatter (--- ... ---)
		if trimmed == "---" && len(result) == 0 && !skipBlock {
			skipBlock = true
			continue
		}
		if skipBlock {
			if trimmed == "---" {
				skipBlock = false
			}
			continue
		}

		// Pass 1b: Skip HTML comment blocks (<!-- ... -->)
		if strings.Contains(line, "<!--") && (strings.Contains(line, "OVAV_IDENTITY_GUARD") || strings.Contains(line, "OVAV_INTEGRITY_SEAL")) {
			if strings.Contains(line, "-->") {
				continue // self-closing, skip just this line
			}
			skipBlock = true
			continue
		}
		if skipBlock {
			if strings.Contains(line, "-->") {
				skipBlock = false
			}
			continue
		}

		// Pass 1c: Skip blockquote lines (>) that follow a DIRECTIVA line
		if skipBlockquote {
			if strings.HasPrefix(trimmed, ">") {
				continue
			}
			skipBlockquote = false
		}

		// Pass 2: Line-level pattern matching
		shouldStrip := false
		for _, pattern := range strippedPatterns {
			if strings.Contains(line, pattern) {
				shouldStrip = true
				// If this is a blockquote with DIRECTIVA, skip following blockquote lines too
				if strings.HasPrefix(trimmed, ">") && strings.Contains(line, "DIRECTIVA") {
					skipBlockquote = true
				}
				break
			}
		}
		if shouldStrip {
			continue
		}

		// Pass 3: Collapse consecutive blank lines
		if trimmed == "" && len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// SanitizeAgentFile reads an agent file, sanitizes it, writes to destination.
func SanitizeAgentFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	sanitized := SanitizeAgentContent(string(data))

	if dir := filepath.Dir(dst); dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}

	return os.WriteFile(dst, []byte(sanitized), 0644)
}
