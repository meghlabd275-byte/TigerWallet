import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { whiteLabelService } from '../services/api';

interface WhiteLabel {
  id: string;
  client_id: string;
  name: string;
  domain: string;
  domain_verified: boolean;
  status: string;
  primary_color: string | null;
  secondary_color: string | null;
  platform_fee_percent: number;
  max_users: number;
  current_users: number;
  created_at: string;
  activated_at: string | null;
  expires_at: string | null;
}

export default function WhiteLabels() {
  const { isDark } = useTheme();
  const [whiteLabels, setWhiteLabels] = useState<WhiteLabel[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [selectedWL, setSelectedWL] = useState<WhiteLabel | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  useEffect(() => {
    loadWhiteLabels();
  }, [page, statusFilter]);

  const loadWhiteLabels = async () => {
    setLoading(true);
    setError('');
    
    try {
      const response = await whiteLabelService.getWhiteLabels({
        page,
        limit: 20,
        status: statusFilter || undefined
      });
      
      setWhiteLabels(response.data);
      setTotalPages(response.meta.total_pages);
    } catch (err: any) {
      setError(err.message || 'Failed to load white labels');
    } finally {
      setLoading(false);
    }
  };

  const handleApprove = async (id: string) => {
    setActionLoading(true);
    try {
      await whiteLabelService.approveWhiteLabel(id);
      alert('White label approved successfully');
      loadWhiteLabels();
    } catch (err: any) {
      alert(err.message || 'Failed to approve white label');
    } finally {
      setActionLoading(false);
    }
  };

  const handleSuspend = async (id: string) => {
    const reason = prompt('Please provide a reason for suspension:');
    if (!reason) return;
    
    setActionLoading(true);
    try {
      await whiteLabelService.suspendWhiteLabel(id, reason);
      alert('White label suspended successfully');
      loadWhiteLabels();
    } catch (err: any) {
      alert(err.message || 'Failed to suspend white label');
    } finally {
      setActionLoading(false);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'bg-green-100 text-green-800';
      case 'pending': return 'bg-yellow-100 text-yellow-800';
      case 'suspended': return 'bg-red-100 text-red-800';
      case 'inactive': return 'bg-gray-100 text-gray-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
      <div className="p-6">
        <div className="flex justify-between items-center mb-6">
          <h1 className={`text-3xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
            White Labels
          </h1>
          <button
            onClick={() => loadWhiteLabels()}
            className={`px-4 py-2 rounded-lg ${isDark ? 'bg-blue-600 hover:bg-blue-700' : 'bg-blue-500 hover:bg-blue-600'} text-white`}
          >
            Refresh
          </button>
        </div>

        <div className={`rounded-lg shadow p-4 mb-6 ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          <select
            value={statusFilter}
            onChange={(e) => { setStatusFilter(e.target.value); setPage(1); }}
            className={`px-4 py-2 rounded-lg border ${
              isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
            }`}
          >
            <option value="">All Status</option>
            <option value="active">Active</option>
            <option value="pending">Pending</option>
            <option value="suspended">Suspended</option>
            <option value="inactive">Inactive</option>
          </select>
        </div>

        {error && (
          <div className="mb-4 p-4 bg-red-100 border border-red-400 text-red-700 rounded">
            {error}
          </div>
        )}

        <div className={`rounded-lg shadow overflow-hidden ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          {loading ? (
            <div className="p-8 text-center">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-blue-500 border-t-transparent"></div>
              <p className={`mt-2 ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Loading...</p>
            </div>
          ) : whiteLabels.length === 0 ? (
            <div className="p-8 text-center">
              <p className={`${isDark ? 'text-gray-400' : 'text-gray-600'}`}>No white labels found</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
                  <tr>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Client</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Domain</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Verified</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Status</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Users</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Platform Fee</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Created</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Actions</th>
                  </tr>
                </thead>
                <tbody className={`divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
                  {whiteLabels.map((wl) => (
                    <tr key={wl.id} className={isDark ? 'hover:bg-gray-700' : 'hover:bg-gray-50'}>
                      <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                        <div>
                          <p className="font-medium">{wl.name}</p>
                          <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>ID: {wl.client_id}</p>
                        </div>
                      </td>
                      <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                        <div className="flex items-center gap-2">
                          <span>{wl.domain}</span>
                          {wl.domain_verified ? (
                            <span className="text-green-500" title="Verified">✓</span>
                          ) : (
                            <span className="text-yellow-500" title="Not Verified">!</span>
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-4">
                        <span className={`px-2 py-1 text-xs font-medium rounded-full ${wl.domain_verified ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'}`}>
                          {wl.domain_verified ? 'Yes' : 'No'}
                        </span>
                      </td>
                      <td className="px-4 py-4">
                        <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(wl.status)}`}>
                          {wl.status}
                        </span>
                      </td>
                      <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                        {wl.current_users} / {wl.max_users}
                      </td>
                      <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                        {wl.platform_fee_percent}%
                      </td>
                      <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                        {new Date(wl.created_at).toLocaleDateString()}
                      </td>
                      <td className="px-4 py-4">
                        <div className="flex gap-2">
                          <button
                            onClick={() => setSelectedWL(wl)}
                            className={`px-3 py-1 text-sm rounded ${isDark ? 'bg-blue-600 hover:bg-blue-700' : 'bg-blue-500 hover:bg-blue-600'} text-white`}
                          >
                            View
                          </button>
                          {wl.status === 'pending' && (
                            <button
                              onClick={() => handleApprove(wl.id)}
                              disabled={actionLoading}
                              className={`px-3 py-1 text-sm rounded ${isDark ? 'bg-green-600 hover:bg-green-700' : 'bg-green-500 hover:bg-green-600'} text-white`}
                            >
                              Approve
                            </button>
                          )}
                          {wl.status === 'active' && (
                            <button
                              onClick={() => handleSuspend(wl.id)}
                              disabled={actionLoading}
                              className={`px-3 py-1 text-sm rounded ${isDark ? 'bg-red-600 hover:bg-red-700' : 'bg-red-500 hover:bg-red-600'} text-white`}
                            >
                              Suspend
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {totalPages > 1 && (
          <div className="flex justify-center items-center gap-2 mt-6">
            <button
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={page === 1}
              className={`px-4 py-2 rounded-lg ${isDark ? 'bg-gray-700 hover:bg-gray-600' : 'bg-white hover:bg-gray-50'} ${page === 1 ? 'opacity-50' : ''} ${isDark ? 'text-white' : 'text-gray-700'}`}
            >
              Previous
            </button>
            <span className={`px-4 py-2 ${isDark ? 'text-white' : 'text-gray-700'}`}>
              Page {page} of {totalPages}
            </span>
            <button
              onClick={() => setPage(p => Math.min(totalPages, p + 1))}
              disabled={page === totalPages}
              className={`px-4 py-2 rounded-lg ${isDark ? 'bg-gray-700 hover:bg-gray-600' : 'bg-white hover:bg-gray-50'} ${page === totalPages ? 'opacity-50' : ''} ${isDark ? 'text-white' : 'text-gray-700'}`}
            >
              Next
            </button>
          </div>
        )}

        {selectedWL && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className={`rounded-lg shadow-xl p-6 max-w-2xl w-full mx-4 ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
              <div className="flex justify-between items-start mb-4">
                <h2 className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                  {selectedWL.name}
                </h2>
                <button
                  onClick={() => setSelectedWL(null)}
                  className={`text-2xl ${isDark ? 'text-gray-400 hover:text-white' : 'text-gray-500 hover:text-gray-700'}`}
                >
                  ×
                </button>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Client ID</label>
                  <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedWL.client_id}</p>
                </div>
                <div>
                  <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Domain</label>
                  <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedWL.domain}</p>
                </div>
                <div>
                  <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Status</label>
                  <span className={`px-2 py-1 text-sm font-medium rounded-full ${getStatusColor(selectedWL.status)}`}>
                    {selectedWL.status}
                  </span>
                </div>
                <div>
                  <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Platform Fee</label>
                  <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedWL.platform_fee_percent}%</p>
                </div>
                <div>
                  <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Users</label>
                  <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedWL.current_users} / {selectedWL.max_users}</p>
                </div>
                <div>
                  <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Domain Verified</label>
                  <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedWL.domain_verified ? 'Yes' : 'No'}</p>
                </div>
                {selectedWL.primary_color && (
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Primary Color</label>
                    <div className="flex items-center gap-2 mt-1">
                      <div className="w-6 h-6 rounded" style={{ backgroundColor: selectedWL.primary_color }}></div>
                      <span className={isDark ? 'text-white' : 'text-gray-900'}>{selectedWL.primary_color}</span>
                    </div>
                  </div>
                )}
                {selectedWL.secondary_color && (
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Secondary Color</label>
                    <div className="flex items-center gap-2 mt-1">
                      <div className="w-6 h-6 rounded" style={{ backgroundColor: selectedWL.secondary_color }}></div>
                      <span className={isDark ? 'text-white' : 'text-gray-900'}>{selectedWL.secondary_color}</span>
                    </div>
                  </div>
                )}
                <div>
                  <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Created</label>
                  <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{new Date(selectedWL.created_at).toLocaleString()}</p>
                </div>
                {selectedWL.expires_at && (
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Expires</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{new Date(selectedWL.expires_at).toLocaleString()}</p>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
