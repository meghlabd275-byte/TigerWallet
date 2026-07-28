// Users Management Page

import React, { useState, useEffect } from 'react';

interface User {
  id: string;
  email: string;
  wallet: string;
  kyc: 'none' | 'pending' | 'verified';
  status: 'active' | 'suspended';
  createdAt: string;
  lastLogin: string;
}

const UsersPage: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [filterKyc, setFilterKyc] = useState<string>('all');

  useEffect(() => {
    loadUsers();
  }, []);

  const loadUsers = () => {
    setUsers([
      { id: '1', email: 'user1@example.com', wallet: '0x7a23...8f91', kyc: 'verified', status: 'active', createdAt: '2026-01-15', lastLogin: '2 hours ago' },
      { id: '2', email: 'user2@example.com', wallet: '0x3b14...2c78', kyc: 'pending', status: 'active', createdAt: '2026-02-20', lastLogin: '5 hours ago' },
      { id: '3', email: 'user3@example.com', wallet: 'Sol123...456', kyc: 'none', status: 'suspended', createdAt: '2026-03-10', lastLogin: '1 day ago' },
    ]);
  };

  const filteredUsers = users.filter(user => {
    const matchesSearch = user.email.toLowerCase().includes(searchQuery.toLowerCase()) || 
                         user.wallet.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesKyc = filterKyc === 'all' || user.kyc === filterKyc;
    return matchesSearch && matchesKyc;
  });

  return (
    <div className="users-page">
      <div className="page-header">
        <div>
          <h1>User Management</h1>
          <p>Manage all registered users and their wallets</p>
        </div>
      </div>

      <div className="filters-bar">
        <div className="search-box">
          <span>🔍</span>
          <input 
            type="text" 
            placeholder="Search by email or wallet..." 
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
          />
        </div>
        <select value={filterKyc} onChange={e => setFilterKyc(e.target.value)}>
          <option value="all">All KYC</option>
          <option value="verified">Verified</option>
          <option value="pending">Pending</option>
          <option value="none">None</option>
        </select>
      </div>

      <div className="table-container">
        <table>
          <thead>
            <tr>
              <th>User</th>
              <th>Wallet Address</th>
              <th>KYC Status</th>
              <th>Status</th>
              <th>Created</th>
              <th>Last Login</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {filteredUsers.map(user => (
              <tr key={user.id}>
                <td>{user.email}</td>
                <td className="wallet-cell">{user.wallet}</td>
                <td>
                  <span className={`kyc-badge kyc-${user.kyc}`}>
                    {user.kyc}
                  </span>
                </td>
                <td>
                  <span className={`status-badge status-${user.status}`}>
                    {user.status}
                  </span>
                </td>
                <td>{user.createdAt}</td>
                <td>{user.lastLogin}</td>
                <td>
                  <div className="actions">
                    <button className="action-btn">👁️</button>
                    <button className="action-btn">✏️</button>
                    <button className="action-btn">🗑️</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default UsersPage;
