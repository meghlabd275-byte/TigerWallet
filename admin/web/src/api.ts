// Admin Web - API Service
// Connects to admin backend

const API_BASE_URL = process.env.NEXT_PUBLIC_ADMIN_API || 'https://api.tigerwallet.io/admin';

export interface User {
  id: string;
  email: string;
  name: string;
  kycStatus: 'Pending' | 'Verified' | 'Rejected' | 'Not Submitted';
  createdAt: string;
  lastLogin: string;
  totalVolume: string;
  walletCount: number;
}

export interface AdminTransaction {
  id: string;
  hash: string;
  from: string;
  to: string;
  amount: string;
  token: string;
  chain: string;
  status: 'Pending' | 'Completed' | 'Flagged' | 'Failed';
  type: string;
  timestamp: string;
  flagReason?: string;
}

export interface KycRecord {
  id: string;
  userId: string;
  userEmail: string;
  status: 'Pending' | 'Approved' | 'Rejected';
  submittedAt: string;
  reviewedAt?: string;
  documentType: string;
  documentUrl: string;
}

export interface Token {
  address: string;
  name: string;
  symbol: string;
  decimals: number;
  isListed: boolean;
  marketCap: string;
  volume24h: string;
  price: string;
  chain: string;
}

export interface SystemService {
  name: string;
  status: 'Running' | 'Stopped' | 'Error';
  uptime: string;
  latency: string;
  lastCheck: string;
}

export interface FeeConfig {
  tradingFee: string;
  withdrawalFee: string;
  depositFee: string;
  networkFee: string;
}

export interface Analytics {
  totalUsers: number;
  totalVolume: string;
  dailyTransactions: number;
  activeUsers: number;
  revenue: string;
  growth: string;
}

class AdminAPI {
  private baseURL: string;
  
  constructor(baseURL: string = API_BASE_URL) {
    this.baseURL = baseURL;
  }
  
  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const token = localStorage.getItem('admin_token');
    
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
  
  // User endpoints
  async getUsers(status?: string): Promise<User[]> {
    const query = status ? `?status=${status}` : '';
    return this.request<User[]>(`/api/v1/users${query}`);
  }
  
  async getUser(id: string): Promise<User> {
    return this.request<User>(`/api/v1/users/${id}`);
  }
  
  async updateUser(id: string, data: Partial<User>): Promise<User> {
    return this.request<User>(`/api/v1/users/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }
  
  async deleteUser(id: string): Promise<void> {
    return this.request<void>(`/api/v1/users/${id}`, { method: 'DELETE' });
  }
  
  async suspendUser(id: string, reason: string): Promise<void> {
    return this.request<void>(`/api/v1/users/${id}/suspend`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }
  
  async unsuspendUser(id: string): Promise<void> {
    return this.request<void>(`/api/v1/users/${id}/unsuspend`, { method: 'POST' });
  }
  
  // Transaction endpoints
  async getTransactions(status?: string): Promise<AdminTransaction[]> {
    const query = status ? `?status=${status}` : '';
    return this.request<AdminTransaction[]>(`/api/v1/transactions${query}`);
  }
  
  async getTransaction(id: string): Promise<AdminTransaction> {
    return this.request<AdminTransaction>(`/api/v1/transactions/${id}`);
  }
  
  async flagTransaction(id: string, reason: string): Promise<AdminTransaction> {
    return this.request<AdminTransaction>(`/api/v1/transactions/${id}/flag`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }
  
  async unflagTransaction(id: string): Promise<AdminTransaction> {
    return this.request<AdminTransaction>(`/api/v1/transactions/${id}/unflag`, { method: 'POST' });
  }
  
  async cancelTransaction(id: string): Promise<AdminTransaction> {
    return this.request<AdminTransaction>(`/api/v1/transactions/${id}/cancel`, { method: 'POST' });
  }
  
  // KYC endpoints
  async getKycRecords(status?: string): Promise<KycRecord[]> {
    const query = status ? `?status=${status}` : '';
    return this.request<KycRecord[]>(`/api/v1/kyc${query}`);
  }
  
  async getKycRecord(id: string): Promise<KycRecord> {
    return this.request<KycRecord>(`/api/v1/kyc/${id}`);
  }
  
  async approveKyc(id: string): Promise<KycRecord> {
    return this.request<KycRecord>(`/api/v1/kyc/${id}/approve`, { method: 'POST' });
  }
  
  async rejectKyc(id: string, reason: string): Promise<KycRecord> {
    return this.request<KycRecord>(`/api/v1/kyc/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }
  
  // Token management endpoints
  async getTokens(): Promise<Token[]> {
    return this.request<Token[]>('/api/v1/tokens');
  }
  
  async getToken(address: string): Promise<Token> {
    return this.request<Token>(`/api/v1/tokens/${address}`);
  }
  
  async listToken(address: string): Promise<Token> {
    return this.request<Token>(`/api/v1/tokens/${address}/list`, { method: 'POST' });
  }
  
  async delistToken(address: string): Promise<Token> {
    return this.request<Token>(`/api/v1/tokens/${address}/delist`, { method: 'POST' });
  }
  
  async addToken(token: Omit<Token, 'isListed'>): Promise<Token> {
    return this.request<Token>('/api/v1/tokens', {
      method: 'POST',
      body: JSON.stringify(token),
    });
  }
  
  // Fee configuration endpoints
  async getFeeConfig(): Promise<FeeConfig> {
    return this.request<FeeConfig>('/api/v1/fees');
  }
  
  async updateFeeConfig(config: FeeConfig): Promise<FeeConfig> {
    return this.request<FeeConfig>('/api/v1/fees', {
      method: 'PUT',
      body: JSON.stringify(config),
    });
  }
  
  // System endpoints
  async getSystemStatus(): Promise<SystemService[]> {
    return this.request<SystemService[]>('/api/v1/system/status');
  }
  
  async getSystemService(name: string): Promise<SystemService> {
    return this.request<SystemService>(`/api/v1/system/services/${name}`);
  }
  
  async restartService(name: string): Promise<void> {
    return this.request<void>(`/api/v1/system/services/${name}/restart`, { method: 'POST' });
  }
  
  // Analytics endpoints
  async getAnalytics(): Promise<Analytics> {
    return this.request<Analytics>('/api/v1/analytics');
  }
  
  async getUserAnalytics(period: string = '30d') {
    return this.request(`/api/v1/analytics/users?period=${period}`);
  }
  
  async getVolumeAnalytics(period: string = '30d') {
    return this.request(`/api/v1/analytics/volume?period=${period}`);
  }
  
  // Config endpoints
  async getConfig() {
    return this.request('/api/v1/config');
  }
  
  async updateConfig(config: Record<string, any>) {
    return this.request('/api/v1/config', {
      method: 'PUT',
      body: JSON.stringify(config),
    });
  }
  
  // Admin user management
  async getAdminUsers() {
    return this.request('/api/v1/admins');
  }
  
  async createAdminUser(user: any) {
    return this.request('/api/v1/admins', {
      method: 'POST',
      body: JSON.stringify(user),
    });
  }
  
  async updateAdminUser(id: string, data: any) {
    return this.request(`/api/v1/admins/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }
  
  async deleteAdminUser(id: string) {
    return this.request<void>(`/api/v1/admins/${id}`, { method: 'DELETE' });
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

export const adminAPI = new AdminAPI();
export default adminAPI;
