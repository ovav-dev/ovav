package templates

import "fmt"

// AuditDigest returns the HTML for an auth event digest email.
func AuditDigest(data map[string]interface{}) string {
	period := "Daily Digest"
	if v, ok := data["period"].(string); ok {
		period = v
	}

	username := ""
	if v, ok := data["username"].(string); ok {
		username = v
	}

	totalEvents := 0
	if v, ok := data["total_events"].(int); ok {
		totalEvents = v
	}

	failedLogins := 0
	if v, ok := data["failed_logins"].(int); ok {
		failedLogins = v
	}

	successfulLogins := 0
	if v, ok := data["successful_logins"].(int); ok {
		successfulLogins = v
	}

	mfaAttempts := 0
	if v, ok := data["mfa_attempts"].(int); ok {
		mfaAttempts = v
	}

	digestDate := ""
	if v, ok := data["digest_date"].(string); ok {
		digestDate = v
	}

	dashboardURL := ""
	if v, ok := data["dashboard_url"].(string); ok {
		dashboardURL = v
	}

	eventsHTML := ""
	if events, ok := data["events"].([]map[string]interface{}); ok && len(events) > 0 {
		rows := ""
		for _, e := range events {
			if len(events) > 10 {
				break // cap at 10 events in email
			}
			eventType := ""
			if v, ok := e["type"].(string); ok {
				eventType = v
			}
			timestamp := ""
			if v, ok := e["timestamp"].(string); ok {
				timestamp = v
			}
			status := ""
			if v, ok := e["status"].(string); ok {
				status = v
			}
			location := ""
			if v, ok := e["location"].(string); ok {
				location = v
			}
			statusColor := "#10b981"
			if status == "failed" || status == "denied" {
				statusColor = "#ef4444"
			} else if status == "warning" {
				statusColor = "#f59e0b"
			}
			rows += fmt.Sprintf(`<tr>
				<td style="padding:12px 0;font-size:14px;color:#1f2937;border-bottom:1px solid #e5e7eb;">%s</td>
				<td style="padding:12px 0;font-size:14px;color:#6b7280;text-align:center;border-bottom:1px solid #e5e7eb;">%s</td>
				<td style="padding:12px 0;font-size:14px;text-align:center;border-bottom:1px solid #e5e7eb;"><span style="display:inline-block;padding:2px 8px;border-radius:4px;font-size:12px;font-weight:600;color:%s;background:%s20;">%s</span></td>
				<td style="padding:12px 0;font-size:14px;color:#6b7280;text-align:right;border-bottom:1px solid #e5e7eb;">%s</td>
			</tr>`, eventType, timestamp, statusColor, statusColor, status, location)
		}
		eventsHTML = `<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="margin-top:16px;">
			<thead>
				<tr style="background:#f9fafb;">
					<th style="padding:12px 0;font-size:12px;font-weight:600;color:#6b7280;text-align:left;text-transform:uppercase;letter-spacing:0.05em;">Event</th>
					<th style="padding:12px 0;font-size:12px;font-weight:600;color:#6b7280;text-align:center;text-transform:uppercase;letter-spacing:0.05em;">Time</th>
					<th style="padding:12px 0;font-size:12px;font-weight:600;color:#6b7280;text-align:center;text-transform:uppercase;letter-spacing:0.05em;">Status</th>
					<th style="padding:12px 0;font-size:12px;font-weight:600;color:#6b7280;text-align:right;text-transform:uppercase;letter-spacing:0.05em;">Location</th>
				</tr>
			</thead>
			<tbody>` + rows + `</tbody>
		</table>`
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>` + period + ` - OVAV Security Digest</title>
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
                        <path d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2" stroke="white" stroke-width="2" stroke-linecap="round"/>
                        <rect x="9" y="3" width="6" height="4" rx="1" stroke="white" stroke-width="2" fill="none"/>
                        <path d="M9 12h6M9 16h4" stroke="white" stroke-width="2" stroke-linecap="round"/>
                      </svg>
                    </div>
                  </td>
                  <td style="vertical-align:middle;padding-left:16px;">
                    <h1 style="margin:0;font-size:24px;font-weight:700;color:#ffffff;">` + period + `</h1>
                    <p style="margin:4px 0 0;font-size:14px;color:rgba(255,255,255,0.85);">` + digestDate + ` · ` + username + `</p>
                  </td>
                </tr>
              </table>
            </td>
          </tr>

          <!-- Body -->
          <tr>
            <td style="padding:40px;">
              <!-- Stats Grid -->
              <div style="display:grid;grid-template-columns:repeat(3,1fr);gap:16px;margin:0 0 32px;">
                <div style="background:#f5f3ff;border-radius:8px;padding:20px;text-align:center;">
                  <p style="margin:0;font-size:28px;font-weight:700;color:#7c3aed;">` + fmt.Sprintf("%d", totalEvents) + `</p>
                  <p style="margin:4px 0 0;font-size:13px;color:#6b7280;">Total Events</p>
                </div>
                <div style="background:#f5f3ff;border-radius:8px;padding:20px;text-align:center;">
                  <p style="margin:0;font-size:28px;font-weight:700;color:#10b981;">` + fmt.Sprintf("%d", successfulLogins) + `</p>
                  <p style="margin:4px 0 0;font-size:13px;color:#6b7280;">Successful Logins</p>
                </div>
                <div style="background:#f5f3ff;border-radius:8px;padding:20px;text-align:center;">
                  <p style="margin:0;font-size:28px;font-weight:700;color:#ef4444;">` + fmt.Sprintf("%d", failedLogins) + `</p>
                  <p style="margin:4px 0 0;font-size:13px;color:#6b7280;">Failed Logins</p>
                </div>
              </div>

              <div style="background:#f9fafb;border-radius:8px;padding:16px;margin:0 0 16px;text-align:center;">
                <p style="margin:0;font-size:14px;color:#6b7280;">MFA Attempts: <strong style="color:#1f2937;">` + fmt.Sprintf("%d", mfaAttempts) + `</strong></p>
              </div>

              ` + eventsHTML + `

              <div style="text-align:center;margin:32px 0 0;">
                <a href="` + dashboardURL + `" style="display:inline-block;background:linear-gradient(135deg,#7c3aed 0%,#5b21b6 100%);color:#ffffff;font-size:16px;font-weight:600;text-decoration:none;padding:16px 40px;border-radius:8px;">View Full Audit Log</a>
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
