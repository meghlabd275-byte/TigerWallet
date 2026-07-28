/**
 * Withdrawal Management
 */

import React, { useState } from 'react';

function WithdrawalManagement() {
  const [activeTab, setActiveTab] = useState('pending');

  const withdrawals = [
    { id: '0x1234', user: 'user1@example.com', token: 'ETH', amount: '1.5', status: 'Pending', fee: '0.005', time: '2 min ago', address: '0x5678...' },
    { id: '0x2345', user: 'user2@example.com', token: 'BTC', amount: '0.25', status: 'Approved', fee: '0.0005', time: '15 min ago', address: 'bc1q...' },
    { id: '0x3456', user: 'user3@example.com', token: 'USDT', amount: '5000', status: 'Processing', fee: '1', time: '30 min ago', address: '0x7890...' },
    { id: '0x4567', user: 'user4@example.com', token: 'ETH', amount: '5.0', status: 'Completed', fee: '0.005', time: '1 hour ago', address: '0xabcd...' },
    { id: '0x5678', user: 'user5@example.com', token: 'BNB', amount: '10', status: 'Rejected', fee: '0.001', time: '2 hours ago', address: '0xefgh...' },
  ];

  const stats = [
    { label: 'Pending', value: '12', color: 'yellow' },
    { label: 'Processing', value: '5', color: 'blue' },
    { label: 'Completed', value: '156', color: 'green' },
    { label: 'Rejected', value: '3', color: 'red' },
  ];

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Withdrawal Management</h1>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        {stats.map((s, i) => (
          <div key={i} className="bg-slate-800 p-4 rounded-lg">
            <p className="text-sm opacity-60">{s.label}</p>
            <p className={`text-2xl font-bold text-${s.color}-500`}>{s.value}</p>
          </div>
        ))}
      </div>

      <div className="flex gap-2 mb-6">
        {['Pending', 'Processing', 'Completed', 'Rejected'].map(tab => (
          <button key={tab} onClick={() => setActiveTab(tab.toLowerCase())} className={`px-4 py-2 rounded-lg ${activeTab === tab.toLowerCase() ? 'bg-amber-500 text-black' : 'bg-slate-800'}`}>
            {tab}
          </button>
        ))}
      </div>

      <div className="bg-slate-800 rounded-lg overflow-hidden">
        <table className="w-full">
          <thead className="bg-slate-700">
            <tr>
              <th className="p-3 text-left">ID</th>
              <th className="p-3 text-left">User</th>
              <th className="p-3 text-left">Token</th>
              <th className="p-3 text-left">Amount</th>
              <th className="p-3 text-left">Fee</th>
              <th className="p-3 text-left">Address</th>
              <th className="p-3 text-left">Time</th>
              <th className="p-3 text-left">Actions</th>
            </tr>
          </thead>
          <tbody>
            {withdrawals.filter(w => w.status.toLowerCase() === activeTab).map(w => (
              <tr key={w.id} className="border-t border-slate-700">
                <td className="p-3 font-mono text-sm">{w.id}</td>
                <td className="p-3">{w.user}</td>
                <td className="p-3 font-bold text-amber-500">{w.token}</td>
                <td className="p-3">{w.amount}</td>
                <td className="p-3">{w.fee}</td>
                <td className="p-3 font-mono text-sm">{w.address}</td>
                <td className="p-3">{w.time}</td>
                <td className="p-3">
                  {activeTab === 'pending' && (
                    <div className="flex gap-2">
                      <button className="btn btn-primary text-xs">Approve</button>
                      <button className="btn btn-danger text-xs">Reject</button>
                    </div>
                  )}
                  {activeTab === 'processing' && (
                    <button className="btn btn-secondary text-xs">Process</button>
                  )}
                  {activeTab === 'completed' && (
                    <button className="btn btn-secondary text-xs">View Tx</button>
                  )}
                  {activeTab === 'rejected' && (
                    <button className="btn btn-secondary text-xs">Details</button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export default WithdrawalManagement;
