// TigerAdmin - Desktop Admin Application
// Complete implementation with API connection and light/dark theme

import React, { useState, useEffect } from 'react';
import { DomainPage, DOMAIN_PAGES } from './DomainPage';

const API_BASE_URL = process.env.REACT_APP_ADMIN_API || 'http://localhost:9093';

// API Service
class DesktopAdminAPI {
  private baseURL: string;
  private token: string | null = null;

  constructor() {
    this.baseURL = API_BASE_URL;
    this.token = localStorage.getItem('admin_token');
  }

  setToken(token: string) {
    this.token = token;
    localStorage.setItem('admin_token', token);
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...(this.token && { 'Authorization': `Bearer ${this.token}` }),
      ...options.headers,
    };

    const response = await fetch(url, { ...options, headers });
    if (!response.ok) throw new Error(`API Error: ${response.status}`);
    return response.json();
  }

  async getAnalytics() { return this.request('/api/v1/analytics'); }
  async getUsers(params?: any) { 
    const query = params ? '?' + new URLSearchParams(params).toString() : '';
    return this.request(`/api/v1/users${query}`);
  }
  async getTransactions(params?: any) { 
    const query = params ? '?' + new URLSearchParams(params).toString() : '';
    return this.request(`/api/v1/transactions${query}`);
  }
  async getSystemStatus() { return this.request('/api/v1/system/status'); }
  async getKycRecords(params?: any) {
    const query = params ? '?' + new URLSearchParams(params).toString() : '';
    return this.request(`/api/v1/kyc${query}`);
  }
  async approveKyc(id: string) { 
    return this.request(`/api/v1/kyc/${id}/approve`, { method: 'POST' });
  }
  async rejectKyc(id: string, reason: string) { 
    return this.request(`/api/v1/kyc/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) });
  }
  async getTokens() { return this.request('/api/v1/tokens'); }
  async getWithdrawals() { return this.request('/api/v1/withdrawals'); }
  async approveWithdrawal(id: string) { 
    return this.request(`/api/v1/withdrawals/${id}/approve`, { method: 'POST' });
  }
  async getFeeConfig() { return this.request('/api/v1/fees'); }
  async updateFeeConfig(config: any) {
    return this.request('/api/v1/fees', { method: 'PUT', body: JSON.stringify(config) });
  }
  // ---- 12 domain endpoints (admin/go backend, /api/v1/<domain>) ----
  async listDomain(domain: string) { return this.request(`/api/v1/${domain}`); }
  async createDomain(domain: string, body: any) {
    return this.request(`/api/v1/${domain}`, { method: 'POST', body: JSON.stringify(body) });
  }
  async updateDomain(domain: string, id: string, body: any) {
    return this.request(`/api/v1/${domain}/${id}`, { method: 'PUT', body: JSON.stringify(body) });
  }
  async deleteDomain(domain: string, id: string) {
    return this.request(`/api/v1/${domain}/${id}`, { method: 'DELETE' });
  }
  async setDomainStatus(domain: string, id: string, status: string) {
    return this.request(`/api/v1/${domain}/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }
  async approveDomain(domain: string, id: string) {
    return this.request(`/api/v1/${domain}/${id}/approve`, { method: 'POST' });
  }
  async rejectDomain(domain: string, id: string, reason: string) {
    return this.request(`/api/v1/${domain}/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) });
  }
  // RBAC
  async assignRole(adminId: string, roleId: string) {
    return this.request(`/api/v1/admins/${adminId}/roles`, { method: 'POST', body: JSON.stringify({ role_id: roleId }) });
  }
  async revokeRole(adminId: string, roleId: string) {
    return this.request(`/api/v1/admins/${adminId}/roles/${roleId}`, { method: 'DELETE' });
  }
  async getAdminPermissions(adminId: string) {
    return this.request(`/api/v1/admins/${adminId}/permissions`);
  }
  async getAdminRoles(adminId: string) {
    return this.request(`/api/v1/admins/${adminId}/roles`);
  }
  // p2p-merchants exposes a transactions sub-resource (no delete/status on backend).
  async listP2PMerchantTransactions(id: string) {
    return this.request(`/api/v1/p2p-merchants/${id}/transactions`);
  }

  // ---- new admin domains: bots, bots-clients, project-teams, liquidity-sources ----
  // Each domain reuses the generic CRUD helpers (listDomain/createDomain/...).
  // Domain-specific sub-resources are exposed below.

  // bots: stats + tiers sub-resource
  async getBotStats() { return this.request('/api/v1/bots/stats'); }
  async getBotTiers(botId: string) { return this.request(`/api/v1/bots/${botId}/tiers`); }
  async createBotTier(botId: string, body: any) {
    return this.request(`/api/v1/bots/${botId}/tiers`, { method: 'POST', body: JSON.stringify(body) });
  }
  async updateBotTier(botId: string, tierId: string, body: any) {
    return this.request(`/api/v1/bots/${botId}/tiers/${tierId}`, { method: 'PUT', body: JSON.stringify(body) });
  }
  async deleteBotTier(botId: string, tierId: string) {
    return this.request(`/api/v1/bots/${botId}/tiers/${tierId}`, { method: 'DELETE' });
  }

  // project-teams: members sub-resource
  async getTeamMembers(teamId: string) { return this.request(`/api/v1/project-teams/${teamId}/members`); }
  async addTeamMember(teamId: string, body: any) {
    return this.request(`/api/v1/project-teams/${teamId}/members`, { method: 'POST', body: JSON.stringify(body) });
  }
  async removeTeamMember(teamId: string, memberId: string) {
    return this.request(`/api/v1/project-teams/${teamId}/members/${memberId}`, { method: 'DELETE' });
  }

  // liquidity-sources: priority + health + stats
  async setLiquiditySourcePriority(id: string, priority: number) {
    return this.request(`/api/v1/liquidity-sources/${id}/priority`, { method: 'PUT', body: JSON.stringify({ priority }) });
  }
  async liquiditySourceHealthCheck(id: string) { return this.request(`/api/v1/liquidity-sources/${id}/health`); }
  async getLiquiditySourceStats() { return this.request('/api/v1/liquidity-sources/stats'); }
}

export const api = new DesktopAdminAPI();

// Theme System
export type Theme = 'light' | 'dark';

export const getColors = (theme: Theme) => ({
  bg: theme === 'dark' ? '#0f172a' : '#f9fafb',
  bgCard: theme === 'dark' ? '#1e293b' : '#ffffff',
  bgHover: theme === 'dark' ? '#334155' : '#f3f4f6',
  text: theme === 'dark' ? '#f9fafb' : '#111827',
  textSecondary: theme === 'dark' ? '#9ca3af' : '#6b7280',
  border: theme === 'dark' ? '#374151' : '#e5e7eb',
  primary: '#dc2626',
  primaryHover: '#b91c1c',
  success: '#22c55e',
  warning: '#f59e0b',
  error: '#ef4444',
});

// Sidebar Component
const AdminSidebar: React.FC<{ currentPage: string; setCurrentPage: (page: string) => void; theme: Theme }> = ({ 
  currentPage, setCurrentPage, theme 
}) => {
  const colors = getColors(theme);
  
  const menuItems = [
    { id: 'dashboard', label: 'Dashboard', icon: '📊' },
    { id: 'users', label: 'Users', icon: '👥' },
    { id: 'transactions', label: 'Transactions', icon: '📜' },
    { id: 'kyc', label: 'KYC', icon: '✅' },
    { id: 'tokens', label: 'Tokens', icon: '🪙' },
    { id: 'withdrawals', label: 'Withdrawals', icon: '💸' },
    { id: 'fees', label: 'Fees', icon: '💰' },
    { id: 'system', label: 'System', icon: '🖥️' },
    { id: 'futures', label: 'Futures', icon: '📈' },
    { id: 'options', label: 'Options', icon: '🎚️' },
    { id: 'copy-trading', label: 'Copy Trading', icon: '🧑‍🤝‍🧑' },
    { id: 'convert', label: 'Convert', icon: '🔄' },
    { id: 'onramp', label: 'On-Ramp', icon: '⬇️' },
    { id: 'offramp', label: 'Off-Ramp', icon: '⬆️' },
    { id: 'p2p-clients', label: 'P2P Clients', icon: '🧑' },
    { id: 'p2p-merchants', label: 'P2P Merchants', icon: '🏪' },
    { id: 'partners', label: 'Partners', icon: '🤝' },
    { id: 'rewards', label: 'Rewards', icon: '🎁' },
    { id: 'marketing', label: 'Marketing', icon: '📣' },
    { id: 'roles', label: 'Roles', icon: '🛡️' },
    { id: 'permissions', label: 'Permissions', icon: '🔐' },
    { id: 'bots', label: 'Bots', icon: '🤖' },
    { id: 'bots-clients', label: 'Bots Clients', icon: '👤' },
    { id: 'project-teams', label: 'Project Teams', icon: '👥' },
    { id: 'liquidity-sources', label: 'Liquidity Sources', icon: '💧' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ];
  
  return (
    <aside className="w-64 flex flex-col" style={{ backgroundColor: colors.primary, color: 'white' }}>
      <div className="p-6 border-b" style={{ borderColor: 'rgba(255,255,255,0.2)' }}>
        <div className="flex items-center space-x-3">
          <span className="text-2xl">🔧</span>
          <span className="text-xl font-bold">Admin Panel</span>
        </div>
      </div>
      
      <nav className="flex-1 p-4">
        {menuItems.map(item => (
          <button
            key={item.id}
            onClick={() => setCurrentPage(item.id)}
            className="w-full flex items-center space-x-3 px-4 py-3 rounded-lg mb-2 transition-colors"
            style={{
              backgroundColor: currentPage === item.id ? colors.primaryHover : 'transparent',
              color: currentPage === item.id ? 'white' : 'rgba(255,255,255,0.8)',
            }}
          >
            <span>{item.icon}</span>
            <span>{item.label}</span>
          </button>
        ))}
      </nav>
    </aside>
  );
};

// Header Component
const AdminHeader: React.FC<{ theme: Theme; toggleTheme: () => void }> = ({ theme, toggleTheme }) => {
  const colors = getColors(theme);
  const [searchTerm, setSearchTerm] = useState('');
  
  return (
    <header className="h-16 flex items-center justify-between px-6" style={{ 
      backgroundColor: colors.bgCard, borderBottom: `1px solid ${colors.border}`
    }}>
      <input
        type="text"
        placeholder="Search users, transactions..."
        value={searchTerm}
        onChange={(e) => setSearchTerm(e.target.value)}
        className="px-4 py-2 rounded-lg text-sm w-96"
        style={{ backgroundColor: colors.bg, border: `1px solid ${colors.border}`, color: colors.text }}
      />
      
      <div className="flex items-center space-x-4">
        <button onClick={toggleTheme} className="p-2 rounded-lg" style={{ backgroundColor: colors.bgHover }}>
          {theme === 'dark' ? '☀️' : '🌙'}
        </button>
        <div className="w-8 h-8 rounded-full flex items-center justify-center" style={{ backgroundColor: colors.primary, color: 'white' }}>
          <span className="text-sm font-bold">A</span>
        </div>
      </div>
    </header>
  );
};

// Dashboard Component
const AdminDashboard: React.FC<{ theme: Theme }> = ({ theme }) => {
  const colors = getColors(theme);
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getAnalytics()
      .then(setStats)
      .catch(() => setStats({ totalUsers: 0, dailyTransactions: 0, totalVolume: '0', revenue: '0' }))
      .finally(() => setLoading(false));
  }, []);

  const statCards = [
    { label: 'Total Users', value: stats?.totalUsers || 0, icon: '👥' },
    { label: 'Volume (24h)', value: `$${parseFloat(stats?.totalVolume || '0').toLocaleString()}`, icon: '💰' },
    { label: 'Transactions', value: stats?.dailyTransactions || 0, icon: '📜' },
    { label: 'Revenue', value: `$${parseFloat(stats?.revenue || '0').toLocaleString()}`, icon: '💵' },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Dashboard</h1>
      
      <div className="grid grid-cols-4 gap-6">
        {statCards.map((stat, i) => (
          <div key={i} className="rounded-xl p-6" style={{ backgroundColor: colors.bgCard }}>
            <div className="mb-2" style={{ color: colors.textSecondary }}>{stat.icon} {stat.label}</div>
            <div className="text-3xl font-bold" style={{ color: colors.text }}>
              {loading ? '...' : stat.value}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

// Users Component
const AdminUsers: React.FC<{ theme: Theme }> = ({ theme }) => {
  const colors = getColors(theme);
  const [users, setUsers] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getUsers({ pageSize: 20 })
      .then((res: any) => setUsers(res.data || []))
      .catch(() => setUsers([]))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Users</h1>
      
      <div className="rounded-xl overflow-hidden" style={{ backgroundColor: colors.bgCard }}>
        <table className="w-full">
          <thead style={{ backgroundColor: colors.bgHover }}>
            <tr>
              {['Email', 'Status', 'KYC', 'Volume', 'Joined'].map(h => (
                <th key={h} className="px-6 py-3 text-left" style={{ color: colors.text }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={5} className="p-6 text-center" style={{ color: colors.textSecondary }}>Loading...</td></tr>
            ) : users.length === 0 ? (
              <tr><td colSpan={5} className="p-6 text-center" style={{ color: colors.textSecondary }}>No users found</td></tr>
            ) : users.map(user => (
              <tr key={user.id} className="border-b" style={{ borderColor: colors.border }}>
                <td className="px-6 py-4" style={{ color: colors.text }}>{user.email}</td>
                <td className="px-6 py-4">
                  <span className="px-2 py-1 rounded text-xs" style={{ 
                    backgroundColor: user.status === 'active' ? colors.success : colors.warning,
                    color: 'white'
                  }}>{user.status}</span>
                </td>
                <td className="px-6 py-4" style={{ color: colors.textSecondary }}>{user.kycStatus}</td>
                <td className="px-6 py-4" style={{ color: colors.text }}>${parseFloat(user.totalVolume || '0').toLocaleString()}</td>
                <td className="px-6 py-4" style={{ color: colors.textSecondary }}>
                  {new Date(user.createdAt).toLocaleDateString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Transactions Component
const AdminTransactions: React.FC<{ theme: Theme }> = ({ theme }) => {
  const colors = getColors(theme);
  const [transactions, setTransactions] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getTransactions({ pageSize: 20 })
      .then((res: any) => setTransactions(res.data || []))
      .catch(() => setTransactions([]))
      .finally(() => setLoading(false));
  }, []);

  const truncate = (s: string) => s ? s.slice(0, 6) + '...' + s.slice(-4) : '';

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Transactions</h1>
      
      <div className="rounded-xl overflow-hidden" style={{ backgroundColor: colors.bgCard }}>
        <table className="w-full">
          <thead style={{ backgroundColor: colors.bgHover }}>
            <tr>
              {['Hash', 'Type', 'Amount', 'Status', 'Time'].map(h => (
                <th key={h} className="px-6 py-3 text-left" style={{ color: colors.text }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={5} className="p-6 text-center" style={{ color: colors.textSecondary }}>Loading...</td></tr>
            ) : transactions.length === 0 ? (
              <tr><td colSpan={5} className="p-6 text-center" style={{ color: colors.textSecondary }}>No transactions</td></tr>
            ) : transactions.map(tx => (
              <tr key={tx.id} className="border-b" style={{ borderColor: colors.border }}>
                <td className="px-6 py-4 font-mono text-sm" style={{ color: colors.textSecondary }}>{truncate(tx.hash)}</td>
                <td className="px-6 py-4" style={{ color: colors.text }}>{tx.type}</td>
                <td className="px-6 py-4" style={{ color: colors.text }}>{tx.amount} {tx.tokenSymbol}</td>
                <td className="px-6 py-4">
                  <span className="px-2 py-1 rounded text-xs" style={{ 
                    backgroundColor: tx.status === 'confirmed' ? colors.success : tx.status === 'pending' ? colors.warning : colors.error,
                    color: 'white'
                  }}>{tx.status}</span>
                </td>
                <td className="px-6 py-4" style={{ color: colors.textSecondary }}>
                  {new Date(tx.timestamp).toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// KYC Component
const AdminKYC: React.FC<{ theme: Theme }> = ({ theme }) => {
  const colors = getColors(theme);
  const [records, setRecords] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getKycRecords({ pageSize: 20 })
      .then((res: any) => setRecords(res.data || []))
      .catch(() => setRecords([]))
      .finally(() => setLoading(false));
  }, []);

  const handleApprove = async (id: string) => {
    try {
      await api.approveKyc(id);
      setRecords(records.map(r => r.id === id ? { ...r, status: 'approved' } : r));
    } catch (e) { console.error(e); }
  };

  const handleReject = async (id: string) => {
    const reason = prompt('Rejection reason:');
    if (!reason) return;
    try {
      await api.rejectKyc(id, reason);
      setRecords(records.map(r => r.id === id ? { ...r, status: 'rejected' } : r));
    } catch (e) { console.error(e); }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold" style={{ color: colors.text }}>KYC Verification</h1>
      
      <div className="rounded-xl overflow-hidden" style={{ backgroundColor: colors.bgCard }}>
        <table className="w-full">
          <thead style={{ backgroundColor: colors.bgHover }}>
            <tr>
              {['User', 'Type', 'Status', 'Submitted', 'Actions'].map(h => (
                <th key={h} className="px-6 py-3 text-left" style={{ color: colors.text }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={5} className="p-6 text-center" style={{ color: colors.textSecondary }}>Loading...</td></tr>
            ) : records.length === 0 ? (
              <tr><td colSpan={5} className="p-6 text-center" style={{ color: colors.textSecondary }}>No records</td></tr>
            ) : records.map(r => (
              <tr key={r.id} className="border-b" style={{ borderColor: colors.border }}>
                <td className="px-6 py-4" style={{ color: colors.text }}>{r.userEmail}</td>
                <td className="px-6 py-4" style={{ color: colors.textSecondary }}>{r.documentType}</td>
                <td className="px-6 py-4">
                  <span className="px-2 py-1 rounded text-xs" style={{ 
                    backgroundColor: r.status === 'approved' ? colors.success : r.status === 'rejected' ? colors.error : colors.warning,
                    color: 'white'
                  }}>{r.status}</span>
                </td>
                <td className="px-6 py-4" style={{ color: colors.textSecondary }}>
                  {new Date(r.submittedAt).toLocaleDateString()}
                </td>
                <td className="px-6 py-4">
                  {r.status === 'pending' && (
                    <div className="flex gap-2">
                      <button onClick={() => handleApprove(r.id)} className="px-2 py-1 rounded text-xs" style={{ backgroundColor: colors.success, color: 'white' }}>Approve</button>
                      <button onClick={() => handleReject(r.id)} className="px-2 py-1 rounded text-xs" style={{ backgroundColor: colors.error, color: 'white' }}>Reject</button>
                    </div>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Tokens Component
const AdminTokens: React.FC<{ theme: Theme }> = ({ theme }) => {
  const colors = getColors(theme);
  const [tokens, setTokens] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getTokens()
      .then((res: any) => setTokens(res.data || []))
      .catch(() => setTokens([]))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Tokens</h1>
      
      <div className="rounded-xl overflow-hidden" style={{ backgroundColor: colors.bgCard }}>
        <table className="w-full">
          <thead style={{ backgroundColor: colors.bgHover }}>
            <tr>
              {['Name', 'Symbol', 'Chain', 'Price', 'Listed'].map(h => (
                <th key={h} className="px-6 py-3 text-left" style={{ color: colors.text }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={5} className="p-6 text-center" style={{ color: colors.textSecondary }}>Loading...</td></tr>
            ) : tokens.length === 0 ? (
              <tr><td colSpan={5} className="p-6 text-center" style={{ color: colors.textSecondary }}>No tokens</td></tr>
            ) : tokens.map(t => (
              <tr key={t.id} className="border-b" style={{ borderColor: colors.border }}>
                <td className="px-6 py-4" style={{ color: colors.text }}>{t.name}</td>
                <td className="px-6 py-4" style={{ color: colors.textSecondary }}>{t.symbol}</td>
                <td className="px-6 py-4" style={{ color: colors.textSecondary }}>{t.chain}</td>
                <td className="px-6 py-4" style={{ color: colors.text }}>${t.price || '0'}</td>
                <td className="px-6 py-4">
                  <span className="px-2 py-1 rounded text-xs" style={{ 
                    backgroundColor: t.isListed ? colors.success : colors.textSecondary,
                    color: 'white'
                  }}>{t.isListed ? 'Yes' : 'No'}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Withdrawals Component
const AdminWithdrawals: React.FC<{ theme: Theme }> = ({ theme }) => {
  const colors = getColors(theme);
  const [withdrawals, setWithdrawals] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getWithdrawals()
      .then((res: any) => setWithdrawals(res.data || []))
      .catch(() => setWithdrawals([]))
      .finally(() => setLoading(false));
  }, []);

  const handleApprove = async (id: string) => {
    try {
      await api.approveWithdrawal(id);
      setWithdrawals(withdrawals.map(w => w.id === id ? { ...w, status: 'approved' } : w));
    } catch (e) { console.error(e); }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Withdrawals</h1>
      
      <div className="rounded-xl overflow-hidden" style={{ backgroundColor: colors.bgCard }}>
        <table className="w-full">
          <thead style={{ backgroundColor: colors.bgHover }}>
            <tr>
              {['User', 'Amount', 'Token', 'Status', 'Actions'].map(h => (
                <th key={h} className="px-6 py-3 text-left" style={{ color: colors.text }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={5} className="p-6 text-center" style={{ color: colors.textSecondary }}>Loading...</td></tr>
            ) : withdrawals.length === 0 ? (
              <tr><td colSpan={5} className="p-6 text-center" style={{ color: colors.textSecondary }}>No withdrawals</td></tr>
            ) : withdrawals.map(w => (
              <tr key={w.id} className="border-b" style={{ borderColor: colors.border }}>
                <td className="px-6 py-4" style={{ color: colors.text }}>{w.userEmail}</td>
                <td className="px-6 py-4" style={{ color: colors.text }}>{w.amount}</td>
                <td className="px-6 py-4" style={{ color: colors.textSecondary }}>{w.token}</td>
                <td className="px-6 py-4">
                  <span className="px-2 py-1 rounded text-xs" style={{ 
                    backgroundColor: w.status === 'approved' || w.status === 'completed' ? colors.success : w.status === 'pending' ? colors.warning : colors.error,
                    color: 'white'
                  }}>{w.status}</span>
                </td>
                <td className="px-6 py-4">
                  {w.status === 'pending' && (
                    <button onClick={() => handleApprove(w.id)} className="px-2 py-1 rounded text-xs" style={{ backgroundColor: colors.success, color: 'white' }}>
                      Approve
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Fees Component
const AdminFees: React.FC<{ theme: Theme }> = ({ theme }) => {
  const colors = getColors(theme);
  const [fees, setFees] = useState<any>({});
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getFeeConfig()
      .then(setFees)
      .catch(() => ({}))
      .finally(() => setLoading(false));
  }, []);

  const handleSave = async () => {
    try {
      await api.updateFeeConfig(fees);
      alert('Fees updated!');
    } catch (e) { console.error(e); }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Fee Configuration</h1>
      
      <div className="rounded-xl p-6" style={{ backgroundColor: colors.bgCard }}>
        {loading ? (
          <p style={{ color: colors.textSecondary }}>Loading...</p>
        ) : (
          <div className="space-y-4">
            {['tradingFee', 'withdrawalFee', 'depositFee', 'makerFee', 'takerFee'].map(field => (
              <div key={field} className="flex items-center justify-between">
                <label style={{ color: colors.text }}>{field.replace(/([A-Z])/g, ' $1').replace(/^./, s => s.toUpperCase())}</label>
                <input
                  type="text"
                  value={fees[field] || ''}
                  onChange={(e) => setFees({ ...fees, [field]: e.target.value })}
                  className="px-3 py-2 rounded-lg w-32"
                  style={{ backgroundColor: colors.bg, border: `1px solid ${colors.border}`, color: colors.text }}
                />
              </div>
            ))}
            <button onClick={handleSave} className="px-4 py-2 rounded-lg" style={{ backgroundColor: colors.primary, color: 'white' }}>
              Save Changes
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

// System Component
const AdminSystem: React.FC<{ theme: Theme }> = ({ theme }) => {
  const colors = getColors(theme);
  const [services, setServices] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getSystemStatus()
      .then(setServices)
      .catch(() => setServices([]))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold" style={{ color: colors.text }}>System Status</h1>
      
      <div className="grid grid-cols-2 gap-6">
        <div className="rounded-xl p-6" style={{ backgroundColor: colors.bgCard }}>
          <h2 className="text-lg font-semibold mb-4" style={{ color: colors.text }}>Services</h2>
          {loading ? (
            <p style={{ color: colors.textSecondary }}>Loading...</p>
          ) : services.length === 0 ? (
            <p style={{ color: colors.textSecondary }}>No services</p>
          ) : (
            services.slice(0, 6).map((s, i) => (
              <div key={i} className="flex items-center justify-between py-2 border-b" style={{ borderColor: colors.border }}>
                <div className="flex items-center gap-2">
                  <span style={{ color: s.status === 'running' ? colors.success : colors.error }}>●</span>
                  <span style={{ color: colors.text }}>{s.name}</span>
                </div>
                <span style={{ color: colors.textSecondary }}>{s.uptime}</span>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
};

// Settings Component  
const AdminSettings: React.FC<{ theme: Theme; setTheme: (t: Theme) => void }> = ({ theme, setTheme }) => {
  const colors = getColors(theme);
  
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Settings</h1>
      
      <div className="rounded-xl p-6 space-y-4" style={{ backgroundColor: colors.bgCard }}>
        <h2 className="text-lg font-semibold" style={{ color: colors.text }}>Appearance</h2>
        <div className="flex items-center justify-between">
          <span style={{ color: colors.text }}>Dark Mode</span>
          <button 
            onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
            className="w-14 h-7 rounded-full transition-colors"
            style={{ backgroundColor: theme === 'dark' ? colors.primary : colors.textSecondary }}
          >
            <div className={`w-5 h-5 bg-white rounded-full transform transition-transform ${theme === 'dark' ? 'translate-x-7' : 'translate-x-1'}`} />
          </button>
        </div>
      </div>
    </div>
  );
};

// Main App
const AdminDesktopApp: React.FC = () => {
  const [currentPage, setCurrentPage] = useState('dashboard');
  const [theme, setTheme] = useState<Theme>(() => {
    const stored = localStorage.getItem('admin_theme');
    return (stored as Theme) || 'dark';
  });

  useEffect(() => {
    localStorage.setItem('admin_theme', theme);
    document.documentElement.classList.remove('light', 'dark');
    document.documentElement.classList.add(theme);
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  const toggleTheme = () => setTheme(t => t === 'dark' ? 'light' : 'dark');

  const renderPage = () => {
    const props = { theme };
    switch (currentPage) {
      case 'dashboard': return <AdminDashboard {...props} />;
      case 'users': return <AdminUsers {...props} />;
      case 'transactions': return <AdminTransactions {...props} />;
      case 'kyc': return <AdminKYC {...props} />;
      case 'tokens': return <AdminTokens {...props} />;
      case 'withdrawals': return <AdminWithdrawals {...props} />;
      case 'fees': return <AdminFees {...props} />;
      case 'system': return <AdminSystem {...props} />;
      case 'settings': return <AdminSettings {...props} theme={theme} setTheme={setTheme} />;
      default: {
        const dp = DOMAIN_PAGES.find(p => p.id === currentPage);
        if (dp) return <DomainPage config={dp} theme={theme} />;
        return <AdminDashboard {...props} />;
      }
    }
  };

  const colors = getColors(theme);

  return (
    <div className="flex h-screen" style={{ backgroundColor: colors.bg }}>
      <AdminSidebar currentPage={currentPage} setCurrentPage={setCurrentPage} theme={theme} />
      <div className="flex-1 flex flex-col overflow-hidden">
        <AdminHeader theme={theme} toggleTheme={toggleTheme} />
        <main className="flex-1 overflow-auto p-6">
          {renderPage()}
        </main>
      </div>
    </div>
  );
};

export default AdminDesktopApp;
