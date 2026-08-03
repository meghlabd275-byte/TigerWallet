const API_BASE_URL = 'https://admin-api.tigerwallet.com';

class ApiService {
  private token: string | null = localStorage.getItem('admin_token');

  private getHeaders(): HeadersInit {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    };
    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }
    return headers;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers: {
        ...this.getHeaders(),
        ...options.headers,
      },
    });

    if (!response.ok) {
      throw new Error(`API Error: ${response.status}`);
    }

    return response.json();
  }

  // Auth
  async login(email: string, password: string) {
    const data = await this.request<any>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    if (data.data?.token) {
      this.token = data.data.token;
      localStorage.setItem('admin_token', data.data.token);
    }
    return data;
  }

  async logout() {
    await this.request('/api/v1/auth/logout', { method: 'POST' });
    this.token = null;
    localStorage.removeItem('admin_token');
  }

  // Users
  async getUsers(params: any = {}) {
    const query = new URLSearchParams(params).toString();
    return this.request(`/api/v1/users?${query}`);
  }

  async getUser(id: string) {
    return this.request(`/api/v1/users/${id}`);
  }

  async updateUser(id: string, data: any) {
    return this.request(`/api/v1/users/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async suspendUser(id: string, reason: string) {
    return this.request(`/api/v1/users/${id}/suspend`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  async banUser(id: string, reason: string) {
    return this.request(`/api/v1/users/${id}/ban`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  // KYC
  async getKYC(params: any = {}) {
    const query = new URLSearchParams(params).toString();
    return this.request(`/api/v1/kyc?${query}`);
  }

  async approveKYC(id: string, notes?: string) {
    return this.request(`/api/v1/kyc/${id}/approve`, {
      method: 'POST',
      body: JSON.stringify({ notes }),
    });
  }

  async rejectKYC(id: string, reason: string) {
    return this.request(`/api/v1/kyc/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  // Tokens
  async getTokens(params: any = {}) {
    const query = new URLSearchParams(params).toString();
    return this.request(`/api/v1/tokens?${query}`);
  }

  async createToken(data: any) {
    return this.request('/api/v1/tokens', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async verifyToken(id: string) {
    return this.request(`/api/v1/tokens/${id}/verify`, {
      method: 'POST',
    });
  }

  async deleteToken(id: string) {
    return this.request(`/api/v1/tokens/${id}`, {
      method: 'DELETE',
    });
  }

  // Pairs
  async getPairs(params: any = {}) {
    const query = new URLSearchParams(params).toString();
    return this.request(`/api/v1/pairs?${query}`);
  }

  async updatePair(id: string, data: any) {
    return this.request(`/api/v1/pairs/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  // Transactions
  async getTransactions(params: any = {}) {
    const query = new URLSearchParams(params).toString();
    return this.request(`/api/v1/transactions?${query}`);
  }

  // Withdrawals
  async getWithdrawals(params: any = {}) {
    const query = new URLSearchParams(params).toString();
    return this.request(`/api/v1/withdrawals?${query}`);
  }

  async approveWithdrawal(id: string) {
    return this.request(`/api/v1/withdrawals/${id}/approve`, {
      method: 'POST',
    });
  }

  async rejectWithdrawal(id: string, reason: string) {
    return this.request(`/api/v1/withdrawals/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  // Chains
  async getChains() {
    return this.request('/api/v1/chains');
  }

  async updateChain(id: string, data: any) {
    return this.request(`/api/v1/chains/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  // Fees
  async getFees() {
    return this.request('/api/v1/fees');
  }

  async updateFee(id: string, data: any) {
    return this.request(`/api/v1/fees/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  // White Labels
  async getWhiteLabels(params: any = {}) {
    const query = new URLSearchParams(params).toString();
    return this.request(`/api/v1/white-labels?${query}`);
  }

  async approveWhiteLabel(id: string) {
    return this.request(`/api/v1/white-labels/${id}/approve`, {
      method: 'POST',
    });
  }

  async suspendWhiteLabel(id: string, reason: string) {
    return this.request(`/api/v1/white-labels/${id}/suspend`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  // Dashboard
  async getDashboard() {
    return this.request('/api/v1/dashboard');
  }
}

export const apiService = new ApiService();
export const authService = {
  login: (email: string, password: string) => apiService.login(email, password),
  logout: () => apiService.logout(),
};
export const userService = {
  getUsers: (params: any) => apiService.getUsers(params),
  getUser: (id: string) => apiService.getUser(id),
  updateUser: (id: string, data: any) => apiService.updateUser(id, data),
  suspendUser: (id: string, reason: string) => apiService.suspendUser(id, reason),
  banUser: (id: string, reason: string) => apiService.banUser(id, reason),
};
export const kycService = {
  getSubmissions: (params: any) => apiService.getKYC(params),
  approveKYC: (id: string, notes?: string) => apiService.approveKYC(id, notes),
  rejectKYC: (id: string, reason: string) => apiService.rejectKYC(id, reason),
};
export const tokenService = {
  getTokens: (params: any) => apiService.getTokens(params),
  createToken: (data: any) => apiService.createToken(data),
  verifyToken: (id: string) => apiService.verifyToken(id),
  deleteToken: (id: string) => apiService.deleteToken(id),
};
export const pairService = {
  getPairs: (params: any) => apiService.getPairs(params),
  updatePair: (id: string, data: any) => apiService.updatePair(id, data),
};
export const transactionService = {
  getTransactions: (params: any) => apiService.getTransactions(params),
};
export const withdrawalService = {
  getWithdrawals: (params: any) => apiService.getWithdrawals(params),
  approveWithdrawal: (id: string) => apiService.approveWithdrawal(id),
  rejectWithdrawal: (id: string, reason: string) => apiService.rejectWithdrawal(id, reason),
};
export const chainService = {
  getChains: () => apiService.getChains(),
  updateChain: (id: string, data: any) => apiService.updateChain(id, data),
};
export const feeService = {
  getFees: () => apiService.getFees(),
  updateFee: (id: string, data: any) => apiService.updateFee(id, data),
};
export const whiteLabelService = {
  getWhiteLabels: (params: any) => apiService.getWhiteLabels(params),
  approveWhiteLabel: (id: string) => apiService.approveWhiteLabel(id),
  suspendWhiteLabel: (id: string, reason: string) => apiService.suspendWhiteLabel(id, reason),
};
export const dashboardService = {
  getDashboard: () => apiService.getDashboard(),
};
