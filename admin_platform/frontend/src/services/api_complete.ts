/**
 * TigerWallet Admin Platform - Complete Frontend API Service
 * Connects all admin frontends to all backend services
 * 
 * Features:
 * - Complete CRUD operations
 * - Authentication (JWT, 2FA)
 * - Real-time notifications
 * - Dark/Light theme support
 * - Multi-language (i18n)
 * - Rate limiting
 * - Error handling
 * - Request/Response interceptors
 */

import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import { BehaviorSubject, Observable } from 'rxjs';

// ============================================================================
// Types
// ============================================================================

export interface Admin {
  id: number;
  username: string;
  email: string;
  role: string;
  status: string;
  permissions: string[];
  two_factor_enabled: boolean;
  security_level: number;
  ip_whitelist: string[];
  session_count: number;
  max_sessions: number;
  last_login?: string;
  last_ip?: string;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  email: string;
  password: string;
  two_factor_code?: string;
  ip_address?: string;
  user_agent?: string;
}

export interface LoginResponse {
  token: string;
  refresh_token: string;
  admin: Admin;
  expires_in: number;
}

export interface Session {
  id: number;
  admin_id: number;
  token: string;
  ip_address: string;
  user_agent: string;
  expires_at: string;
  last_activity: string;
  is_active: boolean;
}

export interface AuditLog {
  id: number;
  admin_id: number;
  admin_email: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  details?: string;
  ip_address: string;
  user_agent: string;
  status: string;
  created_at: string;
}

export interface Notification {
  id: number;
  admin_id: number;
  title: string;
  message: string;
  notification_type: string;
  is_read: boolean;
  created_at: string;
}

export interface ScheduledTask {
  id: number;
  name: string;
  description: string;
  cron_expression: string;
  task_type: string;
  config: any;
  status: string;
  last_run?: string;
  next_run?: string;
  created_at: string;
  updated_at: string;
}

export interface WebhookConfig {
  id: number;
  name: string;
  url: string;
  events: string[];
  secret: string;
  is_active: boolean;
  retry_count: number;
  timeout_seconds: number;
  created_at: string;
  updated_at: string;
}

export interface ThemePreference {
  admin_id: number;
  theme_mode: 'light' | 'dark' | 'system';
  language: string;
}

export interface ApprovalWorkflow {
  id: number;
  name: string;
  description: string;
  resource_type: string;
  required_roles: string[];
  approval_levels: number;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface ApprovalRequest {
  id: number;
  workflow_id: number;
  resource_type: string;
  resource_id: string;
  requester_id: number;
  requester_email: string;
  details: string;
  status: string;
  current_level: number;
  approvals: Approval[];
  created_at: string;
  updated_at: string;
}

export interface Approval {
  id: string;
  request_id: string;
  approver_id: number;
  approver_email: string;
  level: number;
  decision: string;
  comments?: string;
  created_at: string;
}

export interface Ticket {
  id: number;
  title: string;
  description: string;
  category: string;
  priority: string;
  status: string;
  creator_id: number;
  creator_email: string;
  assigned_to?: number;
  comments?: TicketComment[];
  attachments?: string[];
  created_at: string;
  updated_at: string;
  resolved_at?: string;
}

export interface TicketComment {
  id: number;
  ticket_id: number;
  author_id: number;
  author_email: string;
  content: string;
  created_at: string;
}

export interface KnowledgeArticle {
  id: number;
  title: string;
  content: string;
  category: string;
  tags: string[];
  author_id: number;
  status: string;
  view_count: number;
  created_at: string;
  updated_at: string;
}

export interface SLAMetric {
  id: number;
  metric_name: string;
  target_value: number;
  current_value: number;
  time_window: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface FraudAlert {
  id: number;
  admin_id: number;
  alert_type: string;
  severity: string;
  description: string;
  details: any;
  status: string;
  created_at: string;
  resolved_at?: string;
  resolved_by?: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
}

export interface ApiError {
  error: string;
  message?: string;
  code?: string;
}

// Theme types
export type ThemeMode = 'light' | 'dark' | 'system';

// Language types
export type Language = 'en' | 'es' | 'fr' | 'de' | 'zh' | 'ja' | 'ko' | 'ar';

// ============================================================================
// API Service
// ============================================================================

class AdminApiService {
  private api: AxiosInstance;
  private token: string | null = null;
  private refreshToken: string | null = null;
  
  // State observables
  private adminSubject = new BehaviorSubject<Admin | null>(null);
  private themeSubject = new BehaviorSubject<ThemeMode>('system');
  private languageSubject = new BehaviorSubject<Language>('en');
  private notificationsSubject = new BehaviorSubject<Notification[]>([]);
  private isLoadingSubject = new BehaviorSubject<boolean>(false);

  // API base URLs (can be configured for different deployments)
  private baseURLs = {
    go: process.env.REACT_APP_GO_API_URL || 'http://localhost:9093',
    rust: process.env.REACT_APP_RUST_API_URL || 'http://localhost:9094',
    cpp: process.env.REACT_APP_CPP_API_URL || 'http://localhost:9095',
  };

  // Current backend selection
  private currentBackend: 'go' | 'rust' | 'cpp' = 'go';

  constructor() {
    this.api = this.createAxiosInstance();
    this.loadStoredAuth();
    this.loadThemePreference();
    this.loadLanguagePreference();
  }

  private createAxiosInstance(): AxiosInstance {
    const instance = axios.create({
      baseURL: this.baseURLs[this.currentBackend],
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Request interceptor
    instance.interceptors.request.use(
      (config: InternalAxiosRequestConfig) => {
        // Add auth token
        if (this.token) {
          config.headers.Authorization = `Bearer ${this.token}`;
        }

        // Add language header
        config.headers['X-Language'] = this.languageSubject.value;

        // Add theme header
        config.headers['X-Theme'] = this.themeSubject.value;

        this.isLoadingSubject.next(true);
        return config;
      },
      (error) => {
        this.isLoadingSubject.next(false);
        return Promise.reject(error);
      }
    );

    // Response interceptor
    instance.interceptors.response.use(
      (response: AxiosResponse) => {
        this.isLoadingSubject.next(false);
        return response;
      },
      async (error) => {
        this.isLoadingSubject.next(false);

        const originalRequest = error.config;

        // Handle 401 - Unauthorized
        if (error.response?.status === 401 && !originalRequest._retry) {
          originalRequest._retry = true;

          try {
            await this.refreshTokenAuth();
            originalRequest.headers.Authorization = `Bearer ${this.token}`;
            return instance(originalRequest);
          } catch (refreshError) {
            this.logout();
            return Promise.reject(refreshError);
          }
        }

        // Handle rate limiting
        if (error.response?.status === 429) {
          console.error('Rate limit exceeded. Please try again later.');
        }

        return Promise.reject(error);
      }
    );

    return instance;
  }

  // ============================================================================
  // Authentication
  // ============================================================================

  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await this.api.post<LoginResponse>('/api/v1/auth/login', credentials);
    
    this.token = response.data.token;
    this.refreshToken = response.data.refresh_token;
    this.adminSubject.next(response.data.admin);
    
    this.storeAuth(response.data.token, response.data.refresh_token);
    
    return response.data;
  }

  async register(data: {
    username: string;
    email: string;
    password: string;
    role?: string;
  }): Promise<Admin> {
    const response = await this.api.post<Admin>('/api/v1/auth/register', data);
    return response.data;
  }

  async logout(): Promise<void> {
    try {
      await this.api.post('/api/v1/auth/logout');
    } catch (error) {
      // Continue with logout even if API call fails
    }
    
    this.token = null;
    this.refreshToken = null;
    this.adminSubject.next(null);
    this.clearStoredAuth();
  }

  async refreshTokenAuth(): Promise<void> {
    if (!this.refreshToken) {
      throw new Error('No refresh token available');
    }

    const response = await this.api.post<LoginResponse>('/api/v1/auth/refresh', {
      refresh_token: this.refreshToken,
    });

    this.token = response.data.token;
    this.refreshToken = response.data.refresh_token;
    this.adminSubject.next(response.data.admin);
    
    this.storeAuth(response.data.token, response.data.refresh_token);
  }

  // ============================================================================
  // Admin Management
  // ============================================================================

  async getAdmin(id: number): Promise<Admin> {
    const response = await this.api.get<Admin>(`/api/v1/admins/${id}`);
    return response.data;
  }

  async listAdmins(page: number = 1, limit: number = 20): Promise<PaginatedResponse<Admin>> {
    const response = await this.api.get<PaginatedResponse<Admin>>('/api/v1/admins', {
      params: { page, limit },
    });
    return response.data;
  }

  async updateAdmin(id: number, data: Partial<Admin>): Promise<Admin> {
    const response = await this.api.put<Admin>(`/api/v1/admins/${id}`, data);
    return response.data;
  }

  async deleteAdmin(id: number): Promise<void> {
    await this.api.delete(`/api/v1/admins/${id}`);
  }

  async suspendAdmin(id: number): Promise<void> {
    await this.api.post(`/api/v1/admins/${id}/suspend`);
  }

  // ============================================================================
  // Two-Factor Authentication
  // ============================================================================

  async enableTwoFactor(adminId: number): Promise<string> {
    const response = await this.api.post<{ secret: string }>(`/api/v1/admins/${adminId}/two-factor/enable`);
    return response.data.secret;
  }

  async disableTwoFactor(adminId: number, code: string): Promise<void> {
    await this.api.post(`/api/v1/admins/${adminId}/two-factor/disable`, { code });
  }

  // ============================================================================
  // Password Management
  // ============================================================================

  async changePassword(adminId: number, oldPassword: string, newPassword: string): Promise<void> {
    await this.api.post(`/api/v1/admins/${adminId}/password`, {
      old_password: oldPassword,
      new_password: newPassword,
    });
  }

  // ============================================================================
  // Session Management
  // ============================================================================

  async listSessions(adminId: number): Promise<Session[]> {
    const response = await this.api.get<Session[]>(`/api/v1/admins/${adminId}/sessions`);
    return response.data;
  }

  async revokeSession(adminId: number, sessionId: number): Promise<void> {
    await this.api.delete(`/api/v1/admins/${adminId}/sessions/${sessionId}`);
  }

  async revokeAllSessions(adminId: number): Promise<void> {
    await this.api.delete(`/api/v1/admins/${adminId}/sessions`);
  }

  // ============================================================================
  // IP Whitelist
  // ============================================================================

  async addIPToWhitelist(adminId: number, ip: string): Promise<Admin> {
    const response = await this.api.post<Admin>(`/api/v1/admins/${adminId}/whitelist`, { ip });
    return response.data;
  }

  async removeIPFromWhitelist(adminId: number, ip: string): Promise<Admin> {
    const response = await this.api.delete<Admin>(`/api/v1/admins/${adminId}/whitelist/${ip}`);
    return response.data;
  }

  // ============================================================================
  // Notifications
  // ============================================================================

  async getNotifications(): Promise<Notification[]> {
    const response = await this.api.get<Notification[]>('/api/v1/notifications');
    this.notificationsSubject.next(response.data);
    return response.data;
  }

  async markNotificationRead(notificationId: number): Promise<void> {
    await this.api.put(`/api/v1/notifications/${notificationId}/read`);
    
    // Update local state
    const notifications = this.notificationsSubject.value.map(n =>
      n.id === notificationId ? { ...n, is_read: true } : n
    );
    this.notificationsSubject.next(notifications);
  }

  // ============================================================================
  // Audit Logs
  // ============================================================================

  async getAuditLogs(
    adminId?: number,
    action?: string,
    page: number = 1,
    limit: number = 50
  ): Promise<PaginatedResponse<AuditLog>> {
    const response = await this.api.get<PaginatedResponse<AuditLog>>('/api/v1/audit-logs', {
      params: { admin_id: adminId, action, page, limit },
    });
    return response.data;
  }

  // ============================================================================
  // Scheduled Tasks
  // ============================================================================

  async listScheduledTasks(): Promise<ScheduledTask[]> {
    const response = await this.api.get<ScheduledTask[]>('/api/v1/tasks');
    return response.data;
  }

  async createScheduledTask(task: Partial<ScheduledTask>): Promise<ScheduledTask> {
    const response = await this.api.post<ScheduledTask>('/api/v1/tasks', task);
    return response.data;
  }

  async updateScheduledTask(id: number, task: Partial<ScheduledTask>): Promise<ScheduledTask> {
    const response = await this.api.put<ScheduledTask>(`/api/v1/tasks/${id}`, task);
    return response.data;
  }

  async deleteScheduledTask(id: number): Promise<void> {
    await this.api.delete(`/api/v1/tasks/${id}`);
  }

  async runScheduledTask(id: number): Promise<void> {
    await this.api.post(`/api/v1/tasks/${id}/run`);
  }

  // ============================================================================
  // Webhooks
  // ============================================================================

  async listWebhooks(): Promise<WebhookConfig[]> {
    const response = await this.api.get<WebhookConfig[]>('/api/v1/webhooks');
    return response.data;
  }

  async createWebhook(webhook: Partial<WebhookConfig>): Promise<WebhookConfig> {
    const response = await this.api.post<WebhookConfig>('/api/v1/webhooks', webhook);
    return response.data;
  }

  async updateWebhook(id: number, webhook: Partial<WebhookConfig>): Promise<WebhookConfig> {
    const response = await this.api.put<WebhookConfig>(`/api/v1/webhooks/${id}`, webhook);
    return response.data;
  }

  async deleteWebhook(id: number): Promise<void> {
    await this.api.delete(`/api/v1/webhooks/${id}`);
  }

  // ============================================================================
  // Theme & Language
  // ============================================================================

  async getThemePreference(): Promise<ThemePreference> {
    const response = await this.api.get<ThemePreference>('/api/v1/theme');
    return response.data;
  }

  async setThemePreference(themeMode: ThemeMode): Promise<ThemePreference> {
    const response = await this.api.put<ThemePreference>('/api/v1/theme', { theme_mode: themeMode });
    this.themeSubject.next(themeMode);
    this.applyTheme(themeMode);
    this.storeThemePreference(themeMode);
    return response.data;
  }

  async getLanguagePreference(): Promise<{ language: Language }> {
    const response = await this.api.get<{ language: Language }>('/api/v1/language');
    return response.data;
  }

  async setLanguagePreference(language: Language): Promise<void> {
    await this.api.put('/api/v1/language', { language });
    this.languageSubject.next(language);
    this.storeLanguagePreference(language);
  }

  // ============================================================================
  // Approval Workflows
  // ============================================================================

  async listApprovalWorkflows(): Promise<ApprovalWorkflow[]> {
    const response = await this.api.get<ApprovalWorkflow[]>('/api/v1/workflows');
    return response.data;
  }

  async createApprovalWorkflow(workflow: Partial<ApprovalWorkflow>): Promise<ApprovalWorkflow> {
    const response = await this.api.post<ApprovalWorkflow>('/api/v1/workflows', workflow);
    return response.data;
  }

  async submitApprovalRequest(request: Partial<ApprovalRequest>): Promise<ApprovalRequest> {
    const response = await this.api.post<ApprovalRequest>('/api/v1/workflows/requests', request);
    return response.data;
  }

  async approveRequest(requestId: number, comments?: string): Promise<ApprovalRequest> {
    const response = await this.api.post<ApprovalRequest>(`/api/v1/workflows/requests/${requestId}/approve`, {
      comments,
    });
    return response.data;
  }

  async rejectRequest(requestId: number, comments?: string): Promise<ApprovalRequest> {
    const response = await this.api.post<ApprovalRequest>(`/api/v1/workflows/requests/${requestId}/reject`, {
      comments,
    });
    return response.data;
  }

  // ============================================================================
  // Tickets
  // ============================================================================

  async listTickets(
    status?: string,
    page: number = 1,
    limit: number = 20
  ): Promise<PaginatedResponse<Ticket>> {
    const response = await this.api.get<PaginatedResponse<Ticket>>('/api/v1/tickets', {
      params: { status, page, limit },
    });
    return response.data;
  }

  async createTicket(ticket: Partial<Ticket>): Promise<Ticket> {
    const response = await this.api.post<Ticket>('/api/v1/tickets', ticket);
    return response.data;
  }

  async updateTicket(id: number, ticket: Partial<Ticket>): Promise<Ticket> {
    const response = await this.api.put<Ticket>(`/api/v1/tickets/${id}`, ticket);
    return response.data;
  }

  async addTicketComment(ticketId: number, content: string): Promise<TicketComment> {
    const response = await this.api.post<TicketComment>(`/api/v1/tickets/${ticketId}/comments`, {
      content,
    });
    return response.data;
  }

  // ============================================================================
  // Knowledge Base
  // ============================================================================

  async searchKnowledgeBase(query: string): Promise<KnowledgeArticle[]> {
    const response = await this.api.get<KnowledgeArticle[]>('/api/v1/knowledge/search', {
      params: { query },
    });
    return response.data;
  }

  async createKnowledgeArticle(article: Partial<KnowledgeArticle>): Promise<KnowledgeArticle> {
    const response = await this.api.post<KnowledgeArticle>('/api/v1/knowledge/articles', article);
    return response.data;
  }

  async updateKnowledgeArticle(id: number, article: Partial<KnowledgeArticle>): Promise<KnowledgeArticle> {
    const response = await this.api.put<KnowledgeArticle>(`/api/v1/knowledge/articles/${id}`, article);
    return response.data;
  }

  // ============================================================================
  // SLA Metrics
  // ============================================================================

  async getSLAMetrics(): Promise<SLAMetric[]> {
    const response = await this.api.get<SLAMetric[]>('/api/v1/sla');
    return response.data;
  }

  async createSLAMetric(metric: Partial<SLAMetric>): Promise<SLAMetric> {
    const response = await this.api.post<SLAMetric>('/api/v1/sla', metric);
    return response.data;
  }

  async updateSLAMetric(id: number, metric: Partial<SLAMetric>): Promise<SLAMetric> {
    const response = await this.api.put<SLAMetric>(`/api/v1/sla/${id}`, metric);
    return response.data;
  }

  // ============================================================================
  // Fraud Alerts
  // ============================================================================

  async getFraudAlerts(status?: string): Promise<FraudAlert[]> {
    const response = await this.api.get<FraudAlert[]>('/api/v1/fraud', {
      params: { status },
    });
    return response.data;
  }

  async createFraudAlert(alert: Partial<FraudAlert>): Promise<FraudAlert> {
    const response = await this.api.post<FraudAlert>('/api/v1/fraud', alert);
    return response.data;
  }

  async resolveFraudAlert(alertId: number, status: string): Promise<void> {
    await this.api.post(`/api/v1/fraud/${alertId}/resolve`, { status });
  }

  // ============================================================================
  // Rate Limiting
  // ============================================================================

  async getRateLimitStatus(): Promise<any> {
    const response = await this.api.get('/api/v1/rate-limit');
    return response.data;
  }

  // ============================================================================
  // Backend Switching
  // ============================================================================

  setBackend(backend: 'go' | 'rust' | 'cpp'): void {
    this.currentBackend = backend;
    this.api = this.createAxiosInstance();
  }

  getBackend(): string {
    return this.currentBackend;
  }

  // ============================================================================
  // Theme & Language Management (Client-side)
  // ============================================================================

  private applyTheme(mode: ThemeMode): void {
    const root = document.documentElement;
    
    if (mode === 'system') {
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      root.setAttribute('data-theme', prefersDark ? 'dark' : 'light');
    } else {
      root.setAttribute('data-theme', mode);
    }
  }

  private loadThemePreference(): void {
    const stored = localStorage.getItem('admin_theme');
    if (stored && ['light', 'dark', 'system'].includes(stored)) {
      this.themeSubject.next(stored as ThemeMode);
      this.applyTheme(stored as ThemeMode);
    } else {
      this.applyTheme('system');
    }

    // Listen for system theme changes
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if (this.themeSubject.value === 'system') {
        this.applyTheme('system');
      }
    });
  }

  private storeThemePreference(theme: ThemeMode): void {
    localStorage.setItem('admin_theme', theme);
  }

  private loadLanguagePreference(): void {
    const stored = localStorage.getItem('admin_language');
    if (stored && ['en', 'es', 'fr', 'de', 'zh', 'ja', 'ko', 'ar'].includes(stored)) {
      this.languageSubject.next(stored as Language);
    }
  }

  private storeLanguagePreference(language: Language): void {
    localStorage.setItem('admin_language', language);
  }

  // ============================================================================
  // Auth Storage
  // ============================================================================

  private loadStoredAuth(): void {
    const token = localStorage.getItem('admin_token');
    const refreshToken = localStorage.getItem('admin_refresh_token');
    
    if (token) {
      this.token = token;
    }
    if (refreshToken) {
      this.refreshToken = refreshToken;
    }
  }

  private storeAuth(token: string, refreshToken: string): void {
    localStorage.setItem('admin_token', token);
    localStorage.setItem('admin_refresh_token', refreshToken);
  }

  private clearStoredAuth(): void {
    localStorage.removeItem('admin_token');
    localStorage.removeItem('admin_refresh_token');
  }

  // ============================================================================
  // Observables
  // ============================================================================

  get admin$(): Observable<Admin | null> {
    return this.adminSubject.asObservable();
  }

  get theme$(): Observable<ThemeMode> {
    return this.themeSubject.asObservable();
  }

  get language$(): Observable<Language> {
    return this.languageSubject.asObservable();
  }

  get notifications$(): Observable<Notification[]> {
    return this.notificationsSubject.asObservable();
  }

  get isLoading$(): Observable<boolean> {
    return this.isLoadingSubject.asObservable();
  }

  // ============================================================================
  // Utility Methods
  // ============================================================================

  isAuthenticated(): boolean {
    return !!this.token;
  }

  hasPermission(permission: string): boolean {
    const admin = this.adminSubject.value;
    if (!admin) return false;
    return admin.permissions.includes(permission);
  }

  hasRole(role: string): boolean {
    const admin = this.adminSubject.value;
    if (!admin) return false;
    return admin.role === role;
  }

  isSuperAdmin(): boolean {
    return this.hasRole('super_admin');
  }

  isComplianceAdmin(): boolean {
    return this.hasRole('compliance_admin');
  }

  isFinanceAdmin(): boolean {
    return this.hasRole('finance_admin');
  }

  isSecurityAdmin(): boolean {
    return this.hasRole('security_admin');
  }
}

// Export singleton instance
export const adminApi = new AdminApiService();
export default adminApi;

// ============================================================================
// React Hook
// ============================================================================

import { useState, useEffect, useCallback } from 'react';

export function useAdmin() {
  const [admin, setAdmin] = useState<Admin | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const subscription = adminApi.admin$.subscribe(setAdmin);
    return () => subscription.unsubscribe();
  }, []);

  useEffect(() => {
    const subscription = adminApi.isLoading$.subscribe(setLoading);
    return () => subscription.unsubscribe();
  }, []);

  return { admin, loading, isAuthenticated: adminApi.isAuthenticated() };
}

export function useTheme() {
  const [theme, setTheme] = useState<ThemeMode>('system');

  useEffect(() => {
    const subscription = adminApi.theme$.subscribe(setTheme);
    return () => subscription.unsubscribe();
  }, []);

  const changeTheme = useCallback(async (newTheme: ThemeMode) => {
    await adminApi.setThemePreference(newTheme);
  }, []);

  return { theme, changeTheme };
}

export function useLanguage() {
  const [language, setLanguage] = useState<Language>('en');

  useEffect(() => {
    const subscription = adminApi.language$.subscribe(setLanguage);
    return () => subscription.unsubscribe();
  }, []);

  const changeLanguage = useCallback(async (newLanguage: Language) => {
    await adminApi.setLanguagePreference(newLanguage);
  }, []);

  return { language, changeLanguage };
}

export function useNotifications() {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);

  useEffect(() => {
    const subscription = adminApi.notifications$.subscribe(setNotifications);
    return () => subscription.unsubscribe();
  }, []);

  useEffect(() => {
    setUnreadCount(notifications.filter(n => !n.is_read).length);
  }, [notifications]);

  const markAsRead = useCallback(async (id: number) => {
    await adminApi.markNotificationRead(id);
  }, []);

  const refresh = useCallback(async () => {
    await adminApi.getNotifications();
  }, []);

  return { notifications, unreadCount, markAsRead, refresh };
}
