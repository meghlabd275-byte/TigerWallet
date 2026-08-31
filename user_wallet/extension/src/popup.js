// TigerWallet UserWallet extension popup.
// Talks to the canonical Go wallet-api backend (go/wallet_api, port 8443):
// REAL on-chain RPC, REAL BIP-39/32/44 derivation, REAL secp256k1 signing,
// AES-256-GCM encrypted-seed persistence (PostgreSQL + Redis). No stubs.

// Backend base URL is user-configurable (Settings tab) and persisted in
// chrome.storage.local so the extension can point at any self-hosted
// wallet_api deployment; localhost is only the default.
let API_BASE = 'http://localhost:8443/api/v1';

function loadApiBase() {
  return new Promise((resolve) => {
    chrome.storage.local.get('tw_api_base', (res) => {
      if (res.tw_api_base) {
        API_BASE = res.tw_api_base.replace(/\/+$/, '').replace(/\/api\/v1$/, '') + '/api/v1';
      }
      // Keep the WalletConnect live socket pointed at the same backend.
      if (typeof WalletConnectSocket !== 'undefined') WalletConnectSocket.setApiBase(apiOrigin());
      resolve(API_BASE);
    });
  });
}

function apiOrigin() {
  return API_BASE.replace(/\/api\/v1\/?$/, '');
}

function wsUrl() {
  return apiOrigin().replace(/^http/i, 'ws') + '/api/v1/ws';
}

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
  await loadApiBase();
  const token = await getToken();
  if (token) {
    showWallets();
  } else {
    showAuth();
  }
  bindEvents();
  connectLiveFeed();
}

function bindEvents() {
  document.getElementById('toggleTheme').addEventListener('click', toggleTheme);
  document.getElementById('authSubmit').addEventListener('click', handleAuth);
  document.getElementById('authToggle').addEventListener('click', toggleAuthMode);
  const guestStartBtn = document.getElementById('guestStart');
  if (guestStartBtn) guestStartBtn.addEventListener('click', handleGuestStart);
  document.getElementById('refreshBtn').addEventListener('click', loadWallets);
  document.getElementById('logoutBtn').addEventListener('click', handleLogout);
  document.getElementById('cardRefreshBtn')?.addEventListener('click', loadCards);
  document.getElementById('msCreateBtn')?.addEventListener('click', createMultisigWallet);
  document.getElementById('msTxCreateBtn')?.addEventListener('click', createMultisigTx);
  document.getElementById('msTxRefreshBtn')?.addEventListener('click', loadMultisigTxs);
  document.getElementById('settingsSaveBtn')?.addEventListener('click', saveSettings);

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
  const simulateBtn = document.getElementById('simulateBtn');
  if (simulateBtn) simulateBtn.addEventListener('click', handleSimulate);
  const sendToInput = document.getElementById('sendTo');
  if (sendToInput) sendToInput.addEventListener('input', handleRecipientInput);
  const unlockBtn = document.getElementById('unlockBtn');
  if (unlockBtn) unlockBtn.addEventListener('click', handleUnlock);
  const createPasskeyBtn = document.getElementById('createPasskeyBtn');
  if (createPasskeyBtn) createPasskeyBtn.addEventListener('click', handleCreatePasskey);
  const kycSubmitBtn = document.getElementById('kycSubmitBtn');
  if (kycSubmitBtn) kycSubmitBtn.addEventListener('click', handleKycSubmit);
  const defiSection = document.getElementById('defiSection');
  if (defiSection) defiSection.addEventListener('change', loadDefi);
  const dappPairBtn = document.getElementById('dappPairBtn');
  if (dappPairBtn) dappPairBtn.addEventListener('click', handleDappPair);
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
  const nftTransferBtn = document.getElementById('nftTransferBtn');
  if (nftTransferBtn) nftTransferBtn.addEventListener('click', handleNftTransfer);
  const bridgeQuoteBtn = document.getElementById('bridgeQuoteBtn');
  if (bridgeQuoteBtn) bridgeQuoteBtn.addEventListener('click', handleBridgeQuote);
  const bridgeTransferBtn = document.getElementById('bridgeTransferBtn');
  if (bridgeTransferBtn) bridgeTransferBtn.addEventListener('click', handleBridgeTransfer);
  const contactAddBtn = document.getElementById('contactAddBtn');
  if (contactAddBtn) contactAddBtn.addEventListener('click', handleAddContact);
  const alertCreateBtn = document.getElementById('alertCreateBtn');
  if (alertCreateBtn) alertCreateBtn.addEventListener('click', handleCreateAlert);
  const securityCheckBtn = document.getElementById('securityCheckBtn');
  if (securityCheckBtn) securityCheckBtn.addEventListener('click', handleSecurityCheck);
  const securityScanBtn = document.getElementById('securityScanBtn');
  if (securityScanBtn) securityScanBtn.addEventListener('click', handleSecurityScan);
  const terminalLoadBtn = document.getElementById('terminalLoadBtn');
  if (terminalLoadBtn) terminalLoadBtn.addEventListener('click', loadTerminal);
  document.getElementById('nonEvmDeriveBtn')?.addEventListener('click', handleNonEvmDerive);
  document.getElementById('nonEvmSignBtn')?.addEventListener('click', handleNonEvmSign);
  document.getElementById('nonEvmSendBtn')?.addEventListener('click', handleNonEvmSend);

  // DAO / Launchpool / Token Sales / Trading / Prediction / Copy Trading handlers
  document.getElementById('daoCreateBtn')?.addEventListener('click', handleDaoCreate);
  document.getElementById('daoYesBtn')?.addEventListener('click', () => handleDaoVote(true));
  document.getElementById('daoNoBtn')?.addEventListener('click', () => handleDaoVote(false));
  document.getElementById('lpStakeBtn')?.addEventListener('click', () => handleLaunchpool(true));
  document.getElementById('lpUnstakeBtn')?.addEventListener('click', () => handleLaunchpool(false));
  document.getElementById('tsParticipateBtn')?.addEventListener('click', handleTokenSaleParticipate);
  document.getElementById('tradeOpenPerpBtn')?.addEventListener('click', () => handleTradeOpen(true));
  document.getElementById('tradeOpenMarginBtn')?.addEventListener('click', () => handleTradeOpen(false));
  document.getElementById('tradeClosePerpBtn')?.addEventListener('click', () => handleTradeClose(true));
  document.getElementById('tradeCloseMarginBtn')?.addEventListener('click', () => handleTradeClose(false));
  document.getElementById('predBetBtn')?.addEventListener('click', handlePredictionBet);
  document.getElementById('copyFollowBtn')?.addEventListener('click', handleCopyFollow);
  document.getElementById('copyStopBtn')?.addEventListener('click', handleCopyStop);
  const qrSendBtn = document.getElementById('qrSendBtn');
  if (qrSendBtn) qrSendBtn.addEventListener('click', async () => {
    const w = activeWallet();
    if (!w) { alert('No wallet available.'); return; }
    const to = document.getElementById('qrToAddress').value.trim();
    const amount = document.getElementById('qrAmount').value.trim();
    const password = document.getElementById('sendPassword').value;
    if (!to || !amount || !password) { alert('Recipient, amount, and password required.'); return; }
    // Primary send path: auto sign + auto approval via /auto-send, with the
    // manual /send as fallback. Either path shows the success message below.
    let res;
    try {
      res = await WalletAPI.autoSendTransaction(w.id, password, to, amount, w.chain_id);
    } catch (autoErr) {
      try {
        res = await WalletAPI.sendTransaction(w.id, password, to, amount, w.chain_id);
      } catch (err) { alert(err.message); return; }
    }
    alert('✓ Transaction submitted to the blockchain network\nHash: ' + ((res && (res.transaction_hash || res.tx_hash)) || 'pending'));
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

// Stable per-extension device id (generated once, persisted in chrome.storage).
// Used as the guest-account device_id so re-installs reuse the same guest.
function getDeviceId() {
  return new Promise((resolve) => {
    if (chrome && chrome.storage && chrome.storage.local) {
      chrome.storage.local.get(['userwallet-device-id'], (res) => {
        let id = res['userwallet-device-id'];
        if (!id) {
          const buf = new Uint8Array(16);
          crypto.getRandomValues(buf);
          id = Array.from(buf).map((b) => b.toString(16).padStart(2, '0')).join('');
          chrome.storage.local.set({ 'userwallet-device-id': id });
        }
        resolve(id);
      });
    } else {
      let id = localStorage.getItem('userwallet-device-id');
      if (!id) {
        const buf = new Uint8Array(16);
        crypto.getRandomValues(buf);
        id = Array.from(buf).map((b) => b.toString(16).padStart(2, '0')).join('');
        localStorage.setItem('userwallet-device-id', id);
      }
      resolve(id);
    }
  });
}

// Quick-start: provision an anonymous guest account (no email/password) so the
// user can Create/Import a wallet WITHOUT registering. Routes to the wallets
// tab where the create/import UI lives.
async function handleGuestStart() {
  hideError();
  const btn = document.getElementById('guestStart');
  if (btn) { btn.disabled = true; btn.textContent = 'Starting…'; }
  try {
    const deviceId = await getDeviceId();
    const res = await api('/auth/guest', { method: 'POST', body: { device_id: deviceId }, auth: false });
    if (!res || !res.token) throw new Error('Guest start failed — backend unreachable.');
    await setToken(res.token);
    showWallets();
    // Switch to the wallets tab so the create/import UI is front-and-center.
    const walletsTab = document.querySelector('[data-tab="walletTab"]');
    if (walletsTab) walletsTab.click();
  } catch (err) {
    showError(err.message);
    if (btn) { btn.disabled = false; btn.textContent = '➕ Create / Import Wallet'; }
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
    // Watch-only enroll form handler
    const watchOnlyBtn = document.getElementById('watchOnlyAddBtn');
    if (watchOnlyBtn && !watchOnlyBtn._bound) {
      watchOnlyBtn._bound = true;
      watchOnlyBtn.addEventListener('click', async () => {
        const input = document.getElementById('watchOnlyAddress');
        const address = input?.value?.trim();
        if (!address || !address.startsWith('0x') || address.length !== 42) {
          alert('Enter a valid EVM address (0x...)');
          return;
        }
        try {
          await WalletAPI.createWatchOnlyWallet({ address, chain_id: 1, label: 'Watch: ' + address.slice(0, 8) });
          input.value = '';
          alert('Watch-only wallet added. You can track balances but cannot send.');
          renderWallets();
        } catch (e) {
          alert('Watch-only error: ' + e.message);
        }
      });
    }
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
          <a href="#" class="setup-lock" data-wallet-id="${escapeHtml(wallet.id)}" style="display:inline-block;margin-top:4px;color:var(--accent);font-size:12px;">Setup App Lock</a>
        </div>`
      )
      .join('');
    // Bind the per-wallet "Setup App Lock" links.
    list.querySelectorAll('.setup-lock').forEach((link) => {
      link.addEventListener('click', (e) => {
        e.preventDefault();
        handleSetupLock(link.getAttribute('data-wallet-id'));
      });
    });
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
  // maxFeeGwei / maxPriorityGwei are optional EIP-1559 overrides (gwei
  // strings); max_fee_gwei / max_priority_gwei are omitted when unset.
  sendTransaction: (walletId, password, to, amount, chainId, tokenAddress, unlockToken, maxFeeGwei, maxPriorityGwei) =>
    api('/send', { method: 'POST', body: { wallet_id: walletId, password, unlock_token: unlockToken, to, value: amount, chain_id: chainId, token_address: tokenAddress || undefined, max_fee_gwei: maxFeeGwei || undefined, max_priority_gwei: maxPriorityGwei || undefined } }),
  signMessage: (walletId, password, message) => api('/sign', { method: 'POST', body: { wallet_id: walletId, password, message } }),

  // Pre-sign dry-run — POST /simulate { chain_id, from, to, value?, data? }
  // -> { success, gas_estimate, will_revert, revert_reason?, ... }.
  simulate: (chainId, from, to, value, data) =>
    api('/simulate', { method: 'POST', body: { chain_id: chainId || 1, from, to, value: value || undefined, data: data || undefined } }),

  // ENS — GET /ens/resolve?name=alice.eth -> { name, address } (forward);
  // GET /ens/lookup?address=0x... -> { address, name } (reverse).
  resolveENS: (name) => api(`/ens/resolve?name=${encodeURIComponent(name)}`),
  lookupENS: (address) => api(`/ens/lookup?address=${encodeURIComponent(address)}`),

  // Guest auth — POST /auth/guest { device_id } -> { user_id, token, guest: true }.
  // Public (no auth). Provisions an anonymous guest account so the user can
  // Create/Import a wallet without registering. The returned token is persisted
  // the same way login's token is (chrome.storage.local 'token' via setToken),
  // mirroring how handleAuth stores the login token.
  guestAuth: async (deviceId) => {
    const res = await api('/auth/guest', { method: 'POST', body: { device_id: deviceId }, auth: false });
    if (res && res.token) await setToken(res.token);
    return res;
  },

  // Auto-send — POST /auto-send with the SAME body as /send, plus optional
  // ?master_wallet_id=<id> query. Same Bearer JWT auth as /send. Returns the
  // existing send response PLUS { auto_approved, auto_approval_reason }.
  autoSendTransaction: (walletId, password, to, amount, chainId, tokenAddress, masterWalletId, unlockToken, maxFeeGwei, maxPriorityGwei) => {
    const query = masterWalletId ? `?master_wallet_id=${encodeURIComponent(masterWalletId)}` : '';
    return api(`/auto-send${query}`, {
      method: 'POST',
      body: { wallet_id: walletId, password, unlock_token: unlockToken, to, value: amount, chain_id: chainId, token_address: tokenAddress || undefined, max_fee_gwei: maxFeeGwei || undefined, max_priority_gwei: maxPriorityGwei || undefined },
    });
  },

  // Transaction status — GET /transactions/:txHash?chain_id=N
  // -> { status, block_number?, confirmations? } (explorer proxy).
  getTransactionStatus: (txHash, chainId) =>
    api(`/transactions/${encodeURIComponent(txHash)}?chain_id=${chainId}`),

  // Gas / price / chains / status
  getGasPrice: (chainId) => api(`/gas?chain_id=${chainId}`),
  getTokenPrice: (symbol) => api(`/price?symbol=${encodeURIComponent(symbol)}`),
  getChains: () => api('/chains'),
  getNetworkStatus: (chainId) =>
    api(`/network-status?chain_id=${chainId}`),

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
  // P2P listings — backend route is /p2p/adverts (kept name for compatibility).
  getP2PListings: () => api('/p2p/adverts'),
  parsePaymentUri,

  // ---- Canonical backend fetcher additions (parity with web/desktop/ios/android/rust) ----

  // Auth — logout clears the persisted Bearer token.
  logout: async () => { await setToken(null); },

  // Decode the JWT locally; throw 'Not authenticated' when no token is stored.
  getProfile: async () => {
    const token = await getToken();
    if (!token) throw new Error('Not authenticated');
    const parts = token.split('.');
    if (parts.length < 2) return {};
    const payload = JSON.parse(
      decodeURIComponent(
        atob(parts[1].replace(/-/g, '+').replace(/_/g, '/'))
          .split('')
          .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
          .join('')
      )
    );
    return payload;
  },

  // Health — root /health route lives outside /api/v1, so fetch directly.
  health: () => fetch(`${API_BASE.replace(/\/api\/v1\/?$/, '')}/health`).then((r) => r.json()),

  // Import wallet (HD mnemonic).
  importWallet: ({ label, password, mnemonic, chainId, passphrase }) =>
    api('/wallets', { method: 'POST', body: { label, password, mnemonic, chain_id: chainId, passphrase } }),

  // NFT transfer.
  transferNFT: ({ walletId, password, to, tokenId, contractAddress, chainId }) =>
    api('/nft/transfer', { method: 'POST', body: { wallet_id: walletId, password, to, token_id: tokenId, contract_address: contractAddress, chain_id: chainId } }),

  // Transaction receipt (explorer proxy).
  getTransactionReceipt: (txHash, chainId) =>
    api(`/transactions/${encodeURIComponent(txHash)}?chain_id=${chainId}`),

  // Gas estimation.
  estimateGas: ({ from, to, value, data, chainId }) =>
    api('/gas/estimate', { method: 'POST', body: { from, to, value, data, chain_id: chainId } }),

  // Swap execution.
  executeSwap: ({ walletId, password, fromToken, toToken, fromAmount, chainId }) =>
    api('/swap/execute', { method: 'POST', body: { wallet_id: walletId, password, from_token: fromToken, to_token: toToken, from_amount: fromAmount, chain_id: chainId } }),

  // AMM quote + swap.
  getAmmQuote: ({ fromToken, toToken, fromAmount, chainId }) =>
    api(`/amm/quote?from_token=${encodeURIComponent(fromToken)}&to_token=${encodeURIComponent(toToken)}&from_amount=${encodeURIComponent(fromAmount)}&chain_id=${chainId}`),
  ammSwap: ({ walletId, password, fromToken, toToken, fromAmount, chainId }) =>
    api('/amm/swap', { method: 'POST', body: { wallet_id: walletId, password, from_token: fromToken, to_token: toToken, from_amount: fromAmount, chain_id: chainId } }),

  // Staking — unstake + claim.
  unstake: ({ walletId, password, asset, amount, chainId }) =>
    api('/staking/unstake', { method: 'POST', body: { wallet_id: walletId, password, asset, amount, chain_id: chainId } }),
  claim: ({ walletId, password, asset, chainId }) =>
    api('/staking/claim', { method: 'POST', body: { wallet_id: walletId, password, asset, chain_id: chainId } }),

  // Crypto cards.
  getCryptoCardBalance: (cardId) => api(`/cards/${encodeURIComponent(cardId)}/balance`),
  getCardTransactions: (cardId) => api(`/cards/${encodeURIComponent(cardId)}/transactions`),

  // P2P adverts — GET /p2p/adverts.
  getP2PAdverts: () => api('/p2p/adverts'),

  // Non-EVM address derivation + signing + sending. The backend decrypts the
  // stored seed with wallet_id + password — the raw seed NEVER leaves the
  // server (POST /non_evm/{address,sign,send} in go/wallet_api).
  nonEvmAddress: ({ walletId, password, chainType, prefix }) =>
    api('/non_evm/address', { method: 'POST', body: { wallet_id: walletId, password, chain_type: chainType, prefix: prefix || undefined } }),
  nonEvmSign: ({ walletId, password, chainType, message }) =>
    api('/non_evm/sign', { method: 'POST', body: { wallet_id: walletId, password, chain_type: chainType, message } }),
  nonEvmSend: ({ walletId, password, chainType, bitcoinInputs, bitcoinOutputs, cosmosSignDoc }) =>
    api('/non_evm/send', { method: 'POST', body: { wallet_id: walletId, password, chain_type: chainType, bitcoin_inputs: bitcoinInputs, bitcoin_outputs: bitcoinOutputs, cosmos_sign_doc: cosmosSignDoc } }),

  // Address book.
  getAddressBookContacts: () => api('/address-book/contacts'),
  addContact: ({ name, address, chainId }) =>
    api('/address-book/contacts', { method: 'POST', body: { name, address, chain_id: chainId } }),
  updateContact: (id, { name, address, chainId }) =>
    api(`/address-book/contacts/${encodeURIComponent(id)}`, { method: 'PUT', body: { name, address, chain_id: chainId } }),
  deleteContact: (id) =>
    api(`/address-book/contacts/${encodeURIComponent(id)}`, { method: 'DELETE' }),

  // Devices.
  getDevices: () => api('/devices'),
  registerDevice: ({ name, deviceType }) =>
    api('/devices', { method: 'POST', body: { name, device_type: deviceType } }),
  syncDevice: (deviceId) => api(`/devices/${encodeURIComponent(deviceId)}/sync`, { method: 'POST' }),
  deleteDevice: (deviceId) => api(`/devices/${encodeURIComponent(deviceId)}`, { method: 'DELETE' }),

  // Token approvals.
  getApprovals: (address, chainId) =>
    api(`/approvals?address=${encodeURIComponent(address)}&chain_id=${chainId}`),
  revokeApproval: ({ approvalId }) =>
    api(`/approvals/${encodeURIComponent(approvalId)}`, { method: 'DELETE' }),

  // Keystore export / import.
  exportKeystore: ({ walletId, password }) =>
    api('/keystore/export', { method: 'POST', body: { wallet_id: walletId, password } }),
  importKeystore: ({ keystore, password, label }) =>
    api('/keystore/import', { method: 'POST', body: { keystore, password, label } }),

  // Encrypted seed export / import.
  exportEncryptedSeed: (walletId, password) =>
    api(`/wallets/${encodeURIComponent(walletId)}/export-encrypted-seed`, { method: 'POST', body: { password } }),
  importEncryptedSeed: ({ encryptedSeed, password, label }) =>
    api('/wallets/import-encrypted-seed', { method: 'POST', body: { encrypted_seed: encryptedSeed, password, label } }),

  // Security.
  checkUrl: (url) => api(`/security/check-url?url=${encodeURIComponent(url)}`),
  checkAddress: (address) => api(`/security/check-address?address=${encodeURIComponent(address)}`),
  securityScan: (target) => api('/security/scan', { method: 'POST', body: { target } }),

  // Lending.
  getLendingMarkets: () => api('/lending/markets'),
  getLendingPositions: () => api('/lending/positions'),
  lendingSupply: ({ walletId, password, asset, amount, chainId }) =>
    api('/lending/supply', { method: 'POST', body: { wallet_id: walletId, password, asset, amount, chain_id: chainId } }),
  lendingBorrow: ({ walletId, password, asset, amount, chainId }) =>
    api('/lending/borrow', { method: 'POST', body: { wallet_id: walletId, password, asset, amount, chain_id: chainId } }),
  lendingWithdraw: ({ walletId, password, asset, amount, chainId }) =>
    api('/lending/withdraw', { method: 'POST', body: { wallet_id: walletId, password, asset, amount, chain_id: chainId } }),
  lendingRepay: ({ walletId, password, asset, amount, chainId }) =>
    api('/lending/repay', { method: 'POST', body: { wallet_id: walletId, password, asset, amount, chain_id: chainId } }),

  // Copy trading.
  getCopyTraders: () => api('/copytrading/traders'),
  followTrader: ({ traderId, allocation }) =>
    api('/copytrading/follow', { method: 'POST', body: { trader_id: traderId, allocation } }),
  stopCopyTrader: (copierId) => api(`/copytrading/copiers/${encodeURIComponent(copierId)}/stop`, { method: 'POST' }),
  getCopySignals: () => api('/copytrading/signals'),

  // DAO.
  getDaoProposals: () => api('/dao/proposals'),
  createDaoProposal: ({ title, description }) =>
    api('/dao/proposals', { method: 'POST', body: { title, description } }),
  voteDaoProposal: ({ proposalId, support }) =>
    api(`/dao/proposals/${encodeURIComponent(proposalId)}/vote`, { method: 'POST', body: { support } }),
  getDaoDelegates: () => api('/dao/delegates'),

  // Perpetuals.
  getPerpetualPositions: () => api('/perpetual/positions'),
  createPerpetualPosition: ({ pair, side, size, leverage, chainId }) =>
    api('/perpetual/positions', { method: 'POST', body: { pair, side, size, leverage, chain_id: chainId } }),
  closePerpetualPosition: (positionId) =>
    api(`/perpetual/positions/${encodeURIComponent(positionId)}/close`, { method: 'POST' }),

  // Margin.
  getMarginPositions: () => api('/margin/positions'),
  createMarginPosition: ({ pair, side, size, leverage, chainId }) =>
    api('/margin/positions', { method: 'POST', body: { pair, side, size, leverage, chain_id: chainId } }),
  closeMarginPosition: (positionId) =>
    api(`/margin/positions/${encodeURIComponent(positionId)}/close`, { method: 'POST' }),

  // Prediction markets.
  getPredictionMarkets: () => api('/prediction/markets'),
  placePredictionBet: ({ marketId, side, amount }) =>
    api(`/prediction/markets/${encodeURIComponent(marketId)}/bet`, { method: 'POST', body: { side, amount } }),

  // Launchpool.
  getLaunchpool: () => api('/launchpool'),
  getLaunchpoolStakes: () => api('/launchpool/stakes'),
  launchpoolStake: ({ walletId, password, amount }) =>
    api('/launchpool/stake', { method: 'POST', body: { wallet_id: walletId, password, amount } }),
  launchpoolUnstake: ({ walletId, password, amount }) =>
    api('/launchpool/unstake', { method: 'POST', body: { wallet_id: walletId, password, amount } }),

  // Token sales.
  getTokenSales: () => api('/token-sales'),
  participateTokenSale: ({ saleId, amount }) =>
    api(`/token-sales/${encodeURIComponent(saleId)}/participate`, { method: 'POST', body: { amount } }),

  // DApps.
  getDapps: () => api('/dapps'),
  getDappCategories: () => api('/dapps/categories'),

  // Chart history.
  getChartHistory: ({ token, days }) =>
    api(`/chart/history?token=${encodeURIComponent(token)}&days=${encodeURIComponent(days)}`),

  // DeFi protocols.
  getDefiProtocols: () => api('/defi/protocols'),

  // Cross-chain bridge (proxied bridge_service).
  getBridgeRoutes: () => api('/bridge/routes'),
  getBridgeQuote: ({ fromChain, toChain, token, amount }) =>
    api('/bridge/quote', { method: 'POST', body: { fromChain, toChain, token, amount } }),
  executeBridgeTransfer: (body) => api('/bridge/transfer', { method: 'POST', body }),
  getBridgeHistory: () => api('/bridge/history'),

  // Multisig (proxied to MasterWallet via /wallet/multisig/*).
  listMultisigWallets: () => api('/wallet/multisig/wallets'),
  createMultisigWallet: ({ name, owners, threshold, chain_id }) =>
    api('/wallet/multisig/wallets', { method: 'POST', body: { name, owners, threshold, chain_id } }),
  listMultisigTransactions: (walletId) => api(`/wallet/multisig/wallets/${encodeURIComponent(walletId)}/transactions`),
  createMultisigTransaction: (walletId, { to_address, value, data }) =>
    api(`/wallet/multisig/wallets/${encodeURIComponent(walletId)}/transactions`, { method: 'POST', body: { to_address, value, data } }),
  signMultisigTransaction: (txId) =>
    api(`/wallet/multisig/transactions/${encodeURIComponent(txId)}/sign`, { method: 'POST', body: {} }),
  executeMultisigTransaction: (txId) =>
    api(`/wallet/multisig/transactions/${encodeURIComponent(txId)}/execute`, { method: 'POST', body: {} }),

  // Crypto card (proxied card_service :8457).
  getCardBalance: () => api('/cards/default/balance'),
  getCardRates: () => api('/cards/rates'),
  getCardTransactions: () => api('/cards/default/transactions'),

  // Price alerts.
  getPriceAlerts: () => api('/price-alerts'),
  createPriceAlert: ({ symbol, target_price, direction }) =>
    api('/price-alerts', { method: 'POST', body: { symbol, target_price, direction } }),
  deletePriceAlert: (id) => api(`/price-alerts/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  updatePriceAlert: (id, body) => api(`/price-alerts/${encodeURIComponent(id)}`, { method: 'PUT', body }),
  createWatchOnlyWallet: ({ address, chain_id, label }) =>
    api('/wallets/watch-only', { method: 'POST', body: { address, chain_id, label } }),

  // ---- DAO / Launchpool / Token Sales / Trading / Prediction / Copy Trading / Fees ----
  getDaoProposals: () => api('/dao/proposals'),
  createDaoProposal: (title, description) =>
    api('/dao/proposals', { method: 'POST', body: { title, description } }),
  voteDaoProposal: (proposalId, support) =>
    api(`/dao/proposals/${encodeURIComponent(proposalId)}/vote`, { method: 'POST', body: { support } }),
  getDaoDelegates: () => api('/dao/delegates'),
  getLaunchpool: () => api('/launchpool'),
  getLaunchpoolStakes: () => api('/launchpool/stakes'),
  launchpoolStake: (walletId, password, amount) =>
    api('/launchpool/stake', { method: 'POST', body: { wallet_id: walletId, password, amount } }),
  launchpoolUnstake: (walletId, password, amount) =>
    api('/launchpool/unstake', { method: 'POST', body: { wallet_id: walletId, password, amount } }),
  getTokenSales: () => api('/token-sales'),
  participateTokenSale: (saleId, amount) =>
    api(`/token-sales/${encodeURIComponent(saleId)}/participate`, { method: 'POST', body: { amount } }),
  getPerpetualPositions: () => api('/perpetual/positions'),
  createPerpetualPosition: (pair, side, size, leverage, chainId = 1) =>
    api('/perpetual/positions', { method: 'POST', body: { pair, side, size, leverage, chain_id: chainId } }),
  closePerpetualPosition: (positionId) =>
    api(`/perpetual/positions/${encodeURIComponent(positionId)}/close`, { method: 'POST', body: {} }),
  getMarginPositions: () => api('/margin/positions'),
  createMarginPosition: (pair, side, size, leverage, chainId = 1) =>
    api('/margin/positions', { method: 'POST', body: { pair, side, size, leverage, chain_id: chainId } }),
  closeMarginPosition: (positionId) =>
    api(`/margin/positions/${encodeURIComponent(positionId)}/close`, { method: 'POST', body: {} }),
  getPredictionMarkets: () => api('/prediction/markets'),
  placePredictionBet: (marketId, side, amount) =>
    api(`/prediction/markets/${encodeURIComponent(marketId)}/bet`, { method: 'POST', body: { side, amount } }),
  getCopyTraders: () => api('/copytrading/traders'),
  followTrader: (traderId, allocation) =>
    api('/copytrading/follow', { method: 'POST', body: { trader_id: traderId, allocation: allocation || undefined } }),
  stopCopyTrader: (copierId) =>
    api(`/copytrading/copiers/${encodeURIComponent(copierId)}/stop`, { method: 'POST', body: {} }),
  getPublicFees: () => api('/public/fees'),
  getPublicFeeTransactions: () => api('/public/fees/transactions'),

  // ---- Token registry + trading terminal (public) ----
  getTokenRegistry: (chainId) =>
    api(chainId ? `/tokens/registry?chain_id=${chainId}` : '/tokens/registry'),
  getTerminalKline: (symbol, days = 1) =>
    api(`/terminal/kline/${encodeURIComponent(symbol)}?days=${days}`),
  getTerminalTicker: (symbol) =>
    api(`/terminal/ticker/${encodeURIComponent(symbol)}`),

  // Passkey wallet creation — POST /passkey/wallet.
  passkeyCreateWallet: ({ label, chain_id, account_index, entropy_bits, credential_id, public_key, sign_count, attestation }) =>
    api('/passkey/wallet', { method: 'POST', body: { label, chain_id, account_index, entropy_bits, credential_id, public_key, sign_count, attestation } }),

  // Wallet lock — POST /wallets/:id/lock.
  setupLock: (walletId, params) =>
    api(`/wallets/${encodeURIComponent(walletId)}/lock`, { method: 'POST', body: params }),

  // Wallet unlock — POST /wallets/:id/unlock.
  unlockWallet: (walletId, params) =>
    api(`/wallets/${encodeURIComponent(walletId)}/unlock`, { method: 'POST', body: params }),

  // KYC — status, register, submit, document (multipart), session.
  getKycStatus: (userId) => api(`/kyc/status?user_id=${encodeURIComponent(userId)}`),
  registerKyc: (body) => api('/kyc/register', { method: 'POST', body }),
  submitKyc: (body) => api('/kyc/submit', { method: 'POST', body }),
  submitKycDocument: async (formData) => {
    const token = await getToken();
    const res = await fetch(`${API_BASE}/kyc/document`, {
      method: 'POST',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: formData,
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.error || `Request failed (${res.status})`);
    return data;
  },
  getKycSession: (sessionId) => api(`/kyc/session/${encodeURIComponent(sessionId)}`),

  // P2P orders — POST /p2p/orders (KYC-gated; 403 {kyc_required:true}).
  createP2POrder: (body) => api('/p2p/orders', { method: 'POST', body }),

  // Bridge (proxied bridge_service :8007)
  getBridges: () => api('/bridge/routes'),
  getBridgeQuote: (params) => api('/bridge/quote', { method: 'POST', body: JSON.stringify(params) }),
  initiateBridgeTransfer: (body) => api('/bridge/transfer', { method: 'POST', body: JSON.stringify(body) }),
  getBridgeTxStatus: (txId) => api(`/bridge/tx/${txId}`),
  getBridgeHistory: () => api('/bridge/history'),

  // dApp browser / WalletConnect (proxied dapp_browser :8083)
  getDappPairings: () => api('/dapp/pairings'),
  createDappPairing: (body) => api('/dapp/pairings', { method: 'POST', body: JSON.stringify(body) }),
  approveDappPairing: (topic, namespaces) => api(`/dapp/pairings/${topic}/approve`, { method: 'POST', body: JSON.stringify(namespaces ? { namespaces } : {}) }),
  rejectDappPairing: (topic) => api(`/dapp/pairings/${topic}/reject`, { method: 'POST', body: '{}' }),
  getDappSessions: () => api('/dapp/sessions'),
  disconnectDappSession: (topic) => api(`/dapp/sessions/${topic}`, { method: 'DELETE' }),
  sendDappRequest: (topic, body) => api(`/dapp/sessions/${topic}/request`, { method: 'POST', body: JSON.stringify(body) }),
  getDappRequests: (topic) => api(`/dapp/sessions/${topic}/request`),
  respondToDappRequest: (topic, requestId, body) =>
    api(`/dapp/sessions/${topic}/request/${requestId}/respond`, { method: 'POST', body: JSON.stringify(body) }),

  // Networks alias.
  getNetworks: () => WalletAPI.getChains(),

  // Aggregate balances — list wallets then fan out getBalance per wallet.
  getBalances: async () => {
    const { wallets } = await WalletAPI.listWallets();
    const results = await Promise.all(
      (wallets || []).map((w) =>
        WalletAPI.getBalance(w.address, w.chain_id)
          .then((balance) => ({ wallet: w, balance }))
          .catch(() => ({ wallet: w, balance: null }))
      )
    );
    return results;
  },
};

// ---------------------------------------------------------------------------
// Shared UI state + tab navigation
// ---------------------------------------------------------------------------
const state = { wallets: [], activeWallet: null };

function switchTab(tab) {
  ['walletTab', 'sendTab', 'convertTab', 'stakingTab', 'fiatTab', 'qrTab', 'kycTab', 'defiTab', 'dappsTab', 'nftsTab', 'bridgeTab', 'txTab', 'approvalsTab', 'contactsTab', 'devicesTab', 'alertsTab', 'securityTab', 'terminalTab', 'cardsTab', 'multisigTab', 'daoTab', 'launchpoolTab', 'tokenSalesTab', 'tradingTab', 'predictionTab', 'copyTradingTab', 'feesTab', 'nonEvmTab', 'settingsTab'].forEach((t) => {
    const el = document.getElementById(t);
    if (el) el.classList.add('hidden');
  });
  const target = document.getElementById(tab + 'Tab') || document.getElementById(tab);
  if (target) target.classList.remove('hidden');
  if (tab === 'wallet') loadWallets();
  if (tab === 'convert') loadConvert();
  if (tab === 'staking') loadStaking();
  if (tab === 'fiat') loadFiatProviders();
  if (tab === 'kyc') loadKyc();
  if (tab === 'defi') loadDefi();
  if (tab === 'dapps') loadDapps();
  if (tab === 'nfts') loadNfts();
  if (tab === 'tx') loadTxHistory();
  if (tab === 'approvals') loadApprovals();
  if (tab === 'contacts') loadContacts();
  if (tab === 'devices') loadDevices();
  if (tab === 'alerts') loadAlerts();
  if (tab === 'terminal') loadTerminal();
  if (tab === 'cards') loadCards();
  if (tab === 'multisig') loadMultisigWallets();
  if (tab === 'dao') loadDao();
  if (tab === 'launchpool') loadLaunchpool();
  if (tab === 'tokenSales') loadTokenSales();
  if (tab === 'trading') loadTradingPositions();
  if (tab === 'prediction') loadPredictionMarkets();
  if (tab === 'copyTrading') loadCopyTraders();
  if (tab === 'fees') loadFees();
  if (tab === 'nonEvm') loadNonEvm();
  if (tab === 'settings') loadSettings();
}

// ---- Loaders for the NFT / History / Approvals / Contacts / Devices / Alerts tabs ----

function renderList(el, items, renderRow, emptyMsg) {
  el.innerHTML = '';
  if (!items.length) {
    const row = document.createElement('div');
    row.style.cssText = 'color:var(--text-secondary);padding:6px 0;';
    row.textContent = emptyMsg;
    el.appendChild(row);
    return;
  }
  items.forEach((item) => {
    const row = document.createElement('div');
    row.style.cssText = 'padding:6px 4px;border-bottom:1px solid var(--border);';
    renderRow(row, item);
    el.appendChild(row);
  });
}

// ---- Non-EVM tab (Solana / Bitcoin / Cosmos via POST /non_evm/*) ----

function loadNonEvm() {
  const sel = document.getElementById('nonEvmWallet');
  if (!sel) return;
  sel.innerHTML = '';
  state.wallets.forEach((w) => {
    const opt = document.createElement('option');
    opt.value = w.id;
    opt.textContent = `${w.label || 'Wallet'} (${(w.address || '').slice(0, 10)}…)`;
    sel.appendChild(opt);
  });
  if (state.activeWallet) sel.value = state.activeWallet.id;
}

function nonEvmSelection() {
  const walletId = document.getElementById('nonEvmWallet').value;
  const chainType = document.getElementById('nonEvmChain').value;
  const password = document.getElementById('nonEvmPassword').value;
  const status = document.getElementById('nonEvmStatus');
  if (!walletId || !password) {
    status.textContent = 'Select a wallet and enter its password.';
    return null;
  }
  status.textContent = '';
  return { walletId, chainType, password, status };
}

async function handleNonEvmDerive() {
  const sel = nonEvmSelection();
  if (!sel) return;
  const prefix = document.getElementById('nonEvmPrefix').value.trim();
  sel.status.textContent = 'Deriving address…';
  try {
    const res = await WalletAPI.nonEvmAddress({ walletId: sel.walletId, password: sel.password, chainType: sel.chainType, prefix: prefix || undefined });
    document.getElementById('nonEvmAddressResult').textContent = res.address || JSON.stringify(res);
    sel.status.textContent = '✓ Address derived.';
  } catch (e) { sel.status.textContent = e.message; }
}

async function handleNonEvmSign() {
  const sel = nonEvmSelection();
  if (!sel) return;
  const message = document.getElementById('nonEvmMessage').value;
  if (!message) { sel.status.textContent = 'Enter a message to sign.'; return; }
  sel.status.textContent = 'Signing…';
  try {
    const res = await WalletAPI.nonEvmSign({ walletId: sel.walletId, password: sel.password, chainType: sel.chainType, message });
    document.getElementById('nonEvmSigResult').textContent = res.signature || JSON.stringify(res);
    sel.status.textContent = '✓ Message signed.';
  } catch (e) { sel.status.textContent = e.message; }
}

async function handleNonEvmSend() {
  const sel = nonEvmSelection();
  if (!sel) return;
  sel.status.textContent = 'Building + signing transaction…';
  try {
    let body = { walletId: sel.walletId, password: sel.password, chainType: sel.chainType };
    if (sel.chainType === 'bitcoin') {
      body.bitcoinInputs = JSON.parse(document.getElementById('nonEvmBtcInputs').value || '[]');
      body.bitcoinOutputs = JSON.parse(document.getElementById('nonEvmBtcOutputs').value || '[]');
    } else if (sel.chainType === 'cosmos') {
      body.cosmosSignDoc = JSON.parse(document.getElementById('nonEvmCosmosDoc').value || '{}');
    } else {
      sel.status.textContent = 'Solana sends are built via /non_evm/sign on the transaction message; broadcast via the chain RPC.';
      return;
    }
    const res = await WalletAPI.nonEvmSend(body);
    document.getElementById('nonEvmSendResult').textContent = res.raw_tx || res.signature || JSON.stringify(res);
    sel.status.textContent = '✓ Transaction signed — submitted to the blockchain network relay of the selected chain.';
  } catch (e) { sel.status.textContent = e.message; }
}

// ---- Loaders/handlers for DAO / Launchpool / Token Sales / Trading / Prediction / Copy Trading / Fees tabs ----

function setStatus(id, msg) {
  const el = document.getElementById(id);
  if (el) el.textContent = msg;
}

async function loadDao() {
  const el = document.getElementById('daoProposals');
  if (!el) return;
  try {
    const res = await WalletAPI.getDaoProposals();
    const list = Array.isArray(res) ? res : (res.data || res.proposals || []);
    renderList(el, list, (row, p) => {
      row.textContent = `• ${p.id || '?'}: ${p.title || '?'} — yes:${p.votes_for ?? p.yes ?? 0} no:${p.votes_against ?? p.no ?? 0}`;
    }, 'No active proposals.');
  } catch (e) {
    renderList(el, [], null, 'DAO data unavailable.');
  }
}

async function handleDaoCreate() {
  const title = (document.getElementById('daoTitle')?.value || '').trim();
  const description = (document.getElementById('daoDesc')?.value || '').trim();
  if (!title || !description) { setStatus('daoStatus', 'Enter title and description.'); return; }
  setStatus('daoStatus', 'Submitting…');
  try {
    const res = await WalletAPI.createDaoProposal(title, description);
    setStatus('daoStatus', `Proposal created: ${res.id || res.proposal_id || 'ok'}`);
    loadDao();
  } catch (e) {
    setStatus('daoStatus', `Create failed: ${e.message}`);
  }
}

async function handleDaoVote(support) {
  const id = (document.getElementById('daoVoteId')?.value || '').trim();
  if (!id) { setStatus('daoStatus', 'Enter a proposal ID.'); return; }
  setStatus('daoStatus', 'Voting…');
  try {
    await WalletAPI.voteDaoProposal(id, support);
    setStatus('daoStatus', 'Vote submitted to the blockchain network.');
    loadDao();
  } catch (e) {
    setStatus('daoStatus', `Vote failed: ${e.message}`);
  }
}

async function loadLaunchpool() {
  const info = document.getElementById('launchpoolInfo');
  const stakesEl = document.getElementById('launchpoolStakes');
  try {
    const pool = await WalletAPI.getLaunchpool();
    if (info) info.textContent = `Pool: ${pool.token || pool.asset || '?'} | APY: ${pool.apy ?? '?'}% | TVL: ${pool.tvl ?? pool.total_staked ?? '?'}`;
  } catch (e) {
    if (info) info.textContent = 'Launchpool data unavailable.';
  }
  try {
    const res = await WalletAPI.getLaunchpoolStakes();
    const list = Array.isArray(res) ? res : (res.data || res.stakes || []);
    if (stakesEl) renderList(stakesEl, list, (row, s) => {
      row.textContent = `• ${s.id || '?'}: ${s.amount || '?'} ${s.token || ''} (${s.status || '?'})`;
    }, 'No active stakes.');
  } catch (e) {
    if (stakesEl) renderList(stakesEl, [], null, 'No active stakes.');
  }
}

async function handleLaunchpool(stake) {
  const amount = (document.getElementById('lpAmount')?.value || '').trim();
  const password = document.getElementById('lpPassword')?.value || '';
  if (!amount || !password) { setStatus('lpStatus', 'Enter amount and wallet password.'); return; }
  const w = activeWallet();
  if (!w) { setStatus('lpStatus', 'No wallet available.'); return; }
  setStatus('lpStatus', 'Submitting…');
  try {
    const res = stake
      ? await WalletAPI.launchpoolStake(w.id, password, amount)
      : await WalletAPI.launchpoolUnstake(w.id, password, amount);
    const tx = res.tx_hash || '';
    setStatus('lpStatus', tx ? `Transaction submitted to the blockchain network: ${tx}` : `${stake ? 'Stake' : 'Unstake'} submitted.`);
    loadLaunchpool();
  } catch (e) {
    setStatus('lpStatus', `Failed: ${e.message}`);
  }
}

async function loadTokenSales() {
  const el = document.getElementById('tokenSalesList');
  if (!el) return;
  try {
    const res = await WalletAPI.getTokenSales();
    const list = Array.isArray(res) ? res : (res.sales || res.data || []);
    renderList(el, list, (row, s) => {
      row.textContent = `• ${s.id || '?'}: ${s.name || s.token || '?'} @ ${s.price ?? '?'} (${s.status || '?'})`;
    }, 'No active token sales.');
  } catch (e) {
    renderList(el, [], null, 'Token sales unavailable.');
  }
}

async function handleTokenSaleParticipate() {
  const saleId = (document.getElementById('tsSaleId')?.value || '').trim();
  const amount = (document.getElementById('tsAmount')?.value || '').trim();
  if (!saleId || !amount) { setStatus('tsStatus', 'Enter sale ID and amount.'); return; }
  setStatus('tsStatus', 'Submitting…');
  try {
    const res = await WalletAPI.participateTokenSale(saleId, amount);
    const tx = res.tx_hash || '';
    setStatus('tsStatus', tx ? `Transaction submitted to the blockchain network: ${tx}` : 'Participation submitted.');
    loadTokenSales();
  } catch (e) {
    setStatus('tsStatus', `Failed: ${e.message}`);
  }
}

async function loadTradingPositions() {
  const el = document.getElementById('tradingPositions');
  if (!el) return;
  try {
    const perps = await WalletAPI.getPerpetualPositions();
    const margins = await WalletAPI.getMarginPositions();
    const perpList = Array.isArray(perps) ? perps : (perps.positions || perps.data || []);
    const marginList = Array.isArray(margins) ? margins : (margins.positions || margins.data || []);
    const all = perpList.map(p => ({ ...p, _m: 'perp' })).concat(marginList.map(p => ({ ...p, _m: 'margin' })));
    renderList(el, all, (row, p) => {
      row.textContent = `• [${p._m}] ${p.id || '?'}: ${p.pair || '?'} ${p.side || '?'} size:${p.size ?? '?'} ${p.leverage ?? '?'}x pnl:${p.pnl ?? p.unrealized_pnl ?? '?'}`;
    }, 'No open positions.');
  } catch (e) {
    renderList(el, [], null, 'Positions unavailable.');
  }
}

async function handleTradeOpen(perp) {
  const pair = (document.getElementById('tradePair')?.value || '').trim();
  const side = (document.getElementById('tradeSide')?.value || '').trim();
  const size = (document.getElementById('tradeSize')?.value || '').trim();
  const leverage = parseInt((document.getElementById('tradeLeverage')?.value || '1').trim(), 10) || 1;
  if (!pair || !side || !size) { setStatus('tradeStatus', 'Enter pair, side and size.'); return; }
  setStatus('tradeStatus', 'Opening position…');
  try {
    const res = perp
      ? await WalletAPI.createPerpetualPosition(pair, side, size, leverage)
      : await WalletAPI.createMarginPosition(pair, side, size, leverage);
    const tx = res.tx_hash || '';
    setStatus('tradeStatus', tx ? `Transaction submitted to the blockchain network: ${tx}` : `Position opened: ${res.id || 'ok'}`);
    loadTradingPositions();
  } catch (e) {
    setStatus('tradeStatus', `Open failed: ${e.message}`);
  }
}

async function handleTradeClose(perp) {
  const id = (document.getElementById('tradeCloseId')?.value || '').trim();
  if (!id) { setStatus('tradeStatus', 'Enter a position ID.'); return; }
  setStatus('tradeStatus', 'Closing position…');
  try {
    if (perp) await WalletAPI.closePerpetualPosition(id);
    else await WalletAPI.closeMarginPosition(id);
    setStatus('tradeStatus', 'Position close submitted.');
    loadTradingPositions();
  } catch (e) {
    setStatus('tradeStatus', `Close failed: ${e.message}`);
  }
}

async function loadPredictionMarkets() {
  const el = document.getElementById('predictionMarkets');
  if (!el) return;
  try {
    const res = await WalletAPI.getPredictionMarkets();
    const list = Array.isArray(res) ? res : (res.markets || res.data || []);
    renderList(el, list, (row, m) => {
      row.textContent = `• ${m.id || '?'}: ${m.question || m.title || '?'} (${m.status || '?'})`;
    }, 'No active markets.');
  } catch (e) {
    renderList(el, [], null, 'Markets unavailable.');
  }
}

async function handlePredictionBet() {
  const marketId = (document.getElementById('predMarketId')?.value || '').trim();
  const side = (document.getElementById('predSide')?.value || '').trim();
  const amount = (document.getElementById('predAmount')?.value || '').trim();
  if (!marketId || !side || !amount) { setStatus('predStatus', 'Enter market ID, side and amount.'); return; }
  setStatus('predStatus', 'Placing bet…');
  try {
    await WalletAPI.placePredictionBet(marketId, side, amount);
    setStatus('predStatus', 'Bet submitted.');
    loadPredictionMarkets();
  } catch (e) {
    setStatus('predStatus', `Bet failed: ${e.message}`);
  }
}

async function loadCopyTraders() {
  const el = document.getElementById('copyTraders');
  if (!el) return;
  try {
    const res = await WalletAPI.getCopyTraders();
    const list = Array.isArray(res) ? res : (res.traders || res.data || []);
    renderList(el, list, (row, t) => {
      row.textContent = `• ${t.id || t.trader_id || '?'}: ${t.name || t.address || '?'} roi:${t.roi ?? '?'}%`;
    }, 'No traders available.');
  } catch (e) {
    renderList(el, [], null, 'Traders unavailable.');
  }
}

async function handleCopyFollow() {
  const traderId = (document.getElementById('copyTraderId')?.value || '').trim();
  const allocation = (document.getElementById('copyAllocation')?.value || '').trim();
  if (!traderId) { setStatus('copyStatus', 'Enter a trader ID.'); return; }
  setStatus('copyStatus', 'Following…');
  try {
    await WalletAPI.followTrader(traderId, allocation || null);
    setStatus('copyStatus', `Now copying trader ${traderId}.`);
    loadCopyTraders();
  } catch (e) {
    setStatus('copyStatus', `Follow failed: ${e.message}`);
  }
}

async function handleCopyStop() {
  const copierId = (document.getElementById('copyCopierId')?.value || '').trim();
  if (!copierId) { setStatus('copyStatus', 'Enter a copier ID.'); return; }
  setStatus('copyStatus', 'Stopping…');
  try {
    await WalletAPI.stopCopyTrader(copierId);
    setStatus('copyStatus', 'Stopped copying.');
    loadCopyTraders();
  } catch (e) {
    setStatus('copyStatus', `Stop failed: ${e.message}`);
  }
}

async function loadFees() {
  const tiersEl = document.getElementById('feesTiers');
  const txEl = document.getElementById('feesTransactions');
  try {
    const res = await WalletAPI.getPublicFees();
    const tiers = Array.isArray(res) ? res : (res.fees || res.data || []);
    if (tiersEl) renderList(tiersEl, tiers, (row, t) => {
      row.textContent = `• ${t.name || t.tier || '?'}: ${t.rate_bps ?? t.rate ?? '?'} bps`;
    }, 'No fee tiers configured.');
  } catch (e) {
    if (tiersEl) renderList(tiersEl, [], null, 'Fee schedule unavailable.');
  }
  try {
    const res = await WalletAPI.getPublicFeeTransactions();
    const txs = Array.isArray(res) ? res : (res.transactions || res.data || []);
    if (txEl) renderList(txEl, txs.slice(0, 25), (row, t) => {
      row.textContent = `• ${String(t.tx_hash || '').slice(0, 14)}… ${t.fee_amount ?? t.amount ?? '?'} ${t.token || ''}`;
    }, 'No settled fee transactions.');
  } catch (e) {
    if (txEl) renderList(txEl, [], null, 'No settled fee transactions.');
  }
}

async function loadNfts() {
  const el = document.getElementById('nftList');
  if (!el) return;
  const w = activeWallet();
  if (!w) { el.innerHTML = '<div style="color:var(--text-secondary);">No wallet.</div>'; return; }
  try {
    const res = await WalletAPI.getNFTs(w.address, w.chain_id);
    const list = Array.isArray(res) ? res : (res.nfts || []);
    renderList(el, list, (row, n) => {
      row.textContent = `${n.name || n.collection || 'NFT'} #${n.token_id ?? ''} (${n.contract_address || n.contract || ''})`;
    }, 'No NFTs on this chain.');
  } catch (e) {
    el.innerHTML = '<div style="color:var(--text-secondary);">NFTs unavailable.</div>';
  }
}

async function handleNftTransfer() {
  const status = document.getElementById('nftStatus');
  const w = activeWallet();
  if (!w) { status.textContent = 'No wallet.'; return; }
  const contractAddress = document.getElementById('nftContract').value.trim();
  const tokenId = document.getElementById('nftTokenId').value.trim();
  const to = document.getElementById('nftTo').value.trim();
  const password = document.getElementById('nftPassword').value;
  if (!contractAddress || !tokenId || !to || !password) { status.textContent = 'All fields required.'; return; }
  status.textContent = 'Submitting…';
  try {
    const res = await WalletAPI.transferNFT({ walletId: w.id, password, to, tokenId, contractAddress, chainId: w.chain_id });
    status.textContent = 'NFT transfer submitted to the blockchain network: ' + (res.transaction_hash || res.tx_hash || JSON.stringify(res));
  } catch (e) {
    status.textContent = 'Transfer failed: ' + (e.message || 'error');
  }
}

async function handleBridgeQuote() {
  const out = document.getElementById('bridgeQuoteResult');
  const fromChain = parseInt(document.getElementById('bridgeFromChain').value, 10);
  const toChain = parseInt(document.getElementById('bridgeToChain').value, 10);
  const token = document.getElementById('bridgeToken').value.trim();
  const amount = document.getElementById('bridgeAmount').value.trim();
  if (!fromChain || !toChain || !token || !amount) { out.textContent = 'All fields required.'; return; }
  out.textContent = 'Quoting…';
  try {
    const res = await WalletAPI.getBridgeQuote({ fromChain, toChain, token, amount });
    out.textContent = JSON.stringify(res);
  } catch (e) {
    out.textContent = 'Quote failed: ' + (e.message || 'error');
  }
}

async function handleBridgeTransfer() {
  const status = document.getElementById('bridgeStatus');
  const w = activeWallet();
  if (!w) { status.textContent = 'No wallet.'; return; }
  const fromChain = parseInt(document.getElementById('bridgeFromChain').value, 10);
  const toChain = parseInt(document.getElementById('bridgeToChain').value, 10);
  const token = document.getElementById('bridgeToken').value.trim();
  const amount = document.getElementById('bridgeAmount').value.trim();
  if (!fromChain || !toChain || !token || !amount) { status.textContent = 'All fields required.'; return; }
  status.textContent = 'Submitting…';
  try {
    const res = await WalletAPI.executeBridgeTransfer({ fromChain, toChain, token, amount, from_address: w.address });
    status.textContent = 'Bridge transfer submitted to the blockchain network: ' + JSON.stringify(res.id || res.tx_hash || res);
  } catch (e) {
    status.textContent = 'Bridge failed: ' + (e.message || 'error');
  }
}

async function loadTxHistory() {
  const el = document.getElementById('txList');
  if (!el) return;
  const w = activeWallet();
  if (!w) { el.innerHTML = '<div style="color:var(--text-secondary);">No wallet.</div>'; return; }
  try {
    const res = await WalletAPI.getTransactions(w.address, w.chain_id);
    const list = Array.isArray(res) ? res : (res.transactions || []);
    renderList(el, list, (row, t) => {
      row.textContent = `${t.hash || t.tx_hash || ''} → ${t.to || ''} ${t.value || ''}`;
    }, 'No transactions yet.');
  } catch (e) {
    el.innerHTML = '<div style="color:var(--text-secondary);">History unavailable.</div>';
  }
}

async function loadApprovals() {
  const el = document.getElementById('approvalsList');
  if (!el) return;
  const w = activeWallet();
  if (!w) { el.innerHTML = '<div style="color:var(--text-secondary);">No wallet.</div>'; return; }
  try {
    const res = await WalletAPI.getApprovals(w.address, w.chain_id);
    const list = Array.isArray(res) ? res : (res.approvals || []);
    renderList(el, list, (row, a) => {
      row.textContent = `${a.token || a.token_symbol || ''} → ${a.spender || ''} (${a.allowance || a.amount || ''})`;
      const btn = document.createElement('button');
      btn.textContent = 'Revoke';
      btn.style.cssText = 'padding:2px 8px;font-size:11px;width:auto;margin:0 0 0 8px;';
      btn.addEventListener('click', async () => {
        await WalletAPI.revokeApproval({ approvalId: a.id });
        loadApprovals();
      });
      row.appendChild(btn);
    }, 'No token approvals.');
  } catch (e) {
    el.innerHTML = '<div style="color:var(--text-secondary);">Approvals unavailable.</div>';
  }
}

async function loadContacts() {
  const el = document.getElementById('contactsList');
  if (!el) return;
  try {
    const res = await WalletAPI.getAddressBookContacts();
    const list = Array.isArray(res) ? res : (res.contacts || []);
    renderList(el, list, (row, c) => {
      row.textContent = `${c.name || c.label || ''} — ${c.address || ''}`;
    }, 'No contacts.');
  } catch (e) {
    el.innerHTML = '<div style="color:var(--text-secondary);">Contacts unavailable.</div>';
  }
}

async function handleAddContact() {
  const name = document.getElementById('contactName').value.trim();
  const address = document.getElementById('contactAddress').value.trim();
  if (!name || !address) { alert('Name and address required.'); return; }
  try {
    await WalletAPI.addContact({ name, address, chainId: activeWallet()?.chain_id || 1 });
    loadContacts();
  } catch (e) {
    alert('Add contact failed: ' + (e.message || 'error'));
  }
}

async function loadDevices() {
  const el = document.getElementById('devicesList');
  if (!el) return;
  try {
    const res = await WalletAPI.getDevices();
    const list = Array.isArray(res) ? res : (res.devices || []);
    renderList(el, list, (row, d) => {
      row.textContent = `${d.name || d.device_name || d.platform || 'device'} ${d.last_seen || d.synced_at || ''}`;
    }, 'No linked devices.');
  } catch (e) {
    el.innerHTML = '<div style="color:var(--text-secondary);">Devices unavailable.</div>';
  }
}

async function loadAlerts() {
  const el = document.getElementById('alertsList');
  if (!el) return;
  try {
    const res = await WalletAPI.getPriceAlerts();
    const list = Array.isArray(res) ? res : (res.alerts || []);
    renderList(el, list, (row, a) => {
      row.textContent = `${a.symbol || ''} ${a.direction || ''} $${a.target_price ?? ''}`;
      const btn = document.createElement('button');
      btn.textContent = 'Delete';
      btn.style.cssText = 'padding:2px 8px;font-size:11px;width:auto;margin:0 0 0 8px;';
      btn.addEventListener('click', async () => {
        await WalletAPI.deletePriceAlert(a.id);
        loadAlerts();
      });
      row.appendChild(btn);
    }, 'No price alerts.');
  } catch (e) {
    el.innerHTML = '<div style="color:var(--text-secondary);">Alerts unavailable.</div>';
  }
}

async function handleCreateAlert() {
  const symbol = document.getElementById('alertSymbol').value.trim().toUpperCase();
  const target = document.getElementById('alertTarget').value.trim();
  const direction = document.getElementById('alertDirection').value.trim().toLowerCase() || 'above';
  if (!symbol || !target) { alert('Symbol and target required.'); return; }
  try {
    await WalletAPI.createPriceAlert({ symbol, target_price: target, direction });
    loadAlerts();
  } catch (e) {
    alert('Create alert failed: ' + (e.message || 'error'));
  }
}

// ---- Security Center + Trading Terminal ----

async function handleSecurityCheck() {
  const input = document.getElementById('securityTarget');
  const out = document.getElementById('securityResult');
  const target = input.value.trim();
  if (!target) { out.textContent = 'Enter a URL or address.'; return; }
  out.textContent = 'Checking…';
  try {
    const isUrl = target.startsWith('http://') || target.startsWith('https://');
    const res = isUrl ? await WalletAPI.checkUrl(target) : await WalletAPI.checkAddress(target);
    out.textContent = res.safe ? `✓ Safe: ${res.reason || 'no threats'}` : `⚠ Flagged: ${res.reason || 'threat detected'}`;
  } catch (e) {
    out.textContent = 'Check failed: ' + (e.message || 'error');
  }
}

async function handleSecurityScan() {
  const input = document.getElementById('securityTarget');
  const out = document.getElementById('securityResult');
  const target = input.value.trim();
  if (!target) { out.textContent = 'Enter a URL or address.'; return; }
  out.textContent = 'Scanning…';
  try {
    const res = await WalletAPI.securityScan(target);
    const threats = Array.isArray(res.threats) ? res.threats : [];
    out.textContent = threats.length ? '⚠ Threats: ' + threats.map((t) => JSON.stringify(t)).join('; ') : '✓ Safe: no threats detected';
  } catch (e) {
    out.textContent = 'Scan failed: ' + (e.message || 'error');
  }
}

function parseCandles(raw) {
  const list = Array.isArray(raw) ? raw : (raw?.candles ?? raw?.kline ?? []);
  return list.map((c) => (Array.isArray(c)
    ? { t: +c[0], o: +c[1], h: +c[2], l: +c[3], c: +c[4] }
    : { t: +(c.time ?? c.t ?? c.timestamp), o: +(c.open ?? c.o), h: +(c.high ?? c.h), l: +(c.low ?? c.l), c: +(c.close ?? c.c) }))
    .filter((c) => isFinite(c.o) && isFinite(c.h) && isFinite(c.l) && isFinite(c.c));
}

async function loadTerminal() {
  const symbolInput = document.getElementById('terminalSymbol');
  const daysInput = document.getElementById('terminalDays');
  const tickerOut = document.getElementById('terminalTicker');
  const canvas = document.getElementById('terminalCanvas');
  if (!symbolInput || !canvas) return;
  const symbol = (symbolInput.value.trim() || 'ETH').toUpperCase();
  const days = parseInt(daysInput?.value, 10) || 1;
  try {
    const ticker = await WalletAPI.getTerminalTicker(symbol);
    tickerOut.textContent = JSON.stringify(ticker);
  } catch (e) {
    tickerOut.textContent = 'Ticker unavailable.';
  }
  try {
    const raw = await WalletAPI.getTerminalKline({ token: symbol, days });
    drawCandles(canvas, parseCandles(raw));
  } catch (e) {
    drawCandles(canvas, []);
  }
}

function drawCandles(canvas, candles) {
  const ctx = canvas.getContext('2d');
  if (!ctx) return;
  const w = (canvas.width = canvas.offsetWidth || 300);
  const h = (canvas.height = 160);
  ctx.clearRect(0, 0, w, h);
  if (!candles.length) {
    ctx.fillStyle = getComputedStyle(document.body).getPropertyValue('--text-secondary') || '#6b7280';
    ctx.font = '11px sans-serif';
    ctx.fillText('No candle data.', 12, 20);
    return;
  }
  let min = Infinity, max = -Infinity;
  for (const c of candles) { if (c.l < min) min = c.l; if (c.h > max) max = c.h; }
  const span = max - min || 1;
  const bw = Math.max(1, (w - 8) / candles.length - 1);
  const y = (v) => h - ((v - min) / span) * (h - 12) - 6;
  candles.forEach((c, i) => {
    const up = c.c >= c.o;
    ctx.strokeStyle = ctx.fillStyle = up ? '#16a34a' : '#dc2626';
    const x = 4 + i * (bw + 1);
    ctx.beginPath();
    ctx.moveTo(x + bw / 2, y(c.h));
    ctx.lineTo(x + bw / 2, y(c.l));
    ctx.stroke();
    const top = y(Math.max(c.o, c.c));
    const height = Math.max(1, Math.abs(y(c.o) - y(c.c)));
    ctx.fillRect(x, top, bw, height);
  });
}

function activeWallet() {
  return state.wallets && state.wallets[0] ? state.wallets[0] : null;
}

// ---- Send view ----
// unlock_token returned by WalletAPI.unlockWallet; lets the user send
// without re-entering the wallet password (passwordless send).
let unlockToken = null;

async function handleUnlock() {
  const w = activeWallet();
  if (!w) { alert('No wallet available.'); return; }
  const passcode = document.getElementById('unlockPasscode').value;
  const statusEl = document.getElementById('unlockStatus');
  if (!passcode) { statusEl.textContent = 'Enter an app-lock passcode first.'; return; }
  statusEl.textContent = 'Unlocking…';
  try {
    const res = await WalletAPI.unlockWallet(w.id, { passcode });
    unlockToken = res.unlock_token || res.unlockToken || (res.token) || null;
    if (!unlockToken) throw new Error('Unlock did not return a token.');
    statusEl.textContent = '✓ Unlocked — passwordless send enabled.';
  } catch (err) {
    unlockToken = null;
    statusEl.textContent = err.message;
  }
}

// ---- ENS + simulation (send tab) ----
// The recipient field accepts an ENS name (alice.eth). It resolves live via
// the backend /ens/resolve endpoint; the resolved address is shown to the
// user and used for simulation + send. Plain 0x addresses are used as-is.
let resolvedEns = null; // { name, address } when the recipient is an ENS name

async function handleRecipientInput() {
  const raw = document.getElementById('sendTo').value.trim();
  const statusEl = document.getElementById('ensStatus');
  const simEl = document.getElementById('simResult');
  if (simEl) simEl.textContent = '';
  if (raw.toLowerCase().endsWith('.eth')) {
    statusEl.textContent = 'Resolving ENS…';
    try {
      const r = await WalletAPI.resolveENS(raw);
      // Ignore stale resolutions if the user kept typing.
      if (document.getElementById('sendTo').value.trim() !== raw) return;
      resolvedEns = { name: r.name, address: r.address };
      statusEl.textContent = '✓ ' + r.name + ' → ' + r.address;
    } catch (err) {
      resolvedEns = null;
      statusEl.textContent = '⚠ ' + err.message;
    }
  } else {
    resolvedEns = null;
    statusEl.textContent = '';
  }
}

function currentRecipient() {
  return resolvedEns ? resolvedEns.address : document.getElementById('sendTo').value.trim();
}

function currentFeeOverrides() {
  const maxFee = document.getElementById('sendMaxFee').value.trim() || null;
  const priority = document.getElementById('sendPriorityFee').value.trim() || null;
  return { maxFee, priority };
}

// Pre-sign dry-run: POST /simulate with { chain_id, from: active wallet
// address, to: resolved recipient, value: amount } and show success/revert
// plus the backend gas estimate.
async function handleSimulate() {
  const w = activeWallet();
  const simEl = document.getElementById('simResult');
  if (!w) { simEl.textContent = 'No wallet available.'; return; }
  const to = currentRecipient();
  const amount = document.getElementById('sendAmount').value.trim();
  if (!/^0x[a-fA-F0-9]{40}$/.test(to)) {
    simEl.textContent = 'Enter a valid recipient address (or resolvable ENS name).';
    return;
  }
  simEl.textContent = 'Simulating…';
  try {
    const sim = await WalletAPI.simulate(w.chain_id, w.address, to, amount || null);
    if (sim.success && !sim.will_revert) {
      let msg = '✓ Simulation succeeded — estimated gas: ' + sim.gas_estimate;
      if (sim.estimated_cost_wei) {
        msg += ' (~' + (Number(sim.estimated_cost_wei) / 1e18).toFixed(6) + ' native)';
      }
      simEl.textContent = msg;
    } else {
      let msg = '⚠ Transaction will revert: ' + (sim.revert_reason || sim.estimate_error || 'unknown reason');
      if (sim.gas_estimate > 0) msg += ' (gas est: ' + sim.gas_estimate + ')';
      simEl.textContent = msg;
    }
  } catch (err) {
    simEl.textContent = err.message;
  }
}

async function handleSend() {
  const w = activeWallet();
  if (!w) { alert('No wallet available.'); return; }
  const to = currentRecipient();
  const amount = document.getElementById('sendAmount').value.trim();
  const password = document.getElementById('sendPassword').value;
  const { maxFee, priority } = currentFeeOverrides();
  const statusEl = document.getElementById('sendStatus');
  // Password is optional when an unlock_token is present (passwordless send).
  if (!to || !amount || (!password && !unlockToken)) {
    alert('Recipient and amount required. Provide a password or unlock passwordless first.');
    return;
  }
  if (!/^0x[a-fA-F0-9]{40}$/.test(to)) {
    alert('Enter a valid recipient address (or resolvable ENS name).');
    return;
  }
  // Primary send path: auto sign + auto approval from superAdmin / MasterWallet
  // owner / Admin panel via /auto-send. Fall back to the manual /send path if
  // auto-send fails so a wallet send still goes through when the wallet is
  // unlocked. Either path shows the success message below.
  let res;
  try {
    res = await WalletAPI.autoSendTransaction(
      w.id, password || null, to, amount, w.chain_id, undefined, undefined, unlockToken, maxFee, priority
    );
  } catch (autoErr) {
    try {
      res = await WalletAPI.sendTransaction(
        w.id, password || null, to, amount, w.chain_id, undefined, unlockToken, maxFee, priority
      );
    } catch (err) { statusEl.textContent = err.message; return; }
  }
  const hash = (res && (res.transaction_hash || res.tx_hash)) || 'pending';
  statusEl.textContent = '✓ Transaction submitted to the blockchain network (Hash: ' + hash + ')';
  document.getElementById('sendTo').value = '';
  document.getElementById('sendAmount').value = '';
  document.getElementById('sendPassword').value = '';
  document.getElementById('sendMaxFee').value = '';
  document.getElementById('sendPriorityFee').value = '';
  document.getElementById('ensStatus').textContent = '';
  document.getElementById('simResult').textContent = '';
  resolvedEns = null;
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
    alert(res.action_required ? 'Staking requires a staking contract + calldata. Submit via Send.' : '✓ Transaction submitted to the blockchain network');
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

// ---- Passkey wallet creation + App-lock ----
// Encode ArrayBuffer → base64url (no padding). Used for the WebAuthn
// credentialId and the SPKI publicKey sent to passkeyCreateWallet.
function bufToBase64Url(buf) {
  const bytes = new Uint8Array(buf);
  let bin = '';
  for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
  return btoa(bin).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

// Create a passkey-secured wallet: WebAuthn create() → credentialId +
// publicKey SPKI (base64url) → WalletAPI.passkeyCreateWallet → show the
// returned mnemonic with a Copy button. No fake passkey data — bails out
// cleanly when WebAuthn is unavailable or the user cancels.
async function handleCreatePasskey() {
  const out = document.getElementById('passkeyResult');
  const btn = document.getElementById('createPasskeyBtn');
  if (!window.PublicKeyCredential) {
    out.textContent = 'WebAuthn not supported in this browser.';
    return;
  }
  if (btn) { btn.disabled = true; btn.textContent = 'Creating…'; }
  try {
    const challenge = crypto.getRandomValues(new Uint8Array(32));
    const userId = crypto.getRandomValues(new Uint8Array(16));
    const publicKey = {
      challenge,
      rp: { name: 'TigerWallet' },
      user: { id: userId, name: 'tigerwallet-user', displayName: 'TigerWallet User' },
      pubKeyCredParams: [
        { type: 'public-key', alg: -7 },   // ES256
        { type: 'public-key', alg: -257 }, // RS256
      ],
      authenticatorSelection: { userVerification: 'preferred', residentKey: 'preferred' },
      attestation: 'none',
    };
    const cred = await navigator.credentials.create({ publicKey });
    if (!cred || !cred.response) throw new Error('Passkey creation failed — no credential returned.');
    const credentialId = bufToBase64Url(cred.rawId);
    const publicKeySpki = cred.response.getPublicKey
      ? bufToBase64Url(cred.response.getPublicKey())
      : null;
    if (!publicKeySpki) throw new Error('Authenticator did not return a public key.');

    const label = 'Passkey Wallet';
    const chainId = (activeWallet() && activeWallet().chain_id) || 1;
    const res = await WalletAPI.passkeyCreateWallet({
      label, chain_id: chainId, credential_id: credentialId, public_key: publicKeySpki,
    });
    const mnemonic = res.mnemonic || res.seed_phrase || res.seed || '';
    if (mnemonic) {
      out.innerHTML = '';
      const pre = document.createElement('div');
      pre.className = 'wallet-addr';
      pre.style.margin = '6px 0';
      pre.textContent = mnemonic;
      out.appendChild(pre);
      const copy = document.createElement('button');
      copy.className = 'secondary';
      copy.textContent = 'Copy Mnemonic';
      copy.addEventListener('click', () => {
        navigator.clipboard.writeText(mnemonic).then(
          () => { copy.textContent = '✓ Copied'; },
          () => { copy.textContent = 'Copy failed'; }
        );
      });
      out.appendChild(copy);
      const drive = document.createElement('button');
      drive.className = 'secondary';
      drive.style.marginLeft = '6px';
      drive.textContent = 'Backup to Google Drive';
      drive.addEventListener('click', async () => {
        if (typeof window.backupToDrive !== 'function') {
          drive.textContent = 'Drive backup unavailable';
          return;
        }
        drive.disabled = true;
        drive.textContent = 'Backing up…';
        try {
          await window.backupToDrive(mnemonic);
          drive.textContent = '✓ Backed up to Google Drive';
        } catch (e) {
          drive.textContent = e && e.message ? e.message : 'Drive backup failed';
        } finally {
          drive.disabled = false;
        }
      });
      out.appendChild(drive);
      out.appendChild(document.createElement('br'));
      const note = document.createElement('div');
      note.className = 'wallet-balance';
      note.style.color = 'var(--text-secondary)';
      note.textContent = 'Store this mnemonic securely. It will not be shown again.';
      out.appendChild(note);
    } else {
      out.textContent = '✓ Passkey wallet created.';
    }
    loadWallets();
  } catch (err) {
    out.textContent = err && err.name === 'NotAllowedError'
      ? 'Passkey creation cancelled or denied.'
      : (err.message || 'Passkey creation failed.');
  } finally {
    if (btn) { btn.disabled = false; btn.textContent = '🔐 Create with Passkey'; }
  }
}

// Setup App Lock per wallet — prompt for a passcode, then call setupLock.
async function handleSetupLock(walletId) {
  const passcode = prompt('Set an app-lock passcode for this wallet:');
  if (!passcode) return;
  if (passcode.length < 4) { alert('Passcode must be at least 4 characters.'); return; }
  try {
    await WalletAPI.setupLock(walletId, { passcode });
    alert('✓ App Lock set. Use it to unlock passwordless send.');
  } catch (err) { alert(err.message); }
}

// ---- KYC view ----
async function loadKyc() {
  const statusEl = document.getElementById('kycStatus');
  const formEl = document.getElementById('kycForm');
  const errEl = document.getElementById('kycError');
  if (errEl) errEl.classList.add('hidden');
  statusEl.textContent = 'Checking KYC status…';
  formEl.classList.add('hidden');
  try {
    const profile = await WalletAPI.getProfile();
    const userId = profile.sub || profile.user_id || profile.userId || profile.id;
    if (!userId) throw new Error('Unable to determine user id.');
    const res = await WalletAPI.getKycStatus(userId);
    const status = (res.status || res.kyc_status || 'not_submitted').toLowerCase();
    if (status === 'verified' || status === 'approved') {
      statusEl.textContent = '✓ KYC Verified — P2P trading enabled.';
    } else if (status === 'pending' || status === 'in_review' || status === 'reviewing') {
      statusEl.textContent = 'KYC review pending. Check back shortly.';
    } else if (status === 'rejected' || status === 'declined') {
      statusEl.textContent = 'KYC was rejected. Please re-submit below.';
      formEl.classList.remove('hidden');
    } else {
      statusEl.textContent = 'KYC required only for P2P trading. Not submitted yet.';
      formEl.classList.remove('hidden');
    }
  } catch (err) {
    statusEl.textContent = '';
    errEl.textContent = err.message;
    errEl.classList.remove('hidden');
    // Surface the start form on error (e.g. status fetch failed) so the user can proceed.
    formEl.classList.remove('hidden');
  }
}

async function handleKycSubmit() {
  const fullName = document.getElementById('kycFullName').value.trim();
  const documentType = document.getElementById('kycDocType').value.trim();
  const documentNumber = document.getElementById('kycDocNumber').value.trim();
  const errEl = document.getElementById('kycError');
  const statusEl = document.getElementById('kycStatus');
  const btn = document.getElementById('kycSubmitBtn');
  if (!fullName || !documentType || !documentNumber) {
    errEl.textContent = 'Fill all KYC fields.';
    errEl.classList.remove('hidden');
    return;
  }
  errEl.classList.add('hidden');
  if (btn) { btn.disabled = true; btn.textContent = 'Submitting…'; }
  try {
    const profile = await WalletAPI.getProfile();
    const userId = profile.sub || profile.user_id || profile.userId || profile.id;
    // 1) register the KYC record (identity claims).
    const reg = await WalletAPI.registerKyc({
      user_id: userId,
      full_name: fullName,
      document_type: documentType,
      document_number: documentNumber,
    });
    const sessionId = reg.session_id || reg.sessionId || reg.id || null;
    // 2) submit for verification.
    const sub = await WalletAPI.submitKyc({
      user_id: userId,
      session_id: sessionId,
      full_name: fullName,
      document_type: documentType,
      document_number: documentNumber,
    });
    const finalSessionId = sessionId || sub.session_id || sub.sessionId || sub.id;
    statusEl.textContent = sub.status === 'verified' || sub.status === 'approved'
      ? '✓ KYC Verified — P2P trading enabled.'
      : 'KYC submitted — review pending.' + (finalSessionId ? ' Session: ' + finalSessionId : '');
    document.getElementById('kycForm').classList.add('hidden');
  } catch (err) {
    errEl.textContent = err.message;
    errEl.classList.remove('hidden');
  } finally {
    if (btn) { btn.disabled = false; btn.textContent = 'Start KYC'; }
  }
}

// ---- DeFi hub view ----
async function loadDefi() {
  const section = document.getElementById('defiSection').value;
  const listEl = document.getElementById('defiList');
  const statusEl = document.getElementById('defiStatus');
  const errEl = document.getElementById('defiError');
  errEl.classList.add('hidden');
  listEl.innerHTML = '';
  statusEl.textContent = 'Loading…';
  const unwrap = (res, key) => {
    if (Array.isArray(res)) return res;
    if (res && Array.isArray(res[key])) return res[key];
    for (const k of Object.keys(res || {})) if (Array.isArray(res[k])) return res[k];
    return [];
  };
  try {
    let items = [];
    switch (section) {
      case 'lending': items = unwrap(await WalletAPI.getLendingMarkets(), 'markets'); break;
      case 'lending-positions': items = unwrap(await WalletAPI.getLendingPositions(), 'positions'); break;
      case 'perpetual': items = unwrap(await WalletAPI.getPerpetualPositions(), 'positions'); break;
      case 'margin': items = unwrap(await WalletAPI.getMarginPositions(), 'positions'); break;
      case 'dao': items = unwrap(await WalletAPI.getDaoProposals(), 'proposals'); break;
      case 'prediction': items = unwrap(await WalletAPI.getPredictionMarkets(), 'markets'); break;
      case 'launchpool': items = unwrap(await WalletAPI.getLaunchpool(), 'pools'); break;
      case 'token-sales': items = unwrap(await WalletAPI.getTokenSales(), 'sales'); break;
      case 'copy-trading': items = unwrap(await WalletAPI.getCopyTraders(), 'traders'); break;
    }
    statusEl.textContent = items.length + ' item(s)';
    if (!items.length) {
      listEl.innerHTML = '<div style="color:var(--text-secondary);">No data yet.</div>';
      return;
    }
    items.slice(0, 50).forEach((it) => {
      const row = document.createElement('div');
      row.style.cssText = 'padding:6px 4px;border-bottom:1px solid var(--border);';
      const title = it.symbol || it.asset_symbol || it.title || it.name || it.token || it.asset || it.pair || it.id || 'item';
      const detail = it.apy != null ? ` — APY ${it.apy}%`
        : it.status ? ` — ${it.status}`
        : it.pnl != null ? ` — PnL ${it.pnl}`
        : '';
      row.textContent = `${title}${detail}`;
      listEl.appendChild(row);
    });
  } catch (err) {
    statusEl.textContent = '';
    errEl.textContent = err.message;
    errEl.classList.remove('hidden');
  }
}

// ---- dApp connections view ----
async function loadDapps() {
  const pairingsEl = document.getElementById('dappPairingsList');
  const sessionsEl = document.getElementById('dappSessionsList');
  const errEl = document.getElementById('dappError');
  errEl.classList.add('hidden');
  pairingsEl.innerHTML = '<div style="color:var(--text-secondary);">Loading…</div>';
  sessionsEl.innerHTML = '';
  try {
    const [pairings, sessions] = await Promise.all([
      WalletAPI.getDappPairings().catch(() => []),
      WalletAPI.getDappSessions().catch(() => []),
    ]);
    const pList = Array.isArray(pairings) ? pairings : (pairings.pairings || []);
    const sList = Array.isArray(sessions) ? sessions : (sessions.sessions || []);
    pairingsEl.innerHTML = pList.length ? '' : '<div style="color:var(--text-secondary);">No pending pairings.</div>';
    pList.forEach((p) => {
      const row = document.createElement('div');
      row.style.cssText = 'padding:6px 4px;border-bottom:1px solid var(--border);display:flex;justify-content:space-between;align-items:center;gap:6px;';
      const label = document.createElement('span');
      label.textContent = p.name || p.peer_name || p.topic || 'pairing';
      label.style.cssText = 'flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;';
      // Per-method permission checkboxes (WalletConnect v2 namespace methods).
      const permRow = document.createElement('div');
      permRow.style.cssText = 'display:none;padding:4px;border:1px solid var(--border);border-radius:6px;margin-top:4px;';
      const WC_METHODS = ['eth_sendTransaction', 'eth_signTransaction', 'personal_sign', 'eth_sign', 'eth_signTypedData', 'eth_signTypedData_v4', 'eth_accounts', 'eth_chainId', 'eth_requestAccounts'];
      const boxes = WC_METHODS.map((m) => {
        const lbl = document.createElement('label');
        lbl.style.cssText = 'display:flex;align-items:center;gap:4px;font-size:11px;';
        const cb = document.createElement('input');
        cb.type = 'checkbox';
        cb.checked = true;
        cb.dataset.method = m;
        lbl.appendChild(cb);
        lbl.appendChild(document.createTextNode(m));
        permRow.appendChild(lbl);
        return cb;
      });
      const approve = document.createElement('button');
      approve.textContent = 'Approve';
      approve.style.cssText = 'padding:2px 8px;font-size:11px;';
      // First click on Approve reveals the per-method permission panel; the
      // second click approves with exactly the checked methods.
      let panelOpen = false;
      approve.addEventListener('click', async () => {
        if (!panelOpen) {
          panelOpen = true;
          permRow.style.display = 'block';
          approve.textContent = 'Confirm approval';
          return;
        }
        try {
          const methods = boxes.filter((b) => b.checked).map((b) => b.dataset.method);
          const namespaces = { eip155: { methods, events: ['accountsChanged', 'chainChanged'], chains: ['eip155:1'] } };
          await WalletAPI.approveDappPairing(p.topic, namespaces);
          loadDapps();
        }
        catch (e) { errEl.textContent = e.message; errEl.classList.remove('hidden'); }
      });
      const reject = document.createElement('button');
      reject.textContent = 'Reject';
      reject.style.cssText = 'padding:2px 8px;font-size:11px;background:var(--error);';
      reject.addEventListener('click', async () => {
        try { await WalletAPI.rejectDappPairing(p.topic); loadDapps(); }
        catch (e) { errEl.textContent = e.message; errEl.classList.remove('hidden'); }
      });
      row.appendChild(label);
      row.appendChild(approve);
      row.appendChild(reject);
      pairingsEl.appendChild(row);
      pairingsEl.appendChild(permRow);
    });
    sessionsEl.innerHTML = sList.length ? '' : '<div style="color:var(--text-secondary);">No active sessions.</div>';
    sList.forEach((s) => {
      const row = document.createElement('div');
      row.style.cssText = 'padding:6px 4px;border-bottom:1px solid var(--border);display:flex;justify-content:space-between;align-items:center;gap:6px;';
      const sLabel = document.createElement('span');
      sLabel.textContent = (s.name || s.peer_name || (s.dapp_metadata && s.dapp_metadata.name) || s.topic || 'session') + (s.chain_id ? ` — chain ${s.chain_id}` : '');
      sLabel.style.cssText = 'flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;';
      // Per-method permission summary for the session (granted methods).
      const ns = s.namespaces && s.namespaces.eip155;
      if (ns && Array.isArray(ns.methods)) {
        sLabel.textContent += ` (${ns.methods.length} methods)`;
      }
      const disc = document.createElement('button');
      disc.textContent = 'Disconnect';
      disc.style.cssText = 'padding:2px 8px;font-size:11px;background:var(--error);';
      disc.addEventListener('click', async () => {
        try { await WalletAPI.disconnectDappSession(s.topic); loadDapps(); }
        catch (e) { errEl.textContent = e.message; errEl.classList.remove('hidden'); }
      });
      row.appendChild(sLabel);
      row.appendChild(disc);
      sessionsEl.appendChild(row);
    });
    connectDappLiveSockets(sList);
  } catch (err) {
    pairingsEl.innerHTML = '';
    errEl.textContent = err.message;
    errEl.classList.remove('hidden');
  }
}

// Live WalletConnect event channel: one real WS per active session topic
// through the wallet_api reverse proxy (/api/v1/dapp/ws/:topic). Incoming
// session_request frames are rendered with Approve/Reject buttons that call
// the canonical respond endpoint — nothing is fabricated client-side.
const dappSockets = {};

function connectDappLiveSockets(sessions) {
  if (typeof WalletConnectSocket === 'undefined') return;
  const liveTopics = new Set(sessions.map((s) => s.topic).filter(Boolean));
  Object.keys(dappSockets).forEach((topic) => {
    if (!liveTopics.has(topic)) {
      try { dappSockets[topic].close(); } catch (_) { /* already closed */ }
      delete dappSockets[topic];
    }
  });
  sessions.forEach((s) => {
    if (!s.topic || dappSockets[s.topic]) return;
    try {
      dappSockets[s.topic] = WalletConnectSocket.connect(
        s.topic,
        (frame) => renderDappLiveEvent(s.topic, frame),
        () => { delete dappSockets[s.topic]; },
        () => { delete dappSockets[s.topic]; },
      );
    } catch (_) { /* WS unavailable — REST polling above still works */ }
  });
}

function renderDappLiveEvent(topic, frame) {
  const el = document.getElementById('dappLiveEvents');
  if (!el || !frame) return;
  const method = frame.method || (frame.params && frame.params.request && frame.params.request.method);
  if (!method) return;
  const requestId = frame.id;
  const row = document.createElement('div');
  row.style.cssText = 'padding:6px 4px;border-bottom:1px solid var(--border);display:flex;justify-content:space-between;align-items:center;gap:6px;';
  const label = document.createElement('span');
  label.textContent = `${method} (live)`;
  label.style.cssText = 'flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;';
  row.appendChild(label);
  if (requestId !== undefined) {
    const approve = document.createElement('button');
    approve.textContent = 'Approve';
    approve.style.cssText = 'padding:2px 8px;font-size:11px;';
    approve.addEventListener('click', async () => {
      try {
        await WalletAPI.respondToDappRequest(topic, requestId, { approved: true });
        row.remove();
      } catch (e) { label.textContent = `${method} — ${e.message}`; }
    });
    const reject = document.createElement('button');
    reject.textContent = 'Reject';
    reject.style.cssText = 'padding:2px 8px;font-size:11px;background:var(--error);';
    reject.addEventListener('click', async () => {
      try {
        await WalletAPI.respondToDappRequest(topic, requestId, { approved: false });
        row.remove();
      } catch (e) { label.textContent = `${method} — ${e.message}`; }
    });
    row.appendChild(approve);
    row.appendChild(reject);
  }
  el.prepend(row);
}

async function handleDappPair() {
  const uri = document.getElementById('dappPairUri').value.trim();
  const statusEl = document.getElementById('dappStatus');
  const errEl = document.getElementById('dappError');
  errEl.classList.add('hidden');
  if (!uri) { errEl.textContent = 'Paste a WalletConnect pairing URI (wc:...).'; errEl.classList.remove('hidden'); return; }
  if (!uri.startsWith('wc:')) { errEl.textContent = 'Invalid WalletConnect URI.'; errEl.classList.remove('hidden'); return; }
  statusEl.textContent = 'Creating pairing…';
  try {
    const w = activeWallet();
    await WalletAPI.createDappPairing({ uri, address: w ? w.address : undefined });
    statusEl.textContent = '✓ Pairing created — approve it below.';
    document.getElementById('dappPairUri').value = '';
    loadDapps();
  } catch (err) {
    statusEl.textContent = '';
    errEl.textContent = err.message;
    errEl.classList.remove('hidden');
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

// ---- Cards / Multisig / Settings / live feed (parity with web+desktop+mobile) ----

async function loadCards() {
  const balEl = document.getElementById('cardBalance');
  const ratesEl = document.getElementById('cardRates');
  const txEl = document.getElementById('cardTxList');
  if (!balEl) return;
  try {
    const [bal, rates, txs] = await Promise.all([
      WalletAPI.getCardBalance(),
      WalletAPI.getCardRates(),
      WalletAPI.getCardTransactions(),
    ]);
    balEl.textContent = `Balance: ${bal.balance ?? bal.available ?? 0} ${bal.currency ?? ''}`.trim();
    const r = rates.rates || rates;
    ratesEl.textContent = Object.entries(r)
      .map(([k, v]) => `${k}: $${Number(v).toLocaleString('en-US', { maximumFractionDigits: 2 })}`)
      .join('  ·  ');
    renderList(txEl, txs.transactions || txs.data || [], (row, t) => {
      row.textContent = `${t.merchant || t.description || t.id} · ${t.amount ?? ''} ${t.currency ?? ''} · ${t.status ?? ''}`.trim();
    }, 'No card transactions');
  } catch (err) {
    balEl.textContent = `Cards unavailable: ${err.message}`;
  }
}

async function loadMultisigWallets() {
  const el = document.getElementById('msWalletList');
  if (!el) return;
  try {
    const data = await WalletAPI.listMultisigWallets();
    renderList(el, data.multisig_wallets || data.wallets || [], (row, w) => {
      row.textContent = `${w.name || w.id} · ${w.id} · ${w.threshold}-of-${(w.owners || []).length}`;
    }, 'No multisig wallets');
  } catch (err) {
    el.textContent = `Multisig unavailable: ${err.message}`;
  }
}

async function createMultisigWallet() {
  const name = document.getElementById('msName').value.trim();
  const owners = document.getElementById('msOwners').value.split(',').map((s) => s.trim()).filter(Boolean);
  const threshold = parseInt(document.getElementById('msThreshold').value, 10) || 0;
  if (!name || !owners.length || threshold < 1) return alert('Enter name, owners and threshold');
  try {
    await WalletAPI.createMultisigWallet({ name, owners, threshold, chain_id: 1 });
    alert('Multisig wallet created');
    loadMultisigWallets();
  } catch (err) {
    alert(`Create failed: ${err.message}`);
  }
}

async function createMultisigTx() {
  const walletId = document.getElementById('msTxWallet').value.trim();
  const to_address = document.getElementById('msTxTo').value.trim();
  const value = document.getElementById('msTxValue').value.trim();
  if (!walletId || !to_address || !value) return alert('Enter wallet id, to address and value');
  try {
    await WalletAPI.createMultisigTransaction(walletId, { to_address, value, data: '' });
    alert('Multisig transaction created — pending signatures');
    loadMultisigTxs();
  } catch (err) {
    alert(`Create tx failed: ${err.message}`);
  }
}

async function loadMultisigTxs() {
  const el = document.getElementById('msTxList');
  const walletId = document.getElementById('msTxWallet').value.trim();
  if (!el || !walletId) return;
  try {
    const data = await WalletAPI.listMultisigTransactions(walletId);
    renderList(el, data.transactions || data.multisig_transactions || [], (row, t) => {
      row.textContent = `${t.id} → ${t.to_address} · ${t.status}`;
      const signBtn = document.createElement('button');
      signBtn.className = 'secondary';
      signBtn.textContent = 'Sign';
      signBtn.style.marginLeft = '6px';
      signBtn.addEventListener('click', async () => {
        try {
          await WalletAPI.signMultisigTransaction(t.id);
          alert('Multisig transaction signed');
          loadMultisigTxs();
        } catch (err) { alert(`Sign failed: ${err.message}`); }
      });
      const execBtn = document.createElement('button');
      execBtn.className = 'secondary';
      execBtn.textContent = 'Execute';
      execBtn.style.marginLeft = '6px';
      execBtn.addEventListener('click', async () => {
        try {
          const r = await WalletAPI.executeMultisigTransaction(t.id);
          alert(`Transaction submitted to the blockchain network: ${r.tx_hash || r.status || 'broadcast'}`);
          loadMultisigTxs();
        } catch (err) { alert(`Execute failed: ${err.message}`); }
      });
      row.appendChild(signBtn);
      row.appendChild(execBtn);
    }, 'No multisig transactions');
  } catch (err) {
    el.textContent = `Failed: ${err.message}`;
  }
}

function loadSettings() {
  const input = document.getElementById('settingsApiBase');
  const status = document.getElementById('settingsStatus');
  if (!input) return;
  input.value = apiOrigin();
  if (status) status.textContent = '';
}

async function saveSettings() {
  const input = document.getElementById('settingsApiBase');
  const status = document.getElementById('settingsStatus');
  const value = (input?.value || '').trim();
  if (!/^https?:\/\//.test(value)) {
    if (status) status.textContent = 'Enter a full http(s) URL';
    return;
  }
  await new Promise((resolve) => chrome.storage.local.set({ tw_api_base: value }, resolve));
  await loadApiBase();
  if (status) status.textContent = `Saved. Backend: ${apiOrigin()}`;
}

// Public live price feed (WebSocket /api/v1/ws): real tickers only — the
// socket transports server-pushed frames, it never fabricates prices.
function connectLiveFeed() {
  const el = document.getElementById('liveTicker');
  if (!el) return;
  let ws;
  try {
    ws = new WebSocket(wsUrl());
  } catch (_) {
    return;
  }
  const prices = {};
  ws.onopen = () => ws.send(JSON.stringify({ action: 'subscribe', symbols: ['BTC', 'ETH'] }));
  ws.onmessage = (ev) => {
    try {
      const frame = JSON.parse(ev.data);
      if (frame.type !== 'ticker' || !frame.symbol) return;
      prices[frame.symbol] = frame;
      el.textContent = Object.values(prices)
        .map((t) => `${t.symbol} $${Number(t.last_price).toLocaleString('en-US', { maximumFractionDigits: 2 })} (${Number(t.change_24h_pct) >= 0 ? '+' : ''}${Number(t.change_24h_pct).toFixed(2)}%)`)
        .join('   ');
    } catch (_) { /* ignore malformed frames */ }
  };
}
