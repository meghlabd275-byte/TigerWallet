// TigerWallet Admin - Chains (Blockchains) Page
// Manage blockchain configurations

import React, { useState, useEffect } from 'react';
import { adminApi } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

interface Blockchain {
  id: string;
  name: string;
  symbol: string;
  chainId: number;
  rpcUrl: string;
  explorerUrl: string;
  isActive: boolean;
  isTestnet: boolean;
  nativeToken: string;
  avgBlockTime: number;
}

const ChainsPage: React.FC = () => {
  const { resolvedTheme } = useTheme();
  const [chains, setChains] = useState<Blockchain[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newChain, setNewChain] = useState({
    name: '',
    symbol: '',
    chainId: 1,
    rpcUrl: '',
    explorerUrl: '',
    nativeToken: '',
    avgBlockTime: 15,
    isTestnet: false,
  });

  useEffect(() => {
    loadChains();
  }, []);

  const loadChains = async () => {
    try {
      setLoading(true);
      const response = await adminApi.getBlockchains();
      setChains(response.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load blockchains');
    } finally {
      setLoading(false);
    }
  };

  const getColors = () => ({
    text: resolvedTheme === 'dark' ? '#f9fafb' : '#111827',
    textSecondary: resolvedTheme === 'dark' ? '#9ca3af' : '#6b7280',
    bgCard: resolvedTheme === 'dark' ? '#1e293b' : '#ffffff',
    border: resolvedTheme === 'dark' ? '#374151' : '#e5e7eb',
  });

  const colors = getColors();

  const handleCreateChain = async () => {
    try {
      await adminApi.createBlockchain(newChain);
      setShowCreateModal(false);
      loadChains();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create blockchain');
    }
  };

  const handleUpdateChain = async (id: string, active: boolean) => {
    try {
      await adminApi.updateBlockchain(id, { isActive: active });
      loadChains();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update blockchain');
    }
  };

  const handleDeleteChain = async (id: string) => {
    if (!confirm('Are you sure you want to delete this blockchain?')) return;
    try {
      await adminApi.deleteBlockchain(id);
      loadChains();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete blockchain');
    }
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Blockchains</h1>
        <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>
          + Add Blockchain
        </button>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {loading ? (
          <div className="col-span-full flex items-center justify-center p-8">
            <div className="loader"></div>
          </div>
        ) : chains.length === 0 ? (
          <div className="col-span-full text-center py-8" style={{ color: colors.textSecondary }}>
            No blockchains found
          </div>
        ) : (
          chains.map((chain) => (
            <div key={chain.id} className="card" style={{ backgroundColor: colors.bgCard }}>
              <div className="card-body">
                <div className="flex justify-between items-start mb-4">
                  <div>
                    <h3 className="font-semibold text-lg" style={{ color: colors.text }}>
                      {chain.name}
                    </h3>
                    <span className="badge badge-neutral">{chain.symbol}</span>
                    {chain.isTestnet && <span className="badge badge-warning ml-2">Testnet</span>}
                  </div>
                  <span className={`badge ${chain.isActive ? 'badge-success' : 'badge-neutral'}`}>
                    {chain.isActive ? 'Active' : 'Inactive'}
                  </span>
                </div>
                <div className="text-sm" style={{ color: colors.textSecondary }}>
                  <p>Chain ID: {chain.chainId}</p>
                  <p>Native Token: {chain.nativeToken}</p>
                  <p>Block Time: ~{chain.avgBlockTime}s</p>
                </div>
                <div className="flex gap-2 mt-4">
                  <button
                    className="btn btn-sm btn-outline flex-1"
                    onClick={() => handleUpdateChain(chain.id, !chain.isActive)}
                  >
                    {chain.isActive ? 'Disable' : 'Enable'}
                  </button>
                  <button
                    className="btn btn-sm btn-danger"
                    onClick={() => handleDeleteChain(chain.id)}
                  >
                    Delete
                  </button>
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      {showCreateModal && (
        <div className="modal-overlay" onClick={() => setShowCreateModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Add Blockchain</h3>
              <button className="btn btn-sm btn-outline" onClick={() => setShowCreateModal(false)}>✕</button>
            </div>
            <div className="modal-body">
              <div className="form-group">
                <label className="form-label">Name</label>
                <input
                  type="text"
                  className="form-input"
                  value={newChain.name}
                  onChange={(e) => setNewChain({ ...newChain, name: e.target.value })}
                />
              </div>
              <div className="form-group">
                <label className="form-label">Symbol</label>
                <input
                  type="text"
                  className="form-input"
                  value={newChain.symbol}
                  onChange={(e) => setNewChain({ ...newChain, symbol: e.target.value })}
                />
              </div>
              <div className="form-group">
                <label className="form-label">Chain ID</label>
                <input
                  type="number"
                  className="form-input"
                  value={newChain.chainId}
                  onChange={(e) => setNewChain({ ...newChain, chainId: parseInt(e.target.value) })}
                />
              </div>
              <div className="form-group">
                <label className="form-label">RPC URL</label>
                <input
                  type="text"
                  className="form-input"
                  value={newChain.rpcUrl}
                  onChange={(e) => setNewChain({ ...newChain, rpcUrl: e.target.value })}
                />
              </div>
              <div className="form-group">
                <label className="form-label">Explorer URL</label>
                <input
                  type="text"
                  className="form-input"
                  value={newChain.explorerUrl}
                  onChange={(e) => setNewChain({ ...newChain, explorerUrl: e.target.value })}
                />
              </div>
              <div className="form-group">
                <label className="form-label">Native Token</label>
                <input
                  type="text"
                  className="form-input"
                  value={newChain.nativeToken}
                  onChange={(e) => setNewChain({ ...newChain, nativeToken: e.target.value })}
                />
              </div>
              <div className="form-group">
                <label className="form-label flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={newChain.isTestnet}
                    onChange={(e) => setNewChain({ ...newChain, isTestnet: e.target.checked })}
                  />
                  Is Testnet
                </label>
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowCreateModal(false)}>Cancel</button>
              <button className="btn btn-primary" onClick={handleCreateChain}>Create</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default ChainsPage;
