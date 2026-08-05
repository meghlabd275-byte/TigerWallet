// TigerWallet Admin - Trading Pairs Page
// Manage trading pairs

import React, { useState, useEffect } from 'react';
import { adminApi } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

interface TradingPair {
  id: string;
  baseToken: string;
  quoteToken: string;
  baseSymbol: string;
  quoteSymbol: string;
  price: string;
  priceChange24h: string;
  volume24h: string;
  liquidity: string;
  isActive: boolean;
}

const PairsPage: React.FC = () => {
  const { resolvedTheme } = useTheme();
  const [pairs, setPairs] = useState<TradingPair[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newPair, setNewPair] = useState({
    baseToken: '',
    quoteToken: '',
    minTradeAmount: '0',
    maxTradeAmount: '1000000',
    makerFee: '0.001',
    takerFee: '0.002',
  });

  useEffect(() => {
    loadPairs();
  }, []);

  const loadPairs = async () => {
    try {
      setLoading(true);
      const response = await adminApi.getPairs({});
      setPairs(response.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load pairs');
    } finally {
      setLoading(false);
    }
  };

  const getColors = () => ({
    text: resolvedTheme === 'dark' ? '#f9fafb' : '#111827',
    textSecondary: resolvedTheme === 'dark' ? '#9ca3af' : '#6b7280',
    bgCard: resolvedTheme === 'dark' ? '#1e293b' : '#ffffff',
    border: resolvedTheme === 'dark' ? '#374151' : '#e5e7eb',
    primary: '#dc2626',
  });

  const colors = getColors();

  const handleCreatePair = async () => {
    try {
      await adminApi.createPair(newPair);
      setShowCreateModal(false);
      loadPairs();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create pair');
    }
  };

  const handleUpdateStatus = async (id: string, active: boolean) => {
    try {
      await adminApi.updatePairStatus(id, active);
      loadPairs();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update pair');
    }
  };

  const handleDeletePair = async (id: string) => {
    if (!confirm('Are you sure you want to delete this pair?')) return;
    try {
      await adminApi.deletePair(id);
      loadPairs();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete pair');
    }
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Trading Pairs</h1>
        <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>
          + Create Pair
        </button>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      <div className="card" style={{ backgroundColor: colors.bgCard }}>
        <div className="card-body p-0">
          {loading ? (
            <div className="flex items-center justify-center p-8">
              <div className="loader"></div>
            </div>
          ) : (
            <table className="table">
              <thead>
                <tr>
                  <th>Pair</th>
                  <th>Price</th>
                  <th>24h Change</th>
                  <th>24h Volume</th>
                  <th>Liquidity</th>
                  <th>Status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {pairs.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="text-center py-8" style={{ color: colors.textSecondary }}>
                      No pairs found
                    </td>
                  </tr>
                ) : (
                  pairs.map((pair) => (
                    <tr key={pair.id}>
                      <td style={{ color: colors.text }}>
                        {pair.baseSymbol}/{pair.quoteSymbol}
                      </td>
                      <td style={{ color: colors.text }}>${pair.price}</td>
                      <td style={{ color: parseFloat(pair.priceChange24h) >= 0 ? '#22c55e' : '#ef4444' }}>
                        {pair.priceChange24h}%
                      </td>
                      <td style={{ color: colors.textSecondary }}>${pair.volume24h}</td>
                      <td style={{ color: colors.textSecondary }}>${pair.liquidity}</td>
                      <td>
                        <span className={`badge ${pair.isActive ? 'badge-success' : 'badge-neutral'}`}>
                          {pair.isActive ? 'Active' : 'Inactive'}
                        </span>
                      </td>
                      <td>
                        <div className="flex gap-2">
                          <button
                            className="btn btn-sm btn-outline"
                            onClick={() => handleUpdateStatus(pair.id, !pair.isActive)}
                          >
                            {pair.isActive ? 'Disable' : 'Enable'}
                          </button>
                          <button
                            className="btn btn-sm btn-danger"
                            onClick={() => handleDeletePair(pair.id)}
                          >
                            Delete
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {showCreateModal && (
        <div className="modal-overlay" onClick={() => setShowCreateModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Create Trading Pair</h3>
              <button className="btn btn-sm btn-outline" onClick={() => setShowCreateModal(false)}>✕</button>
            </div>
            <div className="modal-body">
              <div className="form-group">
                <label className="form-label">Base Token</label>
                <input
                  type="text"
                  className="form-input"
                  value={newPair.baseToken}
                  onChange={(e) => setNewPair({ ...newPair, baseToken: e.target.value })}
                />
              </div>
              <div className="form-group">
                <label className="form-label">Quote Token</label>
                <input
                  type="text"
                  className="form-input"
                  value={newPair.quoteToken}
                  onChange={(e) => setNewPair({ ...newPair, quoteToken: e.target.value })}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="form-group">
                  <label className="form-label">Maker Fee</label>
                  <input
                    type="text"
                    className="form-input"
                    value={newPair.makerFee}
                    onChange={(e) => setNewPair({ ...newPair, makerFee: e.target.value })}
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Taker Fee</label>
                  <input
                    type="text"
                    className="form-input"
                    value={newPair.takerFee}
                    onChange={(e) => setNewPair({ ...newPair, takerFee: e.target.value })}
                  />
                </div>
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowCreateModal(false)}>Cancel</button>
              <button className="btn btn-primary" onClick={handleCreatePair}>Create</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default PairsPage;
