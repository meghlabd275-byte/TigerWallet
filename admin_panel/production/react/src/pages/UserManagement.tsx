/**
 * User Management - Complete User Management
 */

import React, { useState } from 'react';

function UserManagement() {
  const [activeTab, setActiveTab] = useState('users');
  
  const users = [
    { id: 1, email: 'user1@example.com', username: 'crypto_fan', status: 'Active', kyc: 'Verified', created: '2024-01-15', balance: '$12,345' },
    { id: 2, email: 'user2@example.com', username: 'trader_pro', status: 'Active', kyc: 'Pending', created: '2024-02-20', balance: '$45,678' },
    { id: 3, email: 'user3@example.com', username: 'hodler_btc', status: 'Suspended', kyc: 'Rejected', created: '2024-03-10', balance: '$0' },
    { id: 4, email: 'user4@example.com', username: 'defi_degen', status: 'Active', kyc: 'Verified', created: '2024-04-05', balance: '$23,456' },
  ];

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">User Management</h1>
        <button className="btn btn-primary">+ Add User</button>
      </div>

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
              {users.map(user => (
                <tr key={user.id} className="border-t border-slate-700">
                  <td className="p-3">#{user.id}</td>
                  <td className="p-3">{user.email}</td>
                  <td className="p-3 font-semibold">{user.username}</td>
                  <td className="p-3">
                    <span className={`px-2 py-1 rounded text-xs ${user.status === 'Active' ? 'bg-green-500/20 text-green-500' : 'bg-red-500/20 text-red-500'}`}>
                      {user.status}
                    </span>
                  </td>
                  <td className="p-3">
                    <span className={`px-2 py-1 rounded text-xs ${user.kyc === 'Verified' ? 'bg-green-500/20 text-green-500' : user.kyc === 'Pending' ? 'bg-yellow-500/20 text-yellow-500' : 'bg-red-500/20 text-red-500'}`}>
                      {user.kyc}
                    </span>
                  </td>
                  <td className="p-3">{user.created}</td>
                  <td className="p-3">{user.balance}</td>
                  <td className="p-3">
                    <div className="flex gap-2">
                      <button className="btn btn-secondary text-xs">Edit</button>
                      <button className="btn btn-danger text-xs">{user.status === 'Active' ? 'Suspend' : 'Activate'}</button>
                    </div>
                  </td>
                </tr>
              ))}
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
