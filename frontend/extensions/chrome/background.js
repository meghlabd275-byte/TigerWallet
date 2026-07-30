// TigerWallet Background Service Worker
// Handles wallet operations, Web3 provider, and DApp communication

const WALLET_STATE_KEY = 'tigerwallet_state';
const NETWORKS_KEY = 'tigerwallet_networks';

// Default networks
const DEFAULT_NETWORKS = [
  { id: 1, name: 'Ethereum', symbol: 'ETH', rpc: 'https://eth.llamarpc.com', chainId: '0x1', explorer: 'https://etherscan.io' },
  { id: 56, name: 'BNB Chain', symbol: 'BNB', rpc: 'https://bsc-dataseed.binance.org', chainId: '0x38', explorer: 'https://bscscan.com' },
  { id: 137, name: 'Polygon', symbol: 'MATIC', rpc: 'https://polygon-rpc.com', chainId: '0x89', explorer: 'https://polygonscan.com' },
  { id: 42161, name: 'Arbitrum', symbol: 'ETH', rpc: 'https://arb1.arbitrum.io/rpc', chainId: '0xa4b1', explorer: 'https://arbiscan.io' },
  { id: 10, name: 'Optimism', symbol: 'ETH', rpc: 'https://mainnet.optimism.io', chainId: '0xa', explorer: 'https://optimistic.etherscan.io' },
  { id: 43114, name: 'Avalanche', symbol: 'AVAX', rpc: 'https://api.avax.network/ext/bc/C/rpc', chainId: '0xa86a', explorer: 'https://snowtrace.io' },
];

// Current wallet state
let currentState = {
  isUnlocked: false,
  address: null,
  network: DEFAULT_NETWORKS[0],
  balance: '0',
  tokens: [],
};

// Initialize
chrome.runtime.onInstalled.addListener(() => {
  console.log('TigerWallet installed');
  initializeWallet();
});

async function initializeWallet() {
  const stored = await chrome.storage.local.get(WALLET_STATE_KEY);
  if (stored[WALLET_STATE_KEY]) {
    currentState = { ...currentState, ...stored[WALLET_STATE_KEY] };
  }
  
  const networks = await chrome.storage.local.get(NETWORKS_KEY);
  if (!networks[NETWORKS_KEY]) {
    await chrome.storage.local.set({ [NETWORKS_KEY]: DEFAULT_NETWORKS });
  }
}

// Handle messages from content scripts and popup
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  handleMessage(message, sender).then(sendResponse);
  return true;
});

async function handleMessage(message, sender) {
  switch (message.type) {
    case 'GET_WALLET_STATE':
      return currentState;
    
    case 'UNLOCK_WALLET':
      return await unlockWallet(message.payload);
    
    case 'LOCK_WALLET':
      return await lockWallet();
    
    case 'SET_NETWORK':
      return await setNetwork(message.payload);
    
    case 'GET_ACCOUNTS':
      return currentState.isUnlocked ? [currentState.address] : [];
    
    case 'SIGN_MESSAGE':
      return await signMessage(message.payload);
    
    case 'SIGN_TRANSACTION':
      return await signTransaction(message.payload);
    
    case 'SEND_TRANSACTION':
      return await sendTransaction(message.payload);
    
    case 'ADD_TOKEN':
      return await addToken(message.payload);
    
    case 'GET_BALANCE':
      return await getBalance(message.payload);
    
    default:
      return { error: 'Unknown message type' };
  }
}

async function unlockWallet(payload) {
  // In production, this would validate credentials securely
  // For demo, generate a deterministic address from payload
  const { password, seedPhrase } = payload;
  
  // Generate address from seed phrase (simplified)
  const address = generateAddressFromSeed(seedPhrase || password || 'default_seed');
  
  currentState = {
    ...currentState,
    isUnlocked: true,
    address: address,
    balance: '1.5', // Demo balance
  };
  
  await chrome.storage.local.set({ [WALLET_STATE_KEY]: currentState });
  
  // Notify all tabs
  notifyAllTabs({ type: 'WALLET_STATE_CHANGED', state: currentState });
  
  return { success: true, address };
}

async function lockWallet() {
  currentState = {
    ...currentState,
    isUnlocked: false,
    address: null,
    balance: '0',
  };
  
  await chrome.storage.local.set({ [WALLET_STATE_KEY]: currentState });
  notifyAllTabs({ type: 'WALLET_STATE_CHANGED', state: currentState });
  
  return { success: true };
}

async function setNetwork(networkId) {
  const networks = await chrome.storage.local.get(NETWORKS_KEY);
  const network = networks[NETWORKS_KEY].find(n => n.id === networkId);
  
  if (!network) {
    return { error: 'Network not found' };
  }
  
  currentState.network = network;
  await chrome.storage.local.set({ [WALLET_STATE_KEY]: currentState });
  
  notifyAllTabs({ type: 'NETWORK_CHANGED', network });
  
  return { success: true, network };
}

async function signMessage(payload) {
  if (!currentState.isUnlocked) {
    return { error: 'Wallet is locked' };
  }
  
  // In production, this would use actual cryptographic signing
  const signature = '0x' + btoa(payload.message).substring(0, 130);
  
  return { signature };
}

async function signTransaction(payload) {
  if (!currentState.isUnlocked) {
    return { error: 'Wallet is locked' };
  }
  
  // In production, this would validate and sign the transaction
  const txHash = '0x' + Math.random().toString(16).substring(2, 66).padEnd(64, '0');
  
  return { txHash };
}

async function sendTransaction(payload) {
  if (!currentState.isUnlocked) {
    return { error: 'Wallet is locked' };
  }
  
  // Simulate transaction
  const txHash = '0x' + Math.random().toString(16).substring(2, 66).padEnd(64, '0');
  
  return { txHash, status: 'pending' };
}

async function addToken(token) {
  if (!currentState.tokens.find(t => t.address === token.address)) {
    currentState.tokens.push(token);
    await chrome.storage.local.set({ [WALLET_STATE_KEY]: currentState });
  }
  return { success: true };
}

async function getBalance(address) {
  // In production, this would query the blockchain
  return { balance: '1.5', tokens: currentState.tokens };
}

function generateAddressFromSeed(seed) {
  // Simplified address generation
  const hash = btoa(seed).replace(/[^a-zA-Z0-9]/g, '').substring(0, 40);
  return '0x' + hash.padEnd(40, '0').substring(0, 40);
}

function notifyAllTabs(message) {
  chrome.tabs.query({}, (tabs) => {
    tabs.forEach(tab => {
      chrome.tabs.sendMessage(tab.id, message).catch(() => {});
    });
  });
}

// Handle DApp connections
chrome.webRequest.onBeforeRequest.addListener(
  (details) => {
    // Handle RPC requests to DApps
    if (details.url.startsWith('https://') && details.method === 'POST') {
      // Could implement request filtering here
    }
  },
  { urls: ['<all_urls>'] }
);

// Network switch handler
chrome.webNavigation.onCompleted.addListener((details) => {
  if (details.frameId === 0) {
    // Page loaded, inject Web3 provider if needed
  }
});
