// P2P Trading Management Admin Page
// Full control over P2P orders, disputes, and user management

import React, { useState, useEffect } from 'react';
import './P2PTradingPage.css';

interface P2POrder {
  id: string;
  orderId: string;
  type: 'buy' | 'sell';
  token: string;
  amount: number;
  price: number;
  total: number;
  fiat: string;
  paymentMethod: string;
  seller: string;
  sellerId: string;
  buyer: string;
  buyerId: string;
  status: 'pending' | 'paid' | 'released' | 'cancelled' | 'disputed' | 'expired';
  createdAt: string;
  completedAt?: string;
  cancelReason?: string;
  disputeReason?: string;
}

interface P2PUser {
  id: string;
  username: string;
  email: string;
  tradesCompleted: number;
  rating: number;
  completionRate: number;
  status: 'active' | 'banned' | 'suspended';
  registeredAt: string;
  totalVolume: number;
}

interface Dispute {
  id: string;
  orderId: string;
  raisedBy: string;
  reason: string;
  description: string;
  evidence: string[];
  status: 'open' | 'under_review' | 'resolved' | 'closed';
  resolution?: 'buyer_wins' | 'seller_wins' | 'cancelled';
  adminNote?: string;
  createdAt: string;
}

const P2PTradingPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'orders' | 'users' | 'disputes' | 'settings'>('orders');
  const [orders, setOrders] = useState<P2POrder[]>([]);
  const [users, setUsers] = useState<P2PUser[]>([]);
  const [disputes, setDisputes] = useState<Dispute[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [filterToken, setFilterToken] = useState<string>('all');
  const [showOrderModal, setShowOrderModal] = useState(false);
  const [selectedOrder, setSelectedOrder] = useState<P2POrder | null>(null);
  const [showDisputeModal, setShowDisputeModal] = useState(false);
  const [selectedDispute, setSelectedDispute] = useState<Dispute | null>(null);

  // Initialize with sample data
  useEffect(() => {
    // Generate sample orders
    const sampleOrders: P2POrder[] = [
      { id: '1', orderId: 'P2P-2024-001', type: 'buy', token: 'USDT', amount: 1000, price: 1.02, total: 1020, fiat: 'USD', paymentMethod: 'Bank Transfer', seller: 'CryptoKing', sellerId: 'u1', buyer: 'TraderJoe', buyerId: 'u2', status: 'pending', createdAt: '2024-01-15 10:30:00' },
      { id: '2', orderId: 'P2P-2024-002', type: 'sell', token: 'BTC', amount: 0.5, price: 43000, total: 21500, fiat: 'USD', paymentMethod: 'PayPal', seller: 'BitcoinPro', sellerId: 'u3', buyer: 'NewUser', buyerId: 'u4', status: 'paid', createdAt: '2024-01-15 09:15:00' },
      { id: '3', orderId: 'P2P-2024-003', type: 'buy', token: 'ETH', amount: 5, price: 2350, total: 11750, fiat: 'EUR', paymentMethod: 'SEPA', seller: 'EthereumMaster', sellerId: 'u5', buyer: 'EuroTrader', buyerId: 'u6', status: 'released', createdAt: '2024-01-14 16:45:00', completedAt: '2024-01-14 17:00:00' },
      { id: '4', orderId: 'P2P-2024-004', type: 'sell', token: 'USDT', amount: 5000, price: 1.01, total: 5050, fiat: 'GBP', paymentMethod: 'Bank Transfer', seller: 'UKTrader', sellerId: 'u7', buyer: 'BritishCoin', buyerId: 'u8', status: 'disputed', createdAt: '2024-01-14 14:20:00', disputeReason: 'Payment not received' },
      { id: '5', orderId: 'P2P-2024-005', type: 'buy', token: 'SOL', amount: 100, price: 105, total: 10500, fiat: 'USD', paymentMethod: 'Venmo', seller: 'SolanaWhale', sellerId: 'u9', buyer: 'SolLover', buyerId: 'u10', status: 'cancelled', createdAt: '2024-01-14 12:00:00', cancelReason: 'Better price elsewhere' },
      { id: '6', orderId: 'P2P-2024-006', type: 'sell', token: 'USDT', amount: 2500, price: 1.015, total: 2537.50, fiat: 'USD', paymentMethod: 'Cash Deposit', seller: 'CashTrader', sellerId: 'u11', buyer: 'QuickBuyer', buyerId: 'u12', status: 'expired', createdAt: '2024-01-13 20:00:00' },
    ];

    // Generate more orders to show variety
    const tokens = ['USDT', 'BTC', 'ETH', 'BNB', 'SOL', 'XRP', 'ADA', 'DOGE'];
    const fiatCurrencies = ['USD', 'EUR', 'GBP', 'CNY', 'KRW', 'JPY'];
    const paymentMethods = ['Bank Transfer', 'PayPal', 'Venmo', 'Cash Deposit', 'SEPA', 'AliPay', 'WeChat Pay'];
    
    for (let i = 7; i <= 100; i++) {
      const token = tokens[Math.floor(Math.random() * tokens.length)];
      const amount = Math.random() * 1000 + 10;
      const price = token === 'BTC' ? 42000 + Math.random() * 2000 : 
                   token === 'ETH' ? 2200 + Math.random() * 300 :
                   0.9 + Math.random() * 0.2;
      
      const statuses: P2POrder['status'][] = ['pending', 'paid', 'released', 'cancelled', 'disputed', 'expired'];
      const status = statuses[Math.floor(Math.random() * statuses.length)];
      
      sampleOrders.push({
        id: String(i),
        orderId: `P2P-2024-${String(i).padStart(3, '0')}`,
        type: Math.random() > 0.5 ? 'buy' : 'sell',
        token,
        amount,
        price,
        total: amount * price,
        fiat: fiatCurrencies[Math.floor(Math.random() * fiatCurrencies.length)],
        paymentMethod: paymentMethods[Math.floor(Math.random() * paymentMethods.length)],
        seller: `Seller${i}`,
        sellerId: `u${i}`,
        buyer: `Buyer${i}`,
        buyerId: `u${i + 100}`,
        status,
        createdAt: `2024-01-${String(10 + Math.floor(i / 10)).padStart(2, '0')} ${String(Math.floor(Math.random() * 24)).padStart(2, '0')}:${String(Math.floor(Math.random() * 60)).padStart(2, '0')}:00`,
      });
    }

    setOrders(sampleOrders);

    // Sample users
    const sampleUsers: P2PUser[] = [
      { id: 'u1', username: 'CryptoKing', email: 'king@crypto.com', tradesCompleted: 245, rating: 4.9, completionRate: 98.5, status: 'active', registeredAt: '2023-06-15', totalVolume: 1250000 },
      { id: 'u2', username: 'TraderJoe', email: 'joe@trader.com', tradesCompleted: 89, rating: 4.7, completionRate: 95.2, status: 'active', registeredAt: '2023-09-20', totalVolume: 450000 },
      { id: 'u3', username: 'BitcoinPro', email: 'pro@btc.com', tradesCompleted: 512, rating: 4.95, completionRate: 99.1, status: 'active', registeredAt: '2023-01-10', totalVolume: 5200000 },
      { id: 'u4', username: 'NewUser', email: 'new@user.com', tradesCompleted: 5, rating: 4.0, completionRate: 80.0, status: 'active', registeredAt: '2024-01-01', totalVolume: 15000 },
      { id: 'u5', username: 'ScammerAlert', email: 'scam@alert.com', tradesCompleted: 15, rating: 2.5, completionRate: 45.0, status: 'banned', registeredAt: '2023-12-01', totalVolume: 5000 },
    ];
    setUsers(sampleUsers);

    // Sample disputes
    const sampleDisputes: Dispute[] = [
      { id: 'd1', orderId: 'P2P-2024-004', raisedBy: 'UKTrader', reason: 'Payment not received', description: 'Buyer claims to have made payment but I have not received any funds', evidence: ['screenshot1.png', 'bank_statement.pdf'], status: 'under_review', createdAt: '2024-01-14 15:00:00' },
      { id: 'd2', orderId: 'P2P-2024-008', raisedBy: 'Buyer123', reason: 'Token not released', description: 'I made payment but seller has not released the tokens', evidence: ['receipt.jpg'], status: 'open', createdAt: '2024-01-15 08:30:00' },
    ];
    setDisputes(sampleDisputes);
  }, []);

  const filteredOrders = orders.filter(order => {
    if (filterStatus !== 'all' && order.status !== filterStatus) return false;
    if (filterToken !== 'all' && order.token !== filterToken) return false;
    if (searchTerm && !order.orderId.toLowerCase().includes(searchTerm.toLowerCase()) &&
        !order.seller.toLowerCase().includes(searchTerm.toLowerCase()) &&
        !order.buyer.toLowerCase().includes(searchTerm.toLowerCase())) return false;
    return true;
  });

  const handleReleaseOrder = (orderId: string) => {
    setOrders(orders.map(o => 
      o.id === orderId ? { ...o, status: 'released' as const, completedAt: new Date().toLocaleString() } : o
    ));
  };

  const handleCancelOrder = (orderId: string, reason: string) => {
    setOrders(orders.map(o => 
      o.id === orderId ? { ...o, status: 'cancelled' as const, cancelReason: reason } : o
    ));
  };

  const handleResolveDispute = (disputeId: string, resolution: 'buyer_wins' | 'seller_wins' | 'cancelled', note: string) => {
    setDisputes(disputes.map(d => 
      d.id === disputeId ? { ...d, status: 'resolved' as const, resolution, adminNote: note } : d
    ));
    // Also update the associated order
    const dispute = disputes.find(d => d.id === disputeId);
    if (dispute) {
      if (resolution === 'buyer_wins') {
        handleReleaseOrder(dispute.orderId.replace('P2P-2024-', String(parseInt(dispute.orderId.split('-')[2]) - 3)));
      } else if (resolution === 'seller_wins') {
        handleCancelOrder(dispute.orderId.replace('P2P-2024-', String(parseInt(dispute.orderId.split('-')[2]) - 3)), 'Dispute resolved in favor of seller');
      }
    }
  };

  const handleBanUser = (userId: string) => {
    setUsers(users.map(u => 
      u.id === userId ? { ...u, status: 'banned' as const } : u
    ));
  };

  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      pending: '#FFA500',
      paid: '#4169E1',
      released: '#28A745',
      cancelled: '#6C757D',
      disputed: '#DC3545',
      expired: '#6C757D',
      open: '#DC3545',
      under_review: '#FFA500',
      resolved: '#28A745',
      closed: '#6C757D',
    };
    return colors[status] || '#6C757D';
  };

  // Stats
  const stats = {
    totalOrders: orders.length,
    pendingOrders: orders.filter(o => o.status === 'pending').length,
    activeOrders: orders.filter(o => ['pending', 'paid'].includes(o.status)).length,
    disputedOrders: orders.filter(o => o.status === 'disputed').length,
    totalVolume: orders.filter(o => o.status === 'released').reduce((sum, o) => sum + o.total, 0),
  };

  return (
    <div className="p2p-trading-page">
      <div className="page-header">
        <h1>P2P Trading Management</h1>
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
          <div className="stat-icon">⚡</div>
          <div className="stat-info">
            <span className="stat-value">{stats.activeOrders}</span>
            <span className="stat-label">Active</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">⚠️</div>
          <div className="stat-info">
            <span className="stat-value">{stats.disputedOrders}</span>
            <span className="stat-label">Disputed</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">💰</div>
          <div className="stat-info">
            <span className="stat-value">${(stats.totalVolume / 1000000).toFixed(2)}M</span>
            <span className="stat-label">Completed Volume</span>
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
          className={activeTab === 'users' ? 'active' : ''} 
          onClick={() => setActiveTab('users')}
        >
          👥 Users ({users.length})
        </button>
        <button 
          className={activeTab === 'disputes' ? 'active' : ''} 
          onClick={() => setActiveTab('disputes')}
        >
          ⚠️ Disputes ({disputes.filter(d => d.status !== 'resolved').length})
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
              <option value="paid">Paid</option>
              <option value="released">Released</option>
              <option value="cancelled">Cancelled</option>
              <option value="disputed">Disputed</option>
              <option value="expired">Expired</option>
            </select>
            <select 
              value={filterToken} 
              onChange={(e) => setFilterToken(e.target.value)}
              className="filter-select"
            >
              <option value="all">All Tokens</option>
              <option value="USDT">USDT</option>
              <option value="BTC">BTC</option>
              <option value="ETH">ETH</option>
              <option value="BNB">BNB</option>
              <option value="SOL">SOL</option>
            </select>
          </div>

          <div className="orders-table">
            <table>
              <thead>
                <tr>
                  <th>Order ID</th>
                  <th>Type</th>
                  <th>Token</th>
                  <th>Amount</th>
                  <th>Price</th>
                  <th>Total</th>
                  <th>Fiat</th>
                  <th>Payment</th>
                  <th>Seller</th>
                  <th>Buyer</th>
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
                      <span className={`type-badge ${order.type}`}>
                        {order.type.toUpperCase()}
                      </span>
                    </td>
                    <td>{order.token}</td>
                    <td>{order.amount.toLocaleString()}</td>
                    <td>${order.price.toLocaleString()}</td>
                    <td>${order.total.toLocaleString()}</td>
                    <td>{order.fiat}</td>
                    <td>{order.paymentMethod}</td>
                    <td>{order.seller}</td>
                    <td>{order.buyer}</td>
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
                        {order.status === 'paid' && (
                          <button 
                            className="action-btn release"
                            onClick={() => handleReleaseOrder(order.id)}
                          >
                            Release
                          </button>
                        )}
                        {['pending', 'paid'].includes(order.status) && (
                          <button 
                            className="action-btn cancel"
                            onClick={() => handleCancelOrder(order.id, 'Admin cancelled')}
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

      {activeTab === 'users' && (
        <div className="users-section">
          <div className="filters">
            <input
              type="text"
              placeholder="Search users..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="search-input"
            />
          </div>

          <div className="users-table">
            <table>
              <thead>
                <tr>
                  <th>Username</th>
                  <th>Email</th>
                  <th>Trades</th>
                  <th>Rating</th>
                  <th>Completion Rate</th>
                  <th>Total Volume</th>
                  <th>Status</th>
                  <th>Registered</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {users.filter(u => 
                  searchTerm ? u.username.toLowerCase().includes(searchTerm.toLowerCase()) || 
                  u.email.toLowerCase().includes(searchTerm.toLowerCase()) : true
                ).map(user => (
                  <tr key={user.id}>
                    <td>{user.username}</td>
                    <td>{user.email}</td>
                    <td>{user.tradesCompleted}</td>
                    <td>
                      <span className="rating">⭐ {user.rating.toFixed(1)}</span>
                    </td>
                    <td>{user.completionRate.toFixed(1)}%</td>
                    <td>${user.totalVolume.toLocaleString()}</td>
                    <td>
                      <span className={`status-badge ${user.status}`}>
                        {user.status}
                      </span>
                    </td>
                    <td>{user.registeredAt}</td>
                    <td>
                      <div className="actions">
                        {user.status === 'active' ? (
                          <button 
                            className="action-btn ban"
                            onClick={() => handleBanUser(user.id)}
                          >
                            Ban
                          </button>
                        ) : (
                          <button className="action-btn unban">
                            Unban
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === 'disputes' && (
        <div className="disputes-section">
          <div className="disputes-list">
            {disputes.map(dispute => (
              <div key={dispute.id} className="dispute-card">
                <div className="dispute-header">
                  <h3>Dispute #{dispute.id}</h3>
                  <span 
                    className="status-badge"
                    style={{ backgroundColor: getStatusColor(dispute.status) }}
                  >
                    {dispute.status}
                  </span>
                </div>
                <div className="dispute-body">
                  <div className="dispute-detail">
                    <label>Order ID:</label>
                    <span>{dispute.orderId}</span>
                  </div>
                  <div className="dispute-detail">
                    <label>Raised By:</label>
                    <span>{dispute.raisedBy}</span>
                  </div>
                  <div className="dispute-detail">
                    <label>Reason:</label>
                    <span>{dispute.reason}</span>
                  </div>
                  <div className="dispute-detail">
                    <label>Description:</label>
                    <p>{dispute.description}</p>
                  </div>
                  <div className="dispute-detail">
                    <label>Evidence:</label>
                    <div className="evidence-files">
                      {dispute.evidence.map((file, idx) => (
                        <span key={idx} className="evidence-file">📎 {file}</span>
                      ))}
                    </div>
                  </div>
                  <div className="dispute-detail">
                    <label>Created:</label>
                    <span>{dispute.createdAt}</span>
                  </div>
                </div>
                {dispute.status !== 'resolved' && dispute.status !== 'closed' && (
                  <div className="dispute-actions">
                    <button 
                      className="action-btn buyer-wins"
                      onClick={() => handleResolveDispute(dispute.id, 'buyer_wins', 'Order released to buyer')}
                    >
                      Buyer Wins
                    </button>
                    <button 
                      className="action-btn seller-wins"
                      onClick={() => handleResolveDispute(dispute.id, 'seller_wins', 'Order cancelled, funds returned to seller')}
                    >
                      Seller Wins
                    </button>
                    <button 
                      className="action-btn cancel-order"
                      onClick={() => handleResolveDispute(dispute.id, 'cancelled', 'Order cancelled by admin')}
                    >
                      Cancel Order
                    </button>
                  </div>
                )}
                {dispute.status === 'resolved' && (
                  <div className="resolution-info">
                    <strong>Resolution:</strong> {dispute.resolution?.replace('_', ' ')}
                    {dispute.adminNote && <p>Note: {dispute.adminNote}</p>}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {activeTab === 'settings' && (
        <div className="settings-section">
          <div className="settings-group">
            <h3>P2P General Settings</h3>
            <div className="setting-item">
              <label>
                <input type="checkbox" defaultChecked />
                Enable P2P Trading
              </label>
            </div>
            <div className="setting-item">
              <label>Minimum Order Amount (USDT)</label>
              <input type="number" defaultValue={10} />
            </div>
            <div className="setting-item">
              <label>Maximum Order Amount (USDT)</label>
              <input type="number" defaultValue={100000} />
            </div>
            <div className="setting-item">
              <label>Order Expiry Time (minutes)</label>
              <input type="number" defaultValue={30} />
            </div>
            <div className="setting-item">
              <label>Payment Window (minutes)</label>
              <input type="number" defaultValue={15} />
            </div>
          </div>

          <div className="settings-group">
            <h3>Fiat Currencies</h3>
            <div className="currencies-list">
              {['USD', 'EUR', 'GBP', 'CNY', 'KRW', 'JPY', 'AUD', 'CAD', 'INR'].map(currency => (
                <label key={currency} className="currency-item">
                  <input type="checkbox" defaultChecked />
                  {currency}
                </label>
              ))}
            </div>
          </div>

          <div className="settings-group">
            <h3>Payment Methods</h3>
            <div className="payment-methods">
              {['Bank Transfer', 'PayPal', 'Venmo', 'Cash Deposit', 'SEPA', 'AliPay', 'WeChat Pay', 'Crypto'].map(method => (
                <label key={method} className="payment-item">
                  <input type="checkbox" defaultChecked />
                  {method}
                </label>
              ))}
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
                <label>Type:</label>
                <span className={`type-badge ${selectedOrder.type}`}>
                  {selectedOrder.type.toUpperCase()}
                </span>
              </div>
              <div className="detail-row">
                <label>Token:</label>
                <span>{selectedOrder.token}</span>
              </div>
              <div className="detail-row">
                <label>Amount:</label>
                <span>{selectedOrder.amount.toLocaleString()} {selectedOrder.token}</span>
              </div>
              <div className="detail-row">
                <label>Price:</label>
                <span>${selectedOrder.price.toLocaleString()}</span>
              </div>
              <div className="detail-row">
                <label>Total:</label>
                <span>${selectedOrder.total.toLocaleString()} {selectedOrder.fiat}</span>
              </div>
              <div className="detail-row">
                <label>Payment Method:</label>
                <span>{selectedOrder.paymentMethod}</span>
              </div>
              <div className="detail-row">
                <label>Seller:</label>
                <span>{selectedOrder.seller} ({selectedOrder.sellerId})</span>
              </div>
              <div className="detail-row">
                <label>Buyer:</label>
                <span>{selectedOrder.buyer} ({selectedOrder.buyerId})</span>
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
              {selectedOrder.cancelReason && (
                <div className="detail-row">
                  <label>Cancel Reason:</label>
                  <span>{selectedOrder.cancelReason}</span>
                </div>
              )}
              {selectedOrder.disputeReason && (
                <div className="detail-row">
                  <label>Dispute Reason:</label>
                  <span>{selectedOrder.disputeReason}</span>
                </div>
              )}
            </div>
            <div className="modal-actions">
              <button className="cancel-btn" onClick={() => setShowOrderModal(false)}>
                Close
              </button>
              {selectedOrder.status === 'paid' && (
                <button 
                  className="submit-btn"
                  onClick={() => { handleReleaseOrder(selectedOrder.id); setShowOrderModal(false); }}
                >
                  Release Funds
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default P2PTradingPage;
