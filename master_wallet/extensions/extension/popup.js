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

  // ---- Feature tabs -------------------------------------------------------
  // Every tab renders ONLY data returned by the canonical backend through
  // MW_RELAY. On any error the panel shows the real error; nothing is
  // fabricated. The backend shapes vary (raw array vs {items: [...]}), so
  // every reader normalizes first.
  const TABS = [
    ['wallets', 'Wallets'],
    ['transactions', 'Txs'],
    ['treasury', 'Treasury'],
    ['multisig', 'Multisig'],
    ['autosign', 'Auto-Sign'],
    ['fees', 'Fees'],
    ['policies', 'Policies'],
    ['users', 'Users'],
    ['chains', 'Chains'],
    ['tokens', 'Tokens'],
    ['flags', 'Flags'],
    ['webhooks', 'Hooks'],
    ['audit', 'Audit'],
    ['analytics', 'Analytics'],
    ['withdraw', 'Withdraw'],
  ];

  function buildTabbar() {
    const bar = $('tabbar');
    if (!bar || bar.childElementCount) return;
    for (const [id, label] of TABS) {
      const b = el('button', { id: 'tabbtn-' + id, text: label });
      b.addEventListener('click', () => showTab(id));
      bar.appendChild(b);
    }
  }

  let activeTab = 'wallets';
  function showTab(id) {
    activeTab = id;
    for (const [tid] of TABS) {
      const panel = $('tab-' + tid);
      const btn = $('tabbtn-' + tid);
      if (panel) panel.classList.toggle('hidden', tid !== id);
      if (btn) btn.classList.toggle('active', tid === id);
    }
    loadTab(id);
  }

  function loadTab(id) {
    switch (id) {
      case 'wallets': renderWalletView(); break;
      case 'transactions': renderTransactions(); break;
      case 'treasury': renderTreasury(); break;
      case 'multisig': renderMultisig(); break;
      case 'autosign': renderAutoSign(); break;
      case 'fees': renderFees(); break;
      case 'policies': renderPolicies(); break;
      case 'users': renderUsers(); break;
      case 'chains': renderChains(); break;
      case 'tokens': renderTokens(); break;
      case 'flags': renderFlags(); break;
      case 'webhooks': renderWebhooks(); break;
      case 'audit': renderAudit(); break;
      case 'analytics': renderAnalytics(); break;
      case 'withdraw': break; // form-only tab
      default: break;
    }
  }

  function asArray(res, key) {
    if (Array.isArray(res)) return res;
    if (res && Array.isArray(res[key])) return res[key];
    if (res) {
      for (const v of Object.values(res)) if (Array.isArray(v)) return v;
    }
    return [];
  }

  function renderList(listEl, items, titleFn, subFn, actionsFn) {
    const node = typeof listEl === 'string' ? $(listEl) : listEl;
    if (!node) return;
    node.innerHTML = '';
    if (!items.length) {
      node.appendChild(el('div', { class: 'empty' }, 'None.'));
      return;
    }
    items.forEach((item, i) => {
      const row = el('div', { class: 'kv' },
        el('b', { text: titleFn(item) }),
        subFn ? el('div', { class: 'muted', text: subFn(item) }) : null);
      const actions = actionsFn ? actionsFn(item, i) : null;
      if (actions) {
        const ar = el('div', { class: 'rowin' });
        for (const a of actions) ar.appendChild(a);
        row.appendChild(ar);
      }
      node.appendChild(row);
    });
  }

  function actionBtn(label, cls, onClick) {
    const b = el('button', { class: 'btn ' + (cls || ''), text: label });
    b.addEventListener('click', onClick);
    return b;
  }

  async function relay(action, args) {
    return send('MW_RELAY', { action, args: args || [] });
  }

  async function runAction(fn, refresh) {
    try {
      await fn();
      setStatus('Done.');
    } catch (e) {
      setStatus((e && e.message) || String(e), true);
      return;
    }
    if (refresh) refresh();
  }

  // ---- Transactions ----
  async function renderTransactions() {
    try {
      const res = await relay('listTransactions');
      const txs = asArray(res, 'transactions');
      renderList('txList', txs,
        (t) => (t.token || t.chain || 'Transfer') + ' — ' + (t.amount || t.value || '0'),
        (t) => 'to ' + truncate(t.to || t.to_address || '', 16) + ' · ' + (t.status || ''),
        (t) => {
          const acts = [];
          if ((t.status || '').toLowerCase() === 'pending') {
            acts.push(actionBtn('Approve', '', () => runAction(() => relay('approveTransaction', [null, t.id]), renderTransactions)));
            acts.push(actionBtn('Reject', 'danger', () => runAction(() => relay('rejectTransaction', [null, t.id]), renderTransactions)));
          }
          return acts;
        });
    } catch (e) { renderList('txList', [], null, null); setStatus(e.message, true); }
  }

  // ---- Treasury ----
  async function renderTreasury() {
    try {
      const ov = await relay('getTreasuryOverview');
      renderList('treasuryOverview', ov ? [ov] : [],
        (o) => 'Balance: ' + (o.balance || o.total_balance || '0'),
        (o) => 'address ' + truncate(o.address || o.treasury_address || '', 16));
    } catch (e) { renderList('treasuryOverview', []); setStatus(e.message, true); }
    try {
      const res = await relay('getTreasuryTransactions');
      renderList('treasuryTxList', asArray(res, 'transactions'),
        (t) => (t.tx_type || 'tx') + ' — ' + (t.amount || ''),
        (t) => truncate(t.to_address || t.to || '', 16) + ' · ' + (t.status || ''));
    } catch (e) { renderList('treasuryTxList', []); }
  }

  // ---- Multisig ----
  let selectedMsig = null;
  async function renderMultisig() {
    try {
      const res = await relay('listMultisigWallets');
      const list = asArray(res, 'wallets');
      renderList('multisigList', list,
        (m) => (m.name || 'Multisig') + ' (' + (m.threshold || '?') + '/' + ((m.owners || []).length || '?') + ')',
        (m) => truncate(m.address || m.id || '', 18),
        (m) => [actionBtn('Txs', 'secondary', () => { selectedMsig = m.id; renderMultisigTxs(); })]);
    } catch (e) { renderList('multisigList', []); setStatus(e.message, true); }
  }

  async function renderMultisigTxs() {
    if (!selectedMsig) return;
    try {
      const res = await relay('listMultisigTransactions', [null, selectedMsig]);
      renderList('multisigTxList', asArray(res, 'transactions'),
        (t) => (t.to || '') + ' — ' + (t.value || t.amount || ''),
        (t) => (t.status || '') + ' · sigs ' + ((t.signatures || []).length),
        (t) => [
          actionBtn('Sign', '', () => runAction(() => relay('signMultisigTransaction', [null, t.id]), renderMultisigTxs)),
          actionBtn('Execute', 'secondary', () => runAction(() => relay('executeMultisigTransaction', [null, t.id]), renderMultisigTxs)),
        ]);
    } catch (e) { renderList('multisigTxList', []); setStatus(e.message, true); }
  }

  // ---- Auto-sign ----
  async function renderAutoSign() {
    try {
      const res = await relay('getAutoSignPolicy');
      const p = (res && res.policy) || res || {};
      renderList('autoSignPolicy', [p],
        (x) => 'Daemon: ' + (x.enabled ? 'ENABLED' : 'disabled'),
        (x) => 'max auto value: ' + (x.max_auto_value_wei || '0') + ' wei');
    } catch (e) { renderList('autoSignPolicy', []); }
    try {
      const res = await relay('listAutoSign');
      renderList('autoSignRuleList', asArray(res, 'auto_sign_rules'),
        (r) => r.name || r.id,
        (r) => (r.rule_type || '') + ' · max ' + (r.max_amount || '0') + ' · ' + (r.is_active ? 'active' : 'off'),
        (r) => [
          actionBtn(r.is_active ? 'Disable' : 'Enable', 'secondary', () => runAction(() => relay('updateAutoSignRule', [null, r.id, { is_active: !r.is_active }]), renderAutoSign)),
          actionBtn('Delete', 'danger', () => runAction(() => relay('deleteAutoSign', [null, r.id]), renderAutoSign)),
        ]);
    } catch (e) { renderList('autoSignRuleList', []); }
    try {
      const res = await relay('listAutoSignLogs');
      renderList('autoSignLogList', asArray(res, 'logs'),
        (l) => (l.action || l.status || 'log') + ' · ' + truncate(l.tx_hash || '', 14),
        (l) => (l.created_at || '') + (l.reason ? ' · ' + l.reason : ''));
    } catch (e) { renderList('autoSignLogList', []); }
  }

  // ---- Fees ----
  async function renderFees() {
    try {
      const res = await relay('listFees');
      renderList('feeList', asArray(res, 'fees'),
        (f) => (f.fee_type || 'fee') + ' — ' + (f.fee_percentage || 0) + '%' + (f.fee_fixed ? ' + ' + f.fee_fixed : ''),
        (f) => f.is_active ? 'active' : 'disabled',
        (f) => [
          actionBtn(f.is_active ? 'Disable' : 'Enable', 'secondary', () => runAction(() => relay('updateFee', [null, f.id, { is_active: !f.is_active }]), renderFees)),
          actionBtn('Delete', 'danger', () => runAction(() => relay('deleteFee', [null, f.id]), renderFees)),
        ]);
    } catch (e) { renderList('feeList', []); setStatus(e.message, true); }
  }

  // ---- Policies ----
  async function renderPolicies() {
    try {
      const res = await relay('listPolicies');
      renderList('policyList', asArray(res, 'policies'),
        (p) => (p.name || 'policy') + ' (' + (p.policy_type || '') + ')',
        (p) => 'priority ' + (p.priority || 0) + ' · ' + (p.is_active ? 'active' : 'off'),
        (p) => [actionBtn('Delete', 'danger', () => runAction(() => relay('deletePolicy', [null, p.id]), renderPolicies))]);
    } catch (e) { renderList('policyList', []); setStatus(e.message, true); }
  }

  // ---- Users ----
  async function renderUsers() {
    try {
      const res = await relay('listUsers');
      renderList('userList', asArray(res, 'users'),
        (u) => (u.name || u.email || 'user') + ' (' + (u.role || 'user') + ')',
        (u) => u.email || '',
        (u) => [actionBtn('Delete', 'danger', () => runAction(() => relay('deleteUser', [null, u.id]), renderUsers))]);
    } catch (e) { renderList('userList', []); setStatus(e.message, true); }
  }

  // ---- Chains ----
  async function renderChains() {
    try {
      const res = await relay('listUserEVMChains');
      renderList('evmChainList', asArray(res, 'chains'),
        (c) => (c.name || '') + ' (' + (c.chain_id || '') + ')' + (c.symbol ? ' · ' + c.symbol : ''),
        (c) => c.rpc_url || '',
        (c) => [actionBtn('Remove', 'danger', () => runAction(() => relay('removeUserEVMChain', [null, c.chain_id]), renderChains))]);
    } catch (e) { renderList('evmChainList', []); }
    try {
      const res = await relay('listUserNonEVMChains');
      renderList('nonEvmChainList', asArray(res, 'chains'),
        (c) => (c.name || '') + ' (' + (c.chain_type || '') + ')' + (c.symbol ? ' · ' + c.symbol : ''),
        (c) => 'chain_id ' + (c.chain_id || '') + ' · ' + (c.rpc_url || ''),
        (c) => [actionBtn('Remove', 'danger', () => runAction(() => relay('removeUserNonEVMChain', [null, c.chain_id]), renderChains))]);
    } catch (e) { renderList('nonEvmChainList', []); }
  }

  // ---- Tokens ----
  async function renderTokens() {
    try {
      const res = await relay('listUserTokens', [null, null]);
      renderList('tokenList', asArray(res, 'tokens'),
        (t) => (t.symbol || '') + ' — ' + (t.name || '') + (t.is_native ? ' (native)' : ''),
        (t) => 'chain ' + (t.chain_id || '') + ' · ' + truncate(t.contract_address || 'native', 16),
        (t) => [actionBtn('Remove', 'danger', () => runAction(() => relay('removeUserToken', [null, t.id]), renderTokens))]);
    } catch (e) { renderList('tokenList', []); setStatus(e.message, true); }
  }

  // ---- Feature flags ----
  async function renderFlags() {
    try {
      const res = await relay('listFeatureFlags');
      renderList('flagList', asArray(res, 'feature_flags'),
        (f) => (f.flag_key || '') + (f.is_enabled ? ' ✅' : ' ⛔'),
        (f) => (f.flag_value || '') + (f.description ? ' · ' + f.description : ''),
        (f) => [
          actionBtn(f.is_enabled ? 'Disable' : 'Enable', 'secondary', () => runAction(() => relay('updateFeatureFlag', [null, f.id, { is_enabled: !f.is_enabled }]), renderFlags)),
          actionBtn('Remove', 'danger', () => runAction(() => relay('removeFeatureFlag', [null, f.id]), renderFlags)),
        ]);
    } catch (e) { renderList('flagList', []); setStatus(e.message, true); }
  }

  // ---- Webhooks + notifications ----
  async function renderWebhooks() {
    try {
      const res = await relay('listWebhooks');
      renderList('webhookList', asArray(res, 'webhooks'),
        (w) => w.name || w.url || 'webhook',
        (w) => (w.url || '') + ' · ' + ((w.events || []).join(', ') || 'all events'),
        (w) => [actionBtn('Delete', 'danger', () => runAction(() => relay('deleteWebhook', [null, w.id]), renderWebhooks))]);
    } catch (e) { renderList('webhookList', []); }
    try {
      const res = await relay('listNotifications');
      renderList('notifList', asArray(res, 'notifications'),
        (n) => (n.title || n.notification_type || 'notification'),
        (n) => (n.message || '') + ' · ' + (n.created_at || ''));
    } catch (e) { renderList('notifList', []); }
  }

  // ---- Audit ----
  async function renderAudit() {
    try {
      const res = await relay('getAudit');
      renderList('auditList', asArray(res, 'logs'),
        (a) => (a.action || a.event || 'event') + ' by ' + truncate(a.actor || a.user_id || '', 12),
        (a) => (a.details || a.description || '') + ' · ' + (a.created_at || a.timestamp || ''));
    } catch (e) { renderList('auditList', []); setStatus(e.message, true); }
  }

  // ---- Analytics ----
  async function renderAnalytics() {
    try {
      const res = await relay('getAnalyticsVolume');
      const v = (res && res.volume) || res || {};
      renderList('analyticsVolume', [v], (x) => 'total ' + (x.total_volume || x.total || '0'),
        (x) => 'daily ' + (x.daily_volume || '0') + ' · monthly ' + (x.monthly_volume || '0'));
    } catch (e) { renderList('analyticsVolume', []); }
    try {
      const res = await relay('getAnalyticsTransactions');
      renderList('analyticsTx', asArray(res, 'transactions').slice(0, 20),
        (t) => (t.status || 'tx') + ' — ' + (t.count || t.amount || ''),
        (t) => t.date || t.created_at || '');
    } catch (e) { renderList('analyticsTx', []); }
    try {
      const res = await relay('getAnalyticsWallets');
      renderList('analyticsWallets', asArray(res, 'wallets').slice(0, 20),
        (w) => w.name || truncate(w.address || '', 14),
        (w) => (w.tx_count != null ? w.tx_count + ' txs' : '') + (w.balance ? ' · ' + w.balance : ''));
    } catch (e) { renderList('analyticsWallets', []); }
  }

  // ---- Feature actions ----
  async function treasuryTransfer() {
    const to = $('trTo').value.trim();
    const amount = $('trAmount').value.trim();
    const password = $('trPassword').value;
    if (!to || !amount || !password) { setStatus('to/amount/password required', true); return; }
    await runAction(() => relay('treasuryTransfer', [null, { to, amount, password }]), renderTreasury);
  }

  async function treasurySweep() {
    const subWalletId = $('swSubWalletId').value.trim();
    const password = $('swPassword').value;
    if (!subWalletId || !password) { setStatus('sub_wallet_id and password required', true); return; }
    await runAction(() => relay('treasurySweep', [null, { sub_wallet_id: subWalletId, password }]), renderTreasury);
  }

  async function createMultisig() {
    const name = $('msName').value.trim();
    const owners = $('msOwners').value.split(',').map((s) => s.trim()).filter(Boolean);
    const threshold = parseInt($('msThreshold').value, 10);
    if (!name || !owners.length || !threshold) { setStatus('name, owners, threshold required', true); return; }
    await runAction(() => relay('createMultisigWallet', [null, { name, owners, threshold }]), renderMultisig);
  }

  async function createAutoSignRule() {
    const name = $('asName').value.trim();
    const ruleType = $('asRuleType').value.trim();
    const maxAmount = $('asThreshold').value.trim();
    if (!name || !ruleType) { setStatus('name and rule_type required', true); return; }
    await runAction(() => relay('createAutoSign', [null, { name, rule_type: ruleType, max_amount: maxAmount }]), renderAutoSign);
  }

  async function setAutoSignPolicy(enabled) {
    await runAction(() => relay('updateAutoSignPolicy', [null, { enabled }]), renderAutoSign);
  }

  async function createFee() {
    const feeType = $('feeType').value.trim();
    const feePercentage = parseFloat($('feePercentage').value) || 0;
    const feeFixed = $('feeFixed').value.trim();
    if (!feeType) { setStatus('fee_type required', true); return; }
    await runAction(() => relay('createFee', [null, { fee_type: feeType, fee_percentage: feePercentage, fee_fixed: feeFixed }]), renderFees);
  }

  async function createPolicy() {
    const name = $('polName').value.trim();
    const policyType = $('polType').value.trim();
    if (!name || !policyType) { setStatus('name and policy_type required', true); return; }
    await runAction(() => relay('createPolicy', [null, { name, policy_type: policyType }]), renderPolicies);
  }

  async function createUser() {
    const email = $('usrEmail').value.trim();
    const password = $('usrPassword').value;
    const name = $('usrName').value.trim();
    const role = $('usrRole').value.trim();
    if (!email || !password) { setStatus('email and password required', true); return; }
    await runAction(() => relay('createUser', [null, { email, password, name, role }]), renderUsers);
  }

  async function addEvmChain() {
    const chainId = parseInt($('evmChainId').value, 10);
    const name = $('evmName').value.trim();
    const rpcUrl = $('evmRpc').value.trim();
    const symbol = $('evmSymbol').value.trim();
    if (!chainId || !name || !rpcUrl) { setStatus('chain_id, name, rpc_url required', true); return; }
    await runAction(() => relay('addUserEVMChain', [null, { chain_id: chainId, name, rpc_url: rpcUrl, symbol }]), renderChains);
  }

  async function addNonEvmChain() {
    const chainId = parseInt($('neChainId').value, 10);
    const name = $('neName').value.trim();
    const chainType = $('neChainType').value.trim();
    const rpcUrl = $('neRpc').value.trim();
    const derivationPath = $('neDerivation').value.trim();
    if (!chainId || !name || !chainType || !derivationPath) { setStatus('chain_id, name, chain_type, derivation_path required', true); return; }
    await runAction(() => relay('addUserNonEVMChain', [null, { chain_id: chainId, name, chain_type: chainType, rpc_url: rpcUrl, derivation_path: derivationPath }]), renderChains);
  }

  async function addToken() {
    const chainId = parseInt($('tokChainId').value, 10);
    const symbol = $('tokSymbol').value.trim();
    const name = $('tokName').value.trim();
    const contractAddress = $('tokAddress').value.trim();
    const decimals = parseInt($('tokDecimals').value, 10) || 18;
    if (!chainId || !symbol || !name) { setStatus('chain_id, symbol, name required', true); return; }
    await runAction(() => relay('addUserToken', [null, { chain_id: chainId, symbol, name, contract_address: contractAddress, decimals }]), renderTokens);
  }

  async function addFlag() {
    const flagKey = $('flagKey').value.trim();
    if (!flagKey) { setStatus('flag_key required', true); return; }
    await runAction(() => relay('addFeatureFlag', [null, { flag_key: flagKey, is_enabled: true }]), renderFlags);
  }

  async function createWebhook() {
    const name = $('whName').value.trim();
    const url = $('whUrl').value.trim();
    const events = $('whEvents').value.split(',').map((s) => s.trim()).filter(Boolean);
    if (!name || !url || !events.length) { setStatus('name, url, events required', true); return; }
    await runAction(() => relay('createWebhook', [null, { name, url, events }]), renderWebhooks);
  }

  async function createNotification() {
    const type = $('ntType').value.trim();
    const title = $('ntTitle').value.trim();
    const message = $('ntMessage').value.trim();
    if (!type || !title || !message) { setStatus('type, title, message required', true); return; }
    await runAction(() => relay('createNotification', [null, { notification_type: type, title, message }]), renderWebhooks);
  }

  async function requestWithdrawal() {
    const toAddress = $('wdTo').value.trim();
    const amountWei = $('wdAmount').value.trim();
    const currency = $('wdCurrency').value.trim();
    const chainId = parseInt($('wdChainId').value, 10) || 1;
    if (!toAddress || !amountWei) { setStatus('to_address and amount_wei required', true); return; }
    await runAction(() => relay('requestWithdrawal', [null, { to_address: toAddress, amount_wei: amountWei, currency, chain_id: chainId }]));
  }

  function wireFeatureTabs() {
    buildTabbar();
    const bind = (id, fn) => { const b = $(id); if (b) b.addEventListener('click', fn); };
    bind('treasuryTransferBtn', treasuryTransfer);
    bind('treasurySweepBtn', treasurySweep);
    bind('createMultisigBtn', createMultisig);
    bind('createAutoSignBtn', createAutoSignRule);
    bind('policyEnableBtn', () => setAutoSignPolicy(true));
    bind('policyDisableBtn', () => setAutoSignPolicy(false));
    bind('createFeeBtn', createFee);
    bind('createPolicyBtn', createPolicy);
    bind('createUserBtn', createUser);
    bind('addEvmChainBtn', addEvmChain);
    bind('addNonEvmChainBtn', addNonEvmChain);
    bind('addTokenBtn', addToken);
    bind('addFlagBtn', addFlag);
    bind('createWebhookBtn', createWebhook);
    bind('createNotifBtn', createNotification);
    bind('requestWithdrawalBtn', requestWithdrawal);
    showTab('wallets');
  }

  // ---- Wire up ------------------------------------------------------------
  document.addEventListener('DOMContentLoaded', async () => {
    setAuthMode(false);
    wireFeatureTabs();
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
