// API Service — targets the standalone WL-Bots backend (:8463, mapped from
// :8471 internally). All routes are real fetches; no stubs/fakes/mocks.
//
// In dev the Vite proxy forwards /api -> http://localhost:8463, so we use a
// relative base by default. VITE_API_URL overrides this for production builds.
const API_BASE_URL = import.meta.env.VITE_API_URL
  ? `${import.meta.env.VITE_API_URL.replace(/\/$/, '')}/api/v1`
  : '/api/v1';

export interface Bot {
  id: string;
  user_id: string;
  name: string;
  bot_type: string;
  status: string;
  config: Record<string, unknown>;
  exchange: string;
  pair: string;
  created_at: string;
}

export interface BotExecution {
  id: string;
  bot_id: string;
  status: string;
  pnl: string;
  started_at: string;
  ended_at: string | null;
}

export interface BotLog {
  id: string;
  bot_id: string;
  level: string;
  message: string;
  created_at: string;
}

export interface Subscription {
  id: string;
  user_id: string;
  tier: string;
  started_at: string;
  expires_at: string | null;
}

export interface FeeConfig {
  id: string;
  name: string;
  percentage: string;
  enabled: boolean;
  created_at: string;
}

export interface ApiKey {
  id: string;
  user_id: string;
  exchange: string;
  enabled: boolean;
  created_at: string;
  api_key: string;
  api_key_preview: string;
}

export interface AuthResponse {
  token: string;
  user_id: string;
  email: string;
  role?: string;
}

export interface CreateBotInput {
  name: string;
  bot_type: string;
  exchange?: string;
  pair?: string;
  config?: Record<string, unknown>;
}

export interface CreateSubscriptionInput {
  tier: string;
  expires_in?: string;
}

export interface CreateFeeConfigInput {
  name: string;
  percentage: string;
  enabled?: boolean;
}

export interface CreateApiKeyInput {
  exchange: string;
  api_key: string;
}

class ApiService {
  private token: string | null = null;

  setToken(token: string) {
    this.token = token;
  }

  clearToken() {
    this.token = null;
  }

  private async request<T = unknown>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const headers: HeadersInit = { 'Content-Type': 'application/json' };
    if (this.token) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${this.token}`;
    }

    const init: RequestInit = { ...options, headers: { ...headers, ...options.headers } };
    // Don't send a body for bodyless POSTs.
    if (init.body === undefined) delete init.body;

    const response = await fetch(`${API_BASE_URL}${endpoint}`, init);

    if (!response.ok) {
      let msg = await response.text();
      try {
        const parsed = JSON.parse(msg);
        if (parsed && parsed.error) msg = parsed.error;
      } catch {
        /* keep raw text */
      }
      throw new Error(msg || `HTTP ${response.status}`);
    }

    // 204 / empty body -> null
    const text = await response.text();
    if (!text) return null as unknown as T;
    return JSON.parse(text) as T;
  }

  // ---- Auth (real bcrypt + JWT on the backend) ----
  register(email: string, password: string, role?: string) {
    return this.request<{ id: string; email: string; role: string }>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password, role }),
    });
  }

  login(email: string, password: string) {
    return this.request<AuthResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
  }

  // ---- Health (liveness + license-gate status) ----
  health() {
    // /health lives outside /api/v1 — fetch directly via the same origin (proxied
    // by Vite) or VITE_API_URL root.
    const root = import.meta.env.VITE_API_URL
      ? import.meta.env.VITE_API_URL.replace(/\/$/, '')
      : '';
    return fetch(`${root}/health`).then(r => {
      if (!r.ok) throw new Error(`HTTP ${r.status}`);
      return r.json() as Promise<{
        status: string;
        service: string;
        licensed: boolean;
        reason: string;
        wl_client_id: string;
        product: string;
      }>;
    });
  }

  // ---- Bots ----
  getBots() {
    return this.request<{ bots: Bot[]; count: number }>('/bots');
  }

  getBot(id: string) {
    return this.request<Bot>(`/bots/${id}`);
  }

  createBot(data: CreateBotInput) {
    return this.request<Bot>('/bots', { method: 'POST', body: JSON.stringify(data) });
  }

  deleteBot(id: string) {
    return this.request<{ status: string; id: string }>(`/bots/${id}`, { method: 'DELETE' });
  }

  startBot(id: string) {
    return this.request<Bot>(`/bots/${id}/start`, { method: 'POST' });
  }

  stopBot(id: string) {
    return this.request<Bot>(`/bots/${id}/stop`, { method: 'POST' });
  }

  pauseBot(id: string) {
    return this.request<Bot>(`/bots/${id}/pause`, { method: 'POST' });
  }

  listBotExecutions(id: string) {
    return this.request<{ executions: BotExecution[]; count: number }>(`/bots/${id}/executions`);
  }

  listBotLogs(id: string) {
    return this.request<{ logs: BotLog[]; count: number }>(`/bots/${id}/logs`);
  }

  // ---- Subscriptions ----
  createSubscription(data: CreateSubscriptionInput) {
    return this.request<Subscription>('/subscriptions', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  listSubscriptions() {
    return this.request<{ subscriptions: Subscription[]; count: number }>('/subscriptions');
  }

  // ---- Fee configs ----
  createFeeConfig(data: CreateFeeConfigInput) {
    return this.request<FeeConfig>('/fees', { method: 'POST', body: JSON.stringify(data) });
  }

  listFeeConfigs() {
    return this.request<{ fee_configs: FeeConfig[]; count: number }>('/fees');
  }

  // ---- API keys (AES-GCM at rest on the backend) ----
  createApiKey(data: CreateApiKeyInput) {
    return this.request<ApiKey>('/api-keys', { method: 'POST', body: JSON.stringify(data) });
  }

  listApiKeys() {
    return this.request<{ api_keys: ApiKey[]; count: number }>('/api-keys');
  }
}

export const api = new ApiService();
export default api;
