/**
 * Super Admin Service - React/Web Implementation
 * Identical across ALL platforms
 */

export enum UserRole {
  SUPER_ADMIN = 'super_admin',
  MASTER_ADMIN = 'master_admin',
  WHITE_LABEL_ADMIN = 'white_label_admin',
  USER = 'user',
}

export enum AdminStatus {
  ACTIVE = 'active',
  INACTIVE = 'inactive',
  PENDING = 'pending',
  SUSPENDED = 'suspended',
}

export enum AuthorizationStatus {
  AUTHORIZED = 'authorized',
  PENDING = 'pending',
  REVOKED = 'revoked',
  REJECTED = 'rejected',
}

export interface SuperAdmin {
  id: string;
  email: string;
  passwordHash: string;
  secretKey: string;
  twoFactorEnabled: boolean;
  twoFactorSecret: string;
  phone: string;
  createdAt: number;
  lastLogin: number;
  isActive: boolean;
  permissions: string[];
}

export interface MasterAdmin {
  id: string;
  email: string;
  passwordHash: string;
  authorizedBy: string;
  authorizationStatus: AuthorizationStatus;
  twoFactorEnabled: boolean;
  twoFactorSecret: string;
  phone: string;
  canCreateWhiteLabel: boolean;
  canManageUsers: boolean;
  canManageWallets: boolean;
  canAccessFinance: boolean;
  canModifyFeatures: boolean;
  canManageTokens: boolean;
  canManageNetworks: boolean;
  canViewAnalytics: boolean;
  canManageAdmins: boolean;
  maxWhiteLabels: number;
  whiteLabelCount: number;
  status: AdminStatus;
  createdAt: number;
  lastLogin: number;
  passwordChangedAt: number;
  failedAttempts: number;
  lockedUntil: number;
}

export interface WhiteLabelAdmin {
  id: string;
  email: string;
  passwordHash: string;
  masterAdminId: string;
  brandName: string;
  brandLogo: string;
  brandColor: string;
  customDomain: string;
  authorizationStatus: AuthorizationStatus;
  twoFactorEnabled: boolean;
  twoFactorSecret: string;
  canCustomizeUi: boolean;
  canCustomizeFees: boolean;
  canManageUsers: boolean;
  canManageWallets: boolean;
  canAccessAnalytics: boolean;
  canManageTokens: boolean;
  feePercentage: number;
  status: AdminStatus;
  createdAt: number;
  lastLogin: number;
}

export interface FeatureControl {
  featureName: string;
  enabled: boolean;
  globalEnabled: boolean;
  masterAdminId: string;
  whiteLabelId: string;
  updatedBy: string;
  updatedAt: number;
}

export interface AuditLog {
  id: string;
  adminId: string;
  adminRole: UserRole;
  action: string;
  details: string;
  ipAddress: string;
  userAgent: string;
  timestamp: number;
}

// PROFIT SHARING
export interface ProfitShareConfig {
  id: string;
  whiteLabelId: string;
  superAdminWallet: string;
  masterWalletAddress: string;
  profitPercentage: number;
  minPercentage: number;
  maxPercentage: number;
  isActive: boolean;
  autoTransfer: boolean;
  transferFrequency: string;
  lastTransfer: number;
  totalTransferred: number;
  createdAt: number;
  updatedAt: number;
}

export interface ProfitTransaction {
  id: string;
  whiteLabelId: string;
  superAdminWallet: string;
  amount: number;
  percentage: number;
  grossRevenue: number;
  netRevenue: number;
  token: string;
  txHash: string;
  status: string;
  createdAt: number;
}

class SuperAdminServiceClass {
  private static instance: SuperAdminServiceClass;
  private superAdmins: Map<string, SuperAdmin> = new Map();
  private masterAdmins: Map<string, MasterAdmin> = new Map();
  private whiteLabelAdmins: Map<string, WhiteLabelAdmin> = new Map();
  private featureControls: Map<string, FeatureControl> = new Map();
  private auditLogs: AuditLog[] = [];
  // PROFIT SHARING
  private profitShareConfigs: Map<string, ProfitShareConfig> = new Map();
  private profitTransactions: ProfitTransaction[] = [];

  static getInstance(): SuperAdminServiceClass {
    if (!SuperAdminServiceClass.instance) {
      SuperAdminServiceClass.instance = new SuperAdminServiceClass();
      SuperAdminServiceClass.instance.initialize();
    }
    return SuperAdminServiceClass.instance;
  }

  private initialize(): void {
    this.createDefaultSuperAdmin();
    this.initializeFeatureControls();
  }

  private createDefaultSuperAdmin(): void {
    const superAdmin: SuperAdmin = {
      id: 'super_admin_001',
      email: 'superadmin@tigerwallet.com',
      passwordHash: this.hashPassword(process.env.REACT_APP_SUPER_ADMIN_PASSWORD || ''),
      secretKey: this.generateSecretKey(),
      twoFactorEnabled: false,
      twoFactorSecret: '',
      phone: '',
      createdAt: Date.now(),
      lastLogin: 0,
      isActive: true,
      permissions: ['*'],
    };
    this.superAdmins.set(superAdmin.id, superAdmin);
    this.superAdmins.set(superAdmin.email, superAdmin);
  }

  private initializeFeatureControls(): void {
    const features = [
      'master_wallet_creation', 'multi_blockchain', 'token_management',
      'user_wallet_ownership', 'hd_wallet', 'biometric_auth',
      'pin_code_auth', 'nft_support', 'defi_integration', 'staking',
      'bridge_support', 'mev_protection', 'swap_trading', 'hardware_wallet',
      'admin_controls', 'network_management', 'gas_optimization', 'multi_sig',
      'transaction_history', 'price_alerts', 'privacy_zk', 'coinjoin',
      'account_abstraction', 'session_keys', 'paymaster', 'passkeys',
      'tax_integration', 'analytics', 'cross_chain_intent', 'dapp_browser',
    ];
    features.forEach(feature => {
      this.featureControls.set(feature, {
        featureName: feature,
        enabled: true,
        globalEnabled: true,
        masterAdminId: '',
        whiteLabelId: '',
        updatedBy: '',
        updatedAt: Date.now(),
      });
    });
  }

  // Super Admin Login
  superAdminLogin(email: string, password: string, twoFactorCode: string = ''): SuperAdmin | null {
    const superAdmin = this.superAdmins.get(email);
    if (!superAdmin || !superAdmin.isActive) return null;

    if (this.hashPassword(password) !== superAdmin.passwordHash) {
      this.logAudit(superAdmin.id, UserRole.SUPER_ADMIN, 'LOGIN_FAILED', 'Invalid password');
      return null;
    }

    if (superAdmin.twoFactorEnabled && !this.verifyTwoFactor(superAdmin.twoFactorSecret, twoFactorCode)) {
      return null;
    }

    this.logAudit(superAdmin.id, UserRole.SUPER_ADMIN, 'LOGIN_SUCCESS', 'Super admin logged in');
    return superAdmin;
  }

  // Master Admin Operations
  createMasterAdminRequest(email: string, requestedBy: string): MasterAdmin {
    const masterAdmin: MasterAdmin = {
      id: this.generateId(),
      email,
      passwordHash: this.hashPassword(this.generateTempPassword()),
      authorizedBy: '',
      authorizationStatus: AuthorizationStatus.PENDING,
      twoFactorEnabled: false,
      twoFactorSecret: '',
      phone: '',
      canCreateWhiteLabel: false,
      canManageUsers: false,
      canManageWallets: false,
      canAccessFinance: false,
      canModifyFeatures: false,
      canManageTokens: false,
      canManageNetworks: false,
      canViewAnalytics: false,
      canManageAdmins: false,
      maxWhiteLabels: 0,
      whiteLabelCount: 0,
      status: AdminStatus.PENDING,
      createdAt: Date.now(),
      lastLogin: 0,
      passwordChangedAt: 0,
      failedAttempts: 0,
      lockedUntil: 0,
    };
    this.masterAdmins.set(masterAdmin.id, masterAdmin);
    this.masterAdmins.set(email, masterAdmin);
    this.logAudit('SYSTEM', UserRole.SUPER_ADMIN, 'MASTER_ADMIN_REQUEST', `New request: ${email}`);
    return masterAdmin;
  }

  authorizeMasterAdmin(superAdminId: string, masterAdminId: string, authorized: boolean, notes: string = ''): boolean {
    if (!this.superAdmins.has(superAdminId)) {
      throw new Error('Only super admin can authorize');
    }

    const masterAdmin = this.masterAdmins.get(masterAdminId);
    if (!masterAdmin) return false;

    const updated: MasterAdmin = {
      ...masterAdmin,
      authorizedBy: superAdminId,
      authorizationStatus: authorized ? AuthorizationStatus.AUTHORIZED : AuthorizationStatus.REJECTED,
      status: authorized ? AdminStatus.ACTIVE : AdminStatus.INACTIVE,
    };

    this.masterAdmins.set(masterAdmin.id, updated);
    this.masterAdmins.set(updated.email, updated);

    const action = authorized ? 'AUTHORIZED' : 'REJECTED';
    this.logAudit(superAdminId, UserRole.SUPER_ADMIN, `MASTER_ADMIN_${action}`, `${action} ${masterAdmin.email}`);
    return true;
  }

  masterAdminLogin(email: string, password: string, twoFactorCode: string = ''): MasterAdmin | null {
    const masterAdmin = this.masterAdmins.get(email);
    if (!masterAdmin) return null;

    if (masterAdmin.authorizationStatus !== AuthorizationStatus.AUTHORIZED) return null;
    if (masterAdmin.status !== AdminStatus.ACTIVE) return null;
    if (masterAdmin.lockedUntil > Date.now()) return null;

    if (this.hashPassword(password) !== masterAdmin.passwordHash) {
      this.logAudit(masterAdmin.id, UserRole.MASTER_ADMIN, 'LOGIN_FAILED', 'Invalid password');
      return null;
    }

    if (masterAdmin.twoFactorEnabled && !this.verifyTwoFactor(masterAdmin.twoFactorSecret, twoFactorCode)) {
      return null;
    }

    this.logAudit(masterAdmin.id, UserRole.MASTER_ADMIN, 'LOGIN_SUCCESS', 'Master admin logged in');
    return masterAdmin;
  }

  changeMasterAdminPassword(adminId: string, oldPassword: string, newPassword: string): boolean {
    const masterAdmin = this.masterAdmins.get(adminId) || Array.from(this.masterAdmins.values()).find(m => m.email === adminId);
    if (!masterAdmin) return false;

    if (this.hashPassword(oldPassword) !== masterAdmin.passwordHash) return false;
    if (newPassword.length < 8) return false;

    masterAdmin.passwordHash = this.hashPassword(newPassword);
    masterAdmin.passwordChangedAt = Date.now();
    this.masterAdmins.set(masterAdmin.id, masterAdmin);

    this.logAudit(adminId, UserRole.MASTER_ADMIN, 'PASSWORD_CHANGED', 'Password changed');
    return true;
  }

  enableMasterAdmin2FA(adminId: string, secret: string): boolean {
    const masterAdmin = this.masterAdmins.get(adminId);
    if (!masterAdmin) return false;

    masterAdmin.twoFactorEnabled = true;
    masterAdmin.twoFactorSecret = secret;
    this.masterAdmins.set(masterAdmin.id, masterAdmin);

    this.logAudit(adminId, UserRole.MASTER_ADMIN, '2FA_ENABLED', '2FA enabled');
    return true;
  }

  // White Label Admin
  createWhiteLabelAdmin(masterAdminId: string, email: string, brandName: string): WhiteLabelAdmin | null {
    const masterAdmin = this.masterAdmins.get(masterAdminId);
    if (!masterAdmin || !masterAdmin.canCreateWhiteLabel) return null;
    if (masterAdmin.whiteLabelCount >= masterAdmin.maxWhiteLabels) return null;

    const whiteLabel: WhiteLabelAdmin = {
      id: this.generateId(),
      email,
      passwordHash: this.hashPassword(this.generateTempPassword()),
      masterAdminId,
      brandName,
      brandLogo: '',
      brandColor: '#000000',
      customDomain: '',
      authorizationStatus: AuthorizationStatus.AUTHORIZED,
      twoFactorEnabled: false,
      twoFactorSecret: '',
      canCustomizeUi: true,
      canCustomizeFees: true,
      canManageUsers: true,
      canManageWallets: true,
      canAccessAnalytics: true,
      canManageTokens: true,
      feePercentage: 0,
      status: AdminStatus.ACTIVE,
      createdAt: Date.now(),
      lastLogin: 0,
    };

    this.whiteLabelAdmins.set(whiteLabel.id, whiteLabel);
    this.whiteLabelAdmins.set(email, whiteLabel);

    this.logAudit(masterAdminId, UserRole.MASTER_ADMIN, 'WHITE_LABEL_CREATED', `Created: ${email} - ${brandName}`);
    return whiteLabel;
  }

  // Feature Control
  setGlobalFeature(superAdminId: string, featureName: string, enabled: boolean): boolean {
    if (!this.superAdmins.has(superAdminId)) {
      throw new Error('Only super admin can modify features');
    }

    const feature = this.featureControls.get(featureName);
    if (!feature) return false;

    feature.enabled = enabled;
    feature.globalEnabled = enabled;
    feature.updatedBy = superAdminId;
    feature.updatedAt = Date.now();

    this.logAudit(superAdminId, UserRole.SUPER_ADMIN, 'FEATURE_TOGGLE', `Set ${featureName} = ${enabled}`);
    return true;
  }

  getAllFeatures(): FeatureControl[] {
    return Array.from(this.featureControls.values());
  }

  isFeatureEnabled(featureName: string, adminId: string, role: UserRole): boolean {
    const feature = this.featureControls.get(featureName);
    if (!feature || !feature.globalEnabled) return false;

    switch (role) {
      case UserRole.SUPER_ADMIN:
        return true;
      case UserRole.MASTER_ADMIN:
        if (feature.masterAdminId && feature.masterAdminId !== adminId) return false;
        return feature.enabled;
      case UserRole.WHITE_LABEL_ADMIN:
        if (feature.whiteLabelId && feature.whiteLabelId !== adminId) return false;
        return feature.enabled;
      default:
        return false;
    }
  }

  // Audit
  private logAudit(adminId: string, role: UserRole, action: string, details: string): void {
    const log: AuditLog = {
      id: this.generateId(),
      adminId,
      adminRole: role,
      action,
      details,
      ipAddress: '',
      userAgent: '',
      timestamp: Date.now(),
    };
    this.auditLogs.push(log);
    console.log(`[AUDIT] ${role} | ${action} | ${details}`);
  }

  getAuditLogs(adminId: string = '', limit: number = 100): AuditLog[] {
    if (!adminId) return this.auditLogs.slice(-limit);
    return this.auditLogs.filter(l => l.adminId === adminId).slice(-limit);
  }

  // Helpers — use the Web Crypto API (CSPRNG), never Math.random.
  private generateId(): string { return `id_${Date.now()}_${this.randomHex(4)}`; }
  private generateSecretKey(): string { return this.randomHex(32); }
  private generateTempPassword(): string {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
    const bytes = new Uint32Array(16);
    crypto.getRandomValues(bytes);
    return Array.from(bytes, (b) => chars[b % chars.length]).join('');
  }
  private randomHex(byteLength: number): string {
    const bytes = new Uint8Array(byteLength);
    crypto.getRandomValues(bytes);
    return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
  }
  private hashPassword(password: string): string {
    let hash = 0;
    for (let i = 0; i < password.length; i++) {
      hash = ((hash << 5) - hash + password.charCodeAt(i)) | 0;
    }
    return Math.abs(hash).toString(16);
  }
  private verifyTwoFactor(secret: string, code: string): boolean {
    return code.length === 6 && /^\d+$/.test(code);
  }
}

export const SuperAdminService = SuperAdminServiceClass.getInstance();
export default SuperAdminService;
