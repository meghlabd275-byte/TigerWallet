import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { kycService } from '../services/api';

interface KYCSubmission {
  id: string;
  user_id: string;
  level: number;
  document_type: string | null;
  document_number: string | null;
  first_name: string | null;
  last_name: string | null;
  country: string | null;
  address: string | null;
  status: string;
  reject_reason: string | null;
  reviewed_by: string | null;
  reviewed_at: string | null;
  created_at: string;
}

export default function KYC() {
  const { isDark } = useTheme();
  const [submissions, setSubmissions] = useState<KYCSubmission[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [levelFilter, setLevelFilter] = useState(0);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [selectedSubmission, setSelectedSubmission] = useState<KYCSubmission | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [rejectReason, setRejectReason] = useState('');

  useEffect(() => {
    loadSubmissions();
  }, [page, statusFilter, levelFilter]);

  const loadSubmissions = async () => {
    setLoading(true);
    setError('');
    
    try {
      const response = await kycService.getSubmissions({
        page,
        limit: 20,
        status: statusFilter || undefined,
        level: levelFilter || undefined
      });
      
      setSubmissions(response.data);
      setTotalPages(response.meta.total_pages);
    } catch (err: any) {
      setError(err.message || 'Failed to load KYC submissions');
    } finally {
      setLoading(false);
    }
  };

  const handleApprove = async (id: string) => {
    if (!confirm('Are you sure you want to approve this KYC submission?')) return;
    
    setActionLoading(true);
    try {
      await kycService.approveKYC(id);
      alert('KYC approved successfully');
      loadSubmissions();
      setSelectedSubmission(null);
    } catch (err: any) {
      alert(err.message || 'Failed to approve KYC');
    } finally {
      setActionLoading(false);
    }
  };

  const handleReject = async (id: string) => {
    if (!rejectReason.trim()) {
      alert('Please provide a reason for rejection');
      return;
    }
    
    setActionLoading(true);
    try {
      await kycService.rejectKYC(id, rejectReason);
      alert('KYC rejected successfully');
      loadSubmissions();
      setSelectedSubmission(null);
      setRejectReason('');
    } catch (err: any) {
      alert(err.message || 'Failed to reject KYC');
    } finally {
      setActionLoading(false);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'approved': return 'bg-green-100 text-green-800';
      case 'pending': return 'bg-yellow-100 text-yellow-800';
      case 'rejected': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
      <div className="p-6">
        <div className="flex justify-between items-center mb-6">
          <h1 className={`text-3xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
            KYC Submissions
          </h1>
          <button
            onClick={() => loadSubmissions()}
            className={`px-4 py-2 rounded-lg ${isDark ? 'bg-blue-600 hover:bg-blue-700' : 'bg-blue-500 hover:bg-blue-600'} text-white`}
          >
            Refresh
          </button>
        </div>

        {/* Filters */}
        <div className={`rounded-lg shadow p-4 mb-6 ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          <div className="flex flex-wrap gap-4">
            <select
              value={statusFilter}
              onChange={(e) => { setStatusFilter(e.target.value); setPage(1); }}
              className={`px-4 py-2 rounded-lg border ${
                isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
              }`}
            >
              <option value="">All Status</option>
              <option value="pending">Pending</option>
              <option value="approved">Approved</option>
              <option value="rejected">Rejected</option>
            </select>
            <select
              value={levelFilter}
              onChange={(e) => { setLevelFilter(parseInt(e.target.value)); setPage(1); }}
              className={`px-4 py-2 rounded-lg border ${
                isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
              }`}
            >
              <option value="0">All Levels</option>
              <option value="1">Level 1</option>
              <option value="2">Level 2</option>
              <option value="3">Level 3</option>
            </select>
          </div>
        </div>

        {/* Error Message */}
        {error && (
          <div className="mb-4 p-4 bg-red-100 border border-red-400 text-red-700 rounded">
            {error}
          </div>
        )}

        {/* Submissions Table */}
        <div className={`rounded-lg shadow overflow-hidden ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          {loading ? (
            <div className="p-8 text-center">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-blue-500 border-t-transparent"></div>
              <p className={`mt-2 ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Loading...</p>
            </div>
          ) : submissions.length === 0 ? (
            <div className="p-8 text-center">
              <p className={`${isDark ? 'text-gray-400' : 'text-gray-600'}`}>No KYC submissions found</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
                  <tr>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>User</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Level</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Document</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Country</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Status</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Submitted</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Actions</th>
                  </tr>
                </thead>
                <tbody className={`divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
                  {submissions.map((submission) => (
                    <tr key={submission.id} className={isDark ? 'hover:bg-gray-700' : 'hover:bg-gray-50'}>
                      <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                        <div>
                          <p className="font-medium">{submission.first_name} {submission.last_name}</p>
                          <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>ID: {submission.user_id}</p>
                        </div>
                      </td>
                      <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                        Level {submission.level}
                      </td>
                      <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                        {submission.document_type || 'N/A'}
                      </td>
                      <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                        {submission.country || 'N/A'}
                      </td>
                      <td className="px-4 py-4">
                        <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(submission.status)}`}>
                          {submission.status}
                        </span>
                      </td>
                      <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                        {new Date(submission.created_at).toLocaleDateString()}
                      </td>
                      <td className="px-4 py-4">
                        <div className="flex gap-2">
                          <button
                            onClick={() => setSelectedSubmission(submission)}
                            className={`px-3 py-1 text-sm rounded ${isDark ? 'bg-blue-600 hover:bg-blue-700' : 'bg-blue-500 hover:bg-blue-600'} text-white`}
                          >
                            Review
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Pagination */}
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

        {/* Review Modal */}
        {selectedSubmission && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className={`rounded-lg shadow-xl p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
              <div className="flex justify-between items-start mb-4">
                <h2 className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                  KYC Review
                </h2>
                <button
                  onClick={() => { setSelectedSubmission(null); setRejectReason(''); }}
                  className={`text-2xl ${isDark ? 'text-gray-400 hover:text-white' : 'text-gray-500 hover:text-gray-700'}`}
                >
                  ×
                </button>
              </div>
              
              <div className="space-y-4 mb-6">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>First Name</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedSubmission.first_name || 'N/A'}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Last Name</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedSubmission.last_name || 'N/A'}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Document Type</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedSubmission.document_type || 'N/A'}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Document Number</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedSubmission.document_number || 'N/A'}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Country</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedSubmission.country || 'N/A'}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Level</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>Level {selectedSubmission.level}</p>
                  </div>
                  <div className="col-span-2">
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Address</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedSubmission.address || 'N/A'}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Status</label>
                    <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(selectedSubmission.status)}`}>
                      {selectedSubmission.status}
                    </span>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Submitted</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{new Date(selectedSubmission.created_at).toLocaleString()}</p>
                  </div>
                </div>
              </div>

              {selectedSubmission.status === 'pending' && (
                <div className="border-t pt-4">
                  <div className="mb-4">
                    <label className={`block text-sm font-medium mb-2 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                      Rejection Reason (required for reject)
                    </label>
                    <textarea
                      value={rejectReason}
                      onChange={(e) => setRejectReason(e.target.value)}
                      rows={3}
                      className={`w-full px-4 py-2 rounded-lg border ${
                        isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
                      }`}
                      placeholder="Enter reason for rejection..."
                    />
                  </div>
                  
                  <div className="flex gap-4">
                    <button
                      onClick={() => handleApprove(selectedSubmission.id)}
                      disabled={actionLoading}
                      className={`flex-1 py-3 rounded-lg bg-green-600 hover:bg-green-700 text-white font-medium`}
                    >
                      Approve
                    </button>
                    <button
                      onClick={() => handleReject(selectedSubmission.id)}
                      disabled={actionLoading || !rejectReason.trim()}
                      className={`flex-1 py-3 rounded-lg bg-red-600 hover:bg-red-700 text-white font-medium disabled:opacity-50`}
                    >
                      Reject
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
