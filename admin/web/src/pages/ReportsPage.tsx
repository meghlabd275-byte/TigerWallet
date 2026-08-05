// TigerWallet Admin - Reports Page
// Generate and export reports

import React, { useState, useEffect } from 'react';
import { adminApi } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

interface Report {
  id: string;
  name: string;
  type: string;
  period: string;
  generatedAt: string;
  status: 'ready' | 'generating' | 'failed';
  downloadUrl?: string;
}

const ReportsPage: React.FC = () => {
  const { resolvedTheme } = useTheme();
  const [reports, setReports] = useState<Report[]>([]);
  const [loading, setLoading] = useState(true);
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [reportType, setReportType] = useState('users');
  const [period, setPeriod] = useState('30d');

  useEffect(() => {
    loadReports();
  }, []);

  const loadReports = async () => {
    try {
      setLoading(true);
      // Simulated reports - in real implementation would call API
      setReports([
        { id: '1', name: 'User Report', type: 'users', period: '30d', generatedAt: new Date().toISOString(), status: 'ready', downloadUrl: '#' },
        { id: '2', name: 'Transaction Report', type: 'transactions', period: '7d', generatedAt: new Date().toISOString(), status: 'ready', downloadUrl: '#' },
      ]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load reports');
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

  const handleGenerateReport = async () => {
    setGenerating(true);
    try {
      // Simulate report generation
      await new Promise(resolve => setTimeout(resolve, 2000));
      const newReport: Report = {
        id: Date.now().toString(),
        name: `${reportType.charAt(0).toUpperCase() + reportType.slice(1)} Report`,
        type: reportType,
        period: period,
        generatedAt: new Date().toISOString(),
        status: 'ready',
        downloadUrl: '#',
      };
      setReports([newReport, ...reports]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate report');
    } finally {
      setGenerating(false);
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'ready': return 'badge-success';
      case 'generating': return 'badge-warning';
      case 'failed': return 'badge-error';
      default: return 'badge-neutral';
    }
  };

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Reports</h1>
        <p style={{ color: colors.textSecondary }}>Generate and export platform reports</p>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      {/* Generate Report */}
      <div className="card mb-6" style={{ backgroundColor: colors.bgCard }}>
        <div className="card-body">
          <h3 className="font-semibold mb-4" style={{ color: colors.text }}>Generate New Report</h3>
          <div className="flex gap-4 flex-wrap">
            <div className="w-48">
              <label className="form-label">Report Type</label>
              <select
                className="form-select"
                value={reportType}
                onChange={(e) => setReportType(e.target.value)}
              >
                <option value="users">User Report</option>
                <option value="transactions">Transaction Report</option>
                <option value="revenue">Revenue Report</option>
                <option value="kyc">KYC Report</option>
                <option value="withdrawals">Withdrawal Report</option>
              </select>
            </div>
            <div className="w-48">
              <label className="form-label">Period</label>
              <select
                className="form-select"
                value={period}
                onChange={(e) => setPeriod(e.target.value)}
              >
                <option value="7d">Last 7 days</option>
                <option value="30d">Last 30 days</option>
                <option value="90d">Last 90 days</option>
                <option value="1y">Last year</option>
              </select>
            </div>
            <div className="flex items-end">
              <button
                className="btn btn-primary"
                onClick={handleGenerateReport}
                disabled={generating}
              >
                {generating ? 'Generating...' : 'Generate Report'}
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Reports List */}
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
                  <th>Name</th>
                  <th>Type</th>
                  <th>Period</th>
                  <th>Generated</th>
                  <th>Status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {reports.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="text-center py-8" style={{ color: colors.textSecondary }}>
                      No reports generated yet
                    </td>
                  </tr>
                ) : (
                  reports.map((report) => (
                    <tr key={report.id}>
                      <td style={{ color: colors.text }}>{report.name}</td>
                      <td style={{ color: colors.textSecondary }}>{report.type}</td>
                      <td style={{ color: colors.textSecondary }}>{report.period}</td>
                      <td style={{ color: colors.textSecondary }}>
                        {new Date(report.generatedAt).toLocaleString()}
                      </td>
                      <td>
                        <span className={`badge ${getStatusBadge(report.status)}`}>
                          {report.status}
                        </span>
                      </td>
                      <td>
                        {report.status === 'ready' && (
                          <button className="btn btn-sm btn-outline">
                            Download
                          </button>
                        )}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
};

export default ReportsPage;
