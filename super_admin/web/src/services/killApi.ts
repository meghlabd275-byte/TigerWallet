/**
 * TigerWallet Super Admin - Kill-Switch API client
 *
 * Talks to the kill_switch control plane (:8469) via the Vite dev proxy
 * (/kill-api -> http://localhost:8469). In production set VITE_KILL_API_URL.
 * Uses the same SuperAdmin JWT stored by api.ts under `super_admin_token` —
 * kill_switch verifies it with the shared control-plane JWT secret and only
 * accepts role "superadmin".
 *
 * Scopes: global (whole platform) | client (one WL client) | product (one
 * product of one client) | fetcher (one fetcher of one product).
 * Halts are durable (PostgreSQL), propagate over Redis in under a second,
 * and are enforced by license_service heartbeats (fail-closed).
 */

export const KILL_API_URL: string =
  (typeof process !== 'undefined' && (process as any).env?.VITE_KILL_API_URL) ||
  '/kill-api';

export type KillScopeType = 'global' | 'client' | 'product' | 'fetcher';

export interface KillScope {
  scope_type: KillScopeType;
  wl_client_id?: string;
  product?: string;
  fetcher?: string;
  reason?: string;
}

export interface ActiveHalt {
  scope_type: KillScopeType;
  wl_client_id: string | null;
  product: string;
  fetcher: string;
  reason: string;
  since: string;
}

export interface KillEvent {
  action: 'halt' | 'resume';
  scope_type: KillScopeType;
  wl_client_id: string | null;
  product: string;
  fetcher: string;
  reason: string;
  issued_by: string | null;
  at: string;
}

function authHeaders(): HeadersInit {
  const token = localStorage.getItem('super_admin_token');
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
}

async function req<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${KILL_API_URL}${path}`, {
    ...init,
    headers: { ...authHeaders(), ...(init.headers || {}) },
  });
  if (!res.ok) {
    let msg = `${res.status} ${res.statusText}`;
    try {
      const body = await res.json();
      if (body && body.error) msg = body.error;
    } catch {
      /* non-JSON error body */
    }
    throw new Error(msg);
  }
  return res.json() as Promise<T>;
}

export const killApi = {
  halt(scope: KillScope): Promise<{ halted: boolean }> {
    return req('/api/v1/kill/halt', { method: 'POST', body: JSON.stringify(scope) });
  },
  resume(scope: KillScope): Promise<{ halted: boolean }> {
    return req('/api/v1/kill/resume', { method: 'POST', body: JSON.stringify(scope) });
  },
  state(): Promise<{ halts: ActiveHalt[] }> {
    return req('/api/v1/kill/state');
  },
  audit(): Promise<{ events: KillEvent[] }> {
    return req('/api/v1/kill/audit');
  },
};

export default killApi;
