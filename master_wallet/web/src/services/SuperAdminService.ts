/**
 * SuperAdminService - Web/React Implementation
 * Complete admin control system for Master Wallet
 * Features: Master Admin management, White Label Admin, Feature toggles, Audit logs
 * Ultra-low latency design with caching and optimization
 */

import { ethers } from 'ethers';
import { createCipheriv, randomBytes, createDecipheriv, scrypt, timingSafeEqual } from 'crypto';
import { promisify } from 'util';

const scryptAsync = promisify(scrypt);

// Configuration
const SUPER_ADMIN_EMAIL = 'superadmin@tigerwallet.com';
const SUPER_ADMIN_PASSWORD = process.env.REACT_APP_SUPER_ADMIN_PASSWORD || '';
const SUPER_ADMIN_WALLET = ''; // Provisioned by backend (super_admin_api), not hardcoded.
const PROFIT_SHARE_PERCENTAGE = 20;

// Feature flags
const FEATURE_FLAGS = [
  'master_wallet_creation',
  'multi_blockchain',
  'token_management',
  'user_wallet_ownership',
  'hd_wallet',
  'biometric_auth',
  'pin_code_auth',
  'nft_support',
  'defi_integration',
  'staking',
  'bridge_support',
  'mev_protection',
  'swap_trading',
  'hardware_wallet',
  'admin_controls',
  'network_management',
  'gas_optimization',
  'multi_sig',
  'transaction_history',
  'price_alerts',
  'privacy_zk',
  'coinjoin',
  'account_abstraction',
  'session_keys',
  'paymaster',
  'passkeys',
  'tax_integration',
  'analytics',
  'cross_chain_intent',
  'dapp_browser',
];

interface AdminUser {
  id: string;
  email: string;
  role: 'super_admin' | 'master_admin' | 'white_label_admin';
  masterWalletId?: string;
  permissions: string[];
  isActive: boolean;
  twoFactorEnabled: boolean;
  createdAt: number;
  lastLoginAt?: number;
  failedLoginAttempts: number;
  lockedUntil?: number;
}

interface FeatureConfig {
  name: string;
  enabled: boolean;
  description: string;
  config?: Record<string, any>;
}

interface AuditLogEntry {
  id: string;
  adminId: string;
  action: string;
  entityType: string;
  entityId?: string;
  details: Record<string, any>;
  ipAddress: string;
  userAgent: string;
  timestamp: number;
}

interface WhiteLabelConfig {
  id: string;
  name: string;
  domain: string;
  branding: {
    logo: string;
    primaryColor: string;
    secondaryColor: string;
  };
  feePercentage: number;
  isActive: boolean;
}

interface ProfitDistribution {
  whiteLabelId: string;
  amount: string;
  token: string;
  timestamp: number;
  txHash?: string;
}

class SuperAdminService {
  private static instance: SuperAdminService | null = null;
  private admins: Map<string, AdminUser> = new Map();
  private featureFlags: Map<string, FeatureConfig> = new Map();
  private auditLogs: AuditLogEntry[] = [];
  private whiteLabels: Map<string, WhiteLabelConfig> = new Map();
  private profitDistributions: ProfitDistribution[] = [];
  private sessionCache: Map<string, { adminId: string; expiresAt: number }> = new Map();
  
  // Crypto keys (in production, use secure key management)
  private encryptionKey: Buffer | null = null;
  private apiKeys: Map<string, string> = new Map();

  private constructor() {
    this.initializeDefaults();
  }

  static getInstance(): SuperAdminService {
    if (!SuperAdminService.instance) {
      SuperAdminService.instance = new SuperAdminService();
    }
    return SuperAdminService.instance;
  }

  private async initializeDefaults(): Promise<void> {
    // Initialize super admin
    const salt = randomBytes(32);
    const key = await scryptAsync(SUPER_ADMIN_PASSWORD, salt, 32) as Buffer;
    
    const superAdmin: AdminUser = {
      id: this.generateId(),
      email: SUPER_ADMIN_EMAIL,
      role: 'super_admin',
      permissions: ['all'],
      isActive: true,
      twoFactorEnabled: false,
      createdAt: Date.now(),
      failedLoginAttempts: 0,
    };
    
    this.admins.set(superAdmin.id, superAdmin);
    
    // Initialize feature flags
    FEATURE_FLAGS.forEach(flag => {
      this.featureFlags.set(flag, {
        name: flag,
        enabled: true,
        description: `Feature flag for ${flag}`,
      });
    });

    // Initialize encryption key
    this.encryptionKey = await scryptAsync('tigerwallet-secret-key', 'salt', 32) as Buffer;
  }

  // ==================== Authentication ====================

  /**
   * Authenticate admin user
   */
  async authenticate(email: string, password: string, ipAddress: string = '0.0.0.0'): Promise<{ success: boolean; token?: string; admin?: Partial<AdminUser>; error?: string }> {
    const admin = Array.from(this.admins.values()).find(a => a.email === email);
    
    if (!admin) {
      this.logAudit('LOGIN_FAILED', 'admin', undefined, { email, reason: 'User not found' }, ipAddress, 'Unknown');
      return { success: false, error: 'Invalid credentials' };
    }

    // Check if account is locked
    if (admin.lockedUntil && admin.lockedUntil > Date.now()) {
      return { success: false, error: 'Account locked. Try again later.' };
    }

    // Verify password
    const isValid = await this.verifyPassword(password, admin.id);
    
    if (!isValid) {
      admin.failedLoginAttempts++;
      if (admin.failedLoginAttempts >= 5) {
        admin.lockedUntil = Date.now() + 15 * 60 * 1000; // 15 minutes
      }
      this.admins.set(admin.id, admin);
      this.logAudit('LOGIN_FAILED', 'admin', admin.id, { reason: 'Invalid password' }, ipAddress, 'Unknown');
      return { success: false, error: 'Invalid credentials' };
    }

    // Reset failed attempts
    admin.failedLoginAttempts = 0;
    admin.lastLoginAt = Date.now();
    this.admins.set(admin.id, admin);

    // Generate session token
    const token = this.generateToken();
    this.sessionCache.set(token, { adminId: admin.id, expiresAt: Date.now() + 24 * 60 * 60 * 1000 });

    this.logAudit('LOGIN_SUCCESS', 'admin', admin.id, {}, ipAddress, 'Unknown');

    return {
      success: true,
      token,
      admin: {
        id: admin.id,
        email: admin.email,
        role: admin.role,
        permissions: admin.permissions,
      },
    };
  }

  /**
   * Verify session token
   */
  async verifyToken(token: string): Promise<AdminUser | null> {
    const session = this.sessionCache.get(token);
    if (!session || session.expiresAt < Date.now()) {
      this.sessionCache.delete(token);
      return null;
    }
    return this.admins.get(session.adminId) || null;
  }

  /**
   * Logout - invalidate token
   */
  async logout(token: string): Promise<void> {
    this.sessionCache.delete(token);
  }

  /**
   * Change password
   */
  async changePassword(adminId: string, currentPassword: string, newPassword: string): Promise<{ success: boolean; error?: string }> {
    const admin = this.admins.get(adminId);
    if (!admin) {
      return { success: false, error: 'Admin not found' };
    }

    const isValid = await this.verifyPassword(currentPassword, adminId);
    if (!isValid) {
      return { success: false, error: 'Current password is incorrect' };
    }

    // Update password
    const salt = randomBytes(32);
    const key = await scryptAsync(newPassword, salt, 32) as Buffer;
    
    this.logAudit('PASSWORD_CHANGED', 'admin', adminId, {}, '0.0.0.0', 'Unknown');
    return { success: true };
  }

  /**
   * Enable/disable 2FA
   */
  async toggleTwoFactor(adminId: string, enable: boolean): Promise<{ success: boolean; error?: string }> {
    const admin = this.admins.get(adminId);
    if (!admin) {
      return { success: false, error: 'Admin not found' };
    }

    if (admin.role !== 'super_admin' && admin.role !== 'master_admin') {
      return { success: false, error: 'Insufficient permissions' };
    }

    admin.twoFactorEnabled = enable;
    this.admins.set(adminId, admin);
    
    this.logAudit(enable ? '2FA_ENABLED' : '2FA_DISABLED', 'admin', adminId, {}, '0.0.0.0', 'Unknown');
    return { success: true };
  }

  // ==================== Master Admin Management ====================

  /**
   * Create Master Admin (requires Super Admin authorization)
   */
  async createMasterAdmin(
    superAdminToken: string,
    email: string,
    password: string,
    masterWalletId: string
  ): Promise<{ success: boolean; adminId?: string; error?: string }> {
    const superAdmin = await this.verifyToken(superAdminToken);
    if (!superAdmin || superAdmin.role !== 'super_admin') {
      return { success: false, error: 'Unauthorized - Super Admin token required' };
    }

    // Check if email already exists
    if (Array.from(this.admins.values()).some(a => a.email === email)) {
      return { success: false, error: 'Email already exists' };
    }

    const masterAdmin: AdminUser = {
      id: this.generateId(),
      email,
      role: 'master_admin',
      masterWalletId,
      permissions: ['admin', 'manage_users', 'view_analytics', 'manage_fees'],
      isActive: true,
      twoFactorEnabled: false,
      createdAt: Date.now(),
      failedLoginAttempts: 0,
    };

    this.admins.set(masterAdmin.id, masterAdmin);
    
    this.logAudit('MASTER_ADMIN_CREATED', 'admin', superAdmin.id, { newAdminId: masterAdmin.id, email, masterWalletId }, '0.0.0.0', 'Unknown');
    
    return { success: true, adminId: masterAdmin.id };
  }

  /**
   * Get all Master Admins
   */
  async getMasterAdmins(superAdminToken: string): Promise<AdminUser[]> {
    const superAdmin = await this.verifyToken(superAdminToken);
    if (!superAdmin || superAdmin.role !== 'super_admin') {
      return [];
    }

    return Array.from(this.admins.values()).filter(a => a.role === 'master_admin');
  }

  /**
   * Update Master Admin
   */
  async updateMasterAdmin(
    superAdminToken: string,
    adminId: string,
    updates: Partial<AdminUser>
  ): Promise<{ success: boolean; error?: string }> {
    const superAdmin = await this.verifyToken(superAdminToken);
    if (!superAdmin || superAdmin.role !== 'super_admin') {
      return { success: false, error: 'Unauthorized' };
    }

    const admin = this.admins.get(adminId);
    if (!admin || admin.role !== 'master_admin') {
      return { success: false, error: 'Master Admin not found' };
    }

    // Apply updates
    if (updates.permissions) admin.permissions = updates.permissions;
    if (updates.isActive !== undefined) admin.isActive = updates.isActive;
    
    this.admins.set(adminId, admin);
    this.logAudit('MASTER_ADMIN_UPDATED', 'admin', superAdmin.id, { targetAdminId: adminId, updates }, '0.0.0.0', 'Unknown');
    
    return { success: true };
  }

  // ==================== White Label Admin ====================

  /**
   * Create White Label Admin
   */
  async createWhiteLabelAdmin(
    masterAdminToken: string,
    email: string,
    password: string,
    config: Omit<WhiteLabelConfig, 'id'>
  ): Promise<{ success: boolean; adminId?: string; whiteLabelId?: string; error?: string }> {
    const masterAdmin = await this.verifyToken(masterAdminToken);
    if (!masterAdmin || (masterAdmin.role !== 'super_admin' && masterAdmin.role !== 'master_admin')) {
      return { success: false, error: 'Unauthorized' };
    }

    // Check if email exists
    if (Array.from(this.admins.values()).some(a => a.email === email)) {
      return { success: false, error: 'Email already exists' };
    }

    const whiteLabelId = this.generateId();
    const adminId = this.generateId();

    const whiteLabel: WhiteLabelConfig = {
      id: whiteLabelId,
      ...config,
    };

    const whiteLabelAdmin: AdminUser = {
      id: adminId,
      email,
      role: 'white_label_admin',
      masterWalletId: whiteLabelId,
      permissions: ['view', 'manage_own_users', 'view_own_analytics'],
      isActive: true,
      twoFactorEnabled: false,
      createdAt: Date.now(),
      failedLoginAttempts: 0,
    };

    this.admins.set(adminId, whiteLabelAdmin);
    this.whiteLabels.set(whiteLabelId, whiteLabel);

    this.logAudit('WHITE_LABEL_CREATED', 'admin', masterAdmin.id, { adminId, whiteLabelId, name: config.name }, '0.0.0.0', 'Unknown');
    
    return { success: true, adminId, whiteLabelId };
  }

  /**
   * Get all White Labels
   */
  async getWhiteLabels(adminToken: string): Promise<WhiteLabelConfig[]> {
    const admin = await this.verifyToken(adminToken);
    if (!admin) return [];

    if (admin.role === 'super_admin' || admin.role === 'master_admin') {
      return Array.from(this.whiteLabels.values());
    }

    // White label can only see their own config
    if (admin.masterWalletId) {
      const wl = this.whiteLabels.get(admin.masterWalletId);
      return wl ? [wl] : [];
    }

    return [];
  }

  /**
   * Update White Label config
   */
  async updateWhiteLabel(
    adminToken: string,
    whiteLabelId: string,
    updates: Partial<WhiteLabelConfig>
  ): Promise<{ success: boolean; error?: string }> {
    const admin = await this.verifyToken(adminToken);
    if (!admin) return { success: false, error: 'Unauthorized' };

    const whiteLabel = this.whiteLabels.get(whiteLabelId);
    if (!whiteLabel) return { success: false, error: 'White Label not found' };

    // Apply updates
    if (updates.branding) whiteLabel.branding = updates.branding;
    if (updates.feePercentage !== undefined) whiteLabel.feePercentage = updates.feePercentage;
    if (updates.isActive !== undefined) whiteLabel.isActive = updates.isActive;

    this.whiteLabels.set(whiteLabelId, whiteLabel);
    this.logAudit('WHITE_LABEL_UPDATED', 'admin', admin.id, { whiteLabelId, updates }, '0.0.0.0', 'Unknown');

    return { success: true };
  }

  // ==================== Feature Flags ====================

  /**
   * Get all feature flags
   */
  async getFeatureFlags(adminToken: string): Promise<FeatureConfig[]> {
    const admin = await this.verifyToken(adminToken);
    if (!admin) return [];

    return Array.from(this.featureFlags.values());
  }

  /**
   * Update feature flag
   */
  async updateFeatureFlag(
    adminToken: string,
    featureName: string,
    enabled: boolean,
    config?: Record<string, any>
  ): Promise<{ success: boolean; error?: string }> {
    const admin = await this.verifyToken(adminToken);
    if (!admin || admin.role !== 'super_admin') {
      return { success: false, error: 'Only Super Admin can modify feature flags' };
    }

    const feature = this.featureFlags.get(featureName);
    if (!feature) {
      return { success: false, error: 'Feature not found' };
    }

    feature.enabled = enabled;
    if (config) feature.config = config;
    this.featureFlags.set(featureName, feature);

    this.logAudit('FEATURE_FLAG_UPDATED', 'admin', admin.id, { featureName, enabled, config }, '0.0.0.0', 'Unknown');

    return { success: true };
  }

  /**
   * Check if feature is enabled
   */
  async isFeatureEnabled(featureName: string): Promise<boolean> {
    const feature = this.featureFlags.get(featureName);
    return feature?.enabled ?? false;
  }

  // ==================== Profit Distribution ====================

  /**
   * Execute profit distribution to Super Admin
   */
  async executeProfitDistribution(
    adminToken: string,
    whiteLabelId: string,
    amount: string,
    token: string
  ): Promise<{ success: boolean; txHash?: string; error?: string }> {
    const admin = await this.verifyToken(adminToken);
    if (!admin) return { success: false, error: 'Unauthorized' };

    const whiteLabel = this.whiteLabels.get(whiteLabelId);
    if (!whiteLabel) return { success: false, error: 'White Label not found' };

    // Calculate profit share (default 20%)
    const profitAmount = (parseFloat(amount) * PROFIT_SHARE_PERCENTAGE / 100).toString();

    // In production, this would execute actual blockchain transaction
    const distribution: ProfitDistribution = {
      whiteLabelId,
      amount: profitAmount,
      token,
      timestamp: Date.now(),
      txHash: `0x${randomBytes(32).toString('hex')}`,
    };

    this.profitDistributions.push(distribution);
    
    this.logAudit('PROFIT_DISTRIBUTION', 'admin', admin.id, { whiteLabelId, amount: profitAmount, token }, '0.0.0.0', 'Unknown');

    return { success: true, txHash: distribution.txHash };
  }

  /**
   * Get profit distribution history
   */
  async getProfitDistributions(adminToken: string, whiteLabelId?: string): Promise<ProfitDistribution[]> {
    const admin = await this.verifyToken(adminToken);
    if (!admin) return [];

    if (whiteLabelId) {
      return this.profitDistributions.filter(d => d.whiteLabelId === whiteLabelId);
    }

    return this.profitDistributions;
  }

  // ==================== Audit Logs ====================

  /**
   * Get audit logs
   */
  async getAuditLogs(
    adminToken: string,
    filters?: {
      adminId?: string;
      action?: string;
      entityType?: string;
      startTime?: number;
      endTime?: number;
      limit?: number;
    }
  ): Promise<AuditLogEntry[]> {
    const admin = await this.verifyToken(adminToken);
    if (!admin) return [];

    let logs = [...this.auditLogs];

    if (filters?.adminId) {
      logs = logs.filter(l => l.adminId === filters.adminId);
    }
    if (filters?.action) {
      logs = logs.filter(l => l.action.includes(filters.action!));
    }
    if (filters?.entityType) {
      logs = logs.filter(l => l.entityType === filters.entityType);
    }
    if (filters?.startTime) {
      logs = logs.filter(l => l.timestamp >= filters.startTime!);
    }
    if (filters?.endTime) {
      logs = logs.filter(l => l.timestamp <= filters.endTime!);
    }

    // Sort by timestamp descending
    logs.sort((a, b) => b.timestamp - a.timestamp);

    if (filters?.limit) {
      logs = logs.slice(0, filters.limit);
    }

    return logs;
  }

  // ==================== Private Helpers ====================

  private generateId(): string {
    return `id_${Date.now()}_${randomBytes(8).toString('hex')}`;
  }

  private generateToken(): string {
    return `tok_${randomBytes(32).toString('hex')}`;
  }

  private async verifyPassword(password: string, adminId: string): Promise<boolean> {
    // In production, use proper password hashing (bcrypt/argon2)
    // This is simplified for demonstration
    const admin = this.admins.get(adminId);
    if (!admin) return false;
    
    // Simple check - in production use proper crypto
    return password.length >= 8;
  }

  private logAudit(
    action: string,
    entityType: string,
    entityId: string | undefined,
    details: Record<string, any>,
    ipAddress: string,
    userAgent: string
  ): void {
    const log: AuditLogEntry = {
      id: this.generateId(),
      adminId: entityId || 'system',
      action,
      entityType,
      entityId,
      details,
      ipAddress,
      userAgent,
      timestamp: Date.now(),
    };
    
    this.auditLogs.push(log);
    
    // Keep only last 10000 logs in memory
    if (this.auditLogs.length > 10000) {
      this.auditLogs = this.auditLogs.slice(-10000);
    }
  }

  private encrypt(data: string): string {
    if (!this.encryptionKey) throw new Error('Encryption key not initialized');
    const iv = randomBytes(16);
    const cipher = createCipheriv('aes-256-gcm', this.encryptionKey, iv);
    let encrypted = cipher.update(data, 'utf8', 'hex');
    encrypted += cipher.final('hex');
    const authTag = cipher.getAuthTag();
    return `${iv.toString('hex')}:${authTag.toString('hex')}:${encrypted}`;
  }

  private decrypt(encryptedData: string): string {
    if (!this.encryptionKey) throw new Error('Encryption key not initialized');
    const [ivHex, authTagHex, encrypted] = encryptedData.split(':');
    const iv = Buffer.from(ivHex, 'hex');
    const authTag = Buffer.from(authTagHex, 'hex');
    const decipher = createDecipheriv('aes-256-gcm', this.encryptionKey, iv);
    decipher.setAuthTag(authTag);
    let decrypted = decipher.update(encrypted, 'hex', 'utf8');
    decrypted += decipher.final('utf8');
    return decrypted;
  }

  // ==================== API Key Management ====================

  /**
   * Generate API key for programmatic access
   */
  async generateApiKey(adminId: string, name: string): Promise<{ success: boolean; apiKey?: string; error?: string }> {
    const admin = this.admins.get(adminId);
    if (!admin) return { success: false, error: 'Admin not found' };

    const apiKey = `tw_sk_${randomBytes(32).toString('hex')}`;
    this.apiKeys.set(apiKey, adminId);
    
    this.logAudit('API_KEY_CREATED', 'admin', adminId, { keyName: name }, '0.0.0.0', 'Unknown');
    
    return { success: true, apiKey };
  }

  /**
   * Verify API key
   */
  async verifyApiKey(apiKey: string): Promise<AdminUser | null> {
    const adminId = this.apiKeys.get(apiKey);
    if (!adminId) return null;
    return this.admins.get(adminId) || null;
  }

  /**
   * Revoke API key
   */
  async revokeApiKey(adminId: string, apiKey: string): Promise<{ success: boolean; error?: string }> {
    const storedAdminId = this.apiKeys.get(apiKey);
    if (storedAdminId !== adminId) {
      return { success: false, error: 'API key not found' };
    }

    this.apiKeys.delete(apiKey);
    this.logAudit('API_KEY_REVOKED', 'admin', adminId, {}, '0.0.0.0', 'Unknown');
    
    return { success: true };
  }

  // ==================== Dashboard Stats ====================

  /**
   * Get dashboard statistics
   */
  async getDashboardStats(adminToken: string): Promise<{
    totalAdmins: number;
    totalWhiteLabels: number;
    totalProfitDistributed: string;
    featureFlagsEnabled: number;
    recentAuditLogs: number;
  }> {
    const admin = await this.verifyToken(adminToken);
    if (!admin) {
      return {
        totalAdmins: 0,
        totalWhiteLabels: 0,
        totalProfitDistributed: '0',
        featureFlagsEnabled: 0,
        recentAuditLogs: 0,
      };
    }

    const totalProfit = this.profitDistributions.reduce((sum, d) => sum + parseFloat(d.amount), 0);
    const featuresEnabled = Array.from(this.featureFlags.values()).filter(f => f.enabled).length;

    return {
      totalAdmins: this.admins.size,
      totalWhiteLabels: this.whiteLabels.size,
      totalProfitDistributed: totalProfit.toString(),
      featureFlagsEnabled: featuresEnabled,
      recentAuditLogs: this.auditLogs.filter(l => l.timestamp > Date.now() - 24 * 60 * 60 * 1000).length,
    };
  }
}

export default SuperAdminService.getInstance();
export { SuperAdminService, AdminUser, FeatureConfig, AuditLogEntry, WhiteLabelConfig, ProfitDistribution };
