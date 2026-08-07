/**
 * TigerWallet Admin - API Service
 * Complete backend connectivity for all features
 */

import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'https://api.tigerwallet.com/admin/v1';

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor for auth token
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('admin_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor for error handling
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('admin_token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// Auth API
export const authAPI = {
  login: (email: string, password: string) =>
    apiClient.post('/auth/login', { email, password }),
  logout: () => apiClient.post('/auth/logout'),
  refreshToken: (refreshToken: string) =>
    apiClient.post('/auth/refresh', { refreshToken }),
  verify2FA: (code: string) => apiClient.post('/auth/verify-2fa', { code }),
  enable2FA: () => apiClient.post('/auth/enable-2fa'),
  disable2FA: (password: string) => apiClient.post('/auth/disable-2fa', { password }),
};

// Users API
export const userAPI = {
  getAll: (page = 1, limit = 20, filters = {}) =>
    apiClient.get('/users', { params: { page, limit, ...filters } }),
  getById: (id: string) => apiClient.get(`/users/${id}`),
  create: (data: any) => apiClient.post('/users', data),
  update: (id: string, data: any) => apiClient.put(`/users/${id}`, data),
  delete: (id: string) => apiClient.delete(`/users/${id}`),
  updateStatus: (id: string, status: string) =>
    apiClient.put(`/users/${id}/status`, { status }),
  search: (query: string) => apiClient.get('/users/search', { params: { q: query } }),
};

// KYC API
export const kycAPI = {
  getAll: (status?: string, page = 1) =>
    apiClient.get('/kyc', { params: { status, page } }),
  getById: (id: string) => apiClient.get(`/kyc/${id}`),
  approve: (id: string, data?: any) =>
    apiClient.post(`/kyc/${id}/approve`, data),
  reject: (id: string, reason: string) =>
    apiClient.post(`/kyc/${id}/reject`, { reason }),
  requestInfo: (id: string, message: string) =>
    apiClient.post(`/kyc/${id}/request-info`, { message }),
};

// Transactions API
export const transactionAPI = {
  getAll: (page = 1, limit = 20, filters = {}) =>
    apiClient.get('/transactions', { params: { page, limit, ...filters } }),
  getById: (id: string) => apiClient.get(`/transactions/${id}`),
  flag: (id: string, reason: string) =>
    apiClient.post(`/transactions/${id}/flag`, { reason }),
  unflag: (id: string) => apiClient.post(`/transactions/${id}/unflag`),
  approve: (id: string) => apiClient.post(`/transactions/${id}/approve`),
  reject: (id: string, reason: string) =>
    apiClient.post(`/transactions/${id}/reject`, { reason }),
  getStats: () => apiClient.get('/transactions/stats'),
};

// Tokens API
export const tokenAPI = {
  getAll: () => apiClient.get('/tokens'),
  getById: (id: string) => apiClient.get(`/tokens/${id}`),
  create: (data: any) => apiClient.post('/tokens', data),
  update: (id: string, data: any) => apiClient.put(`/tokens/${id}`, data),
  delete: (id: string) => apiClient.delete(`/tokens/${id}`),
  updateStatus: (id: string, status: string) =>
    apiClient.put(`/tokens/${id}/status`, { status }),
};

// Pairs API
export const pairAPI = {
  getAll: () => apiClient.get('/pairs'),
  getById: (id: string) => apiClient.get(`/pairs/${id}`),
  create: (data: any) => apiClient.post('/pairs', data),
  update: (id: string, data: any) => apiClient.put(`/pairs/${id}`, data),
  updateStatus: (id: string, status: string) =>
    apiClient.put(`/pairs/${id}/status`, { status }),
};

// Blockchains API
export const blockchainAPI = {
  getAll: () => apiClient.get('/blockchains'),
  getById: (id: string) => apiClient.get(`/blockchains/${id}`),
  create: (data: any) => apiClient.post('/blockchains', data),
  update: (id: string, data: any) => apiClient.put(`/blockchains/${id}`, data),
  delete: (id: string) => apiClient.delete(`/blockchains/${id}`),
};

// White Labels API
export const whiteLabelAPI = {
  getAll: () => apiClient.get('/whitelabels'),
  getById: (id: string) => apiClient.get(`/whitelabels/${id}`),
  create: (data: any) => apiClient.post('/whitelabels', data),
  update: (id: string, data: any) => apiClient.put(`/whitelabels/${id}`, data),
  delete: (id: string) => apiClient.delete(`/whitelabels/${id}`),
  updateStatus: (id: string, status: string) =>
    apiClient.put(`/whitelabels/${id}/status`, { status }),
};

// Support Tickets API
export const ticketAPI = {
  getAll: (status?: string, page = 1) =>
    apiClient.get('/tickets', { params: { status, page } }),
  getById: (id: string) => apiClient.get(`/tickets/${id}`),
  create: (data: any) => apiClient.post('/tickets', data),
  update: (id: string, data: any) => apiClient.put(`/tickets/${id}`, data),
  assign: (id: string, adminId: string) =>
    apiClient.post(`/tickets/${id}/assign`, { adminId }),
  addReply: (id: string, message: string) =>
    apiClient.post(`/tickets/${id}/reply`, { message }),
  close: (id: string) => apiClient.post(`/tickets/${id}/close`),
};

// Analytics API
export const analyticsAPI = {
  getDashboard: () => apiClient.get('/analytics/dashboard'),
  getUsers: (period = '7d') =>
    apiClient.get('/analytics/users', { params: { period } }),
  getTransactions: (period = '7d') =>
    apiClient.get('/analytics/transactions', { params: { period } }),
  getRevenue: (period = '30d') =>
    apiClient.get('/analytics/revenue', { params: { period } }),
  getKYC: (period = '30d') =>
    apiClient.get('/analytics/kyc', { params: { period } }),
};

// Crypto Cards API
export const cryptoCardAPI = {
  getAll: (status?: string) =>
    apiClient.get('/crypto-cards', { params: { status } }),
  getById: (id: string) => apiClient.get(`/crypto-cards/${id}`),
  create: (data: any) => apiClient.post('/crypto-cards', data),
  block: (id: string) => apiClient.post(`/crypto-cards/${id}/block`),
  activate: (id: string) => apiClient.post(`/crypto-cards/${id}/activate`),
  setLimit: (id: string, limit: number) =>
    apiClient.put(`/crypto-cards/${id}/limit`, { limit }),
};

// Margin Trading API
export const marginTradingAPI = {
  getPositions: () => apiClient.get('/margin/positions'),
  getHistory: (page = 1) => apiClient.get('/margin/history', { params: { page } }),
  liquidate: (positionId: string) =>
    apiClient.post(`/margin/positions/${positionId}/liquidate`),
  getLiquidationStats: () => apiClient.get('/margin/liquidation-stats'),
};

// P2P Merchant API
export const p2pMerchantAPI = {
  getMerchants: (status?: string) =>
    apiClient.get('/p2p/merchants', { params: { status } }),
  getMerchantById: (id: string) => apiClient.get(`/p2p/merchants/${id}`),
  approveMerchant: (id: string) =>
    apiClient.post(`/p2p/merchants/${id}/approve`),
  rejectMerchant: (id: string, reason: string) =>
    apiClient.post(`/p2p/merchants/${id}/reject`, { reason }),
  getTransactions: (merchantId: string, page = 1) =>
    apiClient.get(`/p2p/merchants/${merchantId}/transactions`, { params: { page } }),
};

// Liquidity API
export const liquidityAPI = {
  getPools: () => apiClient.get('/liquidity/pools'),
  getPoolById: (id: string) => apiClient.get(`/liquidity/pools/${id}`),
  createPool: (data: any) => apiClient.post('/liquidity/pools', data),
  addLiquidity: (poolId: string, data: any) =>
    apiClient.post(`/liquidity/pools/${poolId}/add`, data),
  removeLiquidity: (poolId: string, data: any) =>
    apiClient.post(`/liquidity/pools/${poolId}/remove`, data),
  getStats: () => apiClient.get('/liquidity/stats'),
};

// Features API
export const featuresAPI = {
  getAll: () => apiClient.get('/features'),
  getById: (id: string) => apiClient.get(`/features/${id}`),
  create: (data: any) => apiClient.post('/features', data),
  update: (id: string, data: any) => apiClient.put(`/features/${id}`, data),
  toggle: (id: string) => apiClient.post(`/features/${id}/toggle`),
};

// Master Wallet API
export const masterWalletAPI = {
  getWallets: () => apiClient.get('/master-wallets'),
  getWalletById: (id: string) => apiClient.get(`/master-wallets/${id}`),
  createWallet: (data: any) => apiClient.post('/master-wallets', data),
  getBalance: (walletId: string) =>
    apiClient.get(`/master-wallets/${walletId}/balance`),
  getTransactions: (walletId: string, page = 1) =>
    apiClient.get(`/master-wallets/${walletId}/transactions`, { params: { page } }),
  transfer: (walletId: string, data: any) =>
    apiClient.post(`/master-wallets/${walletId}/transfer`, data),
};

// Billing API
export const billingAPI = {
  getPlans: () => apiClient.get('/billing/plans'),
  getSubscription: () => apiClient.get('/billing/subscription'),
  createSubscription: (data: any) =>
    apiClient.post('/billing/subscription', data),
  updateSubscription: (data: any) =>
    apiClient.put('/billing/subscription', data),
  cancelSubscription: () => apiClient.post('/billing/subscription/cancel'),
  getInvoices: (page = 1) =>
    apiClient.get('/billing/invoices', { params: { page } }),
  getPaymentMethods: () => apiClient.get('/billing/payment-methods'),
  addPaymentMethod: (data: any) =>
    apiClient.post('/billing/payment-methods', data),
};

// Settings API
export const settingsAPI = {
  getProfile: () => apiClient.get('/settings/profile'),
  updateProfile: (data: any) => apiClient.put('/settings/profile', data),
  changePassword: (data: any) => apiClient.post('/settings/change-password', data),
  getPreferences: () => apiClient.get('/settings/preferences'),
  updatePreferences: (data: any) =>
    apiClient.put('/settings/preferences', data),
  getSecurity: () => apiClient.get('/settings/security'),
  updateSecurity: (data: any) => apiClient.put('/settings/security', data),
};

export default apiClient;
