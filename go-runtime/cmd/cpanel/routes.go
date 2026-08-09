// OVAV cPanel v5.0 — Route registration.
//
// All API routes registered here. Exact match + prefix match.
// Contract matches Python router.py (only endpoints the frontend uses).

package main

func registerRoutes() {
	// ── Core ───────────────────────────────────────────────────────────────
	addRoute("GET", "/", spaIngress)
	addRoute("GET", "/health", handleHealth)
	addRoute("GET", "/api/v1/health", handleHealth)
	addRoute("GET", "/api/v1/healthz", handleHealthz) // deep health — bypasses Access
	addRoute("GET", "/api/v1/status", handleStatus)

	// ── Static assets ──────────────────────────────────────────────────────
	// Prefix routes for static files (Vite build output)
	addPrefix("GET", "/assets/", serveAsset)
	addPrefix("GET", "/css/", serveAsset)

	// ── Validators ─────────────────────────────────────────────────────────
	addRoute("GET", "/api/v1/validators", handleValidatorsList)
	addRoute("POST", "/api/v1/validators/run", handleValidatorsRun)
	addPrefix("GET", "/api/v1/validators/status/", handleValidatorsStatus)

	// ── Security ───────────────────────────────────────────────────────────
	addRoute("GET", "/api/v1/security/audit-log", handleAuditLog)
	addRoute("DELETE", "/api/v1/security/canary-alarms", handleClearAlarms)
	addRoute("GET", "/api/v1/security/living-integrity", handleLivingIntegrity)
	addRoute("GET", "/api/v1/security/defense/status", handleDefenseStatus)
	addRoute("POST", "/api/v1/security/defense/scan", handleDefenseScan)

	// ── Memory ─────────────────────────────────────────────────────────────
	addRoute("GET", "/api/v1/memory/status", handleMemoryStatus)
	addRoute("GET", "/api/v1/memory/ledger", handleLedger)
	addRoute("GET", "/api/v1/memory/beliefs", handleBeliefs)
	// ── Agents ─────────────────────────────────────────────────────────────
	addRoute("GET", "/api/v1/agents", handleAgentList)
	addRoute("GET", "/api/v1/agents/topology", handleTopology)
	addRoute("GET", "/api/v1/agents/profiles", handleServiceProfiles)
	addRoute("GET", "/api/v1/agents/permissions", handlePermissions)

	// ── Profiles ───────────────────────────────────────────────────────────
	addRoute("GET", "/api/v1/profiles", handleProfileList)
	addRoute("POST", "/api/v1/profiles/apply", handleProfileApply)

	// ── Governor ───────────────────────────────────────────────────────────
	addRoute("GET", "/api/v1/governor/health", handleGovernorHealth)
	addRoute("GET", "/api/v1/governor/decisions", handleGovernorDecisions)
	addRoute("GET", "/api/v1/governor/tasks", handleGovernorTasks)
	addRoute("POST", "/api/v1/governor/tasks", handleGovernorTasks)
	addRoute("GET", "/api/v1/governor/counts", handleGovernorCounts) // OVAV: real counts
	addRoute("POST", "/api/v1/governor/trust", handleGovernorTrust)

	// ── System ─────────────────────────────────────────────────────────────
	addRoute("GET", "/api/v1/system/health", handleSystemHealth)
	addPrefix("GET", "/api/v1/system/registry/", handleRegistry)
	addRoute("GET", "/api/v1/system/config", handleConfig)
	addRoute("GET", "/api/v1/system/sbom", handleSBOM)
	addRoute("GET", "/api/v1/system/kc", handleKCStatus)
	addRoute("GET", "/api/v1/system/operations", handleOperations)

	// ── Economy ────────────────────────────────────────────────────────────
	addRoute("GET", "/api/v1/economy", handleEconomyDetail)

	// ── Git ────────────────────────────────────────────────────────────────
	addRoute("GET", "/api/v1/git/branches", handleGitBranches)
	addRoute("GET", "/api/v1/git/log", handleGitLog)
	addRoute("GET", "/api/v1/git/worktrees", handleGitWorktrees)
	addRoute("POST", "/api/v1/git/fetch", handleGitFetch)

	// ── Auth (v5.0 + user accounts) ───────────────────────────────────────
	addRoute("POST", "/api/v1/auth/login", handleAuthLogin)      // token-based (CLI)
	addRoute("POST", "/api/v1/auth/user-login", handleUserLogin) // email+password
	addRoute("POST", "/api/v1/auth/register", handleRegister)    // new user signup
	addRoute("GET", "/api/v1/auth/session", handleAuthSession)
	addRoute("GET", "/api/v1/auth/me", handleMe) // current user info
	addRoute("GET", "/api/v1/auth/config", handleAuthConfig)
	addPrefix("GET", "/api/v1/auth/oauth/", handleOAuthCallback)      // GOV-010: GET for Google redirect
	addRoute("GET", "/api/v1/auth/oauth/google", handleOAuthCallback) // GOV-010: exact match fallback
	addPrefix("POST", "/api/v1/auth/oauth/", handleOAuthCallback)
	addRoute("POST", "/api/v1/auth/cli-verify", handleCLIVerify)

	// ── Admin Login (GOV-010) — branded OVAV login page ──────────────────
	addRoute("GET", "/login", handleLoginPage)
	addRoute("GET", "/login-portal", handleLoginPortal) // standalone web login page (no CF Access)
	addRoute("GET", "/login/verify", handleTOTPPage)
	addRoute("POST", "/api/v1/auth/admin-login", handleAdminLogin)
	addRoute("GET", "/api/v1/auth/oauth-url", handleOAuthURL)
	addRoute("GET", "/api/v1/auth/login-challenge", handleLoginChallenge)
	addRoute("GET", "/api/v1/auth/login-challenge-web", handleLoginChallengeWeb) // CLI web login
	addRoute("GET", "/api/v1/auth/login-status", handleLoginStatus)              // poll login completion
	addRoute("POST", "/api/v1/auth/totp/setup", handleTOTPSetup)
	addRoute("POST", "/api/v1/auth/totp/verify", handleTOTPVerify)

	// ── Auth Analytics (native OVAV Systems audit) ─────────────────────────
	addRoute("GET", "/api/v1/auth/analytics", handleAuthAnalytics)   // GET /api/v1/auth/analytics?days=7
	addRoute("GET", "/api/v1/auth/audit-log", handleAuthAuditLog)    // GET /api/v1/auth/audit-log?email=&ip=&action=&status=&limit=50&offset=0
	addRoute("POST", "/api/v1/auth/portal-event", handlePortalEvent) // login portal → cPanel analytics

	// ── Product (v5.2 GOV-007) ─────────────────────────────────────────────
	addRoute("GET", "/api/v1/product/version", handleProductVersion)
	addRoute("POST", "/api/v1/product/update", handleProductUpdate)

	// ── Product Sync (GOV-009) — Protected: auth + rate-limited ──────────
	addRoute("GET", "/api/v1/product/sync/status", syncProtect(handleProductSyncStatus))
	addRoute("POST", "/api/v1/product/sync/stage", syncProtect(handleProductSyncStage))
	addRoute("POST", "/api/v1/product/sync/push", syncProtect(handleProductSyncPush))
	addRoute("POST", "/api/v1/product/sync/apply", syncProtect(handleProductSyncApply))

	// ── SSE Events (v5.2 GOV-007) ──────────────────────────────────────────
	addRoute("GET", "/api/v1/events", handleEventsSSE)

	// ── Vault Sync (Phase 6.2) ──────────────────────────────────────────────
	// Health check — public
	addRoute("GET", "/api/v1/vault/health", handleVaultHealth)
	// Auth — public (rate-limited)
	addRoute("POST", "/api/v1/vault/auth", handleVaultAuth)

	// ── Vault Server (per-user secrets) ────────────────────────────────────
	// Vault login — email+password → JWT + vault key set
	addRoute("POST", "/api/v1/vault/login", handleVaultUserLogin)
	// Secrets CRUD — authenticated (JWT with user_id)
	addRoute("GET", "/api/v1/vault/secrets", handleVaultSecretsList)
	addRoute("GET", "/api/v1/vault/secrets/:name", handleVaultSecretGet)
	addRoute("POST", "/api/v1/vault/secrets", handleVaultSecretAdd)
	addRoute("DELETE", "/api/v1/vault/secrets/:name", handleVaultSecretDelete)

	// ── Portal (user-facing: usage, API keys) ──────────────────────────────
	// All require user JWT auth via portalAuth middleware
	addRoute("GET", "/api/v1/portal/me", portalAuth(handlePortalMe))
	addRoute("GET", "/api/v1/portal/usage", portalAuth(handlePortalUsage))
	addRoute("GET", "/api/v1/portal/api-keys", portalAuth(handleAPIKeysList))
	addRoute("POST", "/api/v1/portal/api-keys", portalAuth(handleAPIKeyCreate))
	addRoute("DELETE", "/api/v1/portal/api-keys/:id", portalAuth(handleAPIKeyDelete))

	// ── Vault blob sync (Phase 6.2 — legacy, kept for compatibility) ───────
	// Blobs — authenticated
	addRoute("GET", "/api/v1/vault/blobs", vaultAuthenticate(handleVaultBlobs))
	// Upload — authenticated
	addRoute("POST", "/api/v1/vault/upload", vaultAuthenticate(handleVaultUpload))
	// Download blob by device ID — authenticated
	addRoute("GET", "/api/v1/vault/blob/:deviceID", vaultAuthenticate(handleVaultGetBlob))
	// Delete blob — authenticated
	addRoute("DELETE", "/api/v1/vault/blob/:deviceID", vaultAuthenticate(handleVaultDeleteBlob))
}

// ── Additional handler aliases for exact path matching ───────────────────────

func init() {
	// Ensure static asset routes handle path variations
	// The router handles prefix matching for /assets/* and /css/*
}

// force rebuild 1782749473
