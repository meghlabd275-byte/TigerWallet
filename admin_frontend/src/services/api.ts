/**
 * TigerWallet Admin API Service
 * Complete API integration with authentication
 */

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8085';

interface ApiResponse<T> {
  data?: T;
  error?: string;
  message?: string;
}

class ApiService {
  private token: string | null = null;

  constructor() {
    this.token = localStorage.getItem('tigerwallet-token');
  }

  setToken(token: string) {
    this.token = token;
    localStorage.setItem('tigerwallet-token', token);
  }

  clearToken() {
    this.token = null;
    localStorage.removeItem('tigerwallet-token');
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    const url = `${API_BASE_URL}${endpoint}`;

    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    };

    if (this.token) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${this.token}`;
    }

    try {
      const response = await fetch(url, {
        ...options,
        headers,
      });

      const data = await response.json();

      if (!response.ok) {
        return { error: data.error || 'Request failed' };
      }

      return { data };
    } catch (error) {
      return { error: error instanceof Error ? error.message : 'Network error' };
    }
  }

  // ==================== AUTH ====================

  async login(username: string, password: string) {
    const response = await this.request<{ token: string; admin: any }>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });

    if (response.data?.token) {
      this.setToken(response.data.token);
    }

    return response;
  }

  logout() {
    this.clearToken();
  }

  // ==================== ADMIN ====================

  async getAdmins(params?: { role?: string; search?: string }) {
    const queryParams = new URLSearchParams();
    if (params?.role) queryParams.set('role', params.role);
    if (params?.search) queryParams.set('search', params.search);

    const query = queryParams.toString() ? `?${queryParams.toString()}` : '';
    return this.request<{ admins: any[] }>(`/api/v1/admins${query}`);
  }

  async createAdmin(data: {
    username: string;
    email: string;
    password: string;
    role: string;
  }) {
    return this.request<any>('/api/v1/admins', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateAdmin(id: string, data: Partial<{
    username: string;
    email: string;
    role: string;
    permissions: string[];
  }>) {
    return this.request<any>(`/api/v1/admins/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async suspendAdmin(id: string) {
    return this.request<any>(`/api/v1/admins/${id}/suspend`, {
      method: 'POST',
    });
  }

  // ==================== WHITE LABELS ====================

  async getWhiteLabels(params?: { status?: string; search?: string }) {
    const queryParams = new URLSearchParams();
    if (params?.status) queryParams.set('status', params.status);
    if (params?.search) queryParams.set('search', params.search);

    const query = queryParams.toString() ? `?${queryParams.toString()}` : '';
    return this.request<{ white_labels: any[] }>(`/api/v1/white-labels${query}`);
  }

  async getWhiteLabel(id: string) {
    return this.request<{ white_label: any }>(`/api/v1/white-labels/${id}`);
  }

  async createWhiteLabel(data: {
    name: string;
    domain: string;
    plan_tier?: string;
    max_users?: number;
    monthly_fee?: number;
  }) {
    return this.request<any>('/api/v1/white-labels', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateWhiteLabel(id: string, data: Partial<{
    name: string;
    domain: string;
    plan_tier: string;
    max_users: number;
    monthly_fee: number;
    fee_percent: number;
    profit_share_percent: number;
    branding_config: any;
    features: string[];
  }>) {
    return this.request<any>(`/api/v1/white-labels/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async approveWhiteLabel(id: string) {
    return this.request<any>(`/api/v1/white-labels/${id}/approve`, {
      method: 'POST',
    });
  }

  async suspendWhiteLabel(id: string) {
    return this.request<any>(`/api/v1/white-labels/${id}/suspend`, {
      method: 'POST',
    });
  }

  async revokeWhiteLabel(id: string) {
    return this.request<any>(`/api/v1/white-labels/${id}/revoke`, {
      method: 'POST',
    });
  }

  async destroyWhiteLabel(id: string) {
    return this.request<any>(`/api/v1/white-labels/${id}`, {
      method: 'DELETE',
    });
  }

  async getWhiteLabelStats(id: string) {
    return this.request<{ stats: any }>(`/api/v1/white-labels/${id}/stats`);
  }

  async getWhiteLabelAPIKeys(id: string) {
    return this.request<{ api_keys: any[] }>(`/api/v1/white-labels/${id}/api-keys`);
  }

  async createWhiteLabelAPIKey(id: string, data: {
    name: string;
    permissions?: string[];
    rate_limit_minute?: number;
    rate_limit_day?: number;
    expires_at?: string;
  }) {
    return this.request<any>(`/api/v1/white-labels/${id}/api-keys`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async revokeAPIKey(keyId: string) {
    return this.request<any>(`/api/v1/api-keys/${keyId}`, {
      method: 'DELETE',
    });
  }

  // ==================== PRODUCTS ====================

  async getProducts(whiteLabelId?: string) {
    const query = whiteLabelId ? `?white_label_id=${whiteLabelId}` : '';
    return this.request<{ products: any[] }>(`/api/v1/products${query}`);
  }

  async createProduct(data: {
    white_label_id?: string;
    name: string;
    type: string;
    description?: string;
    fee_percent?: number;
    min_deposit?: number;
    max_deposit?: number;
    features?: string[];
    supported_chains?: string[];
    is_global?: boolean;
  }) {
    return this.request<any>('/api/v1/products', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateProduct(id: string, data: Partial<{
    name: string;
    description: string;
    status: string;
    fee_percent: number;
    min_deposit: number;
    max_deposit: number;
    features: string[];
    supported_chains: string[];
  }>) {
    return this.request<any>(`/api/v1/products/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  // ==================== TRADING PAIRS ====================

  async getTradingPairs(whiteLabelId?: string) {
    const query = whiteLabelId ? `?white_label_id=${whiteLabelId}` : '';
    return this.request<{ pairs: any[] }>(`/api/v1/trading-pairs${query}`);
  }

  async createTradingPair(data: {
    white_label_id?: string;
    base_token: string;
    quote_token: string;
    pair_symbol: string;
    min_trade_amount?: number;
    max_trade_amount?: number;
    maker_fee?: number;
    taker_fee?: number;
    chain_id?: string;
    price_precision?: number;
    quantity_precision?: number;
  }) {
    return this.request<any>('/api/v1/trading-pairs', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // ==================== USERS ====================

  async getUsers(params?: {
    white_label_id?: string;
    status?: string;
    search?: string;
    limit?: number;
    offset?: number;
  }) {
    const queryParams = new URLSearchParams();
    if (params?.white_label_id) queryParams.set('white_label_id', params.white_label_id);
    if (params?.status) queryParams.set('status', params.status);
    if (params?.search) queryParams.set('search', params.search);
    if (params?.limit) queryParams.set('limit', params.limit.toString());
    if (params?.offset) queryParams.set('offset', params.offset.toString());

    const query = queryParams.toString() ? `?${queryParams.toString()}` : '';
    return this.request<{ users: any[]; total: number; limit: number; offset: number }>(`/api/v1/users${query}`);
  }

  // ==================== BLOCKCHAINS ====================

  async getBlockchains() {
    return this.request<{ blockchains: any[] }>('/api/v1/blockchains');
  }

  // ==================== FEE STRUCTURES ====================

  async getFeeStructures(whiteLabelId?: string) {
    const query = whiteLabelId ? `?white_label_id=${whiteLabelId}` : '';
    return this.request<{ fees: any[] }>(`/api/v1/fee-structures${query}`);
  }

  // ==================== AUDIT LOGS ====================

  async getAuditLogs(params?: { admin_id?: string; action?: string; limit?: number }) {
    const queryParams = new URLSearchParams();
    if (params?.admin_id) queryParams.set('admin_id', params.admin_id);
    if (params?.action) queryParams.set('action', params.action);
    if (params?.limit) queryParams.set('limit', params.limit.toString());

    const query = queryParams.toString() ? `?${queryParams.toString()}` : '';
    return this.request<{ audit_logs: any[] }>(`/api/v1/audit-logs${query}`);
  }

  // ==================== DASHBOARD ====================

  async getDashboardStats() {
    return this.request<{ stats: {
      active_white_labels: number;
      pending_white_labels: number;
      total_users: number;
      active_users: number;
      transactions_24h: number;
      total_revenue: number;
      total_admins: number;
      total_products: number;
    } }>('/api/v1/dashboard/stats');
  }

  // ==================== HEALTH ====================

  async healthCheck() {
    return this.request<{ status: string; database: string; redis: string }>('/health');
  }
}

export const api = new ApiService();
export default api;
