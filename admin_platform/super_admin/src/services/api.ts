/**
 * TigerWallet Super Admin Panel - API Service
 * Production-ready API client that connects to Go backend
 * No stubs - Real API calls only
 */

const API_BASE_URL = 'http://localhost:8080/api/v1';

// ==================== TYPES ====================

export interface Admin {
  id: string;
  username: string;
  email: string;
  role: 'super_admin' | 'admin' | 'manager' | 'support';
  security_level: number;
  permissions: string[];
  two_factor_enabled: boolean;
  created_at: string;
  last_login: string;
  status: 'active' | 'suspended' | 'blocked';
  failed_attempts: number;
  locked_until: string;
}

export interface WhiteLabel {
  id: string;
  name: string;
  domain: string;
  api_key: string;
  fee_percent: number;
  status: 'pending' | 'active' | 'suspended' | 'revoked';
  approved_by?: string;
  approved_at?: string;
  created_at: string;
  features: string[];
  custom_branding: boolean;
}

export interface AuditLog {
  id: string;
  admin_id: string;
  admin_username: string;
  action: string;
  details: string;
  ip_address: string;
  user_agent: string;
  timestamp: string;
}

export interface ProfitTransaction {
  id: string;
  white_label_id: string;
  super_admin_wallet: string;
  amount: number;
  percentage: number;
  gross_revenue: number;
  net_revenue: number;
  token: string;
  tx_hash?: string;
  status: 'pending' | 'completed' | 'failed';
  created_at: string;
}

export interface ProfitShareConfig {
  id: string;
  white_label_id: string;
  super_admin_wallet: string;
  profit_percentage: number;
  min_percentage: number;
  max_percentage: number;
  is_active: boolean;
  auto_transfer: boolean;
  transfer_frequency: 'daily' | 'weekly' | 'monthly';
  last_transfer: string;
  total_transferred: number;
}

export interface FeatureFlag {
  id: string;
  name: string;
  description: string;
  global_enabled: boolean;
  enabled: boolean;
  master_admin_id?: string;
  white_label_id?: string;
  updated_by?: string;
  updated_at: string;
}

export interface LoginRequest {
  username: string;
  password: string;
  two_factor_code?: string;
}

export interface AuthResponse {
  success: boolean;
  error?: string;
  session_token?: string;
  admin_id?: string;
  username?: string;
  role?: number;
}

export interface Session {
  id: string;
  admin_id: string;
  token: string;
  expires_at: string;
  ip_address: string;
  user_agent: string;
  created_at: string;
  is_valid: boolean;
}

// ==================== API CLIENT ====================

class SuperAdminAPIClient {
  private baseUrl: string;
  private token: string | null = null;

  constructor(baseUrl: string = API_BASE_URL) {
    this.baseUrl = baseUrl;
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('super_admin_token');
    }
  }

  setToken(token: string) {
    this.token = token;
    if (typeof window !== 'undefined') {
      localStorage.setItem('super_admin_token', token);
    }
  }

  clearToken() {
    this.token = null;
    if (typeof window !== 'undefined') {
      localStorage.removeItem('super_admin_token');
    }
  }

  getToken(): string | null {
    return this.token;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    
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

      if (!response.ok) {
        const error = await response.json().catch(() => ({ error: 'Request failed' }));
        throw new Error(error.error || `HTTP ${response.status}`);
      }

      return response.json();
    } catch (error) {
      console.error(`Super Admin API Error [${endpoint}]:`, error);
      throw error;
    }
  }

  // ==================== AUTHENTICATION ====================

  async login(credentials: LoginRequest): Promise<AuthResponse> {
    const response = await this.request<AuthResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    });
    
    if (response.success && response.session_token) {
      this.setToken(response.session_token);
    }
    
    return response;
  }

  async logout(): Promise<void> {
    try {
      await this.request('/auth/logout', {
        method: 'POST',
      });
    } finally {
      this.clearToken();
    }
  }

  async validateSession(): Promise<boolean> {
    try {
      const response = await this.request<{ valid: boolean }>('/auth/validate');
      return response.valid;
    } catch {
      return false;
    }
  }

  // ==================== ADMIN MANAGEMENT ====================

  async getAdmins(): Promise<{ admins: Admin[] }> {
    return this.request<{ admins: Admin[] }>('/admin/admins');
  }

  async getAdmin(id: string): Promise<{ admin: Admin }> {
    return this.request<{ admin: Admin }>(`/admin/admins/${id}`);
  }

  async createAdmin(data: Partial<Admin>): Promise<{ admin: Admin }> {
    return this.request<{ admin: Admin }>('/admin/admins', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateAdmin(id: string, data: Partial<Admin>): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/admin/admins/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteAdmin(id: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/admin/admins/${id}`, {
      method: 'DELETE',
    });
  }

  async suspendAdmin(id: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/admin/admins/${id}/suspend`, {
      method: 'POST',
    });
  }

  async activateAdmin(id: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/admin/admins/${id}/activate`, {
      method: 'POST',
    });
  }

  async updatePermissions(id: string, permissions: string[]): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/admin/admins/${id}/permissions`, {
      method: 'PUT',
      body: JSON.stringify({ permissions }),
    });
  }

  // ==================== WHITE LABEL MANAGEMENT ====================

  async getWhiteLabels(): Promise<{ white_labels: WhiteLabel[] }> {
    return this.request<{ white_labels: WhiteLabel[] }>('/admin/white-labels');
  }

  async getWhiteLabel(id: string): Promise<{ white_label: WhiteLabel }> {
    return this.request<{ white_label: WhiteLabel }>(`/admin/white-labels/${id}`);
  }

  async createWhiteLabel(data: Partial<WhiteLabel>): Promise<{ white_label: WhiteLabel }> {
    return this.request<{ white_label: WhiteLabel }>('/admin/white-labels', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateWhiteLabel(id: string, data: Partial<WhiteLabel>): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/admin/white-labels/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteWhiteLabel(id: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/admin/white-labels/${id}`, {
      method: 'DELETE',
    });
  }

  async approveWhiteLabel(id: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/admin/white-labels/${id}/approve`, {
      method: 'POST',
    });
  }

  async suspendWhiteLabel(id: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/admin/white-labels/${id}/suspend`, {
      method: 'POST',
    });
  }

  async revokeWhiteLabel(id: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/admin/white-labels/${id}/revoke`, {
      method: 'POST',
    });
  }

  async updateWhiteLabelFee(id: string, feePercent: number): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/admin/white-labels/${id}/fee`, {
      method: 'PUT',
      body: JSON.stringify({ fee_percent: feePercent }),
    });
  }

  async regenerateAPIKey(id: string): Promise<{ api_key: string }> {
    return this.request<{ api_key: string }>(`/admin/white-labels/${id}/regenerate-key`, {
      method: 'POST',
    });
  }

  async validateAPIKey(apiKey: string): Promise<{ white_label: WhiteLabel }> {
    return this.request<{ white_label: WhiteLabel }>('/admin/validate-key', {
      method: 'POST',
      body: JSON.stringify({ api_key: apiKey }),
    });
  }

  // ==================== AUDIT LOGS ====================

  async getAuditLogs(adminId?: string, limit: number = 100): Promise<{ audit_logs: AuditLog[] }> {
    const params = new URLSearchParams();
    if (adminId) params.append('admin_id', adminId);
    params.append('limit', limit.toString());
    return this.request<{ audit_logs: AuditLog[] }>(`/admin/audit-logs?${params}`);
  }

  async searchAuditLogs(query: string): Promise<{ audit_logs: AuditLog[] }> {
    return this.request<{ audit_logs: AuditLog[] }>(`/admin/audit-logs/search?q=${encodeURIComponent(query)}`);
  }

  async exportAuditLogs(adminId?: string, format: 'json' | 'csv' = 'json'): Promise<{ data: string }> {
    const params = new URLSearchParams();
    if (adminId) params.append('admin_id', adminId);
    params.append('format', format);
    return this.request<{ data: string }>(`/admin/audit-logs/export?${params}`);
  }

  // ==================== PROFIT SHARING ====================

  async setProfitShare(whiteLabelId: string, percentage: number): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/admin/profit-share/${whiteLabelId}`, {
      method: 'PUT',
      body: JSON.stringify({ percentage }),
    });
  }

  async getProfitShare(whiteLabelId: string): Promise<{ config: ProfitShareConfig }> {
    return this.request<{ config: ProfitShareConfig }>(`/admin/profit-share/${whiteLabelId}`);
  }

  async getProfitHistory(whiteLabelId?: string, limit: number = 50): Promise<{ transactions: ProfitTransaction[] }> {
    const params = new URLSearchParams();
    params.append('limit', limit.toString());
    if (whiteLabelId) params.append('white_label_id', whiteLabelId);
    return this.request<{ transactions: ProfitTransaction[] }>(`/admin/profit-history?${params}`);
  }

  async getTotalProfits(): Promise<{ total: number }> {
    return this.request<{ total: number }>('/admin/total-profits');
  }

  async executeProfitTransfer(whiteLabelId: string, token: string, amount: number): Promise<{ transaction: ProfitTransaction }> {
    return this.request<{ transaction: ProfitTransaction }>(`/admin/profit-transfer`, {
      method: 'POST',
      body: JSON.stringify({ white_label_id: whiteLabelId, token, amount }),
    });
  }

  // ==================== FEATURE FLAGS ====================

  async getFeatures(): Promise<{ features: FeatureFlag[] }> {
    return this.request<{ features: FeatureFlag[] }>('/admin/features');
  }

  async setFeature(featureName: string, enabled: boolean): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/admin/features/${featureName}`, {
      method: 'PUT',
      body: JSON.stringify({ enabled }),
    });
  }

  async isFeatureEnabled(featureName: string): Promise<{ enabled: boolean }> {
    return this.request<{ enabled: boolean }>(`/admin/features/${featureName}/check`);
  }

  // ==================== SESSIONS ====================

  async getSessions(): Promise<{ sessions: Session[] }> {
    return this.request<{ sessions: Session[] }>('/admin/sessions');
  }

  async revokeSession(sessionId: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/admin/sessions/${sessionId}/revoke`, {
      method: 'POST',
    });
  }

  async revokeAllSessions(): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>('/admin/sessions/revoke-all', {
      method: 'POST',
    });
  }
}

// ==================== EXPORT SINGLETON ====================

export const superAdminApi = new SuperAdminAPIClient();

// Export types
export type {
  Admin,
  WhiteLabel,
  AuditLog,
  ProfitTransaction,
  ProfitShareConfig,
  FeatureFlag,
  LoginRequest,
  AuthResponse,
  Session,
};

export default superAdminApi;
