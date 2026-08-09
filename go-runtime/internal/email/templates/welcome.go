package templates

// Welcome returns the HTML for a new user welcome email.
func Welcome(data map[string]interface{}) string {
	username := ""
	if v, ok := data["username"].(string); ok {
		username = v
	}
	firstName := ""
	if v, ok := data["first_name"].(string); ok {
		firstName = v
	}
	loginURL := ""
	if v, ok := data["login_url"].(string); ok {
		loginURL = v
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Welcome to OVAV</title>
</head>
<body style="margin:0;padding:0;background:#f5f3ff;font-family:'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
  <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#f5f3ff;">
    <tr>
      <td align="center" style="padding:40px 20px;">
        <table role="presentation" width="100%" max-width="600" cellspacing="0" cellpadding="0" style="background:#ffffff;border-radius:12px;overflow:hidden;box-shadow:0 4px 24px rgba(124,58,237,0.1);">
          <!-- Header -->
          <tr>
            <td style="background:linear-gradient(135deg,#7c3aed 0%,#5b21b6 100%);padding:40px 40px 32px;text-align:center;">
              <div style="display:inline-block;width:64px;height:64px;background:rgba(255,255,255,0.2);border-radius:16px;margin-bottom:16px;">
                <svg width="64" height="64" viewBox="0 0 64 64" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path d="M32 8L8 20v24l24 12 24-12V20L32 8z" stroke="white" stroke-width="3" fill="none"/>
                  <path d="M32 20v24M20 26l24 12M44 26L20 38" stroke="white" stroke-width="2.5" stroke-linecap="round"/>
                </svg>
              </div>
              <h1 style="margin:0;font-size:28px;font-weight:700;color:#ffffff;">Welcome to OVAV</h1>
              <p style="margin:12px 0 0;font-size:16px;color:rgba(255,255,255,0.85);">Your secure identity and secrets management platform</p>
            </td>
          </tr>

          <!-- Body -->
          <tr>
            <td style="padding:40px;">
              <p style="margin:0 0 24px;font-size:16px;line-height:1.6;color:#1f2937;">
                Hello ` + firstName + `,
              </p>
              <p style="margin:0 0 24px;font-size:16px;line-height:1.6;color:#1f2937;">
                Your OVAV account is ready. You're now part of a secure identity governance platform built for teams that take security seriously.
              </p>

              <div style="background:#f5f3ff;border-radius:8px;padding:24px;margin:24px 0;">
                <h3 style="margin:0 0 16px;font-size:16px;font-weight:600;color:#7c3aed;">Your account details</h3>
                <table role="presentation" width="100%" cellspacing="0" cellpadding="0">
                  <tr>
                    <td style="padding:8px 0;font-size:14px;color:#6b7280;">Username</td>
                    <td style="padding:8px 0;font-size:14px;font-weight:600;color:#1f2937;text-align:right;">` + username + `</td>
                  </tr>
                </table>
              </div>

              <p style="margin:0 0 32px;font-size:16px;line-height:1.6;color:#1f2937;">
                With OVAV you can manage TOTP authenticators, rotate secrets automatically, monitor authentication events, and maintain full audit compliance.
              </p>

              <div style="text-align:center;margin:32px 0;">
                <a href="` + loginURL + `" style="display:inline-block;background:linear-gradient(135deg,#7c3aed 0%,#5b21b6 100%);color:#ffffff;font-size:16px;font-weight:600;text-decoration:none;padding:16px 40px;border-radius:8px;">Access Your Dashboard</a>
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
