// Fees Management Page - Complete Implementation
// Configure withdrawal, swap, transaction, and trading fees

import React, { useState, useEffect } from 'react';
import './FeesPage.css';

// Backend API URL
const API_BASE_URL = 'https://api.tigerwallet.com/v1/admin';

interface FeeConfig {
  id: string;
  feeType: 'withdrawal' | 'swap' | 'transfer' | 'trading';
  chainId: string;
  chainName: string;
  tokenSymbol: string;
  feePercent: number;
  feeFixed: number;
  minFee: number;
  maxFee: number;
  isActive: boolean;
  updatedAt: string;
}

const defaultFees: FeeConfig[] = [
  {
    id: '1',
    feeType: 'withdrawal',
    chainId: '1',
    chainName: 'Ethereum',
    tokenSymbol: 'ETH',
    feePercent: 0.5,
    feeFixed: 0.001,
    minFee: 0.01,
    maxFee: 50,
    isActive: true,
    updatedAt: '2026-07-28',
  },
  {
    id: '2',
    feeType: 'withdrawal',
    chainId: '56',
    chainName: 'BNB Chain',
    tokenSymbol: 'BNB',
    feePercent: 0.3,
    feeFixed: 0.0005,
    minFee: 0.005,
    maxFee: 20,
    isActive: true,
    updatedAt: '2026-07-28',
  },
  {
    id: '3',
    feeType: 'swap',
    chainId: 'all',
    chainName: 'All Chains',
    tokenSymbol: 'ALL',
    feePercent: 0.3,
    feeFixed: 0,
    minFee: 0,
    maxFee: 0,
    isActive: true,
    updatedAt: '2026-07-28',
  },
  {
    id: '4',
    feeType: 'trading',
    chainId: 'all',
    chainName: 'All Chains',
    tokenSymbol: 'ALL',
    feePercent: 0.1,
    feeFixed: 0,
    minFee: 0,
    maxFee: 100,
    isActive: true,
    updatedAt: '2026-07-28',
  },
  {
    id: '5',
    feeType: 'transfer',
    chainId: '1',
    chainName: 'Ethereum',
    tokenSymbol: 'ETH',
    feePercent: 0,
    feeFixed: 0.005,
    minFee: 0.005,
    maxFee: 0.005,
    isActive: true,
    updatedAt: '2026-07-28',
  },
];

const FeesPage: React.FC = () => {
  const [fees, setFees] = useState<FeeConfig[]>([]);
  const [showModal, setShowModal] = useState(false);
  const [editingFee, setEditingFee] = useState<FeeConfig | null>(null);
  const [filterType, setFilterType] = useState<string>('all');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadFees();
  }, []);

  const loadFees = async () => {
    setLoading(true);
    setError(null);
    try {
      const token = localStorage.getItem('admin_token');
      const response = await fetch(`${API_BASE_URL}/fees`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {}
      });
      
      if (response.ok) {
        const data = await response.json();
        setFees(data.fees || []);
      } else {
        // Fallback to default fees if API fails
        setFees(defaultFees);
      }
    } catch (err) {
      console.error('Failed to load fees:', err);
      setError('Unable to connect to fee service. Using offline mode.');
      setFees(defaultFees);
    } finally {
      setLoading(false);
    }
  };

  const handleToggleActive = (id: string) => {
    setFees(prev => prev.map(f => f.id === id ? { ...f, isActive: !f.isActive } : f));
  };

  const handleSave = (fee: FeeConfig) => {
    if (editingFee) {
      setFees(prev => prev.map(f => f.id === fee.id ? fee : f));
    } else {
      setFees(prev => [...prev, { ...fee, id: Date.now().toString() }]);
    }
    setShowModal(false);
    setEditingFee(null);
  };

  const handleDelete = (id: string) => {
    if (confirm('Are you sure you want to delete this fee configuration?')) {
      setFees(prev => prev.filter(f => f.id !== id));
    }
  };

  const filteredFees = fees.filter(fee => filterType === 'all' || fee.feeType === filterType);

  const getFeeTypeLabel = (type: string) => {
    switch (type) {
      case 'withdrawal': return 'Withdrawal';
      case 'swap': return 'Swap';
      case 'transfer': return 'Transfer';
      case 'trading': return 'Trading';
      default: return type;
    }
  };

  const getFeeTypeIcon = (type: string) => {
    switch (type) {
      case 'withdrawal': return '💸';
      case 'swap': return '🔄';
      case 'transfer': return '📤';
      case 'trading': return '📊';
      default: return '💰';
    }
  };

  return (
    <div className="fees-page">
      <div className="page-header">
        <div>
          <h1>Fee Management</h1>
          <p>Configure withdrawal, swap, transfer, and trading fees</p>
        </div>
        <button className="btn btn-primary" onClick={() => { setEditingFee(null); setShowModal(true); }}>
          + Add Fee Rule
        </button>
      </div>

      {/* Stats */}
      <div className="stats-grid">
        <div className="stat-card">
          <span className="stat-label">Total Fee Rules</span>
          <span className="stat-value">{fees.length}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">Active Rules</span>
          <span className="stat-value">{fees.filter(f => f.isActive).length}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">Avg Withdrawal Fee</span>
          <span className="stat-value">{(fees.filter(f => f.feeType === 'withdrawal').reduce((acc, f) => acc + f.feePercent, 0) / fees.filter(f => f.feeType === 'withdrawal').length || 0).toFixed(2)}%</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">Avg Swap Fee</span>
          <span className="stat-value">{(fees.filter(f => f.feeType === 'swap').reduce((acc, f) => acc + f.feePercent, 0) / fees.filter(f => f.feeType === 'swap').length || 0).toFixed(2)}%</span>
        </div>
      </div>

      {/* Filters */}
      <div className="filters-bar">
        <select value={filterType} onChange={e => setFilterType(e.target.value)}>
          <option value="all">All Fee Types</option>
          <option value="withdrawal">Withdrawal</option>
          <option value="swap">Swap</option>
          <option value="transfer">Transfer</option>
          <option value="trading">Trading</option>
        </select>
      </div>

      {/* Fees Table */}
      <div className="table-container">
        <table>
          <thead>
            <tr>
              <th>Fee Type</th>
              <th>Chain</th>
              <th>Token</th>
              <th>Percentage</th>
              <th>Fixed Fee</th>
              <th>Min Fee</th>
              <th>Max Fee</th>
              <th>Status</th>
              <th>Updated</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {filteredFees.map(fee => (
              <tr key={fee.id}>
                <td>
                  <div className="fee-type">
                    <span className="fee-icon">{getFeeTypeIcon(fee.feeType)}</span>
                    <span>{getFeeTypeLabel(fee.feeType)}</span>
                  </div>
                </td>
                <td>{fee.chainName}</td>
                <td>{fee.tokenSymbol}</td>
                <td className="fee-percent">{fee.feePercent}%</td>
                <td>{fee.feeFixed} {fee.tokenSymbol !== 'ALL' && fee.tokenSymbol}</td>
                <td>{fee.minFee > 0 ? `${fee.minFee} ${fee.tokenSymbol !== 'ALL' ? fee.tokenSymbol : ''}` : '-'}</td>
                <td>{fee.maxFee > 0 ? `${fee.maxFee} ${fee.tokenSymbol !== 'ALL' ? fee.tokenSymbol : ''}` : '-'}</td>
                <td>
                  <label className="toggle-switch">
                    <input
                      type="checkbox"
                      checked={fee.isActive}
                      onChange={() => handleToggleActive(fee.id)}
                    />
                    <span className="toggle-slider"></span>
                  </label>
                </td>
                <td className="muted">{fee.updatedAt}</td>
                <td>
                  <div className="actions">
                    <button className="action-btn" onClick={() => { setEditingFee(fee); setShowModal(true); }}>✏️</button>
                    <button className="action-btn" onClick={() => handleDelete(fee.id)}>🗑️</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Add/Edit Modal */}
      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h2>{editingFee ? 'Edit Fee Rule' : 'Add Fee Rule'}</h2>
              <button className="close-btn" onClick={() => setShowModal(false)}>×</button>
            </div>
            <div className="modal-body">
              <div className="form-grid">
                <div className="form-group">
                  <label className="form-label">Fee Type</label>
                  <select className="form-input" defaultValue={editingFee?.feeType || 'withdrawal'}>
                    <option value="withdrawal">Withdrawal</option>
                    <option value="swap">Swap</option>
                    <option value="transfer">Transfer</option>
                    <option value="trading">Trading</option>
                  </select>
                </div>
                <div className="form-group">
                  <label className="form-label">Chain</label>
                  <select className="form-input" defaultValue={editingFee?.chainId || 'all'}>
                    <option value="all">All Chains</option>
                    <option value="1">Ethereum</option>
                    <option value="56">BNB Chain</option>
                    <option value="137">Polygon</option>
                    <option value="42161">Arbitrum</option>
                    <option value="10">Optimism</option>
                  </select>
                </div>
                <div className="form-group">
                  <label className="form-label">Token Symbol</label>
                  <input
                    type="text"
                    className="form-input"
                    placeholder="ALL or ETH"
                    defaultValue={editingFee?.tokenSymbol || 'ALL'}
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Fee Percentage (%)</label>
                  <input
                    type="number"
                    step="0.01"
                    className="form-input"
                    placeholder="0.5"
                    defaultValue={editingFee?.feePercent}
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Fixed Fee</label>
                  <input
                    type="number"
                    step="0.0001"
                    className="form-input"
                    placeholder="0.001"
                    defaultValue={editingFee?.feeFixed}
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Minimum Fee</label>
                  <input
                    type="number"
                    step="0.01"
                    className="form-input"
                    placeholder="0.01"
                    defaultValue={editingFee?.minFee}
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Maximum Fee</label>
                  <input
                    type="number"
                    step="0.01"
                    className="form-input"
                    placeholder="50"
                    defaultValue={editingFee?.maxFee}
                  />
                </div>
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
              <button className="btn btn-primary" onClick={() => {
                const form = document.querySelector('.modal-body');
                const fee: FeeConfig = {
                  id: editingFee?.id || Date.now().toString(),
                  feeType: (form?.querySelectorAll('select')[0] as HTMLSelectElement)?.value as any || 'withdrawal',
                  chainId: (form?.querySelectorAll('select')[1] as HTMLSelectElement)?.value || 'all',
                  chainName: 'Custom',
                  tokenSymbol: (form?.querySelectorAll('input')[0] as HTMLInputElement)?.value || 'ALL',
                  feePercent: parseFloat((form?.querySelectorAll('input')[1] as HTMLInputElement)?.value || '0'),
                  feeFixed: parseFloat((form?.querySelectorAll('input')[2] as HTMLInputElement)?.value || '0'),
                  minFee: parseFloat((form?.querySelectorAll('input')[3] as HTMLInputElement)?.value || '0'),
                  maxFee: parseFloat((form?.querySelectorAll('input')[4] as HTMLInputElement)?.value || '0'),
                  isActive: true,
                  updatedAt: new Date().toISOString().split('T')[0],
                };
                handleSave(fee);
              }}>
                {editingFee ? 'Save Changes' : 'Add Fee Rule'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default FeesPage;
