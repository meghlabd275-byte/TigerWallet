// TigerWallet Admin - Admin Management Page
// Manage admin users

import React, { useState, useEffect } from 'react';
import { adminApi } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

interface Admin {
  id: string;
  email: string;
  username: string;
  role: string;
  status: string;
  twoFactorEnabled: boolean;
  createdAt: string;
  lastLogin?: string;
}

const AdminsPage: React.FC = () => {
  const { resolvedTheme } = useTheme();
  const [admins, setAdmins] = useState<Admin[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newAdmin, setNewAdmin] = useState({
    email: '',
    username: '',
    password: '',
    role: 'admin',
  });

  useEffect(() => {
    loadAdmins();
  }, []);

  const loadAdmins = async () => {
    try {
      setLoading(true);
      const response = await adminApi.listAdmins();
      setAdmins(response.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load admins');
    } finally {
      setLoading(false);
    }
  };

  const getColors = () => ({
    text: resolvedTheme === 'dark' ? '#f9fafb' : '#111827',
    textSecondary: resolvedTheme === 'dark' ? '#9ca3af' : '#6b7280',
    bgCard: resolvedTheme === 'dark' ? '#1e293b' : '#ffffff',
    border: resolvedTheme === 'dark' ? '#374151' : '#e5e7eb',
  });

  const colors = getColors();

  const handleCreateAdmin = async () => {
    try {
      await adminApi.createAdmin(newAdmin);
      setShowCreateModal(false);
      loadAdmins();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create admin');
    }
  };

  const handleSuspendAdmin = async (id: string) => {
    if (!confirm('Are you sure you want to suspend this admin?')) return;
    try {
      await adminApi.suspendAdmin(id);
      loadAdmins();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to suspend admin');
    }
  };

  const handleDeleteAdmin = async (id: string) => {
    if (!confirm('Are you sure you want to delete this admin?')) return;
    try {
      await adminApi.deleteAdmin(id);
      loadAdmins();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete admin');
    }
  };

  const getRoleBadge = (role: string) => {
    switch (role) {
      case 'super_admin': return 'badge-error';
      case 'admin': return 'badge-warning';
      case 'support': return 'badge-info';
      default: return 'badge-neutral';
    }
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Admin Users</h1>
        <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>
          + Add Admin
        </button>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      <div className="card" style={{ backgroundColor: colors.bgCard }}>
        <div className="card-body p-0">
          {loading ? (
            <div className="flex items-center justify-center p-8">
              <div className="loader"></div>
            </div>
          ) : (
            <table className="table">
              <thead>
                <tr>
                  <th>Username</th>
                  <th>Email</th>
                  <th>Role</th>
                  <th>Status</th>
                  <th>2FA</th>
                  <th>Created</th>
                  <th>Last Login</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {admins.length === 0 ? (
                  <tr>
                    <td colSpan={8} className="text-center py-8" style={{ color: colors.textSecondary }}>
                      No admins found
                    </td>
                  </tr>
                ) : (
                  admins.map((admin) => (
                    <tr key={admin.id}>
                      <td style={{ color: colors.text }}>{admin.username}</td>
                      <td style={{ color: colors.textSecondary }}>{admin.email}</td>
                      <td>
                        <span className={`badge ${getRoleBadge(admin.role)}`}>
                          {admin.role}
                        </span>
                      </td>
                      <td>
                        <span className={`badge ${admin.status === 'active' ? 'badge-success' : 'badge-neutral'}`}>
                          {admin.status}
                        </span>
                      </td>
                      <td>
                        {admin.twoFactorEnabled ? '✓' : '✗'}
                      </td>
                      <td style={{ color: colors.textSecondary }}>
                        {new Date(admin.createdAt).toLocaleDateString()}
                      </td>
                      <td style={{ color: colors.textSecondary }}>
                        {admin.lastLogin ? new Date(admin.lastLogin).toLocaleDateString() : 'Never'}
                      </td>
                      <td>
                        <div className="flex gap-2">
                          <button
                            className="btn btn-sm btn-outline"
                            onClick={() => handleSuspendAdmin(admin.id)}
                          >
                            Suspend
                          </button>
                          <button
                            className="btn btn-sm btn-danger"
                            onClick={() => handleDeleteAdmin(admin.id)}
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
          )}
        </div>
      </div>

      {showCreateModal && (
        <div className="modal-overlay" onClick={() => setShowCreateModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Add Admin User</h3>
              <button className="btn btn-sm btn-outline" onClick={() => setShowCreateModal(false)}>✕</button>
            </div>
            <div className="modal-body">
              <div className="form-group">
                <label className="form-label">Username</label>
                <input
                  type="text"
                  className="form-input"
                  value={newAdmin.username}
                  onChange={(e) => setNewAdmin({ ...newAdmin, username: e.target.value })}
                />
              </div>
              <div className="form-group">
                <label className="form-label">Email</label>
                <input
                  type="email"
                  className="form-input"
                  value={newAdmin.email}
                  onChange={(e) => setNewAdmin({ ...newAdmin, email: e.target.value })}
                />
              </div>
              <div className="form-group">
                <label className="form-label">Password</label>
                <input
                  type="password"
                  className="form-input"
                  value={newAdmin.password}
                  onChange={(e) => setNewAdmin({ ...newAdmin, password: e.target.value })}
                />
              </div>
              <div className="form-group">
                <label className="form-label">Role</label>
                <select
                  className="form-select"
                  value={newAdmin.role}
                  onChange={(e) => setNewAdmin({ ...newAdmin, role: e.target.value })}
                >
                  <option value="admin">Admin</option>
                  <option value="support">Support</option>
                  <option value="finance">Finance</option>
                  <option value="compliance">Compliance</option>
                </select>
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowCreateModal(false)}>Cancel</button>
              <button className="btn btn-primary" onClick={handleCreateAdmin}>Create</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default AdminsPage;
