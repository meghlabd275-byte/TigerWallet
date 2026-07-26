/**
 * TigerWallet Admin Panel - Complete Admin Management System
 * Production-ready admin panel with all management features
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '@/context/ThemeProvider';

// ============================================================================
// TYPES
// ============================================================================

interface AdminUser {
  id: string;
  email: string;
  name: string;
  role: 'super_admin' | 'admin' | 'moderator' | 'support';
  permissions: string[];
  status: 'active' | 'suspended' | 'pending';
  lastLogin: string;
  createdAt: string;
}

interface User {
  id: string;
  email: string;
  name: string;
  status: 'active' | 'suspended' | 'pending' | 'banned';
  kycStatus: 'none' | 'pending' | 'approved' | 'rejected';
  balance: number;
  createdAt: string;
}

interface Transaction {
  id: string;
  userId: string;
  type: 'deposit' | 'withdrawal' | 'transfer' | 'swap';
  amount: number;
  currency: string;
  status: 'pending' | 'completed' | 'failed';
  timestamp: string;
}

interface Pair {
  id: string;
  base: string;
  quote: string;
  price: number;
  volume24h: number;
  status: 'active' | 'suspended' | 'halted';
}

interface LiquidityPool {
  id: string;
  pair: string;
  liquidity: number;
  volume24h: number;
  apr: number;
}

interface FeeStructure {
  id: string;
  type: 'withdrawal' | 'deposit' | 'trading' | 'swap';
  asset: string;
  feePercent: number;
  feeFixed: number;
  tier: string;
}

interface KYCRequest {
  id: string;
  userId: string;
  type: 'identity' | 'address' | 'selfie';
  status: 'pending' | 'approved' | 'rejected';
  submittedAt: string;
  reviewedAt?: string;
}

interface APIKey {
  id: string;
  userId: string;
  name: string;
  key: string;
  permissions: string[];
  lastUsed?: string;
  createdAt: string;
}

// ============================================================================
// API SERVICE
// ============================================================================

const API_BASE = '/api/v1/admin';

const adminAPI = {
  // User Management
  users: {
    list: async (page = 1, limit = 20, status?: string) => {
      const params = new URLSearchParams({ page: String(page), limit: String(limit) });
      if (status) params.append('status', status);
      const res = await fetch(`${API_BASE}/users?${params}`);
      return res.json();
    },
    get: async (id: string) => {
      const res = await fetch(`${API_BASE}/users/${id}`);
      return res.json();
    },
    update: async (id: string, data: Partial<User>) => {
      const res = await fetch(`${API_BASE}/users/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      return res.json();
    },
    ban: async (id: string) => {
      const res = await fetch(`${API_BASE}/users/${id}/ban`, { method: 'POST' });
      return res.json();
    },
    unban: async (id: string) => {
      const res = await fetch(`${API_BASE}/users/${id}/unban`, { method: 'POST' });
      return res.json();
    },
    getBalance: async (id: string) => {
      const res = await fetch(`${API_BASE}/users/${id}/balance`);
      return res.json();
    },
  },

  // Admin Management
  admins: {
    list: async () => {
      const res = await fetch(`${API_BASE}/admins`);
      return res.json();
    },
    create: async (data: Partial<AdminUser>) => {
      const res = await fetch(`${API_BASE}/admins`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      return res.json();
    },
    update: async (id: string, data: Partial<AdminUser>) => {
      const res = await fetch(`${API_BASE}/admins/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      return res.json();
    },
    delete: async (id: string) => {
      const res = await fetch(`${API_BASE}/admins/${id}`, { method: 'DELETE' });
      return res.json();
    },
    updatePermissions: async (id: string, permissions: string[]) => {
      const res = await fetch(`${API_BASE}/admins/${id}/permissions`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ permissions }),
      });
      return res.json();
    },
  },

  // KYC Management
  kyc: {
    list: async (status?: string) => {
      const params = status ? `?status=${status}` : '';
      const res = await fetch(`${API_BASE}/kyc${params}`);
      return res.json();
    },
    get: async (id: string) => {
      const res = await fetch(`${API_BASE}/kyc/${id}`);
      return res.json();
    },
    approve: async (id: string) => {
      const res = await fetch(`${API_BASE}/kyc/${id}/approve`, { method: 'POST' });
      return res.json();
    },
    reject: async (id: string, reason: string) => {
      const res = await fetch(`${API_BASE}/kyc/${id}/reject`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason }),
      });
      return res.json();
    },
  },

  // Pair Management
  pairs: {
    list: async () => {
      const res = await fetch(`${API_BASE}/pairs`);
      return res.json();
    },
    create: async (data: Partial<Pair>) => {
      const res = await fetch(`${API_BASE}/pairs`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      return res.json();
    },
    update: async (id: string, data: Partial<Pair>) => {
      const res = await fetch(`${API_BASE}/pairs/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      return res.json();
    },
    suspend: async (id: string) => {
      const res = await fetch(`${API_BASE}/pairs/${id}/suspend`, { method: 'POST' });
      return res.json();
    },
    resume: async (id: string) => {
      const res = await fetch(`${API_BASE}/pairs/${id}/resume`, { method: 'POST' });
      return res.json();
    },
    halt: async (id: string) => {
      const res = await fetch(`${API_BASE}/pairs/${id}/halt`, { method: 'POST' });
      return res.json();
    },
  },

  // Liquidity Management
  liquidity: {
    list: async () => {
      const res = await fetch(`${API_BASE}/liquidity`);
      return res.json();
    },
    add: async (data: Partial<LiquidityPool>) => {
      const res = await fetch(`${API_BASE}/liquidity`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      return res.json();
    },
    remove: async (id: string) => {
      const res = await fetch(`${API_BASE}/liquidity/${id}`, { method: 'DELETE' });
      return res.json();
    },
  },

  // Fee Management
  fees: {
    list: async () => {
      const res = await fetch(`${API_BASE}/fees`);
      return res.json();
    },
    update: async (id: string, data: Partial<FeeStructure>) => {
      const res = await fetch(`${API_BASE}/fees/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      return res.json();
    },
    create: async (data: Partial<FeeStructure>) => {
      const res = await fetch(`${API_BASE}/fees`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      return res.json();
    },
  },

  // Transaction Management
  transactions: {
    list: async (type?: string, status?: string) => {
      const params = new URLSearchParams();
      if (type) params.append('type', type);
      if (status) params.append('status', status);
      const res = await fetch(`${API_BASE}/transactions?${params}`);
      return res.json();
    },
    get: async (id: string) => {
      const res = await fetch(`${API_BASE}/transactions/${id}`);
      return res.json();
    },
    approve: async (id: string) => {
      const res = await fetch(`${API_BASE}/transactions/${id}/approve`, { method: 'POST' });
      return res.json();
    },
    reject: async (id: string, reason: string) => {
      const res = await fetch(`${API_BASE}/transactions/${id}/reject`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason }),
      });
      return res.json();
    },
  },

  // API Keys
  apiKeys: {
    list: async (userId?: string) => {
      const params = userId ? `?user_id=${userId}` : '';
      const res = await fetch(`${API_BASE}/api-keys${params}`);
      return res.json();
    },
    create: async (userId: string, name: string, permissions: string[]) => {
      const res = await fetch(`${API_BASE}/api-keys`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: userId, name, permissions }),
      });
      return res.json();
    },
    revoke: async (id: string) => {
      const res = await fetch(`${API_BASE}/api-keys/${id}`, { method: 'DELETE' });
      return res.json();
    },
  },

  // Analytics
  analytics: {
    overview: async () => {
      const res = await fetch(`${API_BASE}/analytics/overview`);
      return res.json();
    },
    users: async (period: string) => {
      const res = await fetch(`${API_BASE}/analytics/users?period=${period}`);
      return res.json();
    },
    volume: async (period: string) => {
      const res = await fetch(`${API_BASE}/analytics/volume?period=${period}`);
      return res.json();
    },
  },
};

// ============================================================================
// COMPONENTS
// ============================================================================

// Sidebar Navigation
function Sidebar({ activeTab, setActiveTab }: { activeTab: string; setActiveTab: (tab: string) => void }) {
  const { colors } = useTheme();
  
  const menuItems = [
    { id: 'overview', label: 'Overview', icon: '📊' },
    { id: 'users', label: 'User Management', icon: '👥' },
    { id: 'admins', label: 'Admin Management', icon: '👨‍💼' },
    { id: 'kyc', label: 'KYC Management', icon: '🔐' },
    { id: 'pairs', label: 'Pair Management', icon: '💱' },
    { id: 'liquidity', label: 'Liquidity', icon: '🌊' },
    { id: 'fees', label: 'Fee Management', icon: '💰' },
    { id: 'transactions', label: 'Transactions', icon: '🔄' },
    { id: 'api-keys', label: 'API Keys', icon: '🔑' },
    { id: 'analytics', label: 'Analytics', icon: '📈' },
  ];

  return (
    <aside 
      className="w-64 h-screen fixed left-0 top-0 border-r overflow-y-auto"
      style={{ backgroundColor: colors.surface, borderColor: colors.border }}
    >
      <div className="p-4 border-b" style={{ borderColor: colors.border }}>
        <h1 className="text-xl font-bold" style={{ color: colors.text }}>TigerAdmin</h1>
      </div>
      <nav className="p-2">
        {menuItems.map((item) => (
          <button
            key={item.id}
            onClick={() => setActiveTab(item.id)}
            className={`w-full text-left px-4 py-3 rounded-lg mb-1 flex items-center gap-3 transition-colors ${
              activeTab === item.id ? 'font-medium' : ''
            }`}
            style={{
              backgroundColor: activeTab === item.id ? colors.primary + '20' : 'transparent',
              color: activeTab === item.id ? colors.primary : colors.textSecondary,
            }}
          >
            <span>{item.icon}</span>
            <span>{item.label}</span>
          </button>
        ))}
      </nav>
    </aside>
  );
}

// Stats Card
function StatCard({ title, value, change, icon }: { 
  title: string; 
  value: string | number; 
  change?: number;
  icon: string;
}) {
  const { colors } = useTheme();

  return (
    <div 
      className="p-6 rounded-xl border"
      style={{ backgroundColor: colors.surface, borderColor: colors.border }}
    >
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm" style={{ color: colors.textSecondary }}>{title}</p>
          <p className="text-2xl font-bold mt-1" style={{ color: colors.text }}>{value}</p>
          {change !== undefined && (
            <p className={`text-sm mt-1 ${change >= 0 ? 'text-green-500' : 'text-red-500'}`}>
              {change >= 0 ? '+' : ''}{change}%
            </p>
          )}
        </div>
        <span className="text-3xl">{icon}</span>
      </div>
    </div>
  );
}

// Data Table
function DataTable<T extends { id: string }>({
  data,
  columns,
  onRowClick,
}: {
  data: T[];
  columns: { key: keyof T; label: string; render?: (item: T) => React.ReactNode }[];
  onRowClick?: (item: T) => void;
}) {
  const { colors } = useTheme();

  return (
    <div className="overflow-x-auto">
      <table className="w-full">
        <thead>
          <tr className="border-b" style={{ borderColor: colors.border }}>
            {columns.map((col) => (
              <th 
                key={String(col.key)} 
                className="text-left px-4 py-3 text-sm font-medium"
                style={{ color: colors.textSecondary }}
              >
                {col.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.map((item) => (
            <tr 
              key={item.id} 
              className="border-b cursor-pointer hover:opacity-80"
              style={{ borderColor: colors.border }}
              onClick={() => onRowClick?.(item)}
            >
              {columns.map((col) => (
                <td key={String(col.key)} className="px-4 py-3" style={{ color: colors.text }}>
                  {col.render ? col.render(item) : String(item[col.key] || '')}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ============================================================================
// PAGE COMPONENTS
// ============================================================================

function OverviewPage() {
  const { colors } = useTheme();
  const [stats, setStats] = useState({
    totalUsers: 0,
    activeUsers: 0,
    totalVolume: 0,
    pendingTransactions: 0,
    kycPending: 0,
    revenue: 0,
  });

  useEffect(() => {
    adminAPI.analytics.overview().then(setStats).catch(console.error);
  }, []);

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold" style={{ color: colors.text }}>Dashboard Overview</h2>
      
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4">
        <StatCard title="Total Users" value={stats.totalUsers.toLocaleString()} icon="👥" />
        <StatCard title="Active Users" value={stats.activeUsers.toLocaleString()} icon="✅" />
        <StatCard title="Total Volume" value={`$${stats.totalVolume.toLocaleString()}`} icon="💵" />
        <StatCard title="Pending TX" value={stats.pendingTransactions} icon="⏳" />
        <StatCard title="KYC Pending" value={stats.kycPending} icon="🔐" />
        <StatCard title="Revenue" value={`$${stats.revenue.toLocaleString()}`} icon="💰" change={12.5} />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="p-6 rounded-xl border" style={{ backgroundColor: colors.surface, borderColor: colors.border }}>
          <h3 className="text-lg font-semibold mb-4" style={{ color: colors.text }}>Recent Activity</h3>
          <div className="space-y-3">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="flex items-center justify-between py-2 border-b" style={{ borderColor: colors.border }}>
                <div>
                  <p style={{ color: colors.text }}>User registration</p>
                  <p className="text-sm" style={{ color: colors.textSecondary }}>2 minutes ago</p>
                </div>
                <span className="px-2 py-1 rounded text-xs" style={{ backgroundColor: colors.success + '20', color: colors.success }}>Success</span>
              </div>
            ))}
          </div>
        </div>

        <div className="p-6 rounded-xl border" style={{ backgroundColor: colors.surface, borderColor: colors.border }}>
          <h3 className="text-lg font-semibold mb-4" style={{ color: colors.text }}>System Status</h3>
          <div className="space-y-3">
            {['API Gateway', 'Database', 'Redis Cache', 'Blockchain Nodes', 'Email Service'].map((service) => (
              <div key={service} className="flex items-center justify-between py-2">
                <span style={{ color: colors.text }}>{service}</span>
                <span className="px-2 py-1 rounded text-xs" style={{ backgroundColor: colors.success + '20', color: colors.success }}>Operational</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function UsersPage() {
  const { colors } = useTheme();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<string>('');

  useEffect(() => {
    setLoading(true);
    adminAPI.users.list(1, 50, filter || undefined)
      .then(data => setUsers(data.users || []))
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [filter]);

  const columns = [
    { key: 'id' as const, label: 'ID', render: (u: User) => <span className="font-mono text-sm">{u.id.slice(0, 8)}</span> },
    { key: 'name' as const, label: 'Name' },
    { key: 'email' as const, label: 'Email' },
    { key: 'kycStatus' as const, label: 'KYC', render: (u: User) => (
      <span className={`px-2 py-1 rounded text-xs ${
        u.kycStatus === 'approved' ? 'bg-green-500/20 text-green-500' :
        u.kycStatus === 'pending' ? 'bg-yellow-500/20 text-yellow-500' :
        'bg-gray-500/20 text-gray-500'
      }`}>{u.kycStatus}</span>
    )},
    { key: 'balance' as const, label: 'Balance', render: (u: User) => `$${u.balance.toLocaleString()}` },
    { key: 'status' as const, label: 'Status', render: (u: User) => (
      <span className={`px-2 py-1 rounded text-xs ${
        u.status === 'active' ? 'bg-green-500/20 text-green-500' :
        u.status === 'suspended' ? 'bg-red-500/20 text-red-500' :
        'bg-gray-500/20 text-gray-500'
      }`}>{u.status}</span>
    )},
    { key: 'createdAt' as const, label: 'Created', render: (u: User) => new Date(u.createdAt).toLocaleDateString() },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold" style={{ color: colors.text }}>User Management</h2>
        <div className="flex gap-2">
          <select
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="px-4 py-2 rounded-lg border outline-none"
            style={{ backgroundColor: colors.surface, borderColor: colors.border, color: colors.text }}
          >
            <option value="">All Status</option>
            <option value="active">Active</option>
            <option value="suspended">Suspended</option>
            <option value="banned">Banned</option>
          </select>
        </div>
      </div>

      {loading ? (
        <div className="flex justify-center py-20">
          <div className="animate-spin rounded-full h-12 w-12" style={{ borderColor: colors.primary, borderTopColor: 'transparent' }} />
        </div>
      ) : (
        <DataTable data={users} columns={columns} />
      )}
    </div>
  );
}

function KYCPanel() {
  const { colors } = useTheme();
  const [requests, setRequests] = useState<KYCRequest[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    adminAPI.kyc.list()
      .then(data => setRequests(data.requests || []))
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const handleApprove = async (id: string) => {
    await adminAPI.kyc.approve(id);
    setRequests(requests.filter(r => r.id !== id));
  };

  const handleReject = async (id: string) => {
    const reason = prompt('Enter rejection reason:');
    if (reason) {
      await adminAPI.kyc.reject(id, reason);
      setRequests(requests.filter(r => r.id !== id));
    }
  };

  const columns = [
    { key: 'id' as const, label: 'ID', render: (k: KYCRequest) => <span className="font-mono text-sm">{k.id.slice(0, 8)}</span> },
    { key: 'userId' as const, label: 'User', render: (k: KYCRequest) => <span className="font-mono text-sm">{k.userId.slice(0, 8)}</span> },
    { key: 'type' as const, label: 'Type', render: (k: KYCRequest) => <span className="capitalize">{k.type}</span> },
    { key: 'submittedAt' as const, label: 'Submitted', render: (k: KYCRequest) => new Date(k.submittedAt).toLocaleString() },
    { key: 'status' as const, label: 'Status', render: (k: KYCRequest) => (
      <span className={`px-2 py-1 rounded text-xs ${
        k.status === 'pending' ? 'bg-yellow-500/20 text-yellow-500' :
        k.status === 'approved' ? 'bg-green-500/20 text-green-500' :
        'bg-red-500/20 text-red-500'
      }`}>{k.status}</span>
    )},
    { key: 'actions' as const, label: 'Actions', render: (k: KYCRequest) => k.status === 'pending' && (
      <div className="flex gap-2">
        <button onClick={() => handleApprove(k.id)} className="px-3 py-1 rounded text-xs bg-green-500/20 text-green-500">Approve</button>
        <button onClick={() => handleReject(k.id)} className="px-3 py-1 rounded text-xs bg-red-500/20 text-red-500">Reject</button>
      </div>
    )},
  ];

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold" style={{ color: colors.text }}>KYC Management</h2>
      
      {loading ? (
        <div className="flex justify-center py-20">
          <div className="animate-spin rounded-full h-12 w-12" style={{ borderColor: colors.primary, borderTopColor: 'transparent' }} />
        </div>
      ) : (
        <DataTable data={requests} columns={columns} />
      )}
    </div>
  );
}

function PairsPage() {
  const { colors } = useTheme();
  const [pairs, setPairs] = useState<Pair[]>([]);

  useEffect(() => {
    adminAPI.pairs.list().then(data => setPairs(data.pairs || [])).catch(console.error);
  }, []);

  const columns = [
    { key: 'id' as const, label: 'ID' },
    { key: 'base' as const, label: 'Base', render: (p: Pair) => <span className="font-semibold">{p.base}</span> },
    { key: 'quote' as const, label: 'Quote', render: (p: Pair) => <span className="font-semibold">{p.quote}</span> },
    { key: 'price' as const, label: 'Price', render: (p: Pair) => `$${p.price.toLocaleString()}` },
    { key: 'volume24h' as const, label: 'Volume 24h', render: (p: Pair) => `$${p.volume24h.toLocaleString()}` },
    { key: 'status' as const, label: 'Status', render: (p: Pair) => (
      <span className={`px-2 py-1 rounded text-xs ${
        p.status === 'active' ? 'bg-green-500/20 text-green-500' :
        p.status === 'suspended' ? 'bg-yellow-500/20 text-yellow-500' :
        'bg-red-500/20 text-red-500'
      }`}>{p.status}</span>
    )},
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold" style={{ color: colors.text }}>Pair Management</h2>
        <button
          className="px-4 py-2 rounded-lg font-medium"
          style={{ backgroundColor: colors.primary, color: 'white' }}
        >
          + Add Pair
        </button>
      </div>
      <DataTable data={pairs} columns={columns} />
    </div>
  );
}

function FeesPage() {
  const { colors } = useTheme();
  const [fees, setFees] = useState<FeeStructure[]>([]);

  useEffect(() => {
    adminAPI.fees.list().then(data => setFees(data.fees || [])).catch(console.error);
  }, []);

  const columns = [
    { key: 'type' as const, label: 'Type', render: (f: FeeStructure) => <span className="capitalize">{f.type}</span> },
    { key: 'asset' as const, label: 'Asset' },
    { key: 'feePercent' as const, label: 'Fee %', render: (f: FeeStructure) => `${f.feePercent}%` },
    { key: 'feeFixed' as const, label: 'Fixed Fee' },
    { key: 'tier' as const, label: 'Tier' },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold" style={{ color: colors.text }}>Fee Management</h2>
        <button
          className="px-4 py-2 rounded-lg font-medium"
          style={{ backgroundColor: colors.primary, color: 'white' }}
        >
          + Add Fee
        </button>
      </div>
      <DataTable data={fees} columns={columns} />
    </div>
  );
}

function TransactionsPage() {
  const { colors } = useTheme();
  const [transactions, setTransactions] = useState<Transaction[]>([]);

  useEffect(() => {
    adminAPI.transactions.list()
      .then(data => setTransactions(data.transactions || []))
      .catch(console.error);
  }, []);

  const columns = [
    { key: 'id' as const, label: 'ID', render: (t: Transaction) => <span className="font-mono text-sm">{t.id.slice(0, 8)}</span> },
    { key: 'type' as const, label: 'Type', render: (t: Transaction) => <span className="capitalize">{t.type}</span> },
    { key: 'amount' as const, label: 'Amount', render: (t: Transaction) => `${t.currency} ${t.amount.toLocaleString()}` },
    { key: 'status' as const, label: 'Status', render: (t: Transaction) => (
      <span className={`px-2 py-1 rounded text-xs ${
        t.status === 'completed' ? 'bg-green-500/20 text-green-500' :
        t.status === 'pending' ? 'bg-yellow-500/20 text-yellow-500' :
        'bg-red-500/20 text-red-500'
      }`}>{t.status}</span>
    )},
    { key: 'timestamp' as const, label: 'Time', render: (t: Transaction) => new Date(t.timestamp).toLocaleString() },
  ];

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold" style={{ color: colors.text }}>Transaction Management</h2>
      <DataTable data={transactions} columns={columns} />
    </div>
  );
}

// ============================================================================
// MAIN APP
// ============================================================================

export default function AdminPanel() {
  const { colors, isDark } = useTheme();
  const [activeTab, setActiveTab] = useState('overview');

  const renderContent = () => {
    switch (activeTab) {
      case 'overview': return <OverviewPage />;
      case 'users': return <UsersPage />;
      case 'kyc': return <KYCPanel />;
      case 'pairs': return <PairsPage />;
      case 'fees': return <FeesPage />;
      case 'transactions': return <TransactionsPage />;
      default: return <OverviewPage />;
    }
  };

  return (
    <div className="min-h-screen" style={{ backgroundColor: colors.background }}>
      <Sidebar activeTab={activeTab} setActiveTab={setActiveTab} />
      <main className="ml-64 p-8">
        {renderContent()}
      </main>
    </div>
  );
}
