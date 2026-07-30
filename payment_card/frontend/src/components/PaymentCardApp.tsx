/**
 * TigerWallet Payment Card - Frontend
 * Complete card management UI with dark/light theme
 */

import React, { useState, useEffect, useCallback } from 'react';

// ============================================================================
// Types
// ============================================================================

interface CardToken {
  token_id: string;
  last_four: string;
  card_type: string;
  network: string;
  exp_month: number;
  exp_year: number;
  cardholder_name: string;
  created_at: number;
  last_used_at: number;
  is_frozen: boolean;
  daily_limit: number;
  monthly_limit: number;
}

interface Transaction {
  transaction_id: number;
  user_id: number;
  card_id: string;
  transaction_type: string;
  status: string;
  amount: number;
  original_amount: number;
  currency: string;
  merchant_id: number;
  merchant_name: string;
  merchant_category: string;
  country: string;
  fees: number;
  cashback: number;
  timestamp: number;
  processed_at: number;
  risk_score: number;
  auth_code: string;
}

interface CardStats {
  total_cards: number;
  active_cards: number;
  total_volume: number;
  volume_24h: number;
  total_transactions: number;
  approved_count: number;
  declined_count: number;
  avg_transaction_size: number;
}

interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: { code: string; message: string };
}

// ============================================================================
// API Service
// ============================================================================

const API_BASE = process.env.NEXT_PUBLIC_PAYMENT_API_URL || 'http://localhost:8444/api/v1';

class PaymentAPI {
  private baseUrl: string;

  constructor(baseUrl: string = API_BASE) {
    this.baseUrl = baseUrl;
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    });
    const data: APIResponse<T> = await response.json();
    if (!data.success || !data.data) throw new Error(data.error?.message || 'Request failed');
    return data.data;
  }

  async getCards(userId: string = '1'): Promise<CardToken[]> {
    return this.request<CardToken[]>('/cards');
  }

  async createCard(card: {
    card_type: string;
    network: string;
    cardholder_name: string;
    exp_month: number;
    exp_year: number;
  }): Promise<CardToken> {
    return this.request<CardToken>('/cards', { method: 'POST', body: JSON.stringify(card) });
  }

  async freezeCard(tokenId: string): Promise<void> {
    await this.request<void>(`/cards/${tokenId}/freeze`, { method: 'POST' });
  }

  async unfreezeCard(tokenId: string): Promise<void> {
    await this.request<void>(`/cards/${tokenId}/unfreeze`, { method: 'POST' });
  }

  async authorize(req: {
    user_id: number;
    card_token: string;
    amount: number;
    currency: string;
    merchant_id: number;
    merchant_name: string;
    merchant_category: string;
    country: string;
  }): Promise<any> {
    return this.request<any>('/authorize', { method: 'POST', body: JSON.stringify(req) });
  }

  async getTransactions(userId: string = '1', limit: number = 50): Promise<Transaction[]> {
    return this.request<Transaction[]>(`/transactions?limit=${limit}`);
  }

  async getStats(): Promise<CardStats> {
    return this.request<CardStats>('/stats');
  }
}

const api = new PaymentAPI();

// ============================================================================
// Context
// ============================================================================

interface PaymentContextType {
  cards: CardToken[];
  transactions: Transaction[];
  stats: CardStats | null;
  loading: boolean;
  error: string | null;
  theme: 'light' | 'dark';
  setTheme: (t: 'light' | 'dark') => void;
  refreshCards: () => Promise<void>;
  refreshTransactions: () => Promise<void>;
  refreshStats: () => Promise<void>;
  createCard: (card: any) => Promise<CardToken>;
  freezeCard: (tokenId: string) => Promise<void>;
  unfreezeCard: (tokenId: string) => Promise<void>;
}

const PaymentContext = React.createContext<PaymentContextType | null>(null);

export function usePayment() {
  const ctx = React.useContext(PaymentContext);
  if (!ctx) throw new Error('usePayment must be within PaymentProvider');
  return ctx;
}

export function PaymentProvider({ children }: { children: React.ReactNode }) {
  const [cards, setCards] = useState<CardToken[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [stats, setStats] = useState<CardStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [theme, setTheme] = useState<'light' | 'dark'>('dark');

  const refreshCards = useCallback(async () => {
    try {
      const data = await api.getCards();
      setCards(data);
    } catch (e) { setError((e as Error).message); }
  }, []);

  const refreshTransactions = useCallback(async () => {
    try {
      const data = await api.getTransactions();
      setTransactions(data);
    } catch (e) { setError((e as Error).message); }
  }, []);

  const refreshStats = useCallback(async () => {
    try {
      const data = await api.getStats();
      setStats(data);
    } catch (e) { console.error(e); }
  }, []);

  const createCard = useCallback(async (card: any) => {
    const newCard = await api.createCard(card);
    await refreshCards();
    return newCard;
  }, [refreshCards]);

  const freezeCard = useCallback(async (tokenId: string) => {
    await api.freezeCard(tokenId);
    await refreshCards();
  }, [refreshCards]);

  const unfreezeCard = useCallback(async (tokenId: string) => {
    await api.unfreezeCard(tokenId);
    await refreshCards();
  }, [refreshCards]);

  useEffect(() => {
    const init = async () => {
      setLoading(true);
      await Promise.all([refreshCards(), refreshTransactions(), refreshStats()]);
      setLoading(false);
    };
    init();
  }, [refreshCards, refreshTransactions, refreshStats]);

  return (
    <PaymentContext.Provider value={{
      cards, transactions, stats, loading, error, theme, setTheme,
      refreshCards, refreshTransactions, refreshStats,
      createCard, freezeCard, unfreezeCard
    }}>
      <div className="payment-app" data-theme={theme}>{children}</div>
    </PaymentContext.Provider>
  );
}

// ============================================================================
// Format Utilities
// ============================================================================

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(amount / 100);
}

function formatDate(timestamp: number): string {
  return new Date(timestamp).toLocaleDateString('en-US', {
    month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit'
  });
}

// ============================================================================
// Main Component
// ============================================================================

export function PaymentCardApp() {
  const {
    cards, transactions, stats, loading, error, theme, setTheme,
    refreshCards, refreshTransactions, createCard, freezeCard, unfreezeCard
  } = usePayment();

  const [activeTab, setActiveTab] = useState<'cards' | 'transactions'>('cards');
  const [showCreateCard, setShowCreateCard] = useState(false);
  const [newCard, setNewCard] = useState({ card_type: 'debit', network: 'visa', cardholder_name: '', exp_month: 12, exp_year: 2028 });

  const handleCreateCard = async () => {
    try {
      await createCard(newCard);
      setShowCreateCard(false);
      setNewCard({ card_type: 'debit', network: 'visa', cardholder_name: '', exp_month: 12, exp_year: 2028 });
    } catch (e) { alert((e as Error).message); }
  };

  const getNetworkColor = (network: string) => {
    switch (network) {
      case 'visa': return '#1A1F71';
      case 'mastercard': return '#EB001B';
      case 'amex': return '#006FCF';
      default: return '#666';
    }
  };

  return (
    <div className={`payment-card-app ${theme}`} data-theme={theme}>
      {/* Header */}
      <header className="pc-header">
        <div className="pc-header-left">
          <h1 className="pc-title"><span className="pc-icon">💳</span> Payment Cards</h1>
        </div>
        <div className="pc-header-right">
          {stats && (
            <div className="pc-stats">
              <div className="stat"><span className="stat-label">Volume</span><span className="stat-value">{formatCurrency(stats.total_volume)}</span></div>
              <div className="stat"><span className="stat-label">24h</span><span className="stat-value">{formatCurrency(stats.volume_24h)}</span></div>
              <div className="stat"><span className="stat-label">Txns</span><span className="stat-value">{stats.total_transactions}</span></div>
            </div>
          )}
          <button className="theme-toggle" onClick={() => setTheme(theme === 'light' ? 'dark' : 'light')}>
            {theme === 'light' ? '🌙' : '☀️'}
          </button>
        </div>
      </header>

      {/* Tabs */}
      <div className="pc-tabs">
        <button className={`tab ${activeTab === 'cards' ? 'active' : ''}`} onClick={() => setActiveTab('cards')}>
          My Cards ({cards.length})
        </button>
        <button className={`tab ${activeTab === 'transactions' ? 'active' : ''}`} onClick={() => setActiveTab('transactions')}>
          Transactions ({transactions.length})
        </button>
      </div>

      {/* Cards Tab */}
      {activeTab === 'cards' && (
        <div className="pc-cards">
          <div className="pc-actions">
            <button className="add-card-btn" onClick={() => setShowCreateCard(true)}>+ Add New Card</button>
          </div>

          {showCreateCard && (
            <div className="modal-overlay" onClick={() => setShowCreateCard(false)}>
              <div className="modal" onClick={e => e.stopPropagation()}>
                <h2>Add New Card</h2>
                <div className="form-group">
                  <label>Card Type</label>
                  <select value={newCard.card_type} onChange={e => setNewCard({...newCard, card_type: e.target.value})}>
                    <option value="debit">Debit</option>
                    <option value="credit">Credit</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Network</label>
                  <select value={newCard.network} onChange={e => setNewCard({...newCard, network: e.target.value})}>
                    <option value="visa">Visa</option>
                    <option value="mastercard">Mastercard</option>
                    <option value="amex">American Express</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Cardholder Name</label>
                  <input type="text" value={newCard.cardholder_name} onChange={e => setNewCard({...newCard, cardholder_name: e.target.value})} placeholder="John Doe" />
                </div>
                <div className="form-row">
                  <div className="form-group">
                    <label>Expiry Month</label>
                    <input type="number" value={newCard.exp_month} onChange={e => setNewCard({...newCard, exp_month: parseInt(e.target.value)})} min={1} max={12} />
                  </div>
                  <div className="form-group">
                    <label>Expiry Year</label>
                    <input type="number" value={newCard.exp_year} onChange={e => setNewCard({...newCard, exp_year: parseInt(e.target.value)})} min={2024} max={2035} />
                  </div>
                </div>
                <div className="modal-actions">
                  <button className="cancel-btn" onClick={() => setShowCreateCard(false)}>Cancel</button>
                  <button className="confirm-btn" onClick={handleCreateCard}>Add Card</button>
                </div>
              </div>
            </div>
          )}

          {loading ? <div className="loading">Loading...</div> : error ? <div className="error">{error}</div> : (
            <div className="cards-grid">
              {cards.map(card => (
                <div key={card.token_id} className={`card-item ${card.is_frozen ? 'frozen' : ''}`}>
                  <div className="card-visual" style={{ background: `linear-gradient(135deg, ${getNetworkColor(card.network)} 0%, #333 100%)` }}>
                    <div className="card-network">{card.network.toUpperCase()}</div>
                    <div className="card-number">•••• •••• •••• {card.last_four}</div>
                    <div className="card-details">
                      <div className="card-holder">{card.cardholder_name}</div>
                      <div className="card-expiry">{String(card.exp_month).padStart(2, '0')}/{card.exp_year}</div>
                    </div>
                    <div className="card-type">{card.card_type.toUpperCase()}</div>
                  </div>
                  <div className="card-footer">
                    <span className={`status ${card.is_frozen ? 'frozen' : 'active'}`}>
                      {card.is_frozen ? '❄️ Frozen' : '✅ Active'}
                    </span>
                    <div className="card-actions">
                      {card.is_frozen ? (
                        <button onClick={() => unfreezeCard(card.token_id)}>Unfreeze</button>
                      ) : (
                        <button onClick={() => freezeCard(card.token_id)}>Freeze</button>
                      )}
                    </div>
                  </div>
                  <div className="card-limits">
                    <div className="limit">Daily: {formatCurrency(card.daily_limit)}</div>
                    <div className="limit">Monthly: {formatCurrency(card.monthly_limit)}</div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Transactions Tab */}
      {activeTab === 'transactions' && (
        <div className="pc-transactions">
          <h2>Transaction History</h2>
          {transactions.length === 0 ? (
            <div className="empty">No transactions yet</div>
          ) : (
            <div className="transactions-list">
              {transactions.map(tx => (
                <div key={tx.transaction_id} className="transaction-item">
                  <div className="tx-main">
                    <span className={`tx-type ${tx.transaction_type}`}>{tx.transaction_type}</span>
                    <span className="tx-merchant">{tx.merchant_name}</span>
                    <span className="tx-date">{formatDate(tx.timestamp)}</span>
                  </div>
                  <div className="tx-details">
                    <span className={`tx-amount ${tx.status === 'approved' || tx.status === 'completed' ? 'positive' : 'negative'}`}>
                      {tx.transaction_type === 'refund' ? '+' : '-'}{formatCurrency(tx.amount)}
                    </span>
                    {tx.cashback > 0 && <span className="tx-cashback">+{formatCurrency(tx.cashback)} cashback</span>}
                    <span className={`tx-status ${tx.status}`}>{tx.status}</span>
                  </div>
                  {tx.auth_code && <div className="tx-auth">Auth: {tx.auth_code}</div>}
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default PaymentCardApp;
