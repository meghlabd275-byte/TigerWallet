/**
 * TigerWallet White Labels Management Page
 * Complete CRUD operations with full functionality
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import api from '../services/api';

interface WhiteLabel {
  id: string;
  name: string;
  domain: string;
  status: string;
  fee_percent: number;
  profit_share_percent: number;
  plan_tier: string;
  max_users: number;
  monthly_fee: number;
  custom_branding: boolean;
  created_at: string;
  updated_at: string;
}

interface WhiteLabelFormData {
  name: string;
  domain: string;
  plan_tier: string;
  max_users: number;
  monthly_fee: number;
}

const WhiteLabels: React.FC = () => {
  const { theme, toggleTheme } = useTheme();
  const [whiteLabels, setWhiteLabels] = useState<WhiteLabel[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [showModal, setShowModal] = useState(false);
  const [editingLabel, setEditingLabel] = useState<WhiteLabel | null>(null);
  const [formData, setFormData] = useState<WhiteLabelFormData>({
    name: '',
    domain: '',
    plan_tier: 'basic',
    max_users: 1000,
    monthly_fee: 0,
  });
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    loadWhiteLabels();
  }, [searchTerm, statusFilter]);

  const loadWhiteLabels = async () => {
    try {
      setLoading(true);
      const response = await api.getWhiteLabels({
        status: statusFilter || undefined,
        search: searchTerm || undefined,
      });
      if (response.data) {
        setWhiteLabels(response.data.white_labels);
      } else {
        setError(response.error || 'Failed to load white labels');
      }
    } catch (err) {
      setError('Failed to connect to server');
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async () => {
    if (!formData.name || !formData.domain) {
      setError('Name and domain are required');
      return;
    }

    try {
      setSubmitting(true);
      const response = await api.createWhiteLabel(formData);
      if (response.error) {
        setError(response.error);
      } else {
        setShowModal(false);
        setFormData({ name: '', domain: '', plan_tier: 'basic', max_users: 1000, monthly_fee: 0 });
        loadWhiteLabels();
      }
    } catch (err) {
      setError('Failed to create white label');
    } finally {
      setSubmitting(false);
    }
  };

  const handleUpdate = async () => {
    if (!editingLabel) return;

    try {
      setSubmitting(true);
      const response = await api.updateWhiteLabel(editingLabel.id, formData);
      if (response.error) {
        setError(response.error);
      } else {
        setEditingLabel(null);
        setShowModal(false);
        setFormData({ name: '', domain: '', plan_tier: 'basic', max_users: 1000, monthly_fee: 0 });
        loadWhiteLabels();
      }
    } catch (err) {
      setError('Failed to update white label');
    } finally {
      setSubmitting(false);
    }
  };

  const handleApprove = async (id: string) => {
    try {
      await api.approveWhiteLabel(id);
      loadWhiteLabels();
    } catch (err) {
      setError('Failed to approve white label');
    }
  };

  const handleSuspend = async (id: string) => {
    try {
      await api.suspendWhiteLabel(id);
      loadWhiteLabels();
    } catch (err) {
      setError('Failed to suspend white label');
    }
  };

  const handleRevoke = async (id: string) => {
    if (!confirm('Are you sure you want to revoke this white label?')) return;
    
    try {
      await api.revokeWhiteLabel(id);
      loadWhiteLabels();
    } catch (err) {
      setError('Failed to revoke white label');
    }
  };

  const handleDestroy = async (id: string) => {
    if (!confirm('Are you sure you want to permanently destroy this white label? This cannot be undone!')) return;
    
    try {
      await api.destroyWhiteLabel(id);
      loadWhiteLabels();
    } catch (err) {
      setError('Failed to destroy white label');
    }
  };

  const openEditModal = (label: WhiteLabel) => {
    setEditingLabel(label);
    setFormData({
      name: label.name,
      domain: label.domain,
      plan_tier: label.plan_tier,
      max_users: label.max_users,
      monthly_fee: label.monthly_fee,
    });
    setShowModal(true);
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      case 'pending': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
      case 'suspended': return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      case 'revoked': return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <div className="min-h-screen bg-[var(--bg-primary)]">
      {/* Header */}
      <header className="bg-[var(--bg-secondary)] border-b border-[var(--border-color)] px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-[var(--text-primary)]">White Labels</h1>
            <p className="text-[var(--text-muted)]">Manage white label clients</p>
          </div>
          <div className="flex items-center gap-4">
            <button
              onClick={toggleTheme}
              className="p-2 rounded-lg bg-[var(--bg-tertiary)] hover:bg-[var(--hover-bg)] transition-colors"
            >
              {theme === 'dark' ? (
                <svg className="w-6 h-6 text-[var(--text-primary)]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
                </svg>
              ) : (
                <svg className="w-6 h-6 text-[var(--text-primary)]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
                </svg>
              )}
            </button>
            <button
              onClick={() => {
                setEditingLabel(null);
                setFormData({ name: '', domain: '', plan_tier: 'basic', max_users: 1000, monthly_fee: 0 });
                setShowModal(true);
              }}
              className="px-4 py-2 bg-[var(--accent-primary)] text-white rounded-lg hover:opacity-90 transition-opacity"
            >
              Create White Label
            </button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="p-6">
        {/* Filters */}
        <div className="flex flex-wrap gap-4 mb-6">
          <input
            type="text"
            placeholder="Search white labels..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="px-4 py-2 bg-[var(--input-bg)] border border-[var(--border-color)] rounded-lg text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--accent-primary)]"
          />
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="px-4 py-2 bg-[var(--input-bg)] border border-[var(--border-color)] rounded-lg text-[var(--text-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--accent-primary)]"
          >
            <option value="">All Status</option>
            <option value="pending">Pending</option>
            <option value="active">Active</option>
            <option value="suspended">Suspended</option>
            <option value="revoked">Revoked</option>
          </select>
          <button
            onClick={loadWhiteLabels}
            className="px-4 py-2 bg-[var(--bg-tertiary)] text-[var(--text-primary)] rounded-lg hover:bg-[var(--hover-bg)] transition-colors"
          >
            Refresh
          </button>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-[var(--error)] bg-opacity-10 border border-[var(--error)] rounded-lg text-[var(--error)]">
            {error}
            <button onClick={() => setError(null)} className="ml-4 underline">Dismiss</button>
          </div>
        )}

        {/* Table */}
        <div className="bg-[var(--card-bg)] rounded-xl border border-[var(--border-color)] overflow-hidden">
          <table className="w-full">
            <thead className="bg-[var(--bg-tertiary)]">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">Name</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">Domain</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">Plan</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">Fee %</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">Users</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">Created</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--border-color)]">
              {loading ? (
                <tr>
                  <td colSpan={8} className="px-6 py-8 text-center text-[var(--text-muted)]">
                    Loading...
                  </td>
                </tr>
              ) : whiteLabels.length === 0 ? (
                <tr>
                  <td colSpan={8} className="px-6 py-8 text-center text-[var(--text-muted)]">
                    No white labels found
                  </td>
                </tr>
              ) : (
                whiteLabels.map((label) => (
                  <tr key={label.id} className="hover:bg-[var(--hover-bg)] transition-colors">
                    <td className="px-6 py-4 whitespace-nowrap text-[var(--text-primary)] font-medium">
                      {label.name}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-[var(--text-secondary)]">
                      {label.domain}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(label.status)}`}>
                        {label.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-[var(--text-secondary)]">
                      {label.plan_tier}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-[var(--text-secondary)]">
                      {label.fee_percent}%
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-[var(--text-secondary)]">
                      {label.max_users.toLocaleString()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-[var(--text-muted)]">
                      {new Date(label.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => openEditModal(label)}
                          className="text-[var(--accent-primary)] hover:underline text-sm"
                        >
                          Edit
                        </button>
                        {label.status === 'pending' && (
                          <button
                            onClick={() => handleApprove(label.id)}
                            className="text-green-600 hover:underline text-sm"
                          >
                            Approve
                          </button>
                        )}
                        {label.status === 'active' && (
                          <button
                            onClick={() => handleSuspend(label.id)}
                            className="text-yellow-600 hover:underline text-sm"
                          >
                            Suspend
                          </button>
                        )}
                        {label.status === 'suspended' && (
                          <button
                            onClick={() => handleApprove(label.id)}
                            className="text-green-600 hover:underline text-sm"
                          >
                            Activate
                          </button>
                        )}
                        {label.status !== 'revoked' && label.status !== 'pending' && (
                          <button
                            onClick={() => handleRevoke(label.id)}
                            className="text-red-600 hover:underline text-sm"
                          >
                            Revoke
                          </button>
                        )}
                        <button
                          onClick={() => handleDestroy(label.id)}
                          className="text-red-800 hover:underline text-sm font-bold"
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </main>

      {/* Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-[var(--card-bg)] rounded-xl p-6 w-full max-w-md mx-4">
            <h2 className="text-xl font-bold text-[var(--text-primary)] mb-4">
              {editingLabel ? 'Edit White Label' : 'Create White Label'}
            </h2>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-[var(--text-secondary)] mb-1">
                  Name *
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-4 py-2 bg-[var(--input-bg)] border border-[var(--border-color)] rounded-lg text-[var(--text-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--accent-primary)]"
                  placeholder="My White Label"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-[var(--text-secondary)] mb-1">
                  Domain *
                </label>
                <input
                  type="text"
                  value={formData.domain}
                  onChange={(e) => setFormData({ ...formData, domain: e.target.value })}
                  className="w-full px-4 py-2 bg-[var(--input-bg)] border border-[var(--border-color)] rounded-lg text-[var(--text-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--accent-primary)]"
                  placeholder="mydomain.com"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-[var(--text-secondary)] mb-1">
                  Plan Tier
                </label>
                <select
                  value={formData.plan_tier}
                  onChange={(e) => setFormData({ ...formData, plan_tier: e.target.value })}
                  className="w-full px-4 py-2 bg-[var(--input-bg)] border border-[var(--border-color)] rounded-lg text-[var(--text-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--accent-primary)]"
                >
                  <option value="basic">Basic</option>
                  <option value="professional">Professional</option>
                  <option value="enterprise">Enterprise</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-[var(--text-secondary)] mb-1">
                  Max Users
                </label>
                <input
                  type="number"
                  value={formData.max_users}
                  onChange={(e) => setFormData({ ...formData, max_users: parseInt(e.target.value) || 0 })}
                  className="w-full px-4 py-2 bg-[var(--input-bg)] border border-[var(--border-color)] rounded-lg text-[var(--text-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--accent-primary)]"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-[var(--text-secondary)] mb-1">
                  Monthly Fee (USD)
                </label>
                <input
                  type="number"
                  step="0.01"
                  value={formData.monthly_fee}
                  onChange={(e) => setFormData({ ...formData, monthly_fee: parseFloat(e.target.value) || 0 })}
                  className="w-full px-4 py-2 bg-[var(--input-bg)] border border-[var(--border-color)] rounded-lg text-[var(--text-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--accent-primary)]"
                />
              </div>
            </div>

            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => {
                  setShowModal(false);
                  setEditingLabel(null);
                }}
                className="px-4 py-2 text-[var(--text-secondary)] hover:bg-[var(--hover-bg)] rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={editingLabel ? handleUpdate : handleCreate}
                disabled={submitting}
                className="px-4 py-2 bg-[var(--accent-primary)] text-white rounded-lg hover:opacity-90 transition-opacity disabled:opacity-50"
              >
                {submitting ? 'Saving...' : editingLabel ? 'Update' : 'Create'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default WhiteLabels;
