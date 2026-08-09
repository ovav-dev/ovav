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

export { ApiError };
export default api;
