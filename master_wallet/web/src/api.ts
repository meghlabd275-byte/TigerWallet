// MasterWallet Web - API Service
// Connects to master wallet backend

const API_BASE_URL = process.env.NEXT_PUBLIC_MASTER_WALLET_API || 'https://api.tigerwallet.io/master';

export interface SubWallet {
  id: string;
  name: string;
  address: string;
  balance: string;
  chain: string;
  status: 'Active' | 'Inactive';
  userCount: number;
  createdAt: string;
}

export interface Transaction {
  id: string;
  hash: string;
  from: string;
  to: string;
  amount: string;
  token: string;
  chain: string;
  status: 'Pending' | 'Confirmed' | 'Failed';
  type: string;
  timestamp: string;
}

export interface AutoSignRule {
  id: string;
  name: string;
  maxAmount: string;
  chain: string;
  enabled: boolean;
  conditions: string[];
}

export interface MasterUser {
  id: string;
  email: string;
  name: string;
  role: string;
  permissions: string[];
  walletAddress: string;
  createdAt: string;
}

export interface VolumeStats {
  totalVolume: string;
  dailyVolume: string;
  monthlyVolume: string;
  txCount: number;
}

class MasterWalletAPI {
  private baseURL: string;
  
  constructor(baseURL: string = API_BASE_URL) {
    this.baseURL = baseURL;
  }
  
  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const token = localStorage.getItem('master_wallet_token');
    
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...(token && { 'Authorization': `Bearer ${token}` }),
      ...options.headers,
    };
    
    const response = await fetch(url, { ...options, headers });
    
    if (!response.ok) {
      throw new Error(`API Error: ${response.status}`);
    }
    
    return response.json();
  }
  
  // Sub-wallet endpoints
  async getSubWallets(): Promise<SubWallet[]> {
    return this.request<SubWallet[]>('/api/v1/subwallets');
  }
  
  async createSubWallet(name: string, chain: string): Promise<SubWallet> {
    return this.request<SubWallet>('/api/v1/subwallets', {
      method: 'POST',
      body: JSON.stringify({ name, chain }),
    });
  }
  
  async getSubWallet(id: string): Promise<SubWallet> {
    return this.request<SubWallet>(`/api/v1/subwallets/${id}`);
  }
  
  async updateSubWallet(id: string, data: Partial<SubWallet>): Promise<SubWallet> {
    return this.request<SubWallet>(`/api/v1/subwallets/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }
  
  async deleteSubWallet(id: string): Promise<void> {
    return this.request<void>(`/api/v1/subwallets/${id}`, { method: 'DELETE' });
  }
  
  // Transaction endpoints
  async getTransactions(status?: string): Promise<Transaction[]> {
    const query = status ? `?status=${status}` : '';
    return this.request<Transaction[]>(`/api/v1/transactions${query}`);
  }
  
  async getTransaction(id: string): Promise<Transaction> {
    return this.request<Transaction>(`/api/v1/transactions/${id}`);
  }
  
  async approveTransaction(id: string): Promise<Transaction> {
    return this.request<Transaction>(`/api/v1/transactions/${id}/approve`, { method: 'POST' });
  }
  
  async rejectTransaction(id: string, reason: string): Promise<Transaction> {
    return this.request<Transaction>(`/api/v1/transactions/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }
  
  async signTransaction(id: string): Promise<Transaction> {
    return this.request<Transaction>(`/api/v1/transactions/${id}/sign`, { method: 'POST' });
  }
  
  // Auto-sign endpoints
  async getAutoSignRules(): Promise<AutoSignRule[]> {
    return this.request<AutoSignRule[]>('/api/v1/auto-sign/rules');
  }
  
  async createAutoSignRule(rule: Omit<AutoSignRule, 'id'>): Promise<AutoSignRule> {
    return this.request<AutoSignRule>('/api/v1/auto-sign/rules', {
      method: 'POST',
      body: JSON.stringify(rule),
    });
  }
  
  async updateAutoSignRule(id: string, data: Partial<AutoSignRule>): Promise<AutoSignRule> {
    return this.request<AutoSignRule>(`/api/v1/auto-sign/rules/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }
  
  async deleteAutoSignRule(id: string): Promise<void> {
    return this.request<void>(`/api/v1/auto-sign/rules/${id}`, { method: 'DELETE' });
  }
  
  async toggleAutoSignRule(id: string): Promise<AutoSignRule> {
    return this.request<AutoSignRule>(`/api/v1/auto-sign/rules/${id}/toggle`, { method: 'POST' });
  }
  
  // User management endpoints
  async getUsers(): Promise<MasterUser[]> {
    return this.request<MasterUser[]>('/api/v1/users');
  }
  
  async getUser(id: string): Promise<MasterUser> {
    return this.request<MasterUser>(`/api/v1/users/${id}`);
  }
  
  async createUser(user: Omit<MasterUser, 'id' | 'createdAt'>): Promise<MasterUser> {
    return this.request<MasterUser>('/api/v1/users', {
      method: 'POST',
      body: JSON.stringify(user),
    });
  }
  
  async updateUser(id: string, data: Partial<MasterUser>): Promise<MasterUser> {
    return this.request<MasterUser>(`/api/v1/users/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }
  
  async deleteUser(id: string): Promise<void> {
    return this.request<void>(`/api/v1/users/${id}`, { method: 'DELETE' });
  }
  
  async updateUserPermissions(id: string, permissions: string[]): Promise<MasterUser> {
    return this.request<MasterUser>(`/api/v1/users/${id}/permissions`, {
      method: 'PUT',
      body: JSON.stringify({ permissions }),
    });
  }
  
  // Volume/Analytics endpoints
  async getVolumeStats(): Promise<VolumeStats> {
    return this.request<VolumeStats>('/api/v1/analytics/volume');
  }
  
  async getAnalytics(period: string = '7d') {
    return this.request(`/api/v1/analytics?period=${period}`);
  }
  
  // Whitelist endpoints
  async getWhitelist(): Promise<string[]> {
    return this.request<string[]>('/api/v1/whitelist');
  }
  
  async addToWhitelist(address: string): Promise<void> {
    return this.request<void>('/api/v1/whitelist', {
      method: 'POST',
      body: JSON.stringify({ address }),
    });
  }
  
  async removeFromWhitelist(address: string): Promise<void> {
    return this.request<void>(`/api/v1/whitelist/${address}`, { method: 'DELETE' });
  }
  
  // Master wallet endpoints
  async getMasterWalletInfo() {
    return this.request('/api/v1/master-wallet');
  }
  
  async getMasterWalletAddress(): Promise<string> {
    const data = await this.request<{ address: string }>('/api/v1/master-wallet/address');
    return data.address;
  }
  
  // Chain endpoints
  async getSupportedChains() {
    return this.request('/api/v1/chains');
  }
  
  // Health check
  async healthCheck(): Promise<boolean> {
    try {
      await this.request('/api/v1/health');
      return true;
    } catch {
      return false;
    }
  }
}

export const masterWalletAPI = new MasterWalletAPI();
export default masterWalletAPI;
