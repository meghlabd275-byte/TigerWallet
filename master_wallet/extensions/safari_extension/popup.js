/**
 * MasterWallet extension popup UI.
 *
 * Talks to the background service worker via chrome.runtime.sendMessage. The
 * popup holds NO business logic and NO fake data. Every value shown comes from
 * the real backend through the MW_RELAY / MW_AUTH_* messages handled in
 * background.js. Theme (light/dark) is persisted via the background's
 * MW_THEME_GET/SET handlers (which store `mw_theme` in chrome.storage and
 * broadcast MW_THEME_CHANGED to content scripts + injected provider).
 */

'use strict';

(function () {
  const $ = (id) => document.getElementById(id);

  function send(type, payload) {
    return new Promise((resolve, reject) => {
      chrome.runtime.sendMessage({ type, payload }, (res) => {
        const err = chrome.runtime.lastError;
        if (err) return reject(new Error(err.message || String(err)));
        if (!res) return reject(new Error('No response from background'));
        if (!res.ok) return reject(new Error(res.error || 'Unknown error'));
        resolve(res.data);
      });
    });
  }

  function el(tag, attrs, ...children) {
    const node = document.createElement(tag);
    if (attrs) {
      for (const [k, v] of Object.entries(attrs)) {
        if (k === 'class') node.className = v;
        else if (k === 'text') node.textContent = v;
        else node.setAttribute(k, v);
      }
    }
    for (const c of children) {
      if (c == null) continue;
      node.appendChild(typeof c === 'string' ? document.createTextNode(c) : c);
    }
    return node;
  }

  function setStatus(msg, isError) {
    const node = $('status');
    if (!node) return;
    node.textContent = msg || '';
    node.classList.toggle('error', !!isError);
  }

  // ---- Theme --------------------------------------------------------------
  async function applyTheme() {
    const theme = await send('MW_THEME_GET', {});
    document.documentElement.setAttribute('data-theme', theme);
  }

  async function toggleTheme() {
    const current = document.documentElement.getAttribute('data-theme') || 'light';
    const next = current === 'dark' ? 'light' : 'dark';
    await send('MW_THEME_SET', { theme: next });
    document.documentElement.setAttribute('data-theme', next);
  }

  // Listen for live theme broadcasts (e.g. changed elsewhere).
  chrome.runtime.onMessage.addListener((msg) => {
    if (msg && msg.type === 'MW_THEME_CHANGED') {
      document.documentElement.setAttribute('data-theme', msg.theme);
    }
    return false;
  });

  // ---- Auth ---------------------------------------------------------------
  let registering = false;

  function setAuthMode(register) {
    registering = register;
    $('authTitle').textContent = register ? 'Create account' : 'Sign In';
    $('authBtn').textContent = register ? 'Register' : 'Sign In';
    $('name').classList.toggle('hidden', !register);
    $('password').setAttribute('placeholder', register ? 'New password' : 'Password');
  }

  async function doAuth() {
    const email = $('email').value.trim();
    const password = $('password').value;
    const name = $('name').value.trim();
    if (!email || !password) { setStatus('Email and password required', true); return; }
    if (registering && !name) { setStatus('Name required to register', true); return; }

    setStatus('Contacting backendâ¦');
    try {
      const data = registering
        ? await send('MW_AUTH_REGISTER', { email, password, name })
        : await send('MW_AUTH_LOGIN', { email, password });
      if (!data || !data.token) {
        setStatus('Backend did not return a token', true);
        return;
      }
      setStatus('Signed in as ' + (data.email || email));
      await renderWalletView();
    } catch (e) {
      setStatus((e && e.message) || String(e), true);
    }
  }

  async function logout() {
    try { await send('MW_AUTH_LOGOUT', {}); } catch (_) { /* ignore */ }
    showAuth();
  }

  // ---- Wallet view --------------------------------------------------------
  async function renderWalletView() {
    const ctx = await send('MW_AUTH_CONTEXT', {});
    if (!ctx || !ctx.token) { showAuth(); return; }
    $('authView').classList.add('hidden');
    $('walletView').classList.remove('hidden');
    setStatus('Loading walletsâ¦');
    try {
      const res = await send('MW_RELAY', { action: 'listMasterWallets', args: [] });
      const wallets = (res && (res.wallets || res)) || [];
      renderWalletList(Array.isArray(wallets) ? wallets : []);
      renderPasskeys();
    } catch (e) {
      setStatus((e && e.message) || String(e), true);
      renderWalletList([]);
      renderPasskeys();
    }
  }

  function renderWalletList(wallets) {
    const list = $('walletList');
    list.innerHTML = '';
    if (!wallets.length) {
      list.appendChild(el('div', { class: 'empty' }, 'No master wallets. Create one from the web app.'));
      return;
    }
    for (const w of wallets) {
      const id = String(w.id || w.ID || '');
      const name = w.name || 'Untitled';
      const address = w.address || (w.wallet && w.wallet.address) || 'â';
      list.appendChild(
        el('div', { class: 'row' },
          el('div', null,
            el('div', { text: name }),
            el('div', { class: 'muted', text: truncate(address, 14) })
          ),
          el('button', { class: 'btn secondary', style: 'width:auto;margin:0;', id: 'w-' + id }, 'Select')
        )
      );
      const btn = document.getElementById('w-' + id);
      if (btn) btn.addEventListener('click', () => selectWallet(id, name));
    }
  }

  function truncate(s, n) {
    if (!s) return '';
    return s.length <= n ? s : s.slice(0, 6) + 'â¦' + s.slice(-4);
  }

  async function selectWallet(id, name) {
    try {
      await send('MW_RELAY', { action: 'setCurrentWallet', args: [id] });
      setStatus('Selected: ' + name);
      renderPasskeys();
    } catch (e) {
      setStatus((e && e.message) || String(e), true);
    }
  }

  // ---- Passkeys -----------------------------------------------------------
  // Authoritative list comes from the backend via the MW_RELAY 'listPasskeys'
  // handler in background.js, which calls masterWalletService.listPasskeys on
  // the currently selected master wallet. No passkeys are fabricated.
  async function renderPasskeys() {
    const list = $('passkeyList');
    if (!list) return;
    list.innerHTML = '';
    let passkeys = [];
    try {
      const res = await send('MW_RELAY', { action: 'listPasskeys', args: [] });
      passkeys = (res && (res.passkeys || res)) || [];
      if (!Array.isArray(passkeys)) passkeys = [];
    } catch (e) {
      list.appendChild(el('div', { class: 'muted' }, '\u26A0 ' + ((e && e.message) || String(e))));
      return;
    }
    if (!passkeys.length) {
      list.appendChild(el('div', { class: 'empty' }, 'No passkeys registered.'));
      return;
    }
    for (const p of passkeys) {
      const credId = p.credential_id || p.id || '';
      const label = p.label || '';
      const created = p.created_at ? new Date(p.created_at).toLocaleString() : '';
      const row = el('div', { class: 'row' },
        el('div', null,
          el('div', { text: label ? (label + ' (' + truncate(credId, 12) + ')') : truncate(credId, 18) }),
          el('div', { class: 'muted', text: created })
        ),
        el('button', { class: 'btn danger', style: 'width:auto;margin:0;', id: 'pk-' + String(credId).slice(0, 12) }, 'Delete')
      );
      list.appendChild(row);
      const btn = document.getElementById('pk-' + String(credId).slice(0, 12));
      if (btn) btn.addEventListener('click', () => deletePasskey(credId));
    }
  }

  async function deletePasskey(credentialId) {
    try {
      await send('MW_RELAY', { action: 'deletePasskey', args: [credentialId] });
      setStatus('Passkey deleted');
      renderPasskeys();
    } catch (e) {
      setStatus((e && e.message) || String(e), true);
    }
  }

  // Real WebAuthn registration: run navigator.credentials.create in the popup
  // (an https/secure context), then relay the resulting credential_id +
  // SPKI public key to the backend's /passkey/register route via MW_RELAY
  // 'registerPasskey'. The backend is the relying party and stores the key.
  async function registerPasskey() {
    const ctx = await send('MW_AUTH_CONTEXT', {});
    if (!ctx || !ctx.token) { setStatus('Sign in first', true); return; }
    if (!navigator.credentials || !navigator.credentials.create || !window.PublicKeyCredential) {
      setStatus('WebAuthn not supported here', true);
      return;
    }
    const origin = (location && location.hostname) ? location.hostname : 'localhost';
    setStatus('Creating passkey\u2026');
    let credential;
    try {
      const challenge = new Uint8Array(32);
      crypto.getRandomValues(challenge);
      const userId = new Uint8Array(16);
      crypto.getRandomValues(userId);
      credential = await navigator.credentials.create({
        publicKey: {
          rp: { name: 'TigerMasterWallet', id: origin },
          user: { id: userId, name: ctx.email || 'user', displayName: ctx.email || 'user' },
          challenge,
          pubKeyCredParams: [
            { type: 'public-key', alg: -7 },
            { type: 'public-key', alg: -257 },
          ],
          timeout: 60000,
          authenticatorSelection: {
            authenticatorAttachment: 'platform',
            userVerification: 'required',
            requireResidentKey: true,
          },
          attestation: 'none',
        },
      });
    } catch (e) {
      setStatus('WebAuthn create failed: ' + ((e && e.message) || String(e)), true);
      return;
    }
    if (!credential || !credential.id) {
      setStatus('WebAuthn returned no credential', true);
      return;
    }
    const resp = credential.response || {};
    let publicKeyBytes = null;
    if (typeof resp.getPublicKey === 'function') {
      const spki = resp.getPublicKey();
      if (spki) publicKeyBytes = spki;
    }
    if (!publicKeyBytes && resp.publicKey) publicKeyBytes = resp.publicKey;
    if (!publicKeyBytes) {
      setStatus('Credential is missing a SPKI public key', true);
      return;
    }
    const transports = (typeof credential.getTransports === 'function')
      ? (credential.getTransports() || [])
      : (credential.transports || []);
    try {
      const res = await send('MW_RELAY', {
        action: 'registerPasskey',
        args: [{
          credential_id: credential.id,
          public_key: bufToB64url(publicKeyBytes),
          transports,
          label: ctx.email || 'popup',
        }],
      });
      if (!res || !res.registered) {
        setStatus('Backend rejected passkey registration', true);
        return;
      }
      setStatus('Passkey registered');
      renderPasskeys();
    } catch (e) {
      setStatus((e && e.message) || String(e), true);
    }
  }

  function bufToB64url(buf) {
    const bytes = buf instanceof Uint8Array ? buf : new Uint8Array(buf);
    let binary = '';
    for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  }

  function showAuth() {
    $('walletView').classList.add('hidden');
    $('authView').classList.remove('hidden');
    setStatus('');
    $('password').value = '';
  }

  // ---- Wire up ------------------------------------------------------------
  document.addEventListener('DOMContentLoaded', async () => {
    setAuthMode(false);
    $('authBtn').addEventListener('click', doAuth);
    $('toggleAuthMode').addEventListener('click', () => setAuthMode(!registering));
    $('logoutBtn').addEventListener('click', logout);
    $('refreshBtn').addEventListener('click', renderWalletView);
    $('themeToggleBtn').addEventListener('click', toggleTheme);
    const regPkBtn = $('registerPasskeyBtn');
    if (regPkBtn) regPkBtn.addEventListener('click', registerPasskey);

    await applyTheme();
    try {
      const ctx = await send('MW_AUTH_CONTEXT', {});
      if (ctx && ctx.token) {
        await renderWalletView();
      } else {
        showAuth();
      }
    } catch (e) {
      showAuth();
      setStatus((e && e.message) || String(e), true);
    }
  });
})();
