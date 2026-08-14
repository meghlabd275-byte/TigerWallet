// Admin panel API helper. All calls go to the canonical wallet_api (Go) admin
// endpoints (proxied via /api/v1/* in dev/production). Reads the JWT bearer
// token from localStorage. Never returns fabricated data — on error throws.

const TOKEN_KEY = 'tigerwallet-admin-token';

export function getAdminToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
}

export function setAdminToken(token: string): void {
  try {
    localStorage.setItem(TOKEN_KEY, token);
  } catch {
    /* ignore */
  }
}

export function clearAdminToken(): void {
  try {
    localStorage.removeItem(TOKEN_KEY);
  } catch {
    /* ignore */
  }
}

export interface FetchOptions extends RequestInit {
  auth?: boolean;
}

export async function adminFetch<T = unknown>(
  path: string,
  options: FetchOptions = {}
): Promise<T> {
  const { auth = true, headers, ...rest } = options;
  const reqHeaders: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(headers as Record<string, string> | undefined),
  };
  if (auth) {
    const token = getAdminToken();
    if (token) reqHeaders['Authorization'] = `Bearer ${token}`;
  }
  const res = await fetch(path, { ...rest, headers: reqHeaders });
  if (!res.ok) {
    let msg = `Request failed (${res.status})`;
    try {
      const body = await res.json();
      if (body?.error) msg = body.error;
    } catch {
      /* non-JSON error body */
    }
    throw new Error(msg);
  }
  if (res.status === 204) return undefined as unknown as T;
  return (await res.json()) as T;
}
