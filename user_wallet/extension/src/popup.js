// TigerWallet UserWallet extension popup.
// Talks to the canonical Go wallet-api backend (go/wallet_api, port 8443):
// REAL on-chain RPC, REAL BIP-39/32/44 derivation, REAL secp256k1 signing,
// AES-256-GCM encrypted-seed persistence (PostgreSQL + Redis). No stubs.

const API_BASE = 'http://localhost:8443/api/v1';

function getToken() {
  return new Promise((resolve) => {
    chrome.storage.local.get('token', (res) => resolve(res.token || null));
  });
}
function setToken(token) {
  return new Promise((resolve) => {
    chrome.storage.local.set(token ? { token } : { token: null }, resolve);
  });
}

async function api(path, { method = 'GET', body, auth = true } = {}) {
  const headers = { 'Content-Type': 'application/json', Accept: 'application/json' };
  if (auth) {
    const token = await getToken();
    if (token) headers['Authorization'] = `Bearer ${token}`;
  }
  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || `Request failed (${res.status})`);
  }
  return data;
}

let isRegister = false;

document.addEventListener('DOMContentLoaded', init);

async function init() {
  loadTheme();
  const token = await getToken();
  if (token) {
    showWallets();
  } else {
    showAuth();
  }
  bindEvents();
}

function bindEvents() {
  document.getElementById('toggleTheme').addEventListener('click', toggleTheme);
  document.getElementById('authSubmit').addEventListener('click', handleAuth);
  document.getElementById('authToggle').addEventListener('click', toggleAuthMode);
  document.getElementById('refreshBtn').addEventListener('click', loadWallets);
  document.getElementById('logoutBtn').addEventListener('click', handleLogout);

  document.getElementById('email').addEventListener('keydown', (e) => {
    if (e.key === 'Enter') document.getElementById('password').focus();
  });
  document.getElementById('password').addEventListener('keydown', (e) => {
    if (e.key === 'Enter') handleAuth();
  });
}

function toggleAuthMode() {
  isRegister = !isRegister;
  document.getElementById('authTitle').textContent = isRegister ? 'Create Account' : 'Login';
  document.getElementById('authSubmit').textContent = isRegister ? 'Register' : 'Login';
  document.getElementById('authToggle').textContent = isRegister
    ? 'Already have an account? Login'
    : "Don't have an account? Register";
  document.getElementById('usernameField').classList.toggle('hidden', !isRegister);
  hideError();
}

async function handleAuth() {
  hideError();
  const email = document.getElementById('email').value.trim();
  const password = document.getElementById('password').value;
  if (!email || password.length < 8) {
    showError('Enter a valid email and a password of at least 8 characters.');
    return;
  }
  const btn = document.getElementById('authSubmit');
  btn.disabled = true;
  btn.textContent = 'Please wait...';
  try {
    const path = isRegister ? '/auth/register' : '/auth/login';
    const body = isRegister
      ? { email, username: document.getElementById('username').value.trim() || email, password }
      : { email, password };
    const res = await api(path, { method: 'POST', body, auth: false });
    await setToken(res.token);
    showWallets();
  } catch (err) {
    showError(err.message);
    btn.disabled = false;
    btn.textContent = isRegister ? 'Register' : 'Login';
  }
}

async function handleLogout() {
  await setToken(null);
  showAuth();
}

function showAuth() {
  document.getElementById('authSection').classList.remove('hidden');
  document.getElementById('walletSection').classList.add('hidden');
  document.getElementById('password').value = '';
}

function showWallets() {
  document.getElementById('authSection').classList.add('hidden');
  document.getElementById('walletSection').classList.remove('hidden');
  loadWallets();
}

async function loadWallets() {
  const list = document.getElementById('walletList');
  list.innerHTML = '<div class="spinner">Loading...</div>';
  try {
    const { wallets } = await api('/wallets');
    if (!wallets || wallets.length === 0) {
      list.innerHTML = '<div class="wallet-label">No wallets yet.</div>';
      document.getElementById('totalUsd').textContent = '$0.00';
      return;
    }
    const balances = await Promise.all(
      wallets.map((w) =>
        api(`/public/balance?address=${w.address}&chain_id=${w.chain_id}`, { auth: false })
          .then((b) => ({ wallet: w, balance: b }))
          .catch(() => ({ wallet: w, balance: null }))
      )
    );
    list.innerHTML = balances
      .map(
        ({ wallet, balance }) => `
        <div class="wallet-item">
          <div class="wallet-label">${escapeHtml(wallet.label)} <span style="color:var(--text-secondary);font-weight:400">· Chain #${wallet.chain_id}</span></div>
          <div class="wallet-addr">${escapeHtml(wallet.address)}</div>
          ${balance ? `<div class="wallet-balance">${escapeHtml(balance.symbol)}: ${balance.balance_f.toFixed(6)} ($${balance.usd_value.toFixed(2)})</div>` : '<div class="wallet-balance" style="color:var(--text-secondary)">Balance unavailable</div>'}
        </div>`
      )
      .join('');
    const total = balances.reduce((sum, b) => sum + (b.balance ? b.balance.usd_value : 0), 0);
    document.getElementById('totalUsd').textContent = `$${total.toFixed(2)}`;
  } catch (err) {
    list.innerHTML = `<div class="wallet-label" style="color:var(--error)">${escapeHtml(err.message)}</div>`;
  }
}

function showError(msg) {
  const el = document.getElementById('authError');
  el.textContent = msg;
  el.classList.remove('hidden');
}
function hideError() {
  document.getElementById('authError').classList.add('hidden');
}

function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, (c) =>
    ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])
  );
}

// Theme
function loadTheme() {
  chrome.storage.local.get('theme', (res) => {
    const theme = res.theme || (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
    applyTheme(theme);
  });
}
function applyTheme(theme) {
  document.documentElement.setAttribute('data-theme', theme);
  document.getElementById('toggleTheme').textContent = theme === 'dark' ? '☀️' : '🌙';
}
function toggleTheme() {
  const current = document.documentElement.getAttribute('data-theme') || 'light';
  const next = current === 'dark' ? 'light' : 'dark';
  chrome.storage.local.set({ theme: next });
  applyTheme(next);
}
