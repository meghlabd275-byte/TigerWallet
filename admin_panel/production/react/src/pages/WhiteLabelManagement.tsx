/**
 * White Label Management - Complete Client Management
 * Connected to backend APIs for real data
 */

import React, { useState, useEffect, useCallback } from 'react';

interface WhiteLabelClient {
  id: string;
  name: string;
  domain: string;
  subdomain?: string;
  status: 'active' | 'suspended' | 'pending' | 'halted' | 'expired';
  authorized: boolean;
  authorizedAt?: string;
  authorizedBy?: string;
  plan: 'starter' | 'professional' | 'enterprise' | 'custom';
  features: Record<string, boolean>;
  branding: Record<string, string>;
  maxUsers: number;
  currentUsers: number;
  revenue: number;
  profitSharePercent: number;
  totalRevenue: number;
  totalProfitShared: number;
  canSell: boolean;
  createdAt: string;
  updatedAt?: string;
}

interface WhiteLabelProduct {
  id: string;
  name: string;
  type: 'spot' | 'perpetual' | 'staking' | 'nft' | 'wallet';
  status: 'enabled' | 'disabled' | 'maintenance';
  fee: number;
  minDeposit: number;
  maxDeposit: number;
}

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

// Super Admin API endpoints
const SUPER_ADMIN_API = {
  clients: `${API_BASE_URL}/super-admin/clients`,
  clientDetails: (id: string) => `${API_BASE_URL}/super-admin/clients/${id}`,
  authorizeClient: (id: string) => `${API_BASE_URL}/super-admin/clients/${id}/authorize`,
  suspendClient: (id: string) => `${API_BASE_URL}/super-admin/clients/${id}/suspend`,
  resumeClient: (id: string) => `${API_BASE_URL}/super-admin/clients/${id}/resume`,
  haltClient: (id: string) => `${API_BASE_URL}/super-admin/clients/${id}/halt`,
  deleteClient: (id: string) => `${API_BASE_URL}/super-admin/clients/${id}`,
  products: `${API_BASE_URL}/super-admin/products`,
  revenue: `${API_BASE_URL}/super-admin/revenue`,
  dashboard: `${API_BASE_URL}/super-admin/dashboard`,
};

function WhiteLabelManagement() {
  const [activeTab, setActiveTab] = useState('clients');
  const [showCreate, setShowCreate] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [clients, setClients] = useState<WhiteLabelClient[]>([]);
  const [products, setProducts] = useState<WhiteLabelProduct[]>([]);
  const [selectedClient, setSelectedClient] = useState<WhiteLabelClient | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    domain: '',
    subdomain: '',
    plan: 'starter',
    adminEmail: '',
    adminName: '',
  });

  // Fetch clients from backend (Super Admin API)
  const fetchClients = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const token = localStorage.getItem('superadmin_token');
      const response = await fetch(SUPER_ADMIN_API.clients, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error(`Failed to fetch clients: ${response.statusText}`);
      }
      
      const data = await response.json();
      setClients(data.clients || []);
    } catch (err) {
      console.error('Error fetching clients:', err);
      setError(err instanceof Error ? err.message : 'Failed to load clients');
      // Fallback: empty array - no mock data
      setClients([]);
    } finally {
      setLoading(false);
    }
  }, []);

  // Fetch products from backend (Super Admin API)
  const fetchProducts = useCallback(async () => {
    try {
      const token = localStorage.getItem('superadmin_token');
      const response = await fetch(SUPER_ADMIN_API.products, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error(`Failed to fetch products: ${response.statusText}`);
      }
      
      const data = await response.json();
      setProducts(data.products || []);
    } catch (err) {
      console.error('Error fetching products:', err);
      setProducts([]);
    }
  }, []);

  // Load data on mount
  useEffect(() => {
    fetchClients();
    fetchProducts();
  }, [fetchClients, fetchProducts]);

  // Create new client (Super Admin)
  const handleCreateClient = async () => {
    setLoading(true);
    try {
      const token = localStorage.getItem('superadmin_token');
      const response = await fetch(SUPER_ADMIN_API.clients, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: formData.name,
          domain: formData.domain,
          subdomain: formData.subdomain,
          plan: formData.plan,
          adminEmail: formData.adminEmail,
          adminName: formData.adminName,
        }),
      });
      
      if (!response.ok) {
        throw new Error(`Failed to create client: ${response.statusText}`);
      }
      
      await fetchClients();
      setShowCreate(false);
      setFormData({ name: '', domain: '', subdomain: '', plan: 'starter', adminEmail: '', adminName: '' });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create client');
    } finally {
      setLoading(false);
    }
  };

  // Authorize client (Super Admin only)
  const handleAuthorizeClient = async (clientId: string, authorized: boolean) => {
    try {
      const token = localStorage.getItem('superadmin_token');
      const response = await fetch(SUPER_ADMIN_API.authorizeClient(clientId), {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ authorized }),
      });
      
      if (!response.ok) {
        throw new Error(`Failed to ${authorized ? 'authorize' : 'deauthorize'} client`);
      }
      
      await fetchClients();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Authorization failed');
    }
  };

  // Suspend/Resume client
  const handleSuspendClient = async (clientId: string, status: 'suspended' | 'active') => {
    try {
      const token = localStorage.getItem('superadmin_token');
      const endpoint = status === 'suspended' 
        ? SUPER_ADMIN_API.suspendClient(clientId)
        : SUPER_ADMIN_API.resumeClient(clientId);
      
      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error(`Failed to ${status} client`);
      }
      
      await fetchClients();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Status update failed');
    }
  };

  const features = [
    { name: 'Custom Branding', enabled: true },
    { name: 'Custom Domain', enabled: true },
    { name: 'White Label Wallet', enabled: true },
    { name: 'Custom Trading Pairs', enabled: true },
    { name: 'API Access', enabled: true },
    { name: 'Dedicated Support', enabled: false },
  ];

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">White Label Management</h1>
        <button onClick={() => setShowCreate(true)} className="btn btn-primary">
          + Create New Client
        </button>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 mb-6">
        {['Clients', 'Plans', 'Features', 'Analytics'].map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab.toLowerCase())}
            className={`px-4 py-2 rounded-lg ${
              activeTab === tab.toLowerCase() ? 'bg-amber-500 text-black' : 'bg-slate-800'
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Loading/Error State */}
      {loading && (
        <div className="flex justify-center items-center py-12">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-amber-500"></div>
        </div>
      )}

      {error && (
        <div className="bg-red-500/20 border border-red-500 text-red-500 px-4 py-3 rounded-lg mb-4">
          {error}
        </div>
      )}

      {/* Clients Tab */}
      {activeTab === 'clients' && !loading && (
        <div className="bg-slate-800 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-slate-700">
              <tr>
                <th className="p-3 text-left">ID</th>
                <th className="p-3 text-left">Client Name</th>
                <th className="p-3 text-left">Domain</th>
                <th className="p-3 text-left">Status</th>
                <th className="p-3 text-left">Authorized</th>
                <th className="p-3 text-left">Users</th>
                <th className="p-3 text-left">Revenue</th>
                <th className="p-3 text-left">Profit Share</th>
                <th className="p-3 text-left">Plan</th>
                <th className="p-3 text-left">Actions</th>
              </tr>
            </thead>
            <tbody>
              {clients.length === 0 ? (
                <tr>
                  <td colSpan={10} className="p-8 text-center text-gray-500">
                    No white label clients found
                  </td>
                </tr>
              ) : (
                clients.map((client) => (
                  <tr key={client.id} className="border-t border-slate-700">
                    <td className="p-3">#{client.id.substring(0, 8)}</td>
                    <td className="p-3 font-semibold">{client.name}</td>
                    <td className="p-3 font-mono text-sm">{client.domain}</td>
                    <td className="p-3">
                      <span className={`px-2 py-1 rounded text-xs ${
                        client.status === 'active' ? 'bg-green-500/20 text-green-500' : 
                        client.status === 'suspended' ? 'bg-red-500/20 text-red-500' :
                        client.status === 'pending' ? 'bg-yellow-500/20 text-yellow-500' :
                        'bg-gray-500/20 text-gray-500'
                      }`}>
                        {client.status}
                      </span>
                    </td>
                    <td className="p-3">
                      <span className={`px-2 py-1 rounded text-xs ${
                        client.authorized ? 'bg-blue-500/20 text-blue-500' : 'bg-gray-500/20 text-gray-500'
                      }`}>
                        {client.authorized ? '✓ Authorized' : '⏳ Pending'}
                      </span>
                    </td>
                    <td className="p-3">{(client.currentUsers || 0).toLocaleString()} / {(client.maxUsers || 0).toLocaleString()}</td>
                    <td className="p-3">${(client.totalRevenue || client.revenue || 0).toLocaleString()}</td>
                    <td className="p-3 text-amber-500">{client.profitSharePercent || 20}%</td>
                    <td className="p-3 text-amber-500">{client.plan}</td>
                    <td className="p-3">
                      <div className="flex gap-2">
                        <button 
                          onClick={() => setSelectedClient(client)}
                          className="btn btn-secondary text-xs"
                        >
                          Edit
                        </button>
                        <button 
                          onClick={() => handleAuthorizeClient(client.id, !client.authorized)}
                          className={`btn text-xs ${client.authorized ? 'btn-danger' : 'btn-primary'}`}
                        >
                          {client.authorized ? 'Deauthorize' : 'Authorize'}
                        </button>
                        <button 
                          onClick={() => handleSuspendClient(client.id, client.status === 'suspended' ? 'active' : 'suspended')}
                          className="btn btn-danger text-xs"
                        >
                          {client.status === 'suspended' ? 'Resume' : 'Suspend'}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {/* Plans Tab */}
      {activeTab === 'plans' && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {['Starter', 'Business', 'Enterprise'].map((plan, i) => (
            <div key={plan} className="bg-slate-800 p-6 rounded-lg">
              <h3 className="text-xl font-bold mb-2">{plan}</h3>
              <p className="text-3xl font-bold text-amber-500 mb-4">
                ${(i + 1) * 999}<span className="text-sm opacity-60">/mo</span>
              </p>
              <ul className="space-y-2 mb-6">
                <li>✓ Up to {((i + 1) * 5000).toLocaleString()} users</li>
                <li>✓ Custom branding</li>
                <li>✓ Custom domain</li>
                <li>✓ {i + 1} Admin accounts</li>
                {i > 0 && <li>✓ API access</li>}
                {i > 1 && <li>✓ Dedicated support</li>}
              </ul>
              <button className="btn btn-primary w-full">Edit Plan</button>
            </div>
          ))}
        </div>
      )}

      {/* Features Tab */}
      {activeTab === 'features' && (
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Platform Features</h3>
          <div className="grid grid-cols-2 gap-4">
            {features.map((feature, i) => (
              <div key={i} className="flex justify-between items-center p-3 bg-slate-700 rounded-lg">
                <span>{feature.name}</span>
                <button className={`w-12 h-6 rounded-full ${
                  feature.enabled ? 'bg-amber-500' : 'bg-slate-600'
                }`}>
                  <div className={`w-5 h-5 bg-white rounded-full transition ${
                    feature.enabled ? 'translate-x-6' : 'translate-x-0.5'
                  }`} />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Create Modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-slate-800 p-6 rounded-lg w-full max-w-lg">
            <h2 className="text-xl font-bold mb-4">Create New White Label Client</h2>
            <div className="space-y-4">
              <div>
                <label className="label">Client Name</label>
                <input 
                  type="text" 
                  className="input" 
                  placeholder="Company Name"
                  value={formData.name}
                  onChange={(e) => setFormData({...formData, name: e.target.value})}
                />
              </div>
              <div>
                <label className="label">Domain</label>
                <input 
                  type="text" 
                  className="input" 
                  placeholder="yourdomain.com"
                  value={formData.domain}
                  onChange={(e) => setFormData({...formData, domain: e.target.value})}
                />
              </div>
              <div>
                <label className="label">Subdomain (optional)</label>
                <input 
                  type="text" 
                  className="input" 
                  placeholder="client"
                  value={formData.subdomain}
                  onChange={(e) => setFormData({...formData, subdomain: e.target.value})}
                />
              </div>
              <div>
                <label className="label">Plan</label>
                <select 
                  className="input"
                  value={formData.plan}
                  onChange={(e) => setFormData({...formData, plan: e.target.value})}
                >
                  <option value="starter">Starter</option>
                  <option value="professional">Professional</option>
                  <option value="enterprise">Enterprise</option>
                  <option value="custom">Custom</option>
                </select>
              </div>
              <div>
                <label className="label">Admin Email</label>
                <input 
                  type="email" 
                  className="input" 
                  placeholder="admin@company.com"
                  value={formData.adminEmail}
                  onChange={(e) => setFormData({...formData, adminEmail: e.target.value})}
                />
              </div>
              <div>
                <label className="label">Admin Name</label>
                <input 
                  type="text" 
                  className="input" 
                  placeholder="John Doe"
                  value={formData.adminName}
                  onChange={(e) => setFormData({...formData, adminName: e.target.value})}
                />
              </div>
            </div>
            <div className="flex gap-2 mt-6">
              <button onClick={() => setShowCreate(false)} className="btn btn-secondary flex-1">Cancel</button>
              <button 
                onClick={handleCreateClient} 
                disabled={loading || !formData.name || !formData.domain || !formData.adminEmail}
                className="btn btn-primary flex-1"
              >
                {loading ? 'Creating...' : 'Create Client'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default WhiteLabelManagement;
