// TigerWallet MasterWallet - Super Admin Service (Chrome Extension)
// Admin controls for MasterWallet
// Production-ready

class MasterSuperAdminService {
  constructor(masterWalletId) {
    this.masterWalletId = masterWalletId;
    this.adminId = null;
    this.role = null;
    this.isAuthenticated = false;
    this.isInitialized = false;
    this.featureFlags = new Map();
  }

  async initialize() {
    if (this.isInitialized) return true;
    
    try {
      // Load admin session
      await this.loadSession();
      
      // Load feature flags
      await this.loadFeatureFlags();
      
      this.isInitialized = true;
      return true;
    } catch (error) {
      console.error('SuperAdminService initialization failed:', error);
      return false;
    }
  }

  async loadSession() {
    const result = await chrome.storage.local.get(['adminId', 'adminRole', 'isAuthenticated']);
    if (result.isAuthenticated) {
      this.adminId = result.adminId;
      this.role = result.adminRole;
      this.isAuthenticated = true;
    }
  }

  async loadFeatureFlags() {
    const result = await chrome.storage.local.get('featureFlags');
    if (result.featureFlags) {
      this.featureFlags = new Map(Object.entries(result.featureFlags));
    }
    
    // Set defaults if none exist
    if (this.featureFlags.size === 0) {
      const defaults = {
        master_wallet_creation: true,
        multi_blockchain: true,
        token_management: true,
        user_wallet_ownership: true,
        hd_wallet: true,
        biometric_auth: true,
        pin_code_auth: true,
        nft_support: true,
        defi_integration: true,
        staking: true,
        bridge_support: true,
        mev_protection: true,
        swap_trading: true,
        hardware_wallet: true,
        admin_controls: true,
        network_management: true,
        gas_optimization: true,
        multi_sig: true,
        transaction_history: true,
        price_alerts: true,
        privacy_zk: true,
        coinjoin: true,
        account_abstraction: true,
        session_keys: true,
        paymaster: true,
        passkeys: true,
        tax_integration: true,
        analytics: true,
        cross_chain_intent: true,
        dapp_browser: true,
      };
      
      for (const [key, value] of Object.entries(defaults)) {
        this.featureFlags.set(key, { enabled: value, updatedAt: Date.now() });
      }
      
      await this.saveFeatureFlags();
    }
  }

  // Authentication
  async authenticate(email, password) {
    const response = await fetch('/api/super-admin/authenticate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });
    
    if (!response.ok) {
      throw new Error('Authentication failed');
    }
    
    const data = await response.json();
    
    this.adminId = data.adminId;
    this.role = data.role;
    this.isAuthenticated = true;
    
    // Save session
    await chrome.storage.local.set({
      adminId: this.adminId,
      adminRole: this.role,
      isAuthenticated: true,
    });
    
    // Log audit
    await this.logAudit('LOGIN', 'session', null, true);
    
    return data;
  }

  async logout() {
    // Log audit
    await this.logAudit('LOGOUT', 'session', this.adminId, true);
    
    // Clear session
    this.adminId = null;
    this.role = null;
    this.isAuthenticated = false;
    
    await chrome.storage.local.remove(['adminId', 'adminRole', 'isAuthenticated']);
    
    return true;
  }

  // Password management
  async changePassword(oldPassword, newPassword) {
    if (!this.isAuthenticated) {
      throw new Error('Not authenticated');
    }
    
    const response = await fetch('/api/super-admin/change-password', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        adminId: this.adminId,
        oldPassword,
        newPassword,
      }),
    });
    
    if (!response.ok) {
      throw new Error('Password change failed');
    }
    
    await this.logAudit('PASSWORD_CHANGED', 'admin', this.adminId, true);
    
    return true;
  }

  // Two-factor authentication
  async enable2FA() {
    if (!this.isAuthenticated) {
      throw new Error('Not authenticated');
    }
    
    const response = await fetch('/api/super-admin/enable-2fa', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ adminId: this.adminId }),
    });
    
    const data = await response.json();
    return data.secret;
  }

  async verify2FA(code) {
    if (!this.isAuthenticated) {
      throw new Error('Not authenticated');
    }
    
    const response = await fetch('/api/super-admin/verify-2fa', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ adminId: this.adminId, code }),
    });
    
    if (!response.ok) {
      throw new Error('Invalid code');
    }
    
    await this.logAudit('2FA_ENABLED', 'admin', this.adminId, true);
    
    return true;
  }

  async disable2FA(code) {
    if (!this.isAuthenticated) {
      throw new Error('Not authenticated');
    }
    
    const response = await fetch('/api/super-admin/disable-2fa', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ adminId: this.adminId, code }),
    });
    
    if (!response.ok) {
      throw new Error('Invalid code');
    }
    
    await this.logAudit('2FA_DISABLED', 'admin', this.adminId, true);
    
    return true;
  }

  // Feature flags
  async setFeatureFlag(name, enabled) {
    if (!this.isAuthenticated || this.role !== 'SUPER_ADMIN') {
      throw new Error('Not authorized');
    }
    
    this.featureFlags.set(name, {
      enabled,
      updatedAt: Date.now(),
      updatedBy: this.adminId,
    });
    
    await this.saveFeatureFlags();
    
    await this.logAudit('FEATURE_UPDATED', 'feature', name, true);
    
    return true;
  }

  async getFeatureFlag(name) {
    return this.featureFlags.get(name);
  }

  async listFeatureFlags() {
    const flags = {};
    this.featureFlags.forEach((value, key) => {
      flags[key] = value;
    });
    return flags;
  }

  async isFeatureEnabled(name) {
    const flag = this.featureFlags.get(name);
    return flag ? flag.enabled : false;
  }

  async saveFeatureFlags() {
    const flags = {};
    this.featureFlags.forEach((value, key) => {
      flags[key] = value;
    });
    
    await chrome.storage.local.set({ featureFlags: flags });
  }

  // Admin management (Super Admin only)
  async createAdmin(adminData) {
    if (!this.isAuthenticated || this.role !== 'SUPER_ADMIN') {
      throw new Error('Not authorized');
    }
    
    const response = await fetch('/api/super-admin/create-admin', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(adminData),
    });
    
    if (!response.ok) {
      throw new Error('Failed to create admin');
    }
    
    const data = await response.json();
    
    await this.logAudit('ADMIN_CREATED', 'admin', data.adminId, true);
    
    return data.adminId;
  }

  async listAdmins(role = null) {
    if (!this.isAuthenticated) {
      throw new Error('Not authenticated');
    }
    
    const url = role ? `/api/super-admin/admins?role=${role}` : '/api/super-admin/admins';
    const response = await fetch(url);
    const data = await response.json();
    
    return data.admins;
  }

  async deactivateAdmin(adminId) {
    if (!this.isAuthenticated || this.role !== 'SUPER_ADMIN') {
      throw new Error('Not authorized');
    }
    
    const response = await fetch(`/api/super-admin/admin/${adminId}/deactivate`, {
      method: 'POST',
    });
    
    if (!response.ok) {
      throw new Error('Failed to deactivate admin');
    }
    
    await this.logAudit('ADMIN_DEACTIVATED', 'admin', adminId, true);
    
    return true;
  }

  // Authorization requests
  async authorizeMasterAdmin(masterWalletId) {
    if (!this.isAuthenticated || this.role !== 'SUPER_ADMIN') {
      throw new Error('Not authorized');
    }
    
    const response = await fetch('/api/super-admin/authorize-master', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ masterWalletId }),
    });
    
    if (!response.ok) {
      throw new Error('Failed to authorize');
    }
    
    await this.logAudit('MASTER_AUTHORIZED', 'master_wallet', masterWalletId, true);
    
    return true;
  }

  // Audit logging
  async logAudit(action, resourceType, resourceId, success) {
    const entry = {
      adminId: this.adminId,
      action,
      resourceType,
      resourceId,
      success,
      timestamp: Date.now(),
      ipAddress: 'chrome_extension',
    };
    
    // Store locally
    const result = await chrome.storage.local.get('auditLogs');
    const logs = result.auditLogs || [];
    logs.push(entry);
    
    // Keep last 1000 entries
    if (logs.length > 1000) {
      logs.splice(0, logs.length - 1000);
    }
    
    await chrome.storage.local.set({ auditLogs: logs });
    
    // Send to backend
    try {
      await fetch('/api/super-admin/audit', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(entry),
      });
    } catch (error) {
      console.error('Failed to send audit log:', error);
    }
  }

  async getAuditLogs(filters = {}) {
    const result = await chrome.storage.local.get('auditLogs');
    let logs = result.auditLogs || [];
    
    // Apply filters
    if (filters.adminId) {
      logs = logs.filter(l => l.adminId === filters.adminId);
    }
    if (filters.action) {
      logs = logs.filter(l => l.action === filters.action);
    }
    if (filters.startTime) {
      logs = logs.filter(l => l.timestamp >= filters.startTime);
    }
    if (filters.endTime) {
      logs = logs.filter(l => l.timestamp <= filters.endTime);
    }
    
    // Sort by timestamp descending
    logs.sort((a, b) => b.timestamp - a.timestamp);
    
    // Apply limit
    if (filters.limit) {
      logs = logs.slice(0, filters.limit);
    }
    
    return logs;
  }

  // Statistics
  async getStats() {
    const result = await chrome.storage.local.get('auditLogs');
    const logs = result.auditLogs || [];
    
    return {
      totalAdmins: await this.listAdmins(),
      recentAuditLogs: logs.length,
      failedLogins: logs.filter(l => l.action === 'LOGIN_FAILED').length,
    };
  }
}

// Export
if (typeof module !== 'undefined' && module.exports) {
  module.exports = MasterSuperAdminService;
}
