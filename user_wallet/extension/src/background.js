// Background service worker for the UserWallet extension.
// Handles EIP-1193 provider RPC relayed from dApp pages (via content scripts)
// against the canonical go/wallet_api backend. Keys never live here — all
// signing is delegated to the backend exactly like the popup does.

const API_BASE = 'http://localhost:8443/api/v1';
const DEFAULT_CHAIN_ID = 1;

chrome.runtime.onInstalled.addListener(() => {
  chrome.storage.local.get(['theme', 'chainId', 'apiBase'], (res) => {
    const init = {};
    if (!res.theme) init.theme = 'light';
    if (!res.chainId) init.chainId = DEFAULT_CHAIN_ID;
    if (!res.apiBase) init.apiBase = API_BASE;
    if (Object.keys(init).length) chrome.storage.local.set(init);
  });
});

function storageGet(keys) {
  return new Promise((resolve) => chrome.storage.local.get(keys, resolve));
}
function storageSet(obj) {
  return new Promise((resolve) => chrome.storage.local.set(obj, resolve));
}

async function getApiBase() {
  const { apiBase } = await storageGet(['apiBase']);
  return apiBase || API_BASE;
}

async function getToken() {
  const { token } = await storageGet(['token']);
  return token || null;
}

async function api(path, { method = 'GET', body, auth = true } = {}) {
  const base = await getApiBase();
  const headers = { 'Content-Type': 'application/json', Accept: 'application/json' };
  if (auth) {
    const token = await getToken();
    if (token) headers['Authorization'] = `Bearer ${token}`;
  }
  const res = await fetch(`${base}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || `request failed (${res.status})`);
  return data;
}

// Notify all tabs of a provider event (accountsChanged / chainChanged).
function broadcastEvent(event, payload) {
  chrome.tabs.query({}, (tabs) => {
    for (const tab of tabs) {
      if (tab.id) chrome.tabs.sendMessage(tab.id, { kind: 'provider-event', event, payload }).catch(() => {});
    }
  });
}

async function getActiveWallet() {
  const { activeWalletId } = await storageGet(['activeWalletId']);
  const { wallets } = await api('/wallets');
  if (!wallets || wallets.length === 0) throw new Error('No wallet — create one in the UserWallet popup');
  const wallet = wallets.find((w) => w.id === activeWalletId) || wallets[0];
  return wallet;
}

async function getAccounts() {
  try {
    const wallet = await getActiveWallet();
    return wallet.address ? [wallet.address] : [];
  } catch {
    return [];
  }
}

function hexChainId(id) {
  return '0x' + Number(id).toString(16);
}

// Convert a hex quantity (wei) to an ether decimal string the backend expects.
function weiHexToEther(hexValue) {
  if (!hexValue) return '0';
  const wei = BigInt(hexValue);
  const whole = wei / 10n ** 18n;
  const frac = (wei % 10n ** 18n).toString().padStart(18, '0').replace(/0+$/, '');
  return frac ? `${whole}.${frac}` : `${whole}`;
}

async function handleProviderRpc(method, params) {
  const { chainId } = await storageGet(['chainId']);
  const currentChainId = chainId || DEFAULT_CHAIN_ID;

  switch (method) {
    case 'eth_chainId':
      return hexChainId(currentChainId);
    case 'net_version':
      return String(currentChainId);

    case 'eth_requestAccounts':
    case 'eth_accounts': {
      const accounts = await getAccounts();
      if (accounts.length === 0) throw new Error('UserWallet is locked or has no account');
      return accounts;
    }

    case 'wallet_switchEthereumChain': {
      const target = params && params[0] && params[0].chainId;
      if (!target) throw new Error('chainId required');
      const id = parseInt(target, 16);
      if (!Number.isFinite(id)) throw new Error('invalid chainId');
      await storageSet({ chainId: id });
      broadcastEvent('chainChanged', hexChainId(id));
      return null;
    }

    case 'personal_sign': {
      const [messageHex, address] = params || [];
      const wallet = await getActiveWallet();
      if (address && wallet.address && address.toLowerCase() !== wallet.address.toLowerCase()) {
        throw new Error('Requested account is not the active wallet');
      }
      // personal_sign message arrives hex-encoded; decode to UTF-8 bytes.
      let message = messageHex;
      if (typeof messageHex === 'string' && messageHex.startsWith('0x')) {
        const hexStr = messageHex.slice(2);
        const bytes = new Uint8Array(hexStr.length / 2);
        for (let i = 0; i < bytes.length; i++) bytes[i] = parseInt(hexStr.substr(i * 2, 2), 16);
        message = new TextDecoder().decode(bytes);
      }
      const { signature } = await api('/sign', {
        method: 'POST',
        body: { wallet_id: wallet.id, unlock_token: (await storageGet(['unlockToken'])).unlockToken, message },
      });
      return signature;
    }

    case 'eth_sendTransaction': {
      const tx = (params || [])[0] || {};
      const wallet = await getActiveWallet();
      const { unlockToken } = await storageGet(['unlockToken']);
      const body = {
        wallet_id: wallet.id,
        unlock_token: unlockToken,
        to: tx.to,
        value: weiHexToEther(tx.value),
        chain_id: currentChainId,
      };
      if (tx.data && tx.data !== '0x') body.data = tx.data;
      if (tx.gas) body.gas_limit = parseInt(tx.gas, 16);
      const res = await api('/send', { method: 'POST', body });
      return res.tx_hash || res.txHash || res.hash;
    }

    // Read-only chain queries go straight to the active chain's own node
    // (the backend has no generic RPC proxy; only signing is delegated).
    case 'eth_getBalance':
    case 'eth_call':
    case 'eth_blockNumber':
    case 'eth_estimateGas':
    case 'eth_gasPrice':
    case 'eth_getTransactionReceipt':
    case 'eth_getTransactionByHash':
    case 'eth_getCode':
    case 'eth_getLogs':
      return chainRpc(currentChainId, method, params || []);

    default:
      // Unknown read-only method: try the chain node; fail closed otherwise.
      return chainRpc(currentChainId, method, params || []);
  }
}

// Active chain's RPC endpoint from the backend chain registry (public route).
async function getChainRpcUrl(chainId) {
  const cache = (await storageGet(['chainRpcCache'])).chainRpcCache || {};
  if (cache[chainId]) return cache[chainId];
  const { chains } = await api('/chains', { auth: false });
  const chain = (chains || []).find((c) => c.id === chainId || c.chain_id === chainId);
  const url = chain && (chain.rpc_endpoint || chain.rpcEndpoint || chain.rpc_url);
  if (!url) throw new Error(`No RPC endpoint for chain ${chainId}`);
  cache[chainId] = url;
  await storageSet({ chainRpcCache: cache });
  return url;
}

let rpcId = 1;
async function chainRpc(chainId, method, params) {
  const url = await getChainRpcUrl(chainId);
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ jsonrpc: '2.0', id: rpcId++, method, params }),
  });
  const data = await res.json().catch(() => ({}));
  if (data.error) throw new Error(data.error.message || 'RPC error');
  return data.result;
}

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (!msg || msg.kind !== 'provider-rpc') return false;
  handleProviderRpc(msg.method, msg.params)
    .then((result) => sendResponse({ result }))
    .catch((error) => sendResponse({ error: { code: 4001, message: error.message } }));
  return true; // async response
});

// React to popup-driven account changes so dApps see accountsChanged live.
chrome.storage.onChanged.addListener((changes, area) => {
  if (area !== 'local') return;
  if (changes.activeWalletId || changes.token) {
    getAccounts().then((accounts) => broadcastEvent('accountsChanged', accounts));
  }
  if (changes.chainId) {
    broadcastEvent('chainChanged', hexChainId(changes.chainId.newValue));
  }
});

