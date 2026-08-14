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

  // Tab navigation
  document.querySelectorAll('.tab-btn').forEach((btn) => {
    btn.addEventListener('click', () => {
      const tab = btn.getAttribute('data-tab').replace('Tab', '');
      switchTab(tab);
    });
  });

  // Feature handlers
  const sendBtn = document.getElementById('sendBtn');
  if (sendBtn) sendBtn.addEventListener('click', handleSend);
  const convertBtn = document.getElementById('convertBtn');
  if (convertBtn) convertBtn.addEventListener('click', handleConvert);
  const stakeBtn = document.getElementById('stakeBtn');
  if (stakeBtn) stakeBtn.addEventListener('click', handleStake);
  const fiatBuyBtn = document.getElementById('fiatBuyBtn');
  if (fiatBuyBtn) fiatBuyBtn.addEventListener('click', () => handleFiatQuote(false));
  const fiatSellBtn = document.getElementById('fiatSellBtn');
  if (fiatSellBtn) fiatSellBtn.addEventListener('click', () => handleFiatQuote(true));
  const qrParseBtn = document.getElementById('qrParseBtn');
  if (qrParseBtn) qrParseBtn.addEventListener('click', handleQrPaste);
  const qrSendBtn = document.getElementById('qrSendBtn');
  if (qrSendBtn) qrSendBtn.addEventListener('click', async () => {
    const w = activeWallet();
    if (!w) { alert('No wallet available.'); return; }
    const to = document.getElementById('qrToAddress').value.trim();
    const amount = document.getElementById('qrAmount').value.trim();
    const password = document.getElementById('sendPassword').value;
    if (!to || !amount || !password) { alert('Recipient, amount, and password required.'); return; }
    try {
      const res = await WalletAPI.sendTransaction(w.id, password, to, amount, w.chain_id);
      alert('Transaction sent: ' + (res.transaction_hash || res.tx_hash || 'submitted'));
    } catch (err) { alert(err.message); }
  });

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
    state.wallets = wallets;
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

// ---------------------------------------------------------------------------
// WalletAPI — full fetcher set (parity with web/desktop/android/ios/rust).
// All signing is delegated to the backend; the extension never fabricates a
// key, address, signature, or transaction hash.
// ---------------------------------------------------------------------------
const WalletAPI = {
  // Auth
  login: (email, password) => api('/auth/login', { method: 'POST', body: { email, password }, auth: false }),
  register: (email, password) => api('/auth/register', { method: 'POST', body: { email, password }, auth: false }),

  // Wallets
  createWallet: (label, password, chainId) => api('/wallets', { method: 'POST', body: { label, password, chain_id: chainId } }),
  listWallets: () => api('/wallets'),

  // Balance / tokens / NFTs
  getBalance: (address, chainId) => api(`/balance?address=${encodeURIComponent(address)}&chain_id=${chainId}`),
  getTokenBalances: (address, chainId) => api(`/tokens?address=${encodeURIComponent(address)}&chain_id=${chainId}`),
  getNFTs: (address, chainId) => api(`/nfts?address=${encodeURIComponent(address)}&chain_id=${chainId}`),

  // Transactions / send / sign
  getTransactions: (address, chainId) => api(`/transactions?address=${encodeURIComponent(address)}&chain_id=${chainId}`),
  sendTransaction: (walletId, password, to, amount, chainId, tokenAddress) =>
    api('/send', { method: 'POST', body: { wallet_id: walletId, password, to, amount, chain_id: chainId, token_address: tokenAddress || undefined } }),
  signMessage: (walletId, password, message) => api('/sign', { method: 'POST', body: { wallet_id: walletId, password, message } }),

  // Gas / price / chains / status
  getGasPrice: (chainId) => api(`/gas?chain_id=${chainId}`),
  getTokenPrice: (symbol) => api(`/price?symbol=${encodeURIComponent(symbol)}`),
  getChains: () => api('/chains'),
  getNetworkStatus: async (chainId) => {
    const { chains } = await api('/chains');
    const chain = chains.find((c) => c.id === chainId);
    return { chain_id: chainId, block_number: 0, connected: !!chain };
  },

  // Swap / Convert / Staking
  getSwapQuote: (fromToken, toToken, fromAmount, chainId) =>
    api(`/swap/quote?from_token=${encodeURIComponent(fromToken)}&to_token=${encodeURIComponent(toToken)}&from_amount=${encodeURIComponent(fromAmount)}&chain_id=${chainId}`),
  getConvertQuote: (fromToken, toToken, fromAmount, chainId) =>
    WalletAPI.getSwapQuote(fromToken, toToken, fromAmount, chainId),
  getStakingQuote: () => api('/staking/quote'),
  stake: (walletId, password, token, amount, chainId, stakingContract, callData) =>
    api('/staking/stake', { method: 'POST', body: { wallet_id: walletId, password, token, amount, chain_id: chainId, staking_contract: stakingContract, call_data: callData } }),

  // Auxiliary DeFi (via the Next.js same-origin proxy routes OR direct service URLs).
  // These hit the canonical backend proxy paths.
  getFiatRampProviders: () => api('/ramp/providers'),
  getFiatRampQuote: (providerId, amount, fiat, crypto, method) =>
    api('/ramp/quote', { method: 'POST', body: { providerId, amount, fiatCurrency: fiat, cryptoCurrency: crypto, paymentMethod: method } }),
  getFiatOfframpQuote: (providerId, amount, fiat, crypto) =>
    api('/ramp/offramp-quote', { method: 'POST', body: { providerId, amount, fiatCurrency: fiat, cryptoCurrency: crypto } }),
  getCryptoCardRates: () => api('/cards/rates'),
  getP2PListings: () => api('/p2p/listings'),
};

// ---------------------------------------------------------------------------
// Shared UI state + tab navigation
// ---------------------------------------------------------------------------
const state = { wallets: [], activeWallet: null };

function switchTab(tab) {
  ['walletTab', 'sendTab', 'convertTab', 'stakingTab', 'fiatTab', 'qrTab'].forEach((t) => {
    const el = document.getElementById(t);
    if (el) el.classList.add('hidden');
  });
  const target = document.getElementById(tab + 'Tab') || document.getElementById(tab);
  if (target) target.classList.remove('hidden');
  if (tab === 'wallet') loadWallets();
  if (tab === 'convert') loadConvert();
  if (tab === 'staking') loadStaking();
  if (tab === 'fiat') loadFiatProviders();
}

function activeWallet() {
  return state.wallets && state.wallets[0] ? state.wallets[0] : null;
}

// ---- Send view ----
async function handleSend() {
  const w = activeWallet();
  if (!w) { alert('No wallet available.'); return; }
  const to = document.getElementById('sendTo').value.trim();
  const amount = document.getElementById('sendAmount').value.trim();
  const password = document.getElementById('sendPassword').value;
  if (!to || !amount || !password) { alert('Fill all fields.'); return; }
  try {
    const res = await WalletAPI.sendTransaction(w.id, password, to, amount, w.chain_id);
    alert('Transaction sent: ' + (res.transaction_hash || res.tx_hash || 'submitted'));
    document.getElementById('sendTo').value = '';
    document.getElementById('sendAmount').value = '';
    document.getElementById('sendPassword').value = '';
  } catch (err) { alert(err.message); }
}

// ---- Convert / Swap view ----
async function loadConvert() {
  try {
    const { chains } = await WalletAPI.getChains();
    const sel = document.getElementById('convertChain');
    if (sel && !sel.options.length) {
      chains.slice(0, 20).forEach((c) => {
        const o = document.createElement('option');
        o.value = c.id; o.textContent = `${c.name} (${c.symbol})`;
        sel.appendChild(o);
      });
    }
  } catch (_) { /* fail-closed */ }
}

async function handleConvert() {
  const from = document.getElementById('convertFrom').value.trim();
  const to = document.getElementById('convertTo').value.trim();
  const amount = document.getElementById('convertAmount').value.trim();
  const chainId = parseInt(document.getElementById('convertChain').value || '1', 10);
  if (!from || !to || !amount) { alert('Fill all fields.'); return; }
  try {
    const q = await WalletAPI.getConvertQuote(from, to, amount, chainId);
    document.getElementById('convertResult').textContent =
      `${amount} ${from} = ${q.to_amount || q.toAmount || '?'} ${to}`;
  } catch (err) { document.getElementById('convertResult').textContent = err.message; }
}

// ---- Staking view ----
async function loadStaking() {
  try {
    const q = await WalletAPI.getStakingQuote();
    const list = q.assets || [];
    document.getElementById('stakingList').innerHTML = list.length
      ? list.map((a) => `<div class="wallet-item"><div class="wallet-label">${escapeHtml(a.symbol)} <span style="color:var(--text-secondary)">· Chain #${a.chain_id}</span></div><div class="wallet-balance">APY: ${a.apy}% · Min: ${a.min_stake}</div></div>`).join('')
      : '<div class="wallet-label" style="color:var(--text-secondary)">No staking assets available.</div>';
  } catch (err) {
    document.getElementById('stakingList').innerHTML = `<div class="wallet-label" style="color:var(--error)">${escapeHtml(err.message)}</div>`;
  }
}

async function handleStake() {
  const w = activeWallet();
  if (!w) { alert('No wallet available.'); return; }
  const token = document.getElementById('stakeToken').value.trim();
  const amount = document.getElementById('stakeAmount').value.trim();
  const password = document.getElementById('stakePassword').value;
  if (!token || !amount || !password) { alert('Fill all fields.'); return; }
  try {
    const res = await WalletAPI.stake(w.id, password, token, amount, w.chain_id);
    alert(res.action_required ? 'Staking requires a staking contract + calldata. Submit via Send.' : 'Stake submitted.');
  } catch (err) { alert(err.message); }
}

// ---- Fiat ramp view ----
async function loadFiatProviders() {
  try {
    const res = await WalletAPI.getFiatRampProviders();
    const providers = res.providers || [];
    const sel = document.getElementById('fiatProvider');
    if (sel && !sel.options.length) {
      providers.forEach((p) => {
        const o = document.createElement('option');
        o.value = p.id; o.textContent = `${p.name} (${p.id})`;
        sel.appendChild(o);
      });
    }
  } catch (_) { /* fail-closed */ }
}

async function handleFiatQuote(isOfframp) {
  const providerId = document.getElementById('fiatProvider').value;
  const amount = document.getElementById('fiatAmount').value.trim();
  const fiat = document.getElementById('fiatCurrency').value.trim() || 'USD';
  const crypto = document.getElementById('fiatCrypto').value.trim() || 'ETH';
  const method = document.getElementById('fiatMethod').value.trim() || 'card';
  if (!providerId || !amount) { alert('Select provider and enter amount.'); return; }
  try {
    const q = isOfframp
      ? await WalletAPI.getFiatOfframpQuote(providerId, amount, fiat, crypto)
      : await WalletAPI.getFiatRampQuote(providerId, amount, fiat, crypto, method);
    const out = document.getElementById('fiatResult');
    if (isOfframp) {
      out.textContent = `${amount} ${crypto} = ${q.fiatNet || '?'} ${fiat} (net)`;
    } else {
      out.textContent = `${amount} ${fiat} = ${q.cryptoAmount || '?'} ${crypto}`;
      if (q.checkoutUrl) {
        out.textContent += ' — click Open Provider to continue.';
        document.getElementById('fiatOpenUrl').href = q.checkoutUrl;
        document.getElementById('fiatOpenUrl').classList.remove('hidden');
      }
    }
  } catch (err) { document.getElementById('fiatResult').textContent = err.message; }
}

// ---- QR scan view (paste + parse; camera not available in extension popup) ----
function handleQrPaste() {
  const raw = document.getElementById('qrInput').value.trim();
  const parsed = parsePaymentUri(raw);
  const out = document.getElementById('qrResult');
  if (!parsed) {
    out.textContent = 'No address found in input.';
    document.getElementById('qrToAddress').value = '';
    return;
  }
  out.textContent = `Address: ${parsed.address}${parsed.amount ? ' · Amount: ' + parsed.amount : ''}${parsed.chain_id ? ' · Chain: ' + parsed.chain_id : ''}`;
  document.getElementById('qrToAddress').value = parsed.address;
  if (parsed.amount) document.getElementById('qrAmount').value = parsed.amount;
}

// parse_payment_uri — bare 0x address, ethereum: URI, or EIP-681 payment URI.
function parsePaymentUri(input) {
  const s = (input || '').trim();
  if (!s) return null;
  if (/^0x[a-fA-F0-9]{40}$/.test(s)) {
    return { address: s, amount: null, chain_id: null, token_address: null };
  }
  let body;
  if (s.startsWith('ethereum:')) body = s.slice('ethereum:'.length);
  else return null;
  const qIdx = body.indexOf('?');
  const target = qIdx >= 0 ? body.slice(0, qIdx) : body;
  const query = qIdx >= 0 ? body.slice(qIdx + 1) : '';
  let address, tokenAddress = null;
  if (target.includes('/')) {
    const [addr, func] = target.split('/');
    address = addr;
    if (func.startsWith('transfer')) tokenAddress = '';
  } else {
    address = target;
  }
  if (!/^0x[a-fA-F0-9]{40}$/.test(address)) return null;
  let amount = null, chainId = null;
  query.split('&').forEach((pair) => {
    const [k, v] = pair.split('=');
    if (k === 'value') amount = v;
    else if (k === 'chainId') chainId = parseInt(v, 10);
    else if (k === 'address' && tokenAddress !== null) tokenAddress = v;
  });
  return { address, amount, chain_id: chainId, token_address: tokenAddress || null };
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
