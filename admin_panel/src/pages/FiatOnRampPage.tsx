// Fiat On-Ramp Management Admin Page
// Manage fiat to crypto purchases, payment providers, and transactions

import React, { useState, useEffect } from 'react';
import './FiatOnRampPage.css';

interface FiatOrder {
  id: string;
  orderId: string;
  userId: string;
  username: string;
  crypto: string;
  cryptoAmount: number;
  fiatAmount: number;
  fiatCurrency: string;
  paymentMethod: string;
  provider: string;
  status: 'pending' | 'processing' | 'completed' | 'failed' | 'cancelled' | 'refunded';
  kycStatus: 'none' | 'pending' | 'approved' | 'rejected';
  createdAt: string;
  completedAt?: string;
  failureReason?: string;
}

interface PaymentProvider {
  id: string;
  name: string;
  logo: string;
  supportedFiat: string[];
  supportedCrypto: string[];
  fees: {
    fixed: number;
    percentage: number;
  };
  limits: {
    min: number;
    max: number;
  };
  status: 'active' | 'inactive' | 'maintenance';
  countries: string[];
}

interface FiatSettings {
  enabled: boolean;
  defaultProvider: string;
  autoApproval: boolean;
  kycRequired: boolean;
  minOrderAmount: number;
  maxOrderAmount: number;
}

const FiatOnRampPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'orders' | 'providers' | 'settings'>('orders');
  const [orders, setOrders] = useState<FiatOrder[]>([]);
  const [providers, setProviders] = useState<PaymentProvider[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [filterProvider, setFilterProvider] = useState<string>('all');
  const [showOrderModal, setShowOrderModal] = useState(false);
  const [selectedOrder, setSelectedOrder] = useState<FiatOrder | null>(null);

  const [fiatSettings, setFiatSettings] = useState<FiatSettings>({
    enabled: true,
    defaultProvider: 'stripe',
    autoApproval: false,
    kycRequired: true,
    minOrderAmount: 30,
    maxOrderAmount: 50000,
  });

  // Initialize with sample data
  useEffect(() => {
    // Sample orders
    const sampleOrders: FiatOrder[] = [
      { id: '1', orderId: 'FIAT-2024-001', userId: 'u1', username: 'CryptoNewbie', crypto: 'BTC', cryptoAmount: 0.025, fiatAmount: 1050, fiatCurrency: 'USD', paymentMethod: 'Credit Card', provider: 'Stripe', status: 'completed', kycStatus: 'approved', createdAt: '2024-01-15 10:30:00', completedAt: '2024-01-15 10:35:00' },
      { id: '2', orderId: 'FIAT-2024-002', userId: 'u2', username: 'BitcoinFan', crypto: 'ETH', cryptoAmount: 1.5, fiatAmount: 3450, fiatCurrency: 'EUR', paymentMethod: 'SEPA Transfer', provider: 'MoonPay', status: 'processing', kycStatus: 'approved', createdAt: '2024-01-15 09:15:00' },
      { id: '3', orderId: 'FIAT-2024-003', userId: 'u3', username: 'TokenTrader', crypto: 'USDT', cryptoAmount: 5000, fiatAmount: 5000, fiatCurrency: 'USD', paymentMethod: 'Bank Wire', provider: 'Transak', status: 'pending', kycStatus: 'pending', createdAt: '2024-01-15 08:00:00' },
      { id: '4', orderId: 'FIAT-2024-004', userId: 'u4', username: 'WhaleAlert', crypto: 'BTC', cryptoAmount: 0.1, fiatAmount: 4200, fiatCurrency: 'GBP', paymentMethod: 'Faster Payments', provider: 'Stripe', status: 'failed', kycStatus: 'approved', createdAt: '2024-01-14 16:45:00', failureReason: 'Payment declined by bank' },
      { id: '5', orderId: 'FIAT-2024-005', userId: 'u5', username: 'AltcoinKing', crypto: 'SOL', cryptoAmount: 50, fiatAmount: 5250, fiatCurrency: 'USD', paymentMethod: 'Apple Pay', provider: 'MoonPay', status: 'completed', kycStatus: 'approved', createdAt: '2024-01-14 14:20:00', completedAt: '2024-01-14 14:25:00' },
      { id: '6', orderId: 'FIAT-2024-006', userId: 'u6', username: 'DefiUser', crypto: 'USDC', cryptoAmount: 1000, fiatAmount: 1000, fiatCurrency: 'EUR', paymentMethod: 'Giropay', provider: 'Transak', status: 'cancelled', kycStatus: 'none', createdAt: '2024-01-14 12:00:00' },
    ];

    // Generate more orders
    const cryptos = ['BTC', 'ETH', 'USDT', 'USDC', 'BNB', 'SOL', 'XRP', 'ADA'];
    const fiatCurrencies = ['USD', 'EUR', 'GBP', 'AUD', 'CAD'];
    const paymentMethods = ['Credit Card', 'Debit Card', 'SEPA Transfer', 'Bank Wire', 'Faster Payments', 'Apple Pay', 'Google Pay'];
    const providersList = ['Stripe', 'MoonPay', 'Transak', 'Simplex'];
    const statuses: FiatOrder['status'][] = ['pending', 'processing', 'completed', 'failed', 'cancelled'];
    const kycStatuses: FiatOrder['kycStatus'][] = ['none', 'pending', 'approved', 'rejected'];

    for (let i = 7; i <= 100; i++) {
      const crypto = cryptos[Math.floor(Math.random() * cryptos.length)];
      const fiatAmount = Math.floor(Math.random() * 4900) + 100;
      const cryptoAmount = crypto === 'BTC' ? fiatAmount / 42000 :
                          crypto === 'ETH' ? fiatAmount / 2300 :
                          crypto === 'USDT' || crypto === 'USDC' ? fiatAmount :
                          fiatAmount / 100;

      const status = statuses[Math.floor(Math.random() * statuses.length)];
      
      sampleOrders.push({
        id: String(i),
        orderId: `FIAT-2024-${String(i).padStart(3, '0')}`,
        userId: `u${i}`,
        username: `User${i}`,
        crypto,
        cryptoAmount,
        fiatAmount,
        fiatCurrency: fiatCurrencies[Math.floor(Math.random() * fiatCurrencies.length)],
        paymentMethod: paymentMethods[Math.floor(Math.random() * paymentMethods.length)],
        provider: providersList[Math.floor(Math.random() * providersList.length)],
        status,
        kycStatus: kycStatuses[Math.floor(Math.random() * kycStatuses.length)],
        createdAt: `2024-01-${String(10 + Math.floor(i / 10)).padStart(2, '0')} ${String(Math.floor(Math.random() * 24)).padStart(2, '0')}:${String(Math.floor(Math.random() * 60)).padStart(2, '0')}:00`,
        completedAt: status === 'completed' ? `2024-01-${String(10 + Math.floor(i / 10)).padStart(2, '0')} ${String(Math.floor(Math.random() * 24)).padStart(2, '0')}:${String(Math.floor(Math.random() * 60)).padStart(2, '0')}:00` : undefined,
      });
    }
    setOrders(sampleOrders);

    // Sample providers
    const sampleProviders: PaymentProvider[] = [
      { id: 'stripe', name: 'Stripe', logo: '💳', supportedFiat: ['USD', 'EUR', 'GBP'], supportedCrypto: ['BTC', 'ETH', 'USDT', 'USDC'], fees: { fixed: 0, percentage: 2.5 }, limits: { min: 30, max: 50000 }, status: 'active', countries: ['US', 'EU', 'UK'] },
      { id: 'moonpay', name: 'MoonPay', logo: '🌙', supportedFiat: ['USD', 'EUR', 'GBP', 'AUD'], supportedCrypto: ['BTC', 'ETH', 'USDT', 'USDC', 'BNB', 'SOL'], fees: { fixed: 0, percentage: 4.5 }, limits: { min: 20, max: 25000 }, status: 'active', countries: ['US', 'EU', 'UK', 'AU'] },
      { id: 'transak', name: 'Transak', logo: '🔄', supportedFiat: ['USD', 'EUR', 'GBP', 'INR'], supportedCrypto: ['BTC', 'ETH', 'USDT', 'USDC', 'MATIC'], fees: { fixed: 0, percentage: 3.5 }, limits: { min: 50, max: 10000 }, status: 'active', countries: ['US', 'EU', 'IN'] },
      { id: 'simplex', name: 'Simplex', logo: '⚡', supportedFiat: ['USD', 'EUR'], supportedCrypto: ['BTC', 'ETH', 'USDT', 'USDC', 'BNB'], fees: { fixed: 0, percentage: 3.9 }, limits: { min: 50, max: 20000 }, status: 'inactive', countries: ['US', 'EU'] },
      { id: 'banxa', name: 'Banxa', logo: '🏦', supportedFiat: ['USD', 'EUR', 'AUD', 'CAD'], supportedCrypto: ['BTC', 'ETH', 'USDT', 'USDC'], fees: { fixed: 0, percentage: 2.8 }, limits: { min: 100, max: 50000 }, status: 'active', countries: ['US', 'EU', 'AU', 'CA'] },
    ];
    setProviders(sampleProviders);
  }, []);

  const filteredOrders = orders.filter(order => {
    if (filterStatus !== 'all' && order.status !== filterStatus) return false;
    if (filterProvider !== 'all' && order.provider.toLowerCase() !== filterProvider) return false;
    if (searchTerm && !order.orderId.toLowerCase().includes(searchTerm.toLowerCase()) &&
        !order.username.toLowerCase().includes(searchTerm.toLowerCase()) &&
        !order.crypto.toLowerCase().includes(searchTerm.toLowerCase())) return false;
    return true;
  });

  const handleProcessOrder = (orderId: string) => {
    setOrders(orders.map(o => 
      o.id === orderId ? { ...o, status: 'completed' as const, completedAt: new Date().toLocaleString() } : o
    ));
  };

  const handleCancelOrder = (orderId: string) => {
    setOrders(orders.map(o => 
      o.id === orderId ? { ...o, status: 'cancelled' as const } : o
    ));
  };

  const handleToggleProvider = (providerId: string) => {
    setProviders(providers.map(p => 
      p.id === providerId ? { ...p, status: p.status === 'active' ? 'inactive' as const : 'active' as const } : p
    ));
  };

  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      pending: '#ffc107',
      processing: '#17a2b8',
      completed: '#28a745',
      failed: '#dc3545',
      cancelled: '#6c757d',
      refunded: '#17a2b8',
      active: '#28a745',
      inactive: '#6c757d',
      maintenance: '#ffc107',
      none: '#6c757d',
      approved: '#28a745',
      rejected: '#dc3545',
    };
    return colors[status] || '#6c757d';
  };

  // Stats
  const stats = {
    totalOrders: orders.length,
    pendingOrders: orders.filter(o => o.status === 'pending').length,
    completedOrders: orders.filter(o => o.status === 'completed').length,
    totalVolume: orders.filter(o => o.status === 'completed').reduce((sum, o) => sum + o.fiatAmount, 0),
    successRate: ((orders.filter(o => o.status === 'completed').length / orders.length) * 100).toFixed(1),
  };

  return (
    <div className="fiat-onramp-page">
      <div className="page-header">
        <h1>Fiat On-Ramp Management</h1>
        <div className="header-actions">
          <button className="export-btn">Export Data</button>
        </div>
      </div>

      <div className="stats-cards">
        <div className="stat-card">
          <div className="stat-icon">📋</div>
          <div className="stat-info">
            <span className="stat-value">{stats.totalOrders}</span>
            <span className="stat-label">Total Orders</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">⏳</div>
          <div className="stat-info">
            <span className="stat-value">{stats.pendingOrders}</span>
            <span className="stat-label">Pending</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">✅</div>
          <div className="stat-info">
            <span className="stat-value">{stats.completedOrders}</span>
            <span className="stat-label">Completed</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">💰</div>
          <div className="stat-info">
            <span className="stat-value">${(stats.totalVolume / 1000).toFixed(1)}K</span>
            <span className="stat-label">Total Volume</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">📈</div>
          <div className="stat-info">
            <span className="stat-value">{stats.successRate}%</span>
            <span className="stat-label">Success Rate</span>
          </div>
        </div>
      </div>

      <div className="tabs">
        <button 
          className={activeTab === 'orders' ? 'active' : ''} 
          onClick={() => setActiveTab('orders')}
        >
          📋 Orders ({orders.length})
        </button>
        <button 
          className={activeTab === 'providers' ? 'active' : ''} 
          onClick={() => setActiveTab('providers')}
        >
          🏦 Providers ({providers.length})
        </button>
        <button 
          className={activeTab === 'settings' ? 'active' : ''} 
          onClick={() => setActiveTab('settings')}
        >
          ⚙️ Settings
        </button>
      </div>

      {activeTab === 'orders' && (
        <div className="orders-section">
          <div className="filters">
            <input
              type="text"
              placeholder="Search orders..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="search-input"
            />
            <select 
              value={filterStatus} 
              onChange={(e) => setFilterStatus(e.target.value)}
              className="filter-select"
            >
              <option value="all">All Status</option>
              <option value="pending">Pending</option>
              <option value="processing">Processing</option>
              <option value="completed">Completed</option>
              <option value="failed">Failed</option>
              <option value="cancelled">Cancelled</option>
            </select>
            <select 
              value={filterProvider} 
              onChange={(e) => setFilterProvider(e.target.value)}
              className="filter-select"
            >
              <option value="all">All Providers</option>
              <option value="stripe">Stripe</option>
              <option value="moonpay">MoonPay</option>
              <option value="transak">Transak</option>
              <option value="simplex">Simplex</option>
            </select>
          </div>

          <div className="orders-table">
            <table>
              <thead>
                <tr>
                  <th>Order ID</th>
                  <th>User</th>
                  <th>Crypto</th>
                  <th>Crypto Amount</th>
                  <th>Fiat Amount</th>
                  <th>Payment</th>
                  <th>Provider</th>
                  <th>KYC</th>
                  <th>Status</th>
                  <th>Created</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {filteredOrders.slice(0, 50).map(order => (
                  <tr key={order.id}>
                    <td className="order-id">{order.orderId}</td>
                    <td>
                      <div className="user-info">
                        <span className="username">{order.username}</span>
                        <span className="user-id">{order.userId}</span>
                      </div>
                    </td>
                    <td>
                      <span className="crypto-badge">{order.crypto}</span>
                    </td>
                    <td>{order.cryptoAmount.toFixed(6)}</td>
                    <td className="fiat-amount">
                      {order.fiatAmount.toLocaleString()} {order.fiatCurrency}
                    </td>
                    <td>{order.paymentMethod}</td>
                    <td>
                      <span className="provider-badge">{order.provider}</span>
                    </td>
                    <td>
                      <span 
                        className="status-badge small"
                        style={{ backgroundColor: getStatusColor(order.kycStatus) }}
                      >
                        {order.kycStatus}
                      </span>
                    </td>
                    <td>
                      <span 
                        className="status-badge"
                        style={{ backgroundColor: getStatusColor(order.status) }}
                      >
                        {order.status}
                      </span>
                    </td>
                    <td>{order.createdAt}</td>
                    <td>
                      <div className="actions">
                        <button 
                          className="action-btn view"
                          onClick={() => { setSelectedOrder(order); setShowOrderModal(true); }}
                        >
                          View
                        </button>
                        {order.status === 'processing' && (
                          <button 
                            className="action-btn complete"
                            onClick={() => handleProcessOrder(order.id)}
                          >
                            Complete
                          </button>
                        )}
                        {['pending', 'processing'].includes(order.status) && (
                          <button 
                            className="action-btn cancel"
                            onClick={() => handleCancelOrder(order.id)}
                          >
                            Cancel
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="pagination">
            <span className="page-info">
              Showing {Math.min(50, filteredOrders.length)} of {filteredOrders.length} orders
            </span>
          </div>
        </div>
      )}

      {activeTab === 'providers' && (
        <div className="providers-section">
          <div className="providers-grid">
            {providers.map(provider => (
              <div key={provider.id} className={`provider-card ${provider.status}`}>
                <div className="provider-header">
                  <span className="provider-logo">{provider.logo}</span>
                  <div className="provider-info">
                    <h3>{provider.name}</h3>
                    <span 
                      className="status-badge"
                      style={{ backgroundColor: getStatusColor(provider.status) }}
                    >
                      {provider.status}
                    </span>
                  </div>
                </div>
                
                <div className="provider-details">
                  <div className="detail-section">
                    <h4>Fiat Currencies</h4>
                    <div className="tags">
                      {provider.supportedFiat.map(fiat => (
                        <span key={fiat} className="tag">{fiat}</span>
                      ))}
                    </div>
                  </div>
                  
                  <div className="detail-section">
                    <h4>Crypto Supported</h4>
                    <div className="tags">
                      {provider.supportedCrypto.map(crypto => (
                        <span key={crypto} className="tag crypto">{crypto}</span>
                      ))}
                    </div>
                  </div>
                  
                  <div className="detail-section">
                    <h4>Fees</h4>
                    <p className="fee-info">{provider.fees.percentage}% + ${provider.fees.fixed} fixed</p>
                  </div>
                  
                  <div className="detail-section">
                    <h4>Limits</h4>
                    <p className="limit-info">${provider.limits.min} - ${provider.limits.max.toLocaleString()}</p>
                  </div>
                  
                  <div className="detail-section">
                    <h4>Countries</h4>
                    <p className="countries">{provider.countries.join(', ')}</p>
                  </div>
                </div>
                
                <div className="provider-actions">
                  <button 
                    className={`toggle-btn ${provider.status}`}
                    onClick={() => handleToggleProvider(provider.id)}
                  >
                    {provider.status === 'active' ? 'Disable' : 'Enable'}
                  </button>
                  <button className="edit-btn">Configure</button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {activeTab === 'settings' && (
        <div className="settings-section">
          <div className="settings-group">
            <h3>General Settings</h3>
            <div className="setting-item">
              <label>
                <input 
                  type="checkbox" 
                  checked={fiatSettings.enabled}
                  onChange={(e) => setFiatSettings({ ...fiatSettings, enabled: e.target.checked })}
                />
                Enable Fiat On-Ramp
              </label>
            </div>
            <div className="setting-item">
              <label>
                <input 
                  type="checkbox" 
                  checked={fiatSettings.autoApproval}
                  onChange={(e) => setFiatSettings({ ...fiatSettings, autoApproval: e.target.checked })}
                />
                Auto-approve orders under $1000
              </label>
            </div>
            <div className="setting-item">
              <label>
                <input 
                  type="checkbox" 
                  checked={fiatSettings.kycRequired}
                  onChange={(e) => setFiatSettings({ ...fiatSettings, kycRequired: e.target.checked })}
                />
                Require KYC verification
              </label>
            </div>
          </div>

          <div className="settings-group">
            <h3>Order Limits</h3>
            <div className="setting-item">
              <label>Minimum Order Amount (USD)</label>
              <input 
                type="number" 
                value={fiatSettings.minOrderAmount}
                onChange={(e) => setFiatSettings({ ...fiatSettings, minOrderAmount: parseInt(e.target.value) })}
              />
            </div>
            <div className="setting-item">
              <label>Maximum Order Amount (USD)</label>
              <input 
                type="number" 
                value={fiatSettings.maxOrderAmount}
                onChange={(e) => setFiatSettings({ ...fiatSettings, maxOrderAmount: parseInt(e.target.value) })}
              />
            </div>
          </div>

          <div className="settings-group">
            <h3>Default Provider</h3>
            <div className="setting-item">
              <select 
                value={fiatSettings.defaultProvider}
                onChange={(e) => setFiatSettings({ ...fiatSettings, defaultProvider: e.target.value })}
              >
                <option value="stripe">Stripe</option>
                <option value="moonpay">MoonPay</option>
                <option value="transak">Transak</option>
                <option value="simplex">Simplex</option>
              </select>
            </div>
          </div>

          <div className="settings-actions">
            <button className="save-btn">Save Settings</button>
          </div>
        </div>
      )}

      {/* Order Detail Modal */}
      {showOrderModal && selectedOrder && (
        <div className="modal-overlay" onClick={() => setShowOrderModal(false)}>
          <div className="modal-content" onClick={e => e.stopPropagation()}>
            <h2>Order Details - {selectedOrder.orderId}</h2>
            <div className="order-details">
              <div className="detail-row">
                <label>User:</label>
                <span>{selectedOrder.username} ({selectedOrder.userId})</span>
              </div>
              <div className="detail-row">
                <label>Crypto:</label>
                <span>{selectedOrder.crypto}</span>
              </div>
              <div className="detail-row">
                <label>Crypto Amount:</label>
                <span>{selectedOrder.cryptoAmount.toFixed(6)} {selectedOrder.crypto}</span>
              </div>
              <div className="detail-row">
                <label>Fiat Amount:</label>
                <span>{selectedOrder.fiatAmount.toLocaleString()} {selectedOrder.fiatCurrency}</span>
              </div>
              <div className="detail-row">
                <label>Payment Method:</label>
                <span>{selectedOrder.paymentMethod}</span>
              </div>
              <div className="detail-row">
                <label>Provider:</label>
                <span>{selectedOrder.provider}</span>
              </div>
              <div className="detail-row">
                <label>KYC Status:</label>
                <span 
                  className="status-badge"
                  style={{ backgroundColor: getStatusColor(selectedOrder.kycStatus) }}
                >
                  {selectedOrder.kycStatus}
                </span>
              </div>
              <div className="detail-row">
                <label>Status:</label>
                <span 
                  className="status-badge"
                  style={{ backgroundColor: getStatusColor(selectedOrder.status) }}
                >
                  {selectedOrder.status}
                </span>
              </div>
              <div className="detail-row">
                <label>Created:</label>
                <span>{selectedOrder.createdAt}</span>
              </div>
              {selectedOrder.completedAt && (
                <div className="detail-row">
                  <label>Completed:</label>
                  <span>{selectedOrder.completedAt}</span>
                </div>
              )}
              {selectedOrder.failureReason && (
                <div className="detail-row">
                  <label>Failure Reason:</label>
                  <span style={{ color: '#dc2626' }}>{selectedOrder.failureReason}</span>
                </div>
              )}
            </div>
            <div className="modal-actions">
              <button className="cancel-btn" onClick={() => setShowOrderModal(false)}>
                Close
              </button>
              {selectedOrder.status === 'processing' && (
                <button 
                  className="submit-btn"
                  onClick={() => { handleProcessOrder(selectedOrder.id); setShowOrderModal(false); }}
                >
                  Mark Complete
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default FiatOnRampPage;
