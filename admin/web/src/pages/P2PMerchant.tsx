/**
 * TigerWallet Admin - P2P Merchant Management Page
 * Complete implementation with backend connectivity
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { p2pMerchantAPI } from '../services/api';

interface Merchant {
  id: string;
  businessName: string;
  email: string;
  phone: string;
  country: string;
  status: 'pending' | 'approved' | 'rejected' | 'suspended';
  verified: boolean;
  totalVolume: number;
  transactionCount: number;
  rating: number;
  createdAt: string;
}

interface MerchantTransaction {
  id: string;
  merchantId: string;
  buyerId: string;
  sellerId: string;
  amount: number;
  currency: string;
  status: 'pending' | 'completed' | 'cancelled' | 'disputed';
  paymentMethod: string;
  createdAt: string;
}

export const P2PMerchantPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [merchants, setMerchants] = useState<Merchant[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('all');
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedMerchant, setSelectedMerchant] = useState<Merchant | null>(null);
  const [transactions, setTransactions] = useState<MerchantTransaction[]>([]);

  useEffect(() => {
    loadMerchants();
  }, [filter]);

  const loadMerchants = async () => {
    setLoading(true);
    try {
      const response = await p2pMerchantAPI.getMerchants(filter !== 'all' ? filter : undefined);
      setMerchants(response.data);
    } catch (error) {
      console.error('Failed to load merchants:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleApprove = async (merchantId: string) => {
    try {
      await p2pMerchantAPI.approveMerchant(merchantId);
      loadMerchants();
    } catch (error) {
      console.error('Failed to approve merchant:', error);
    }
  };

  const handleReject = async (merchantId: string) => {
    const reason = prompt('Enter rejection reason:');
    if (!reason) return;
    try {
      await p2pMerchantAPI.rejectMerchant(merchantId, reason);
      loadMerchants();
    } catch (error) {
      console.error('Failed to reject merchant:', error);
    }
  };

  const viewMerchantDetails = async (merchant: Merchant) => {
    setSelectedMerchant(merchant);
    try {
      const response = await p2pMerchantAPI.getTransactions(merchant.id);
      setTransactions(response.data);
    } catch (error) {
      console.error('Failed to load transactions:', error);
    }
  };

  const filteredMerchants = merchants.filter(m =>
    m.businessName.toLowerCase().includes(searchTerm.toLowerCase()) ||
    m.email.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className={`page-container ${isDark ? 'dark' : 'light'}`}>
      <div className="page-header">
        <h1>P2P Merchant Management</h1>
        <button className="theme-btn" onClick={toggleTheme}>
          {isDark ? '☀️ Light' : '🌙 Dark'}
        </button>
      </div>

      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-value">{merchants.length}</div>
          <div className="stat-label">Total Merchants</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{merchants.filter(m => m.status === 'approved').length}</div>
          <div className="stat-label">Approved</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{merchants.filter(m => m.status === 'pending').length}</div>
          <div className="stat-label">Pending</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">${merchants.reduce((sum, m) => sum + m.totalVolume, 0).toLocaleString()}</div>
          <div className="stat-label">Total Volume</div>
        </div>
      </div>

      <div className="filters">
        <select value={filter} onChange={(e) => setFilter(e.target.value)}>
          <option value="all">All Merchants</option>
          <option value="pending">Pending</option>
          <option value="approved">Approved</option>
          <option value="rejected">Rejected</option>
          <option value="suspended">Suspended</option>
        </select>
        <input
          type="text"
          placeholder="Search merchants..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
        />
      </div>

      {loading ? (
        <div className="loading">Loading merchants...</div>
      ) : (
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th>Business Name</th>
                <th>Email</th>
                <th>Country</th>
                <th>Volume</th>
                <th>Transactions</th>
                <th>Rating</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredMerchants.map(merchant => (
                <tr key={merchant.id}>
                  <td>
                    <button
                      className="link-btn"
                      onClick={() => viewMerchantDetails(merchant)}
                    >
                      {merchant.businessName}
                    </button>
                  </td>
                  <td>{merchant.email}</td>
                  <td>{merchant.country}</td>
                  <td>${merchant.totalVolume.toLocaleString()}</td>
                  <td>{merchant.transactionCount}</td>
                  <td>{merchant.rating.toFixed(1)} ★</td>
                  <td>
                    <span className={`status-badge ${merchant.status}`}>
                      {merchant.status}
                    </span>
                  </td>
                  <td>
                    {merchant.status === 'pending' && (
                      <>
                        <button
                          className="btn-success"
                          onClick={() => handleApprove(merchant.id)}
                        >
                          Approve
                        </button>
                        <button
                          className="btn-danger"
                          onClick={() => handleReject(merchant.id)}
                        >
                          Reject
                        </button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {selectedMerchant && (
        <div className="modal-overlay" onClick={() => setSelectedMerchant(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>{selectedMerchant.businessName} - Transactions</h2>
            <button className="close-btn" onClick={() => setSelectedMerchant(null)}>×</button>
            <div className="table-container">
              <table>
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>Amount</th>
                    <th>Status</th>
                    <th>Payment Method</th>
                    <th>Date</th>
                  </tr>
                </thead>
                <tbody>
                  {transactions.map(tx => (
                    <tr key={tx.id}>
                      <td>{tx.id.slice(0, 8)}</td>
                      <td>{tx.amount} {tx.currency}</td>
                      <td>
                        <span className={`status-badge ${tx.status}`}>
                          {tx.status}
                        </span>
                      </td>
                      <td>{tx.paymentMethod}</td>
                      <td>{new Date(tx.createdAt).toLocaleDateString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default P2PMerchantPage;
