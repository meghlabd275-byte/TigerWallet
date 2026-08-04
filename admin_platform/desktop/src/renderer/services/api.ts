import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'https://admin-api.tigerwallet.com';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('admin_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('admin_token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export const authService = {
  login: async (email: string, password: string) => {
    const response = await api.post('/api/v1/auth/login', { email, password });
    localStorage.setItem('admin_token', response.data.token);
    return response.data;
  },
  logout: async () => {
    await api.post('/api/v1/auth/logout');
    localStorage.removeItem('admin_token');
  },
  getCurrentAdmin: async () => {
    const response = await api.get('/api/v1/auth/me');
    return response.data;
  },
};

export const userService = {
  getUsers: async (params?: { page?: number; limit?: number; status?: string; search?: string }) => {
    const response = await api.get('/api/v1/users', { params });
    return response.data;
  },
  getUser: async (id: string) => {
    const response = await api.get(`/api/v1/users/${id}`);
    return response.data;
  },
  updateUser: async (id: string, data: any) => {
    const response = await api.put(`/api/v1/users/${id}`, data);
    return response.data;
  },
  suspendUser: async (id: string, reason: string) => {
    const response = await api.post(`/api/v1/users/${id}/suspend`, { reason });
    return response.data;
  },
  banUser: async (id: string, reason: string) => {
    const response = await api.post(`/api/v1/users/${id}/ban`, { reason });
    return response.data;
  },
};

export const kycService = {
  getSubmissions: async (params?: { page?: number; limit?: number; status?: string }) => {
    const response = await api.get('/api/v1/kyc', { params });
    return response.data;
  },
  approveKYC: async (id: string, notes?: string) => {
    const response = await api.post(`/api/v1/kyc/${id}/approve`, { notes });
    return response.data;
  },
  rejectKYC: async (id: string, reason: string) => {
    const response = await api.post(`/api/v1/kyc/${id}/reject`, { reason });
    return response.data;
  },
};

export const tokenService = {
  getTokens: async (params?: { page?: number; limit?: number; status?: string }) => {
    const response = await api.get('/api/v1/tokens', { params });
    return response.data;
  },
  createToken: async (data: any) => {
    const response = await api.post('/api/v1/tokens', data);
    return response.data;
  },
  verifyToken: async (id: string) => {
    const response = await api.post(`/api/v1/tokens/${id}/verify`);
    return response.data;
  },
  deleteToken: async (id: string) => {
    const response = await api.delete(`/api/v1/tokens/${id}`);
    return response.data;
  },
};

export const pairService = {
  getPairs: async (params?: { page?: number; limit?: number; status?: string }) => {
    const response = await api.get('/api/v1/pairs', { params });
    return response.data;
  },
  createPair: async (data: any) => {
    const response = await api.post('/api/v1/pairs', data);
    return response.data;
  },
  updatePair: async (id: string, data: any) => {
    const response = await api.put(`/api/v1/pairs/${id}`, data);
    return response.data;
  },
};

export const transactionService = {
  getTransactions: async (params?: { page?: number; limit?: number; status?: string }) => {
    const response = await api.get('/api/v1/transactions', { params });
    return response.data;
  },
};

export const withdrawalService = {
  getWithdrawals: async (params?: { page?: number; limit?: number; status?: string }) => {
    const response = await api.get('/api/v1/withdrawals', { params });
    return response.data;
  },
  approveWithdrawal: async (id: string) => {
    const response = await api.post(`/api/v1/withdrawals/${id}/approve`);
    return response.data;
  },
  rejectWithdrawal: async (id: string, reason: string) => {
    const response = await api.post(`/api/v1/withdrawals/${id}/reject`, { reason });
    return response.data;
  },
};

export const chainService = {
  getChains: async () => {
    const response = await api.get('/api/v1/chains');
    return response.data;
  },
  createChain: async (data: any) => {
    const response = await api.post('/api/v1/chains', data);
    return response.data;
  },
};

export const feeService = {
  getFees: async () => {
    const response = await api.get('/api/v1/fees');
    return response.data;
  },
  createFee: async (data: any) => {
    const response = await api.post('/api/v1/fees', data);
    return response.data;
  },
};

export const whiteLabelService = {
  getWhiteLabels: async (params?: { page?: number; limit?: number; status?: string }) => {
    const response = await api.get('/api/v1/white-labels', { params });
    return response.data;
  },
  createWhiteLabel: async (data: any) => {
    const response = await api.post('/api/v1/white-labels', data);
    return response.data;
  },
  approveWhiteLabel: async (id: string) => {
    const response = await api.post(`/api/v1/white-labels/${id}/approve`);
    return response.data;
  },
  suspendWhiteLabel: async (id: string, reason: string) => {
    const response = await api.post(`/api/v1/white-labels/${id}/suspend`, { reason });
    return response.data;
  },
};

export const dashboardService = {
  getStats: async () => {
    const response = await api.get('/api/v1/dashboard');
    return response.data;
  },
  getAnalytics: async (period: string = '24h') => {
    const response = await api.get('/api/v1/dashboard/analytics', { params: { period } });
    return response.data;
  },
};

export const ticketService = {
  getTickets: async (params?: any) => {
    const response = await api.get('/api/v1/tickets', { params });
    return response.data;
  },
  createTicket: async (data: any) => {
    const response = await api.post('/api/v1/tickets', data);
    return response.data;
  },
  updateTicket: async (id: number, data: any) => {
    const response = await api.put(`/api/v1/tickets/${id}`, data);
    return response.data;
  },
  addMessage: async (ticketId: number, message: string) => {
    const response = await api.post(`/api/v1/tickets/${ticketId}/messages`, { message });
    return response.data;
  },
  getTicketMessages: async (ticketId: number) => {
    const response = await api.get(`/api/v1/tickets/${ticketId}/messages`);
    return response.data;
  },
};

export const knowledgeBaseService = {
  getArticles: async () => {
    const response = await api.get('/api/v1/knowledge-base');
    return response.data;
  },
  createArticle: async (data: any) => {
    const response = await api.post('/api/v1/knowledge-base', data);
    return response.data;
  },
  updateArticle: async (id: string, data: any) => {
    const response = await api.put(`/api/v1/knowledge-base/${id}`, data);
    return response.data;
  },
};

export const approvalService = {
  getWorkflows: async () => {
    const response = await api.get('/api/v1/workflows');
    return response.data;
  },
  createWorkflow: async (data: any) => {
    const response = await api.post('/api/v1/workflows', data);
    return response.data;
  },
  getRequests: async (params?: any) => {
    const response = await api.get('/api/v1/approval-requests', { params });
    return response.data;
  },
  approveRequest: async (id: string) => {
    const response = await api.post(`/api/v1/approval-requests/${id}/approve`);
    return response.data;
  },
  rejectRequest: async (id: string, reason: string) => {
    const response = await api.post(`/api/v1/approval-requests/${id}/reject`, { reason });
    return response.data;
  },
};

export const analyticsService = {
  getComplianceDashboard: async () => {
    const response = await api.get('/api/v1/dashboard/compliance');
    return response.data;
  },
  getFinanceDashboard: async () => {
    const response = await api.get('/api/v1/dashboard/finance');
    return response.data;
  },
  getSecurityDashboard: async () => {
    const response = await api.get('/api/v1/dashboard/security');
    return response.data;
  },
};

export const notificationService = {
  getNotifications: async (params?: any) => {
    const response = await api.get('/api/v1/notifications', { params });
    return response.data;
  },
  markAsRead: async (id: string) => {
    const response = await api.put(`/api/v1/notifications/${id}/read`);
    return response.data;
  },
  send: async (data: any) => {
    const response = await api.post('/api/v1/notifications', data);
    return response.data;
  },
  broadcast: async (data: any) => {
    const response = await api.post('/api/v1/notifications/broadcast', data);
    return response.data;
  },
};

export const apiKeyService = {
  getKeys: async () => {
    const response = await api.get('/api/v1/api-keys');
    return response.data;
  },
  createKey: async (data: any) => {
    const response = await api.post('/api/v1/api-keys', data);
    return response.data;
  },
  revokeKey: async (id: string) => {
    const response = await api.post(`/api/v1/api-keys/${id}/revoke`);
    return response.data;
  },
};

export const webhookService = {
  getWebhooks: async () => {
    const response = await api.get('/api/v1/webhooks');
    return response.data;
  },
  createWebhook: async (data: any) => {
    const response = await api.post('/api/v1/webhooks', data);
    return response.data;
  },
  updateWebhook: async (id: string, data: any) => {
    const response = await api.put(`/api/v1/webhooks/${id}`, data);
    return response.data;
  },
  deleteWebhook: async (id: string) => {
    const response = await api.delete(`/api/v1/webhooks/${id}`);
    return response.data;
  },
};

export const auditService = {
  getLogs: async (params?: any) => {
    const response = await api.get('/api/v1/audit-logs', { params });
    return response.data;
  },
};

export const adminService = {
  getAdmins: async () => {
    const response = await api.get('/api/v1/admins');
    return response.data;
  },
  createAdmin: async (data: any) => {
    const response = await api.post('/api/v1/admins', data);
    return response.data;
  },
  updateAdmin: async (id: string, data: any) => {
    const response = await api.put(`/api/v1/admins/${id}`, data);
    return response.data;
  },
  deleteAdmin: async (id: string) => {
    const response = await api.delete(`/api/v1/admins/${id}`);
    return response.data;
  },
};

export default api;
