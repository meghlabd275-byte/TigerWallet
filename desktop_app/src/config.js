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
