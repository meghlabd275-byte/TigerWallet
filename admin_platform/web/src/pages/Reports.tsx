import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';

interface Report {
  id: number;
  report_id: string;
  report_type: string;
  format: string;
  period_start: string;
  period_end: string;
  status: string;
  file_url?: string;
  created_at: string;
}

export default function Reports() {
  const { isDark } = useTheme();
  const [reports, setReports] = useState<Report[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newReport, setNewReport] = useState({
    report_type: 'transactions',
    format: 'csv',
    period_start: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
    period_end: new Date().toISOString().split('T')[0],
  });

  useEffect(() => {
    loadReports();
  }, []);

  const loadReports = async () => {
    setLoading(true);
    setError('');
    try {
      const response = await fetch('/api/v1/reports');
      const data = await response.json();
      setReports(data.data || []);
    } catch (err: any) {
      setError(err.message || 'Failed to load reports');
    } finally {
      setLoading(false);
    }
  };

  const createReport = async () => {
    try {
      await fetch('/api/v1/reports', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newReport),
      });
      setShowCreateModal(false);
      loadReports();
    } catch (err: any) {
      setError(err.message || 'Failed to create report');
    }
  };

  const downloadReport = (report: Report) => {
    if (report.file_url) {
      window.open(report.file_url, '_blank');
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'ready': return 'text-green-600 bg-green-100 dark:bg-green-900';
      case 'generating': return 'text-yellow-600 bg-yellow-100 dark:bg-yellow-900';
      case 'failed': return 'text-red-600 bg-red-100 dark:bg-red-900';
      default: return 'text-gray-600 bg-gray-100 dark:bg-gray-900';
    }
  };

  const getFormatIcon = (format: string) => {
    switch (format) {
      case 'csv': return '📊';
      case 'xlsx': return '📗';
      case 'pdf': return '📕';
      default: return '📄';
    }
  };

  if (loading) {
    return (
      <div className={`flex items-center justify-center h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
        <div className="text-blue-600 text-xl">Loading...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={`flex items-center justify-center h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
        <div className="text-red-600 text-xl">{error}</div>
      </div>
    );
  }

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
      <div className="p-6">
        <div className="flex justify-between items-center mb-6">
          <h1 className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
            Reports
          </h1>
          <button
            onClick={() => setShowCreateModal(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            Generate Report
          </button>
        </div>

        {/* Report Types */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          {[
            { type: 'transactions', icon: '💸', label: 'Transactions' },
            { type: 'users', icon: '👥', label: 'Users' },
            { type: 'kyc', icon: '📋', label: 'KYC' },
            { type: 'revenue', icon: '💰', label: 'Revenue' },
            { type: 'compliance', icon: '⚖️', label: 'Compliance' },
          ].map((item) => (
            <div
              key={item.type}
              className={`p-4 rounded-lg cursor-pointer hover:opacity-80 transition-opacity ${isDark ? 'bg-gray-800' : 'bg-white'}`}
              onClick={() => {
                setNewReport({ ...newReport, report_type: item.type });
                setShowCreateModal(true);
              }}
            >
              <div className="text-3xl mb-2">{item.icon}</div>
              <p className={`font-medium ${isDark ? 'text-white' : 'text-gray-900'}`}>{item.label}</p>
            </div>
          ))}
        </div>

        {/* Reports Table */}
        <div className={`rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          <div className="p-4 border-b border-gray-200 dark:border-gray-700">
            <h2 className={`text-lg font-semibold ${isDark ? 'text-white' : 'text-gray-900'}`}>
              Generated Reports
            </h2>
          </div>
          {reports.length === 0 ? (
            <div className="p-8 text-center">
              <p className={isDark ? 'text-gray-400' : 'text-gray-600'}>No reports generated yet</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
                  <tr>
                    <th className={`px-4 py-3 text-left text-sm font-semibold ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>Type</th>
                    <th className={`px-4 py-3 text-left text-sm font-semibold ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>Format</th>
                    <th className={`px-4 py-3 text-left text-sm font-semibold ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>Period</th>
                    <th className={`px-4 py-3 text-left text-sm font-semibold ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>Status</th>
                    <th className={`px-4 py-3 text-left text-sm font-semibold ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>Created</th>
                    <th className={`px-4 py-3 text-left text-sm font-semibold ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>Action</th>
                  </tr>
                </thead>
                <tbody>
                  {reports.map((report, idx) => (
                    <tr key={report.id} className={idx % 2 === 0 ? (isDark ? 'bg-gray-800' : 'bg-white') : (isDark ? 'bg-gray-750' : 'bg-gray-50')}>
                      <td className={`px-4 py-3 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
                        <span className="capitalize">{report.report_type}</span>
                      </td>
                      <td className="px-4 py-3">
                        <span className="flex items-center gap-2">
                          <span>{getFormatIcon(report.format)}</span>
                          <span className={`uppercase ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>{report.format}</span>
                        </span>
                      </td>
                      <td className={`px-4 py-3 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
                        {new Date(report.period_start).toLocaleDateString()} - {new Date(report.period_end).toLocaleDateString()}
                      </td>
                      <td className="px-4 py-3">
                        <span className={`px-2 py-1 rounded text-xs ${getStatusColor(report.status)}`}>
                          {report.status}
                        </span>
                      </td>
                      <td className={`px-4 py-3 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
                        {new Date(report.created_at).toLocaleString()}
                      </td>
                      <td className="px-4 py-3">
                        {report.status === 'ready' && (
                          <button
                            onClick={() => downloadReport(report)}
                            className="text-blue-600 hover:text-blue-800"
                          >
                            Download
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Create Modal */}
        {showCreateModal && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className={`p-6 rounded-lg w-full max-w-md ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
              <h2 className={`text-xl font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                Generate Report
              </h2>
              <div className="space-y-4">
                <div>
                  <label className={`block text-sm font-medium mb-1 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>Report Type</label>
                  <select
                    value={newReport.report_type}
                    onChange={(e) => setNewReport({ ...newReport, report_type: e.target.value })}
                    className={`w-full px-4 py-2 rounded-lg border ${isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'}`}
                  >
                    <option value="transactions">Transactions</option>
                    <option value="users">Users</option>
                    <option value="kyc">KYC</option>
                    <option value="revenue">Revenue</option>
                    <option value="compliance">Compliance</option>
                  </select>
                </div>
                <div>
                  <label className={`block text-sm font-medium mb-1 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>Format</label>
                  <select
                    value={newReport.format}
                    onChange={(e) => setNewReport({ ...newReport, format: e.target.value })}
                    className={`w-full px-4 py-2 rounded-lg border ${isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'}`}
                  >
                    <option value="csv">CSV</option>
                    <option value="xlsx">Excel (XLSX)</option>
                    <option value="pdf">PDF</option>
                  </select>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className={`block text-sm font-medium mb-1 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>Start Date</label>
                    <input
                      type="date"
                      value={newReport.period_start}
                      onChange={(e) => setNewReport({ ...newReport, period_start: e.target.value })}
                      className={`w-full px-4 py-2 rounded-lg border ${isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'}`}
                    />
                  </div>
                  <div>
                    <label className={`block text-sm font-medium mb-1 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>End Date</label>
                    <input
                      type="date"
                      value={newReport.period_end}
                      onChange={(e) => setNewReport({ ...newReport, period_end: e.target.value })}
                      className={`w-full px-4 py-2 rounded-lg border ${isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'}`}
                    />
                  </div>
                </div>
              </div>
              <div className="flex justify-end gap-2 mt-6">
                <button
                  onClick={() => setShowCreateModal(false)}
                  className={`px-4 py-2 rounded-lg ${isDark ? 'bg-gray-700 text-white' : 'bg-gray-200 text-gray-800'}`}
                >
                  Cancel
                </button>
                <button
                  onClick={createReport}
                  className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
                >
                  Generate
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
