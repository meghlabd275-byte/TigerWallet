/**
 * Admin Management
 */

import React, { useState } from 'react';

function AdminManagement() {
  const [activeTab, setActiveTab] = useState('admins');

  const admins = [
    { id: 1, name: 'Super Admin', email: 'admin@tigerwallet.com', role: 'Super Admin', permissions: ['all'], lastLogin: '2 hours ago', status: 'Active' },
    { id: 2, name: 'John Doe', email: 'john@tigerwallet.com', role: 'Manager', permissions: ['users', 'kyc', 'wallets'], lastLogin: '1 day ago', status: 'Active' },
    { id: 3, name: 'Jane Smith', email: 'jane@tigerwallet.com', role: 'Support', permissions: ['users', 'kyc'], lastLogin: '3 hours ago', status: 'Active' },
    { id: 4, name: 'Bob Wilson', email: 'bob@tigerwallet.com', role: 'Trader', permissions: ['trading', 'wallets'], lastLogin: '5 days ago', status: 'Inactive' },
  ];

  const permissions = [
    { id: 'users', name: 'User Management' },
    { id: 'kyc', name: 'KYC Management' },
    { id: 'wallets', name: 'Wallet Management' },
    { id: 'trading', name: 'Trading Management' },
    { id: 'fees', name: 'Fee Management' },
    { id: 'liquidity', name: 'Liquidity Management' },
    { id: 'blockchains', name: 'Blockchain Management' },
    { id: 'whitelabel', name: 'White Label Management' },
    { id: 'analytics', name: 'Analytics' },
    { id: 'settings', name: 'Settings' },
  ];

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Admin Management</h1>
        <button className="btn btn-primary">+ Add Admin</button>
      </div>

      <div className="flex gap-2 mb-6">
        {['Admins', 'Roles', 'Permissions', 'Activity'].map(tab => (
          <button key={tab} onClick={() => setActiveTab(tab.toLowerCase())} className={`px-4 py-2 rounded-lg ${activeTab === tab.toLowerCase() ? 'bg-amber-500 text-black' : 'bg-slate-800'}`}>
            {tab}
          </button>
        ))}
      </div>

      {activeTab === 'admins' && (
        <div className="bg-slate-800 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-slate-700">
              <tr>
                <th className="p-3 text-left">ID</th>
                <th className="p-3 text-left">Name</th>
                <th className="p-3 text-left">Email</th>
                <th className="p-3 text-left">Role</th>
                <th className="p-3 text-left">Permissions</th>
                <th className="p-3 text-left">Last Login</th>
                <th className="p-3 text-left">Status</th>
                <th className="p-3 text-left">Actions</th>
              </tr>
            </thead>
            <tbody>
              {admins.map(admin => (
                <tr key={admin.id} className="border-t border-slate-700">
                  <td className="p-3">#{admin.id}</td>
                  <td className="p-3 font-semibold">{admin.name}</td>
                  <td className="p-3">{admin.email}</td>
                  <td className="p-3">
                    <span className={`px-2 py-1 rounded text-xs ${admin.role === 'Super Admin' ? 'bg-amber-500/20 text-amber-500' : 'bg-blue-500/20 text-blue-500'}`}>
                      {admin.role}
                    </span>
                  </td>
                  <td className="p-3">
                    <div className="flex gap-1 flex-wrap">
                      {admin.permissions.map((p, i) => (
                        <span key={i} className="px-2 py-0.5 bg-slate-600 rounded text-xs">{p}</span>
                      ))}
                    </div>
                  </td>
                  <td className="p-3">{admin.lastLogin}</td>
                  <td className="p-3">
                    <span className={`px-2 py-1 rounded text-xs ${admin.status === 'Active' ? 'bg-green-500/20 text-green-500' : 'bg-red-500/20 text-red-500'}`}>
                      {admin.status}
                    </span>
                  </td>
                  <td className="p-3">
                    <div className="flex gap-2">
                      <button className="btn btn-secondary text-xs">Edit</button>
                      <button className="btn btn-danger text-xs">{admin.status === 'Active' ? 'Deactivate' : 'Activate'}</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'roles' && (
        <div className="space-y-4">
          {['Super Admin', 'Manager', 'Support', 'Trader', 'Viewer'].map((role, i) => (
            <div key={role} className="bg-slate-800 p-4 rounded-lg flex justify-between items-center">
              <div>
                <h3 className="font-semibold">{role}</h3>
                <p className="text-sm opacity-60">{i === 0 ? 'Full access to all features' : `Custom role with specific permissions`}</p>
              </div>
              <button className="btn btn-secondary">Edit</button>
            </div>
          ))}
        </div>
      )}

      {activeTab === 'permissions' && (
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Available Permissions</h3>
          <div className="grid grid-cols-2 gap-3">
            {permissions.map(p => (
              <div key={p.id} className="flex items-center gap-2 p-3 bg-slate-700 rounded">
                <input type="checkbox" id={p.id} />
                <label htmlFor={p.id}>{p.name}</label>
              </div>
            ))}
          </div>
        </div>
      )}

      {activeTab === 'activity' && (
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Recent Admin Activity</h3>
          <div className="space-y-3">
            {[
              { admin: 'Super Admin', action: 'Updated fee settings', time: '2 min ago' },
              { admin: 'John Doe', action: 'Approved KYC for user123', time: '15 min ago' },
              { admin: 'Jane Smith', action: 'Reset password for user456', time: '1 hour ago' },
              { admin: 'Super Admin', action: 'Added new blockchain', time: '2 hours ago' },
            ].map((a, i) => (
              <div key={i} className="flex justify-between p-3 bg-slate-700 rounded">
                <div>
                  <span className="font-semibold">{a.admin}</span>
                  <span className="opacity-60"> - {a.action}</span>
                </div>
                <span className="text-sm opacity-60">{a.time}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

export default AdminManagement;
