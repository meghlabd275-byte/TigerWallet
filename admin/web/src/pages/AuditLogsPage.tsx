// TigerWallet Admin - Audit Logs Page
// View admin activity and audit trails

import React, { useState, useEffect } from 'react';
import { adminApi } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

interface AuditLog {
  id: string;
  adminId: string;
  adminEmail: string;
  action: string;
  resource: string;
  resourceId?: string;
  details: Record<string, unknown>;
  ipAddress: string;
  timestamp: string;
}

const AuditLogsPage: React.FC = () => {
  const { resolvedTheme } = useTheme();
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [actionFilter, setActionFilter] = useState('');
  const [adminFilter, setAdminFilter] = useState('');

  useEffect(() => {
    loadAuditLogs();
  }, [page, actionFilter, adminFilter]);

  const loadAuditLogs = async () => {
    try {
      setLoading(true);
      const response = await adminApi.getAuditLogs({
        page,
        pageSize: 20,
        action: actionFilter || undefined,
        adminId: adminFilter || undefined,
      });
      setLogs(response.data || []);
      setTotalPages(response.totalPages || 1);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load audit logs');
    } finally {
      setLoading(false);
    }
  };

  const getColors = () => ({
    text: resolvedTheme === 'dark' ? '#f9fafb' : '#111827',
    textSecondary: resolvedTheme === 'dark' ? '#9ca3af' : '#6b7280',
    bgCard: resolvedTheme === 'dark' ? '#1e293b' : '#ffffff',
    border: resolvedTheme === 'dark' ? '#374151' : '#e5e7eb',
    primary: '#dc2626',
  });

  const colors = getColors();

  const getActionBadgeClass = (action: string): string => {
    if (action.includes('create') || action.includes('approve')) return 'badge-success';
    if (action.includes('delete') || action.includes('reject')) return 'badge-error';
    if (action.includes('update') || action.includes('edit')) return 'badge-warning';
    return 'badge-neutral';
  };

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Audit Logs</h1>
        <p style={{ color: colors.textSecondary }}>Track all admin activities and changes</p>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      {/* Filters */}
      <div className="card mb-6" style={{ backgroundColor: colors.bgCard }}>
        <div className="card-body">
          <div className="flex gap-4">
            <div className="w-48">
              <label className="form-label">Action Type</label>
              <select
                className="form-select"
                value={actionFilter}
                onChange={(e) => setActionFilter(e.target.value)}
              >
                <option value="">All Actions</option>
                <option value="create">Create</option>
                <option value="update">Update</option>
                <option value="delete">Delete</option>
                <option value="login">Login</option>
                <option value="logout">Logout</option>
              </select>
            </div>
            <div className="w-48">
              <label className="form-label">Admin</label>
              <input
                type="text"
                className="form-input"
                placeholder="Search admin..."
                value={adminFilter}
                onChange={(e) => setAdminFilter(e.target.value)}
              />
            </div>
          </div>
        </div>
      </div>

      {/* Audit Logs Table */}
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
                  <th>Timestamp</th>
                  <th>Admin</th>
                  <th>Action</th>
                  <th>Resource</th>
                  <th>IP Address</th>
                </tr>
              </thead>
              <tbody>
                {logs.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="text-center py-8" style={{ color: colors.textSecondary }}>
                      No audit logs found
                    </td>
                  </tr>
                ) : (
                  logs.map((log) => (
                    <tr key={log.id}>
                      <td style={{ color: colors.textSecondary }}>
                        {new Date(log.timestamp).toLocaleString()}
                      </td>
                      <td style={{ color: colors.text }}>{log.adminEmail}</td>
                      <td>
                        <span className={`badge ${getActionBadgeClass(log.action)}`}>
                          {log.action}
                        </span>
                      </td>
                      <td style={{ color: colors.text }}>
                        {log.resource}
                        {log.resourceId && <span className="text-xs ml-2">({log.resourceId})</span>}
                      </td>
                      <td style={{ color: colors.textSecondary }}>{log.ipAddress}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          )}
        </div>

        {totalPages > 1 && (
          <div className="card-footer flex justify-between items-center">
            <span style={{ color: colors.textSecondary }}>Page {page} of {totalPages}</span>
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
    </div>
  );
};

export default AuditLogsPage;
