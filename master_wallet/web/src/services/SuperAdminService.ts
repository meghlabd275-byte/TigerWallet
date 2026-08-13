/**
 * SuperAdminService - Web (React/TypeScript)
 *
 * NOTE: Super-admin / white-label / feature-toggle administration is NOT part
 * of the canonical MasterWallet backend contract (port 8450). This module
 * exposes the public typing surface only; operations that previously returned
 * fabricated admin data now return a descriptive error instead of fake data.
 */

export interface AdminUser {
  id: string;
  email: string;
  name: string;
  role: 'super_admin' | 'white_label_admin' | 'operator';
  isActive: boolean;
  createdAt: string;
}

export interface FeatureConfig {
  key: string;
  enabled: boolean;
  description: string;
}

export interface AuditLogEntry {
  id: string;
  actor: string;
  action: string;
  target: string;
  timestamp: string;
}

export interface WhiteLabelConfig {
  id: string;
  name: string;
  domain: string;
  branding: Record<string, string>;
}

export interface ProfitDistribution {
  recipient: string;
  amount: string;
  percentage: number;
}

class SuperAdminServiceClass {
  private static instance: SuperAdminServiceClass | null = null;
  private constructor() {}
  static getInstance(): SuperAdminServiceClass {
    if (!SuperAdminServiceClass.instance) SuperAdminServiceClass.instance = new SuperAdminServiceClass();
    return SuperAdminServiceClass.instance;
  }

  getAdmins(): AdminUser[] {
    return [];
  }

  createAdmin(_email: string, _name: string, _role: AdminUser['role']): { success: false; error: string } {
    return { success: false, error: 'Admin management is not supported by the canonical MasterWallet backend' };
  }

  getFeatureToggles(): FeatureConfig[] {
    return [];
  }

  setFeatureToggle(_key: string, _enabled: boolean): { success: false; error: string } {
    return { success: false, error: 'Feature toggles are not supported by the canonical MasterWallet backend' };
  }

  getAuditLogs(): AuditLogEntry[] {
    return [];
  }

  getWhiteLabelConfigs(): WhiteLabelConfig[] {
    return [];
  }

  distributeProfits(_distributions: ProfitDistribution[]): { success: false; error: string } {
    return { success: false, error: 'Profit distribution is not supported by the canonical MasterWallet backend' };
  }
}

export const SuperAdminService = SuperAdminServiceClass;
export default SuperAdminServiceClass.getInstance();
