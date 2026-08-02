/**
 * User Management - Complete User Management
 * Connected to backend APIs
 */

import React, { useState, useEffect, useCallback } from 'react';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

interface User {
  id: string;
  email: string;
  username: string;
  status: 'active' | 'suspended' | 'pending';
  kycStatus: 'none' | 'pending' | 'verified' | 'rejected';
  createdAt: string;
  balance: number;
}

function UserManagement() {
  const [activeTab, setActiveTab] = useState('users');
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch users from backend
  const fetchUsers = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const token = localStorage.getItem('superadmin_token');
      const response = await fetch(`${API_BASE_URL}/super-admin/users`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to fetch users');
      }
      
      const data = await response.json();
      setUsers(data.users || []);
    } catch (err) {
      console.error('Error fetching users:', err);
      setError(err instanceof Error ? err.message : 'Failed to load users');
      setUsers([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  // Update user status
  const handleStatusChange = async (userId: string, newStatus: 'active' | 'suspended') => {
    try {
      const token = localStorage.getItem('superadmin_token');
      const response = await fetch(`${API_BASE_URL}/super-admin/users/${userId}/status`, {
        method: 'PATCH',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ status: newStatus }),
      });
      
      if (!response.ok) {
        throw new Error('Failed to update user status');
      }
      
      await fetchUsers();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Status update failed');
    }
  };

  // Format balance
  const formatBalance = (balance: number): string => {
    return '$' + balance.toLocaleString();
  };

  // Format date
  const formatDate = (dateStr: string): string => {
    return new Date(dateStr).toLocaleDateString();
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-amber-500"></div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">User Management</h1>
      </div>

      {error && (
        <div className="bg-red-500/20 border border-red-500 text-red-500 px-4 py-3 rounded-lg mb-4">
          {error}
        </div>
      )}

      <div className="flex gap-2 mb-6">
        {['Users', 'KYC', 'Activity'].map(tab => (
          <button key={tab} onClick={() => setActiveTab(tab.toLowerCase())} className={`px-4 py-2 rounded-lg ${activeTab === tab.toLowerCase() ? 'bg-amber-500 text-black' : 'bg-slate-800'}`}>
            {tab}
          </button>
        ))}
      </div>

      {activeTab === 'users' && (
        <div className="bg-slate-800 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-slate-700">
              <tr>
                <th className="p-3 text-left">ID</th>
                <th className="p-3 text-left">Email</th>
                <th className="p-3 text-left">Username</th>
                <th className="p-3 text-left">Status</th>
                <th className="p-3 text-left">KYC</th>
                <th className="p-3 text-left">Created</th>
                <th className="p-3 text-left">Balance</th>
                <th className="p-3 text-left">Actions</th>
              </tr>
            </thead>
            <tbody>
              {users.length === 0 ? (
                <tr>
                  <td colSpan={8} className="p-8 text-center opacity-60">No users found</td>
                </tr>
              ) : (
                users.map(user => (
                  <tr key={user.id} className="border-t border-slate-700">
                    <td className="p-3">#{user.id.substring(0, 8)}</td>
                    <td className="p-3">{user.email}</td>
                    <td className="p-3 font-semibold">{user.username}</td>
                    <td className="p-3">
                      <span className={`px-2 py-1 rounded text-xs ${user.status === 'active' ? 'bg-green-500/20 text-green-500' : 'bg-red-500/20 text-red-500'}`}>
                        {user.status}
                      </span>
                    </td>
                    <td className="p-3">
                      <span className={`px-2 py-1 rounded text-xs ${
                        user.kycStatus === 'verified' ? 'bg-green-500/20 text-green-500' : 
                        user.kycStatus === 'pending' ? 'bg-yellow-500/20 text-yellow-500' : 
                        user.kycStatus === 'rejected' ? 'bg-red-500/20 text-red-500' : 
                        'bg-gray-500/20 text-gray-500'
                      }`}>
                        {user.kycStatus || 'None'}
                      </span>
                    </td>
                    <td className="p-3">{formatDate(user.createdAt)}</td>
                    <td className="p-3">{formatBalance(user.balance)}</td>
                    <td className="p-3">
                      <div className="flex gap-2">
                        <button 
                          onClick={() => handleStatusChange(user.id, user.status === 'active' ? 'suspended' : 'active')}
                          className="btn btn-danger text-xs"
                        >
                          {user.status === 'active' ? 'Suspend' : 'Activate'}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'kyc' && (
        <div className="space-y-4">
          {users.filter(u => u.kyc !== 'Verified').map(user => (
            <div key={user.id} className="bg-slate-800 p-4 rounded-lg">
              <div className="flex justify-between items-start">
                <div>
                  <h3 className="font-semibold">{user.username}</h3>
                  <p className="text-sm opacity-60">{user.email}</p>
                  <p className="text-sm">Status: {user.kyc}</p>
                </div>
                <div className="flex gap-2">
                  <button className="btn btn-primary text-sm">Approve</button>
                  <button className="btn btn-danger text-sm">Reject</button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {activeTab === 'activity' && (
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-center opacity-60">Recent user activity will appear here</p>
        </div>
      )}
    </div>
  );
}

export default UserManagement;
