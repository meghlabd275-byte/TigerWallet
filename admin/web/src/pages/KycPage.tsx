// TigerWallet Admin - KYC Verification Page
// Review and manage KYC submissions

import React, { useState, useEffect } from 'react';
import { adminApi } from '../services/api';

interface KycRecord {
  id: string;
  userId: string;
  userEmail: string;
  userName: string;
  status: string;
  submittedAt: string;
  reviewedAt?: string;
  documentType: string;
  documentUrl?: string;
  selfieUrl?: string;
  rejectionReason?: string;
  riskScore?: number;
}

const KycPage: React.FC = () => {
  const [kycRecords, setKycRecords] = useState<KycRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState('');
  const [searchTerm, setSearchTerm] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [selectedRecord, setSelectedRecord] = useState<KycRecord | null>(null);
  const [showApproveModal, setShowApproveModal] = useState(false);
  const [showRejectModal, setShowRejectModal] = useState(false);
  const [rejectReason, setRejectReason] = useState('');
  const [approveNotes, setApproveNotes] = useState('');

  useEffect(() => {
    loadKycRecords();
  }, [page, statusFilter]);

  const loadKycRecords = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await adminApi.getKycRecords({
        page,
        pageSize: 20,
        status: statusFilter || undefined,
        search: searchTerm || undefined,
      });

      setKycRecords(response.data || []);
      setTotalPages(response.totalPages || 1);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load KYC records');
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    loadKycRecords();
  };

  const handleApprove = async () => {
    if (!selectedRecord) return;
    
    try {
      await adminApi.approveKyc(selectedRecord.id, approveNotes);
      setShowApproveModal(false);
      setApproveNotes('');
      loadKycRecords();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to approve KYC');
    }
  };

  const handleReject = async () => {
    if (!selectedRecord || !rejectReason) return;
    
    try {
      await adminApi.rejectKyc(selectedRecord.id, rejectReason);
      setShowRejectModal(false);
      setRejectReason('');
      loadKycRecords();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reject KYC');
    }
  };

  const handleRequestResubmission = async (recordId: string) => {
    const reason = prompt('Reason for requesting resubmission:');
    if (!reason) return;
    
    try {
      await adminApi.requestKycResubmission(recordId, reason);
      loadKycRecords();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to request resubmission');
    }
  };

  const getStatusBadgeClass = (status: string): string => {
    switch (status) {
      case 'approved': return 'badge-success';
      case 'pending': case 'under_review': return 'badge-warning';
      case 'rejected': return 'badge-error';
      default: return 'badge-neutral';
    }
  };

  const getRiskColor = (score?: number): string => {
    if (!score) return 'var(--text-tertiary)';
    if (score >= 70) return 'var(--color-error)';
    if (score >= 40) return 'var(--color-warning)';
    return 'var(--color-success)';
  };

  return (
    <div className="p-6">
      {/* Page Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold" style={{ color: 'var(--text-primary)' }}>
          KYC Verification
        </h1>
        <p style={{ color: 'var(--text-secondary)' }}>
          Review and manage identity verification submissions
        </p>
      </div>

      {error && (
        <div className="alert alert-error mb-4">
          {error}
        </div>
      )}

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="stat-card">
          <div className="stat-label">Pending Review</div>
          <div className="stat-value">
            {kycRecords.filter(r => r.status === 'pending' || r.status === 'under_review').length}
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Approved Today</div>
          <div className="stat-value">
            {kycRecords.filter(r => r.status === 'approved' && new Date(r.reviewedAt || '').toDateString() === new Date().toDateString()).length}
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Rejected Today</div>
          <div className="stat-value">
            {kycRecords.filter(r => r.status === 'rejected' && new Date(r.reviewedAt || '').toDateString() === new Date().toDateString()).length}
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Total Processed</div>
          <div className="stat-value">{kycRecords.length}</div>
        </div>
      </div>

      {/* Search and Filters */}
      <div className="card mb-6">
        <div className="card-body">
          <form onSubmit={handleSearch} className="flex flex-wrap gap-4 items-end">
            <div className="flex-1 min-w-64">
              <label className="form-label">Search</label>
              <input
                type="text"
                className="form-input"
                placeholder="Search by email or name..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </div>
            <div className="w-40">
              <label className="form-label">Status</label>
              <select
                className="form-select"
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
              >
                <option value="">All</option>
                <option value="pending">Pending</option>
                <option value="under_review">Under Review</option>
                <option value="approved">Approved</option>
                <option value="rejected">Rejected</option>
              </select>
            </div>
            <button type="submit" className="btn btn-primary">
              Search
            </button>
          </form>
        </div>
      </div>

      {/* KYC Records Table */}
      <div className="card">
        <div className="card-body p-0">
          {loading ? (
            <div className="flex items-center justify-center p-8">
              <div className="loader"></div>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="table">
                <thead>
                  <tr>
                    <th>User</th>
                    <th>Document Type</th>
                    <th>Risk Score</th>
                    <th>Submitted</th>
                    <th>Status</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {kycRecords.length > 0 ? (
                    kycRecords.map((record) => (
                      <tr key={record.id}>
                        <td>
                          <div>
                            <p className="font-medium" style={{ color: 'var(--text-primary)' }}>
                              {record.userName || record.userEmail}
                            </p>
                            <p className="text-sm" style={{ color: 'var(--text-tertiary)' }}>
                              {record.userEmail}
                            </p>
                          </div>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-secondary)' }}>
                            {record.documentType.replace('_', ' ').toUpperCase()}
                          </span>
                        </td>
                        <td>
                          <span style={{ color: getRiskColor(record.riskScore), fontWeight: 500 }}>
                            {record.riskScore ?? 'N/A'}
                          </span>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-secondary)' }}>
                            {new Date(record.submittedAt).toLocaleString()}
                          </span>
                        </td>
                        <td>
                          <span className={`badge ${getStatusBadgeClass(record.status)}`}>
                            {record.status.replace('_', ' ')}
                          </span>
                        </td>
                        <td>
                          <div className="flex gap-2">
                            {record.status === 'pending' || record.status === 'under_review' ? (
                              <>
                                <button
                                  className="btn btn-sm btn-success"
                                  onClick={() => {
                                    setSelectedRecord(record);
                                    setShowApproveModal(true);
                                  }}
                                >
                                  Approve
                                </button>
                                <button
                                  className="btn btn-sm btn-danger"
                                  onClick={() => {
                                    setSelectedRecord(record);
                                    setShowRejectModal(true);
                                  }}
                                >
                                  Reject
                                </button>
                              </>
                            ) : (
                              <button
                                className="btn btn-sm btn-outline"
                                onClick={() => handleRequestResubmission(record.id)}
                              >
                                Request Resubmission
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={6} className="text-center py-8" style={{ color: 'var(--text-tertiary)' }}>
                        No KYC records found
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="card-footer flex justify-between items-center">
            <span style={{ color: 'var(--text-secondary)' }}>
              Page {page} of {totalPages}
            </span>
            <div className="pagination">
              <button
                className="pagination-btn"
                onClick={() => setPage(p => Math.max(1, p - 1))}
                disabled={page === 1}
              >
                Previous
              </button>
              <button
                className="pagination-btn"
                onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                disabled={page === totalPages}
              >
                Next
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Approve Modal */}
      {showApproveModal && selectedRecord && (
        <div className="modal-overlay" onClick={() => setShowApproveModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Approve KYC</h3>
              <button className="btn btn-sm btn-outline" onClick={() => setShowApproveModal(false)}>
                ✕
              </button>
            </div>
            <div className="modal-body">
              <p className="mb-4" style={{ color: 'var(--text-secondary)' }}>
                Approve KYC for <strong>{selectedRecord.userEmail}</strong>
              </p>
              <div className="form-group">
                <label className="form-label">Notes (optional)</label>
                <textarea
                  className="form-textarea"
                  rows={4}
                  value={approveNotes}
                  onChange={(e) => setApproveNotes(e.target.value)}
                  placeholder="Add approval notes..."
                />
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowApproveModal(false)}>
                Cancel
              </button>
              <button className="btn btn-success" onClick={handleApprove}>
                Approve
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Reject Modal */}
      {showRejectModal && selectedRecord && (
        <div className="modal-overlay" onClick={() => setShowRejectModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Reject KYC</h3>
              <button className="btn btn-sm btn-outline" onClick={() => setShowRejectModal(false)}>
                ✕
              </button>
            </div>
            <div className="modal-body">
              <p className="mb-4" style={{ color: 'var(--text-secondary)' }}>
                Reject KYC for <strong>{selectedRecord.userEmail}</strong>
              </p>
              <div className="form-group">
                <label className="form-label">Rejection Reason (required)</label>
                <textarea
                  className="form-textarea"
                  rows={4}
                  value={rejectReason}
                  onChange={(e) => setRejectReason(e.target.value)}
                  placeholder="Enter rejection reason..."
                />
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowRejectModal(false)}>
                Cancel
              </button>
              <button
                className="btn btn-danger"
                onClick={handleReject}
                disabled={!rejectReason}
              >
                Reject
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default KycPage;
