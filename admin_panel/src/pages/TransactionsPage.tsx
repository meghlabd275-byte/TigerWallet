// Transactions Page - Complete Implementation
// View and manage all platform transactions

import React, { useState, useEffect } from 'react';
import './TransactionsPage.css';

interface Transaction {
  id: string;
  txHash: string;
  chainId: string;
  chainName: string;
  type: 'transfer' | 'swap' | 'stake' | 'unstake' | 'bridge' | 'mint' | 'burn' | 'approve';
  fromAddress: string;
  toAddress: string;
  tokenSymbol: string;
  amount: string;
  fee: string;
  status: 'pending' | 'confirmed' | 'failed';
  blockNumber: number;
  confirmations: number;
  timestamp: string;
  userId: string;
}

const TransactionsPage: React.FC = () => {
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [filterType, setFilterType] = useState<string>('all');
  const [filterChain, setFilterChain] = useState<string>('all');
  const [selectedTx, setSelectedTx] = useState<Transaction | null>(null);

  useEffect(() => {
    loadTransactions();
  }, []);

  const loadTransactions = () => {
    setTransactions([
      {
        id: '1',
        txHash: '0x7a23...8f91',
        chainId: '1',
        chainName: 'Ethereum',
        type: 'transfer',
        fromAddress: '0x742d35Cc6634C0532925a3b844Bc9e7595f1234',
        toAddress: '0x8f91...7a23',
        tokenSymbol: 'ETH',
        amount: '5.5',
        fee: '0.005',
        status: 'confirmed',
        blockNumber: 18234567,
        confirmations: 15,
        timestamp: '2026-07-28 14:32:15',
        userId: 'user123',
      },
      {
        id: '2',
        txHash: '0x3b14...2c78',
        chainId: '56',
        chainName: 'BNB Chain',
        type: 'swap',
        fromAddress: '0x111d35Cc6634C0532925a3b844Bc9e7595f5678',
        toAddress: '0x2228914Cc6634C0532925a3b844Bc9e7595f9999',
        tokenSymbol: 'USDT',
        amount: '12500',
        fee: '3.75',
        status: 'confirmed',
        blockNumber: 29384756,
        confirmations: 20,
        timestamp: '2026-07-28 14:28:42',
        userId: 'user456',
      },
      {
        id: '3',
        txHash: '0x9f42...1a63',
        chainId: '42161',
        chainName: 'Arbitrum',
        type: 'bridge',
        fromAddress: '0x333d35Cc6634C0532925a3b844Bc9e7595f1111',
        toAddress: '0x444e35Cc6634C0532925a3b844Bc9e7595f2222',
        tokenSymbol: 'ETH',
        amount: '2.0',
        fee: '0.002',
        status: 'pending',
        blockNumber: 0,
        confirmations: 0,
        timestamp: '2026-07-28 14:25:30',
        userId: 'user789',
      },
      {
        id: '4',
        txHash: '0x2e87...9b12',
        chainId: '137',
        chainName: 'Polygon',
        type: 'stake',
        fromAddress: '0x555d35Cc6634C0532925a3b844Bc9e7595f3333',
        toAddress: '0x666e35Cc6634C0532925a3b844Bc9e7595f4444',
        tokenSymbol: 'MATIC',
        amount: '1000',
        fee: '0.5',
        status: 'confirmed',
        blockNumber: 45678901,
        confirmations: 25,
        timestamp: '2026-07-28 14:20:18',
        userId: 'user321',
      },
      {
        id: '5',
        txHash: '0x5c96...3d45',
        chainId: '10',
        chainName: 'Optimism',
        type: 'approve',
        fromAddress: '0x777f35Cc6634C0532925a3b844Bc9e7595f5555',
        toAddress: '0x888g35Cc6634C0532925a3b844Bc9e7595f6666',
        tokenSymbol: 'USDC',
        amount: 'unlimited',
        fee: '0.001',
        status: 'confirmed',
        blockNumber: 112345678,
        confirmations: 30,
        timestamp: '2026-07-28 14:15:45',
        userId: 'user654',
      },
      {
        id: '6',
        txHash: '0x6d07...4e56',
        chainId: '8453',
        chainName: 'Base',
        type: 'transfer',
        fromAddress: '0x999h35Cc6634C0532925a3b844Bc9e7595f7777',
        toAddress: '0xaaa i35Cc6634C0532925a3b844Bc9e7595f8888',
        tokenSymbol: 'ETH',
        amount: '0.5',
        fee: '0.001',
        status: 'failed',
        blockNumber: 0,
        confirmations: 0,
        timestamp: '2026-07-28 14:10:22',
        userId: 'user987',
      },
    ]);
  };

  const handleStatusUpdate = async (id: string, status: 'confirmed' | 'failed') => {
    setTransactions(prev => prev.map(t => t.id === id ? { ...t, status } : t));
  };

  const filteredTransactions = transactions.filter(tx => {
    const matchesSearch = tx.txHash.toLowerCase().includes(searchQuery.toLowerCase()) ||
                        tx.fromAddress.toLowerCase().includes(searchQuery.toLowerCase()) ||
                        tx.toAddress.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesStatus = filterStatus === 'all' || tx.status === filterStatus;
    const matchesType = filterType === 'all' || tx.type === filterType;
    const matchesChain = filterChain === 'all' || tx.chainId === filterChain;
    return matchesSearch && matchesStatus && matchesType && matchesChain;
  });

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'confirmed': return 'badge-success';
      case 'pending': return 'badge-warning';
      case 'failed': return 'badge-error';
      default: return '';
    }
  };

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'transfer': return '📤';
      case 'swap': return '🔄';
      case 'stake': return '📈';
      case 'unstake': return '📉';
      case 'bridge': return '🌉';
      case 'mint': return '🆕';
      case 'burn': return '🔥';
      case 'approve': return '✅';
      default: return '💰';
    }
  };

  return (
    <div className="transactions-page">
      <div className="page-header">
        <div>
          <h1>Transactions</h1>
          <p>View and manage all platform transactions</p>
        </div>
        <button className="btn btn-primary" onClick={() => loadTransactions()}>
          🔄 Refresh
        </button>
      </div>

      {/* Stats */}
      <div className="stats-grid">
        <div className="stat-card">
          <span className="stat-label">Total Transactions</span>
          <span className="stat-value">{transactions.length.toLocaleString()}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">Confirmed</span>
          <span className="stat-value success">{transactions.filter(t => t.status === 'confirmed').length}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">Pending</span>
          <span className="stat-value warning">{transactions.filter(t => t.status === 'pending').length}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">Failed</span>
          <span className="stat-value error">{transactions.filter(t => t.status === 'failed').length}</span>
        </div>
      </div>

      {/* Filters */}
      <div className="filters-bar">
        <div className="search-box">
          <span>🔍</span>
          <input
            type="text"
            placeholder="Search by hash or address..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
          />
        </div>
        <select value={filterStatus} onChange={e => setFilterStatus(e.target.value)}>
          <option value="all">All Status</option>
          <option value="confirmed">Confirmed</option>
          <option value="pending">Pending</option>
          <option value="failed">Failed</option>
        </select>
        <select value={filterType} onChange={e => setFilterType(e.target.value)}>
          <option value="all">All Types</option>
          <option value="transfer">Transfer</option>
          <option value="swap">Swap</option>
          <option value="stake">Stake</option>
          <option value="bridge">Bridge</option>
          <option value="approve">Approve</option>
        </select>
        <select value={filterChain} onChange={e => setFilterChain(e.target.value)}>
          <option value="all">All Chains</option>
          <option value="1">Ethereum</option>
          <option value="56">BNB Chain</option>
          <option value="137">Polygon</option>
          <option value="42161">Arbitrum</option>
          <option value="10">Optimism</option>
        </select>
      </div>

      {/* Transactions Table */}
      <div className="table-container">
        <table>
          <thead>
            <tr>
              <th>Transaction</th>
              <th>Type</th>
              <th>Chain</th>
              <th>From</th>
              <th>To</th>
              <th>Amount</th>
              <th>Fee</th>
              <th>Status</th>
              <th>Time</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {filteredTransactions.map(tx => (
              <tr key={tx.id} onClick={() => setSelectedTx(tx)} className="clickable">
                <td>
                  <div className="tx-hash">
                    <span className="hash">{tx.txHash}</span>
                    <span className="block">Block #{tx.blockNumber.toLocaleString()}</span>
                  </div>
                </td>
                <td>
                  <div className="tx-type">
                    <span className="type-icon">{getTypeIcon(tx.type)}</span>
                    <span>{tx.type}</span>
                  </div>
                </td>
                <td><span className="chain-badge">{tx.chainName}</span></td>
                <td className="address mono">{tx.fromAddress.slice(0, 10)}...</td>
                <td className="address mono">{tx.toAddress.slice(0, 10)}...</td>
                <td className="amount">{tx.amount} {tx.tokenSymbol}</td>
                <td className="fee">{tx.fee}</td>
                <td>
                  <span className={`badge ${getStatusBadge(tx.status)}`}>
                    {tx.status}
                  </span>
                </td>
                <td className="timestamp">{tx.timestamp}</td>
                <td>
                  <div className="actions" onClick={e => e.stopPropagation()}>
                    <button className="action-btn" title="View Details">👁️</button>
                    {tx.status === 'pending' && (
                      <>
                        <button className="action-btn" onClick={() => handleStatusUpdate(tx.id, 'confirmed')}>✓</button>
                        <button className="action-btn" onClick={() => handleStatusUpdate(tx.id, 'failed')}>✗</button>
                      </>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Transaction Detail Modal */}
      {selectedTx && (
        <div className="modal-overlay" onClick={() => setSelectedTx(null)}>
          <div className="modal modal-large" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h2>Transaction Details</h2>
              <button className="close-btn" onClick={() => setSelectedTx(null)}>×</button>
            </div>
            <div className="modal-body">
              <div className="detail-grid">
                <div className="detail-item">
                  <span className="detail-label">Transaction Hash</span>
                  <span className="detail-value mono">{selectedTx.txHash}</span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">Type</span>
                  <span className="detail-value">{selectedTx.type}</span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">Chain</span>
                  <span className="detail-value">{selectedTx.chainName}</span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">Status</span>
                  <span className={`badge ${getStatusBadge(selectedTx.status)}`}>{selectedTx.status}</span>
                </div>
                <div className="detail-item full-width">
                  <span className="detail-label">From Address</span>
                  <span className="detail-value mono">{selectedTx.fromAddress}</span>
                </div>
                <div className="detail-item full-width">
                  <span className="detail-label">To Address</span>
                  <span className="detail-value mono">{selectedTx.toAddress}</span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">Amount</span>
                  <span className="detail-value">{selectedTx.amount} {selectedTx.tokenSymbol}</span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">Fee</span>
                  <span className="detail-value">{selectedTx.fee}</span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">Block Number</span>
                  <span className="detail-value">{selectedTx.blockNumber.toLocaleString()}</span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">Confirmations</span>
                  <span className="detail-value">{selectedTx.confirmations}</span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">Timestamp</span>
                  <span className="detail-value">{selectedTx.timestamp}</span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">User ID</span>
                  <span className="detail-value">{selectedTx.userId}</span>
                </div>
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setSelectedTx(null)}>Close</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default TransactionsPage;
