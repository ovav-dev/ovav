package templates

// AlertType represents the type of security alert.
type AlertType string

const (
	AlertTypeAnomaly       AlertType = "anomaly"
	AlertTypeBreachAttempt AlertType = "breach_attempt"
	AlertTypeAlarmTrigger  AlertType = "alarm_trigger"
)

// Alert returns the HTML for a security alert email.
func Alert(data map[string]interface{}) string {
	alertType := AlertTypeAnomaly
	if v, ok := data["alert_type"].(AlertType); ok {
		alertType = v
	}

	severity := "Medium"
	if v, ok := data["severity"].(string); ok {
		severity = v
	}

	title := "Security Alert"
	description := "An anomaly was detected in your account activity."
	iconColor := "#f59e0b"

	switch alertType {
	case AlertTypeBreachAttempt:
		title = "Breach Attempt Detected"
		description = "We detected a potential breach attempt on your account. Please review the details below and take immediate action."
		iconColor = "#ef4444"
		severity = "Critical"
	case AlertTypeAlarmTrigger:
		title = "Alarm Triggered"
		description = "A configured security alarm has been triggered. This requires your attention."
		iconColor = "#ef4444"
		severity = "Critical"
	case AlertTypeAnomaly:
		title = "Anomaly Detected"
		description = "Unusual activity was detected in your account. Please review to confirm this was you."
		_ = iconColor // suppress unused warning
	}

	timestamp := ""
	if v, ok := data["timestamp"].(string); ok {
		timestamp = v
	}
	location := ""
	if v, ok := data["location"].(string); ok {
		location = v
	}
	ipAddress := ""
	if v, ok := data["ip_address"].(string); ok {
		ipAddress = v
	}
	actionURL := ""
	if v, ok := data["action_url"].(string); ok {
		actionURL = v
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>` + title + ` - OVAV Security</title>
</head>
<body style="margin:0;padding:0;background:#f5f3ff;font-family:'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
  <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#f5f3ff;">
    <tr>
      <td align="center" style="padding:40px 20px;">
        <table role="presentation" width="100%" max-width="600" cellspacing="0" cellpadding="0" style="background:#ffffff;border-radius:12px;overflow:hidden;box-shadow:0 4px 24px rgba(124,58,237,0.1);">
          <!-- Header -->
          <tr>
            <td style="background:linear-gradient(135deg,#7c3aed 0%,#5b21b6 100%);padding:40px 40px 32px;">
              <table role="presentation" width="100%" cellspacing="0" cellpadding="0">
                <tr>
                  <td style="vertical-align:middle;">
                    <div style="display:inline-block;width:48px;height:48px;background:rgba(255,255,255,0.2);border-radius:12px;text-align:center;line-height:48px;">
                      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" style="vertical-align:middle;">
                        <path d="M12 2L3 7v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V7l-9-5z" stroke="white" stroke-width="2" fill="none"/>
                        <path d="M12 8v4M12 16h.01" stroke="white" stroke-width="2" stroke-linecap="round"/>
                      </svg>
                    </div>
                  </td>
                  <td style="vertical-align:middle;padding-left:16px;">
                    <h1 style="margin:0;font-size:24px;font-weight:700;color:#ffffff;">` + title + `</h1>
                    <p style="margin:4px 0 0;font-size:14px;color:rgba(255,255,255,0.85);">Severity: ` + severity + `</p>
                  </td>
                </tr>
              </table>
            </td>
          </tr>

          <!-- Body -->
          <tr>
            <td style="padding:40px;">
              <div style="background:#fef2f2;border-left:4px solid #ef4444;border-radius:4px;padding:16px;margin:0 0 24px;">
                <p style="margin:0;font-size:15px;line-height:1.5;color:#991b1b;font-weight:500;">` + description + `</p>
              </div>

              <h3 style="margin:0 0 16px;font-size:16px;font-weight:600;color:#1f2937;">Details</h3>
              <div style="background:#f9fafb;border-radius:8px;padding:20px;margin:0 0 24px;">
                <table role="presentation" width="100%" cellspacing="0" cellpadding="0">
                  <tr>
                    <td style="padding:10px 0;font-size:14px;color:#6b7280;border-bottom:1px solid #e5e7eb;">Time</td>
                    <td style="padding:10px 0;font-size:14px;font-weight:500;color:#1f2937;text-align:right;border-bottom:1px solid #e5e7eb;">` + timestamp + `</td>
                  </tr>
                  <tr>
                    <td style="padding:10px 0;font-size:14px;color:#6b7280;border-bottom:1px solid #e5e7eb;">Location</td>
                    <td style="padding:10px 0;font-size:14px;font-weight:500;color:#1f2937;text-align:right;border-bottom:1px solid #e5e7eb;">` + location + `</td>
                  </tr>
                  <tr>
                    <td style="padding:10px 0;font-size:14px;color:#6b7280;">IP Address</td>
                    <td style="padding:10px 0;font-size:14px;font-weight:500;color:#1f2937;text-align:right;">` + ipAddress + `</td>
                  </tr>
                </table>
              </div>

              <p style="margin:0 0 24px;font-size:14px;line-height:1.6;color:#6b7280;">
                If this was you, you can safely ignore this email. If you do not recognize this activity, we recommend immediately securing your account by changing your password and reviewing your security settings.
              </p>

              <div style="text-align:center;margin:32px 0;">
                <a href="` + actionURL + `" style="display:inline-block;background:#ef4444;color:#ffffff;font-size:16px;font-weight:600;text-decoration:none;padding:16px 40px;border-radius:8px;">Review Activity</a>
              </div>
            </td>
          </tr>

          <!-- Footer -->
          <tr>
            <td style="background:#f9fafb;padding:24px 40px;border-top:1px solid #e5e7eb;">
              <p style="margin:0;font-size:13px;color:#6b7280;text-align:center;">
                OVAV — Secure Identity & Secrets Management<br>
                <a href="#" style="color:#7c3aed;text-decoration:underline;">Unsubscribe</a> · <a href="#" style="color:#7c3aed;text-decoration:underline;">Privacy Policy</a> · <a href="#" style="color:#7c3aed;text-decoration:underline;">Support</a>
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`
}
