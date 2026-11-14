// Chrome MV3 exposes the `chrome` namespace; Firefox exposes `browser`.
// Without this shim every browser.* call below throws ReferenceError in Chrome.
const ext = typeof browser !== 'undefined' ? browser : chrome;

const DEFAULT_WEB_URL = 'http://localhost:8472';

async function getWebUrl() {
  const { bots_web_url: url } = await ext.storage.local.get('bots_web_url');
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

  const { bots_jwt: token } = await ext.storage.local.get('bots_jwt');
  const headers = token ? { Authorization: 'Bearer ' + token } : {};

  try {
    const health = await fetch(base + '/api/v1/health', { headers });
    setStatus(health.ok ? 'ok' : 'err', health.ok ? 'online' : 'HTTP ' + health.status);
  } catch (e) {
    setStatus('err', 'unreachable');
  }

  if (!token) {
    document.getElementById('runningBots').textContent = 'sign in';
    document.getElementById('totalBots').textContent = 'sign in';
    return;
  }

  try {
    const res = await fetch(base + '/api/v1/bots', { headers });
    if (!res.ok) throw new Error('HTTP ' + res.status);
    const data = await res.json();
    const bots = Array.isArray(data) ? data : data.bots || [];
    document.getElementById('totalBots').textContent = String(bots.length);
    document.getElementById('runningBots').textContent = String(
      bots.filter((b) => b.status === 'running').length
    );
  } catch (e) {
    document.getElementById('runningBots').textContent = '—';
    document.getElementById('totalBots').textContent = '—';
  }
}

document.addEventListener('DOMContentLoaded', async () => {
  document.getElementById('openDashboard').addEventListener('click', () => openTab('/dashboard'));
  document.getElementById('openBots').addEventListener('click', () => openTab('/bots'));
  document.getElementById('openSubscriptions').addEventListener('click', () => openTab('/subscriptions'));
  document.getElementById('openFees').addEventListener('click', () => openTab('/fees'));

  document.getElementById('saveUrl').addEventListener('click', async () => {
    const value = document.getElementById('webUrl').value.trim().replace(/\/$/, '');
    if (!value) return;
    await ext.storage.local.set({ bots_web_url: value });
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
