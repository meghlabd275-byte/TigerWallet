/**
 * KYC Management - User verification
 */

import React, { useState } from 'react';

function KYCMangement() {
  const [activeTab, setActiveTab] = useState('pending');

  const kycRequests = [
    { id: 1, user: 'user1@example.com', type: 'Individual', status: 'Pending', submitted: '2024-01-15', documents: 3 },
    { id: 2, user: 'user2@example.com', type: 'Business', status: 'Review', submitted: '2024-01-14', documents: 5 },
    { id: 3, user: 'user3@example.com', type: 'Individual', status: 'Approved', submitted: '2024-01-10', documents: 3 },
    { id: 4, user: 'user4@example.com', type: 'Business', status: 'Rejected', submitted: '2024-01-08', documents: 4 },
  ];

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">KYC Management</h1>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">Pending</p>
          <p className="text-2xl font-bold text-yellow-500">12</p>
        </div>
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">Under Review</p>
          <p className="text-2xl font-bold text-blue-500">5</p>
        </div>
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">Approved</p>
          <p className="text-2xl font-bold text-green-500">1,234</p>
        </div>
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">Rejected</p>
          <p className="text-2xl font-bold text-red-500">45</p>
        </div>
      </div>

      <div className="flex gap-2 mb-6">
        {['Pending', 'Review', 'Approved', 'Rejected'].map(tab => (
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
              <th className="p-3 text-left">Type</th>
              <th className="p-3 text-left">Submitted</th>
              <th className="p-3 text-left">Documents</th>
              <th className="p-3 text-left">Actions</th>
            </tr>
          </thead>
          <tbody>
            {kycRequests.filter(k => k.status.toLowerCase() === activeTab).map(req => (
              <tr key={req.id} className="border-t border-slate-700">
                <td className="p-3">#{req.id}</td>
                <td className="p-3">{req.user}</td>
                <td className="p-3">{req.type}</td>
                <td className="p-3">{req.submitted}</td>
                <td className="p-3">{req.documents}</td>
                <td className="p-3">
                  <div className="flex gap-2">
                    <button className="btn btn-secondary text-xs">View</button>
                    {activeTab === 'pending' && <button className="btn btn-primary text-xs">Review</button>}
                    {activeTab === 'review' && (
                      <>
                        <button className="btn btn-primary text-xs">Approve</button>
                        <button className="btn btn-danger text-xs">Reject</button>
                      </>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export default KYCMangement;
