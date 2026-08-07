/**
 * TigerWallet Admin - Crypto Cards Management Page
 * Complete implementation with backend connectivity
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { cryptoCardAPI } from '../services/api';

interface CryptoCard {
  id: string;
  userId: string;
  userName: string;
  cardNumber: string;
  currency: string;
  balance: number;
  limit: number;
  status: 'active' | 'blocked' | 'pending';
  type: 'virtual' | 'physical';
  createdAt: string;
}

export const CryptoCardsPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [cards, setCards] = useState<CryptoCard[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('all');
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    loadCards();
  }, [filter]);

  const loadCards = async () => {
    setLoading(true);
    try {
      const response = await cryptoCardAPI.getAll(filter !== 'all' ? filter : undefined);
      setCards(response.data);
    } catch (error) {
      console.error('Failed to load cards:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleBlockCard = async (cardId: string) => {
    try {
      await cryptoCardAPI.block(cardId);
      loadCards();
    } catch (error) {
      console.error('Failed to block card:', error);
    }
  };

  const handleActivateCard = async (cardId: string) => {
    try {
      await cryptoCardAPI.activate(cardId);
      loadCards();
    } catch (error) {
      console.error('Failed to activate card:', error);
    }
  };

  const filteredCards = cards.filter(card =>
    card.userName.toLowerCase().includes(searchTerm.toLowerCase()) ||
    card.cardNumber.includes(searchTerm)
  );

  return (
    <div className={`page-container ${isDark ? 'dark' : 'light'}`}>
      <div className="page-header">
        <h1>Crypto Cards Management</h1>
        <button className="theme-btn" onClick={toggleTheme}>
          {isDark ? '☀️ Light' : '🌙 Dark'}
        </button>
      </div>

      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-value">{cards.length}</div>
          <div className="stat-label">Total Cards</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{cards.filter(c => c.status === 'active').length}</div>
          <div className="stat-label">Active</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">${cards.reduce((sum, c) => sum + c.balance, 0).toLocaleString()}</div>
          <div className="stat-label">Total Balance</div>
        </div>
      </div>

      <div className="filters">
        <select value={filter} onChange={(e) => setFilter(e.target.value)}>
          <option value="all">All Cards</option>
          <option value="active">Active</option>
          <option value="blocked">Blocked</option>
          <option value="pending">Pending</option>
        </select>
        <input
          type="text"
          placeholder="Search..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
        />
      </div>

      {loading ? (
        <div className="loading">Loading cards...</div>
      ) : (
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th>Card</th>
                <th>User</th>
                <th>Type</th>
                <th>Currency</th>
                <th>Balance</th>
                <th>Limit</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredCards.map(card => (
                <tr key={card.id}>
                  <td>•••• {card.cardNumber.slice(-4)}</td>
                  <td>{card.userName}</td>
                  <td>{card.type}</td>
                  <td>{card.currency}</td>
                  <td>${card.balance.toLocaleString()}</td>
                  <td>${card.limit.toLocaleString()}</td>
                  <td>
                    <span className={`status-badge ${card.status}`}>
                      {card.status}
                    </span>
                  </td>
                  <td>
                    {card.status === 'active' ? (
                      <button className="btn-danger" onClick={() => handleBlockCard(card.id)}>
                        Block
                      </button>
                    ) : (
                      <button className="btn-success" onClick={() => handleActivateCard(card.id)}>
                        Activate
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};

export default CryptoCardsPage;
