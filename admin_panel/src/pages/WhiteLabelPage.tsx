// White Label Management Page
// Complete white label client and product management with API integration

import React, { useState, useEffect, useCallback } from 'react';
import './WhiteLabel.css';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

interface WhiteLabelClient {
  id: string;
  name: string;
  domain: string;
  status: 'active' | 'suspended' | 'pending' | 'halted';
  authorized: boolean;
  plan: 'starter' | 'professional' | 'enterprise' | 'custom';
  createdAt: string;
  revenue: number;
  users: number;
  maxUsers: number;
  profitSharePercent: number;
}

const WhiteLabelPage: React.FC = () => {
  const [clients, setClients] = useState<WhiteLabelClient[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showModal, setShowModal] = useState(false);
  const [selectedClient, setSelectedClient] = useState<WhiteLabelClient | null>(null);
  const [activeTab, setActiveTab] = useState<'clients' | 'products' | 'wallets' | 'blockchains'>('clients');

  // Fetch clients from backend
  const fetchClients = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const token = localStorage.getItem('superadmin_token');
      const response = await fetch(`${API_BASE_URL}/super-admin/clients`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error(`Failed to fetch clients: ${response.statusText}`);
      }
      
      const data = await response.json();
      // Transform backend data to frontend format
      const transformedClients = (data.clients || []).map((c: any) => ({
        id: c.id,
        name: c.name,
        domain: c.domain,
        status: c.status,
        authorized: c.authorized || false,
        plan: c.plan || 'starter',
        createdAt: c.createdAt,
        revenue: c.totalRevenue || c.revenue || 0,
        users: c.currentUsers || 0,
        maxUsers: c.maxUsers || 0,
        profitSharePercent: c.profitSharePercent || 20,
      }));
      setClients(transformedClients);
    } catch (err) {
      console.error('Error fetching clients:', err);
      setError(err instanceof Error ? err.message : 'Failed to load clients');
      setClients([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchClients();
  }, [fetchClients]);

  const handleStatusChange = async (clientId: string, newStatus: 'active' | 'suspended' | 'halted') => {
    try {
      const token = localStorage.getItem('superadmin_token');
      const endpoint = newStatus === 'suspended' 
        ? `${API_BASE_URL}/super-admin/clients/${clientId}/suspend`
        : `${API_BASE_URL}/super-admin/clients/${clientId}/resume`;
      
      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to update status');
      }
      
      await fetchClients();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Status update failed');
    }
  };

  const handleAuthorizeClient = async (clientId: string, authorized: boolean) => {
    try {
      const token = localStorage.getItem('superadmin_token');
      const response = await fetch(`${API_BASE_URL}/super-admin/clients/${clientId}/authorize`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ authorized }),
      });
      
      if (!response.ok) {
        throw new Error('Authorization failed');
      }
      
      await fetchClients();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Authorization failed');
    }
  };

  const handleDeleteClient = async (clientId: string) => {
    if (confirm('Are you sure you want to delete this white label client? This action cannot be undone.')) {
      try {
        const token = localStorage.getItem('superadmin_token');
        const response = await fetch(`${API_BASE_URL}/super-admin/clients/${clientId}`, {
          method: 'DELETE',
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        });
        
        if (!response.ok) {
          throw new Error('Delete failed');
        }
        
        await fetchClients();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Delete failed');
      }
    }
  };

  return (
    <div className="whitelabel-page">
      <div className="page-header">
        <div>
          <h1>White Label Management</h1>
          <p>Manage white label clients, products, and services</p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowModal(true)}>
          + Create New Client
        </button>
      </div>

      {/* Tabs */}
      <div className="tabs">
        <button 
          className={`tab ${activeTab === 'clients' ? 'active' : ''}`}
          onClick={() => setActiveTab('clients')}
        >
          Clients ({clients.length})
        </button>
        <button 
          className={`tab ${activeTab === 'products' ? 'active' : ''}`}
          onClick={() => setActiveTab('products')}
        >
          Products
        </button>
        <button 
          className={`tab ${activeTab === 'wallets' ? 'active' : ''}`}
          onClick={() => setActiveTab('wallets')}
        >
          Wallets
        </button>
        <button 
          className={`tab ${activeTab === 'blockchains' ? 'active' : ''}`}
          onClick={() => setActiveTab('blockchains')}
        >
          Blockchains
        </button>
      </div>

      {/* Clients Tab */}
      {activeTab === 'clients' && (
        <div className="clients-list">
          <div className="table-container">
            <table>
              <thead>
                <tr>
                  <th>Client</th>
                  <th>Domain</th>
                  <th>Plan</th>
                  <th>Users</th>
                  <th>Revenue</th>
                  <th>Status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {clients.map(client => (
                  <tr key={client.id}>
                    <td>
                      <div className="client-info">
                        <span className="client-avatar">{client.name.charAt(0)}</span>
                        <span className="client-name">{client.name}</span>
                      </div>
                    </td>
                    <td>
                      <a href={`https://${client.domain}`} target="_blank" rel="noopener" className="client-domain">
                        {client.domain}
                      </a>
                    </td>
                    <td>
                      <span className={`plan-badge plan-${client.plan}`}>
                        {client.plan.toUpperCase()}
                      </span>
                    </td>
                    <td>{client.users.toLocaleString()}</td>
                    <td>${client.revenue.toLocaleString()}</td>
                    <td>
                      <span className={`status-badge status-${client.status}`}>
                        {client.status}
                      </span>
                    </td>
                    <td>
                      <div className="actions">
                        <button 
                          className="action-btn"
                          onClick={() => setSelectedClient(client)}
                          title="Edit"
                        >
                          ✏️
                        </button>
                        <button 
                          className="action-btn"
                          onClick={() => handleStatusChange(client.id, client.status === 'active' ? 'paused' : 'active')}
                          title={client.status === 'active' ? 'Pause' : 'Activate'}
                        >
                          {client.status === 'active' ? '⏸️' : '▶️'}
                        </button>
                        <button 
                          className="action-btn danger"
                          onClick={() => handleDeleteClient(client.id)}
                          title="Delete"
                        >
                          🗑️
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Products Tab */}
      {activeTab === 'products' && (
        <div className="products-grid">
          <div className="product-card">
            <div className="product-icon">💳</div>
            <h3>Virtual Token Management</h3>
            <p>Create and manage virtual tokens for white label clients</p>
            <span className="product-status enabled">Enabled</span>
          </div>
          <div className="product-card">
            <div className="product-icon">🤖</div>
            <h3>Market Maker Bots</h3>
            <p>Automated market making for trading pairs</p>
            <span className="product-status enabled">Enabled</span>
          </div>
          <div className="product-card">
            <div className="product-icon">📋</div>
            <h3>Listing Management</h3>
            <p>Token listing and delisting management</p>
            <span className="product-status enabled">Enabled</span>
          </div>
          <div className="product-card">
            <div className="product-icon">💼</div>
            <h3>CEX/DEX</h3>
            <p>Centralized and decentralized exchange features</p>
            <span className="product-status enabled">Enabled</span>
          </div>
          <div className="product-card">
            <div className="product-icon">🏦</div>
            <h3>Brokerage</h3>
            <p>Brokerage and institutional client management</p>
            <span className="product-status disabled">Disabled</span>
          </div>
          <div className="product-card">
            <div className="product-icon">🎨</div>
            <h3>NFT Management</h3>
            <p>NFT creation and marketplace features</p>
            <span className="product-status enabled">Enabled</span>
          </div>
        </div>
      )}

      {/* Wallets Tab */}
      {activeTab === 'wallets' && (
        <div className="wallets-section">
          <div className="card">
            <h3>Web3 Wallet Configuration</h3>
            <p>Configure wallet features for white label clients</p>
            <div className="feature-list">
              <label className="feature-item">
                <input type="checkbox" defaultChecked />
                <span>24-word seed phrase support</span>
              </label>
              <label className="feature-item">
                <input type="checkbox" defaultChecked />
                <span>Multi-chain support (100+ chains)</span>
              </label>
              <label className="feature-item">
                <input type="checkbox" defaultChecked />
                <span>Swap functionality</span>
              </label>
              <label className="feature-item">
                <input type="checkbox" defaultChecked />
                <span>Staking</span>
              </label>
              <label className="feature-item">
                <input type="checkbox" defaultChecked />
                <span>NFT support</span>
              </label>
              <label className="feature-item">
                <input type="checkbox" defaultChecked />
                <span>DApp browser</span>
              </label>
            </div>
          </div>
        </div>
      )}

      {/* Blockchains Tab */}
      {activeTab === 'blockchains' && (
        <div className="blockchains-section">
          <div className="card">
            <h3>Blockchain Networks</h3>
            <p>Manage available blockchain networks for white label clients</p>
            <div className="chain-list">
              {['Ethereum', 'BNB Chain', 'Polygon', 'Arbitrum', 'Optimism', 'Base', 'Avalanche', 'Solana', 'Aptos', 'Sui'].map(chain => (
                <div key={chain} className="chain-item">
                  <span className="chain-name">{chain}</span>
                  <label className="toggle">
                    <input type="checkbox" defaultChecked />
                    <span className="toggle-slider"></span>
                  </label>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Create Client Modal */}
      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h2>Create White Label Client</h2>
              <button className="close-btn" onClick={() => setShowModal(false)}>×</button>
            </div>
            <div className="modal-body">
              <div className="form-group">
                <label className="form-label">Client Name</label>
                <input type="text" className="form-input" placeholder="Enter client name" />
              </div>
              <div className="form-group">
                <label className="form-label">Domain</label>
                <input type="text" className="form-input" placeholder="wallet.example.com" />
              </div>
              <div className="form-group">
                <label className="form-label">Plan</label>
                <select className="form-input">
                  <option value="basic">Basic</option>
                  <option value="pro">Pro</option>
                  <option value="enterprise">Enterprise</option>
                </select>
              </div>
              <div className="form-group">
                <label className="form-label">Revenue Share (%)</label>
                <input type="number" className="form-input" placeholder="20" />
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
              <button className="btn btn-primary" onClick={() => setShowModal(false)}>Create Client</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default WhiteLabelPage;
