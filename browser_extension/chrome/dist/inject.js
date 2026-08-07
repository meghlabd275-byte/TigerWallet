"use strict";
(() => {
  // src/content/inject.js
  (function() {
    "use strict";
    const CHAIN_IDS = {
      ethereum: "0x1",
      polygon: "0x89",
      bsc: "0x38",
      arbitrum: "0xa4b1",
      optimism: "0xa",
      avalanche: "0xa86a",
      base: "0x2105",
      zksync: "0x144",
      linea: "0xe708",
      scroll: "0x82750"
    };
    let walletState = {
      isConnected: false,
      chainId: CHAIN_IDS.ethereum,
      accounts: [],
      selectedAddress: ""
    };
    let eventListeners = /* @__PURE__ */ new Map();
    function generateId() {
      if (!globalThis.crypto || typeof globalThis.crypto.randomUUID !== "function") {
        throw new Error("Secure request ID generation is unavailable");
      }
      return `req_${globalThis.crypto.randomUUID()}`;
    }
    function notifyBackground(message) {
      return new Promise((resolve, reject) => {
        chrome.runtime.sendMessage(message, (response) => {
          if (response?.success) {
            resolve(response.data);
          } else {
            reject(new Error(response?.error || "Request failed"));
          }
        });
      });
    }
    function emit(event, data) {
      const listeners = eventListeners.get(event);
      if (listeners) {
        listeners.forEach((listener) => {
          try {
            listener(data);
          } catch (error) {
            console.error("Event listener error:", error);
          }
        });
      }
    }
    class TigerWalletProvider {
      isTigerWallet = true;
      isMetaMask = true;
      // For DApp compatibility
      isConnected = () => walletState.isConnected;
      chainId = () => walletState.chainId;
      selectedAddress = () => walletState.selectedAddress;
      _events = /* @__PURE__ */ new Map();
      async request(args) {
        const id = generateId();
        try {
          const response = await notifyBackground({
            type: "INTERNAL_REQUEST",
            payload: {
              id,
              method: args.method,
              params: args.params || [],
              origin: window.location.origin,
              chainId: walletState.chainId
            }
          });
          return response;
        } catch (error) {
          console.error("Provider request error:", error);
          throw error;
        }
      }
      on(event, listener) {
        if (!this._events.has(event)) {
          this._events.set(event, /* @__PURE__ */ new Set());
        }
        this._events.get(event).add(listener);
      }
      removeListener(event, listener) {
        this._events.get(event)?.delete(listener);
      }
      emit(event, ...args) {
        this._events.get(event)?.forEach((listener) => {
          try {
            listener(...args);
          } catch (error) {
            console.error("Event emission error:", error);
          }
        });
      }
      // Compatibility methods
      enable() {
        return this.request({ method: "eth_requestAccounts" });
      }
      async send(method, params) {
        return this.request({ method, params });
      }
      async sendAsync(payload, callback) {
        try {
          const result = await this.request(payload);
          callback(null, { id: payload.id, jsonrpc: "2.0", result });
        } catch (error) {
          callback(error, null);
        }
      }
      // Event emitter compatibility
      addListener(event, listener) {
        this.on(event, listener);
      }
      removeAllListeners(event) {
        if (event) {
          this._events.delete(event);
        } else {
          this._events.clear();
        }
      }
      listenerCount(event) {
        if (event) {
          return this._events.get(event)?.size || 0;
        }
        let count = 0;
        this._events.forEach((set) => count += set.size);
        return count;
      }
    }
    const provider = new TigerWalletProvider();
    function handleBackgroundMessage(message) {
      switch (message.type) {
        case "CONNECTION_CHANGED":
          walletState.isConnected = message.payload.connected;
          walletState.accounts = message.payload.accounts || [];
          walletState.selectedAddress = walletState.accounts[0] || "";
          provider.emit("connect", { chainId: walletState.chainId });
          provider.emit("accountsChanged", walletState.accounts);
          emit("accountsChanged", walletState.accounts);
          break;
        case "CHAIN_CHANGED":
          walletState.chainId = message.payload.chainId;
          provider.emit("chainChanged", message.payload.chainId);
          emit("chainChanged", message.payload.chainId);
          break;
        case "TRANSACTION_RESULT":
          provider.emit("transactionResult", message.payload);
          emit("transactionResult", message.payload);
          break;
      }
    }
    chrome.runtime?.onMessage?.addListener((message, sender, sendResponse) => {
      handleBackgroundMessage(message);
      sendResponse({ received: true });
    });
    function initializeProvider() {
      const existingProvider = window.ethereum;
      if (existingProvider) {
        window.tigerwallet = provider;
      } else {
        window.ethereum = provider;
        window.web3 = {
          currentProvider: provider
        };
      }
      window.tigerwallet = provider;
      console.log("TigerWallet provider initialized");
    }
    async function checkConnection() {
      try {
        const connections = await notifyBackground({
          type: "GET_CONNECTIONS"
        });
        const currentConnection = connections?.find(
          (c) => c.origin === window.location.origin
        );
        if (currentConnection?.connected) {
          walletState.isConnected = true;
          walletState.accounts = currentConnection.accounts || [];
          walletState.selectedAddress = walletState.accounts[0] || "";
          walletState.chainId = currentConnection.chainId || CHAIN_IDS.ethereum;
          provider.emit("connect", { chainId: walletState.chainId });
        }
      } catch (error) {
        console.error("Connection check failed:", error);
      }
    }
    function init() {
      initializeProvider();
      checkConnection();
    }
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", init);
    } else {
      init();
    }
    if (window.frameElement) {
      init();
    }
  })();
})();
