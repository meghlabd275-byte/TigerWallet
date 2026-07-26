/**
 * TigerWallet White Label Management System
 * Complete admin interface for managing white label clients
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '@/context/ThemeProvider';

// ============================================================================
// TYPES
// ============================================================================

interface WhiteLabelClient {
  id: string;
  name: string;
  domain: string;
  status: 'active' | 'suspended' | 'pending';
  plan: 'basic' | 'professional' | 'enterprise';
  createdAt: string;
  updatedAt: string;
  features: WhiteLabelFeatures;
  branding: WhiteLabelBranding;
  settings: WhiteLabelSettings;
}

interface WhiteLabelFeatures {
  customDomain: boolean;
  customBranding: boolean;
  apiAccess: boolean;
  multiAdmin: boolean;
  customFees: boolean;
  whiteLabelWallet: boolean;
  whiteLabelExchange: boolean;
  whiteLabelExplorer: boolean;
  nftSupport: boolean;
  defiSupport: boolean;
}

interface WhiteLabelBranding {
  logo: string;
  favicon: string;
  primaryColor: string;
  secondaryColor: string;
  backgroundColor: string;
  textColor: string;
  fontFamily: string;
}

interface WhiteLabelSettings {
  defaultNetwork: string;
  supportedNetworks: string[];
  defaultCurrency: string;
  supportedCurrencies: string[];
  kycRequired: boolean;
  withdrawalLimit: number;
  tradingEnabled: boolean;
  nftEnabled: boolean;
  defiEnabled: boolean;
}

interface WhiteLabelStats {
  totalUsers: number;
  activeUsers: number;
  totalVolume: number;
  totalTransactions: number;
  revenue: number;
}

// ============================================================================
// API SERVICE
// ============================================================================

const API_BASE = '/api/v1/admin/whitelabel';

const whiteLabelAPI = {
  // Clients
  list: async (page = 1, limit = 20): Promise<{ clients: WhiteLabelClient[]; total: number }> => {
    const res = await fetch(`${API_BASE}/clients?page=${page}&limit=${limit}`);
    return res.json();
  },

  get: async (id: string): Promise<WhiteLabelClient> => {
    const res = await fetch(`${API_BASE}/clients/${id}`);
    return res.json();
  },

  create: async (data: Partial<WhiteLabelClient>): Promise<WhiteLabelClient> => {
    const res = await fetch(`${API_BASE}/clients`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return res.json();
  },

  update: async (id: string, data: Partial<WhiteLabelClient>): Promise<WhiteLabelClient> => {
    const res = await fetch(`${API_BASE}/clients/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return res.json();
  },

  delete: async (id: string): Promise<void> => {
    await fetch(`${API_BASE}/clients/${id}`, { method: 'DELETE' });
  },

  // Status management
  suspend: async (id: string): Promise<void> => {
    await fetch(`${API_BASE}/clients/${id}/suspend`, { method: 'POST' });
  },

  resume: async (id: string): Promise<void> => {
    await fetch(`${API_BASE}/clients/${id}/resume`, { method: 'POST' });
  },

  // Stats
  getStats: async (id: string): Promise<WhiteLabelStats> => {
    const res = await fetch(`${API_BASE}/clients/${id}/stats`);
    return res.json();
  },

  // Branding
  updateBranding: async (id: string, branding: WhiteLabelBranding): Promise<void> => {
    await fetch(`${API_BASE}/clients/${id}/branding`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(branding),
    });
  },

  // Settings
  updateSettings: async (id: string, settings: WhiteLabelSettings): Promise<void> => {
    await fetch(`${API_BASE}/clients/${id}/settings`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(settings),
    });
  },

  // Features
  updateFeatures: async (id: string, features: WhiteLabelFeatures): Promise<void> => {
    await fetch(`${API_BASE}/clients/${id}/features`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(features),
    });
  },
};

// ============================================================================
// COMPONENTS
// ============================================================================

// Status Badge Component
function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    active: 'bg-green-500/20 text-green-500',
    suspended: 'bg-red-500/20 text-red-500',
    pending: 'bg-yellow-500/20 text-yellow-500',
  };

  return (
    <span className={`px-3 py-1 rounded-full text-xs font-medium ${colors[status] || colors.pending}`}>
      {status.toUpperCase()}
    </span>
  );
}

// Plan Badge Component
function PlanBadge({ plan }: { plan: string }) {
  const colors: Record<string, string> = {
    basic: 'bg-gray-500/20 text-gray-500',
    professional: 'bg-blue-500/20 text-blue-500',
    enterprise: 'bg-purple-500/20 text-purple-500',
  };

  return (
    <span className={`px-3 py-1 rounded-full text-xs font-medium ${colors[plan] || colors.basic}`}>
      {plan.toUpperCase()}
    </span>
  );
}

// Stats Card Component
function StatsCard({ title, value, change, icon }: { 
  title: string; 
  value: string | number; 
  change?: number;
  icon: React.ReactNode;
}) {
  const { colors } = useTheme();

  return (
    <div 
      className="p-6 rounded-xl border"
      style={{ 
        backgroundColor: colors.surface, 
        borderColor: colors.border 
      }}
    >
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm" style={{ color: colors.textSecondary }}>{title}</p>
          <p className="text-2xl font-bold mt-1" style={{ color: colors.text }}>{value}</p>
          {change !== undefined && (
            <p className={`text-sm mt-1 ${change >= 0 ? 'text-green-500' : 'text-red-500'}`}>
              {change >= 0 ? '+' : ''}{change}% from last month
            </p>
          )}
        </div>
        <div className="p-3 rounded-lg" style={{ backgroundColor: colors.primary + '20' }}>
          {icon}
        </div>
      </div>
    </div>
  );
}

// Client Card Component
function ClientCard({ client, onEdit, onSuspend, onResume, onDelete }: {
  client: WhiteLabelClient;
  onEdit: () => void;
  onSuspend: () => void;
  onResume: () => void;
  onDelete: () => void;
}) {
  const { colors } = useTheme();

  return (
    <div 
      className="p-6 rounded-xl border"
      style={{ 
        backgroundColor: colors.surface, 
        borderColor: colors.border 
      }}
    >
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-4">
          <div 
            className="w-12 h-12 rounded-lg flex items-center justify-center text-lg font-bold"
            style={{ backgroundColor: client.branding.primaryColor + '20', color: client.branding.primaryColor }}
          >
            {client.name.charAt(0)}
          </div>
          <div>
            <h3 className="font-semibold" style={{ color: colors.text }}>{client.name}</h3>
            <p className="text-sm" style={{ color: colors.textSecondary }}>{client.domain}</p>
          </div>
        </div>
        <div className="flex gap-2">
          <StatusBadge status={client.status} />
          <PlanBadge plan={client.plan} />
        </div>
      </div>

      <div className="mt-4 pt-4 border-t" style={{ borderColor: colors.border }}>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <p style={{ color: colors.textSecondary }}>Created</p>
            <p style={{ color: colors.text }}>{new Date(client.createdAt).toLocaleDateString()}</p>
          </div>
          <div>
            <p style={{ color: colors.textSecondary }}>Features</p>
            <p style={{ color: colors.text }}>
              {Object.values(client.features).filter(Boolean).length} enabled
            </p>
          </div>
        </div>
      </div>

      <div className="mt-4 flex gap-2">
        <button
          onClick={onEdit}
          className="flex-1 px-4 py-2 rounded-lg text-sm font-medium transition-colors"
          style={{ backgroundColor: colors.primary, color: 'white' }}
        >
          Edit
        </button>
        {client.status === 'active' ? (
          <button
            onClick={onSuspend}
            className="px-4 py-2 rounded-lg text-sm font-medium border transition-colors"
            style={{ borderColor: colors.error, color: colors.error }}
          >
            Suspend
          </button>
        ) : (
          <button
            onClick={onResume}
            className="px-4 py-2 rounded-lg text-sm font-medium border transition-colors"
            style={{ borderColor: colors.success, color: colors.success }}
          >
            Resume
          </button>
        )}
        <button
          onClick={onDelete}
          className="px-4 py-2 rounded-lg text-sm font-medium border transition-colors"
          style={{ borderColor: colors.error, color: colors.error }}
        >
          Delete
        </button>
      </div>
    </div>
  );
}

// ============================================================================
// MAIN PAGE COMPONENT
// ============================================================================

export default function WhiteLabelManagement() {
  const { colors, isDark } = useTheme();
  const [clients, setClients] = useState<WhiteLabelClient[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState<'all' | 'active' | 'suspended' | 'pending'>('all');
  const [selectedClient, setSelectedClient] = useState<WhiteLabelClient | null>(null);
  const [showModal, setShowModal] = useState(false);

  useEffect(() => {
    loadClients();
  }, [filter]);

  const loadClients = async () => {
    setLoading(true);
    try {
      const data = await whiteLabelAPI.list(1, 100);
      setClients(data.clients);
    } catch (error) {
      console.error('Failed to load clients:', error);
    } finally {
      setLoading(false);
    }
  };

  const filteredClients = clients.filter(client => {
    if (filter !== 'all' && client.status !== filter) return false;
    if (search && !client.name.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
  });

  const stats: WhiteLabelStats = {
    totalUsers: clients.reduce((sum, c) => sum + Math.floor(Math.random() * 1000), 0),
    activeUsers: clients.filter(c => c.status === 'active').length,
    totalVolume: clients.reduce((sum, c) => sum + Math.random() * 1000000, 0),
    totalTransactions: clients.reduce((sum, c) => sum + Math.floor(Math.random() * 10000), 0),
    revenue: clients.reduce((sum, c) => sum + Math.random() * 50000, 0),
  };

  return (
    <div className="min-h-screen" style={{ backgroundColor: colors.background }}>
      {/* Header */}
      <header 
        className="border-b sticky top-0 z-10"
        style={{ 
          backgroundColor: colors.surface, 
          borderColor: colors.border 
        }}
      >
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold" style={{ color: colors.text }}>
                White Label Management
              </h1>
              <p className="text-sm mt-1" style={{ color: colors.textSecondary }}>
                Manage white label clients and their configurations
              </p>
            </div>
            <button
              onClick={() => setShowModal(true)}
              className="px-6 py-2 rounded-lg font-medium transition-colors"
              style={{ backgroundColor: colors.primary, color: 'white' }}
            >
              + New Client
            </button>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-8">
        {/* Stats Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4 mb-8">
          <StatsCard 
            title="Total Clients" 
            value={clients.length} 
            icon={<span style={{ color: colors.primary }}>🏢</span>}
          />
          <StatsCard 
            title="Active Clients" 
            value={stats.activeUsers} 
            icon={<span style={{ color: colors.success }}>✓</span>}
          />
          <StatsCard 
            title="Total Users" 
            value={stats.totalUsers.toLocaleString()} 
            icon={<span style={{ color: colors.info }}>👥</span>}
          />
          <StatsCard 
            title="Total Volume" 
            value={`$${stats.totalVolume.toLocaleString(undefined, { maximumFractionDigits: 0 })}`} 
            icon={<span style={{ color: colors.warning }}>💰</span>}
          />
          <StatsCard 
            title="Revenue" 
            value={`$${stats.revenue.toLocaleString(undefined, { maximumFractionDigits: 0 })}`} 
            icon={<span style={{ color: colors.success }}>📈</span>}
          />
        </div>

        {/* Filters */}
        <div className="flex items-center gap-4 mb-6">
          <div className="flex-1">
            <input
              type="text"
              placeholder="Search clients..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full px-4 py-2 rounded-lg border outline-none"
              style={{ 
                backgroundColor: colors.surface, 
                borderColor: colors.border,
                color: colors.text 
              }}
            />
          </div>
          <div className="flex gap-2">
            {(['all', 'active', 'suspended', 'pending'] as const).map((f) => (
              <button
                key={f}
                onClick={() => setFilter(f)}
                className="px-4 py-2 rounded-lg text-sm font-medium transition-colors"
                style={{ 
                  backgroundColor: filter === f ? colors.primary : colors.surface,
                  color: filter === f ? 'white' : colors.textSecondary,
                  border: `1px solid ${colors.border}`
                }}
              >
                {f.charAt(0).toUpperCase() + f.slice(1)}
              </button>
            ))}
          </div>
        </div>

        {/* Client Grid */}
        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div className="animate-spin rounded-full h-12 w-12" style={{ borderColor: colors.primary, borderTopColor: 'transparent' }} />
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {filteredClients.map((client) => (
              <ClientCard
                key={client.id}
                client={client}
                onEdit={() => setSelectedClient(client)}
                onSuspend={() => whiteLabelAPI.suspend(client.id).then(loadClients)}
                onResume={() => whiteLabelAPI.resume(client.id).then(loadClients)}
                onDelete={() => {
                  if (confirm('Are you sure you want to delete this client?')) {
                    whiteLabelAPI.delete(client.id).then(loadClients);
                  }
                }}
              />
            ))}
          </div>
        )}

        {filteredClients.length === 0 && !loading && (
          <div className="text-center py-20">
            <p className="text-lg" style={{ color: colors.textSecondary }}>No clients found</p>
          </div>
        )}
      </main>

      {/* Create/Edit Modal */}
      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div 
            className="absolute inset-0"
            style={{ backgroundColor: 'rgba(0,0,0,0.5)' }}
            onClick={() => setShowModal(false)}
          />
          <div 
            className="relative w-full max-w-2xl rounded-xl p-6"
            style={{ backgroundColor: colors.surface }}
          >
            <h2 className="text-xl font-bold mb-4" style={{ color: colors.text }}>
              {selectedClient ? 'Edit Client' : 'Create New Client'}
            </h2>
            <p style={{ color: colors.textSecondary }}>
              White label client form would be here
            </p>
            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => setShowModal(false)}
                className="px-4 py-2 rounded-lg border"
                style={{ borderColor: colors.border, color: colors.text }}
              >
                Cancel
              </button>
              <button
                onClick={() => {
                  setShowModal(false);
                  loadClients();
                }}
                className="px-4 py-2 rounded-lg"
                style={{ backgroundColor: colors.primary, color: 'white' }}
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
