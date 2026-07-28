// ============================================================================
// TigerWallet Admin API Service - Production Ready
// ============================================================================

import axios, { AxiosInstance, AxiosError } from 'axios';
import {
  User, Admin, WhiteLabelClient, TradingPair, Token,
  Transaction, Withdrawal, FeeStructure, ApiKey, Analytics,
  Notification, Blockchain, AuditLog, KYCSubmission, Broker,
  InstitutionalClient, MarketMakerBot, NFT, MultisigWallet
} from '../types/admin';

// API Configuration
const API_BASE_URL = import.meta.env.VITE_API_URL || 'https://api.tigerwallet.com/v1';

// Create axios instance with interceptors
const createApiClient = (): AxiosInstance => {
  const client = axios.create({
    baseURL: API_BASE_URL,
    headers: {
      'Content-Type': 'application/json',
    },
  });

  // Request interceptor for auth
  client.interceptors.request.use(
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
  client.interceptors.response.use(
    (response) => response,
    (error: AxiosError) => {
      if (error.response?.status === 401) {
        localStorage.removeItem('admin_token');
        window.location.href = '/login';
      }
      return Promise.reject(error);
    }
  );

  return client;
};

const api = createApiClient();

// ==================== AUTH SERVICE ====================
export const authService = {
  login: async (email: string, password: string) => {
    const response = await api.post('/auth/login', { email, password });
    localStorage.setItem('admin_token', response.data.token);
    return response.data;
  },

  logout: async () => {
    await api.post('/auth/logout');
    localStorage.removeItem('admin_token');
  },

  getCurrentAdmin: async () => {
    const response = await api.get('/auth/me');
    return response.data;
  },

  changePassword: async (oldPassword: string, newPassword: string) => {
    const response = await api.post('/auth/change-password', { oldPassword, newPassword });
    return response.data;
  },

  enable2FA: async () => {
    const response = await api.post('/auth/2fa/enable');
    return response.data;
  },

  verify2FA: async (code: string) => {
    const response = await api.post('/auth/2fa/verify', { code });
    return response.data;
  },
};

// ==================== USER MANAGEMENT ====================
export const userService = {
  getAllUsers: async (params?: {
    page?: number;
    limit?: number;
    status?: string;
    kycStatus?: string;
    search?: string;
  }) => {
    const response = await api.get('/users', { params });
    return response.data;
  },

  getUserById: async (id: string) => {
    const response = await api.get(`/users/${id}`);
    return response.data;
  },

  updateUser: async (id: string, data: Partial<User>) => {
    const response = await api.put(`/users/${id}`, data);
    return response.data;
  },

  suspendUser: async (id: string, reason: string) => {
    const response = await api.post(`/users/${id}/suspend`, { reason });
    return response.data;
  },

  banUser: async (id: string, reason: string) => {
    const response = await api.post(`/users/${id}/ban`, { reason });
    return response.data;
  },

  getUserTransactions: async (userId: string, params?: { page?: number; limit?: number }) => {
    const response = await api.get(`/users/${userId}/transactions`, { params });
    return response.data;
  },

  getUserWallets: async (userId: string) => {
    const response = await api.get(`/users/${userId}/wallets`);
    return response.data;
  },

  resetUser2FA: async (userId: string) => {
    const response = await api.post(`/users/${userId}/reset-2fa`);
    return response.data;
  },
};

// ==================== ADMIN MANAGEMENT ====================
export const adminService = {
  getAllAdmins: async (params?: { page?: number; limit?: number; role?: string }) => {
    const response = await api.get('/admins', { params });
    return response.data;
  },

  createAdmin: async (data: Partial<Admin>) => {
    const response = await api.post('/admins', data);
    return response.data;
  },

  updateAdmin: async (id: string, data: Partial<Admin>) => {
    const response = await api.put(`/admins/${id}`, data);
    return response.data;
  },

  deleteAdmin: async (id: string) => {
    const response = await api.delete(`/admins/${id}`);
    return response.data;
  },

  updateAdminPermissions: async (id: string, permissions: string[]) => {
    const response = await api.put(`/admins/${id}/permissions`, { permissions });
    return response.data;
  },

  getAdminActivity: async (adminId: string, params?: { page?: number; limit?: number }) => {
    const response = await api.get(`/admins/${adminId}/activity`, { params });
    return response.data;
  },
};

// ==================== KYC MANAGEMENT ====================
export const kycService = {
  getSubmissions: async (params?: {
    page?: number;
    limit?: number;
    status?: string;
    level?: number;
  }) => {
    const response = await api.get('/kyc', { params });
    return response.data;
  },

  getSubmission: async (id: string) => {
    const response = await api.get(`/kyc/${id}`);
    return response.data;
  },

  approveSubmission: async (id: string, notes?: string) => {
    const response = await api.post(`/kyc/${id}/approve`, { notes });
    return response.data;
  },

  rejectSubmission: async (id: string, reason: string) => {
    const response = await api.post(`/kyc/${id}/reject`, { reason });
    return response.data;
  },

  requestAdditionalInfo: async (id: string, message: string) => {
    const response = await api.post(`/kyc/${id}/request-info`, { message });
    return response.data;
  },
};

// ==================== WHITE LABEL MANAGEMENT ====================
export const whiteLabelService = {
  getAllClients: async (params?: {
    page?: number;
    limit?: number;
    status?: string;
    plan?: string;
  }) => {
    const response = await api.get('/white-label', { params });
    return response.data;
  },

  getClient: async (id: string) => {
    const response = await api.get(`/white-label/${id}`);
    return response.data;
  },

  createClient: async (data: Partial<WhiteLabelClient>) => {
    const response = await api.post('/white-label', data);
    return response.data;
  },

  updateClient: async (id: string, data: Partial<WhiteLabelClient>) => {
    const response = await api.put(`/white-label/${id}`, data);
    return response.data;
  },

  suspendClient: async (id: string, reason: string) => {
    const response = await api.post(`/white-label/${id}/suspend`, { reason });
    return response.data;
  },

  resumeClient: async (id: string) => {
    const response = await api.post(`/white-label/${id}/resume`);
    return response.data;
  },

  deleteClient: async (id: string) => {
    const response = await api.delete(`/white-label/${id}`);
    return response.data;
  },

  updateBranding: async (id: string, branding: WhiteLabelClient['branding']) => {
    const response = await api.put(`/white-label/${id}/branding`, { branding });
    return response.data;
  },

  getClientUsers: async (clientId: string, params?: { page?: number; limit?: number }) => {
    const response = await api.get(`/white-label/${clientId}/users`, { params });
    return response.data;
  },

  setClientFeatures: async (id: string, features: WhiteLabelClient['features']) => {
    const response = await api.put(`/white-label/${id}/features`, { features });
    return response.data;
  },
};

// ==================== TRADING PAIRS ====================
export const pairService = {
  getAllPairs: async (params?: {
    page?: number;
    limit?: number;
    status?: string;
    type?: string;
  }) => {
    const response = await api.get('/pairs', { params });
    return response.data;
  },

  getPair: async (id: string) => {
    const response = await api.get(`/pairs/${id}`);
    return response.data;
  },

  createPair: async (data: Partial<TradingPair>) => {
    const response = await api.post('/pairs', data);
    return response.data;
  },

  updatePair: async (id: string, data: Partial<TradingPair>) => {
    const response = await api.put(`/pairs/${id}`, data);
    return response.data;
  },

  deletePair: async (id: string) => {
    const response = await api.delete(`/pairs/${id}`);
    return response.data;
  },

  suspendPair: async (id: string, reason: string) => {
    const response = await api.post(`/pairs/${id}/suspend`, { reason });
    return response.data;
  },

  resumePair: async (id: string) => {
    const response = await api.post(`/pairs/${id}/resume`);
    return response.data;
  },

  importFromCEX: async (cex: string, pairs: string[]) => {
    const response = await api.post('/pairs/import', { cex, pairs });
    return response.data;
  },

  setPairFees: async (id: string, makerFee: string, takerFee: string) => {
    const response = await api.put(`/pairs/${id}/fees`, { makerFee, takerFee });
    return response.data;
  },
};

// ==================== TOKEN MANAGEMENT ====================
export const tokenService = {
  getAllTokens: async (params?: {
    page?: number;
    limit?: number;
    status?: string;
    chain?: string;
    search?: string;
  }) => {
    const response = await api.get('/tokens', { params });
    return response.data;
  },

  getToken: async (id: string) => {
    const response = await api.get(`/tokens/${id}`);
    return response.data;
  },

  createToken: async (data: Partial<Token>) => {
    const response = await api.post('/tokens', data);
    return response.data;
  },

  updateToken: async (id: string, data: Partial<Token>) => {
    const response = await api.put(`/tokens/${id}`, data);
    return response.data;
  },

  deleteToken: async (id: string) => {
    const response = await api.delete(`/tokens/${id}`);
    return response.data;
  },

  suspendToken: async (id: string, reason: string) => {
    const response = await api.post(`/tokens/${id}/suspend`, { reason });
    return response.data;
  },

  createTokenOnChain: async (chainId: string, data: any) => {
    const response = await api.post(`/chains/${chainId}/tokens`, data);
    return response.data;
  },
};

// ==================== BLOCKCHAIN MANAGEMENT ====================
export const blockchainService = {
  getAllChains: async (params?: { page?: number; limit?: number; status?: string }) => {
    const response = await api.get('/chains', { params });
    return response.data;
  },

  getChain: async (id: string) => {
    const response = await api.get(`/chains/${id}`);
    return response.data;
  },

  addChain: async (data: Partial<Blockchain>) => {
    const response = await api.post('/chains', data);
    return response.data;
  },

  updateChain: async (id: string, data: Partial<Blockchain>) => {
    const response = await api.put(`/chains/${id}`, data);
    return response.data;
  },

  deleteChain: async (id: string) => {
    const response = await api.delete(`/chains/${id}`);
    return response.data;
  },

  setChainStatus: async (id: string, status: 'active' | 'inactive' | 'maintenance') => {
    const response = await api.put(`/chains/${id}/status`, { status });
    return response.data;
  },

  addRPC: async (chainId: string, rpcUrl: string) => {
    const response = await api.post(`/chains/${chainId}/rpc`, { rpcUrl });
    return response.data;
  },
};

// ==================== TRANSACTIONS ====================
export const transactionService = {
  getAllTransactions: async (params?: {
    page?: number;
    limit?: number;
    status?: string;
    type?: string;
    chain?: string;
    userId?: string;
    hash?: string;
    dateFrom?: string;
    dateTo?: string;
  }) => {
    const response = await api.get('/transactions', { params });
    return response.data;
  },

  getTransaction: async (id: string) => {
    const response = await api.get(`/transactions/${id}`);
    return response.data;
  },

  getTransactionByHash: async (hash: string) => {
    const response = await api.get(`/transactions/hash/${hash}`);
    return response.data;
  },

  cancelTransaction: async (id: string, reason: string) => {
    const response = await api.post(`/transactions/${id}/cancel`, { reason });
    return response.data;
  },
};

// ==================== WITHDRAWALS ====================
export const withdrawalService = {
  getAllWithdrawals: async (params?: {
    page?: number;
    limit?: number;
    status?: string;
    chain?: string;
    token?: string;
    userId?: string;
  }) => {
    const response = await api.get('/withdrawals', { params });
    return response.data;
  },

  getWithdrawal: async (id: string) => {
    const response = await api.get(`/withdrawals/${id}`);
    return response.data;
  },

  approveWithdrawal: async (id: string, approvedBy: string) => {
    const response = await api.post(`/withdrawals/${id}/approve`, { approvedBy });
    return response.data;
  },

  rejectWithdrawal: async (id: string, reason: string, rejectedBy: string) => {
    const response = await api.post(`/withdrawals/${id}/reject`, { reason, rejectedBy });
    return response.data;
  },

  processWithdrawal: async (id: string, txHash: string) => {
    const response = await api.post(`/withdrawals/${id}/process`, { txHash });
    return response.data;
  },

  batchApprove: async (ids: string[]) => {
    const response = await api.post('/withdrawals/batch-approve', { ids });
    return response.data;
  },

  setWithdrawalLimits: async (userId: string, limits: { daily?: string; monthly?: string }) => {
    const response = await api.put(`/withdrawals/limits/${userId}`, limits);
    return response.data;
  },
};

// ==================== FEES ====================
export const feeService = {
  getAllFees: async (params?: { type?: string; asset?: string }) => {
    const response = await api.get('/fees', { params });
    return response.data;
  },

  createFee: async (data: Partial<FeeStructure>) => {
    const response = await api.post('/fees', data);
    return response.data;
  },

  updateFee: async (id: string, data: Partial<FeeStructure>) => {
    const response = await api.put(`/fees/${id}`, data);
    return response.data;
  },

  deleteFee: async (id: string) => {
    const response = await api.delete(`/fees/${id}`);
    return response.data;
  },

  setGlobalFees: async (fees: Partial<FeeStructure>[]) => {
    const response = await api.put('/fees/global', { fees });
    return response.data;
  },
};

// ==================== API KEYS ====================
export const apiKeyService = {
  getAllKeys: async (params?: { page?: number; limit?: number; userId?: string }) => {
    const response = await api.get('/api-keys', { params });
    return response.data;
  },

  createKey: async (data: { name: string; permissions: string[]; ipWhitelist?: string[]; rateLimit?: number }) => {
    const response = await api.post('/api-keys', data);
    return response.data;
  },

  revokeKey: async (id: string) => {
    const response = await api.post(`/api-keys/${id}/revoke`);
    return response.data;
  },

  updateKey: async (id: string, data: { name?: string; permissions?: string[]; ipWhitelist?: string[] }) => {
    const response = await api.put(`/api-keys/${id}`, data);
    return response.data;
  },
};

// ==================== ANALYTICS ====================
export const analyticsService = {
  getOverview: async () => {
    const response = await api.get('/analytics/overview');
    return response.data;
  },

  getUserAnalytics: async (params?: { period?: string }) => {
    const response = await api.get('/analytics/users', { params });
    return response.data;
  },

  getTradingAnalytics: async (params?: { period?: string; pairId?: string }) => {
    const response = await api.get('/analytics/trading', { params });
    return response.data;
  },

  getRevenueAnalytics: async (params?: { period?: string }) => {
    const response = await api.get('/analytics/revenue', { params });
    return response.data;
  },

  getChainAnalytics: async (params?: { period?: string }) => {
    const response = await api.get('/analytics/chains', { params });
    return response.data;
  },

  getReport: async (type: string, params: any) => {
    const response = await api.get(`/analytics/reports/${type}`, { params });
    return response.data;
  },

  exportReport: async (type: string, format: 'csv' | 'xlsx' | 'pdf', params: any) => {
    const response = await api.get(`/analytics/reports/${type}/export`, {
      params: { ...params, format },
      responseType: 'blob',
    });
    return response.data;
  },
};

// ==================== NOTIFICATIONS ====================
export const notificationService = {
  getAll: async (params?: { page?: number; limit?: number; status?: string }) => {
    const response = await api.get('/notifications', { params });
    return response.data;
  },

  create: async (data: Partial<Notification>) => {
    const response = await api.post('/notifications', data);
    return response.data;
  },

  markAsRead: async (id: string) => {
    const response = await api.put(`/notifications/${id}/read`);
    return response.data;
  },

  markAllAsRead: async () => {
    const response = await api.put('/notifications/read-all');
    return response.data;
  },

  delete: async (id: string) => {
    const response = await api.delete(`/notifications/${id}`);
    return response.data;
  },

  sendToUser: async (userId: string, notification: Partial<Notification>) => {
    const response = await api.post(`/notifications/user/${userId}`, notification);
    return response.data;
  },

  broadcast: async (notification: Partial<Notification>) => {
    const response = await api.post('/notifications/broadcast', notification);
    return response.data;
  },
};

// ==================== MARKET MAKER ====================
export const marketMakerService = {
  getAllBots: async (params?: { page?: number; limit?: number; status?: string; userId?: string }) => {
    const response = await api.get('/market-maker', { params });
    return response.data;
  },

  getBot: async (id: string) => {
    const response = await api.get(`/market-maker/${id}`);
    return response.data;
  },

  createBot: async (data: Partial<MarketMakerBot>) => {
    const response = await api.post('/market-maker', data);
    return response.data;
  },

  updateBot: async (id: string, data: Partial<MarketMakerBot>) => {
    const response = await api.put(`/market-maker/${id}`, data);
    return response.data;
  },

  startBot: async (id: string) => {
    const response = await api.post(`/market-maker/${id}/start`);
    return response.data;
  },

  stopBot: async (id: string) => {
    const response = await api.post(`/market-maker/${id}/stop`);
    return response.data;
  },

  pauseBot: async (id: string) => {
    const response = await api.post(`/market-maker/${id}/pause`);
    return response.data;
  },

  deleteBot: async (id: string) => {
    const response = await api.delete(`/market-maker/${id}`);
    return response.data;
  },

  getBotHistory: async (id: string, params?: { page?: number; limit?: number }) => {
    const response = await api.get(`/market-maker/${id}/history`, { params });
    return response.data;
  },
};

// ==================== NFT MANAGEMENT ====================
export const nftService = {
  getAllNFTs: async (params?: { page?: number; limit?: number; status?: string; chain?: string }) => {
    const response = await api.get('/nfts', { params });
    return response.data;
  },

  getNFT: async (id: string) => {
    const response = await api.get(`/nfts/${id}`);
    return response.data;
  },

  createNFT: async (data: Partial<NFT>) => {
    const response = await api.post('/nfts', data);
    return response.data;
  },

  updateNFT: async (id: string, data: Partial<NFT>) => {
    const response = await api.put(`/nfts/${id}`, data);
    return response.data;
  },

  suspendNFT: async (id: string, reason: string) => {
    const response = await api.post(`/nfts/${id}/suspend`, { reason });
    return response.data;
  },
};

// ==================== MULTISIG ====================
export const multisigService = {
  getAllWallets: async (params?: { page?: number; limit?: number; status?: string }) => {
    const response = await api.get('/multisig', { params });
    return response.data;
  },

  getWallet: async (id: string) => {
    const response = await api.get(`/multisig/${id}`);
    return response.data;
  },

  createWallet: async (data: Partial<MultisigWallet>) => {
    const response = await api.post('/multisig', data);
    return response.data;
  },

  getTransactions: async (walletId: string, params?: { page?: number; limit?: number; status?: string }) => {
    const response = await api.get(`/multisig/${walletId}/transactions`, { params });
    return response.data;
  },

  approveTransaction: async (walletId: string, txId: string) => {
    const response = await api.post(`/multisig/${walletId}/transactions/${txId}/approve`);
    return response.data;
  },

  rejectTransaction: async (walletId: string, txId: string, reason: string) => {
    const response = await api.post(`/multisig/${walletId}/transactions/${txId}/reject`, { reason });
    return response.data;
  },
};

// ==================== BROKER MANAGEMENT ====================
export const brokerService = {
  getAllBrokers: async (params?: { page?: number; limit?: number; status?: string }) => {
    const response = await api.get('/brokers', { params });
    return response.data;
  },

  getBroker: async (id: string) => {
    const response = await api.get(`/brokers/${id}`);
    return response.data;
  },

  createBroker: async (data: Partial<Broker>) => {
    const response = await api.post('/brokers', data);
    return response.data;
  },

  updateBroker: async (id: string, data: Partial<Broker>) => {
    const response = await api.put(`/brokers/${id}`, data);
    return response.data;
  },

  setBrokerCommission: async (id: string, commission: string) => {
    const response = await api.put(`/brokers/${id}/commission`, { commission });
    return response.data;
  },

  getBrokerClients: async (brokerId: string, params?: { page?: number; limit?: number }) => {
    const response = await api.get(`/brokers/${brokerId}/clients`, { params });
    return response.data;
  },
};

// ==================== INSTITUTIONAL MANAGEMENT ====================
export const institutionalService = {
  getAllClients: async (params?: { page?: number; limit?: number; status?: string; type?: string }) => {
    const response = await api.get('/institutional', { params });
    return response.data;
  },

  getClient: async (id: string) => {
    const response = await api.get(`/institutional/${id}`);
    return response.data;
  },

  createClient: async (data: Partial<InstitutionalClient>) => {
    const response = await api.post('/institutional', data);
    return response.data;
  },

  updateClient: async (id: string, data: Partial<InstitutionalClient>) => {
    const response = await api.put(`/institutional/${id}`, data);
    return response.data;
  },

  setClientLimits: async (id: string, limits: InstitutionalClient['limits']) => {
    const response = await api.put(`/institutional/${id}/limits`, limits);
    return response.data;
  },

  assignAccountManager: async (clientId: string, adminId: string) => {
    const response = await api.put(`/institutional/${clientId}/account-manager`, { adminId });
    return response.data;
  },
};

// ==================== AUDIT LOG ====================
export const auditService = {
  getLogs: async (params?: {
    page?: number;
    limit?: number;
    userId?: string;
    adminId?: string;
    action?: string;
    resource?: string;
    dateFrom?: string;
    dateTo?: string;
  }) => {
    const response = await api.get('/audit', { params });
    return response.data;
  },

  getLog: async (id: string) => {
    const response = await api.get(`/audit/${id}`);
    return response.data;
  },

  exportLogs: async (params: any, format: 'csv' | 'xlsx' | 'pdf') => {
    const response = await api.get('/audit/export', {
      params: { ...params, format },
      responseType: 'blob',
    });
    return response.data;
  },
};

// ==================== LISTING REQUESTS ====================
export const listingService = {
  getRequests: async (params?: { page?: number; limit?: number; status?: string; tier?: string }) => {
    const response = await api.get('/listings', { params });
    return response.data;
  },

  approveRequest: async (id: string, notes?: string) => {
    const response = await api.post(`/listings/${id}/approve`, { notes });
    return response.data;
  },

  rejectRequest: async (id: string, reason: string) => {
    const response = await api.post(`/listings/${id}/reject`, { reason });
    return response.data;
  },
};

// ==================== EXPORT ALL SERVICES ====================
export default {
  auth: authService,
  users: userService,
  admins: adminService,
  kyc: kycService,
  whiteLabel: whiteLabelService,
  pairs: pairService,
  tokens: tokenService,
  blockchains: blockchainService,
  transactions: transactionService,
  withdrawals: withdrawalService,
  fees: feeService,
  apiKeys: apiKeyService,
  analytics: analyticsService,
  notifications: notificationService,
  marketMaker: marketMakerService,
  nfts: nftService,
  multisig: multisigService,
  brokers: brokerService,
  institutional: institutionalService,
  audit: auditService,
  listings: listingService,
};
