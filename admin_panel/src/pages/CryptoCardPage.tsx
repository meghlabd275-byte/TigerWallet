// Crypto Card System Management Admin Page
// Full control over crypto cards, limits, and transactions

import React, { useState, useEffect } from 'react';
import './CryptoCardPage.css';

interface CryptoCard {
  id: string;
  cardNumber: string;
  cardHolder: string;
  userId: string;
  type: 'virtual' | 'physical';
  brand: 'Visa' | 'Mastercard' | 'UnionPay' | 'American Express';
  status: 'active' | 'frozen' | 'blocked' | 'expired';
  balance: number;
  dailyLimit: number;
  monthlyLimit: number;
  dailySpent: number;
  monthlySpent: number;
  createdAt: string;
  expiresAt: string;
  lastTransaction?: string;
}

interface CardTransaction {
  id: string;
  cardId: string;
  cardNumber: string;
  merchant: string;
  merchantCategory: string;
  amount: number;
  currency: string;
  type: 'purchase' | 'withdrawal' | 'refund';
  status: 'pending' | 'completed' | 'declined' | 'refunded';
  location: string;
  timestamp: string;
}

interface CardSettings {
  globalDailyLimit: number;
  globalMonthlyLimit: number;
  defaultCardLimit: number;
  allowedMerchants: string[];
  blockedMerchants: string[];
  enabled: boolean;
}

const CryptoCardPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'cards' | 'transactions' | 'settings'>('cards');
  const [cards, setCards] = useState<CryptoCard[]>([]);
  const [transactions, setTransactions] = useState<CardTransaction[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [filterType, setFilterType] = useState<string>('all');
  const [selectedCard, setSelectedCard] = useState<CryptoCard | null>(null);
  const [showCardModal, setShowCardModal] = useState(false);
  const [showLimitModal, setShowLimitModal] = useState(false);

  const [cardSettings, setCardSettings] = useState<CardSettings>({
    globalDailyLimit: 10000,
    globalMonthlyLimit: 50000,
    defaultCardLimit: 5000,
    allowedMerchants: ['All'],
    blockedMerchants: ['Gambling', 'Adult'],
    enabled: true,
  });

  // Initialize with sample data
  useEffect(() => {
    // Sample cards
    const sampleCards: CryptoCard[] = [
      { id: 'c1', cardNumber: '4532 1234 5678 9010', cardHolder: 'John Doe', userId: 'u1', type: 'physical', brand: 'Visa', status: 'active', balance: 5000, dailyLimit: 2000, monthlyLimit: 10000, dailySpent: 450, monthlySpent: 3200, createdAt: '2024-01-01', expiresAt: '2027-01', lastTransaction: '2024-01-15' },
      { id: 'c2', cardNumber: '5425 2334 3010 9876', cardHolder: 'Jane Smith', userId: 'u2', type: 'virtual', brand: 'Mastercard', status: 'active', balance: 2500, dailyLimit: 1000, monthlyLimit: 5000, dailySpent: 200, monthlySpent: 1800, createdAt: '2024-01-05', expiresAt: '2026-01', lastTransaction: '2024-01-14' },
      { id: 'c3', cardNumber: '4916 5432 1098 7654', cardHolder: 'Mike Johnson', userId: 'u3', type: 'physical', brand: 'Visa', status: 'frozen', balance: 1000, dailyLimit: 500, monthlyLimit: 2000, dailySpent: 0, monthlySpent: 500, createdAt: '2023-12-15', expiresAt: '2026-12', lastTransaction: '2024-01-10' },
      { id: 'c4', cardNumber: '5555 5555 5555 4444', cardHolder: 'Sarah Williams', userId: 'u4', type: 'physical', brand: 'Mastercard', status: 'active', balance: 10000, dailyLimit: 5000, monthlyLimit: 25000, dailySpent: 1200, monthlySpent: 8500, createdAt: '2023-11-20', expiresAt: '2026-11', lastTransaction: '2024-01-15' },
      { id: 'c5', cardNumber: '3782 822463 10005', cardHolder: 'David Brown', userId: 'u5', type: 'virtual', brand: 'American Express', status: 'blocked', balance: 0, dailyLimit: 0, monthlyLimit: 0, dailySpent: 0, monthlySpent: 0, createdAt: '2023-10-01', expiresAt: '2025-10', lastTransaction: '2023-12-20' },
      { id: 'c6', cardNumber: '6250 9876 5432 1098', cardHolder: 'Lisa Chen', userId: 'u6', type: 'physical', brand: 'UnionPay', status: 'active', balance: 3500, dailyLimit: 1500, monthlyLimit: 7500, dailySpent: 300, monthlySpent: 2100, createdAt: '2024-01-08', expiresAt: '2027-01', lastTransaction: '2024-01-13' },
    ];

    // Generate more cards
    const names = ['Alice Wang', 'Bob Lee', 'Carol Zhang', 'Daniel Liu', 'Emma Yang', 'Frank Huang', 'Grace Xu', 'Henry Zhang'];
    for (let i = 7; i <= 50; i++) {
      const brands: CryptoCard['brand'][] = ['Visa', 'Mastercard', 'UnionPay'];
      const statuses: CryptoCard['status'][] = ['active', 'active', 'active', 'frozen', 'blocked'];
      const types: CryptoCard['type'][] = ['virtual', 'physical'];
      
      sampleCards.push({
        id: `c${i}`,
        cardNumber: `4${Math.floor(Math.random() * 9000) + 1000} ${Math.floor(Math.random() * 9000) + 1000} ${Math.floor(Math.random() * 9000) + 1000} ${Math.floor(Math.random() * 9000) + 1000}`,
        cardHolder: names[i % names.length],
        userId: `u${i}`,
        type: types[Math.floor(Math.random() * types.length)],
        brand: brands[Math.floor(Math.random() * brands.length)],
        status: statuses[Math.floor(Math.random() * statuses.length)],
        balance: Math.floor(Math.random() * 10000) + 500,
        dailyLimit: Math.floor(Math.random() * 3000) + 500,
        monthlyLimit: Math.floor(Math.random() * 15000) + 5000,
        dailySpent: Math.floor(Math.random() * 500),
        monthlySpent: Math.floor(Math.random() * 5000),
        createdAt: `2023-${String(Math.floor(Math.random() * 12) + 1).padStart(2, '0')}-${String(Math.floor(Math.random() * 28) + 1).padStart(2, '0')}`,
        expiresAt: `2026-${String(Math.floor(Math.random() * 12) + 1).padStart(2, '0')}`,
      });
    }
    setCards(sampleCards);

    // Sample transactions
    const sampleTransactions: CardTransaction[] = [
      { id: 't1', cardId: 'c1', cardNumber: '4532 1234 5678 9010', merchant: 'Amazon', merchantCategory: 'Online Shopping', amount: 125.50, currency: 'USD', type: 'purchase', status: 'completed', location: 'Online', timestamp: '2024-01-15 14:30:00' },
      { id: 't2', cardId: 'c1', cardNumber: '4532 1234 5678 9010', merchant: 'Apple Store', merchantCategory: 'Electronics', amount: 999.00, currency: 'USD', type: 'purchase', status: 'completed', location: 'New York, NY', timestamp: '2024-01-14 10:15:00' },
      { id: 't3', cardId: 'c2', cardNumber: '5425 2334 3010 9876', merchant: 'Netflix', merchantCategory: 'Streaming', amount: 15.99, currency: 'USD', type: 'purchase', status: 'completed', location: 'Online', timestamp: '2024-01-15 09:00:00' },
      { id: 't4', cardId: 'c4', cardNumber: '5555 5555 5555 4444', merchant: 'Whole Foods', merchantCategory: 'Groceries', amount: 87.32, currency: 'USD', type: 'purchase', status: 'completed', location: 'San Francisco, CA', timestamp: '2024-01-15 16:45:00' },
      { id: 't5', cardId: 'c4', cardNumber: '5555 5555 5555 4444', merchant: 'Uber', merchantCategory: 'Transportation', amount: 24.50, currency: 'USD', type: 'purchase', status: 'completed', location: 'San Francisco, CA', timestamp: '2024-01-15 12:30:00' },
      { id: 't6', cardId: 'c6', cardNumber: '6250 9876 5432 1098', merchant: 'Starbucks', merchantCategory: 'Food & Beverage', amount: 8.50, currency: 'USD', type: 'purchase', status: 'completed', location: 'Los Angeles, CA', timestamp: '2024-01-13 08:00:00' },
      { id: 't7', cardId: 'c3', cardNumber: '4916 5432 1098 7654', merchant: 'Gambling Site', merchantCategory: 'Gambling', amount: 500.00, currency: 'USD', type: 'purchase', status: 'declined', location: 'Online', timestamp: '2024-01-10 20:00:00' },
    ];
    setTransactions(sampleTransactions);
  }, []);

  const filteredCards = cards.filter(card => {
    if (filterStatus !== 'all' && card.status !== filterStatus) return false;
    if (filterType !== 'all' && card.type !== filterType) return false;
    if (searchTerm && !card.cardNumber.includes(searchTerm) && 
        !card.cardHolder.toLowerCase().includes(searchTerm.toLowerCase()) &&
        !card.userId.toLowerCase().includes(searchTerm.toLowerCase())) return false;
    return true;
  });

  const filteredTransactions = transactions.filter(tx => {
    if (searchTerm && !tx.merchant.toLowerCase().includes(searchTerm.toLowerCase()) &&
        !tx.cardNumber.includes(searchTerm)) return false;
    return true;
  });

  const handleFreezeCard = (cardId: string) => {
    setCards(cards.map(c => 
      c.id === cardId ? { ...c, status: c.status === 'frozen' ? 'active' : 'frozen' as const } : c
    ));
  };

  const handleBlockCard = (cardId: string) => {
    setCards(cards.map(c => 
      c.id === cardId ? { ...c, status: 'blocked' as const } : c
    ));
  };

  const handleUpdateLimits = (cardId: string, dailyLimit: number, monthlyLimit: number) => {
    setCards(cards.map(c => 
      c.id === cardId ? { ...c, dailyLimit, monthlyLimit } : c
    ));
    setShowLimitModal(false);
  };

  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      active: '#28a745',
      frozen: '#ffc107',
      blocked: '#dc3545',
      expired: '#6c757d',
      pending: '#ffc107',
      completed: '#28a745',
      declined: '#dc3545',
      refunded: '#17a2b8',
    };
    return colors[status] || '#6c757d';
  };

  // Stats
  const stats = {
    totalCards: cards.length,
    activeCards: cards.filter(c => c.status === 'active').length,
    totalBalance: cards.reduce((sum, c) => sum + c.balance, 0),
    totalTransactions: transactions.length,
    dailyVolume: transactions.filter(t => t.timestamp.startsWith('2024-01-15')).reduce((sum, t) => sum + t.amount, 0),
  };

  return (
    <div className="crypto-card-page">
      <div className="page-header">
        <h1>Crypto Card Management</h1>
        <div className="header-actions">
          <button className="add-btn">+ Issue New Card</button>
        </div>
      </div>

      <div className="stats-cards">
        <div className="stat-card">
          <div className="stat-icon">💳</div>
          <div className="stat-info">
            <span className="stat-value">{stats.totalCards}</span>
            <span className="stat-label">Total Cards</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">✅</div>
          <div className="stat-info">
            <span className="stat-value">{stats.activeCards}</span>
            <span className="stat-label">Active Cards</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">💰</div>
          <div className="stat-info">
            <span className="stat-value">${stats.totalBalance.toLocaleString()}</span>
            <span className="stat-label">Total Balance</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">📊</div>
          <div className="stat-info">
            <span className="stat-value">{stats.totalTransactions}</span>
            <span className="stat-label">Transactions</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">📈</div>
          <div className="stat-info">
            <span className="stat-value">${stats.dailyVolume.toFixed(2)}</span>
            <span className="stat-label">Daily Volume</span>
          </div>
        </div>
      </div>

      <div className="tabs">
        <button 
          className={activeTab === 'cards' ? 'active' : ''} 
          onClick={() => setActiveTab('cards')}
        >
          💳 Cards ({cards.length})
        </button>
        <button 
          className={activeTab === 'transactions' ? 'active' : ''} 
          onClick={() => setActiveTab('transactions')}
        >
          📝 Transactions ({transactions.length})
        </button>
        <button 
          className={activeTab === 'settings' ? 'active' : ''} 
          onClick={() => setActiveTab('settings')}
        >
          ⚙️ Settings
        </button>
      </div>

      {activeTab === 'cards' && (
        <div className="cards-section">
          <div className="filters">
            <input
              type="text"
              placeholder="Search cards..."
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
              <option value="active">Active</option>
              <option value="frozen">Frozen</option>
              <option value="blocked">Blocked</option>
              <option value="expired">Expired</option>
            </select>
            <select 
              value={filterType} 
              onChange={(e) => setFilterType(e.target.value)}
              className="filter-select"
            >
              <option value="all">All Types</option>
              <option value="virtual">Virtual</option>
              <option value="physical">Physical</option>
            </select>
          </div>

          <div className="cards-table">
            <table>
              <thead>
                <tr>
                  <th>Card Number</th>
                  <th>Card Holder</th>
                  <th>Type</th>
                  <th>Brand</th>
                  <th>Balance</th>
                  <th>Daily Limit</th>
                  <th>Monthly Limit</th>
                  <th>Daily Spent</th>
                  <th>Status</th>
                  <th>Expires</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {filteredCards.slice(0, 30).map(card => (
                  <tr key={card.id}>
                    <td className="card-number">{card.cardNumber}</td>
                    <td>
                      <div className="card-holder">
                        <span className="name">{card.cardHolder}</span>
                        <span className="id">{card.userId}</span>
                      </div>
                    </td>
                    <td>
                      <span className={`type-badge ${card.type}`}>
                        {card.type}
                      </span>
                    </td>
                    <td>{card.brand}</td>
                    <td>${card.balance.toLocaleString()}</td>
                    <td>
                      <div className="limit-info">
                        <span>${card.dailyLimit.toLocaleString()}</span>
                        <div className="progress-bar">
                          <div 
                            className="progress" 
                            style={{ width: `${(card.dailySpent / card.dailyLimit) * 100}%` }}
                          />
                        </div>
                      </div>
                    </td>
                    <td>
                      <div className="limit-info">
                        <span>${card.monthlyLimit.toLocaleString()}</span>
                        <div className="progress-bar">
                          <div 
                            className="progress" 
                            style={{ width: `${(card.monthlySpent / card.monthlyLimit) * 100}%` }}
                          />
                        </div>
                      </div>
                    </td>
                    <td>
                      <span className={card.dailySpent > card.dailyLimit * 0.8 ? 'warning' : ''}>
                        ${card.dailySpent.toLocaleString()}
                      </span>
                    </td>
                    <td>
                      <span 
                        className="status-badge"
                        style={{ backgroundColor: getStatusColor(card.status) }}
                      >
                        {card.status}
                      </span>
                    </td>
                    <td>{card.expiresAt}</td>
                    <td>
                      <div className="actions">
                        <button 
                          className="action-btn view"
                          onClick={() => { setSelectedCard(card); setShowCardModal(true); }}
                        >
                          View
                        </button>
                        <button 
                          className="action-btn limit"
                          onClick={() => { setSelectedCard(card); setShowLimitModal(true); }}
                        >
                          Limits
                        </button>
                        {card.status !== 'blocked' && (
                          <button 
                            className="action-btn freeze"
                            onClick={() => handleFreezeCard(card.id)}
                          >
                            {card.status === 'frozen' ? 'Unfreeze' : 'Freeze'}
                          </button>
                        )}
                        {card.status !== 'blocked' && (
                          <button 
                            className="action-btn block"
                            onClick={() => handleBlockCard(card.id)}
                          >
                            Block
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
              Showing {Math.min(30, filteredCards.length)} of {filteredCards.length} cards
            </span>
          </div>
        </div>
      )}

      {activeTab === 'transactions' && (
        <div className="transactions-section">
          <div className="filters">
            <input
              type="text"
              placeholder="Search transactions..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="search-input"
            />
          </div>

          <div className="transactions-table">
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Card</th>
                  <th>Merchant</th>
                  <th>Category</th>
                  <th>Amount</th>
                  <th>Type</th>
                  <th>Status</th>
                  <th>Location</th>
                  <th>Time</th>
                </tr>
              </thead>
              <tbody>
                {filteredTransactions.map(tx => (
                  <tr key={tx.id}>
                    <td className="tx-id">{tx.id}</td>
                    <td className="card-number">{tx.cardNumber}</td>
                    <td>{tx.merchant}</td>
                    <td>
                      <span className="category-badge">
                        {tx.merchantCategory}
                      </span>
                    </td>
                    <td className="amount">${tx.amount.toFixed(2)}</td>
                    <td>
                      <span className={`type-badge ${tx.type}`}>
                        {tx.type}
                      </span>
                    </td>
                    <td>
                      <span 
                        className="status-badge"
                        style={{ backgroundColor: getStatusColor(tx.status) }}
                      >
                        {tx.status}
                      </span>
                    </td>
                    <td>{tx.location}</td>
                    <td>{tx.timestamp}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === 'settings' && (
        <div className="settings-section">
          <div className="settings-group">
            <h3>Global Card Settings</h3>
            <div className="setting-item">
              <label>
                <input 
                  type="checkbox" 
                  checked={cardSettings.enabled}
                  onChange={(e) => setCardSettings({ ...cardSettings, enabled: e.target.checked })}
                />
                Enable Crypto Card System
              </label>
            </div>
            <div className="setting-item">
              <label>Global Daily Limit (per card)</label>
              <input 
                type="number" 
                value={cardSettings.globalDailyLimit}
                onChange={(e) => setCardSettings({ ...cardSettings, globalDailyLimit: parseInt(e.target.value) })}
              />
            </div>
            <div className="setting-item">
              <label>Global Monthly Limit (per card)</label>
              <input 
                type="number" 
                value={cardSettings.globalMonthlyLimit}
                onChange={(e) => setCardSettings({ ...cardSettings, globalMonthlyLimit: parseInt(e.target.value) })}
              />
            </div>
            <div className="setting-item">
              <label>Default Card Limit (new cards)</label>
              <input 
                type="number" 
                value={cardSettings.defaultCardLimit}
                onChange={(e) => setCardSettings({ ...cardSettings, defaultCardLimit: parseInt(e.target.value) })}
              />
            </div>
          </div>

          <div className="settings-group">
            <h3>Merchant Controls</h3>
            <div className="setting-item">
              <label>Blocked Merchant Categories</label>
              <div className="blocked-merchants">
                {cardSettings.blockedMerchants.map((merchant, idx) => (
                  <span key={idx} className="blocked-tag">
                    {merchant} <button onClick={() => {
                      const newBlocked = cardSettings.blockedMerchants.filter((_, i) => i !== idx);
                      setCardSettings({ ...cardSettings, blockedMerchants: newBlocked });
                    }}>×</button>
                  </span>
                ))}
                <input 
                  type="text" 
                  placeholder="Add category..."
                  className="add-merchant-input"
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      const value = (e.target as HTMLInputElement).value;
                      if (value) {
                        setCardSettings({ 
                          ...cardSettings, 
                          blockedMerchants: [...cardSettings.blockedMerchants, value] 
                        });
                        (e.target as HTMLInputElement).value = '';
                      }
                    }
                  }}
                />
              </div>
            </div>
          </div>

          <div className="settings-group">
            <h3>Card Branding</h3>
            <div className="branding-options">
              <label className="branding-item">
                <input type="checkbox" defaultChecked /> Visa
              </label>
              <label className="branding-item">
                <input type="checkbox" defaultChecked /> Mastercard
              </label>
              <label className="branding-item">
                <input type="checkbox" defaultChecked /> UnionPay
              </label>
              <label className="branding-item">
                <input type="checkbox" /> American Express
              </label>
            </div>
          </div>

          <div className="settings-actions">
            <button className="save-btn">Save Settings</button>
          </div>
        </div>
      )}

      {/* Card Detail Modal */}
      {showCardModal && selectedCard && (
        <div className="modal-overlay" onClick={() => setShowCardModal(false)}>
          <div className="modal-content" onClick={e => e.stopPropagation()}>
            <h2>Card Details</h2>
            <div className="card-preview">
              <div className={`card-brand ${selectedCard.brand.toLowerCase().replace(' ', '-')}`}>
                <span className="card-brand-name">{selectedCard.brand}</span>
                <span className="card-type">{selectedCard.type}</span>
              </div>
              <div className="card-number-display">{selectedCard.cardNumber}</div>
              <div className="card-holder-display">{selectedCard.cardHolder}</div>
              <div className="card-expires">Expires: {selectedCard.expiresAt}</div>
            </div>
            <div className="card-details">
              <div className="detail-row">
                <label>User ID:</label>
                <span>{selectedCard.userId}</span>
              </div>
              <div className="detail-row">
                <label>Status:</label>
                <span 
                  className="status-badge"
                  style={{ backgroundColor: getStatusColor(selectedCard.status) }}
                >
                  {selectedCard.status}
                </span>
              </div>
              <div className="detail-row">
                <label>Balance:</label>
                <span>${selectedCard.balance.toLocaleString()}</span>
              </div>
              <div className="detail-row">
                <label>Daily Limit:</label>
                <span>${selectedCard.dailyLimit.toLocaleString()}</span>
              </div>
              <div className="detail-row">
                <label>Daily Spent:</label>
                <span>${selectedCard.dailySpent.toLocaleString()}</span>
              </div>
              <div className="detail-row">
                <label>Monthly Limit:</label>
                <span>${selectedCard.monthlyLimit.toLocaleString()}</span>
              </div>
              <div className="detail-row">
                <label>Monthly Spent:</label>
                <span>${selectedCard.monthlySpent.toLocaleString()}</span>
              </div>
              <div className="detail-row">
                <label>Created:</label>
                <span>{selectedCard.createdAt}</span>
              </div>
              <div className="detail-row">
                <label>Last Transaction:</label>
                <span>{selectedCard.lastTransaction || 'N/A'}</span>
              </div>
            </div>
            <div className="modal-actions">
              <button className="cancel-btn" onClick={() => setShowCardModal(false)}>
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Update Limits Modal */}
      {showLimitModal && selectedCard && (
        <div className="modal-overlay" onClick={() => setShowLimitModal(false)}>
          <div className="modal-content small" onClick={e => e.stopPropagation()}>
            <h2>Update Card Limits</h2>
            <div className="limit-form">
              <div className="form-group">
                <label>Daily Limit ($)</label>
                <input 
                  type="number" 
                  defaultValue={selectedCard.dailyLimit}
                  id="dailyLimitInput"
                />
              </div>
              <div className="form-group">
                <label>Monthly Limit ($)</label>
                <input 
                  type="number" 
                  defaultValue={selectedCard.monthlyLimit}
                  id="monthlyLimitInput"
                />
              </div>
            </div>
            <div className="modal-actions">
              <button className="cancel-btn" onClick={() => setShowLimitModal(false)}>
                Cancel
              </button>
              <button 
                className="submit-btn"
                onClick={() => {
                  const dailyLimit = parseInt((document.getElementById('dailyLimitInput') as HTMLInputElement).value);
                  const monthlyLimit = parseInt((document.getElementById('monthlyLimitInput') as HTMLInputElement).value);
                  handleUpdateLimits(selectedCard.id, dailyLimit, monthlyLimit);
                }}
              >
                Update Limits
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default CryptoCardPage;
