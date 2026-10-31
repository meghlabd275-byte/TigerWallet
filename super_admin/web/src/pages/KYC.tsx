/**
 * TigerWallet Super Admin - KYC Page
 * Complete KYC management functionality
 */

import React, { useState, useEffect } from 'react';
import superAdminApi from '../services/api';
import type { KYCRequest } from '../types';
import { DocumentLink } from '../utils/safeUrl';

export default function KYC() {
  const [requests, setRequests] = useState<KYCRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [selectedRequest, setSelectedRequest] = useState<KYCRequest | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState('');

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
    } catch (err: any) {
      // Never substitute placeholder rows here: an operator reading this table
      // would take fabricated identities for real pending KYC work and could
      // act on them. Surface the failure instead.
      setRequests([]);
      setTotalPages(1);
      setError(err?.message || 'Failed to load KYC requests');
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

      {error && (
        <div className="card mb-6 border-l-4 border-red-500">
          <div className="card-body text-red-600">{error}</div>
        </div>
      )}

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

      {selectedRequest && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setSelectedRequest(null)}>
          <div className="card w-full max-w-lg" onClick={(e) => e.stopPropagation()}>
            <div className="card-body">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold text-primary">KYC Request</h2>
                <button onClick={() => setSelectedRequest(null)} className="btn-ghost px-2 py-1" title="Close">✕</button>
              </div>
              <dl className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <dt className="text-secondary">User</dt>
                  <dd className="text-primary">{selectedRequest.user_email || selectedRequest.user_id}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-secondary">Document type</dt>
                  <dd className="text-primary capitalize">{selectedRequest.doc_type}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-secondary">Status</dt>
                  <dd className="text-primary">{selectedRequest.status}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-secondary">Submitted</dt>
                  <dd className="text-primary">
                    {selectedRequest.submitted_at ? new Date(selectedRequest.submitted_at).toLocaleString() : '—'}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-secondary">Document</dt>
                  <dd className="text-primary">
                    <DocumentLink url={selectedRequest.document_url} />
                    {!selectedRequest.document_url && '—'}
                  </dd>
                </div>
                {selectedRequest.reject_reason && (
                  <div className="flex justify-between">
                    <dt className="text-secondary">Rejection reason</dt>
                    <dd className="text-primary">{selectedRequest.reject_reason}</dd>
                  </div>
                )}
              </dl>
              {selectedRequest.status === 'pending' && (
                <div className="flex gap-2 mt-6">
                  <button
                    onClick={() => { const id = selectedRequest.id; setSelectedRequest(null); handleApprove(id); }}
                    disabled={actionLoading}
                    className="btn-primary px-4 py-2"
                  >Approve</button>
                  <button
                    onClick={() => { const id = selectedRequest.id; setSelectedRequest(null); handleReject(id); }}
                    disabled={actionLoading}
                    className="btn-secondary px-4 py-2"
                  >Reject</button>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

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
