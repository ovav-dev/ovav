package templates

import "fmt"

// VaultRotation returns the HTML for a secret rotation reminder email.
func VaultRotation(data map[string]interface{}) string {
	secretName := ""
	if v, ok := data["secret_name"].(string); ok {
		secretName = v
	}

	secretType := "API Key"
	if v, ok := data["secret_type"].(string); ok {
		secretType = v
	}

	daysUntilExpiration := 0
	if v, ok := data["days_until_expiration"].(int); ok {
		daysUntilExpiration = v
	}

	lastRotated := ""
	if v, ok := data["last_rotated"].(string); ok {
		lastRotated = v
	}

	vaultURL := ""
	if v, ok := data["vault_url"].(string); ok {
		vaultURL = v
	}

	expirationWarning := "upcoming expiration"
	if daysUntilExpiration <= 3 {
		expirationWarning = "expiring soon — immediate action required"
	} else if daysUntilExpiration <= 7 {
		expirationWarning = "expiring within a week"
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Secret Rotation Reminder - OVAV</title>
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
                        <rect x="3" y="11" width="18" height="11" rx="2" stroke="white" stroke-width="2" fill="none"/>
                        <path d="M7 11V7a5 5 0 0110 0v4" stroke="white" stroke-width="2" stroke-linecap="round"/>
                        <circle cx="12" cy="16" r="1.5" fill="white"/>
                      </svg>
                    </div>
                  </td>
                  <td style="vertical-align:middle;padding-left:16px;">
                    <h1 style="margin:0;font-size:24px;font-weight:700;color:#ffffff;">Secret Rotation Reminder</h1>
                    <p style="margin:4px 0 0;font-size:14px;color:rgba(255,255,255,0.85);">` + expirationWarning + `</p>
                  </td>
                </tr>
              </table>
            </td>
          </tr>

          <!-- Body -->
          <tr>
            <td style="padding:40px;">
              <p style="margin:0 0 24px;font-size:16px;line-height:1.6;color:#1f2937;">
                A secret in your OVAV vault requires rotation. Regular rotation is essential for maintaining security compliance and reducing the risk of credential exposure.
              </p>

              <!-- Secret Details Card -->
              <div style="background:#f5f3ff;border-radius:12px;padding:24px;margin:0 0 24px;">
                <h3 style="margin:0 0 16px;font-size:16px;font-weight:600;color:#7c3aed;">Secret Details</h3>
                <table role="presentation" width="100%" cellspacing="0" cellpadding="0">
                  <tr>
                    <td style="padding:10px 0;font-size:14px;color:#6b7280;border-bottom:1px solid #e5e7eb;">Name</td>
                    <td style="padding:10px 0;font-size:14px;font-weight:600;color:#1f2937;text-align:right;border-bottom:1px solid #e5e7eb;">` + secretName + `</td>
                  </tr>
                  <tr>
                    <td style="padding:10px 0;font-size:14px;color:#6b7280;border-bottom:1px solid #e5e7eb;">Type</td>
                    <td style="padding:10px 0;font-size:14px;font-weight:600;color:#1f2937;text-align:right;border-bottom:1px solid #e5e7eb;">` + secretType + `</td>
                  </tr>
                  <tr>
                    <td style="padding:10px 0;font-size:14px;color:#6b7280;border-bottom:1px solid #e5e7eb;">Last Rotated</td>
                    <td style="padding:10px 0;font-size:14px;font-weight:600;color:#1f2937;text-align:right;border-bottom:1px solid #e5e7eb;">` + lastRotated + `</td>
                  </tr>
                  <tr>
                    <td style="padding:10px 0;font-size:14px;color:#6b7280;">Days Until Expiration</td>
                    <td style="padding:10px 0;font-size:14px;font-weight:700;color:#f59e0b;text-align:right;">` + fmt.Sprintf("%d", daysUntilExpiration) + ` days</td>
                  </tr>
                </table>
              </div>

              <!-- Warning Banner -->
              ` + func() string {
		if daysUntilExpiration <= 3 {
			return `<div style="background:#fef2f2;border-left:4px solid #ef4444;border-radius:4px;padding:16px;margin:0 0 24px;">
						<p style="margin:0;font-size:14px;color:#991b1b;font-weight:500;">
							<strong>Urgent:</strong> This secret will expire in ` + fmt.Sprintf("%d", daysUntilExpiration) + ` day(s). Rotating now is highly recommended to avoid service disruption.
						</p>
					</div>`
		}
		return `<div style="background:#fffbeb;border-left:4px solid #f59e0b;border-radius:4px;padding:16px;margin:0 0 24px;">
					<p style="margin:0;font-size:14px;color:#92400e;font-weight:500;">
						This secret will expire in ` + fmt.Sprintf("%d", daysUntilExpiration) + ` days. Schedule rotation at your earliest convenience.
					</p>
				</div>`
	}() + `

              <div style="text-align:center;margin:32px 0;">
                <a href="` + vaultURL + `" style="display:inline-block;background:linear-gradient(135deg,#7c3aed 0%,#5b21b6 100%);color:#ffffff;font-size:16px;font-weight:600;text-decoration:none;padding:16px 40px;border-radius:8px;">Rotate Secret Now</a>
              </div>

              <p style="margin:0;font-size:13px;color:#9ca3af;text-align:center;">
                Need help? Visit the <a href="#" style="color:#7c3aed;">documentation</a> or contact your security administrator.
              </p>
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
