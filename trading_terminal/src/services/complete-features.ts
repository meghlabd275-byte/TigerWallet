/**
 * Trading Terminal - Complete Feature Set
 * Added: MEV Protection, Session Keys, Gas Optimization, Widget SDK, Push Notifications
 */

// Re-export all trading features
export * from './additional-services';

// MEV Protection
export const mevProtection = {
  detectSandwichAttack: async (txHash: string) => {
    const response = await fetch(`http://localhost:8443/api/v1/mev/detect-sandwich?tx=${txHash}`);
    return response.json();
  },
  
  simulateTransaction: async (params: any) => {
    const response = await fetch('http://localhost:8443/api/v1/mev/simulate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params)
    });
    return response.json();
  },
  
  submitWithProtection: async (signedTx: string, protectionLevel: string = 'medium') => {
    const response = await fetch('http://localhost:8443/api/v1/mev/submit', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ signed_tx: signedTx, protection_level: protectionLevel })
    });
    return response.json();
  }
};

// Session Keys
export const sessionKeys = {
  generate: async (walletAddress: string, dappUrl: string, permissions: string[], expiresIn: number = 86400) => {
    const response = await fetch('http://localhost:8443/api/v1/session-keys', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        wallet_address: walletAddress,
        dapp_url: dappUrl,
        permissions,
        expires_in: expiresIn
      })
    });
    return response.json();
  },
  
  list: async (walletAddress: string) => {
    const response = await fetch(`http://localhost:8443/api/v1/session-keys/${walletAddress}`);
    return response.json();
  },
  
  revoke: async (walletAddress: string, sessionKeyId: string) => {
    const response = await fetch(`http://localhost:8443/api/v1/session-keys/${sessionKeyId}`, {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ wallet_address: walletAddress })
    });
    return response.ok;
  }
};

// Gas Optimization
export const gasOptimization = {
  getPrices: async (chain: string = 'ethereum') => {
    const response = await fetch(`http://localhost:8443/api/v1/gas/prices?chain=${chain}`);
    return response.json();
  },
  
  getSuggestions: async (from: string, to: string, data: string) => {
    const response = await fetch('http://localhost:8443/api/v1/gas/optimize', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ from, to, data })
    });
    return response.json();
  },
  
  estimate: async (txData: string, chain: string = 'ethereum') => {
    const response = await fetch(`http://localhost:8443/api/v1/gas/estimate?chain=${chain}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ data: txData })
    });
    return response.json();
  }
};

// Widget SDK for Trading Terminal
export const widgetSDK = {
  createBalanceWidget: (walletAddress: string) => ({
    type: 'balance',
    walletAddress,
    update: () => fetch(`http://localhost:8443/api/v1/wallet/${walletAddress}/balance`).then(r => r.json())
  }),
  
  createPriceWidget: (token: string) => ({
    type: 'price',
    token,
    update: () => fetch(`http://localhost:8443/api/v1/prices/${token}`).then(r => r.json())
  }),
  
  createPortfolioWidget: (walletAddress: string) => ({
    type: 'portfolio',
    walletAddress,
    update: () => fetch(`http://localhost:8443/api/v1/wallet/${walletAddress}/portfolio`).then(r => r.json())
  }),
  
  createQuickSendWidget: () => ({
    type: 'quick_send',
    actions: ['send', 'swap', 'bridge']
  })
};

// Push Notifications for Trading Terminal
export const pushNotifications = {
  subscribe: async (walletAddress: string, channels: string[]) => {
    const response = await fetch('http://localhost:8443/api/v1/notifications/subscribe', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ wallet_address: walletAddress, channels })
    });
    return response.json();
  },
  
  unsubscribe: async (walletAddress: string, channel: string) => {
    const response = await fetch(`http://localhost:8443/api/v1/notifications/unsubscribe/${channel}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ wallet_address: walletAddress })
    });
    return response.ok;
  },
  
  getSettings: async (walletAddress: string) => {
    const response = await fetch(`http://localhost:8443/api/v1/notifications/settings/${walletAddress}`);
    return response.json();
  },
  
  updateSettings: async (walletAddress: string, settings: any) => {
    const response = await fetch(`http://localhost:8443/api/v1/notifications/settings/${walletAddress}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(settings)
    });
    return response.json();
  }
};

// Export all services
export default {
  mevProtection,
  sessionKeys,
  gasOptimization,
  widgetSDK,
  pushNotifications
};
