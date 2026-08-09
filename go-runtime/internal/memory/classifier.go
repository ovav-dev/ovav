package memory

import "strings"

// Classifier decides whether information can be stored in the ledger
// based on privacy sensitivity.
type Classifier struct {
	allowSensitive bool // when true, allows sensitive_local (Systems only)
}

// NewClassifier creates a privacy classifier.
// allowSensitive should be true for Systems, false for Product.
func NewClassifier(allowSensitive bool) *Classifier {
	return &Classifier{allowSensitive: allowSensitive}
}

// ClassifyResult is the output of classification.
type ClassifyResult struct {
	Allow  bool
	Tag    PrivacyTag
	Reason string
}

// Classify evaluates whether a card's content is safe to store.
func (c *Classifier) Classify(card Card) ClassifyResult {
	// Secret data: never store
	if c.containsSecret(card) {
		return ClassifyResult{Allow: false, Tag: PrivacySecret,
			Reason: "contains secret material"}
	}

	// Identity/personal: minimize and scope
	if c.containsIdentity(card) {
		if c.allowSensitive {
			return ClassifyResult{Allow: true, Tag: PrivacyIdentity,
				Reason: "identity data allowed in Systems"}
		}
		return ClassifyResult{Allow: false, Tag: PrivacyIdentity,
			Reason: "identity data blocked outside Systems"}
	}

	// Sensitive local: Systems only
	if c.containsSensitive(card) {
		if c.allowSensitive {
			return ClassifyResult{Allow: true, Tag: PrivacySensitiveLocal,
				Reason: "sensitive data allowed in Systems"}
		}
		return ClassifyResult{Allow: false, Tag: PrivacySensitiveLocal,
			Reason: "sensitive data blocked outside Systems"}
	}

	// Internal project: store scoped
	if c.containsInternal(card) {
		return ClassifyResult{Allow: true, Tag: PrivacyInternalProject,
			Reason: "internal project data — store scoped"}
	}

	// Default: public project
	return ClassifyResult{Allow: true, Tag: PrivacyPublicProject,
		Reason: "public-safe content"}
}

// containsSecret checks for secret-like patterns.
func (c *Classifier) containsSecret(card Card) bool {
	secretIndicators := []string{
		"api_key", "API_KEY", "password", "PASSWORD",
		"token", "TOKEN", "secret", "SECRET",
		"credential", "CREDENTIAL", "private_key", "PRIVATE_KEY",
	}
	for _, indicator := range secretIndicators {
		if strings.Contains(card.Summary, indicator) ||
			strings.Contains(card.OperationalRule, indicator) {
			return true
		}
	}
	return false
}

// containsIdentity checks for personal identity patterns.
func (c *Classifier) containsIdentity(card Card) bool {
	identityIndicators := []string{
		"email", "phone", "address", "passport",
		"@", "personal", "PII",
	}
	for _, indicator := range identityIndicators {
		if strings.Contains(card.Summary, indicator) ||
			strings.Contains(card.OperationalRule, indicator) {
			return true
		}
	}
	return false
}

// containsSensitive checks for sensitive local patterns.
func (c *Classifier) containsSensitive(card Card) bool {
	sensitiveIndicators := []string{
		"internal", "endpoint", "backend", "database",
		"config/", ".ovav/", "workstation", "local_path",
	}
	for _, indicator := range sensitiveIndicators {
		if strings.Contains(card.Summary, indicator) ||
			strings.Contains(card.OperationalRule, indicator) {
			return true
		}
	}
	return false
}

// containsInternal checks for internal project patterns.
func (c *Classifier) containsInternal(card Card) bool {
	return c.containsSensitive(card) || c.containsIdentity(card)
}
