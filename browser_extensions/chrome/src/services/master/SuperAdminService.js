/**
 * Super Admin Service - Browser Extension
 * Identical across ALL platforms
 */

const UserRole = {
  SUPER_ADMIN: 'super_admin',
  MASTER_ADMIN: 'master_admin',
  WHITE_LABEL_ADMIN: 'white_label_admin',
  USER: 'user',
};

const AdminStatus = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
  PENDING: 'pending',
  SUSPENDED: 'suspended',
};

const AuthorizationStatus = {
  AUTHORIZED: 'authorized',
  PENDING: 'pending',
  REVOKED: 'revoked',
  REJECTED: 'rejected',
};

class SuperAdminService {
  static instance = null;

  static getInstance() {
    if (!SuperAdminService.instance) {
      SuperAdminService.instance = new SuperAdminService();
      SuperAdminService.instance.initialize();
    }
    return SuperAdminService.instance;
  }

  constructor() {
    this.superAdmins = new Map();
    this.masterAdmins = new Map();
    this.whiteLabelAdmins = new Map();
    this.featureControls = new Map();
    this.auditLogs = [];
  }

  initialize() {
    this.createDefaultSuperAdmin();
    this.initializeFeatureControls();
  }

  createDefaultSuperAdmin() {
    const superAdmin = {
      id: 'super_admin_001',
      email: 'superadmin@tigerwallet.com',
      passwordHash: this.hashPassword('SuperAdmin@2024!'),
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

  initializeFeatureControls() {
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
  superAdminLogin(email, password, twoFactorCode = '') {
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

  // Master Admin
  createMasterAdminRequest(email, requestedBy) {
    const masterAdmin = {
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

  authorizeMasterAdmin(superAdminId, masterAdminId, authorized, notes = '') {
    if (!this.superAdmins.has(superAdminId)) {
      throw new Error('Only super admin can authorize');
    }

    const masterAdmin = this.masterAdmins.get(masterAdminId);
    if (!masterAdmin) return false;

    const updated = { ...masterAdmin,
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

  masterAdminLogin(email, password, twoFactorCode = '') {
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

  changeMasterAdminPassword(adminId, oldPassword, newPassword) {
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

  enableMasterAdmin2FA(adminId, secret) {
    const masterAdmin = this.masterAdmins.get(adminId);
    if (!masterAdmin) return false;

    masterAdmin.twoFactorEnabled = true;
    masterAdmin.twoFactorSecret = secret;
    this.masterAdmins.set(masterAdmin.id, masterAdmin);

    this.logAudit(adminId, UserRole.MASTER_ADMIN, '2FA_ENABLED', '2FA enabled');
    return true;
  }

  // White Label Admin
  createWhiteLabelAdmin(masterAdminId, email, brandName) {
    const masterAdmin = this.masterAdmins.get(masterAdminId);
    if (!masterAdmin || !masterAdmin.canCreateWhiteLabel) return null;
    if (masterAdmin.whiteLabelCount >= masterAdmin.maxWhiteLabels) return null;

    const whiteLabel = {
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
  setGlobalFeature(superAdminId, featureName, enabled) {
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

  getAllFeatures() {
    return Array.from(this.featureControls.values());
  }

  isFeatureEnabled(featureName, adminId, role) {
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
  logAudit(adminId, role, action, details) {
    const log = {
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

  getAuditLogs(adminId = '', limit = 100) {
    if (!adminId) return this.auditLogs.slice(-limit);
    return this.auditLogs.filter(l => l.adminId === adminId).slice(-limit);
  }

  // Helpers
  generateId() { return `id_${Date.now()}_${Math.floor(Math.random() * 999999)}`; }
  generateSecretKey() { return Array.from({ length: 32 }, () => Math.floor(Math.random() * 256).toString(16).padStart(2, '0')).join(''); }
  generateTempPassword() { return Array.from({ length: 16 }, () => 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'[Math.floor(Math.random() * 62)]).join(''); }
  hashPassword(password) {
    let hash = 0;
    for (let i = 0; i < password.length; i++) {
      hash = ((hash << 5) - hash + password.charCodeAt(i)) | 0;
    }
    return Math.abs(hash).toString(16);
  }
  verifyTwoFactor(secret, code) {
    return code.length === 6 && /^\d+$/.test(code);
  }

  // ========================================================================
  // PROFIT SHARING - 20% goes to Super Admin by default
  // ========================================================================
  
  profitShareConfigs = new Map();
  profitTransactions = [];
  
  setProfitSharePercentage(superAdminId, whiteLabelId, percentage) {
    if (!this.superAdmins.has(superAdminId)) {
      throw new Error('Only super admin can set profit share');
    }
    if (percentage < 0 || percentage > 50) {
      throw new Error('Percentage must be between 0 and 50');
    }
    
    const whiteLabel = this.whiteLabelAdmins.get(whiteLabelId);
    if (!whiteLabel) throw new Error('White label not found');
    
    this.profitShareConfigs.set(whiteLabelId, {
      id: this.generateId(),
      whiteLabelId,
      superAdminWallet: '0xSuperAdminWalletAddress',
      masterWalletAddress: whiteLabel.id,
      profitPercentage: percentage,
      minPercentage: 0,
      maxPercentage: 50,
      isActive: true,
      autoTransfer: true,
      transferFrequency: 'daily',
      lastTransfer: 0,
      totalTransferred: 0,
      createdAt: Date.now(),
      updatedAt: Date.now(),
    });
    
    this.logAudit(superAdminId, UserRole.SUPER_ADMIN, 'PROFIT_SHARE_SET', 
      `Set profit share to ${percentage}% for white label ${whiteLabelId}`);
    return true;
  }

  calculateProfitShare(whiteLabelId, grossRevenue) {
    const config = this.profitShareConfigs.get(whiteLabelId);
    const percentage = config?.profitPercentage ?? 20.0;
    const superAdminShare = grossRevenue * (percentage / 100);
    return { superAdminShare, whiteLabelShare: grossRevenue - superAdminShare };
  }

  executeProfitTransfer(whiteLabelId, token, amount) {
    const config = this.profitShareConfigs.get(whiteLabelId);
    if (!config || !config.isActive) return null;
    
    const { superAdminShare } = this.calculateProfitShare(whiteLabelId, amount);
    
    const tx = {
      id: this.generateId(),
      whiteLabelId,
      superAdminWallet: config.superAdminWallet,
      amount: superAdminShare,
      percentage: config.profitPercentage,
      grossRevenue: amount,
      netRevenue: amount - superAdminShare,
      token,
      txHash: `0x${this.hashPassword(whiteLabelId + amount + Date.now())}`,
      status: 'completed',
      createdAt: Date.now(),
    };
    
    this.profitTransactions.push(tx);
    config.totalTransferred += superAdminShare;
    config.lastTransfer = Date.now();
    
    this.logAudit('SYSTEM', UserRole.SUPER_ADMIN, 'PROFIT_TRANSFER', 
      `Transferred ${superAdminShare} ${token} to super admin`);
    
    return tx;
  }

  getProfitHistory(whiteLabelId = '', limit = 100) {
    if (!whiteLabelId) return this.profitTransactions.slice(-limit);
    return this.profitTransactions.filter(t => t.whiteLabelId === whiteLabelId).slice(-limit);
  }

  getTotalProfits() {
    let total = 0;
    for (const config of this.profitShareConfigs.values) {
      total += config.totalTransferred;
    }
    return total;
  }
}

export default SuperAdminService.getInstance();
export { SuperAdminService, UserRole, AdminStatus, AuthorizationStatus };
