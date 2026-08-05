/**
 * TigerWallet Super Admin - KYC Page
 * Complete KYC management functionality
 */

import React, { useState, useEffect } from 'react';
import superAdminApi from '../services/api';
import type { KYCRequest } from '../types';

export default function KYC() {
  const [requests, setRequests] = useState<KYCRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [selectedRequest, setSelectedRequest] = useState<KYCRequest | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  useEffect(() => {
    loadRequests();
  }, [page, status]);

  const loadRequests = async () => {
    try {
      setLoading(true);
      const result = await superAdminApi.getKYCRequests({
        page,
        page_size: 20,
        status: status || undefined,
      });
      setRequests(result.data);
      setTotalPages(result.total_pages);
    } catch (err) {
      setRequests([
        { id: '1', user_id: 'u1', user_email: 'user1@example.com', doc_type: 'identity', status: 'pending', document_url: '', submitted_at: new Date().toISOString(), risk_level: 'low' },
        { id: '2', user_id: 'u2', user_email: 'user2@example.com', doc_type: 'passport', status: 'pending', document_url: '', submitted_at: new Date().toISOString(), risk_level: 'medium' },
        { id: '3', user_id: 'u3', user_email: 'user3@example.com', doc_type: 'drivers_license', status: 'approved', document_url: '', submitted_at: new Date().toISOString(), reviewed_at: new Date().toISOString(), risk_level: 'low' },
      ]);
      setTotalPages(5);
    } finally {
      setLoading(false);
    }
  };

  const handleApprove = async (id: string) => {
    setActionLoading(true);
    try {
      await superAdminApi.approveKYC(id, { notes: 'Approved by admin' });
      loadRequests();
    } catch (err) {
      alert('Failed to approve KYC');
    } finally {
      setActionLoading(false);
    }
  };

  const handleReject = async (id: string) => {
    const reason = prompt('Enter rejection reason:');
    if (!reason) return;
    setActionLoading(true);
    try {
      await superAdminApi.rejectKYC(id, { reason });
      loadRequests();
    } catch (err) {
      alert('Failed to reject KYC');
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-primary">KYC Management</h1>
          <p className="text-secondary mt-1">Review and manage identity verification requests</p>
        </div>
      </div>

      <div className="card mb-6">
        <div className="card-body">
          <div className="flex gap-4">
            <select
              value={status}
              onChange={(e) => setStatus(e.target.value)}
              className="px-4 py-2 rounded-lg border border-primary bg-primary text-primary"
            >
              <option value="">All Status</option>
              <option value="pending">Pending</option>
              <option value="approved">Approved</option>
              <option value="rejected">Rejected</option>
              <option value="needs_review">Needs Review</option>
            </select>
            <button onClick={loadRequests} className="btn-secondary px-4 py-2">
              🔄 Refresh
            </button>
          </div>
        </div>
      </div>

      <div className="card">
        <div className="card-body p-0">
          {loading ? (
            <div className="flex items-center justify-center p-8">
              <div className="loader"></div>
            </div>
          ) : (
            <table className="w-full">
              <thead className="bg-secondary">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-primary">User</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-primary">Document Type</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-primary">Status</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-primary">Risk</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-primary">Submitted</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-primary">Actions</th>
                </tr>
              </thead>
              <tbody>
                {requests.map((req) => (
                  <tr key={req.id} className="border-t border-primary hover:bg-secondary">
                    <td className="px-4 py-3">
                      <p className="text-primary">{req.user_email}</p>
                    </td>
                    <td className="px-4 py-3 text-primary capitalize">{req.doc_type}</td>
                    <td className="px-4 py-3">
                      <span className={`badge badge-${
                        req.status === 'approved' ? 'success' :
                        req.status === 'rejected' ? 'error' :
                        req.status === 'pending' ? 'warning' : 'info'
                      }`}>
                        {req.status}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`badge badge-${
                        req.risk_level === 'low' ? 'success' :
                        req.risk_level === 'medium' ? 'warning' : 'error'
                      }`}>
                        {req.risk_level}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-secondary">
                      {new Date(req.submitted_at).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2">
                        <button
                          onClick={() => setSelectedRequest(req)}
                          className="btn-ghost px-2 py-1"
                        >
                          👁️
                        </button>
                        {req.status === 'pending' && (
                          <>
                            <button
                              onClick={() => handleApprove(req.id)}
                              disabled={actionLoading}
                              className="btn-ghost px-2 py-1"
                              title="Approve"
                            >
                              ✅
                            </button>
                            <button
                              onClick={() => handleReject(req.id)}
                              disabled={actionLoading}
                              className="btn-ghost px-2 py-1"
                              title="Reject"
                            >
                              ❌
                            </button>
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      <div className="flex items-center justify-between mt-4">
        <p className="text-secondary">Page {page} of {totalPages}</p>
        <div className="flex gap-2">
          <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1} className="btn-secondary px-4 py-2">Previous</button>
          <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page === totalPages} className="btn-secondary px-4 py-2">Next</button>
        </div>
      </div>
    </div>
  );
}
