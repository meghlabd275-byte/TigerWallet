// src/background/service-worker.js
var ExtensionState = class {
  wallet = {
    isUnlocked: false,
    currentChain: "ethereum",
    addresses: {},
    balance: {}
  };
  connections = /* @__PURE__ */ new Map();
  transactions = /* @__PURE__ */ new Map();
  pendingRequests = /* @__PURE__ */ new Map();
  constructor() {
    this.loadState();
  }
  async loadState() {
    try {
      const result = await chrome.storage.local.get([
        "wallet",
        "connections",
        "transactions"
      ]);
      if (result.wallet) {
        this.wallet = { ...this.wallet, ...result.wallet };
      }
      if (result.connections) {
        this.connections = new Map(Object.entries(result.connections));
      }
      if (result.transactions) {
        this.transactions = new Map(Object.entries(result.transactions));
      }
    } catch (error) {
      console.error("Failed to load state:", error);
    }
  }
  async saveState() {
    try {
      await chrome.storage.local.set({
        wallet: this.wallet,
        connections: Object.fromEntries(this.connections),
        transactions: Object.fromEntries(this.transactions)
      });
    } catch (error) {
      console.error("Failed to save state:", error);
    }
  }
  // Wallet methods
  getWallet() {
    return this.wallet;
  }
  setUnlocked(unlocked) {
    this.wallet.isUnlocked = unlocked;
    this.saveState();
  }
  setAddress(chain, address) {
    this.wallet.addresses[chain] = address;
    this.saveState();
  }
  setBalance(chain, balance) {
    this.wallet.balance[chain] = balance;
    this.saveState();
  }
  setCurrentChain(chain) {
    this.wallet.currentChain = chain;
    this.saveState();
  }
  // Connection methods
  addConnection(origin, connection) {
    this.connections.set(origin, connection);
    this.saveState();
  }
  removeConnection(origin) {
    this.connections.delete(origin);
    this.saveState();
  }
  getConnection(origin) {
    return this.connections.get(origin);
  }
  getAllConnections() {
    return Array.from(this.connections.values());
  }
  // Transaction methods
  addTransaction(id, tx) {
    this.transactions.set(id, tx);
    this.saveState();
  }
  getTransaction(id) {
    return this.transactions.get(id);
  }
  removeTransaction(id) {
    this.transactions.delete(id);
    this.saveState();
  }
  getAllTransactions() {
    return Array.from(this.transactions.values());
  }
};
var state = new ExtensionState();
async function handleMessage(message, sender) {
  try {
    switch (message.type) {
      // Wallet state
      case "GET_WALLET_STATE":
        return { success: true, data: state.getWallet() };
      case "UNLOCK_WALLET":
        return { success: false, error: "Wallet unlock is unavailable until the canonical wallet-core bridge is connected." };
      case "LOCK_WALLET":
        state.setUnlocked(false);
        return { success: true };
      case "SET_CHAIN":
        state.setCurrentChain(message.payload.chain);
        return { success: true };
      // Connection management
      case "CONNECT_DAPP":
        const account = state.getWallet().addresses["ethereum"];
        if (!account) {
          return { success: false, error: "No verified wallet account is available." };
        }
        const connection = {
          origin: message.payload.origin,
          chainId: message.payload.chainId || "ethereum",
          connected: true,
          accounts: [account]
        };
        state.addConnection(message.payload.origin, connection);
        notifyContentScript(message.payload.origin, {
          type: "CONNECTION_CHANGED",
          payload: { connected: true, accounts: connection.accounts }
        });
        return { success: true, data: connection };
      case "DISCONNECT_DAPP":
        state.removeConnection(message.payload.origin);
        notifyContentScript(message.payload.origin, {
          type: "CONNECTION_CHANGED",
          payload: { connected: false, accounts: [] }
        });
        return { success: true };
      case "GET_CONNECTIONS":
        return { success: true, data: state.getAllConnections() };
      // Transaction signing
      case "SIGN_TRANSACTION":
        return { success: false, error: "Transaction signing is unavailable until the canonical wallet-core bridge and approval UI are connected." };
      case "APPROVE_TRANSACTION":
        return { success: false, error: "Transaction signing is unavailable until the canonical wallet-core bridge and approval UI are connected." };
      case "REJECT_TRANSACTION":
        const rejectedTx = state.getTransaction(message.payload.id);
        if (rejectedTx) {
          state.removeTransaction(message.payload.id);
          notifyContentScript(rejectedTx.origin, {
            type: "TRANSACTION_RESULT",
            payload: { id: rejectedTx.id, status: "rejected" }
          });
        }
        return { success: true };
      // Network switching
      case "SWITCH_CHAIN":
        state.setCurrentChain(message.payload.chainId);
        for (const conn of state.getAllConnections()) {
          notifyContentScript(conn.origin, {
            type: "CHAIN_CHANGED",
            payload: { chainId: message.payload.chainId }
          });
        }
        return { success: true };
      // Personal signing
      case "SIGN_MESSAGE":
        return { success: false, error: "Message signing is unavailable until the canonical wallet-core bridge is connected." };
      // Token/balance queries
      case "GET_BALANCE":
        const wallet = state.getWallet();
        return {
          success: true,
          data: wallet.balance[message.payload.chain] || "0"
        };
      case "GET_ADDRESS":
        const addr = state.getWallet();
        return {
          success: true,
          data: addr.addresses[message.payload.chain] || ""
        };
      default:
        return { success: false, error: `Unknown message type: ${message.type}` };
    }
  } catch (error) {
    console.error("Message handling error:", error);
    return { success: false, error: String(error) };
  }
}
function notifyContentScript(origin, message) {
  chrome.tabs.query({ url: origin + "*" }, (tabs) => {
    for (const tab of tabs) {
      if (tab.id) {
        chrome.tabs.sendMessage(tab.id, message).catch(() => {
        });
      }
    }
  });
}
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  handleMessage(message, sender).then(sendResponse);
  return true;
});
chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  if (changeInfo.status === "complete" && tab.url) {
    checkAndInjectDApp(tabId, tab.url);
  }
});
chrome.webNavigation.onCompleted.addListener((details) => {
  if (details.frameId === 0) {
    console.log("Navigation completed:", details.url);
  }
});
chrome.runtime.onInstalled.addListener((details) => {
  if (details.reason === "install") {
    console.log("TigerWallet extension installed");
    state.setCurrentChain("ethereum");
  } else if (details.reason === "update") {
    console.log("TigerWallet extension updated to version:", chrome.runtime.getManifest().version);
  }
});
chrome.runtime.onStartup.addListener(() => {
  console.log("TigerWallet extension starting...");
  state.loadState();
});
function setupContextMenus() {
  chrome.contextMenus?.create({
    id: "tiger-send",
    title: "Send with TigerWallet",
    contexts: ["selection"]
  });
  chrome.contextMenus?.create({
    id: "tiger-view-address",
    title: "View Address in TigerWallet",
    contexts: ["page"]
  });
  chrome.contextMenus?.onClicked.addListener((info, tab) => {
    if (info.menuItemId === "tiger-send" && info.selectionText) {
      chrome.runtime.sendMessage({
        type: "OPEN_SEND_DIALOG",
        payload: { data: info.selectionText }
      });
    }
  });
}
setupContextMenus();
chrome.alarms.create("updateBalances", { periodInMinutes: 5 });
chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === "updateBalances") {
    console.log("Updating balances...");
  }
});
chrome.alarms.create("cleanupTransactions", { periodInMinutes: 10 });
chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === "cleanupTransactions") {
    const now = Date.now();
    const maxAge = 5 * 60 * 1e3;
    for (const [id, tx] of state.getAllTransactions()) {
      if (now - tx.timestamp > maxAge) {
        state.removeTransaction(id);
      }
    }
  }
});
