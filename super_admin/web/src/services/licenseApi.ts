/**
 * TigerWallet Super Admin - License Control Plane API Service
 *
 * Talks to the license_service control plane via the Vite dev proxy
 * (/license-api -> http://localhost:8460, configured in vite.config.ts). In
 * production, set VITE_LICENSE_API_URL to point at the license service origin.
 * Uses the same SuperAdmin JWT managed by the main api.ts (stored in
 * localStorage under `super_admin_token`).
 *
 * All endpoints are under /api/v1/super-admin and require SuperAdmin JWT auth.
 * No mocks, no fakes — real data from the license control plane.
 */

export const LICENSE_API_URL: string =
  (typeof process !== 'undefined' && (process as any).env?.VITE_LICENSE_API_URL) ||
  '/license-api';

export type WLClientStatus = 'pending' | 'approved' | 'active' | 'suspended' | 'halted' | 'revoked';

export interface WLClient {
  id: string;
  name: string;
  slug: string;
  contact_email: string;
  tier: string;
  status: WLClientStatus;
  branding?: Record<string, any> | null;
  allowed_products?: string[];
  created_at: string;
  updated_at: string;
}

export interface CreateWLClientInput {
  name: string;
  slug: string;
  contact_email: string;
  tier?: string;
  products?: string[];
}

export interface UpdateWLClientInput {
  tier?: string;
  products?: string[];
}

export interface License {
  id: string;
  wl_client_id: string;
  product: string;
  plan: string;
  status: string;
  license_key: string;
  valid_from: string;
  valid_until: string;
  max_users: number;
  max_wallets: number;
  max_bots: number;
  features?: string[];
  issued_by?: string | null;
  created_at: string;
}

export interface IssueLicenseInput {
  wl_client_id: string;
  product: string;
  plan?: string;
  duration_days?: number;
  max_users?: number;
  max_wallets?: number;
  max_bots?: number;
  features?: string[];
}

export interface FeatureFlag {
  id: string;
  wl_client_id: string;
  product: string;
  fetcher: string;
  enabled: boolean;
  updated_at: string;
}

export interface SetFeatureFlagInput {
  wl_client_id: string;
  product: string;
  fetcher?: string;
  enabled: boolean;
}

export interface WithdrawalApproval {
  id: string;
  wl_client_id: string;
  product: string;
  resource_type: string;
  resource_id: string;
  amount_wei: string;
  to_address: string;
  chain_id: number;
  wl_approver_id?: string | null;
  wl_approved_at?: string | null;
  superadmin_approver_id?: string | null;
  superadmin_approved_at?: string | null;
  status: string;
  created_at: string;
  tx_hash?: string;
}

function getToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('super_admin_token');
}

class LicenseApiService {
  baseURL: string;

  constructor(baseURL: string = LICENSE_API_URL) {
    this.baseURL = baseURL;
  }

  private getHeaders(): HeadersInit {
    const token = getToken();
    return {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    };
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const response = await fetch(url, {
      ...options,
      headers: {
        ...this.getHeaders(),
        ...options.headers,
      },
    });

    if (!response.ok) {
      const err = await response.json().catch(() => ({ error: `API Error: ${response.status}` }));
      throw new Error((err && (err.error || err.message)) || `API Error: ${response.status}`);
    }

    if (response.status === 204) return undefined as unknown as T;
    return response.json();
  }

  // ==================== WL Clients ====================

  async listWLClients(): Promise<WLClient[]> {
    const res: any = await this.request('/api/v1/super-admin/wl-clients');
    return (res && res.clients) || (Array.isArray(res) ? res : []) || [];
  }

  async createWLClient(data: CreateWLClientInput): Promise<WLClient> {
    return this.request('/api/v1/super-admin/wl-clients', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateWLClient(id: string, data: UpdateWLClientInput): Promise<{ updated: string }> {
    return this.request(`/api/v1/super-admin/wl-clients/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async approveWLClient(id: string): Promise<{ transitioned: string; status: string }> {
    return this.request(`/api/v1/super-admin/wl-clients/${id}/approve`, { method: 'POST' });
  }

  async suspendWLClient(id: string): Promise<{ transitioned: string; status: string }> {
    return this.request(`/api/v1/super-admin/wl-clients/${id}/suspend`, { method: 'POST' });
  }

  async haltWLClient(id: string): Promise<{ transitioned: string; status: string }> {
    return this.request(`/api/v1/super-admin/wl-clients/${id}/halt`, { method: 'POST' });
  }

  async revokeWLClient(id: string): Promise<{ transitioned: string; status: string }> {
    return this.request(`/api/v1/super-admin/wl-clients/${id}/revoke`, { method: 'POST' });
  }

  async resumeWLClient(id: string): Promise<{ resumed: string }> {
    return this.request(`/api/v1/super-admin/wl-clients/${id}/resume`, { method: 'POST' });
  }

  // ==================== Licenses ====================

  async listLicenses(wlClientId?: string): Promise<License[]> {
    const query = wlClientId ? `?wl_client_id=${encodeURIComponent(wlClientId)}` : '';
    const res: any = await this.request(`/api/v1/super-admin/licenses${query}`);
    return (res && res.licenses) || (Array.isArray(res) ? res : []) || [];
  }

  async issueLicense(data: IssueLicenseInput): Promise<License> {
    return this.request('/api/v1/super-admin/licenses', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async suspendLicense(id: string): Promise<{ transitioned: string; status: string }> {
    return this.request(`/api/v1/super-admin/licenses/${id}/suspend`, { method: 'POST' });
  }

  async haltLicense(id: string): Promise<{ transitioned: string; status: string }> {
    return this.request(`/api/v1/super-admin/licenses/${id}/halt`, { method: 'POST' });
  }

  async revokeLicense(id: string): Promise<{ transitioned: string; status: string }> {
    return this.request(`/api/v1/super-admin/licenses/${id}/revoke`, { method: 'POST' });
  }

  async resumeLicense(id: string): Promise<{ transitioned: string; status: string }> {
    return this.request(`/api/v1/super-admin/licenses/${id}/resume`, { method: 'POST' });
  }

  // ==================== Feature Flags (per-fetcher) ====================

  async listFeatureFlags(wlClientId: string, product?: string): Promise<FeatureFlag[]> {
    const params = new URLSearchParams({ wl_client_id: wlClientId });
    if (product) params.append('product', product);
    const res: any = await this.request(`/api/v1/super-admin/feature-flags?${params}`);
    return (res && res.flags) || (Array.isArray(res) ? res : []) || [];
  }

  async setFeatureFlag(data: SetFeatureFlagInput): Promise<{ updated: boolean; product: string; fetcher: string; enabled: boolean }> {
    return this.request('/api/v1/super-admin/feature-flags', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // ==================== Two-Party Withdrawals ====================

  async listWithdrawals(wlClientId?: string): Promise<WithdrawalApproval[]> {
    const query = wlClientId ? `?wl_client_id=${encodeURIComponent(wlClientId)}` : '';
    const res: any = await this.request(`/api/v1/super-admin/withdrawals${query}`);
    return (res && res.withdrawals) || (Array.isArray(res) ? res : []) || [];
  }

  async approveWithdrawal(id: string): Promise<{ approved: string; two_party_complete: boolean }> {
    return this.request(`/api/v1/super-admin/withdrawals/${id}/approve`, { method: 'POST' });
  }

  async rejectWithdrawal(id: string): Promise<{ rejected: string }> {
    return this.request(`/api/v1/super-admin/withdrawals/${id}/reject`, { method: 'POST' });
  }

  async isWithdrawalApproved(id: string): Promise<{ approved: boolean }> {
    return this.request(`/api/v1/super-admin/withdrawals/${id}/approved`);
  }
}

export const licenseApi = new LicenseApiService();
export default licenseApi;
