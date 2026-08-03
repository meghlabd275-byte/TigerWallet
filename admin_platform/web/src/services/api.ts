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

export default api;
