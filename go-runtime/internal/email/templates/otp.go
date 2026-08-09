package templates

// OTP returns the HTML for a one-time password verification email.
func OTP(data map[string]interface{}) string {
	otpCode := ""
	if v, ok := data["otp_code"].(string); ok {
		otpCode = v
	}

	username := ""
	if v, ok := data["username"].(string); ok {
		username = v
	}

	purpose := "Verify your identity"
	if v, ok := data["purpose"].(string); ok {
		purpose = v
	}

	expiresIn := "10 minutes"
	if v, ok := data["expires_in"].(string); ok {
		expiresIn = v
	}

	ipAddress := ""
	if v, ok := data["ip_address"].(string); ok {
		ipAddress = v
	}

	location := ""
	if v, ok := data["location"].(string); ok {
		location = v
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Your Verification Code - OVAV</title>
</head>
<body style="margin:0;padding:0;background:#f5f3ff;font-family:'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
  <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#f5f3ff;">
    <tr>
      <td align="center" style="padding:40px 20px;">
        <table role="presentation" width="100%" max-width="560" cellspacing="0" cellpadding="0" style="background:#ffffff;border-radius:12px;overflow:hidden;box-shadow:0 4px 24px rgba(124,58,237,0.1);">
          <!-- Header -->
          <tr>
            <td style="background:linear-gradient(135deg,#7c3aed 0%,#5b21b6 100%);padding:40px 40px 32px;text-align:center;">
              <div style="display:inline-block;width:64px;height:64px;background:rgba(255,255,255,0.2);border-radius:16px;margin-bottom:16px;">
                <svg width="64" height="64" viewBox="0 0 64 64" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <rect x="12" y="24" width="40" height="32" rx="4" stroke="white" stroke-width="3" fill="none"/>
                  <path d="M20 24V16a12 12 0 0124 0v8" stroke="white" stroke-width="3" stroke-linecap="round"/>
                  <circle cx="32" cy="40" r="4" fill="white"/>
                  <path d="M32 44v6" stroke="white" stroke-width="3" stroke-linecap="round"/>
                </svg>
              </div>
              <h1 style="margin:0;font-size:24px;font-weight:700;color:#ffffff;">Verification Code</h1>
              <p style="margin:8px 0 0;font-size:14px;color:rgba(255,255,255,0.85);">` + purpose + `</p>
            </td>
          </tr>

          <!-- Body -->
          <tr>
            <td style="padding:40px;text-align:center;">
              <p style="margin:0 0 24px;font-size:15px;color:#6b7280;">
                Enter this code to complete your verification. This code expires in <strong style="color:#1f2937;">` + expiresIn + `</strong>.
              </p>

              <!-- OTP Code Display -->
              <div style="background:#f5f3ff;border:2px dashed #7c3aed;border-radius:12px;padding:32px;margin:0 auto 24px;max-width:320px;">
                <p style="margin:0;font-size:48px;font-weight:700;color:#7c3aed;letter-spacing:12px;text-align:center;user-select:all;">` + otpCode + `</p>
              </div>

              <p style="margin:0 0 24px;font-size:14px;color:#6b7280;">
                Code requested by <strong style="color:#1f2937;">` + username + `</strong>
              </p>

              <!-- Context Info -->
              <div style="background:#f9fafb;border-radius:8px;padding:16px;margin:24px 0;text-align:left;">
                <table role="presentation" width="100%" cellspacing="0" cellpadding="0">
                  <tr>
                    <td style="padding:6px 0;font-size:13px;color:#6b7280;">IP Address</td>
                    <td style="padding:6px 0;font-size:13px;font-weight:500;color:#1f2937;text-align:right;">` + ipAddress + `</td>
                  </tr>
                  <tr>
                    <td style="padding:6px 0;font-size:13px;color:#6b7280;">Location</td>
                    <td style="padding:6px 0;font-size:13px;font-weight:500;color:#1f2937;text-align:right;">` + location + `</td>
                  </tr>
                </table>
              </div>

              <p style="margin:0;font-size:13px;color:#9ca3af;text-align:center;">
                If you didn't request this code, you can safely ignore this email. It was not triggered by you.
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
