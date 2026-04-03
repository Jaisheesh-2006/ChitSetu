const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

/* ── Token helpers ── */

export function getAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("chitsetu-access-token");
}

export function getRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("chitsetu-refresh-token");
}

export function setTokens(access: string, refresh: string) {
  localStorage.setItem("chitsetu-access-token", access);
  localStorage.setItem("chitsetu-refresh-token", refresh);
}

export function clearTokens() {
  localStorage.removeItem("chitsetu-access-token");
  localStorage.removeItem("chitsetu-refresh-token");
}

export function getCurrentUserId(): string | null {
  const token = getAccessToken();
  if (!token) return null;
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    return payload.user_id || payload.sub || null;
  } catch {
    return null;
  }
}

/* ── Core fetch wrapper ── */

interface FetchOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
  auth?: boolean;
}

async function fetchAPI<T = unknown>(path: string, options: FetchOptions = {}): Promise<T> {
  const { body, auth = true, ...rest } = options;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((rest.headers as Record<string, string>) || {}),
  };

  if (auth) {
    const token = getAccessToken();
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...rest,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  // Attempt token refresh on 401
  if (res.status === 401 && auth) {
    const refreshed = await tryRefresh();
    if (refreshed) {
      headers["Authorization"] = `Bearer ${getAccessToken()}`;
      const retryRes = await fetch(`${API_BASE}${path}`, {
        ...rest,
        headers,
        body: body ? JSON.stringify(body) : undefined,
      });
      if (!retryRes.ok) {
        const err = await retryRes.json().catch(() => ({ error: "Request failed" }));
        throw new Error(err.error || `Request failed with status ${retryRes.status}`);
      }
      return retryRes.json();
    }
    clearTokens();
    if (typeof window !== "undefined") {
      window.location.href = "/";
    }
    throw new Error("Session expired");
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Request failed" }));
    throw new Error(err.error || `Request failed with status ${res.status}`);
  }

  return res.json();
}

async function tryRefresh(): Promise<boolean> {
  const refreshToken = getRefreshToken();
  if (!refreshToken) return false;

  try {
    const res = await fetch(`${API_BASE}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!res.ok) return false;

    const data = await res.json();
    setTokens(data.access_token, data.refresh_token);
    return true;
  } catch {
    return false;
  }
}

/* ── Auth API ── */

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in_sec: number;
}

export async function authRegister(email: string, password: string): Promise<TokenPair> {
  return fetchAPI<TokenPair>("/auth/register", {
    method: "POST",
    body: { email, password },
    auth: false,
  });
}

export async function authLogin(email: string, password: string): Promise<TokenPair> {
  return fetchAPI<TokenPair>("/auth/login", {
    method: "POST",
    body: { email, password },
    auth: false,
  });
}

export async function forgotPassword(email: string): Promise<{ token: string, message: string }> {
  return fetchAPI<{ token: string, message: string }>("/auth/forgot-password", {
    method: "POST",
    body: { email },
    auth: false,
  });
}

export async function resetPassword(token: string, newPassword: string): Promise<{ message: string }> {
  return fetchAPI<{ message: string }>("/auth/reset-password", {
    method: "POST",
    body: { token, new_password: newPassword },
    auth: false,
  });
}