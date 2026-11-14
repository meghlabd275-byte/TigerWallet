// Chrome MV3 exposes the `chrome` namespace; Firefox exposes `browser`.
// Without this shim every browser.* call below throws ReferenceError in Chrome.
const ext = typeof browser !== 'undefined' ? browser : chrome;

const DEFAULT_WEB_URL = 'http://localhost:8107';

async function getWebUrl() {
  const { pp_web_url: url } = await ext.storage.local.get('pp_web_url');
  return (url || DEFAULT_WEB_URL).replace(/\/$/, '');
}

async function openTab(path) {
  const base = await getWebUrl();
  ext.tabs.create({ url: base + path });
}

// MV3 blocks a cross-origin fetch unless the host is granted. Localhost is in
// host_permissions; anything else has to be requested at runtime.
async function ensureAccess(base) {
  let origin;
  try {
    origin = new URL(base).origin;
  } catch (e) {
    return false;
  }
  if (origin.startsWith('http://localhost') || origin.startsWith('http://127.0.0.1')) return true;
  try {
    const res = await ext.runtime.sendMessage({ action: 'ensureHostPermission', origin });
    return !!(res && res.granted);
  } catch (e) {
    return false;
  }
}

function setStatus(state, text) {
  const el = document.getElementById('backendStatus');
  el.innerHTML = '';
  const dot = document.createElement('span');
  dot.className = 'status-dot ' + state;
  el.appendChild(dot);
  el.appendChild(document.createTextNode(text));
}

async function refresh() {
  const base = await getWebUrl();
  document.getElementById('webUrl').value = base;

  if (!(await ensureAccess(base))) {
    setStatus('err', 'permission needed');
    return;
  }

  try {
    const health = await fetch(base + '/api/v1/health');
    setStatus(health.ok ? 'ok' : 'err', health.ok ? 'online' : 'HTTP ' + health.status);
  } catch (e) {
    setStatus('err', 'unreachable');
  }

  // Both endpoints are public read-only on the ProjectParty backend.
  try {
    const res = await fetch(base + '/api/v1/tokens');
    if (!res.ok) throw new Error('HTTP ' + res.status);
    const data = await res.json();
    const tokens = Array.isArray(data) ? data : data.tokens || [];
    document.getElementById('listedTokens').textContent = String(tokens.length);
  } catch (e) {
    document.getElementById('listedTokens').textContent = '—';
  }

  try {
    const res = await fetch(base + '/api/v1/launchpad/pools');
    if (!res.ok) throw new Error('HTTP ' + res.status);
    const data = await res.json();
    const pools = Array.isArray(data) ? data : data.pools || [];
    document.getElementById('launchpadPools').textContent = String(pools.length);
  } catch (e) {
    document.getElementById('launchpadPools').textContent = '—';
  }
}

document.addEventListener('DOMContentLoaded', async () => {
  document.getElementById('openDashboard').addEventListener('click', () => openTab('/dashboard'));
  document.getElementById('openTokens').addEventListener('click', () => openTab('/tokens'));
  document.getElementById('openLaunchpad').addEventListener('click', () => openTab('/launchpad'));
  document.getElementById('openSubmit').addEventListener('click', () => openTab('/submit'));

  document.getElementById('saveUrl').addEventListener('click', async () => {
    const value = document.getElementById('webUrl').value.trim().replace(/\/$/, '');
    if (!value) return;
    await ext.storage.local.set({ pp_web_url: value });
    refresh();
  });

  document.getElementById('toggleTheme').addEventListener('click', async () => {
    const { theme } = await ext.storage.local.get('theme');
    const next = theme === 'dark' ? 'light' : 'dark';
    await ext.storage.local.set({ theme: next });
    document.documentElement.setAttribute('data-theme', next);
  });

  const { theme } = await ext.storage.local.get('theme');
  if (theme) document.documentElement.setAttribute('data-theme', theme);

  refresh();
});
