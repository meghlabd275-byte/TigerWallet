// TigerWallet Desktop - shared runtime configuration.
//
// The backend base URL is user-configurable (Settings -> Backend Server) and
// persisted in localStorage so the desktop app can point at any self-hosted
// wallet_api deployment. Nothing here is environment-locked.

const TW_DEFAULT_API_BASE = 'http://localhost:8443';
const TW_API_BASE_KEY = 'tw_api_base';

// Origin of the wallet_api backend, e.g. http://localhost:8443 (no trailing slash).
function twApiOrigin() {
    const saved = (localStorage.getItem(TW_API_BASE_KEY) || TW_DEFAULT_API_BASE).trim();
    return saved.replace(/\/+$/, '');
}

// Versioned API base, e.g. http://localhost:8443/api/v1.
function twApiBase() {
    return twApiOrigin() + '/api/v1';
}

// WebSocket live-feed URL derived from the configured origin.
function twWsUrl() {
    return twApiOrigin().replace(/^http/i, 'ws') + '/api/v1/ws';
}

// Persist a new backend origin. Accepts with or without the /api/v1 suffix.
function twSetApiBase(value) {
    const v = (value || '').trim().replace(/\/+$/, '').replace(/\/api\/v1$/, '');
    if (!v) return false;
    localStorage.setItem(TW_API_BASE_KEY, v);
    return true;
}

// ==================== Backend authentication (JWT) ====================
//
// UserWallet has NO mandatory registration: the desktop app provisions a
// guest account on the canonical wallet_api backend (POST /api/v1/auth/guest)
// bound to a stable per-install device id, then uses the returned JWT as a
// Bearer token on every protected request. Users may optionally attach an
// email/password account later (Settings) via /auth/register + /auth/login.
// Fail-closed: no request is ever sent without a valid token.

const TW_TOKEN_KEY = 'tw_auth_token';
const TW_DEVICE_ID_KEY = 'tw_device_id';

function twAuthToken() {
    return localStorage.getItem(TW_TOKEN_KEY) || '';
}

function twSetAuthToken(token) {
    if (token) localStorage.setItem(TW_TOKEN_KEY, token);
    else localStorage.removeItem(TW_TOKEN_KEY);
}

// Stable random device id persisted per install (used for guest accounts).
function twDeviceId() {
    let id = localStorage.getItem(TW_DEVICE_ID_KEY);
    if (!id) {
        id = (crypto.randomUUID ? crypto.randomUUID()
            : 'dev-' + Date.now() + '-' + Math.random().toString(36).slice(2));
        localStorage.setItem(TW_DEVICE_ID_KEY, id);
    }
    return id;
}

let twAuthBootstrapPromise = null;

// Obtain a guest JWT from the backend. Single-flighted so concurrent first
// requests do not provision multiple guest accounts.
async function twBootstrapGuestAuth() {
    if (!twAuthBootstrapPromise) {
        twAuthBootstrapPromise = (async () => {
            const res = await fetch(`${twApiBase()}/auth/guest`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ device_id: twDeviceId() })
            });
            if (!res.ok) throw new Error(`authentication failed (HTTP ${res.status}) — backend unreachable`);
            const data = await res.json();
            if (!data || !data.token) throw new Error('authentication failed — backend returned no token');
            twSetAuthToken(data.token);
            return data.token;
        })().finally(() => { twAuthBootstrapPromise = null; });
    }
    return twAuthBootstrapPromise;
}

// Ensure a valid token exists, bootstrapping a guest account if needed.
async function twEnsureAuth() {
    if (twAuthToken()) return twAuthToken();
    return twBootstrapGuestAuth();
}

// fetch() wrapper for every wallet_api call: injects the Bearer token,
// bootstraps guest auth when missing, and retries exactly once on 401
// (expired token) with a fresh guest token. Never sends unauthenticated
// requests to protected routes; throws on persistent auth failure.
// For GET requests, if auth bootstrap fails (backend down), fall back to
// an unauthenticated attempt so public routes (/chains, /price, /gas,
// /dapps, /defi/protocols, /tokens/registry, /terminal/*, /security/check-*)
// still work offline.
async function twFetch(url, opts = {}) {
    const noAuth = opts.auth === false;
    const method = (opts.method || 'GET').toUpperCase();
    const doFetch = async () => {
        const headers = Object.assign({}, opts.headers || {});
        if (!noAuth) {
            const token = await twEnsureAuth();
            headers['Authorization'] = `Bearer ${token}`;
        }
        const o = Object.assign({}, opts, { headers });
        delete o.auth;
        return fetch(url, o);
    };
    let res;
    try {
        res = await doFetch();
    } catch (authErr) {
        // Auth bootstrap failed (backend unreachable). For GET requests,
        // retry without auth so public read-only routes still work.
        if (method === 'GET' && !noAuth) {
            const o = Object.assign({}, opts);
            delete o.auth;
            delete o.headers?.Authorization;
            res = await fetch(url, o);
        } else {
            throw authErr;
        }
    }
    if (!noAuth && res.status === 401) {
        twSetAuthToken('');
        try {
            await twBootstrapGuestAuth();
            res = await doFetch();
        } catch (_) {
            // Re-auth failed; return the 401 so caller can handle it.
        }
    }
    return res;
}
