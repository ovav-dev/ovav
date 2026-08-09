package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ovav/ovav/internal/email/templates"
)

const (
	resendAPIURL = "https://api.resend.com/emails"
	fromAddress  = "OVAV <ovav@ovav.dev>"
	httpTimeout  = 15 * time.Second
)

// TemplateType represents the email template type.
type TemplateType int

const (
	TemplateWelcome TemplateType = iota
	TemplateAlert
	TemplateAuditDigest
	TemplateOTP
	TemplateVaultRotation
)

func (t TemplateType) String() string {
	switch t {
	case TemplateWelcome:
		return "welcome"
	case TemplateAlert:
		return "alert"
	case TemplateAuditDigest:
		return "audit_digest"
	case TemplateOTP:
		return "otp"
	case TemplateVaultRotation:
		return "vault_rotation"
	default:
		return "unknown"
	}
}

func (t TemplateType) subject(data map[string]interface{}) string {
	switch t {
	case TemplateWelcome:
		return "Welcome to OVAV — Your Account is Ready"
	case TemplateAlert:
		if alertType, ok := data["alert_type"].(templates.AlertType); ok {
			switch alertType {
			case templates.AlertTypeBreachAttempt:
				return "⚠️ OVAV Security Alert: Breach Attempt Detected"
			case templates.AlertTypeAlarmTrigger:
				return "⚠️ OVAV Security Alert: Alarm Triggered"
			}
		}
		return "⚠️ OVAV Security Alert: Anomaly Detected"
	case TemplateAuditDigest:
		period := "Daily"
		if p, ok := data["period"].(string); ok {
			period = p
		}
		return fmt.Sprintf("OVAV Auth Digest — %s Summary", period)
	case TemplateOTP:
		return "Your OVAV Verification Code"
	case TemplateVaultRotation:
		return "OVAV Secret Rotation Reminder"
	default:
		return "OVAV Notification"
	}
}

func (t TemplateType) render(data map[string]interface{}) string {
	switch t {
	case TemplateWelcome:
		return templates.Welcome(data)
	case TemplateAlert:
		return templates.Alert(data)
	case TemplateAuditDigest:
		return templates.AuditDigest(data)
	case TemplateOTP:
		return templates.OTP(data)
	case TemplateVaultRotation:
		return templates.VaultRotation(data)
	default:
		return ""
	}
}

// getResendAPIKey loads the Resend API key.
// Production: loaded from OVAV vault (credentialFromVault).
// Development: falls back to RESEND_API_KEY env var.
// The vault path is ~/.local/share/ovav/secrets.vault.
func getResendAPIKey() (string, error) {
	// TODO: Phase 6 — integrate with credentialFromVault pattern
	// For now, env var fallback for local dev only.
	if key := os.Getenv("RESEND_API_KEY"); key != "" {
		return key, nil
	}
	return "", fmt.Errorf("RESEND_API_KEY not set and vault integration not yet implemented")
}

// emailPayload is the JSON structure for the Resend API.
type emailPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

// SendEmail sends an email using the Resend API.
func SendEmail(to string, template TemplateType, data map[string]interface{}) error {
	payload := emailPayload{
		From:    fromAddress,
		To:      to,
		Subject: template.subject(data),
		HTML:    template.render(data),
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, resendAPIURL, bytes.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	apiKey, err := getResendAPIKey()
	if err != nil {
		return fmt.Errorf("resend API key unavailable: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("resend API error: status %d", resp.StatusCode)
	}

	return nil
}
