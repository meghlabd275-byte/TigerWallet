import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { analyticsService } from '../services/api';
import { PieChart, Pie, Cell, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, LineChart, Line } from 'recharts';

interface ComplianceData {
  totalKYC: number;
  pendingKYC: number;
  approvedKYC: number;
  rejectedKYC: number;
  highRiskUsers: number;
  suspiciousActivity: number;
  transactionsFlagged: number;
  complianceScore: number;
}

export default function ComplianceDashboard() {
  const { isDark } = useTheme();
  const [data, setData] = useState<ComplianceData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    loadComplianceData();
  }, []);

  const loadComplianceData = async () => {
    setLoading(true);
    setError('');
    try {
      const response = await analyticsService.getComplianceDashboard();
      setData(response.data);
    } catch (err: any) {
      setError(err.message || 'Failed to load compliance data');
    } finally {
      setLoading(false);
    }
  };

  const kycPieData = data ? [
    { name: 'Pending', value: data.pendingKYC, color: '#F59E0B' },
    { name: 'Approved', value: data.approvedKYC, color: '#10B981' },
    { name: 'Rejected', value: data.rejectedKYC, color: '#EF4444' },
  ] : [];

  const riskData = [
    { name: 'High Risk', value: data?.highRiskUsers || 0, fill: '#EF4444' },
    { name: 'Medium Risk', value: Math.floor((data?.highRiskUsers || 0) * 1.5), fill: '#F59E0B' },
    { name: 'Low Risk', value: Math.floor((data?.highRiskUsers || 0) * 3), fill: '#10B981' },
  ];

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
            Compliance Dashboard
          </h1>
          <button
            onClick={loadComplianceData}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            Refresh
          </button>
        </div>

        {/* Compliance Score */}
        <div className={`mb-6 p-6 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          <div className="flex items-center justify-between">
            <div>
              <h2 className={`text-lg font-semibold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                Compliance Score
              </h2>
              <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>
                Overall platform compliance rating
              </p>
            </div>
            <div className="text-center">
              <div className={`text-5xl font-bold ${data?.complianceScore && data.complianceScore >= 90 ? 'text-green-500' : data?.complianceScore && data.complianceScore >= 70 ? 'text-yellow-500' : 'text-red-500'}`}>
                {data?.complianceScore || 0}%
              </div>
            </div>
          </div>
        </div>

        {/* Stats Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
          <div className={`p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Total KYC</p>
            <p className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
              {data?.totalKYC || 0}
            </p>
          </div>
          <div className={`p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Pending KYC</p>
            <p className="text-2xl font-bold text-yellow-500">{data?.pendingKYC || 0}</p>
          </div>
          <div className={`p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>High Risk Users</p>
            <p className="text-2xl font-bold text-red-500">{data?.highRiskUsers || 0}</p>
          </div>
          <div className={`p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Flagged Transactions</p>
            <p className="text-2xl font-bold text-orange-500">{data?.transactionsFlagged || 0}</p>
          </div>
        </div>

        {/* Charts */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* KYC Distribution */}
          <div className={`p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <h3 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
              KYC Distribution
            </h3>
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={kycPieData}
                  cx="50%"
                  cy="50%"
                  innerRadius={60}
                  outerRadius={100}
                  paddingAngle={5}
                  dataKey="value"
                  label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                >
                  {kycPieData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{
                    backgroundColor: isDark ? '#1F2937' : '#fff',
                    border: isDark ? '1px solid #374151' : '1px solid #E5E7EB',
                    borderRadius: '8px',
                  }}
                />
              </PieChart>
            </ResponsiveContainer>
          </div>

          {/* Risk Distribution */}
          <div className={`p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <h3 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
              User Risk Distribution
            </h3>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={riskData}>
                <CartesianGrid strokeDasharray="3 3" stroke={isDark ? '#374151' : '#E5E7EB'} />
                <XAxis dataKey="name" stroke={isDark ? '#9CA3AF' : '#6B7280'} />
                <YAxis stroke={isDark ? '#9CA3AF' : '#6B7280'} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: isDark ? '#1F2937' : '#fff',
                    border: isDark ? '1px solid #374151' : '1px solid #E5E7EB',
                    borderRadius: '8px',
                  }}
                />
                <Bar dataKey="value" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Suspicious Activity */}
        <div className={`mt-6 p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          <h3 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
            Recent Suspicious Activity
          </h3>
          <div className={`text-center py-8 ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>
            {data?.suspiciousActivity || 0} suspicious activities detected
          </div>
        </div>
      </div>
    </div>
  );
}
