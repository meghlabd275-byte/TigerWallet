// TigerWallet Admin - White Labels Page
// Manage white label partners

import React, { useState, useEffect } from 'react';
import { adminApi } from '../services/api';

interface WhiteLabel {
  id: string;
  name: string;
  domain: string;
  logoUrl?: string;
  primaryColor: string;
  status: string;
  createdAt: string;
  approvedAt?: string;
  ownerEmail: string;
  ownerName: string;
}

const WhiteLabelsPage: React.FC = () => {
  const [whiteLabels, setWhiteLabels] = useState<WhiteLabel[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newWhiteLabel, setNewWhiteLabel] = useState({
    name: '',
    domain: '',
    ownerEmail: '',
    ownerName: '',
    primaryColor: '#dc2626',
  });
  const [processing, setProcessing] = useState<string | null>(null);

  useEffect(() => {
    loadWhiteLabels();
  }, [page, statusFilter]);

  const loadWhiteLabels = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await adminApi.getWhiteLabels({
        page,
        pageSize: 20,
        status: statusFilter || undefined,
      });

      setWhiteLabels(response.data || []);
      setTotalPages(response.totalPages || 1);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load white labels');
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async () => {
    try {
      await adminApi.createWhiteLabel(newWhiteLabel);
      setShowCreateModal(false);
      setNewWhiteLabel({ name: '', domain: '', ownerEmail: '', ownerName: '', primaryColor: '#dc2626' });
      loadWhiteLabels();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create white label');
    }
  };

  const handleApprove = async (id: string) => {
    try {
      setProcessing(id);
      await adminApi.approveWhiteLabel(id);
      loadWhiteLabels();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to approve white label');
    } finally {
      setProcessing(null);
    }
  };

  const handleReject = async (id: string) => {
    const reason = prompt('Rejection reason:');
    if (!reason) return;
    
    try {
      await adminApi.rejectWhiteLabel(id, reason);
      loadWhiteLabels();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reject white label');
    }
  };

  const getStatusBadgeClass = (status: string): string => {
    switch (status) {
      case 'active': return 'badge-success';
      case 'pending': return 'badge-warning';
      case 'suspended': return 'badge-error';
      case 'rejected': return 'badge-error';
      default: return 'badge-neutral';
    }
  };

  return (
    <div className="p-6">
      {/* Page Header */}
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold" style={{ color: 'var(--text-primary)' }}>
            White Label Management
          </h1>
          <p style={{ color: 'var(--text-secondary)' }}>
            Manage white label partners and configurations
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>
          + Create White Label
        </button>
      </div>

      {error && (
        <div className="alert alert-error mb-4">
          {error}
        </div>
      )}

      {/* Filters */}
      <div className="card mb-6">
        <div className="card-body">
          <div className="flex gap-4 items-end">
            <div className="w-40">
              <label className="form-label">Status</label>
              <select
                className="form-select"
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
              >
                <option value="">All</option>
                <option value="pending">Pending</option>
                <option value="active">Active</option>
                <option value="suspended">Suspended</option>
                <option value="rejected">Rejected</option>
              </select>
            </div>
          </div>
        </div>
      </div>

      {/* White Labels Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {loading ? (
          <div className="col-span-full flex items-center justify-center p-8">
            <div className="loader"></div>
          </div>
        ) : whiteLabels.length > 0 ? (
          whiteLabels.map((wl) => (
            <div key={wl.id} className="card">
              <div className="card-header flex items-center gap-3">
                <div
                  className="w-10 h-10 rounded-lg flex items-center justify-center text-white font-bold"
                  style={{ backgroundColor: wl.primaryColor || '#dc2626' }}
                >
                  {wl.name.charAt(0)}
                </div>
                <div className="flex-1">
                  <h3 className="font-semibold" style={{ color: 'var(--text-primary)' }}>
                    {wl.name}
                  </h3>
                  <span className={`badge ${getStatusBadgeClass(wl.status)}`}>
                    {wl.status}
                  </span>
                </div>
              </div>
              <div className="card-body">
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span style={{ color: 'var(--text-tertiary)' }}>Domain</span>
                    <span style={{ color: 'var(--text-secondary)' }}>{wl.domain}</span>
                  </div>
                  <div className="flex justify-between">
                    <span style={{ color: 'var(--text-tertiary)' }}>Owner</span>
                    <span style={{ color: 'var(--text-secondary)' }}>{wl.ownerName}</span>
                  </div>
                  <div className="flex justify-between">
                    <span style={{ color: 'var(--text-tertiary)' }}>Email</span>
                    <span style={{ color: 'var(--text-secondary)' }}>{wl.ownerEmail}</span>
                  </div>
                  <div className="flex justify-between">
                    <span style={{ color: 'var(--text-tertiary)' }}>Created</span>
                    <span style={{ color: 'var(--text-secondary)' }}>
                      {new Date(wl.createdAt).toLocaleDateString()}
                    </span>
                  </div>
                </div>
              </div>
              <div className="card-footer flex gap-2">
                {wl.status === 'pending' && (
                  <>
                    <button
                      className="btn btn-sm btn-success flex-1"
                      onClick={() => handleApprove(wl.id)}
                      disabled={processing === wl.id}
                    >
                      Approve
                    </button>
                    <button
                      className="btn btn-sm btn-danger flex-1"
                      onClick={() => handleReject(wl.id)}
                    >
                      Reject
                    </button>
                  </>
                )}
                {wl.status === 'active' && (
                  <button className="btn btn-sm btn-outline flex-1">
                    Manage
                  </button>
                )}
              </div>
            </div>
          ))
        ) : (
          <div className="col-span-full text-center py-8" style={{ color: 'var(--text-tertiary)' }}>
            No white labels found
          </div>
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="mt-6 flex justify-center">
          <div className="pagination">
            <button
              className="pagination-btn"
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={page === 1}
            >
              Previous
            </button>
            <span className="px-4 py-2" style={{ color: 'var(--text-secondary)' }}>
              Page {page} of {totalPages}
            </span>
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

      {/* Create Modal */}
      {showCreateModal && (
        <div className="modal-overlay" onClick={() => setShowCreateModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Create White Label</h3>
              <button className="btn btn-sm btn-outline" onClick={() => setShowCreateModal(false)}>
                ✕
              </button>
            </div>
            <div className="modal-body">
              <div className="form-group">
                <label className="form-label">Name</label>
                <input
                  type="text"
                  className="form-input"
                  value={newWhiteLabel.name}
                  onChange={(e) => setNewWhiteLabel({ ...newWhiteLabel, name: e.target.value })}
                  placeholder="White Label Name"
                />
              </div>
              <div className="form-group">
                <label className="form-label">Domain</label>
                <input
                  type="text"
                  className="form-input"
                  value={newWhiteLabel.domain}
                  onChange={(e) => setNewWhiteLabel({ ...newWhiteLabel, domain: e.target.value })}
                  placeholder="example.tigerwallet.io"
                />
              </div>
              <div className="form-group">
                <label className="form-label">Owner Name</label>
                <input
                  type="text"
                  className="form-input"
                  value={newWhiteLabel.ownerName}
                  onChange={(e) => setNewWhiteLabel({ ...newWhiteLabel, ownerName: e.target.value })}
                  placeholder="John Doe"
                />
              </div>
              <div className="form-group">
                <label className="form-label">Owner Email</label>
                <input
                  type="email"
                  className="form-input"
                  value={newWhiteLabel.ownerEmail}
                  onChange={(e) => setNewWhiteLabel({ ...newWhiteLabel, ownerEmail: e.target.value })}
                  placeholder="john@example.com"
                />
              </div>
              <div className="form-group">
                <label className="form-label">Primary Color</label>
                <input
                  type="color"
                  className="form-input"
                  value={newWhiteLabel.primaryColor}
                  onChange={(e) => setNewWhiteLabel({ ...newWhiteLabel, primaryColor: e.target.value })}
                  style={{ height: '40px', padding: '4px' }}
                />
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowCreateModal(false)}>
                Cancel
              </button>
              <button className="btn btn-primary" onClick={handleCreate}>
                Create
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default WhiteLabelsPage;
