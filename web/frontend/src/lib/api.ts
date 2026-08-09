/**
 * API client for OVAV backend.
 * Auto-detects base URL from environment or defaults to localhost:8080.
 */

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

interface ApiOptions {
  method?: string;
  body?: unknown;
  headers?: Record<string, string>;
  token?: string;
}

class ApiError extends Error {
  status: number;
  detail: string;

  constructor(status: number, detail: string) {
    super(detail);
    this.status = status;
    this.detail = detail;
    this.name = "ApiError";
  }
}

async function request<T = unknown>(path: string, options: ApiOptions = {}): Promise<T> {
  const { method = "GET", body, headers = {}, token } = options;

  const requestHeaders: Record<string, string> = {
    "Content-Type": "application/json",
    ...headers,
  };

  if (token) {
    requestHeaders["Authorization"] = `Bearer ${token}`;
  } else if (typeof window !== "undefined") {
    const stored = localStorage.getItem("ovav_token");
    if (stored) {
      requestHeaders["Authorization"] = `Bearer ${stored}`;
    }
  }

  const url = `${API_BASE}${path}`;
  const fetchOptions: RequestInit = {
    method,
    headers: requestHeaders,
  };

  if (body && method !== "GET") {
    fetchOptions.body = JSON.stringify(body);
  }

  const response = await fetch(url, fetchOptions);

  if (!response.ok) {
    const errorBody = await response.json().catch(() => ({ detail: "Unknown error" }));
    throw new ApiError(response.status, errorBody.detail || "Request failed");
  }

  return response.json() as Promise<T>;
}

// --- Auth ---

export interface SessionUser {
  user_id: string;
  email: string;
  name: string | null;
  avatar_url: string | null;
}

export interface AuthResponse {
  access_token: string;
  user: SessionUser;
}

export const api = {

  // ── Users (v1) ───────────────────────────────────────────────────────────

  /** Get current user profile */
  async getMe(): Promise<SessionUser> {
    return request("/v1/users/me");
  },

  /** Update user profile */
  async updateMe(data: { name?: string; avatar_url?: string }): Promise<SessionUser> {
    return request("/v1/users/me", { method: "PATCH", body: data });
  },

  // ── API Keys (v1) ──────────────────────────────────────────────────────────

  /** List all API keys */
  async listApiKeys(): Promise<ApiKey[]> {
    return request("/v1/api-keys");
  },

  /** Create a new API key */
  async createApiKey(name: string, expiresInDays?: number): Promise<ApiKeyCreated> {
    return request("/v1/api-keys", { method: "POST", body: { name, expires_in_days: expiresInDays } });
  },

  /** Delete an API key */
  async deleteApiKey(keyId: string): Promise<void> {
    return request(`/v1/api-keys/${keyId}`, { method: "DELETE" });
  },

  /** Rotate an API key */
  async rotateApiKey(keyId: string): Promise<ApiKeyCreated> {
    return request(`/v1/api-keys/${keyId}/rotate`, { method: "POST" });
  },

  // ── Billing (v1) ──────────────────────────────────────────────────────────

  /** List all invoices */
  async listInvoices(): Promise<Invoice[]> {
    return request("/v1/billing/invoices");
  },

  /** Get current subscription */
  async getSubscription(): Promise<Subscription> {
    return request("/v1/billing/subscription");
  },

  /** Update subscription */
  async updateSubscription(data: { tier?: string; cancel?: boolean; reactivate?: boolean }): Promise<Subscription> {
    return request("/v1/billing/subscription", { method: "PATCH", body: data });
  },

  /** Create Stripe portal session */
  async createPortalSession(): Promise<{ url: string }> {
    return request("/v1/billing/portal", { method: "POST" });
  },

  // ── Download (v1) ──────────────────────────────────────────────────────────

  /** Get CLI download URLs */
  async getDownloadUrls(): Promise<DownloadUrls> {
    return request("/v1/download/cli");
  },

  /** Get version info */
  async getVersionInfo(): Promise<VersionInfo> {
    return request("/v1/download/version");
  },

  /** Detect platform */
  async detectPlatform(): Promise<{ platform: string; arch: string; recommended_download: string; version: string }> {
    return request("/v1/download/detect");
  },

  /** Request a magic link for passwordless login */
  async login(email: string): Promise<{ message: string; magic_link?: string; expires_in_minutes: number }> {
    return request("/auth/login", { method: "POST", body: { email } });
  },

  /** Register a new user */
  async register(email: string, name?: string): Promise<AuthResponse> {
    return request("/auth/register", { method: "POST", body: { email, name } });
  },

  /** Verify a magic link token */
  async verifyToken(token: string): Promise<AuthResponse> {
    return request(`/auth/verify?token=${encodeURIComponent(token)}`);
  },

  /** Get current session */
  async getSession(token?: string): Promise<SessionUser> {
    return request("/auth/session", { token });
  },

  /** Initiate OAuth flow */
  async oauthLogin(provider: "google" | "github"): Promise<{ url: string; provider: string }> {
    return request(`/auth/oauth/${provider}`);
  },

  /** OAuth callback */
  async oauthCallback(provider: "google" | "github", code: string): Promise<AuthResponse> {
    return request(`/auth/oauth/${provider}/callback?code=${encodeURIComponent(code)}`);
  },

  /** Save auth token to localStorage */
  saveToken(token: string): void {
    if (typeof window !== "undefined") {
      localStorage.setItem("ovav_token", token);
    }
  },

  /** Get stored auth token */
  getToken(): string | null {
    if (typeof window !== "undefined") {
      return localStorage.getItem("ovav_token");
    }
    return null;
  },

  /** Clear auth token */
  clearToken(): void {
    if (typeof window !== "undefined") {
      localStorage.removeItem("ovav_token");
    }
  },

  /** Check if user is authenticated */
  isAuthenticated(): boolean {
    return !!api.getToken();
  },
};

// ── Types for new endpoints ─────────────────────────────────────────────────────

export interface ApiKey {
  id: string;
  name: string;
  key_prefix: string;
  expires_at?: string;
  created_at: string;
  last_used_at?: string;
}

export interface ApiKeyCreated {
  id: string;
  name: string;
  key: string; // Full key shown only once!
  key_prefix: string;
  expires_at?: string;
  created_at: string;
}

export interface Invoice {
  id: string;
  amount: number;
  currency: string;
  status: string;
  created_at: string;
  paid_at?: string;
  invoice_url?: string;
}

export interface Subscription {
  tier: string;
  status: string;
  current_period_start: string;
  current_period_end?: string;
  cancel_at_period_end: boolean;
}

export interface PlatformDownload {
  platform: string;
  arch: string;
  url: string;
  checksum: string;
  size_mb: number;
}

export interface DownloadUrls {
  version: string;
  platforms: PlatformDownload[];
}

export interface VersionInfo {
  version: string;
  release_date: string;
  release_notes_url: string;
  minimum_os_version: Record<string, string>;
  downloads: PlatformDownload[];
}

export { ApiError };
export default api;
